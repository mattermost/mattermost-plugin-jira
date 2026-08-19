# Phase 3 Plan: Domain logic — cache, tab resolution, JQL

> Prescriptive implementation plan for **Phase 3 only** (tasks 3.1–3.4). An
> Implementation Engineer should be able to land this without making design
> decisions. Do not implement later phases from this file.
>
> **Do not write production/test Go until this plan is followed as written.**
> This document is the plan; it is not the code.
>
> **This is the highest-risk phase (risk #36).** Jira Cloud never errors on a
> bad `statusCategory` operand: `=` matches nothing, `!=` matches **everything**.
> The Assigned tab is `statusCategory != done`. If validation is skipped, this
> phase compiles, tests can pass, Phase 4 will return 200, and Done tickets
> appear. The mitigations below are not optional and Gate 3 must be **executed**.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase 3 (source of truth for WHAT)
- **Spec:** `planner/projects/jira-plugin-rhs/ideas/001-show-tickets-rhs/spec.md`
  (§ "Tabs and JQL"; vanished statuses; 1-hour status cache)
- **Context:** `planner/projects/jira-plugin-rhs/context.md` § "Status categories"
  (silent-widen trap)
- **Orchestration:** Gate 3 / Step 3 of `IMPL_ORCHESTRATION_PLAN.md` —
  **executable, not inspect-only**
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `d34ccb2` (Phase 2 Cloud client — already committed)
- **Package:** `package main` under `server/` (module
  `github.com/mattermost/mattermost-plugin-jira`, `go.mod` at repo root)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer for all of Phase 3.** T3.1 and T3.2/T3.3 are
  file-independent, but do not split across two agents. See
  "Implementation order (single engineer)".

## Scope

**In scope:**

- Append sentinels to `server/rhs_types.go` (`ErrInvalidStatusCategory`,
  `ErrInvalidRHSSort`, `ErrUnknownRHSTabKind`, `ErrInvalidRHSTab`)
- Create `server/rhs_jql.go` (`validateStatusCategoryOperand`, `buildTabJQL`,
  `resolveTabs`, `validCategoryKeysFrom`)
- Create `server/rhs_jql_test.go` (all names `TestRHSJQL*`)
- Create `server/rhs_cache.go` (cache types, `getInstanceStatuses`,
  `invalidateRHSStatusCache`)
- Create `server/rhs_cache_test.go` (all names `TestRHSCache*`)
- Modify `server/plugin.go`: two `Plugin` fields, `OnActivate` map init,
  `invalidateRHSStatusCache()` call after `updateConfig`

**Out of scope (do not do these):**

- HTTP handlers, routes, JSON error convention (`rhs_http.go`, `http.go`)
- `RHSIssue` DTO / browse-link construction (`GetURL` vs `GetJiraBaseURL`)
- Calling `SearchJQL` / hitting `/search/jql` from this phase
- Adding methods to the shared `Client` / `SearchService` tree
- Any file under `webapp/`
- Diffing old vs new tab config (D3: coarse drop-all)
- Writing plugin config / `SavePluginConfig`
- Server/DC status listing
- Changing Phase 1 types or Phase 2 client method signatures

---

## Risk #36 — read this first

Jira Cloud **never errors** on a `statusCategory` operand it cannot resolve
(`context.md` § Status categories; spec § Tabs and JQL; spike `status-tab-jql`
probes 16–17 and 20–21):

| Clause | Unresolvable operand |
|--------|----------------------|
| `statusCategory = <bogus>` | 200, matches **nothing** |
| `statusCategory != <bogus>` | 200, matches **everything** |

The always-present Assigned tab is the negated form
(`statusCategory != done`). A skipped check does not fail CI by accident:

- the package compiles
- a test that only checks `err == nil` and a non-empty JQL string passes
- Phase 4 will return HTTP 200 with a non-empty issue list
- Done tickets appear on Assigned and look like working software

**Three mitigations are mandatory. If any is missing, this phase is not done.**

1. **Validate every `statusCategory` operand against the caller-supplied
   `validCategoryKeys` map, including the Assigned tab’s hardcoded `done`.**
   Assigned must not special-case around the check. The operand is
   `statusCategoryKeyDone` (`"done"`). If `done` is absent from the map,
   `buildTabJQL` returns an error. Do not validate against a hardcoded
   four-key allowlist *instead of* the map — the map is the cache’s canonical
   list, and the omit-`done` test is how we prove Assigned uses it.
2. **A test that fails if validation is removed.**
   `TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys` passes a map that
   contains `new` / `indeterminate` / `undefined` and **omits `done`**, then
   asserts `buildTabJQL` of the Assigned tab returns an error. Gate 3
   temporarily stubs `validateStatusCategoryOperand` to `return nil` and
   requires this test to FAIL.
3. **A negated-clause test that asserts membership, not status / non-emptiness.**
   `TestRHSJQLAssignedExcludesDoneMembership` builds Assigned JQL, applies a
   Cloud-faithful in-memory interpreter to a fixture set that **includes** a
   Done issue (`TES-DONE`), and asserts `TES-DONE` is **absent**. It also
   asserts at least one non-Done issue **is present** (so an empty match set
   cannot pass). It must not be written as `assert.NoError` + `assert.NotEmpty`.

Do **not** add a `skipValidation` test hook. The omit-`done` map is the only
way to trigger the Assigned validation error.

---

## Research (read this before coding)

Line numbers are as of **`d34ccb2`** (Phase 2 committed). Parent `.planning/PLAN.md`
still cites pre-Phase-1 numbers for `plugin.go`. **Use this table, not the
parent’s `:126-169` / `:300`.**

### 1. `Plugin` struct and `teamFieldCache` precedent — `server/plugin.go`

Struct is **`server/plugin.go:132-175`** (parent said `:126-169`; Phase 1 added
eight lines to `externalConfig` and the struct drifted). The existing cache
precedent is at the **end** of the struct:

```173:175:server/plugin.go
	teamFieldCache     map[types.ID]map[string]struct{}
	teamFieldCacheLock sync.RWMutex
}
```

Insert the two RHS fields **immediately after** `teamFieldCacheLock`, before
the closing `}` of `Plugin`. Do not add them next to `confLock`.

`OnActivate` initializes `teamFieldCache` at **`:352`**:

```352:352:server/plugin.go
	p.teamFieldCache = make(map[types.ID]map[string]struct{})
```

Add `p.rhsStatusCache = make(map[types.ID]*rhsStatusCacheEntry)` on the next
line. Tests still lazy-init because `setupTestPlugin` never calls `OnActivate`.

### 2. `OnConfigurationChange` and the invalidation call site

Function is **`server/plugin.go:198-332`**. `p.updateConfig(...)` is
**`:307-310`**, not `:300`. Line 300 is currently
`p.client.Log.Warn("Some team entries were invalid and ignored")` inside the
`TeamIDs` parser. **Do not put invalidation there.**

```306:314:server/plugin.go
	prev := p.getConfig()
	p.updateConfig(func(conf *config) {
		conf.externalConfig = ec
		conf.maxAttachmentSize = maxAttachmentSize
	})

	// OnConfigurationChanged is first called before the plugin is activated,
```

Call `p.invalidateRHSStatusCache()` **immediately after** the `updateConfig`
block (after line 310, before the "OnConfigurationChanged is first called"
comment). Per **D3** this drops **all** instance entries on any config change
— no old-vs-new tab-config diff. Config saves are rare; a diff is extra state
and an extra failure mode.

`OnConfigurationChange` can run **before** `OnActivate` (comment at `:312-314`).
`invalidateRHSStatusCache` must be safe on a zero `Plugin` (nil map, unlocked
`RWMutex`).

Do **not** add TeamIDs-style parsing for `RHSStatusTabs`. Phase 1 already
loads it via `LoadPluginConfiguration`.

### 3. Current `server/rhs_types.go` after Phase 2

File is **83 lines**. Do **not** redefine Phase 1/2 types. Append sentinels
only.

| Symbol | Lines | Notes |
|--------|-------|-------|
| `RHSTabKindAssigned` / `Category` / `Status` | 14–18 | Assigned is a first-class kind (Phase 1 P1-D3). JQL is `statusCategory != done`, not `= key` / `= id` |
| `statusCategoryKeyNew` / `Indeterminate` / `Done` / `Undefined` | 23–26 | Use these as JQL operands. **Never** hardcode numeric ids 2/4/3/1 |
| `RHSTabEntry` | 32–37 | `Key` = category key; `ID` = numeric status id; `Name` = label |
| `rhsDefaultTabs()` | 41–46 | Assigned then In Progress (`indeterminate`). Virtual seed (D2) |
| `JiraStatus` | 50–54 | `ID string`, nested `StatusCategory` |
| `JiraStatusCategory` | 59–63 | `ID int`, `Key` / `Name` string |
| `ErrRateLimited` | 83 | Sentinel style to copy |

Parent T3.3’s return type `[]RHSTab` **does not exist**. Do not invent it.
`resolveTabs` returns `[]RHSTabEntry` (P3-D1).

### 4. Phase 2 client methods the cache calls

```70:84:server/client_cloud_rhs.go
func (client jiraCloudClient) ListStatuses() ([]*JiraStatus, error) {
	var result []*JiraStatus
	if err := client.RESTGet("3/status", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (client jiraCloudClient) ListStatusCategories() ([]*JiraStatusCategory, error) {
	var result []*JiraStatusCategory
	if err := client.RESTGet("3/statuscategory", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
```

Value receivers on `jiraCloudClient`. **Do not change these signatures.** The
cache talks to them through a local interface `rhsStatusLister` (same two
methods) so tests use a handwritten fake, not `httptest` and not the shared
`Client` tree.

`SearchJQL` (`:86-88`) is **not** called in this phase.

Fixtures the cache tests may unmarshal (same package, `loadRHSTestdata` in
`client_cloud_rhs_test.go` is reusable):

- `server/testdata/rhs-status.json` — statuses `id="3"` (In Progress /
  `indeterminate`) and `id="10001"` (Backlog / `new`)
- `server/testdata/rhs-statuscategory.json` — keys `undefined`, `new`,
  `indeterminate`, `done` (ids 1, 2, 4, 3)

### 5. `types.ID` for instance ids

```10:13:server/utils/types/id_set.go
type ID string

func (id ID) GetID() ID      { return id }
func (id ID) String() string { return string(id) }
```

Instance ids in this plugin are `types.ID` values, typically the site URL
(`https://example.atlassian.net`). `InstanceCommon.InstanceID` is `types.ID`
(`server/instance.go:39`); `GetID()` returns it (`:64-66`). `teamFieldCache`
is already `map[types.ID]…`. **Key the RHS cache the same way** — do not use
`string` or the `externalConfig.RHSStatusTabs` map key type as the cache key
type. Phase 4 will convert `instance.GetID()` when calling
`getInstanceStatuses`.

### 6. Existing mutex / cache pattern — `server/team_fields.go`

```12:66:server/team_fields.go
func (p *Plugin) cacheTeamFieldKeys(instanceID types.ID, keys []string) {
	// ...
	p.teamFieldCacheLock.Lock()
	defer p.teamFieldCacheLock.Unlock()
	if p.teamFieldCache == nil {
		p.teamFieldCache = make(map[types.ID]map[string]struct{})
	}
	// ...
}

func (p *Plugin) getTeamFieldKeys(instanceID types.ID) map[string]struct{} {
	p.teamFieldCacheLock.RLock()
	defer p.teamFieldCacheLock.RUnlock()
	// copy under the read lock
}
```

What to copy: `types.ID` map key, dedicated `sync.RWMutex`, lazy `nil` map
init, copy-on-read so callers cannot race with a later write.

What **not** to copy: `teamFieldCache` never fetches under the lock. It is
write-on-put only. `getInstanceStatuses` **does** fetch, so it needs
double-check locking (P3-D2): read-lock check, write-lock re-check, fetch
**while holding the write lock** so a concurrent warm cannot double-fetch.

### 7. How tests construct a `Plugin` with locks — `setupTestPlugin`

```99:114:server/issue_test.go
func setupTestPlugin(api *plugintest.API) *Plugin {
	api.On("LogError", mockAnythingOfTypeBatch("string", 13)...).Return()
	api.On("LogDebug", mockAnythingOfTypeBatch("string", 11)...).Return()

	p := &Plugin{}
	p.SetAPI(api)
	p.initializeRouter()
	p.instanceStore = p.getMockInstanceStoreKV(1)
	p.userStore = getMockUserStoreKV()
	p.client = pluginapi.NewClient(api, p.Driver)
	p.updateConfig(func(conf *config) {
		conf.Secret = someSecret
	})

	return p
}
```

`setupTestPlugin` does **not** initialize `teamFieldCache` or call
`OnActivate`. Zero-value `sync.RWMutex` is usable. Cache methods **must**
lazy-init `rhsStatusCache` so this helper works unchanged.

Use `setupTestPlugin` for cache tests that need a `Plugin` (invalidation,
`OnConfigurationChange`). JQL tests are pure functions — no `Plugin`.

`OnConfigurationChange` tests must additionally mock:

- `api.On("LoadPluginConfiguration", mock.Anything).Return(nil)` —
  `p.client.Configuration.LoadPluginConfiguration` delegates to the API
- `api.On("GetConfig").Return(&model.Config{FileSettings: model.FileSettings{MaxFileSize: &max}})`
  — `:211` reads `p.API.GetConfig().FileSettings.MaxFileSize` and will panic
  on a nil config
- `api.On("LogWarn", …)` if the function logs (empty `AdminAPIToken` skips
  the encrypt path; empty `TeamIDs` skips the warn)

Leave `EnableAutocomplete` unchanged so the command-registration branch at
`:315` does not run.

### 8. Confirmed absent files

As of `d34ccb2`:

| Path | Exists? |
|------|---------|
| `server/rhs_jql.go` | **No** — create |
| `server/rhs_jql_test.go` | **No** — create |
| `server/rhs_cache.go` | **No** — create |
| `server/rhs_cache_test.go` | **No** — create |
| `server/rhs.go` / `rhs_http.go` | **No** — Phase 4 |

`Glob **/rhs_*.go` today is only `rhs_types.go` and `rhs_types_test.go`.

### 9. File header and lint (same as Phases 1–2)

Every new `server/*.go` file starts with:

```go
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
```

Year is **2017**. `package main`.

Enabled linters (`.golangci.yml:25-38`): `bodyclose`, `errcheck`, `gocritic`,
`gosec`, `govet`, `ineffassign`, `misspell`, `nakedret`, `revive`,
`staticcheck`, `unconvert`, `unused`, `whitespace`. This phase holds no raw
`*http.Response`. `revive` `exported` / `var-naming` / `package-comments` are
excluded — unexported helpers are fine.

Formatters: `gofmt` + `goimports` with
`local-prefixes: github.com/mattermost/mattermost-plugin-jira`.

Do not `t.Parallel()` on cache tests that share a `Plugin` or a fake client
with counters.

### 10. Pitfall confirmation vs parent `.planning/PLAN.md`

| Claim in parent PLAN.md | Current worktree (`d34ccb2`) | Drift? |
|-------------------------|------------------------------|--------|
| `Plugin` struct `:126-169` | **`:132-175`** | **Yes — use 132–175** |
| `updateConfig` in `OnConfigurationChange` at `:300` | `updateConfig` is **`:307-310`**; `:300` is a TeamIDs `Log.Warn` | **Yes — invalidate after 310, not at 300** |
| `OnConfigurationChange` `:192-326` | **`:198-332`** | Cosmetic |
| `teamFieldCache` / lock as the cache precedent | `:173-174`; init at `:352`; impl `team_fields.go:12-66` | **No** |
| `rhsDefaultTabs()` Assigned + In Progress | `rhs_types.go:41-46` | **No** |
| `JiraStatus` / `JiraStatusCategory` already landed | `rhs_types.go:50-63` | **No** |
| `ListStatuses` / `ListStatusCategories` | `client_cloud_rhs.go:70-84` | **No** |
| `resolveTabs` returns `[]RHSTab` | **`RHSTab` does not exist** | **Yes — return `[]RHSTabEntry`** |
| `rhs_jql.go` / `rhs_cache.go` exist | **neither exists** | **No** (create) |
| `setupTestPlugin` constructs Plugin + locks | `issue_test.go:99-114`; does **not** init caches | **Yes — lazy-init required** |

---

## Decisions this phase locks (do not reopen)

**P3-D1 — `resolveTabs` returns `[]RHSTabEntry`.** Parent T3.3 said `[]RHSTab`.
That type was never added. Phase 4 serializes the same shape (`kind`, `key`,
`id`, `name`). Do not add a duplicate type.

**P3-D2 — Fetch under the write lock after a double-check.** Read-lock miss →
write-lock → re-check → if still miss, call `ListStatuses` then
`ListStatusCategories` while holding the write lock → store → unlock. This is
the only way “re-check under the write lock so a concurrent warm does not
double-fetch” is true. Do not fetch unlocked with an inflight flag.

**P3-D3 — Coarse invalidation (parent D3).**
`invalidateRHSStatusCache` replaces the whole map. One call from
`OnConfigurationChange` after `updateConfig`. No per-instance delete, no
tab-config diff.

**P3-D4 — Assigned is a kind, not a missing key.** Switch on
`RHSTabKindAssigned`. The operand is the constant `statusCategoryKeyDone`,
not `tab.Key` (Assigned’s Key is empty). That constant still goes through
`validateStatusCategoryOperand`.

**P3-D5 — Operand validation uses the caller’s map, not a hardcoded
allowlist.** `validCategoryKeys` is the canonical list (from the cached
`/statuscategory` result in Phase 4). If the map is `nil` or omits the
operand — **including `done` for Assigned** — return
`ErrInvalidStatusCategory`. A hardcoded `if operand == "done" { return nil }`
bypass would make the omit-`done` test pass without proving anything.

**P3-D6 — No `key → numeric id` translation.** Pass `tab.Key` through
unchanged as the JQL operand (spike-verified for `new`, `indeterminate`,
`done`). Unquoted. Same for `tab.ID` on status tabs.

**P3-D7 — Sort allowlist is exact lowercase `updated` and `created`.** Reject
everything else, including empty, `UPDATED`, `priority`, and
`updated DESC`. Phase 4 defaults empty query params to `updated`; this
function does not default.

**P3-D8 — Vanished tabs are hidden, not errored.** Unknown status id, unknown
category key, empty key/id, unknown kind: omit from the resolved list.
Assigned is never omitted.

**P3-D9 — Empty / nil configured list → `rhsDefaultTabs()` (D2).** Then still
apply vanished-tab hiding to the seed extras (if `indeterminate` is missing
from `categories`, In Progress is hidden; Assigned remains).

**P3-D10 — One implementer.** Do not require two agents. Recommended order
below.

**P3-D11 — Membership tests use a Cloud-faithful interpreter, not HTTP.**
Phase 3 has no handlers. `issuesMatchingJQL` (test-only) models Cloud:
resolvable `=` / `!=` filter by category key; unresolvable `=` matches
nothing; unresolvable `!=` matches **everything**. That is how a negated
bogus operand is proven to include `TES-DONE`.

---

## Implementation order (single engineer)

T3.1 (`rhs_cache.go` + `plugin.go`) and T3.2/T3.3 (`rhs_jql.go`) do not share
production files except the sentinel append on `rhs_types.go`. One engineer
does all of it. **Do not spawn a second implementer.**

Recommended order so Gate 3 can be run before cache work:

1. Append sentinels to `rhs_types.go`
2. T3.2 + T3.3 — `rhs_jql.go`
3. T3.4 JQL tests — `rhs_jql_test.go`
4. **Execute Gate 3 steps 1–3 locally** (see Gate 3 checklist)
5. T3.1 — `plugin.go` + `rhs_cache.go`
6. T3.4 cache tests — `rhs_cache_test.go`

---

## Exact sentinels (`server/rhs_types.go`) — do first

Keep lines 1–83 unchanged. After `ErrRateLimited`, append:

```go
// ErrInvalidStatusCategory is returned when a statusCategory JQL operand is
// missing from validCategoryKeys. Phase 4 maps this to invalid_request.
// Assigned's hardcoded "done" uses this path — do not special-case it.
var ErrInvalidStatusCategory = errors.New("invalid status category operand")

// ErrInvalidRHSSort is returned when sortField is not "updated" or "created".
var ErrInvalidRHSSort = errors.New("invalid rhs sort field")

// ErrUnknownRHSTabKind is returned when buildTabJQL sees an unknown Kind.
var ErrUnknownRHSTabKind = errors.New("unknown rhs tab kind")

// ErrInvalidRHSTab is returned when a status tab has an empty ID.
var ErrInvalidRHSTab = errors.New("invalid rhs tab")
```

`errors` is already imported.

---

## Exact JQL builder (`server/rhs_jql.go`) — Tasks 3.2 / 3.3

Create the file. `package main`. Header as in §9.

### Helpers

```go
func validCategoryKeysFrom(categories []*JiraStatusCategory) map[string]bool {
	out := make(map[string]bool, len(categories))
	for _, c := range categories {
		if c == nil || c.Key == "" {
			continue
		}
		out[c.Key] = true
	}
	return out
}

func statusIDSet(statuses []*JiraStatus) map[string]bool {
	out := make(map[string]bool, len(statuses))
	for _, s := range statuses {
		if s == nil || s.ID == "" {
			continue
		}
		out[s.ID] = true
	}
	return out
}

// validateStatusCategoryOperand is the #36 mitigation.
// Every statusCategory operand — including Assigned's statusCategoryKeyDone —
// must pass through this function. Do not inline the map check at call sites
// in a way that lets Assigned skip it. Gate 3 stubs THIS function to
// `return nil` and requires TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys
// to FAIL.
func validateStatusCategoryOperand(operand string, validCategoryKeys map[string]bool) error {
	if operand == "" || validCategoryKeys == nil || !validCategoryKeys[operand] {
		return fmt.Errorf("%w: %q", ErrInvalidStatusCategory, operand)
	}
	return nil
}

func rhsOrderBy(sortField string) (string, error) {
	switch sortField {
	case "updated", "created":
		return " ORDER BY " + sortField + " DESC, key ASC", nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidRHSSort, sortField)
	}
}
```

### `buildTabJQL` — Task 3.2 ⚠️ risk #36

Signature is locked (parent T3.2):

```go
func buildTabJQL(tab RHSTabEntry, sortField string, validCategoryKeys map[string]bool) (string, error)
```

Body, locked:

```go
func buildTabJQL(tab RHSTabEntry, sortField string, validCategoryKeys map[string]bool) (string, error) {
	order, err := rhsOrderBy(sortField)
	if err != nil {
		return "", err
	}

	const assignee = "assignee = currentUser()"

	switch tab.Kind {
	case RHSTabKindAssigned:
		// Validate the hardcoded done — this is the clause whose failure is
		// invisible on Cloud (!= bogus matches everything).
		if err := validateStatusCategoryOperand(statusCategoryKeyDone, validCategoryKeys); err != nil {
			return "", err
		}
		return assignee + " AND statusCategory != " + statusCategoryKeyDone + order, nil

	case RHSTabKindCategory:
		if err := validateStatusCategoryOperand(tab.Key, validCategoryKeys); err != nil {
			return "", err
		}
		return assignee + " AND statusCategory = " + tab.Key + order, nil

	case RHSTabKindStatus:
		if tab.ID == "" {
			return "", fmt.Errorf("%w: empty status id", ErrInvalidRHSTab)
		}
		return assignee + " AND status = " + tab.ID + order, nil

	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownRHSTabKind, tab.Kind)
	}
}
```

**Locked JQL strings** (exact, including spaces and `, key ASC`):

| Tab | sortField | Exact JQL |
|-----|-----------|-----------|
| Assigned | `updated` | `assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC` |
| Assigned | `created` | `assignee = currentUser() AND statusCategory != done ORDER BY created DESC, key ASC` |
| Category `indeterminate` | `updated` | `assignee = currentUser() AND statusCategory = indeterminate ORDER BY updated DESC, key ASC` |
| Category `indeterminate` | `created` | `assignee = currentUser() AND statusCategory = indeterminate ORDER BY created DESC, key ASC` |
| Category `new` | `updated` | `assignee = currentUser() AND statusCategory = new ORDER BY updated DESC, key ASC` |
| Status `10001` | `updated` | `assignee = currentUser() AND status = 10001 ORDER BY updated DESC, key ASC` |
| Status `3` | `created` | `assignee = currentUser() AND status = 3 ORDER BY created DESC, key ASC` |

Rules:

- Scope is **assignee only**. Never `reporter` or `watcher`.
- Category key and status id are interpolated **unquoted** and **unchanged**.
- The only `statusCategory` construction sites in production Go are the two
  concatenations above, each immediately after a successful
  `validateStatusCategoryOperand`.
- Use `statusCategoryKeyDone` (not the literal `"done"` next to the
  concatenation) so a rename stays consistent. The test still asserts the
  emitted string contains `statusCategory != done`.

### `resolveTabs` — Task 3.3

Signature (return type is `[]RHSTabEntry` per P3-D1):

```go
func resolveTabs(configured []RHSTabEntry, statuses []*JiraStatus, categories []*JiraStatusCategory) []RHSTabEntry
```

Behavior, locked:

1. Result always starts with
   `{Kind: RHSTabKindAssigned, Name: "Assigned"}` (empty Key/ID). Assigned is
   never configurable, never removable, always first.
2. Extra entries:
   - If `len(configured) == 0` (nil or empty slice): extras =
     `rhsDefaultTabs()[1:]` (skip the seed’s Assigned so it is not
     duplicated).
   - Else: extras = `configured`.
3. For each extra, in order:
   - `Kind == RHSTabKindAssigned` → **skip** (already prepended).
   - `Kind == RHSTabKindCategory` → keep iff `tab.Key != ""` and
     `validCategoryKeysFrom(categories)[tab.Key]`.
   - `Kind == RHSTabKindStatus` → keep iff `tab.ID != ""` and
     `statusIDSet(statuses)[tab.ID]`.
   - Any other kind, empty key/id, or failed membership → **hide** (omit).
     Do not error. Do not emit an empty placeholder.
4. Matching is case-sensitive against the **passed-in** slices, not a
   hardcoded four-key table. If `categories` is empty, every category extra
   (including seed In Progress) is hidden; Assigned remains.

Do not write plugin config. Do not call Jira.

---

## Exact cache (`server/rhs_cache.go` + `server/plugin.go`) — Task 3.1

### Types and const (`server/rhs_cache.go`)

```go
const rhsStatusCacheTTL = time.Hour

type rhsStatusCacheEntry struct {
	statuses   []*JiraStatus
	categories []*JiraStatusCategory
	fetchedAt  time.Time
}

// rhsStatusLister is the cache's view of the Cloud client.
// Phase 4's rhsCloudClient will include these methods.
type rhsStatusLister interface {
	ListStatuses() ([]*JiraStatus, error)
	ListStatusCategories() ([]*JiraStatusCategory, error)
}

func copyRHSStatusCacheEntry(in *rhsStatusCacheEntry) *rhsStatusCacheEntry {
	if in == nil {
		return nil
	}
	out := &rhsStatusCacheEntry{fetchedAt: in.fetchedAt}
	if in.statuses != nil {
		out.statuses = append([]*JiraStatus(nil), in.statuses...)
	}
	if in.categories != nil {
		out.categories = append([]*JiraStatusCategory(nil), in.categories...)
	}
	return out
}
```

TTL is **`time.Hour`**, not 24h, not configurable. Known and accepted: a
status renamed or deleted **in Jira** can stay cached up to an hour; each
node warms its own copy. The ticket list is never cached.

### `Plugin` fields — `server/plugin.go` after line 174

```go
	teamFieldCache     map[types.ID]map[string]struct{}
	teamFieldCacheLock sync.RWMutex

	rhsStatusCache     map[types.ID]*rhsStatusCacheEntry
	rhsStatusCacheLock sync.RWMutex
}
```

`sync` and `types` are already imported. `rhsStatusCacheEntry` lives in
`rhs_cache.go` (same package).

### `OnActivate` — after line 352

```go
	p.teamFieldCache = make(map[types.ID]map[string]struct{})
	p.rhsStatusCache = make(map[types.ID]*rhsStatusCacheEntry)
```

### Double-check locking — `getInstanceStatuses`

```go
func (p *Plugin) freshRHSStatusCacheLocked(instanceID types.ID) *rhsStatusCacheEntry {
	entry := p.rhsStatusCache[instanceID]
	if entry == nil {
		return nil
	}
	if time.Since(entry.fetchedAt) >= rhsStatusCacheTTL {
		return nil
	}
	return copyRHSStatusCacheEntry(entry)
}

func (p *Plugin) getInstanceStatuses(instanceID types.ID, client rhsStatusLister) (*rhsStatusCacheEntry, error) {
	p.rhsStatusCacheLock.RLock()
	if entry := p.freshRHSStatusCacheLocked(instanceID); entry != nil {
		p.rhsStatusCacheLock.RUnlock()
		return entry, nil
	}
	p.rhsStatusCacheLock.RUnlock()

	p.rhsStatusCacheLock.Lock()
	defer p.rhsStatusCacheLock.Unlock()

	if entry := p.freshRHSStatusCacheLocked(instanceID); entry != nil {
		return entry, nil
	}

	statuses, err := client.ListStatuses()
	if err != nil {
		return nil, err
	}
	categories, err := client.ListStatusCategories()
	if err != nil {
		return nil, err
	}

	entry := &rhsStatusCacheEntry{
		statuses:   statuses,
		categories: categories,
		fetchedAt:  time.Now(),
	}
	if p.rhsStatusCache == nil {
		p.rhsStatusCache = make(map[types.ID]*rhsStatusCacheEntry)
	}
	p.rhsStatusCache[instanceID] = entry
	return copyRHSStatusCacheEntry(entry), nil
}
```

Rules:

- Fresh means `time.Since(fetchedAt) < rhsStatusCacheTTL`. Equal-to-TTL is
  expired (`>=`).
- Fetch **statuses then categories**, sequential, while holding the write
  lock. If either call fails, **do not store** (no partial cache).
- Return a **copy** (new struct + copied slice headers) so callers cannot
  mutate the cached slices.
- `instanceID` is `types.ID`.

### `invalidateRHSStatusCache` and call site

```go
func (p *Plugin) invalidateRHSStatusCache() {
	p.rhsStatusCacheLock.Lock()
	defer p.rhsStatusCacheLock.Unlock()
	p.rhsStatusCache = make(map[types.ID]*rhsStatusCacheEntry)
}
```

Call site in `OnConfigurationChange`, **after** `:307-310`, **before** the
comment at `:312`:

```go
	prev := p.getConfig()
	p.updateConfig(func(conf *config) {
		conf.externalConfig = ec
		conf.maxAttachmentSize = maxAttachmentSize
	})

	p.invalidateRHSStatusCache()

	// OnConfigurationChanged is first called before the plugin is activated,
```

Drops **all** instances. Safe if the map was nil.

---

## Exact tests — Task 3.4 ⚠️ risk #36

### `server/rhs_jql_test.go`

Create. `package main`. Header as in §9. **Every test name starts with
`TestRHSJQL`.**

Shared helpers (same file, unexported):

```go
func canonicalValidCategoryKeys() map[string]bool {
	return map[string]bool{
		statusCategoryKeyNew:           true,
		statusCategoryKeyIndeterminate: true,
		statusCategoryKeyDone:          true,
		statusCategoryKeyUndefined:     true,
	}
}

type rhsFakeIssue struct {
	Key         string
	CategoryKey string
	StatusID    string
}

func assignedMembershipFixture() []rhsFakeIssue {
	return []rhsFakeIssue{
		{Key: "TES-NEW", CategoryKey: statusCategoryKeyNew, StatusID: "10001"},
		{Key: "TES-IP", CategoryKey: statusCategoryKeyIndeterminate, StatusID: "3"},
		{Key: "TES-DONE", CategoryKey: statusCategoryKeyDone, StatusID: "10000"},
		{Key: "TES-UNDEF", CategoryKey: statusCategoryKeyUndefined, StatusID: "1"},
	}
}

// issuesMatchingJQL is test-only and models Cloud JQL, including the silent-
// widen trap: an unresolvable statusCategory operand makes "=" match nothing
// and "!=" match everything. Canonical resolvable keys are the four Cloud
// keys — independent of validCategoryKeys (that map is OUR validator; this
// function is Jira).
func issuesMatchingJQL(jql string, issues []rhsFakeIssue) []rhsFakeIssue {
	categoryRe := regexp.MustCompile(`statusCategory (!=|=) (\S+)`)
	statusRe := regexp.MustCompile(`status = (\S+)`)

	canonical := canonicalValidCategoryKeys()

	if m := categoryRe.FindStringSubmatch(jql); len(m) == 3 {
		op, operand := m[1], m[2]
		resolved := canonical[operand]
		var out []rhsFakeIssue
		for _, issue := range issues {
			switch {
			case op == "=" && !resolved:
				// Cloud: matches nothing
			case op == "!=" && !resolved:
				out = append(out, issue) // Cloud: matches everything
			case op == "=" && issue.CategoryKey == operand:
				out = append(out, issue)
			case op == "!=" && issue.CategoryKey != operand:
				out = append(out, issue)
			}
		}
		return out
	}
	if m := statusRe.FindStringSubmatch(jql); len(m) == 2 {
		id := m[1]
		var out []rhsFakeIssue
		for _, issue := range issues {
			if issue.StatusID == id {
				out = append(out, issue)
			}
		}
		return out
	}
	return nil
}

func issueKeys(issues []rhsFakeIssue) []string {
	keys := make([]string, 0, len(issues))
	for _, issue := range issues {
		keys = append(keys, issue.Key)
	}
	return keys
}
```

#### `TestRHSJQLBuildExactStrings`

Table: all three kinds × both sorts, plus one extra category key. Assert
**exact** strings, including `, key ASC`. Use `canonicalValidCategoryKeys()`.

```go
func TestRHSJQLBuildExactStrings(t *testing.T) {
	valid := canonicalValidCategoryKeys()
	assigned := RHSTabEntry{Kind: RHSTabKindAssigned, Name: "Assigned"}
	inProgress := RHSTabEntry{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"}
	todo := RHSTabEntry{Kind: RHSTabKindCategory, Key: statusCategoryKeyNew, Name: "To Do"}
	status := RHSTabEntry{Kind: RHSTabKindStatus, ID: "10001", Name: "Backlog"}
	statusOpen := RHSTabEntry{Kind: RHSTabKindStatus, ID: "3", Name: "In Progress"}

	tests := map[string]struct {
		tab  RHSTabEntry
		sort string
		want string
	}{
		"assigned updated": {
			tab: assigned, sort: "updated",
			want: "assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC",
		},
		"assigned created": {
			tab: assigned, sort: "created",
			want: "assignee = currentUser() AND statusCategory != done ORDER BY created DESC, key ASC",
		},
		"category indeterminate updated": {
			tab: inProgress, sort: "updated",
			want: "assignee = currentUser() AND statusCategory = indeterminate ORDER BY updated DESC, key ASC",
		},
		"category indeterminate created": {
			tab: inProgress, sort: "created",
			want: "assignee = currentUser() AND statusCategory = indeterminate ORDER BY created DESC, key ASC",
		},
		"category new updated": {
			tab: todo, sort: "updated",
			want: "assignee = currentUser() AND statusCategory = new ORDER BY updated DESC, key ASC",
		},
		"status 10001 updated": {
			tab: status, sort: "updated",
			want: "assignee = currentUser() AND status = 10001 ORDER BY updated DESC, key ASC",
		},
		"status 3 created": {
			tab: statusOpen, sort: "created",
			want: "assignee = currentUser() AND status = 3 ORDER BY created DESC, key ASC",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := buildTabJQL(tc.tab, tc.sort, valid)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
```

#### `TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys` — **THE validation-removal test**

This is the test Gate 3 stubs validation to prove. If T3.2’s Assigned check
is removed, this test must FAIL. Do not weaken it to accept a JQL string.

```go
func TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys(t *testing.T) {
	valid := map[string]bool{
		statusCategoryKeyNew:           true,
		statusCategoryKeyIndeterminate: true,
		statusCategoryKeyUndefined:     true,
		// done is deliberately omitted
	}
	tab := RHSTabEntry{Kind: RHSTabKindAssigned, Name: "Assigned"}

	got, err := buildTabJQL(tab, "updated", valid)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidStatusCategory)
	assert.Empty(t, got, "Assigned must not emit JQL when done is not in validCategoryKeys")
}
```

Also add a nil-map case in the same test or as a subtest: `validCategoryKeys == nil`
must error for Assigned (nil must not mean “skip validation”).

```go
	t.Run("nil map", func(t *testing.T) {
		got, err := buildTabJQL(RHSTabEntry{Kind: RHSTabKindAssigned, Name: "Assigned"}, "updated", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidStatusCategory)
		assert.Empty(t, got)
	})
```

#### `TestRHSJQLCategoryErrorsWhenKeyMissingFromValidKeys`

```go
func TestRHSJQLCategoryErrorsWhenKeyMissingFromValidKeys(t *testing.T) {
	valid := canonicalValidCategoryKeys()
	tab := RHSTabEntry{Kind: RHSTabKindCategory, Key: "bogusCategoryXYZ", Name: "Bogus"}

	got, err := buildTabJQL(tab, "updated", valid)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidStatusCategory)
	assert.Empty(t, got)
}
```

#### `TestRHSJQLRejectsSortOutsideAllowlist`

Table: `""`, `"UPDATED"`, `"priority"`, `"updated DESC"`. Each must
`errors.Is(err, ErrInvalidRHSSort)` and emit no JQL. Use Assigned +
`canonicalValidCategoryKeys()` so a sort failure is not confused with
category validation.

#### `TestRHSJQLAssignedExcludesDoneMembership` — **THE membership test**

Not a status-code test. Not a non-emptiness test.

```go
func TestRHSJQLAssignedExcludesDoneMembership(t *testing.T) {
	jql, err := buildTabJQL(
		RHSTabEntry{Kind: RHSTabKindAssigned, Name: "Assigned"},
		"updated",
		canonicalValidCategoryKeys(),
	)
	require.NoError(t, err)

	got := issuesMatchingJQL(jql, assignedMembershipFixture())
	keys := issueKeys(got)

	// Primary #36 assertion: a Done-category issue is ABSENT.
	// A bad != operand would include TES-DONE (superset, 200, non-empty).
	assert.NotContains(t, keys, "TES-DONE")

	// Guard against the other silent failure (= bogus → matches nothing).
	// Do not replace the NotContains assertion with NotEmpty.
	assert.Contains(t, keys, "TES-NEW")
	assert.Contains(t, keys, "TES-IP")
	assert.Contains(t, keys, "TES-UNDEF")
}
```

#### `TestRHSJQLNegatedBogusOperandWidensToIncludeDone`

Locks Cloud semantics in the test interpreter so a “helpful” change to
`issuesMatchingJQL` that stops modeling the trap cannot hide a bad Assigned
clause. Uses a **hand-built** JQL string, not `buildTabJQL`.

```go
func TestRHSJQLNegatedBogusOperandWidensToIncludeDone(t *testing.T) {
	jql := "assignee = currentUser() AND statusCategory != bogusCategoryXYZ ORDER BY updated DESC, key ASC"
	got := issuesMatchingJQL(jql, assignedMembershipFixture())
	keys := issueKeys(got)

	assert.Contains(t, keys, "TES-DONE", "Cloud != <bogus> matches everything; the interpreter must model that")
	assert.Contains(t, keys, "TES-NEW")
	assert.GreaterOrEqual(t, len(keys), 4)
}
```

Together: if someone changes Assigned to `!= bogusCategoryXYZ` (and updates
the exact-string test), `TestRHSJQLAssignedExcludesDoneMembership` fails
because `TES-DONE` appears. If they remove validation, the omit-`done` test
fails.

#### `TestRHSJQLUnknownKindAndEmptyStatusID`

- `{Kind: "nope"}` → `ErrUnknownRHSTabKind`, empty string
- `{Kind: status, ID: ""}` → `ErrInvalidRHSTab`, empty string

#### `TestRHSJQLResolveTabsAssignedAlwaysFirst`

Regardless of configured order, index 0 is Assigned. If configured already
contains an Assigned entry, the result still has **exactly one** Assigned,
at index 0.

#### `TestRHSJQLResolveTabsHidesVanishedStatusAndCategory`

Use slices matching the Phase 2 fixtures (or unmarshal them):

- statuses: `id=3`, `id=10001`
- categories: `undefined`, `new`, `indeterminate`, `done`
- configured:
  1. `{status, id: "99999", name: "Gone"}` → hidden
  2. `{category, key: "not-a-real-key", name: "Bogus"}` → hidden
  3. `{category, key: "indeterminate", name: "In Progress"}` → kept
  4. `{status, id: "10001", name: "Backlog"}` → kept
  5. `{Kind: "nope"}` → hidden

Want, in order: Assigned, In Progress, Backlog.

#### `TestRHSJQLResolveTabsEmptyConfigUsesDefaultSeed`

Subtests:

- `configured == nil` and `configured == []RHSTabEntry{}` → Assigned +
  In Progress (`key=indeterminate`), when `categories` includes
  `indeterminate`.
- `configured == nil` and `categories == nil` → **Assigned only** (seed In
  Progress is hidden; Assigned is never hidden).

#### JQL test imports

```go
import (
	"errors"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

`errors` is used if you prefer `errors.Is`; `assert.ErrorIs` is enough and
then `errors` can be omitted (`goimports` will drop it).

Expected `TestRHSJQL*` set: **10** tests
(`BuildExactStrings`, `AssignedErrorsWhenDoneOmittedFromValidKeys`,
`CategoryErrorsWhenKeyMissingFromValidKeys`, `RejectsSortOutsideAllowlist`,
`AssignedExcludesDoneMembership`, `NegatedBogusOperandWidensToIncludeDone`,
`UnknownKindAndEmptyStatusID`, `ResolveTabsAssignedAlwaysFirst`,
`ResolveTabsHidesVanishedStatusAndCategory`,
`ResolveTabsEmptyConfigUsesDefaultSeed`).

---

### `server/rhs_cache_test.go`

Create. `package main`. Header as in §9. **Every test name starts with
`TestRHSCache`.**

Helpers:

```go
type countingRHSStatusClient struct {
	mu          sync.Mutex
	statusCalls int
	catCalls    int
	sleep       time.Duration
	statuses    []*JiraStatus
	categories  []*JiraStatusCategory
	statusErr   error
	catErr      error
}

func (c *countingRHSStatusClient) ListStatuses() ([]*JiraStatus, error) {
	c.mu.Lock()
	c.statusCalls++
	c.mu.Unlock()
	if c.sleep > 0 {
		time.Sleep(c.sleep)
	}
	if c.statusErr != nil {
		return nil, c.statusErr
	}
	return c.statuses, nil
}

func (c *countingRHSStatusClient) ListStatusCategories() ([]*JiraStatusCategory, error) {
	c.mu.Lock()
	c.catCalls++
	c.mu.Unlock()
	if c.catErr != nil {
		return nil, c.catErr
	}
	return c.categories, nil
}

func (c *countingRHSStatusClient) counts() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.statusCalls, c.catCalls
}

