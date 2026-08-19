# Phase 1 Plan: Config foundation and types

> Prescriptive implementation plan for **Phase 1 only** (tasks 1.1–1.4). An
> Implementation Engineer should be able to land this without making design
> decisions. Do not implement later phases from this file.
>
> **Do not write production/test Go until this plan is followed as written.**
> This document is the plan; it is not the code.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase 1 (source of truth for WHAT)
- **Spec:** `planner/projects/jira-plugin-rhs/ideas/001-show-tickets-rhs/spec.md`
- **Orchestration:** Gate 1 / Step 1 of `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Package:** `package main` under `server/` (module `github.com/mattermost/mattermost-plugin-jira`, `go.mod` at repo root)
- **Generated:** 2026-08-19
- **Status:** ready for implementation

## Scope

**In scope:** `server/rhs_types.go` (create), `server/rhs_types_test.go` (create),
`server/plugin.go` (`externalConfig` fields only), `plugin.json` (two settings),
`server/user.go` (`httpGetSettingsInfo` only).

**Out of scope (do not do these):**

- Any file under `webapp/`
- `OnConfigurationChange` parsing, validation, or cache invalidation
- `Plugin` struct cache fields (`rhsStatusCache` is Phase 3)
- Jira HTTP calls, Cloud client methods, `/search/jql`
- HTTP routes `/rhs/issues` or `/rhs/statuses`
- `JiraStatus`, `JiraStatusCategory`, `RHSIssue`, `ErrRateLimited` (later phases add these to `rhs_types.go`)
- Writing plugin config / `SavePluginConfig` / KV

---

## Research (read this before coding)

### 1. `externalConfig` and json tags — `server/plugin.go`

Struct is `server/plugin.go:52-103`. **Every existing `json:` tag is already
all-lowercase.** Untagged exported fields rely on `encoding/json` case-insensitive
matching and must not be used as a model for the new fields (this plugin's
existing style for new tagged fields is explicit lowercase, matching
`EnableJiraUI`).

| Field | Tag | Lines |
|-------|-----|-------|
| `EnableJiraUI` | `json:"enablejiraui"` | 54 |
| `Secret` | `json:"secret"` | 57 |
| `RolesAllowedToEditJiraSubscriptions` | *(none)* | 60 |
| `GroupsAllowedToEditJiraSubscriptions` | *(none)* | 63 |
| `MaxAttachmentSize` | *(none)* | 67 |
| `JiraAdminAdditionalHelpText` | *(none)* | 70 |
| `SecurityLevelEmptyForJiraSubscriptions` | *(none)* | 73 |
| `HideDecriptionComment` | *(none)* | 76 |
| `EnableAutocomplete` | *(none)* | 79 |
| `EnableWebhookEventLogging` | *(none)* | 82 |
| `DisplaySubscriptionNameInNotifications` | *(none)* | 85 |
| `EncryptionKey` | *(none)* | 88 |
| `AdminAPIToken` | *(none)* | 91 |
| `AdminEmail` | *(none)* | 94 |
| `ThreadedJiraCommentSubscriptionDuration` | `json:"threadedjiracommentsubscriptionduration"` | 97 |
| `TeamIDs` | `json:"teamids"` | 100 |
| `TeamIDList` | `json:"teamidlist"` | 102 |

`OnConfigurationChange` is `server/plugin.go:192-326`. It loads config with
`p.client.Configuration.LoadPluginConfiguration(&ec)` at `:198` — a typed
unmarshal, not a string parse. Then it does **string** post-processing for
`MaxAttachmentSize`, `AdminAPIToken` encryption, duration, and `TeamIDs`.

### 2. Why NOT to copy `TeamIDs` parsing (`server/plugin.go:247-298`)

`TeamIDs` is a `type: "text"` setting. The stored value is a **string** of
`[name](uuid)` pairs. `OnConfigurationChange` splits on `,`, regex-matches
`^\[(.*?)\]\((.*?)\)$`, validates UUIDv4, skips malformed entries with a log
warning, and writes a derived `TeamIDList`. That pattern exists because the
manifest type cannot hold structured data.

`RHSStatusTabs` is `type: "custom"`. The stored value is a **native JSON object**
(map of instance id → tab entries). `LoadPluginConfiguration` already
`json.Unmarshal`s it into a typed Go field. Copying the TeamIDs path would mean:

- treating the value as a string and parsing it by hand (wrong type),
- adding error-skipping / log-warning behavior the spec does not want,
- touching `OnConfigurationChange`, which this phase forbids.

Do **not** add any `OnConfigurationChange` code for either new field.

### 3. `plugin.json` insertion points (current lines; not drifted)

`EnableJiraUI` occupies **lines 26–33**. The next object is `"key": "secret"` at
line 35. Insert `EnableJiraRHS` **between** those two objects (after the
`EnableJiraUI` `},`).

The settings array closes at **line 160** (`        ]`). The last setting is
`TeamIDs` at **lines 152–159**. Insert `RHSStatusTabs` **after** the `TeamIDs`
object and **before** that `]`. The `TeamIDs` object currently has **no trailing
comma** (it is last). Adding a following object requires adding a comma after
the `TeamIDs` `}`.

### 4. `httpGetSettingsInfo` — `server/user.go:219-228`

```219:228:server/user.go
func (p *Plugin) httpGetSettingsInfo(w http.ResponseWriter, r *http.Request) (int, error) {
	conf := p.getConfig()
	return respondJSON(w, struct {
		UIEnabled                              bool `json:"ui_enabled"`
		SecurityLevelEmptyForJiraSubscriptions bool `json:"security_level_empty_for_jira_subscriptions"`
	}{
		UIEnabled:                              conf.EnableJiraUI,
		SecurityLevelEmptyForJiraSubscriptions: conf.SecurityLevelEmptyForJiraSubscriptions,
	})
}
```

`ui_enabled` is sourced from `conf.EnableJiraUI`. Add `rhs_enabled` from
`conf.EnableJiraRHS` the same way. Route: `routeAPISettingsInfo = "/settingsinfo"`
at `server/http.go:53`, registered at `:126` on `apiRouter` (prefix `/api/v2`).
Full path: **`GET /api/v2/settingsinfo`**. Gated by `checkAuth` (`:317-326`),
which requires a `Mattermost-User-Id` header (`HeaderMattermostUserID` in
`server/constants.go:7`).

`httpGetSettingsInfo` itself does not read the user id; auth is the wrapper.

### 5. Existing tests — there is NO settings-info test

Repo-wide search of `*_test.go` found **zero** references to `httpGetSettingsInfo`,
`routeAPISettingsInfo`, `settingsinfo`, or `ui_enabled`. T1.4 / DoD cannot "extend"
an existing settings-info test. Add a new handler test in `rhs_types_test.go`
named so `-run TestRHSConfig` picks it up.

**Helper / style to copy (do not invent a new one):**

- Package: `package main` (all `server/*_test.go` files).
- Assertions: `github.com/stretchr/testify/assert` + `require`.
- Plugin bootstrap: `setupTestPlugin(api)` at `server/issue_test.go:99-114`
  (same package, so callable). It `SetAPI`s, `initializeRouter`s, installs mock
  stores, builds `pluginapi.NewClient`, and sets `conf.Secret`.
- HTTP: `httptest.NewRequest` + `httptest.NewRecorder` +
  `p.ServeHTTP(&plugin.Context{}, w, request)` with
  `request.Header.Set(HeaderMattermostUserID, ...)`.
  Closest GET analogue: `TestRouteUserStart` in `server/user_test.go:56-86`.
  Closest `setupTestPlugin` analogue: `server/issue_test.go:99-114`.
  Closest ServeHTTP + user-id header analogue: `server/http_test.go:481-490`.
- Table tests: `for name, tc := range map[string]struct{ ... }{ ... }` with
  `t.Run(name, ...)` — used throughout `http_test.go` / `user_test.go`.

`setupTestPlugin` already mocks `LogError` / `LogDebug`. A 200 from
settings-info does not log (`logResponse` returns early on OK,
`server/http.go:352-355`), so no extra `LogWarn` mock is required for the
happy path.

### 6. File header (required)

golangci does **not** enforce a copyright header. `make check-style` does, via
mattermost-govet:

```204:204:Makefile
	$(GO) vet -vettool=$(GOBIN)/mattermost-govet -license -license.year=2017 $$($(GO) list ./... | grep -v /webapp/)
```

Every existing `server/*.go` file starts with exactly this (blank line, then
`package main`):

```go
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
```

Put that on both new files. Year is **2017**, not 2026.

### 7. `.golangci.yml` linters that apply to this phase

Enabled (`server/.golangci.yml` is repo-root `.golangci.yml:25-38`):
`bodyclose`, `errcheck`, `gocritic`, `gosec`, `govet` (all except
`fieldalignment`), `ineffassign`, `misspell` (US), `nakedret`, `revive`,
`staticcheck`, `unconvert`, `unused`, `whitespace`.

Formatters: `gofmt` (simplify) + `goimports` with
`local-prefixes: github.com/mattermost/mattermost-plugin-jira`.

Phase 1 adds no HTTP bodies and no new errors to check. Still: do not discard
`json.Marshal`/`Unmarshal` errors in tests (`require.NoError`). `revive`
`package-comments` / `exported` / `var-naming` are excluded, so unexported
`rhsDefaultTabs` and `statusCategoryKey*` are fine. `goconst` is configured but
**not enabled**.

### 8. `LoadPluginConfiguration` — local Mattermost checkout

File: `~/git/mattermost/server/channels/app/plugin_api.go:50-80`
(parent plan cited `:50-82`; the function body is 50–80).

```50:80:~/git/mattermost/server/channels/app/plugin_api.go
func (api *PluginAPI) LoadPluginConfiguration(dest any) error {
	finalConfig := make(map[string]any)

	// First set final config to defaults
	if api.manifest.SettingsSchema != nil {
		for _, setting := range api.manifest.SettingsSchema.Settings {
			finalConfig[strings.ToLower(setting.Key)] = setting.Default
		}
		// ... sections likewise ToLower ...
	}

	// If we have settings given we override the defaults with them
	for setting, value := range api.app.Config().PluginSettings.Plugins[api.id] {
		finalConfig[strings.ToLower(setting)] = value
	}

	pluginSettingsJsonBytes, err := json.Marshal(finalConfig)
	// ...
	err = json.Unmarshal(pluginSettingsJsonBytes, dest)
	// ...
	return nil
}
```

This plugin calls it through pluginapi (`~/git/mattermost/server/public/pluginapi/configuration.go:17-20`),
which is a one-line wrap of `api.LoadPluginConfiguration(dest)`. `dest` is "a
pointer to a struct to which the configuration JSON can be unmarshalled" — a
**typed value**, not a JSON string.

**Consequence (D1):** after ToLower, the JSON object keys are `enablejirarhs` and
`rhsstatustabs`. A tag `json:"rhsStatusTabs"` or `json:"EnableJiraRHS"` never
matches. The field stays at its zero value (`false` / `nil`) with **no error**.
Nested fields inside the stored object (`kind`, `key`, `id`, `name`) are **not**
lowercased; camelCase-style tags on `RHSTabEntry` are correct.

Server test that demonstrates mixed incoming casing being lowercased:
`~/git/mattermost/server/channels/app/plugin_api_test.go` `TestPluginAPILoadPluginConfiguration`
(`:935-966`) feeds `{"mystringsetting": "str", "MyIntSetting": 32, "myBoolsetting": true}`.

### 9. Precedent — `type: "custom"` in `plugin.json`

**custom-attributes** (`~/git/mattermost-plugin-custom-attributes/plugin.json:20-24`):

```json
"settings": [{
    "key": "CustomAttributes",
    "type": "custom",
    "display_name": "Custom Attributes"
}]
```

No `default`. Go field is a typed slice `[]CustomAttribute` with **no** json tag
(`server/configuration.go:21`) — it works only because untagged unmarshal is
case-insensitive. Do not copy the missing tag; this plugin tags `EnableJiraUI`.

**plugin-ai** (`~/git/mattermost-plugin-ai/plugin.json:26-29`):

```json
{
  "key": "Config",
  "type": "custom"
}
```

No `default`, no `display_name`. Go: `Config \`json:"config"\`` in
`server/configuration.go:38` — **all-lowercase tag** matching `ToLower("Config")`.
Nested `Config` fields use camelCase tags (`json:"embeddingSearchConfig"` etc.).
That is the D1 pattern to copy: lowercase **top-level** tag, camelCase nested.

### 10. Official docs

- Custom settings: https://developers.mattermost.com/integrate/plugins/best-practices/
  — declare `"type": "custom"` in `settings_schema.settings`, register a component
  with `registerAdminConsoleCustomSetting`. `props.value` is "the setting's
  current **json value** in the config" (a JSON value, not a stringified blob).
  `onChange(id, <json value>)` writes that value back. Example cited: Custom
  Attributes plugin.
- Manifest: https://developers.mattermost.com/integrate/plugins/manifest-reference/

Phase 2 registers the component. Phase 1 only declares the key so the console
will have somewhere to mount it.

### 11. Pitfall confirmation vs parent `.planning/PLAN.md`

| Claim in parent PLAN.md | Current worktree | Drift? |
|-------------------------|------------------|--------|
| `externalConfig` at `plugin.go:52-103` | 52–103 | **No** |
| `EnableJiraUI` ends `plugin.json` line 33 | 26–33 | **No** |
| settings `]` at line 160 | 160 | **No** |
| `httpGetSettingsInfo` at `user.go:219-228` | 219–228 | **No** |
| `OnConfigurationChange` `:192-326` | 192–326 | **No** |
| `TeamIDs` parse `:247-298` | 247–298 | **No** |
| `Plugin` struct `:126-169` | 126–169 | **No** (do not edit in P1) |
| RHS types already exist | **zero** `*rhs*` files | **No** (create as planned) |
| settings-info handler test exists | **none** | **Yes — surprise.** Add `TestRHSConfigSettingsInfoExposesFlag` |
| `LoadPluginConfiguration` `:50-82` | `:50-80` | Cosmetic (function ends 80) |

---

## Decisions this phase locks (do not reopen)

**P1-D1 — json tags on `externalConfig` are all-lowercase.**
`enablejirarhs` and `rhsstatustabs`. See research §8. Nested `RHSTabEntry` tags
stay `kind` / `key` / `id` / `name`.

**P1-D2 — Virtual seed, not a config write.** `rhsDefaultTabs()` returns the
read-time default (Assigned + In Progress). Nothing in this phase writes config.
Phase 3 `resolveTabs` will call this when an instance has no stored entries.
Phase 2's picker must render that seed when `value` is empty (not this phase).

**P1-D3 — Third tab kind `assigned`.** Parent T1.1 listed only `category` and
`status`, but also required `rhsDefaultTabs()` to return Assigned. Assigned is
neither a category (`statusCategory = key`) nor a status (`status = id`); its
JQL is `statusCategory != done`. Represent it as:

```go
RHSTabKindAssigned RHSTabKind = "assigned"
```

Do **not** add a `Locked` JSON field. "Locked" is a Phase 2 picker rule:
`Kind == assigned` cannot be deselected. Assigned is **not** stored in the
admin-saved map (spec: "always-on, not stored as a removable entry").
`rhsDefaultTabs` is the runtime seed, not the persisted shape.

**P1-D4 — Do not parse in `OnConfigurationChange`.** Typed unmarshal is enough.

**P1-D5 — Do not add Phase 2/3/4 types yet.** `rhs_types.go` in this phase
contains only tab kinds, `RHSTabEntry`, category-key constants, and
`rhsDefaultTabs()`. Later phases append to the same file.

---

## Exact types (`server/rhs_types.go`) — Task 1.1

Create the file. Full contents:

```go
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

type RHSTabKind string

const (
	RHSTabKindAssigned RHSTabKind = "assigned"
	RHSTabKindCategory RHSTabKind = "category"
	RHSTabKindStatus   RHSTabKind = "status"
)

// Canonical Jira Cloud status-category keys. Use these as JQL operands.
// Do not hardcode the numeric ids (2/4/3/1) — they are counterintuitive.
const (
	statusCategoryKeyNew           = "new"
	statusCategoryKeyIndeterminate = "indeterminate"
	statusCategoryKeyDone          = "done"
	statusCategoryKeyUndefined     = "undefined"
)

// RHSTabEntry is one admin-configured (or virtual-seed) tab.
// Nested fields keep camelCase-style json names; LoadPluginConfiguration only
// lowercases the top-level setting key.
type RHSTabEntry struct {
	Kind RHSTabKind `json:"kind"`
	Key  string     `json:"key,omitempty"` // status-category key, e.g. "indeterminate"
	ID   string     `json:"id,omitempty"`  // numeric status id, e.g. "10001"
	Name string     `json:"name"`
}

// rhsDefaultTabs is the virtual seed (D2): Assigned (always-on) plus In Progress.
// Called when an instance has no stored tab config. Does not write plugin config.
func rhsDefaultTabs() []RHSTabEntry {
	return []RHSTabEntry{
		{Kind: RHSTabKindAssigned, Name: "Assigned"},
		{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
	}
}
```

`rhsDefaultTabs` return value is **exactly** those two entries, in that order.
Assigned has empty `Key` and `ID`. In Progress uses key `indeterminate` (not
numeric id `4`).

---

## Exact `externalConfig` fields — Task 1.2a (`server/plugin.go`)

Insert immediately after `EnableJiraUI` (after line 54, before the `Secret`
field at 57). Do not reorder existing fields.

```go
type externalConfig struct {
	// Setting to turn on/off the webapp components of this plugin
	EnableJiraUI bool `json:"enablejiraui"`

	// Setting to turn on/off the Cloud-only personal tickets RHS
	EnableJiraRHS bool `json:"enablejirarhs"`

	// Per Cloud instance id → status/category tabs. Native JSON object, not a string.
	RHSStatusTabs map[string][]RHSTabEntry `json:"rhsstatustabs"`

	// Webhook secret
	Secret string `json:"secret"`
	// ... unchanged remainder ...
}
```

**Tags must be literally `enablejirarhs` and `rhsstatustabs`.** Not
`EnableJiraRHS`, `rhsStatusTabs`, `enableJiraRHS`, or `RHSStatusTabs`.

Leave `OnConfigurationChange` untouched. Leave the `Plugin` struct untouched.

---

## Exact `plugin.json` snippets — Task 1.2b

Indentation is 4 spaces, matching the file. Keys in the manifest stay
PascalCase (`EnableJiraRHS`, `RHSStatusTabs`); ToLower happens in
`LoadPluginConfiguration`, not here.

### Insert A — immediately after `EnableJiraUI`

**Current surrounding JSON (lines 26–36):**

```json
            {
                "key": "EnableJiraUI",
                "display_name": "Allow users to attach and create Jira issues in Mattermost:",
                "type": "bool",
                "help_text": "When **false**, users cannot attach and create Jira issues in Mattermost. Does not affect Jira webhook notifications. Select **false** then disable and re-enable this plugin in **System Console \u003e Plugins \u003e Plugin Management** to reset the plugin state for all users. \n \n When **true**, install this plugin to your Jira instance with '/jira install' to allow users to create and manage issues across Mattermost channels. See [documentation](https://about.mattermost.com/default-jira-plugin-link-application) to learn more.",
                "placeholder": "",
                "default": true
            },
            {
                "key": "secret",
```

**Replace the `},` + `"key": "secret"` boundary with:**

```json
            {
                "key": "EnableJiraUI",
                "display_name": "Allow users to attach and create Jira issues in Mattermost:",
                "type": "bool",
                "help_text": "When **false**, users cannot attach and create Jira issues in Mattermost. Does not affect Jira webhook notifications. Select **false** then disable and re-enable this plugin in **System Console \u003e Plugins \u003e Plugin Management** to reset the plugin state for all users. \n \n When **true**, install this plugin to your Jira instance with '/jira install' to allow users to create and manage issues across Mattermost channels. See [documentation](https://about.mattermost.com/default-jira-plugin-link-application) to learn more.",
                "placeholder": "",
                "default": true
            },
            {
                "key": "EnableJiraRHS",
                "display_name": "Show Jira tickets in the right-hand sidebar:",
                "type": "bool",
                "help_text": "When **false**, the Jira tickets right-hand sidebar is not registered. When **true**, users with a connected Jira Cloud account can open their assigned tickets from the App Bar. Jira Cloud only; Server/Data Center instances are never shown.",
                "placeholder": "",
                "default": false
            },
            {
                "key": "secret",
```

`default` is JSON boolean `false` (not `"false"`). `EnableJiraUI` uses `true`
unquoted; match that.

### Insert B — before the settings array close

**Current surrounding JSON (lines 152–161):**

```json
            {
                "key": "TeamIDs",
                "display_name": "Team List",
                "type": "text",
                "help_text": "Comma separated list of team name and IDs to be used for filtering subscriptions\n**Note:** Teams provided here will be used across all configured Jira instances",
                "placeholder": "[team-1-name](team-1-id),[team-2-name](team-2-id)",
                "default": ""
            }
        ]
    }
}
```

**Become:**

```json
            {
                "key": "TeamIDs",
                "display_name": "Team List",
                "type": "text",
                "help_text": "Comma separated list of team name and IDs to be used for filtering subscriptions\n**Note:** Teams provided here will be used across all configured Jira instances",
                "placeholder": "[team-1-name](team-1-id),[team-2-name](team-2-id)",
                "default": ""
            },
            {
                "key": "RHSStatusTabs",
                "type": "custom",
                "display_name": "Jira RHS status tabs"
            }
        ]
    }
}
```

No `default`, no `help_text`, no `placeholder`, no `secret`. Shape matches
custom-attributes (key + type + display_name). The key **must** be
`RHSStatusTabs` or the Phase 2 `registerAdminConsoleCustomSetting` call will
have nothing to bind to.

After the edit, `python3 -m json.tool plugin.json > /dev/null` must succeed.

---

## Exact `httpGetSettingsInfo` change — Task 1.3 (`server/user.go:219-228`)

Replace the function with:

```go
func (p *Plugin) httpGetSettingsInfo(w http.ResponseWriter, r *http.Request) (int, error) {
	conf := p.getConfig()
	return respondJSON(w, struct {
		UIEnabled                              bool `json:"ui_enabled"`
		RHSEnabled                             bool `json:"rhs_enabled"`
		SecurityLevelEmptyForJiraSubscriptions bool `json:"security_level_empty_for_jira_subscriptions"`
	}{
		UIEnabled:                              conf.EnableJiraUI,
		RHSEnabled:                             conf.EnableJiraRHS,
		SecurityLevelEmptyForJiraSubscriptions: conf.SecurityLevelEmptyForJiraSubscriptions,
	})
}
```

JSON key is `rhs_enabled` (snake_case, matching `ui_enabled`). Source field is
`conf.EnableJiraRHS`. Do not expose `RHSStatusTabs` on this route (picker reads
plugin config via the System Console `value` prop in Phase 2; the RHS reads
resolved tabs from `/rhs/issues` in Phase 4).

Do not change the route or `checkAuth` wrapping.

---

## Exact tests — Task 1.4 (`server/rhs_types_test.go`)

Create the file. `package main`. Header as in §6. All test names start with
`TestRHSConfig` so `go test ./... -run TestRHSConfig` runs them.

### Helper (same file, unexported)

```go
// loadConfigLikeMattermost reproduces LoadPluginConfiguration's round-trip:
// lowercase every top-level key, json.Marshal, json.Unmarshal into externalConfig.
func loadConfigLikeMattermost(t *testing.T, stored map[string]any) externalConfig {
	t.Helper()
	finalConfig := make(map[string]any, len(stored))
	for key, value := range stored {
		finalConfig[strings.ToLower(key)] = value
	}
	b, err := json.Marshal(finalConfig)
	require.NoError(t, err)
	var dest externalConfig
	require.NoError(t, json.Unmarshal(b, &dest))
	return dest
}
```

This is the test stand-in for `plugin_api.go:50-80`. Do **not** call
`LoadPluginConfiguration` (needs a live App). Do **not** go through
`OnConfigurationChange`.

### `TestRHSConfigLoadFromLowercaseKeys`

Table test. **This is the Gate 1 regression test.** It must fail if either
top-level tag is camelCase, because the keys after `ToLower` are `enablejirarhs`
and `rhsstatustabs`.

```go
func TestRHSConfigLoadFromLowercaseKeys(t *testing.T) {
	const instanceA = "https://a.example.atlassian.net"
	const instanceB = "https://b.example.atlassian.net"

	categoryEntry := map[string]any{
		"kind": "category",
		"key":  "indeterminate",
		"name": "In Progress",
	}
	statusEntry := map[string]any{
		"kind": "status",
		"id":   "10001",
		"name": "Submitted",
	}

	tests := map[string]struct {
		stored          map[string]any
		wantRHSEnabled  bool
		wantTabs        map[string][]RHSTabEntry
		wantTabsNil     bool
	}{
		"two instances, keys already lowercased": {
			stored: map[string]any{
				"enablejirarhs": true,
				"rhsstatustabs": map[string]any{
					instanceA: []any{categoryEntry},
					instanceB: []any{statusEntry},
				},
			},
			wantRHSEnabled: true,
			wantTabs: map[string][]RHSTabEntry{
				instanceA: {{Kind: RHSTabKindCategory, Key: "indeterminate", Name: "In Progress"}},
				instanceB: {{Kind: RHSTabKindStatus, ID: "10001", Name: "Submitted"}},
			},
		},
		"schema-cased keys are lowercased before unmarshal": {
			stored: map[string]any{
				"EnableJiraRHS": true,
				"RHSStatusTabs": map[string]any{
					instanceA: []any{categoryEntry},
				},
			},
			wantRHSEnabled: true,
			wantTabs: map[string][]RHSTabEntry{
				instanceA: {{Kind: RHSTabKindCategory, Key: "indeterminate", Name: "In Progress"}},
			},
		},
		"absent key yields nil map and false flag": {
			stored:         map[string]any{},
			wantRHSEnabled: false,
			wantTabsNil:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := loadConfigLikeMattermost(t, tc.stored)
			assert.Equal(t, tc.wantRHSEnabled, got.EnableJiraRHS)
			if tc.wantTabsNil {
				assert.Nil(t, got.RHSStatusTabs)
				return
			}
			require.NotNil(t, got.RHSStatusTabs)
			assert.Equal(t, tc.wantTabs, got.RHSStatusTabs)
		})
	}
}
```

If someone changes the struct to `json:"rhsStatusTabs"`, the `"two instances,
keys already lowercased"` case fails: `got.RHSStatusTabs` is `nil`. That is the
proof Gate 1 asks for.

### `TestRHSConfigCamelCaseTagWouldFail`

Independent demonstration so a reviewer can see the failure mode without
editing production tags. Uses a **local** type with the wrong tag. This test
**passes** as written (it asserts the wrong tag does not populate). Together
with `TestRHSConfigLoadFromLowercaseKeys`, it proves the production tag is the
lowercase one.

```go
func TestRHSConfigCamelCaseTagWouldFail(t *testing.T) {
	payload := []byte(`{"rhsstatustabs":{"https://a.example.atlassian.net":[{"kind":"category","key":"indeterminate","name":"In Progress"}]}}`)

	var camel struct {
		RHSStatusTabs map[string][]RHSTabEntry `json:"rhsStatusTabs"`
	}
	require.NoError(t, json.Unmarshal(payload, &camel))
	assert.Nil(t, camel.RHSStatusTabs, "camelCase json tag must not match a lowercased LoadPluginConfiguration key")

	var lower struct {
		RHSStatusTabs map[string][]RHSTabEntry `json:"rhsstatustabs"`
	}
	require.NoError(t, json.Unmarshal(payload, &lower))
	require.NotNil(t, lower.RHSStatusTabs)
	assert.Equal(t, "indeterminate", lower.RHSStatusTabs["https://a.example.atlassian.net"][0].Key)
}
```

### `TestRHSConfigDefaultTabs`

```go
func TestRHSConfigDefaultTabs(t *testing.T) {
	got := rhsDefaultTabs()
	require.Len(t, got, 2)

	assert.Equal(t, RHSTabKindAssigned, got[0].Kind)
	assert.Equal(t, "Assigned", got[0].Name)
	assert.Empty(t, got[0].Key)
	assert.Empty(t, got[0].ID)

	assert.Equal(t, RHSTabKindCategory, got[1].Kind)
	assert.Equal(t, statusCategoryKeyIndeterminate, got[1].Key)
	assert.Equal(t, "indeterminate", got[1].Key)
	assert.Equal(t, "In Progress", got[1].Name)
	assert.Empty(t, got[1].ID)
}
```

### `TestRHSConfigSettingsInfoExposesFlag`

Handler test hitting the **existing** route. There is no prior settings-info
test to extend; this is the assertion the parent DoD asked for.

```go
func TestRHSConfigSettingsInfoExposesFlag(t *testing.T) {
	tests := map[string]struct {
		rhsEnabled bool
	}{
		"enabled":  {rhsEnabled: true},
		"disabled": {rhsEnabled: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := setupTestPlugin(api)
			p.updateConfig(func(conf *config) {
				conf.EnableJiraRHS = tc.rhsEnabled
			})

			request := httptest.NewRequest(http.MethodGet, makeAPIRoute(routeAPISettingsInfo), nil)
			request.Header.Set(HeaderMattermostUserID, "test-user-id")
			w := httptest.NewRecorder()
			p.ServeHTTP(&plugin.Context{}, w, request)

			require.Equal(t, http.StatusOK, w.Result().StatusCode)
			require.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))

			var body struct {
				UIEnabled  bool `json:"ui_enabled"`
				RHSEnabled bool `json:"rhs_enabled"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tc.rhsEnabled, body.RHSEnabled)
		})
	}
}
```

`makeAPIRoute(routeAPISettingsInfo)` is `/api/v2/settingsinfo`
(`server/http.go:88-90` + `:53`). Do not call `httpGetSettingsInfo` directly;
going through `ServeHTTP` covers `checkAuth` + routing.

Imports for the test file:

```go
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

`setupTestPlugin`, `makeAPIRoute`, `routeAPISettingsInfo`, `HeaderMattermostUserID`
are in the same `package main`. `plugin.Context` is required by `ServeHTTP`.
`plugintest` is required by `setupTestPlugin`. `strings` is used by
`loadConfigLikeMattermost`. `goimports` will group stdlib / mattermost /
testify with blank lines (local-prefix is this module; these imports are all
external).

---

## File-by-file change list (current line numbers)

| File | Action | Lines to touch | What |
|------|--------|----------------|------|
| `server/rhs_types.go` | **Create** | n/a | Types + `rhsDefaultTabs` as specified above |
| `server/rhs_types_test.go` | **Create** | n/a | Four `TestRHSConfig*` functions |
| `server/plugin.go` | Modify | Insert after **54** (`EnableJiraUI` field). Do not edit **192–326** (`OnConfigurationChange`) or **126–169** (`Plugin` struct) | Two fields on `externalConfig` |
| `plugin.json` | Modify | Insert after **33** (end of `EnableJiraUI` object). Insert before **160** (settings `]`). Add comma after `TeamIDs` `}` at **159** | `EnableJiraRHS` bool + `RHSStatusTabs` custom |
| `server/user.go` | Modify | **219–228** only | Add `rhs_enabled` to settings-info response |

Do not modify: `server/http.go`, `server/constants.go`, `.golangci.yml`,
`webapp/**`, `OnConfigurationChange`.

---

## Commands

From the worktree root
`~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`:

```bash
# Phase 1 unit tests (all TestRHSConfig* names)
cd server && go test ./... -run TestRHSConfig -v

# JSON validity
python3 -m json.tool plugin.json > /dev/null

# Style + license header (new files must pass mattermost-govet -license.year=2017)
make check-style
```

There is no pre-existing settings-info test name. The handler assertion is
`TestRHSConfigSettingsInfoExposesFlag`. Expected `-run TestRHSConfig` output:
four passing tests (`LoadFromLowercaseKeys` with 3 subtests,
`CamelCaseTagWouldFail`, `DefaultTabs`, `SettingsInfoExposesFlag` with 2
subtests).

---

## Gate 1 checklist (reviewer)

From `IMPL_ORCHESTRATION_PLAN.md` Step 1. A reviewer executes these; they are
not optional.

- [ ] `server/plugin.go`: `EnableJiraRHS` tag is exactly `` json:"enablejirarhs" ``
      (all lowercase, no camelCase).
- [ ] `server/plugin.go`: `RHSStatusTabs` tag is exactly `` json:"rhsstatustabs" ``
      (all lowercase, no camelCase).
- [ ] `plugin.json` declares `"key": "RHSStatusTabs"` with `"type": "custom"` and
      **no** `default`.
- [ ] `plugin.json` declares `"key": "EnableJiraRHS"` with `"default": false`.
- [ ] `TestRHSConfigLoadFromLowercaseKeys` exists and unmarshals a payload whose
      top-level keys are **lowercased** (`rhsstatustabs` / `enablejirarhs`).
- [ ] That test **would actually fail** if the production tag were camelCase:
      confirm by temporarily changing `` json:"rhsstatustabs" `` to
      `` json:"rhsStatusTabs" ``, running
      `cd server && go test ./... -run TestRHSConfigLoadFromLowercaseKeys -v`,
      observing FAIL (`RHSStatusTabs` nil), and restoring the lowercase tag.
- [ ] `TestRHSConfigCamelCaseTagWouldFail` exists and documents the same trap
      on a local type.
- [ ] `OnConfigurationChange` diff is empty (no TeamIDs-style parsing).
- [ ] `webapp/` diff is empty.
- [ ] `Plugin` struct has no new cache fields.

**Commit checkpoint (orchestration):** local commit of this phase only. Do not
push. Phase 3 will edit `plugin.go` again in a different region.

---

## What this phase hands to later phases

- Types: `RHSTabKind`, `RHSTabEntry`, category-key constants, `rhsDefaultTabs()`.
- Config: `conf.EnableJiraRHS`, `conf.RHSStatusTabs` populated by
  `LoadPluginConfiguration` with no extra parsing.
- Webapp gate (milestone 2): `GET /api/v2/settingsinfo` → `{ ..., "rhs_enabled": <bool> }`
  read like `ui_enabled` in `webapp/src/plugin.tsx`.
- D2 constraint for the System Console picker: a never-configured instance
  stores **nothing**; `value` will be empty. The picker must render Assigned
  (locked) + In Progress itself and must not call `onChange` to materialize
  them. Recorded here because Phase 1 is what makes it necessary; implemented
  in milestone 2.

Phase 3 will call `rhsDefaultTabs()` from `resolveTabs` when stored entries are
empty, and will switch on `RHSTabKindAssigned` when building Assigned JQL.

---

## Implementation Summary

Implemented by IE1 on 2026-08-19. No commit. No push. Phase 2 not started.

### Files created
- `server/rhs_types.go` — `RHSTabKind` (`assigned` / `category` / `status`), category-key constants, `RHSTabEntry`, `rhsDefaultTabs()` (Assigned then In Progress / `indeterminate`)
- `server/rhs_types_test.go` — `TestRHSConfig*` tests

### Files modified
- `server/plugin.go` — `externalConfig` only: `EnableJiraRHS` `json:"enablejirarhs"`, `RHSStatusTabs` `json:"rhsstatustabs"`. `OnConfigurationChange` and `Plugin` struct untouched
- `plugin.json` — `EnableJiraRHS` bool default `false`; `RHSStatusTabs` `type: "custom"` with no `default`
- `server/user.go` — `httpGetSettingsInfo` adds `rhs_enabled` from `conf.EnableJiraRHS`

### Tests added
- `TestRHSConfigLoadFromLowercaseKeys` (3 subtests)
- `TestRHSConfigCamelCaseTagWouldFail` (adjusted; see deviation)
- `TestRHSConfigExternalConfigTagsAreLowercase` (**additional** — locks exact production tags via `reflect`)
- `TestRHSConfigDefaultTabs`
- `TestRHSConfigSettingsInfoExposesFlag` (2 subtests: enabled / disabled)

`setupTestPlugin` needed no extra mocks.

### Command results
- `python3 -m json.tool plugin.json > /dev/null` — **pass**
- `cd server && go test ./... -run TestRHSConfig -v` — **pass** (5 tests, 5 subtests). First compile failed until `make apply` generated gitignored `server/manifest.go`; rerun after apply passed
- Full `make check-style` — **not run** (depends on `webapp/node_modules` + `npm run lint`; `webapp/` was not touched)
- Go style equivalents — **pass** on changed files: `gofmt -l` clean; `go vet` on non-webapp packages; `./bin/golangci-lint run ./server/ --new` 0 issues; `go vet -vettool=mattermost-govet -license -license.year=2017` on `server` package

### Deviations
1. **`TestRHSConfigCamelCaseTagWouldFail` as specified cannot pass.** `encoding/json` matches struct tags **case-insensitively**, so `json:"rhsStatusTabs"` **does** populate from key `rhsstatustabs`. The planned `assert.Nil` fails against real Go behavior. The test still exists under the same name: it now asserts camelCase *does* populate, and adds a `json:"rhs_status_tabs"` local type that stays `nil` (true non-match).
2. **Added `TestRHSConfigExternalConfigTagsAreLowercase`.** Gate 1's "temporarily change production tag to camelCase → `LoadFromLowercaseKeys` FAILs" procedure will **not** fail, for the same case-insensitive reason. This extra test is what actually FAILs if `enablejirarhs` / `rhsstatustabs` are changed.

No other deviations. Production tags are still exactly `enablejirarhs` and `rhsstatustabs`. Nested `RHSTabEntry` tags remain `kind` / `key` / `id` / `name`.

### Gate 1 reviewer notes
- Tags: `json:"enablejirarhs"` and `json:"rhsstatustabs"` (all lowercase) on `externalConfig`
- `plugin.json`: `EnableJiraRHS` default JSON boolean `false`; `RHSStatusTabs` custom, no default
- `OnConfigurationChange` diff is empty; `Plugin` struct has no cache fields; `webapp/` diff is empty
- Do **not** expect `TestRHSConfigLoadFromLowercaseKeys` to fail if the production tag is temporarily changed to camelCase — use `TestRHSConfigExternalConfigTagsAreLowercase` instead
- Tests require generated `server/manifest.go` (`make apply`); that file is gitignored
- Do not commit/push as part of this implementation step (orchestration owns the checkpoint)
