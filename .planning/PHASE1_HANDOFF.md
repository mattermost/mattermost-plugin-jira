# Milestone 1 Handoff: Show your tickets in the RHS

> Server foundation (Phases 1–5) for the Cloud-only personal tickets RHS.
> Nothing user-visible has shipped. Milestone 2 (`webapp/`,
> `.planning/PLAN-MILESTONE2.md` W1–W6) is built entirely against this
> contract.
>
> **Do not start Milestone 2 from a memory of this file. Read D2 twice.**

## Verification

- **Branch:** `IDEA-001-show-tickets-rhs`
- **Phase 4 commit:** `df32f47`
- **Phase 5 run date:** 2026-08-19
- **`make test`:** pass. `make test` exit 0. gotestsum packages `ok` (`server` 8.313s). Jest 13 suites / 83 tests passed. gotestsum printed 5 pre-existing `(unknown)` FAIL lines for `TestEditSubscriptionTemplate` / `TestGetSubscriptionTemplate` whose bodies actually `--- PASS` (stdout noise `/n httpGetSubscriptionTemplates`); not new RHS code; Makefile continued to Jest.
- **`make check-style`:** pass. eslint `--quiet` clean; `go vet` silent; `golangci-lint run ./...` **0 issues**; mattermost-govet `-license -license.year=2017` silent.
- **Go-only fallback used?** no
- **Lint fixes applied:** G115 signed shift; G404 `crypto/rand` jitter; revive `maxFileSize` in cache test. No `//nolint`.
- **#36 audit:** stub `validateStatusCategoryOperand` → `TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys` **FAIL**; body restored; `TestRHSJQL|TestRHSCache|TestRHSHTTP` **PASS**.
- **Membership:** `TestRHSJQLAssignedExcludesDoneMembership` asserts `TES-DONE` absent.

## Routes

