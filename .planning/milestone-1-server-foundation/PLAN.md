# Implementation Plan: Show your tickets in the RHS — Phase 1 (Server Foundation)

> Server-side groundwork for a Cloud-only "my Jira tickets" right-hand sidebar: config
> plumbing, a Cloud `/search/jql` client with rate-limit handling, a per-tab JQL builder,
> and two new HTTP endpoints. Ships nothing user-visible.

## Metadata

- **Spec:** `planner/projects/jira-plugin-rhs/ideas/001-show-tickets-rhs/spec.md` (§ "Phase 1 — Server foundation")
- **Target repo:** https://github.com/mattermost/mattermost-plugin-jira
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs` (clean, zero commits ahead of `master`)
- **Generated:** 2026-08-18
- **Status:** draft

## Scope Boundary

**In scope:** everything under `server/` plus `plugin.json`. Phase 1 is deliberately a
foundation with its own test surface — no App Bar, no RHS component, no System Console
component.

**Out of scope (milestone 2):** all of `webapp/` except nothing at all — Phase 1 touches
zero webapp files. The `CLOUD_OAUTH` enum fix, the `hasCloudInstance` selector, the
`test-utils.tsx` fixture rename, the admin picker component, and the RHS UI are all
milestone 2.

**Explicitly not touched:** `SearchIssues` (`server/client.go:304-314`), the
`get-search-issues` route, and all 13 existing `getClient` call sites. The pre-existing
Cloud 410 on `rest/api/2/search` is a separate bug and is not fixed here.

## Architecture Overview

`server/` is a single flat `package main`; there is no convention for feature
sub-packages. All new code lands as new `.go` files in `server/` prefixed `rhs_`, which
keeps the feature reviewable in isolation and keeps existing files nearly untouched.

Six new files carry the feature:

| File | Responsibility |
|------|----------------|
| `server/rhs_types.go` | Config value types, tab types, the normalized issue DTO, error codes |
| `server/client_cloud_rhs.go` | Cloud-only client methods: `3/status`, `3/statuscategory`, `3/search/jql` + retry |
| `server/rhs_cache.go` | Per-instance in-memory status/category cache, 1-hour TTL |
| `server/rhs_jql.go` | Tab resolution (vanished-tab hiding) and JQL construction with operand validation |
| `server/rhs.go` | Business logic: resolve client, resolve tabs, search, normalize to DTO |
| `server/rhs_http.go` | JSON error convention, admin gate, the two handlers |

Five existing files receive small, additive edits: `plugin.json`, `server/plugin.go`,
`server/user.go`, `server/http.go`, and (cache invalidation only) `server/plugin.go`
again. `server/plugin.go` is touched by two phases — see the conflict note in the file
change map.

Three architectural decisions shape the plan:

1. **No new interface on the shared `Client` tree.** `Client`
   (`server/client.go:35-50`) composes five service interfaces used by both Cloud and
   Server/DC clients. The RHS methods are Cloud-only, so they go on `jiraCloudClient`
   (`server/client_cloud.go:16-31`) and are reached by type-asserting against a small
   local interface. Adding them to `SearchService` would force stub implementations onto
   `jiraServerClient` for a feature that never runs there.

2. **A new raw-response client method is unavoidable.** Neither `RESTGet`
   (`server/client.go:99-119`) nor `RESTGetRaw` (`:121-152`) returns the
   `*http.Response`, so neither can read `Retry-After` or `RateLimit-Reason`. The search
   method calls `client.Jira.NewRequest(...)` then `client.Jira.Do(...)` and inspects
   headers before decoding. `3/status` and `3/statuscategory` need no headers and use
   plain `RESTGet`.

3. **Tab config is read-only to the server.** It arrives through
   `LoadPluginConfiguration` as a native JSON value on a real typed struct field. The
   server never writes plugin config, which is why the "seed" is virtual — see
   Decision D2.

### Decisions this plan makes (and why)

**D1 — Config key casing.** `LoadPluginConfiguration`
(`mattermost-server/server/channels/app/plugin_api.go:50-82`) builds an intermediate map
with `finalConfig[strings.ToLower(setting.Key)] = value`, then does a
`json.Marshal` → `json.Unmarshal` round-trip into the destination struct. **Every
top-level `json:` tag on `externalConfig` must therefore be all-lowercase.** A
`json:"rhsStatusTabs"` tag would silently never populate — the field stays nil, no error
is logged, and the feature appears to have no config. Only the top-level key is
lowercased; nested fields inside the stored JSON value keep their casing, so
`RHSTabEntry` sub-fields can use normal camelCase tags.

**D2 — Seeding is virtual, not written.** The spec requires new Cloud instances to
behave as if seeded with Assigned + In Progress, with no admin Jira connection required.
Two options existed: write the seed into plugin config at `InstallInstance`
(`server/instances.go:145-178`), or treat absent/empty config as the seed at read time.
**This plan takes read-time (virtual) seeding.** Writing config from the server would
mean a `SavePluginConfig` call on the instance-install path — a config write on every
node, racy under HA, re-entrant with `OnConfigurationChange`, and contrary to the spec's
"no admin write endpoint" posture. Virtual seeding needs no writes and cannot race.

The cost lands in milestone 2: the System Console picker will receive an empty `value`
for a never-configured instance, so **the Phase 2 component must render Assigned (locked)
and In Progress as selected when `value` is empty, without calling `onChange`.** That
constraint is recorded here because Phase 1 is what makes it necessary.

**D3 — Cache invalidation is coarse.** `OnConfigurationChange` fires on every config
save, not only tab-config saves. Rather than diff old against new tab config, the whole
RHS status cache is dropped on any config change. Config saves are rare; a diff is extra
state and an extra failure mode for no practical gain.

### The trap this plan is built around (risk #36)

Jira Cloud **never errors** on a `statusCategory` operand it cannot resolve:

| Clause | Unresolvable operand |
|--------|---------------------|
| `statusCategory = <bogus>` | 200, matches **nothing** |
| `statusCategory != <bogus>` | 200, matches **everything** |

The always-present **Assigned** tab is the negated form
(`statusCategory != done`). A typo or a bad edit there does not fail — it silently widens
the tab to include Done tickets, returns 200, returns a non-empty list, and looks like
working software. Neither manual testing nor a naive integration test catches it.

Two mitigations are **mandatory** and appear as named tasks: build-time operand
validation (T3.2) and membership-based test assertions (T3.4, T5.2). Because nothing
fails loudly when they are skipped, Review Gate 3 includes explicit verification steps
that a reviewer runs rather than trusts.

---

## Phases

### Phase 1: Config foundation and types

**Goal:** `EnableJiraRHS` and the per-instance tab config load correctly from plugin
config and reach the webapp, with the value types the rest of the feature builds on.
Nothing calls Jira yet.

**Depends on:** none.

#### Tasks

- [ ] **1.1 Define the RHS value types**
  - **Files:** `server/rhs_types.go`
  - **Action:** Create
  - **Details:** Declare the types every later phase depends on. `RHSTabKind` as a string
    type with constants `RHSTabKindCategory = "category"` and `RHSTabKindStatus =
    "status"`. `RHSTabEntry struct { Kind RHSTabKind \`json:"kind"\`; Key string
    \`json:"key,omitempty"\`; ID string \`json:"id,omitempty"\`; Name string
    \`json:"name"\` }` — `Key` carries the status-category key for category entries, `ID`
    the numeric status id for status entries. These are nested fields inside the stored
    JSON value, so camelCase-style tags are fine here (see D1 — only top-level
    `externalConfig` tags must be lowercase).
    Also declare the four canonical status-category keys as constants
    (`statusCategoryKeyNew = "new"`, `Indeterminate = "indeterminate"`, `Done = "done"`,
    `Undefined = "undefined"`) — do **not** hardcode the numeric ids anywhere; the ids
    (2/4/3/1) are counterintuitive and the key is what gets used as the JQL operand.
    Declare `rhsDefaultTabs()` returning the virtual seed per D2: the locked Assigned tab
    plus one category entry `{Kind: category, Key: "indeterminate", Name: "In Progress"}`.

- [ ] **1.2 Add the config fields and the manifest entries**
  - **Files:** `server/plugin.go`, `plugin.json`
  - **Action:** Modify
  - **Details:** Add two fields to `externalConfig` (`server/plugin.go:52-103`), following
    the existing style where `EnableJiraUI bool \`json:"enablejiraui"\`` sits at the top:
    `EnableJiraRHS bool \`json:"enablejirarhs"\`` and
    `RHSStatusTabs map[string][]RHSTabEntry \`json:"rhsstatustabs"\`` (map key is the
    Cloud instance id).
    **Both tags must be all-lowercase — see D1.** A camelCase tag fails silently: the
    field stays nil on every load with no error anywhere.
    In `plugin.json`, insert the bool entry immediately after `EnableJiraUI` (currently
    ending line 33) with `"key": "EnableJiraRHS"`, `"type": "bool"`, `"default": false`,
    and help text noting it is Jira Cloud only. Append the custom entry before the closing
    `]` at line 160: `{"key": "RHSStatusTabs", "type": "custom", "display_name": "Jira
    RHS status tabs"}` — no `default`, matching the shape used by
    `mattermost-plugin-custom-attributes` and `mattermost-plugin-ai`. The key must be
    declared here or the System Console will not render the Phase 2 component at all.
    Do **not** add any parsing to `OnConfigurationChange`: `LoadPluginConfiguration`
    already unmarshals the native JSON value straight into the typed field. The
    `TeamIDs` → `TeamIDList` string-parsing precedent at `server/plugin.go:247-298` is the
    wrong pattern to copy here.

- [ ] **1.3 Expose the RHS flag to the webapp**
  - **Files:** `server/user.go`
  - **Action:** Modify
  - **Details:** Add `RHSEnabled bool \`json:"rhs_enabled"\`` to the anonymous response
    struct in `httpGetSettingsInfo` (`server/user.go:219-228`), sourced from
    `conf.EnableJiraRHS`. This is the only HTTP config exposure point in the plugin;
    milestone 2's registration gate reads it exactly as `plugin.tsx` reads `ui_enabled`.

- [ ] **1.4 Config load tests**
  - **Files:** `server/rhs_types_test.go`
  - **Action:** Create
  - **Details:** Table test that a representative stored config map round-trips into
    `externalConfig` through the same `json.Marshal`/`json.Unmarshal` path
    `LoadPluginConfiguration` uses, with **lowercased top-level keys** — this is the test
    that would catch a wrong tag. Cover: a two-instance map with one category entry and
    one status entry; an absent key yielding a nil map; and `rhsDefaultTabs()` returning
    Assigned plus In Progress with key `indeterminate`.

#### Definition of Done

- [ ] `cd server && go test ./... -run TestRHSConfig -v` passes.
- [ ] A test asserts `RHSStatusTabs` populates from a **lowercased** top-level key and
      would fail if the tag were camelCase.
- [ ] `plugin.json` remains valid JSON and `make check-style` passes.
- [ ] `httpGetSettingsInfo` returns `rhs_enabled` — asserted by a handler test hitting
      the existing settings-info route.

---

### Phase 2: Cloud client extension

**Goal:** Three new Cloud-only client methods — instance-wide statuses, status
categories, and `/search/jql` with full rate-limit handling — each unit-tested against a
local HTTP test server.

**Depends on:** Phase 1 (uses no config, but shares the type file).

#### Tasks

- [ ] **2.1 Status and status-category listing**
  - **Files:** `server/client_cloud_rhs.go`
  - **Action:** Create
  - **Details:** Add two methods on `jiraCloudClient` (`server/client_cloud.go:16-31`).
    `ListStatuses() ([]*JiraStatus, error)` calls `client.RESTGet("3/status", nil,
    &result)`; `endpointURL` (`server/client.go:497-509`) prepends `/rest/api/`, so this
    resolves correctly for both Cloud auth modes — `{site}.atlassian.net/...` for Connect
    JWT and `api.atlassian.com/ex/jira/{cloudid}/...` for OAuth2, with no branching.
    `ListStatusCategories() ([]*JiraStatusCategory, error)` calls
    `RESTGet("3/statuscategory", nil, &result)`. Declare `JiraStatus{ID, Name,
    StatusCategory{ID, Key, Name}}` and `JiraStatusCategory{ID, Key, Name}` in
    `server/rhs_types.go`.
    These are net-new: `ListProjectStatuses` (`server/client.go:64`, impl
    `server/client_cloud.go:143-150`) is per-project and grouped by issue type, and there
    is no instance-wide listing today.

- [ ] **2.2 The `/search/jql` method with retry and rate-limit handling**
  - **Files:** `server/client_cloud_rhs.go`
  - **Action:** Extend
  - **Details:** Add `SearchJQL(params CloudSearchParams) (*CloudSearchResult, error)`
    to `jiraCloudClient`. `CloudSearchParams{JQL, Fields []string, MaxResults int,
    NextPageToken string}`; `CloudSearchResult{Issues []jira.Issue, NextPageToken string,
    IsLast bool}` — note Cloud returns **no `total`**, which is why the UI gets Load more
    rather than "N of M".
    Issue a GET to `3/search/jql`. It must **not** use `client.Jira.Issue.Search` — that
    is `SearchIssues` (`server/client.go:304-314`) over go-jira's hardcoded
    `rest/api/2/search`, which Atlassian sunset and which returns a live **410 Gone** on
    Cloud.
    Because `RESTGet` does not surface the `*http.Response`, build the request with
    `client.Jira.NewRequest(http.MethodGet, url, nil)` and call `client.Jira.Do(req, nil)`,
    which returns `(*jira.Response, error)` embedding `*http.Response`. Read
    `Retry-After` and `RateLimit-Reason` from `resp.Header` before decoding the body.
    Retry policy, exactly as specified: base 2s, double per 429, cap ~30s, jitter
    ×0.7–1.3, **at most 4 attempts**. Prefer the `Retry-After` value when present.
    **Do not retry when `RateLimit-Reason` is `jira-quota-global-based`** — the hourly
    points quota is spent and retrying only burns more. Return a sentinel
    `ErrRateLimited` (declared in `server/rhs_types.go`) when retries are exhausted or
    when a global-quota 429 arrives, so the HTTP layer can map it to a distinct
    `rate_limited` response instead of a 500.
    **Lint:** `bodyclose` and `errcheck` are both enabled in `.golangci.yml`. This is the
    only new code holding a raw `*http.Response`, so close the body on every path
    including the retry and error paths — `defer func() { _ = resp.Body.Close() }()`.

- [ ] **2.3 Client tests against a local test server**
  - **Files:** `server/client_cloud_rhs_test.go`
  - **Action:** Create
  - **Details:** Drive the three methods against `httptest.NewServer` handlers.
    Cover: a happy-path `/search/jql` decode including `nextPageToken` and `isLast`;
    cursor pass-through on the follow-up request; a 429 with `Retry-After` that succeeds
    on retry; a 429 with `RateLimit-Reason: jira-quota-global-based` that returns
    `ErrRateLimited` **after exactly one attempt** (assert the server saw one request —
    this is the assertion that proves the no-retry rule); retry exhaustion after 4
    attempts; and `3/status` / `3/statuscategory` decoding. Keep backoff fast in tests by
    making the base delay injectable rather than sleeping real seconds.
    Add JSON fixtures under `server/testdata/` — it currently holds webhook fixtures only,
    no search fixtures.

#### Definition of Done

- [ ] All three methods have passing tests; `cd server && go test ./... -run TestCloudRHS -v`.
- [ ] A test proves a `jira-quota-global-based` 429 issues **exactly one** upstream
      request.
- [ ] A test proves `Retry-After` is honored and that attempts cap at 4.
- [ ] `golangci-lint run ./...` is clean — specifically no `bodyclose` finding on the new
      raw-response code.
- [ ] `git grep -n "Issue.Search" server/` shows no new call sites.

---

### Phase 3: Domain logic — cache, tab resolution, JQL

**Goal:** Given an instance and a tab, produce a validated JQL string and a resolved tab
list, backed by a 1-hour status cache. This phase contains the #36 mitigation and is the
highest-risk phase in the plan.

**Depends on:** Phases 1 and 2.

#### Tasks

- [ ] **3.1 Per-instance status cache**
  - **Files:** `server/rhs_cache.go`, `server/plugin.go`
  - **Action:** Create + Modify
  - **Details:** Add to the `Plugin` struct (`server/plugin.go:126-169`), mirroring the
    existing `teamFieldCache map[types.ID]map[string]struct{}` /
    `teamFieldCacheLock sync.RWMutex` precedent: `rhsStatusCache
    map[types.ID]*rhsStatusCacheEntry` and `rhsStatusCacheLock sync.RWMutex`.
    `rhsStatusCacheEntry{statuses []*JiraStatus, categories []*JiraStatusCategory,
    fetchedAt time.Time}`, TTL const 1 hour.
    In `server/rhs_cache.go`, implement `getInstanceStatuses(instanceID, client)` — return
    the cached entry when fresh, otherwise fetch both lists via T2.1 and store them.
    Guard with the read lock, then re-check under the write lock so a concurrent warm
    does not double-fetch.
    Add `invalidateRHSStatusCache()` and call it from `OnConfigurationChange`
    (`server/plugin.go:192-326`), after `p.updateConfig(...)` at :300. Per **D3** this
    drops **all** entries on any config change rather than diffing tab config — config
    saves are rare and a diff buys nothing. `OnConfigurationChange` runs on every node, so
    HA stays consistent for admin edits.
    Known and accepted: a status renamed or deleted **in Jira** can stay cached for up to
    an hour, and each node warms its own copy. The ticket list itself is never cached.

- [ ] **3.2 JQL builder with mandatory category-operand validation** ⚠️ **risk #36**
  - **Files:** `server/rhs_jql.go`
  - **Action:** Create
  - **Details:** Implement
    `buildTabJQL(tab RHSTabEntry, sortField string, validCategoryKeys map[string]bool)
    (string, error)`.
    Clauses, all scoped to the current user as assignee — status narrows "my tickets" and
    must never widen to reporter or watched:

    | Tab | JQL |
    |-----|-----|
    | Assigned (always present) | `assignee = currentUser() AND statusCategory != done` |
    | Category entry | `assignee = currentUser() AND statusCategory = <tab.Key>` |
    | Status entry | `assignee = currentUser() AND status = <tab.ID>` |

    The stored category key is a valid Cloud JQL operand and is passed through
    **unchanged** — there is no `key → numeric id` translation step and no
    `/statuscategory` lookup on the query path (spike-verified for `new`,
    `indeterminate`, and `done`).

    **The mitigation, which is the whole point of this task:** before emitting any
    `statusCategory` clause — **including the hardcoded `!= done` in the Assigned tab** —
    assert the operand is present in `validCategoryKeys`, and return an error if it is
    not. Do not rely on Jira to reject a bad value; it will not. `= <bogus>` returns 200
    matching nothing, and `!= <bogus>` returns 200 matching **everything**, so an
    unvalidated Assigned tab silently starts including Done tickets. Validating the
    literal `done` looks redundant and is not: it is the exact clause whose failure is
    invisible, and the check also guarantees the operand came from the canonical list
    rather than from anything user-influenced.

    Append `ORDER BY updated DESC, key ASC` or `ORDER BY created DESC, key ASC` from the
    sort control. The secondary `key ASC` is required for cursor stability and is
    spike-verified compatible with Cloud cursor pagination. Accept only `updated` and
    `created` from an allowlist and reject anything else — never interpolate the raw sort
    parameter into JQL.

- [ ] **3.3 Tab resolution and vanished-tab hiding**
  - **Files:** `server/rhs_jql.go`
  - **Action:** Extend
  - **Details:** Implement `resolveTabs(configured []RHSTabEntry, statuses []*JiraStatus,
    categories []*JiraStatusCategory) []RHSTab`. Assigned is always first and is never
    configurable or removable. A category entry survives if its `Key` matches the
    canonical category list; a status entry survives if its `ID` matches a status on the
    instance. Anything else is **hidden**, not rendered empty and not errored.
    When the instance has no configured entries, fall back to `rhsDefaultTabs()` — this is
    the virtual seed from **D2**, and it is what makes "seeded with no admin Jira
    connection" true without any config write.
    Hiding is a UX rule, not an error-prevention one: an unknown `status = "name"` returns
    200 empty on Cloud rather than 400, so the tab would simply look broken.

- [ ] **3.4 Domain tests, with membership assertions** ⚠️ **risk #36**
  - **Files:** `server/rhs_jql_test.go`, `server/rhs_cache_test.go`
  - **Action:** Create
  - **Details:** JQL table tests over all three tab kinds × both sorts, asserting exact
    output strings including the `, key ASC` secondary sort.
    **The mitigations, tested:** a case passing a `validCategoryKeys` set that omits
    `done` and asserting `buildTabJQL` returns an **error** for the Assigned tab — this is
    the test that fails if T3.2's validation is ever removed. A case passing a bogus
    category key and asserting an error. And a rejected-sort case.
    **Any test covering a negated category clause must assert on result membership, not
    on status code or non-emptiness.** The failure mode returns a *superset* with a 200,
    so "no error and some results" passes while the tab is wrong. For the Assigned tab,
    assert that a Done-category issue in the fake result set is **absent** from what the
    tab is expected to contain.
    Cache tests: fresh hit does not re-fetch, expired entry re-fetches, invalidation
    empties the map, and concurrent warm does not double-fetch.

#### Definition of Done

- [ ] `buildTabJQL` returns an error when the category operand is absent from the
      canonical list, **including for the Assigned tab's hardcoded `done`** — proven by a
      test, not by inspection.
- [ ] At least one test for a negated category clause asserts membership rather than
      status code or non-emptiness.
- [ ] Exact-string tests cover all three tab kinds × both sort orders, including
      `, key ASC`.
- [ ] Unknown status ids and unknown category keys are hidden from the resolved tab list;
      Assigned is always present and always first.
- [ ] An instance with no configured entries resolves to Assigned + In Progress.
- [ ] Cache honors the 1-hour TTL and is emptied by `OnConfigurationChange`.

---

### Phase 4: HTTP layer

**Goal:** Two working endpoints with a machine-readable JSON error convention and an
admin gate.

**Depends on:** Phases 1–3.

#### Tasks

- [ ] **4.1 JSON error convention and the admin gate**
  - **Files:** `server/rhs_http.go`
  - **Action:** Create
  - **Details:** The plugin has **no JSON error shape at all** — `respondErr`
    (`server/http.go:217-220`) calls `http.Error`, so every failure today is `text/plain`
    with a raw Go error string. Introduce, for the new routes only:
    `respondJSONErr(w http.ResponseWriter, code int, errorCode, message string) (int, error)`
    writing `{"error": "<code>", "message": "<human text>"}` with
    `Content-Type: application/json`. Error codes: `not_connected`, `rate_limited`,
    `not_authorized`, `not_cloud`, `invalid_request`, `internal_error`.
    **Leave all 13 existing `getClient` call sites and every existing handler untouched** —
    this convention applies to the new routes only.
    Add `checkSystemAdmin`, an inline guard following the codebase's only precedent at
    `server/subscribe.go:1076`: `p.client.User.HasPermissionTo(userID,
    model.PermissionManageSystem)`, returning `not_authorized` on failure. There is no
    admin middleware in this plugin and this plan does not add one — one helper called by
    one handler is the smaller change.
    Add a helper mapping domain errors to responses: `kvstore.ErrNotFound` from
    `LoadConnection` → `not_connected`; `ErrRateLimited` → `rate_limited`; a non-Cloud
    instance → `not_cloud`.

- [ ] **4.2 Cloud client resolution for both auth modes**
  - **Files:** `server/rhs.go`
  - **Action:** Create
  - **Details:** Declare a narrow local interface the Cloud client satisfies —
    `type rhsCloudClient interface { ListStatuses() ...; ListStatusCategories() ...;
    SearchJQL(...) ... }` — and resolve it two ways rather than widening the shared
    `Client` tree:
    *User path (RHS issues):* `p.getClient(instanceID, mattermostUserID)`
    (`server/issue.go:1568-1582`) then type-assert to `rhsCloudClient`; a failed assertion
    means a non-Cloud instance → `not_cloud`.
    *Admin path (status listing):* branch on instance type
    (`server/instance.go:14-18`). For `cloudInstance` (Connect JWT), use
    `getClientForBot()` (`server/instance_cloud.go:238-253`), which returns a raw
    `*jira.Client` and must be wrapped with `newCloudClient(...)`; the picker then works
    with **no personal connection at all**. For `cloudOAuthInstance` there is **no bot
    client** — only `getClientForConnection` — so load the requesting admin's own
    connection and return `not_connected` if absent. **Do not use the global
    `AdminEmail`/`AdminAPIToken`**: it is a single credential shared across instances and
    already misbehaves with more than one.

- [ ] **4.3 Issue normalization to the DTO**
  - **Files:** `server/rhs.go`
  - **Action:** Extend
  - **Details:** Map each `jira.Issue` to `RHSIssue{Key, Summary, BrowseURL, Status{Name,
    CategoryKey}, Priority, IssueType, Project, Assignee, Reporter, Created, Updated,
    DueDate, Labels}` (declared in `server/rhs_types.go`). Request exactly these fields on
    the search call: `summary`, `status`, `priority`, `issuetype`, `assignee`, `reporter`,
    `created`, `updated`, `duedate`, `project`, `labels`. Field count does not affect
    Cloud quota cost, so there is no reason to trim.
    **Build `BrowseURL` from `instance.GetJiraBaseURL()`, never `GetURL()`** — for
    `cloud-oauth`, `GetURL()` returns the `api.atlassian.com` API URL
    (`server/instance_cloud_oauth.go:228-235`) and yields links that 404 for the user.
    Carry `statusCategory.key` through to the DTO so milestone 2 can color the status pill
    from theme variables without a second lookup.
    This DTO lives on the **new** route only. `get-search-issues` keeps its raw
    `[]jira.Issue` contract: it has two consumers, and the create-issue epic selector
    (`webapp/src/components/jira_epic_selector.tsx:101-105`) reads a *dynamic* custom
    field id, which a fixed DTO cannot serve.

- [ ] **4.4 The two handlers and their routes**
  - **Files:** `server/rhs_http.go`, `server/http.go`
  - **Action:** Extend + Modify
  - **Details:** Add two route constants to the block at `server/http.go:28-71`, after
    `routeOAuth2Complete` at :71 — `routeAPIRHSIssues = "/rhs/issues"` and
    `routeAPIRHSStatuses = "/rhs/statuses"`. Register both on `apiRouter` inside
    `initializeRouter()` (`server/http.go:92-169`), after the subscription-template routes
    ending at :168, following the existing idiom:
    `apiRouter.HandleFunc(routeAPIRHSIssues, p.checkAuth(p.handleResponse(p.httpRHSGetIssues))).Methods(http.MethodGet)`.
    `httpRHSGetIssues`: read `instance_id`, tab identity, `sort` (default `updated`),
    optional `next_page_token`; page size fixed at 20. Resolve the client, load statuses
    from cache, resolve tabs, build JQL, search, normalize, and respond with
    `{issues, tabs, nextPageToken, isLast}`. Follow the handler shape of
    `httpGetSearchIssues` (`server/issue.go:903-916`): extract, delegate, respond, return
    `(int, error)`.
    `httpRHSListStatuses`: `checkSystemAdmin` first, then resolve the admin client per
    T4.2 and return the instance status list for the future picker.
    Both use `respondJSONErr` for every failure path — never a bare 500.

- [ ] **4.5 Handler tests**
  - **Files:** `server/rhs_http_test.go`
  - **Action:** Create
  - **Details:** Use the established pattern: `setupTestPlugin(api)`
    (`server/issue_test.go:99-114`), a handwritten `Client` fake in the shape of
    `testClient` (`server/issue_test.go:39-45`), `httptest.NewRecorder()` +
    `p.ServeHTTP(&plugin.Context{}, w, request)` with a `Mattermost-User-Id` header
    (`server/http_test.go:481-490`), and `makeTestKVStore` (`mock_kv_store.go:13`).
    Cover: happy path with the normalized DTO; cursor pass-through and `isLast`;
    unconnected user → **JSON** `not_connected` (assert the body and
    `Content-Type: application/json`, not just the status); `ErrRateLimited` →
    `rate_limited`; non-Cloud instance → `not_cloud`; a non-admin caller on
    `/rhs/statuses` → `not_authorized`; and an admin caller on a Connect JWT instance
    succeeding with **no personal connection present**.

#### Definition of Done

- [ ] Both routes respond; `cd server && go test ./... -run TestRHSHTTP -v` passes.
- [ ] Every failure path returns JSON with an `error` code and
      `Content-Type: application/json` — asserted, not assumed.
- [ ] A non-admin caller receives `not_authorized` on `/rhs/statuses`.
- [ ] A Connect JWT admin path succeeds with no personal connection stored.
- [ ] `git diff --stat master -- server/issue.go` shows the existing `getClient` call
      sites and `get-search-issues` unchanged.

---

### Phase 5: Quality gates and the #36 audit

**Goal:** The full suite and lint pass, and the two risk-#36 mitigations are verified to
be present rather than assumed.

**Depends on:** Phases 1–4.

#### Tasks

- [ ] **5.1 Full gate run**
  - **Files:** none (verification only)
  - **Action:** Verify
  - **Details:** Run `make test` and `make check-style` from the repo root. `check-style`
    is `go vet` + `golangci-lint run ./...` + `mattermost-govet`. Enabled linters include
    `bodyclose`, `errcheck`, `gosec`, `revive`, and `staticcheck` — the raw-response
    search method from T2.2 is the likeliest source of findings. Fix any finding in the
    new code; do not fix pre-existing findings in untouched files.

- [ ] **5.2 Risk #36 audit** ⚠️
  - **Files:** `server/rhs_jql.go`, `server/rhs_jql_test.go`
  - **Action:** Verify, and add tests if the audit finds gaps
  - **Details:** This exists because a skipped mitigation is invisible at runtime — the
    code compiles, the tests pass, the endpoint returns 200, and the Assigned tab quietly
    contains Done tickets. Do not accept "it looks right."
    1. `rg -n "statusCategory" server/` — every construction site must sit downstream of a
       validation call against the canonical category list. Confirm the Assigned tab's
       `!= done` is validated too, not special-cased around the check.
    2. Temporarily remove the validation and confirm `go test ./...` **fails**. If the
       suite still passes, the mitigation is untested and T3.4 is incomplete. Restore it.
    3. `rg -n "assert" server/rhs_jql_test.go` — confirm at least one negated-category
       test asserts on membership, not on status code or non-emptiness.

- [ ] **5.3 Record what Phase 1 hands to Phase 2**
  - **Files:** `.planning/PHASE1_HANDOFF.md`
  - **Action:** Create
  - **Details:** Short note capturing what milestone 2 needs: the two route shapes and
    their query params, the DTO field list, the JSON error codes and when each fires, the
    `rhs_enabled` settings key, the `RHSStatusTabs` config key and entry shape, and — most
    importantly — the **D2 consequence**: because seeding is virtual, the System Console
    picker must render Assigned (locked) and In Progress as selected when `value` is
    empty, and must not call `onChange` to materialize them.

#### Definition of Done

- [ ] `make test` passes.
- [ ] `make check-style` passes with no new findings.
- [ ] The #36 audit is complete, including the deliberate removal check in step 2.
- [ ] `.planning/PHASE1_HANDOFF.md` exists and covers the D2 constraint.

---

## File Change Map

| File | Phase(s) | Action | Summary |
|------|----------|--------|---------|
| `server/rhs_types.go` | 1, 2, 4 | Create | Tab types, category-key constants, `JiraStatus`/`JiraStatusCategory`, issue DTO, `ErrRateLimited` |
| `server/plugin.go` | **1, 3** | Modify | `externalConfig` fields (P1); cache fields + `OnConfigurationChange` invalidation (P3) |
| `plugin.json` | 1 | Modify | `EnableJiraRHS` bool + `RHSStatusTabs` custom setting |
| `server/user.go` | 1 | Modify | `rhs_enabled` in `httpGetSettingsInfo` |
| `server/rhs_types_test.go` | 1 | Create | Config round-trip incl. the lowercase-key regression test |
| `server/client_cloud_rhs.go` | 2 | Create | `ListStatuses`, `ListStatusCategories`, `SearchJQL` + retry |
| `server/client_cloud_rhs_test.go` | 2 | Create | httptest-driven client tests |
| `server/testdata/` | 2 | Create | Search / status JSON fixtures (none exist today) |
| `server/rhs_cache.go` | 3 | Create | 1-hour per-instance status cache |
| `server/rhs_jql.go` | **3, 5** | Create | Tab resolution + JQL builder with operand validation; audited in P5 |
| `server/rhs_jql_test.go` | **3, 5** | Create | JQL + membership tests; audited and possibly extended in P5 |
| `server/rhs_cache_test.go` | 3 | Create | TTL, invalidation, concurrent warm |
| `server/rhs.go` | 4 | Create | Client resolution for both auth modes, DTO normalization |
| `server/rhs_http.go` | 4 | Create | JSON error helper, admin gate, both handlers |
| `server/http.go` | 4 | Modify | Two route constants + two registrations |
| `server/rhs_http_test.go` | 4 | Create | Handler tests incl. JSON error bodies |
| `.planning/PHASE1_HANDOFF.md` | 5 | Create | Milestone 2 handoff, incl. the D2 constraint |

**Conflict risk — `server/plugin.go` is written by both Phase 1 and Phase 3.** Phase 1
adds `externalConfig` fields (around :52-103); Phase 3 adds `Plugin` struct cache fields
(around :126-169) and one call inside `OnConfigurationChange` (around :300). The regions
are distinct, but if these phases were ever run concurrently by two agents the file would
conflict. The commit checkpoint after Phase 1 exists specifically to prevent that.

**`server/rhs_jql.go` and its test are revisited in Phase 5**, by design — the audit may
add tests.

## Within-Phase Parallelization

Most of this plan is sequential because each phase consumes the previous one's types.
Where two agents can genuinely work at once:

- **Phase 1:** T1.3 (`server/user.go`) is independent of T1.1/T1.2 and can run in
  parallel. T1.4 must follow T1.2.
- **Phase 2:** T2.1 and T2.2 touch the same new file, so they are better done by one
  agent; T2.3 follows both.
- **Phase 3:** T3.1 (cache) and T3.2/T3.3 (JQL, same file) are genuinely independent and
  are the best parallelization opportunity in the plan — but T3.1 also edits
  `server/plugin.go`, so the agent taking it must not be the one holding uncommitted
  Phase 1 edits.
- **Phase 4:** T4.1 is a prerequisite for T4.4. T4.2 and T4.3 both live in `server/rhs.go`
  and should be one agent.

## Testing Strategy

Tests live in `package main` alongside the code, matching the repo. The stack is
`testify` plus handwritten fakes — no gomock, no mockery. `plugintest.API` mocks the
Mattermost API, `makeTestKVStore` (`mock_kv_store.go:13`) provides an in-memory KV, and
`testClient` (`server/issue_test.go:39-45`) is the model for a handwritten `Client` fake.

Per layer: client methods are exercised against `httptest.NewServer` so retry and header
handling are tested for real rather than mocked; the JQL builder is a pure function and
gets exact-string table tests; handlers go through `p.ServeHTTP` with a recorder so
routing, `checkAuth`, and the JSON error bodies are all covered end to end.

Make the backoff base delay injectable so retry tests do not sleep for real seconds.

**The one non-standard rule, from risk #36:** any test touching a negated
`statusCategory` clause asserts on **result membership**. A bad operand returns a
superset with a 200, so status-code and non-emptiness assertions both pass while the
behavior is wrong.

Commands:

```bash
make test          # gotestsum over the whole suite
make check-style   # go vet + golangci-lint run ./... + mattermost-govet
cd server && go test ./... -v
cd server && go test ./... -run TestRHS -v
```

## Definition of Done (Overall)

- [ ] `EnableJiraRHS` loads from plugin config with a manifest default of `false` and is
      exposed to the webapp as `rhs_enabled`.
- [ ] `RHSStatusTabs` deserializes from a `type: "custom"` setting into a typed
      `map[string][]RHSTabEntry`, with a test that would catch a non-lowercase json tag.
- [ ] An instance with no stored tab config resolves to Assigned + In Progress with no
      config write and no admin Jira connection.
- [ ] `GET /api/v2/rhs/statuses` returns the instance-wide status list, is gated on
      `PermissionManageSystem`, works via the bot client on Connect JWT with no personal
      connection, and returns `not_connected` on OAuth2 without one.
- [ ] `GET /api/v2/rhs/issues` returns normalized issues plus `nextPageToken` and
      `isLast`, honors both sorts, pages at 20, and hides vanished tabs.
- [ ] Cloud search goes through `/rest/api/3/search/jql`; no new code calls go-jira's
      `Issue.Search`.
- [ ] A 429 is retried with `Retry-After`-aware backoff and jitter, capped at 4 attempts,
      and is **not** retried when `RateLimit-Reason` is `jira-quota-global-based`.
- [ ] Every new-route failure path returns a JSON error code, never a plain-text 500.
- [ ] **Risk #36:** every `statusCategory` operand — including the Assigned tab's
      `done` — is validated against the canonical list at build time, and at least one
      negated-clause test asserts membership. Verified by the T5.2 audit, including the
      deliberate-removal check.
- [ ] `get-search-issues`, `SearchIssues`, and all 13 existing `getClient` call sites are
      unchanged.
- [ ] `make test` and `make check-style` both pass.