func fixtureStatusesAndCategories(t *testing.T) ([]*JiraStatus, []*JiraStatusCategory) {
	t.Helper()
	var statuses []*JiraStatus
	require.NoError(t, json.Unmarshal(loadRHSTestdata(t, "rhs-status.json"), &statuses))
	var categories []*JiraStatusCategory
	require.NoError(t, json.Unmarshal(loadRHSTestdata(t, "rhs-statuscategory.json"), &categories))
	return statuses, categories
}
```

`loadRHSTestdata` is already in `client_cloud_rhs_test.go` (same package).

Construct the plugin with `setupTestPlugin` for tests that need
`OnConfigurationChange`. For get/invalidate-only tests,
`p := setupTestPlugin(api)` still works; you may also use `&Plugin{}` because
locks are zero-value safe. Prefer `setupTestPlugin` so LogError/LogDebug are
mocked if anything logs.

Test instance id: `types.ID("https://a.example.atlassian.net")`.

#### `TestRHSCacheTTLIsOneHour`

```go
assert.Equal(t, time.Hour, rhsStatusCacheTTL)
```

#### `TestRHSCacheFreshHitDoesNotRefetch`

1. First `getInstanceStatuses` — `statusCalls == 1`, `catCalls == 1`,
   returned slices match fixtures (`len` 2 and 4; first status id `"3"`;
   category keys include `done`).
2. Second call, same id — counts still 1 and 1; result still populated.

#### `TestRHSCacheExpiredEntryRefetches`

Populate `p.rhsStatusCache[id]` under the lock with a valid entry whose
`fetchedAt` is `time.Now().Add(-2 * time.Hour)`. Call
`getInstanceStatuses`. Counts == 1. Map entry’s `fetchedAt` is recent
(`time.Since < time.Minute`).

#### `TestRHSCacheInvalidateEmptiesMap`

Put two instance ids in the map. Call `invalidateRHSStatusCache()`.
`assert.Empty(t, p.rhsStatusCache)` (or `len == 0`). A following
`getInstanceStatuses` for either id refetches.

#### `TestRHSCacheConcurrentWarmDoesNotDoubleFetch`

`sleep: 50 * time.Millisecond` on the fake. Two goroutines call
`getInstanceStatuses` for the **same** id. `sync.WaitGroup`. After both
return: `statusCalls == 1` and `catCalls == 1`. Both results non-nil.

Do not `t.Parallel()`.

#### `TestRHSCacheFetchErrorDoesNotStore`

`statusErr` or `catErr` set. `getInstanceStatuses` returns that error.
Map does not gain an entry (or remains empty). A retry with a healthy
client then stores.

#### `TestRHSCacheOnConfigurationChangeEmptiesAllInstances`

This is the DoD “emptied by `OnConfigurationChange`” proof — not
inspect-only.

```go
func TestRHSCacheOnConfigurationChangeEmptiesAllInstances(t *testing.T) {
	api := &plugintest.API{}
	p := setupTestPlugin(api)

	max := int64(100 * 1024 * 1024)
	api.On("GetConfig").Return(&model.Config{
		FileSettings: model.FileSettings{MaxFileSize: &max},
	})
	api.On("LoadPluginConfiguration", mock.Anything).Return(nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return()

	idA := types.ID("https://a.example.atlassian.net")
	idB := types.ID("https://b.example.atlassian.net")
	p.rhsStatusCacheLock.Lock()
	p.rhsStatusCache = map[types.ID]*rhsStatusCacheEntry{
		idA: {fetchedAt: time.Now()},
		idB: {fetchedAt: time.Now()},
	}
	p.rhsStatusCacheLock.Unlock()

	require.NoError(t, p.OnConfigurationChange())
	assert.Empty(t, p.rhsStatusCache)
}
```

If `OnConfigurationChange` fails on a missing mock, add **only** the mock
the error names. Do not skip this test.

Leave `EnableAutocomplete` as-is so `:315-324` does not run.

#### Cache test imports

```go
import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)
```

Expected `TestRHSCache*` set: **7** tests.

---

## File-by-file change list (current line numbers at `d34ccb2`)

| File | Action | Current lines | What |
|------|--------|---------------|------|
| `server/rhs_types.go` | **Modify (append)** | **1–83** unchanged. Append after `ErrRateLimited` (**83**) | Four sentinels |
| `server/rhs_jql.go` | **Create** | n/a | `validateStatusCategoryOperand`, `buildTabJQL`, `resolveTabs`, `validCategoryKeysFrom` |
| `server/rhs_jql_test.go` | **Create** | n/a | Ten `TestRHSJQL*` tests, including #36 omit-`done` + membership |
| `server/rhs_cache.go` | **Create** | n/a | TTL, entry, `rhsStatusLister`, double-check `getInstanceStatuses`, `invalidateRHSStatusCache` |
| `server/rhs_cache_test.go` | **Create** | n/a | Seven `TestRHSCache*` tests |
| `server/plugin.go` | **Modify** | Insert fields after **174** (`teamFieldCacheLock`). Call invalidate after **310** (`updateConfig` block). Init map after **352** | Fields + call site + `OnActivate` |

**Do not modify:**

| File | Current lines (do not touch) | Why |
|------|------------------------------|-----|
| `server/client_cloud_rhs.go` | `ListStatuses` **70–76**; `ListStatusCategories` **78–84**; `SearchJQL` **86–88** | Cache calls them; do not change |
| `server/client.go` / `client_cloud.go` | shared `Client` tree | No widening |
| `server/rhs_types.go` Phase 1/2 types | **1–82** | Append only |
| `server/rhs_types_test.go` | `TestRHSConfig*` | Phase 1 |
| `server/http.go` | routes | Phase 4 |
| `server/user.go` | settings-info | Phase 1 |
| `plugin.json` | | Phase 1 |
| `webapp/**` | | Milestone 2 |
| `OnConfigurationChange` TeamIDs block | **253–304** | Do not hang invalidation off line 300 |

---

## Commands

From the worktree root
`~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`:

```bash
# If compile fails on missing gitignored server/manifest.go:
make apply

# Phase 3 JQL + #36 tests
cd server && go test ./... -run TestRHSJQL -v

# Phase 3 cache tests
cd server && go test ./... -run TestRHSCache -v

# Combined (also matches TestRHSConfig from Phase 1 — that is fine)
cd server && go test ./... -run 'TestRHSJQL|TestRHSCache' -v
```

Expected new passing names: 10 `TestRHSJQL*` + 7 `TestRHSCache*`.

Do not treat `make check-style` as a Phase 3 requirement if it pulls in
untouched `webapp/` lint. `gofmt` / `go vet` on `./server` plus the tests
above are the implementer bar. Gate 3 is the #36 execution, not lint.

---

## Gate 3 checklist (reviewer) — **execute, do not inspect**

From `IMPL_ORCHESTRATION_PLAN.md` Step 3. A skipped mitigation is invisible
at runtime. These steps are not optional. Run them from the worktree root
unless noted.

### 1. Every `statusCategory` construction sits downstream of validation

```bash
rg -n "statusCategory" server --glob '*.go'
```

Classify every hit:

| Kind of hit | Allowed locations |
|-------------|-------------------|
| JQL **construction** (`statusCategory =` / `statusCategory !=`) | **Only** `server/rhs_jql.go` inside `buildTabJQL`, each immediately after `validateStatusCategoryOperand` returns nil |
| Assigned operand | Must be `statusCategoryKeyDone` / emitted `done`. Must **not** skip the validator |
| JSON tag `json:"statusCategory"` | `server/rhs_types.go` (Phase 2 type) — not a construction site |
| Test fixtures / interpreter / assertions | `*_test.go` — expected |
| Comments | Fine |

There must be **no** `statusCategory` concatenation in `rhs_cache.go` or
`plugin.go`. Assigned must not be special-cased around the check (no
`if kind != assigned { validate(...) }`).

Also:

```bash
rg -n "validateStatusCategoryOperand" server/rhs_jql.go
```

Must be called on **both** the Assigned branch (operand
`statusCategoryKeyDone`) and the category branch (operand `tab.Key`).

### 2. Temporary validation removal — suite must FAIL

In `validateStatusCategoryOperand`, **temporarily** replace the body with:

```go
func validateStatusCategoryOperand(operand string, validCategoryKeys map[string]bool) error {
	return nil
}
```

Then:

```bash
cd server && go test ./... -run TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys -v
```

**Expected: FAIL.** The test must error (it asserts an error when `done` is
omitted). If this test still **passes**, the mitigation is untested — T3.4
is incomplete. Do not “fix” the test during this step.

`TestRHSJQLCategoryErrorsWhenKeyMissingFromValidKeys` should also FAIL
under the same stub. The Assigned test is the required one.

**Restore the original function body** immediately after. Re-run:

```bash
cd server && go test ./... -run 'TestRHSJQL|TestRHSCache' -v
```

Expected: PASS.

### 3. Negated-clause test asserts membership

```bash
rg -n "TES-DONE|NotContains|AssignedExcludesDoneMembership" server/rhs_jql_test.go
```

Confirm **all** of:

- [ ] `TestRHSJQLAssignedExcludesDoneMembership` exists (exact name).
- [ ] It asserts `assert.NotContains(..., "TES-DONE")` (or equivalent
      membership on keys / a Done-category issue).
- [ ] It also asserts a non-Done issue **is present** (`TES-NEW` / `TES-IP`).
- [ ] It does **not** treat success as HTTP 200, `err == nil` alone, or
      `assert.NotEmpty` / `len > 0` alone.
- [ ] `TestRHSJQLNegatedBogusOperandWidensToIncludeDone` exists and asserts
      `TES-DONE` **is present** for `statusCategory != bogusCategoryXYZ`
      (documents the Cloud trap the membership test is guarding).

### 4. Mechanical checks

- [ ] `cd server && go test ./... -run 'TestRHSJQL|TestRHSCache' -v` passes
      with the real validator restored.
- [ ] `buildTabJQL` exact-string table covers all three kinds × both sorts
      and includes `, key ASC`.
- [ ] `TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys` exists and uses
      a map that **omits** `done`.
- [ ] `resolveTabs`: Assigned always first; vanished status id / category
      key hidden; nil/empty config → Assigned + In Progress.
- [ ] Cache: fresh hit does not re-fetch; expired re-fetches; invalidate
      empties the map; concurrent warm does not double-fetch;
      `OnConfigurationChange` empties **all** instances
      (`TestRHSCacheOnConfigurationChangeEmptiesAllInstances`).
- [ ] `invalidateRHSStatusCache()` is called after `updateConfig` at
      `plugin.go` ~311, **not** at the TeamIDs warn on line 300.
- [ ] `rhsStatusCacheTTL == time.Hour`.
- [ ] `webapp/` diff is empty. No new routes. No `SearchJQL` call from
      cache/JQL files.
- [ ] `git grep -n "Issue.Search" server/` still only `server/client.go`.

**Commit checkpoint (orchestration):** local commit of this phase **only
after the three #36 verifications above have been executed**. Do not push.

---

## What this phase hands to Phase 4

Phase 4 handlers will:

1. Resolve a Cloud client and type-assert to
   `rhsCloudClient` (includes `ListStatuses`, `ListStatusCategories`,
   `SearchJQL`).
2. `entry, err := p.getInstanceStatuses(instance.GetID(), cloudClient)`.
3. `tabs := resolveTabs(conf.RHSStatusTabs[string(instance.GetID())], entry.statuses, entry.categories)`.
4. `jql, err := buildTabJQL(selectedTab, sort, validCategoryKeysFrom(entry.categories))`.
5. `SearchJQL(CloudSearchParams{JQL: jql, Fields: …, MaxResults: 20, NextPageToken: …})`.

Map `ErrInvalidStatusCategory`, `ErrInvalidRHSSort`, `ErrUnknownRHSTabKind`,
`ErrInvalidRHSTab` to JSON `invalid_request`. Map `ErrRateLimited` as
already planned.

The webapp must render the server-resolved `tabs` array (W-D4) — vanished
entries already hidden, virtual seed already applied. It must not re-derive
tabs or JQL.

Browse URLs and HTTP routes are still unbuilt.

---

## Official / spike URLs cited

- Status categories (silent-widen): `planner/projects/jira-plugin-rhs/context.md`
  § Status categories
- Spec tabs/JQL: `spec.md` § "Tabs and JQL"
- Spike: `spikes/status-tab-jql/README.md` probes 16–17 (`=` bogus → empty),
  20–21 (`!=` bogus → everything), 30–32 (`done` / `Done` / `3` resolve)
- Cloud status categories:
  https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-workflow-status-categories/#api-rest-api-3-statuscategory-get
- Real keys (docs examples are dummy):
  https://community.developer.atlassian.com/t/bad-documentation-for-rest-api-3-statuscategory/78565

---

## Implementation Summary

Implemented by IE3 (2026-08-19). Did not commit. Did not push. Did not start Phase 4.

### Files created/modified

| File | Action |
|------|--------|
| `server/rhs_types.go` | Modified — appended `ErrInvalidStatusCategory`, `ErrInvalidRHSSort`, `ErrUnknownRHSTabKind`, `ErrInvalidRHSTab` after `ErrRateLimited` |
| `server/rhs_jql.go` | Created — `validCategoryKeysFrom`, `validateStatusCategoryOperand`, `buildTabJQL`, `resolveTabs` |
| `server/rhs_jql_test.go` | Created — 10 `TestRHSJQL*` tests including #36 omit-`done` + membership |
| `server/rhs_cache.go` | Created — TTL, entry, `rhsStatusLister`, double-check `getInstanceStatuses`, `invalidateRHSStatusCache` |
| `server/rhs_cache_test.go` | Created — 7 `TestRHSCache*` tests |
| `server/plugin.go` | Modified — cache fields, `OnActivate` map init, invalidate after `updateConfig` |

### Test results

`cd server && go test ./... -run 'TestRHSJQL|TestRHSCache' -v` — **PASS** (10 `TestRHSJQL*` + 7 `TestRHSCache*`).

`go vet ./server` — **PASS**.

### Gate 3 self-run results

1. **Construction sites.** `statusCategory` JQL concatenation exists only in `server/rhs_jql.go` `buildTabJQL`: Assigned (`statusCategory !=` + `statusCategoryKeyDone`) and category (`statusCategory =` + `tab.Key`), each immediately after `validateStatusCategoryOperand` returns nil. Assigned is not special-cased around the check. JSON tag / constants / tests / comments only elsewhere. No `statusCategory` in `rhs_cache.go` or `plugin.go`.
2. **Temporary validation removal.** Stubbed `validateStatusCategoryOperand` to `return nil`. `go test ./... -run TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys -v` **FAILED** as required: `rhs_jql_test.go:150` — `An error is expected but got nil` (`TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys`). Restored original body. Combined `TestRHSJQL|TestRHSCache` suite **PASS**.
3. **Membership.** `TestRHSJQLAssignedExcludesDoneMembership` asserts `assert.NotContains(..., "TES-DONE")` plus `TES-NEW` / `TES-IP` / `TES-UNDEF` present. `TestRHSJQLNegatedBogusOperandWidensToIncludeDone` asserts `TES-DONE` **is present** for `statusCategory != bogusCategoryXYZ`.

### Deviations

None.

### Current `plugin.go` line numbers

- Cache fields: `rhsStatusCache` / `rhsStatusCacheLock` at **176–177** (immediately after `teamFieldCacheLock` at 174).
- `OnActivate` map init: **358** (`p.rhsStatusCache = make(...)` after `teamFieldCache` at 357).
- Invalidation call: **`p.invalidateRHSStatusCache()` at 315**, immediately after `updateConfig` (**310–313**). TeamIDs `Log.Warn` remains at **303** — invalidation is not there.