Both are GET-only, registered on `apiRouter` (`PathPrefix("/api/v2")`),
wrapped in `checkAuth` + `handleResponse`.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v2/rhs/issues` | logged-in Mattermost user (`Mattermost-User-Id`); user must have a Jira connection for the instance | Normalized “my tickets” page + resolved tabs |
| GET | `/api/v2/rhs/statuses` | logged-in **system admin** (`PermissionManageSystem`) | Instance-wide statuses + categories for the System Console picker |
| GET | `/api/v2/settingsinfo` | logged-in Mattermost user | Includes `rhs_enabled` (Phase 1; unchanged in later phases) |

Plugin HTTP prefix is `/api/v2` (`routeAPI` in `server/http.go`). Do not
add `/api/v2` again in client helpers.

**These routes do not read `EnableJiraRHS`.** Registration gating is a
webapp concern (`rhs_enabled`). The picker (`/rhs/statuses`) must work
while the flag is still off so an admin can configure tabs first
(W5.2).

Missing `Mattermost-User-Id` is **not** a JSON error. `checkAuth` still
returns **plain text** `401 Not authorized`. Do not “fix” that in
Milestone 2 and do not parse it as JSON.

## Query parameters (snake_case)

### `GET /api/v2/rhs/issues`

| Param | Required | Default | Notes |
|-------|----------|---------|-------|
| `instance_id` | yes | — | Cloud instance id (typically the site URL). Empty/unknown → JSON `invalid_request` (not `not_connected`) |
| `tab_kind` | no | `assigned` | `assigned` \| `category` \| `status`. Must match a **server-resolved** tab |
| `tab_key` | for `category` | — | Status-category key, e.g. `indeterminate` |
| `tab_id` | for `status` | — | Numeric status id as a string, e.g. `"10001"` |
| `sort` | no | `updated` | Allowlist **`updated`** \| **`created`** only. Anything else → `invalid_request`. Server default happens **before** `buildTabJQL` (which rejects empty) |
| `next_page_token` | no | empty | Opaque Cloud cursor. Pass through unchanged from the previous page’s `nextPageToken` |

Page size is **fixed at 20**. There is **no** `page_size` / `limit` /
`startAt`. Ignore them if sent.

Cloud search returns **no `total`**. UI is “Load more”, not “N of M”.

### `GET /api/v2/rhs/statuses`

| Param | Required | Notes |
|-------|----------|-------|
| `instance_id` | yes | Empty/unknown → `invalid_request` |

## Response bodies

### Issues — `200 application/json`

```json
{
  "issues": [ { "…RHSIssue…" } ],
  "tabs": [ { "kind": "assigned", "name": "Assigned" } ],
  "nextPageToken": "opaque-or-empty",
  "isLast": true
}
```

`tabs` is the **server-resolved** list (W-D4): Assigned always first;
vanished status ids / category keys already hidden; empty stored config
already replaced with the virtual seed. The webapp **must not**
re-derive tabs or build JQL.

### RHSIssue DTO (issues route only)

`get-search-issues` is unchanged and still returns raw `[]jira.Issue`.
Do not reuse this DTO there.

| JSON field | Go | Notes |
|------------|-----|-------|
| `key` | string | Issue key |
| `summary` | string | |
| `browseUrl` | string | `{GetJiraBaseURL()}/browse/{key}`. **Never** `GetURL()` — on `cloud-oauth` that is `api.atlassian.com` and 404s |
| `status` | object | `{ "name": string, "categoryKey": string }` — `categoryKey` is Cloud `statusCategory.key` (`new` / `indeterminate` / `done` / `undefined`) for pill color |
| `priority` | string | Name; empty if missing |
| `issueType` | string | |
| `project` | string | Project **key** |
| `assignee` | string | Display name |
| `reporter` | string | Display name |
| `created` | string | RFC3339 UTC, or `""` if zero |
| `updated` | string | RFC3339 UTC, or `""` if zero |
| `dueDate` | string | `2006-01-02`, or `""` if zero |
| `labels` | string[] | **Never JSON `null`** — empty array if none |

Requested Jira fields (quota-irrelevant; do not trim): `summary`,
`status`, `priority`, `issuetype`, `assignee`, `reporter`, `created`,
`updated`, `duedate`, `project`, `labels`.

### Statuses — `200 application/json`

```json
{
  "statuses": [
    {
      "id": "3",
      "name": "In Progress",
      "statusCategory": { "id": 4, "key": "indeterminate", "name": "In Progress" }
    }
  ],
  "categories": [
    { "id": 1, "key": "undefined", "name": "No Category" },
    { "id": 2, "key": "new", "name": "To Do" },
    { "id": 4, "key": "indeterminate", "name": "In Progress" },
    { "id": 3, "key": "done", "name": "Done" }
  ]
}
```

Category `id` is JSON number; status `id` is JSON string. Real Cloud keys
are `undefined` / `new` / `indeterminate` / `done` — **not** the dummy
keys in some Atlassian docs.

Connect JWT (`cloud`): succeeds with **no personal Jira connection** (bot
JWT). Cloud OAuth2 (`cloud-oauth`): requires the admin’s own connection;
otherwise `not_connected`. Do **not** use `AdminEmail` / `AdminAPIToken`.

## JSON error convention (new routes only)

Body: `{"error":"<code>","message":"<text>"}` with
`Content-Type: application/json`. Existing plugin routes stay
`text/plain`. Do not change `doFetch` / `doFetchWithResponse` behavior
for old routes (W2 must add an **additive** JSON-error helper).

| `error` | HTTP | When |
|---------|------|------|
| `not_connected` | 401 | User (or OAuth2 admin) has no Jira connection (`kvstore.ErrNotFound` from `LoadConnection`) |
| `not_authorized` | 403 | Non-admin caller on `/rhs/statuses` |
| `not_cloud` | 400 | Server/DC instance, or client is not Cloud |
| `rate_limited` | 429 | Cloud 429 retries exhausted, or `RateLimit-Reason: jira-quota-global-based` (no retry) |
| `invalid_request` | 400 | Missing/unknown `instance_id`; sort outside `updated`\|`created`; tab not in resolved list; category operand missing from **cached** `/statuscategory` keys (including Assigned when cache omits `done`) |
| `internal_error` | 500 | Everything else. JSON `message` is the stable string `internal error` |

Stable messages (parse `error`, not `message`, except for display):

- `not_connected` → `Jira account is not connected`
- `rate_limited` → `Jira is rate limiting requests, try again shortly`
- `not_cloud` → `Jira RHS is available for Jira Cloud only`
- `not_authorized` → `not authorized`
- `invalid_request` → Go `err.Error()` (includes wrapped detail)
- `internal_error` → `internal error`

## `rhs_enabled`

- Manifest: `plugin.json` key `EnableJiraRHS`, type `bool`, **default
  `false`**. Cloud-only in help text.
- Go: `externalConfig.EnableJiraRHS` with tag **`json:"enablejirarhs"`**
  (all lowercase — `LoadPluginConfiguration` lowercases keys).
- Webapp: `GET /api/v2/settingsinfo` JSON field **`rhs_enabled`**
  (boolean), sourced from `conf.EnableJiraRHS`, next to `ui_enabled`.
- Milestone 2 reads it the same way `webapp/src/plugin.tsx` reads
  `settings.ui_enabled` today (`plugin.tsx` ~line 50). App Bar / RHS
  register only when `rhs_enabled` is true **and** a Cloud instance
  exists. **Do not** hide `RHSStatusTabs` behind this flag.

## `RHSStatusTabs` config shape

- Manifest: `plugin.json` key **`RHSStatusTabs`**, `"type": "custom"`,
  `"display_name": "Jira RHS status tabs"`. **No `default`.** Without this
  key, `registerAdminConsoleCustomSetting('RHSStatusTabs', …)` has nothing
  to bind to.
- Go: `externalConfig.RHSStatusTabs map[string][]RHSTabEntry`
  `json:"rhsstatustabs"` (lowercase tag). Map key = Cloud instance id.
- Nested entry (camelCase tags — only the **top-level** key is
  lowercased):

```json
{
  "kind": "category",
  "key": "indeterminate",
  "name": "In Progress"
}
```

| Field | Used when | JSON |
|-------|-----------|------|
| `kind` | always | `assigned` \| `category` \| `status` |
| `key` | `kind=category` | Cloud category key (`indeterminate`, `new`, `done`, `undefined`) |
| `id` | `kind=status` | Numeric status id as string |
| `name` | always | Display label |

**Assigned is not stored.** It is always-on, not a removable saved entry.
The persisted map holds only extra category/status tabs. `kind: assigned`
in saved config is ignored by `resolveTabs` (Assigned is prepended
anyway).

Register: `registry.registerAdminConsoleCustomSetting('RHSStatusTabs', Component)`
inside `setupUILater()`, **unconditionally** (not behind `rhs_enabled`).

## D2 — virtual seed / picker constraint (LOAD-BEARING)

Milestone 1 **does not write** plugin config on instance install. Seeding
is read-time only (`rhsDefaultTabs()` / `resolveTabs` on empty stored
list):

1. `{ "kind": "assigned", "name": "Assigned" }` — locked, not deselectable
2. `{ "kind": "category", "key": "indeterminate", "name": "In Progress" }`

A never-configured instance therefore stores **nothing**. The System
Console custom setting receives an **empty `value`** (`undefined` /
`null` / `{}` / missing or empty array for the selected instance).

**The Phase 2 / W5 picker MUST:**

1. **Render Assigned (locked) and In Progress as selected when `value` is
   empty.**
2. **Must not call `onChange` (or `setSaveNeeded`) to materialize that
   seed.** Doing so dirties the form on mere page load and can write
   config the server does not need.
3. **Must not call `onChange` while disconnected** (`not_connected` on
   `/rhs/statuses` for OAuth2 without a personal connection). If it did,
   a System Console Save of an unrelated setting would write an empty
   list over the admin’s configured tabs. Keep stored chips visible;
   disable the control; show a connect message. Connect JWT instances do
   not hit this path (bot client).

This is Gate 8 of Milestone 2. A component that “helpfully” writes the
default on mount is a **bug**, not a convenience.

## What Milestone 2 must not do

- Rebuild JQL in the browser (`assignee = currentUser() AND statusCategory …`).
- Hide vanished tabs itself — trust `tabs` from `/rhs/issues`.
- Use `instance.GetURL()` / `api.atlassian.com` for browse links — use
  `browseUrl` from the DTO.
- Call go-jira-style `get-search-issues` for this feature (Cloud 410).
- Gate `/rhs/statuses` or the admin setting on `rhs_enabled`.
- Introduce `en.json` / `FormattedMessage` (i18n is future milestone F1).
- Add `eslint-disable` or `max-lines` overrides (W6).

## Rate-limit notes for W2

Cloud `/search/jql` costs `1 + issues returned` (21 points at page size
20). The webapp in-flight debounce (identical `{instance, tab, sort}`)
must produce **no** second network call. Global-quota 429 maps to JSON
`rate_limited` and is not retried server-side.

## File map (server — already landed)

`server/rhs_*.go`, `server/client_cloud_rhs.go`, `plugin.json`
(`EnableJiraRHS`, `RHSStatusTabs`), `server/plugin.go` config + cache,
`server/user.go` (`rhs_enabled`), `server/http.go` (+2 routes).
`server/issue.go` / `SearchIssues` / existing `getClient` sites
untouched except one new call inside `resolveRHSUserClient`.
