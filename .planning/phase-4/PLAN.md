# Phase 4 Plan: HTTP layer

> Prescriptive implementation plan for **Phase 4 only** (tasks 4.1–4.5). An
> Implementation Engineer should be able to land this without making design
> decisions. Do not implement later phases from this file.
>
> **Do not write production/test Go until this plan is followed as written.**
> This document is the plan; it is not the code.
>
> **Gate 3 carry-forward (RE3).** Phase 4 MUST pass
> `validCategoryKeysFrom(cached categories)` into `buildTabJQL`. A hardcoded
> four-key map at the HTTP call site would bypass the omit-`done` guarantee
> and re-open risk #36. Treat cache-returned `*JiraStatus` /
> `*JiraStatusCategory` as read-only (do not mutate).

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase 4 (source of truth for WHAT)
- **Spec:** `planner/projects/jira-plugin-rhs/ideas/001-show-tickets-rhs/spec.md`
  (§ "Server API (net-new)"; JSON errors; page size 20)
- **Context:** `planner/projects/jira-plugin-rhs/context.md`
  (§ HTTP errors; `getClient`; `getClientForBot`; `GetJiraBaseURL`)
- **Orchestration:** Gate 4 / Step 4 of `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `285fe8d` (Phase 3 domain logic — already committed)
- **Package:** `package main` under `server/` (module
  `github.com/mattermost/mattermost-plugin-jira`, `go.mod` at repo root)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer for all of Phase 4.** T4.2 and T4.3 share
  `server/rhs.go`. T4.1 is a prerequisite for T4.4.

## Scope

**In scope:**

- Append `RHSIssue`, `RHSIssueStatus`, and `ErrRHSNotCloud` to
  `server/rhs_types.go`
- Create `server/rhs.go` (Cloud client resolution, DTO normalization, issue
  orchestration)
- Create `server/rhs_http.go` (JSON error helper, admin gate, both handlers)
- Modify `server/http.go`: two route constants + two `apiRouter` registrations
- Create `server/rhs_http_test.go` (all names `TestRHSHTTP*`)

**Out of scope (do not do these):**

- Any file under `webapp/`
- Changing `respondErr` / `respondJSON` / `checkAuth` / `handleResponse`
- Touching `getClient` (`server/issue.go:1568-1582`) or any of its **21**
  existing production call sites
- Touching `SearchIssues` (`server/client.go:304-314`) or
  `httpGetSearchIssues` / `get-search-issues`
- Widening the shared `Client` / `SearchService` / `Instance` interfaces
- Using `AdminEmail` / `AdminAPIToken` / `SetAdminAPITokenRequestHeader` /
  `GetProjectListWithAPIToken`
- Changing `respondErr` to JSON (new helper only; existing routes stay
  `text/plain`)
- Checking `EnableJiraRHS` on these routes (webapp gates registration;
  `/rhs/statuses` must work for the picker while the flag is off)
- `.planning/PHASE1_HANDOFF.md` (Phase 5)
- Server/DC search path
- Adding admin middleware (one helper, one handler)

---

## Gate 3 carry-forward (RE3) — read this first

Phase 3 locked `buildTabJQL(tab, sort, validCategoryKeys)` so **every**
`statusCategory` operand — including Assigned’s hardcoded `done` — is
checked against the **caller-supplied map**, not a hardcoded four-key
allowlist. The omit-`done` test is how that is proven.

Phase 4 is the first production caller. The only legal call is:

```go
jql, err := buildTabJQL(selected, sort, validCategoryKeysFrom(entry.categories))
```

where `entry` is the `*rhsStatusCacheEntry` returned by
`getInstanceStatuses`. **Illegal:**

```go
// FORBIDDEN — bypasses the omit-done guarantee
valid := map[string]bool{"new": true, "indeterminate": true, "done": true, "undefined": true}
jql, err := buildTabJQL(selected, sort, valid)
```

`copyRHSStatusCacheEntry` (`server/rhs_cache.go:27-38`) copies slice
headers only. The `*JiraStatus` / `*JiraStatusCategory` pointers alias the
cached objects. **Do not mutate them** (no `entry.statuses[i].Name = …`, no
in-place append on those slices). `validCategoryKeysFrom` and `resolveTabs`
already read keys/ids only.

`TestRHSHTTPGetIssuesUsesCachedCategoryKeys` is the HTTP-layer proof: a
cache whose categories omit `done` must make the Assigned tab return JSON
`invalid_request` and must **not** call `SearchJQL`. If the handler used a
hardcoded four-key map, that test would pass while Assigned silently
widened. Gate 4 executes this test; it is not inspect-only.

---

## Research (read this before coding)

Line numbers are as of **`285fe8d`** (Phase 3 committed). Parent
`.planning/PLAN.md` still cites some pre-Phase-1 numbers. **Use this
section, not the parent’s stale counts.**

### 1. Routes, router, auth, responses — `server/http.go`

Route constants occupy **`server/http.go:28-71`**. Last constant is
`routeOAuth2Complete = "/oauth2/complete.html"` at **`:70`**. Insert the two
RHS constants **immediately after that line**, before the closing `)` at
`:71`.

`makeAPIRoute` is **`:88-90`**: `return routeAPI + path` with
`routeAPI = "/api/v2"` (`:38`). Tests hit
`makeAPIRoute(routeAPIRHSIssues)` → `/api/v2/rhs/issues`.

`initializeRouter` is **`:92-169`**. `apiRouter` is created at **`:106`**
(`PathPrefix(routeAPI)`). Subscription-template registrations end at
**`:168`**. Insert the two RHS `HandleFunc` lines **after `:168`**, before
the function’s `}`.

Existing idiom (copy this exactly, including `checkAuth` + `handleResponse`
+ `Methods(http.MethodGet)`):

```117:117:server/http.go
	apiRouter.HandleFunc(routeAPIGetSearchIssues, p.checkAuth(p.handleResponse(p.httpGetSearchIssues))).Methods(http.MethodGet)
```

`respondErr` is **`:217-220`**. It calls `http.Error` — **plain text**. Do
**not** edit this function. New routes use `respondJSONErr`.

```217:232:server/http.go
func respondErr(w http.ResponseWriter, code int, err error) (int, error) {
	http.Error(w, err.Error(), code)
	return code, err
}

func respondJSON(w http.ResponseWriter, obj interface{}) (int, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, errors.WithMessage(err, "failed to marshal response"))
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		return http.StatusInternalServerError, errors.WithMessage(err, "failed to write response")
	}
	return http.StatusOK, nil
}
```

`checkAuth` is **`:317-326`**. It reads the literal header
`"Mattermost-User-ID"` (not the constant) and on failure calls
`http.Error(w, "Not authorized", http.StatusUnauthorized)` — **plain
text**. Do **not** change it. Missing-header tests on the new routes must
still see that plain-text 401.

`handleResponse` is **`:328-334`**. It calls the `(int, error)` handler,
then `logResponse`. The handler must write the body itself.
`logResponse` (`:352-363`) calls `p.client.Log.Warn` on any non-OK status
with a non-nil error (msg `"ERROR: "` plus 5 key/value pairs = **11
args**). Error-path tests must mock `LogWarn` or plugintest will panic.

`ServeHTTP` (`:171-173`) is `p.router.ServeHTTP`. Tests go through
`p.ServeHTTP(&plugin.Context{}, w, request)`, not the handler function
directly.

### 2. `getClient` — do not touch

```1568:1582:server/issue.go
func (p *Plugin) getClient(instanceID, mattermostUserID types.ID) (Client, Instance, *Connection, error) {
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		return nil, nil, nil, err
	}
	connection, err := p.userStore.LoadConnection(instance.GetID(), mattermostUserID)
	if err != nil {
		return nil, nil, nil, err
	}
	client, err := instance.GetClient(connection)
	if err != nil {
		return nil, nil, nil, err
	}
	return client, instance, connection, nil
}
```

Parent PLAN.md says “13 existing call sites.” That count is **stale**
(issue.go only). Production call sites as of `285fe8d`:

| File | Lines | Count |
|------|-------|-------|
| `server/issue.go` | 221, 320, 537, 649, 738, 826, 859, 919, 1050, 1062, 1103, 1494, 1605 | 13 |
| `server/subscribe.go` | 1185, 1258, 1394, 1426, 1456, 1503 | 6 |
| `server/autocomplete_search.go` | 23, 56 | 2 |
| **Total production** | | **21** |

Do not add a 22nd site that changes behavior of those 21. The user-path
RHS helper **may call** `p.getClient` (that is the prescribed user
resolver). It must not edit the function or any existing caller.

`LoadInstance` (`server/kv.go:465-474`) returns wrapped `kvstore.ErrNotFound`
for an empty id (`"no instance specified"`) and for a missing key.
`LoadConnection` (`server/kv.go:167-176`) wraps the store error with
`"failed to load connection for Mattermost user ID:…"`. `pkg/errors` v0.9.1
implements `Unwrap`, so `errors.Is(err, kvstore.ErrNotFound)` works through
both wraps.

**Mapping trap:** both instance-missing and connection-missing are
`ErrNotFound`. Blindly mapping every `getClient` error to `not_connected`
would call an unknown `instance_id` “not connected.” Load the instance
**first** (empty/unknown → `invalid_request`); then call `getClient`. After
a successful `LoadInstance`, a `getClient` `ErrNotFound` is the
connection.

### 3. `SearchIssues` / `get-search-issues` — do not touch

```43:43:server/http.go
	routeAPIGetSearchIssues                     = "/get-search-issues"
```

```117:117:server/http.go
	apiRouter.HandleFunc(routeAPIGetSearchIssues, p.checkAuth(p.handleResponse(p.httpGetSearchIssues))).Methods(http.MethodGet)
```

```903:916:server/issue.go
func (p *Plugin) httpGetSearchIssues(w http.ResponseWriter, r *http.Request) (int, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	instanceID := r.FormValue("instance_id")
	// ...
	result, err := p.GetSearchIssues(...)
	if err != nil {
		return respondErr(w, http.StatusInternalServerError, err)
	}
	return respondJSON(w, result)
}
```

`GetSearchIssues` (`:918-973`) calls `p.getClient` at `:919` and
`client.SearchIssues` at `:955`. That is go-jira’s dead
`rest/api/2/search` (Cloud 410). **Do not retrofit it.** RHS search goes
through `jiraCloudClient.SearchJQL` only.

Handler shape to copy: extract query params, delegate, `respondJSON` /
error, return `(int, error)`.

### 4. Connect JWT bot vs Cloud OAuth2 personal — admin path

Instance types (`server/instance.go:14-18`):

```14:18:server/instance.go
const (
	CloudInstanceType      = InstanceType("cloud")
	ServerInstanceType     = InstanceType("server")
	CloudOAuthInstanceType = InstanceType("cloud-oauth")
)
```

`IsCloudInstance()` (`:72-74`) is true for `cloud` and `cloud-oauth`. The
admin path must **type-switch the concrete type**, not only
`Common().Type`, because `getClientForBot` lives only on `*cloudInstance`.

```238:253:server/instance_cloud.go
func (ci *cloudInstance) getClientForBot() (*jira.Client, error) {
	conf := ci.getConfig()
	jwtConf := &ajwt.Config{
		Key:          ci.AtlassianSecurityContext.Key,
		ClientKey:    ci.AtlassianSecurityContext.ClientKey,
		SharedSecret: ci.AtlassianSecurityContext.SharedSecret,
		BaseURL:      ci.AtlassianSecurityContext.BaseURL,
	}
	httpClient := jwtConf.Client()
	httpClient = utils.WrapHTTPClient(httpClient,
		utils.WithRequestSizeLimit(conf.maxAttachmentSize),
		utils.WithResponseSizeLimit(conf.maxAttachmentSize))
	return jira.NewClient(httpClient, jwtConf.BaseURL)
}
```

Returns a raw `*jira.Client`. Wrap with `newCloudClient`
(`server/client_cloud.go:25-31`) which returns the shared `Client`
interface holding `*jiraCloudClient`. Then type-assert to
`rhsCloudClient`.

**No personal connection is involved.** This is the whole point of the
Connect JWT admin path.

`cloudOAuthInstance` has **no** `getClientForBot`. Its `GetClient`
(`server/instance_cloud_oauth.go:126-132`) requires a `*Connection` with an
OAuth2 token (or a carried-over JWT user connection). There is a
`JWTInstance *cloudInstance` field for install migration
(`:36`) — **do not** fall through to it for the admin picker.

**Do not use** `AdminEmail` / `AdminAPIToken` (`server/plugin.go:96-100`).
They are a single global Basic-auth credential, already wrong with more
than one Cloud instance (`SetAdminAPITokenRequestHeader` at
`server/utils.go:253-271`; `GetProjectListWithAPIToken` at
`server/issue.go:1862`). Context.md and the spec both forbid this for the
picker.

### 5. `GetJiraBaseURL()` vs `GetURL()` — browse links

| Type | `GetURL()` | `GetJiraBaseURL()` |
|------|------------|--------------------|
| `*cloudInstance` | site (`ASC.BaseURL`) (`instance_cloud.go:184-189`) | same as `GetURL()` |
| `*cloudOAuthInstance` | `https://api.atlassian.com/ex/jira/{cloudid}` (`instance_cloud_oauth.go:224-226`) | `ci.JiraBaseURL` (the site) (`:228-229`) |
| `*serverInstance` | instance id (`instance_server.go:51-56`) | same as `GetURL()` |
| `testInstance` | instance id (`kv_mock_test.go:44-48`) | same as `GetURL()` |

Browse URL is `{GetJiraBaseURL()}/browse/{KEY}`. Using `GetURL()` on
`cloud-oauth` yields `https://api.atlassian.com/ex/jira/…/browse/TES-41`,
which 404s for the user. Existing production browse links already use
`GetJiraBaseURL()` (`issue.go:435`, `:1180`, `:1395`, `:1479`).

### 6. Admin permission precedent — `subscribe.go:1076`

There is **no** admin HTTP middleware in this plugin. `checkAuth` only
asserts a Mattermost user id is present (`context.md` § Admin-context Jira
access). The only `PermissionManageSystem` check:

```1076:1078:server/subscribe.go
		if !p.client.User.HasPermissionTo(userID, model.PermissionManageSystem) {
			return errors.New("is not system admin")
		}
```

`pluginapi` (`UserService.HasPermissionTo` at
`mattermost/server/public@v0.1.12/pluginapi/user.go:199-201`) delegates to
`api.HasPermissionTo(userID string, permission *model.Permission) bool`.
Tests mock `api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true|false)`
(`server/http_test.go:80`, `:286`).

`setupTestPlugin` does **not** mock `HasPermissionTo`. Every
`/rhs/statuses` test must.

### 7. Test harness — `setupTestPlugin`, `testClient`, KV, ServeHTTP

`setupTestPlugin` (`server/issue_test.go:99-114`): `SetAPI`,
`initializeRouter`, `getMockInstanceStoreKV(1)` (stores `testInstance1` at
`https://jiraurl1.com`), `getMockUserStoreKV()`, `pluginapi.NewClient`,
`conf.Secret`. Does **not** call `OnActivate`, does **not** init
`rhsStatusCache` (cache methods lazy-init), does **not** mock `LogWarn` or
`HasPermissionTo`.

`testClient` (`server/issue_test.go:39-45`) embeds the five `Client`
interfaces and implements a few issue methods. It does **not** implement
`ListStatuses` / `ListStatusCategories` / `SearchJQL`.
`testInstance.GetClient` (`kv_mock_test.go:68-70`) returns `testClient{}`.
A type-assert to `rhsCloudClient` on that value **fails** — that is the
`not_cloud` test.

`getMockUserStoreKV().LoadConnection` (`command_test.go:48-54`) returns a
**custom** `errors.Errorf("TESTING connection … not found")` for unknown
users — **not** `kvstore.ErrNotFound`. `mockUserStore.LoadConnection`
(`kv_mock_test.go:86-92`) always succeeds. The `not_connected` test **must**
install a store that returns `kvstore.ErrNotFound` (the production
sentinel). Do not rely on `getMockUserStoreKV`’s testing error.

`makeTestKVStore` (`server/mock_kv_store.go:13`) is the in-memory KV used
when a test talks to `api.KV*`. Phase 4 tests swap `instanceStore` /
`userStore` and do not need it unless a test uses the real `store`.

ServeHTTP + user header (`server/http_test.go:481-486`; Phase 1
`rhs_types_test.go:173`):

```go
request := httptest.NewRequest(http.MethodGet, makeAPIRoute(routeAPIRHSIssues)+"?"+q, nil)
request.Header.Set(HeaderMattermostUserID, userID)
w := httptest.NewRecorder()
p.ServeHTTP(&plugin.Context{}, w, request)
```

`HeaderMattermostUserID` is `"Mattermost-User-Id"` (`constants.go:7`).
HTTP headers are case-insensitive; `checkAuth`’s `"Mattermost-User-ID"`
matches.

Connected user ids that exist in `getMockUserStoreKV`: `"connected_user"`,
`"1"` (`mockUserIDWithNotifications`). Prefer `"connected_user"`.

### 8. Landed Phase 3 APIs the handlers call

`buildTabJQL` (`server/rhs_jql.go:52-84`) — signature locked. Assigned
validates `statusCategoryKeyDone` against the **passed-in** map.

`validCategoryKeysFrom` (`:8-17`) — builds the map from cached categories.
Skip nil entries and empty keys.

`resolveTabs` (`:86-112`) — returns `[]RHSTabEntry` (there is no
`RHSTab` type). Assigned always first. Nil/empty configured → virtual
seed. Vanished status ids / category keys hidden.

`getInstanceStatuses` (`server/rhs_cache.go:52-86`) — `(instanceID types.ID, client rhsStatusLister)`.
Returns a shallow copy. `rhsStatusLister` (`:22-25`) is
`ListStatuses()` + `ListStatusCategories()`. Phase 4’s `rhsCloudClient`
must include those two plus `SearchJQL` so the same value can be passed
to the cache.

`SearchJQL` (`server/client_cloud_rhs.go:86-88`) —
`SearchJQL(CloudSearchParams) (*CloudSearchResult, error)`. Returns
`ErrRateLimited` on global-quota 429 or retry exhaustion.

`jiraCloudClient` methods are **value receivers**. `newCloudClient`
returns `Client` (`*jiraCloudClient`). Type-assert:
`newCloudClient(raw).(rhsCloudClient)`.

### 9. Confirmed absent files

As of `285fe8d`:

| Path | Exists? |
|------|---------|
| `server/rhs.go` | **No** — create |
| `server/rhs_http.go` | **No** — create |
| `server/rhs_http_test.go` | **No** — create |

`Glob server/rhs*.go` today: `rhs_types.go`, `rhs_types_test.go`,
`rhs_jql.go`, `rhs_jql_test.go`, `rhs_cache.go`, `rhs_cache_test.go`.

### 10. `go-jira` fields for the DTO

`jira.Issue` (`andygrunwald/go-jira@v1.16.0/issue.go:43-53`) has `Key` and
`Fields *IssueFields`. `IssueFields` (`:104-138`): `Summary`,
`Status *Status`, `Priority *Priority`, `Type IssueType` (json
`issuetype`), `Assignee *User`, `Reporter *User`, `Created Time`,
`Updated Time`, `Duedate Date`, `Project Project`, `Labels []string`.

`Status.StatusCategory.Key` is the pill color key (`status.go:15-22`,
`statuscategory.go:14-19`).

`Time` / `Date` are `time.Time` wrappers. Convert with `time.Time(t)`.
Emit RFC3339 for created/updated; `2006-01-02` for due. Zero → `""`.

Nil-safe: `Fields == nil`, `Status == nil`, `Priority == nil`,
`Assignee == nil`, `Reporter == nil`.

### 11. Connect JWT test trap — `maxAttachmentSize == 0`

`getClientForBot` wraps the JWT HTTP client with
`WithResponseSizeLimit(conf.maxAttachmentSize)`.
`NewLimitedReadCloser` (`server/utils/limited_readcloser.go:26-28`)
applies `io.LimitReader` whenever `limit >= 0`. A zero limit yields a
**0-byte** body, so `ListStatuses` decode fails.

`setupTestPlugin` never sets `maxAttachmentSize`. The Connect JWT admin
test **must** set `conf.maxAttachmentSize = defaultMaxAttachmentSize`
(or any positive value) before calling the handler.

### 12. Pitfall confirmation vs parent `.planning/PLAN.md`

| Claim in parent PLAN.md | Current worktree (`285fe8d`) | Drift? |
|-------------------------|------------------------------|--------|
| `respondErr` `:217-220` | matches | **No** |
| `checkAuth` `:267` (context.md) | **`:317-326`** | **Yes — use 317–326** |
| Route block `:28-71`; insert after `routeOAuth2Complete` `:71` | last const is **`:70`**; `)` is `:71` | Cosmetic — insert after `:70` |
| `initializeRouter` `:92-169`; insert after templates `:168` | matches | **No** |
| 13 `getClient` call sites | **21** production | **Yes — do not touch any of the 21** |
| `getClient` `issue.go:1568-1582` | matches | **No** |
| `httpGetSearchIssues` `:903-916` | matches | **No** |
| `SearchIssues` `:304-314` | matches | **No** |
| `getClientForBot` `instance_cloud.go:238-253` | matches | **No** |
| `newCloudClient` `client_cloud.go:25-31` | matches | **No** |
| `HasPermissionTo` `subscribe.go:1076` | matches | **No** |
| `GetJiraBaseURL` / `GetURL` oauth `:228-235` | `GetURL` `:224-226`; `GetJiraBaseURL` `:228-229` | Cosmetic |
| `rhs.go` / `rhs_http.go` exist | **neither exists** | **No** (create) |
| `RHSIssue` already in `rhs_types.go` | **not present** (Phase 1/2/3 left it out) | **No** (append) |
| `setupTestPlugin` `:99-114`; `testClient` `:39-45` | matches | **No** |
| `makeTestKVStore` `mock_kv_store.go:13` | matches | **No** |

---

## Decisions this phase locks (do not reopen)

**P4-D1 — JSON errors on new routes only.** `respondJSONErr` writes
`{"error":"<code>","message":"<text>"}` with
`Content-Type: application/json`. `respondErr` stays `text/plain`.
`checkAuth`’s missing-header 401 stays `text/plain`.

**P4-D2 — Locked error codes and HTTP statuses.**

| `error` | Status | When |
|---------|--------|------|
| `not_connected` | 401 | `LoadConnection` → `kvstore.ErrNotFound` (user path or OAuth2 admin) |
| `not_authorized` | 403 | `/rhs/statuses` and `HasPermissionTo` is false |
| `not_cloud` | 400 | instance is not `*cloudInstance` / `*cloudOAuthInstance`, or `Client` fails `rhsCloudClient` assert |
| `rate_limited` | 429 | `errors.Is(err, ErrRateLimited)` |
| `invalid_request` | 400 | empty/unknown `instance_id`; bad sort; unknown tab kind; tab not in resolved list; `ErrInvalidStatusCategory` (including Assigned when cache omits `done`); `ErrInvalidRHSTab` |
| `internal_error` | 500 | everything else. JSON message is the stable string `internal error`; the original `error` is returned for `logResponse` |

**P4-D3 — User path calls `getClient` then type-asserts.** After a
successful `LoadInstance`, `p.getClient(instanceID, userID)` then
`client.(rhsCloudClient)`. Failed assert → `not_cloud`. Do not add RHS
methods to the shared `Client` tree.

**P4-D4 — Admin path type-switches Instance.** `*cloudInstance` →
`getClientForBot()` → `newCloudClient` → assert. `*cloudOAuthInstance` →
`LoadConnection(admin)` → `GetClient(connection)` → assert. Default →
`not_cloud`. No `getClient` on the Connect JWT admin path (that would
require a personal connection). No `JWTInstance` fallback. No
`AdminAPIToken`.

**P4-D5 — Cached category keys, not a hardcoded map.** See RE3. The
omit-`done` HTTP test is mandatory.

**P4-D6 — Cache pointers are read-only.** Do not mutate
`entry.statuses` / `entry.categories` elements.

**P4-D7 — BrowseURL from `GetJiraBaseURL()` only.**
`strings.TrimRight(base, "/") + "/browse/" + key`.

**P4-D8 — Query params are snake_case; body fields are camelCase.**
Params: `instance_id`, `tab_kind`, `tab_key`, `tab_id`, `sort`,
`next_page_token`. Empty `tab_kind` → `assigned`. Empty `sort` →
`updated` **before** `buildTabJQL` (which rejects empty). Page size is
the constant `20`, not a query param. Ignore `page_size` / `limit` if
sent.

**P4-D9 — `/rhs/statuses` returns statuses **and** categories.** The
picker needs category options (In Progress is a category tab). Shape:
`{"statuses":[...],"categories":[...]}` using the existing
`JiraStatus` / `JiraStatusCategory` json tags. Use `getInstanceStatuses`
(the cache), do not bypass it.

**P4-D10 — Do not gate on `EnableJiraRHS`.** The System Console picker
registers outside that flag (Milestone 2 W5.2).

**P4-D11 — No new middleware.** `checkSystemAdmin` is a function called
from `httpRHSListStatuses` only.

**P4-D12 — One implementer.** Order below.

---

## Implementation order (single engineer)

1. Append DTO + `ErrRHSNotCloud` to `rhs_types.go`
2. T4.1 — `rhs_http.go` helpers (`respondJSONErr`, codes, `checkSystemAdmin`,
   `respondRHSErr`)
3. T4.2 + T4.3 — `rhs.go` (interfaces, resolvers, normalize, orchestration)
4. T4.4 — handlers in `rhs_http.go` + route constants/registrations in
   `http.go`
5. T4.5 — `rhs_http_test.go`
6. Run Gate 4 commands (including the cached-keys test and the blast-radius
   diffs)

---

## Exact type additions (`server/rhs_types.go`) — do first

File is **97 lines**. Keep lines 1–97 unchanged. `errors` is already
imported. After `ErrInvalidRHSTab`, append:

```go
// ErrRHSNotCloud is returned when the instance or client is not Jira Cloud.
// Phase 4 maps this to JSON not_cloud.
var ErrRHSNotCloud = errors.New("jira rhs is available for jira cloud only")

// RHSIssueStatus is the status fragment of the RHS issue DTO.
type RHSIssueStatus struct {
	Name        string `json:"name"`
	CategoryKey string `json:"categoryKey"`
}

// RHSIssue is the normalized issue DTO on GET /api/v2/rhs/issues only.
// get-search-issues keeps raw []jira.Issue.
type RHSIssue struct {
	Key       string         `json:"key"`
	Summary   string         `json:"summary"`
	BrowseURL string         `json:"browseUrl"`
	Status    RHSIssueStatus `json:"status"`
	Priority  string         `json:"priority"`
	IssueType string         `json:"issueType"`
	Project   string         `json:"project"`
	Assignee  string         `json:"assignee"`
	Reporter  string         `json:"reporter"`
	Created   string         `json:"created"`
	Updated   string         `json:"updated"`
	DueDate   string         `json:"dueDate"`
	Labels    []string       `json:"labels"`
}
```

Do not redefine Phase 1–3 types.

---

## Exact JSON errors and admin gate (`server/rhs_http.go`) — Task 4.1

Create the file. `package main`. Header:

```go
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
```

Year is **2017**.

### Constants and error body

```go
const (
	rhsErrNotConnected   = "not_connected"
	rhsErrRateLimited    = "rate_limited"
	rhsErrNotAuthorized  = "not_authorized"
	rhsErrNotCloud       = "not_cloud"
	rhsErrInvalidRequest = "invalid_request"
	rhsErrInternal       = "internal_error"

	queryRHSTabKind       = "tab_kind"
	queryRHSTabKey        = "tab_key"
	queryRHSTabID         = "tab_id"
	queryRHSSort          = "sort"
	queryRHSNextPageToken = "next_page_token"

	rhsDefaultSort = "updated"
)

type rhsJSONError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
```

### `respondJSONErr`

```go
func respondJSONErr(w http.ResponseWriter, status int, errorCode, message string) (int, error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encErr := json.NewEncoder(w).Encode(rhsJSONError{Error: errorCode, Message: message})
	if encErr != nil {
		return status, errors.Wrap(encErr, errorCode+": "+message)
	}
	return status, errors.Errorf("%s: %s", errorCode, message)
}
```

Must `WriteHeader(status)` **before** encoding. Must set
`Content-Type: application/json`. Do not call `respondErr`. Do not call
`http.Error`.

### `checkSystemAdmin`

```go
func (p *Plugin) checkSystemAdmin(userID string) bool {
	return p.client.User.HasPermissionTo(userID, model.PermissionManageSystem)
}
```

Copy the `subscribe.go:1076` call. No extra roles. No channel permission.

### `respondRHSErr` — domain → JSON

```go
func respondRHSErr(w http.ResponseWriter, err error) (int, error) {
	switch {
	case errors.Is(err, kvstore.ErrNotFound):
		return respondJSONErr(w, http.StatusUnauthorized, rhsErrNotConnected, "Jira account is not connected")
	case errors.Is(err, ErrRateLimited):
		return respondJSONErr(w, http.StatusTooManyRequests, rhsErrRateLimited, "Jira is rate limiting requests, try again shortly")
	case errors.Is(err, ErrRHSNotCloud):
		return respondJSONErr(w, http.StatusBadRequest, rhsErrNotCloud, "Jira RHS is available for Jira Cloud only")
	case errors.Is(err, ErrInvalidStatusCategory),
		errors.Is(err, ErrInvalidRHSSort),
		errors.Is(err, ErrUnknownRHSTabKind),
		errors.Is(err, ErrInvalidRHSTab):
		return respondJSONErr(w, http.StatusBadRequest, rhsErrInvalidRequest, err.Error())
	default:
		status, _ := respondJSONErr(w, http.StatusInternalServerError, rhsErrInternal, "internal error")
		return status, err
	}
}
```

Use `github.com/pkg/errors` (already the package alias in this repo) so
`errors.Is` unwraps `Wrap` / `Wrapf` / `WithMessage`. Import
`kvstore` as `github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore`.

`not_authorized` is **not** mapped here — the handler calls
`respondJSONErr` directly after `checkSystemAdmin` fails.

Required imports for this file (handlers added in T4.4 will need `net/http`
too; add them now):

```go
import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)
```

`goimports` will group them.

---

## Exact client resolution + DTO (`server/rhs.go`) — Tasks 4.2 / 4.3

Create the file. `package main`. Header as above.

### Interface (T4.2)

```go
const rhsIssuesPageSize = 20

var rhsSearchFields = []string{
	"summary", "status", "priority", "issuetype", "assignee", "reporter",
	"created", "updated", "duedate", "project", "labels",
}

// rhsCloudClient is the Cloud-only surface the HTTP layer type-asserts to.
// Do not add these methods to the shared Client / SearchService tree.
type rhsCloudClient interface {
	rhsStatusLister
	SearchJQL(params CloudSearchParams) (*CloudSearchResult, error)
}
```

Embedding `rhsStatusLister` (Phase 3) is required so the same value can be
passed to `getInstanceStatuses`.

### User-path resolver (T4.2)

```go
func (p *Plugin) resolveRHSUserClient(instanceID, mattermostUserID types.ID) (Instance, rhsCloudClient, error) {
	if instanceID == "" {
		return nil, nil, errors.Wrap(ErrInvalidRHSTab, "instance_id is required")
	}
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		if errors.Is(err, kvstore.ErrNotFound) {
			return nil, nil, errors.Wrap(ErrInvalidRHSTab, "unknown instance_id")
		}
		return nil, nil, err
	}

	client, instance, _, err := p.getClient(instance.GetID(), mattermostUserID)
	if err != nil {
		return nil, nil, err
	}
	rhs, ok := client.(rhsCloudClient)
	if !ok {
		return nil, nil, ErrRHSNotCloud
	}
	return instance, rhs, nil
}
```

`getClient` is the user-auth path (parent T4.2). The type assert is the
Cloud check — `jiraServerClient` and `testClient` both fail it.

Empty `instance_id` is rejected **before** `getClient` so it cannot become
`not_connected` via `LoadInstance`’s `"no instance specified"` wrap.

Unknown instance uses `ErrInvalidRHSTab` so `respondRHSErr` maps it to
`invalid_request`, not `not_connected`.

### Admin-path resolver (T4.2)

```go
func (p *Plugin) resolveRHSAdminClient(instance Instance, adminUserID types.ID) (rhsCloudClient, error) {
	switch inst := instance.(type) {
	case *cloudInstance:
		raw, err := inst.getClientForBot()
		if err != nil {
			return nil, err
		}
		rhs, ok := newCloudClient(raw).(rhsCloudClient)
		if !ok {
			return nil, ErrRHSNotCloud
		}
		return rhs, nil

	case *cloudOAuthInstance:
		connection, err := p.userStore.LoadConnection(inst.GetID(), adminUserID)
		if err != nil {
			return nil, err
		}
		client, err := inst.GetClient(connection)
		if err != nil {
			return nil, err
		}
		rhs, ok := client.(rhsCloudClient)
		if !ok {
			return nil, ErrRHSNotCloud
		}
		return rhs, nil

	default:
		return nil, ErrRHSNotCloud
	}
}
```

Rules:

- `*cloudInstance`: **no** `LoadConnection`, **no** `getClient`.
- `*cloudOAuthInstance`: **no** `getClientForBot`, **no** `inst.JWTInstance`,
  **no** `AdminEmail` / `AdminAPIToken`.
- `*serverInstance`, `*testInstance`, anything else → `ErrRHSNotCloud`.

### DTO normalization (T4.3)

```go
func rhsBrowseURL(instance Instance, issueKey string) string {
	return strings.TrimRight(instance.GetJiraBaseURL(), "/") + "/browse/" + issueKey
}

func rhsTimeRFC3339(t jira.Time) string {
	tt := time.Time(t)
	if tt.IsZero() {
		return ""
	}
	return tt.UTC().Format(time.RFC3339)
}

func rhsDate(d jira.Date) string {
	tt := time.Time(d)
	if tt.IsZero() {
		return ""
	}
	return tt.Format("2006-01-02")
}

func normalizeRHSIssue(instance Instance, issue jira.Issue) RHSIssue {
	out := RHSIssue{
		Key:       issue.Key,
		BrowseURL: rhsBrowseURL(instance, issue.Key),
		Labels:    []string{},
	}
	if issue.Fields == nil {
		return out
	}
	f := issue.Fields
	out.Summary = f.Summary
	out.IssueType = f.Type.Name
	out.Project = f.Project.Key
	out.Created = rhsTimeRFC3339(f.Created)
	out.Updated = rhsTimeRFC3339(f.Updated)
	out.DueDate = rhsDate(f.Duedate)
	if f.Labels != nil {
		out.Labels = f.Labels
	}
	if f.Status != nil {
		out.Status = RHSIssueStatus{
			Name:        f.Status.Name,
			CategoryKey: f.Status.StatusCategory.Key,
		}
	}
	if f.Priority != nil {
		out.Priority = f.Priority.Name
	}
	if f.Assignee != nil {
		out.Assignee = f.Assignee.DisplayName
	}
	if f.Reporter != nil {
		out.Reporter = f.Reporter.DisplayName
	}
	return out
}

func normalizeRHSIssues(instance Instance, issues []jira.Issue) []RHSIssue {
	out := make([]RHSIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, normalizeRHSIssue(instance, issue))
	}
	return out
}
```

`Labels` is never JSON `null`. Carry `statusCategory.key` through
(`categoryKey`) so Milestone 2 can color the pill without a second lookup.

**Never call `instance.GetURL()` in this file.**

### Tab pick + orchestration

```go
func pickRHSTab(tabs []RHSTabEntry, kind, key, id string) (RHSTabEntry, error) {
	if kind == "" {
		kind = string(RHSTabKindAssigned)
	}
	for _, tab := range tabs {
		if string(tab.Kind) != kind {
			continue
		}
		switch tab.Kind {
		case RHSTabKindAssigned:
			return tab, nil
		case RHSTabKindCategory:
			if tab.Key == key {
				return tab, nil
			}
		case RHSTabKindStatus:
			if tab.ID == id {
				return tab, nil
			}
		}
	}
	return RHSTabEntry{}, errors.Wrapf(ErrInvalidRHSTab, "tab %s/%s/%s is not in the resolved list", kind, key, id)
}

type rhsIssuesResult struct {
	Issues        []RHSIssue    `json:"issues"`
	Tabs          []RHSTabEntry `json:"tabs"`
	NextPageToken string        `json:"nextPageToken"`
	IsLast        bool          `json:"isLast"`
}

func (p *Plugin) getRHSIssues(instanceID, mattermostUserID types.ID, tabKind, tabKey, tabID, sort, nextPageToken string) (*rhsIssuesResult, error) {
	instance, client, err := p.resolveRHSUserClient(instanceID, mattermostUserID)
	if err != nil {
		return nil, err
	}

	entry, err := p.getInstanceStatuses(instance.GetID(), client)
	if err != nil {
		return nil, err
	}
	// Read-only: do not mutate entry.statuses / entry.categories.

	conf := p.getConfig()
	configured := conf.RHSStatusTabs[string(instance.GetID())]
	tabs := resolveTabs(configured, entry.statuses, entry.categories)

	selected, err := pickRHSTab(tabs, tabKind, tabKey, tabID)
	if err != nil {
		return nil, err
	}
	if sort == "" {
		sort = rhsDefaultSort
	}

	// RE3: cached keys only. Do not substitute a hardcoded four-key map.
	jql, err := buildTabJQL(selected, sort, validCategoryKeysFrom(entry.categories))
	if err != nil {
		return nil, err
	}

	search, err := client.SearchJQL(CloudSearchParams{
		JQL:           jql,
		Fields:        rhsSearchFields,
		MaxResults:    rhsIssuesPageSize,
		NextPageToken: nextPageToken,
	})
	if err != nil {
		return nil, err
	}

	return &rhsIssuesResult{
		Issues:        normalizeRHSIssues(instance, search.Issues),
		Tabs:          tabs,
		NextPageToken: search.NextPageToken,
		IsLast:        search.IsLast,
	}, nil
}

type rhsStatusesResult struct {
	Statuses   []*JiraStatus         `json:"statuses"`
	Categories []*JiraStatusCategory `json:"categories"`
}

func (p *Plugin) getRHSStatuses(instanceID, adminUserID types.ID) (*rhsStatusesResult, error) {
	if instanceID == "" {
		return nil, errors.Wrap(ErrInvalidRHSTab, "instance_id is required")
	}
	instance, err := p.instanceStore.LoadInstance(instanceID)
	if err != nil {
		if errors.Is(err, kvstore.ErrNotFound) {
			return nil, errors.Wrap(ErrInvalidRHSTab, "unknown instance_id")
		}
		return nil, err
	}

	client, err := p.resolveRHSAdminClient(instance, adminUserID)
	if err != nil {
		return nil, err
	}

	entry, err := p.getInstanceStatuses(instance.GetID(), client)
	if err != nil {
		return nil, err
	}
	return &rhsStatusesResult{
		Statuses:   entry.statuses,
		Categories: entry.categories,
	}, nil
}
```

Required imports:

```go
import (
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)
```

---

## Exact handlers and routes — Task 4.4

### Route constants — `server/http.go` after line 70

```go
	routeOAuth2Complete                         = "/oauth2/complete.html"
	routeAPIRHSIssues                           = "/rhs/issues"
	routeAPIRHSStatuses                         = "/rhs/statuses"
)
```

Do not put them in the middle of the block. Do not prefix with `/api/v2`
(the subrouter already has that).

### Registrations — `server/http.go` after line 168

```go
	apiRouter.HandleFunc(routeAPISubscriptionTemplates, p.checkAuth(p.handleResponse(p.httpGetSubscriptionTemplates))).Methods(http.MethodGet)

	// Cloud-only personal tickets RHS
	apiRouter.HandleFunc(routeAPIRHSIssues, p.checkAuth(p.handleResponse(p.httpRHSGetIssues))).Methods(http.MethodGet)
	apiRouter.HandleFunc(routeAPIRHSStatuses, p.checkAuth(p.handleResponse(p.httpRHSListStatuses))).Methods(http.MethodGet)
}
```

GET only. Both wrapped in `checkAuth` + `handleResponse`. No
`handleResponseWithCallbackInstance` (these are not instance-path webhook
routes).

Do not edit `respondErr`, `checkAuth`, `handleResponse`,
`routeAPIGetSearchIssues`, or the `get-search-issues` registration.

### Handlers — append to `server/rhs_http.go`

```go
func (p *Plugin) httpRHSGetIssues(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get(HeaderMattermostUserID)
	instanceID := types.ID(r.FormValue(ParamInstanceID))
	result, err := p.getRHSIssues(
		instanceID,
		types.ID(userID),
		r.FormValue(queryRHSTabKind),
		r.FormValue(queryRHSTabKey),
		r.FormValue(queryRHSTabID),
		r.FormValue(queryRHSSort),
		r.FormValue(queryRHSNextPageToken),
	)
	if err != nil {
		return respondRHSErr(w, err)
	}
	return respondJSON(w, result)
}

func (p *Plugin) httpRHSListStatuses(w http.ResponseWriter, r *http.Request) (int, error) {
	userID := r.Header.Get(HeaderMattermostUserID)
	if !p.checkSystemAdmin(userID) {
		return respondJSONErr(w, http.StatusForbidden, rhsErrNotAuthorized, "not authorized")
	}
	instanceID := types.ID(r.FormValue(ParamInstanceID))
	result, err := p.getRHSStatuses(instanceID, types.ID(userID))
	if err != nil {
		return respondRHSErr(w, err)
	}
	return respondJSON(w, result)
}
```

Every failure path of these two handlers goes through `respondJSONErr` /
`respondRHSErr`. Never `respondErr`. Never a bare 500 without a JSON body.

`ParamInstanceID` is `"instance_id"` (`constants.go:9`).

---

## Exact tests (`server/rhs_http_test.go`) — Task 4.5

Create the file. `package main`. Header as above. **Every test name
starts with `TestRHSHTTP`** so `go test ./... -run TestRHSHTTP` runs them.

Do not `t.Parallel()`.

### Helpers

```go
type testRHSCloudClient struct {
	testClient
	mu           sync.Mutex
	statuses     []*JiraStatus
	categories   []*JiraStatusCategory
	search       *CloudSearchResult
	searchErr    error
	searchCalls  int
	lastSearch   CloudSearchParams
	statusCalls  int
	catCalls     int
}

func (c *testRHSCloudClient) ListStatuses() ([]*JiraStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statusCalls++
	return c.statuses, nil
}

func (c *testRHSCloudClient) ListStatusCategories() ([]*JiraStatusCategory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.catCalls++
	return c.categories, nil
}

func (c *testRHSCloudClient) SearchJQL(params CloudSearchParams) (*CloudSearchResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.searchCalls++
	c.lastSearch = params
	if c.searchErr != nil {
		return nil, c.searchErr
	}
	if c.search == nil {
		return &CloudSearchResult{Issues: nil, IsLast: true}, nil
	}
	return c.search, nil
}

type rhsTestCloudInstance struct {
	testInstance
	client      Client
	jiraBaseURL string
	apiURL      string
}

func (i rhsTestCloudInstance) GetURL() string {
	if i.apiURL != "" {
		return i.apiURL
	}
	return i.testInstance.GetURL()
}

func (i rhsTestCloudInstance) GetJiraBaseURL() string {
	if i.jiraBaseURL != "" {
		return i.jiraBaseURL
	}
	return i.testInstance.GetJiraBaseURL()
}

func (i rhsTestCloudInstance) GetClient(*Connection) (Client, error) {
	return i.client, nil
}

type rhsErrNotFoundUserStore struct {
	mockUserStore
}

func (rhsErrNotFoundUserStore) LoadConnection(types.ID, types.ID) (*Connection, error) {
	return nil, kvstore.ErrNotFound
}

func setupRHSHTTPPlugin(t *testing.T, api *plugintest.API) *Plugin {
	t.Helper()
	api.On("LogWarn", mockAnythingOfTypeBatch("string", 11)...).Maybe().Return()
	p := setupTestPlugin(api)
	p.updateConfig(func(conf *config) {
		conf.maxAttachmentSize = defaultMaxAttachmentSize
	})
	return p
}

func storeRHSInstance(t *testing.T, p *Plugin, instance Instance) {
	t.Helper()
	store, ok := p.instanceStore.(*mockInstanceStoreKV)
	require.True(t, ok)
	store.kv.Store(instance.GetID(), instance)
}

func decodeRHSError(t *testing.T, w *httptest.ResponseRecorder) rhsJSONError {
	t.Helper()
	require.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))
	var body rhsJSONError
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func fixtureCategoriesOmitDone() []*JiraStatusCategory {
	return []*JiraStatusCategory{
		{ID: 1, Key: statusCategoryKeyUndefined, Name: "No Category"},
		{ID: 2, Key: statusCategoryKeyNew, Name: "To Do"},
		{ID: 4, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
		// done deliberately omitted
	}
}

func fixtureCanonicalCategories(t *testing.T) []*JiraStatusCategory {
	t.Helper()
	var cats []*JiraStatusCategory
	require.NoError(t, json.Unmarshal(loadRHSTestdata(t, "rhs-statuscategory.json"), &cats))
	return cats
}

func fixtureStatuses(t *testing.T) []*JiraStatus {
	t.Helper()
	var statuses []*JiraStatus
	require.NoError(t, json.Unmarshal(loadRHSTestdata(t, "rhs-status.json"), &statuses))
	return statuses
}
```

`loadRHSTestdata` already exists in `client_cloud_rhs_test.go` (same
package). `mockAnythingOfTypeBatch` is in `server/mock.go`.

Happy-path instance helper:

```go
func installRHSUserCloud(t *testing.T, p *Plugin, client *testRHSCloudClient) rhsTestCloudInstance {
	t.Helper()
	inst := rhsTestCloudInstance{
		testInstance: testInstance{
			InstanceCommon: InstanceCommon{
				InstanceID: types.ID("https://cloud.example.atlassian.net"),
				Type:       CloudInstanceType,
				Plugin:     p,
			},
		},
		client:      client,
		jiraBaseURL: "https://cloud.example.atlassian.net",
		apiURL:      "https://api.atlassian.com/ex/jira/CLOUDID",
	}
	storeRHSInstance(t, p, inst)
	p.userStore = mockUserStore{}
	return inst
}
```

`GetClient` returns `client` which embeds `testClient` (satisfies
`Client`) and implements `rhsCloudClient`. `GetURL()` is the API host;
`GetJiraBaseURL()` is the site. That split is how the BrowseURL test
proves P4-D7.

### `TestRHSHTTPGetIssuesHappyPathDTO`

`SearchJQL` returns one `jira.Issue` with Key `TES-41`, Summary
`"Fix login redirect"`, status name `"In Progress"` / category
`indeterminate`, priority `"High"`, type `"Bug"`, project `"TES"`,
assignee display `"Ada"`, reporter `"Bea"`, labels `["rhs"]`. Categories
are the canonical four (so Assigned JQL builds).

GET `/api/v2/rhs/issues?instance_id=<id>` as `connected_user` (or any id;
`mockUserStore` accepts all). No `tab_kind` (defaults Assigned). No
`sort` (defaults `updated`).

Assert:

- 200, `Content-Type: application/json`
- `issues[0].key == "TES-41"`
- `issues[0].summary == "Fix login redirect"`
- `issues[0].browseUrl == "https://cloud.example.atlassian.net/browse/TES-41"`
  (**not** the `api.atlassian.com` `GetURL()`)
- `issues[0].status.name == "In Progress"`
- `issues[0].status.categoryKey == "indeterminate"`
- `issues[0].priority == "High"`
- `issues[0].issueType == "Bug"`
- `issues[0].project == "TES"`
- `issues[0].assignee == "Ada"`
- `issues[0].reporter == "Bea"`
- `issues[0].labels` contains `"rhs"`
- `tabs[0].kind == "assigned"`
- `len(tabs) >= 1`
- `lastSearch.MaxResults == 20`
- `lastSearch.Fields` equals `rhsSearchFields` (exact, same order)
- `lastSearch.JQL == "assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC"`
- `lastSearch.NextPageToken == ""`
- `searchCalls == 1`

### `TestRHSHTTPGetIssuesCursorAndIsLast`

Same harness. Query `next_page_token=opaque-token-page-2`. Fake returns
`IsLast: true`, `NextPageToken: ""`, one issue.

Assert query token forwarded on `lastSearch.NextPageToken`, response
`isLast == true`, `nextPageToken == ""`.

### `TestRHSHTTPGetIssuesNotConnectedJSON`

Install the cloud test instance. Set
`p.userStore = rhsErrNotFoundUserStore{}`. GET issues.

Assert:

- status **401**
- `Content-Type: application/json`
- body `error == "not_connected"`
- body `message` is non-empty
- **not** `text/plain`
- `searchCalls == 0` if you still have a fake (or no SearchJQL at all)

### `TestRHSHTTPGetIssuesRateLimitedJSON`

Fake `searchErr = ErrRateLimited`. Canonical categories so JQL builds.
Assert **429**, JSON `error == "rate_limited"`, message mentions rate
limiting.

### `TestRHSHTTPGetIssuesNotCloudJSON`

Do **not** replace the instance. Use `setupTestPlugin`’s `testInstance1`
(`https://jiraurl1.com`) and `p.userStore = mockUserStore{}`. GET
`instance_id=https://jiraurl1.com`. `GetClient` returns `testClient{}` →
assert fails.

Assert **400**, JSON `error == "not_cloud"`.

### `TestRHSHTTPGetIssuesUsesCachedCategoryKeys` — **RE3 / Gate 4**

Fake categories = `fixtureCategoriesOmitDone()` (no `done`). Statuses
from the status fixture. GET Assigned (default tab).

Assert:

- JSON `error == "invalid_request"`
- status 400
- `searchCalls == 0` — Assigned must not search when `done` is absent
  from the **cached** list

If this test is skipped or rewritten to pre-seed a four-key map, Phase 4
is not done.

### `TestRHSHTTPGetIssuesBrowseURLUsesJiraBaseURL`

Can be a subtest of the happy path. The happy-path instance already sets
`apiURL != jiraBaseURL`. The assertion on `browseUrl` **is** this test.
Keep a dedicated name so Gate 4 can grep it.

### `TestRHSHTTPGetIssuesInvalidSortJSON`

Query `sort=priority`. Canonical categories. Assert 400,
`invalid_request`, `searchCalls == 0`.

### `TestRHSHTTPGetIssuesMissingInstanceIDJSON`

GET `/api/v2/rhs/issues` with no `instance_id`. Assert 400,
`invalid_request`.

### `TestRHSHTTPListStatusesNotAuthorizedJSON`

`api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(false)`.
GET `/api/v2/rhs/statuses?instance_id=…` with any user.

Assert **403**, JSON `error == "not_authorized"`,
`Content-Type: application/json`.

### `TestRHSHTTPListStatusesConnectJWTNoPersonalConnection`

This is the Gate 4 admin-path proof.

1. `httptest.NewServer` that serves `testdata/rhs-status.json` on
   `GET /rest/api/3/status` and `testdata/rhs-statuscategory.json` on
   `GET /rest/api/3/statuscategory`. `t.Cleanup(ts.Close)`.
2. `p := setupRHSHTTPPlugin(t, api)` (sets `maxAttachmentSize` — required;
   see research §11).
3. `p.userStore = rhsErrNotFoundUserStore{}` — **no personal connection**.
4. `api.On("HasPermissionTo", mock.AnythingOfType("string"), mock.Anything).Return(true)`.
5. Build a real `*cloudInstance`:

```go
ci := newCloudInstance(p, types.ID("https://connect.example.atlassian.net"), true, "", &AtlassianSecurityContext{
	Key:          "test-key",
	ClientKey:    "test-client-key",
	SharedSecret: "test-shared-secret",
	BaseURL:      ts.URL,
})
storeRHSInstance(t, p, ci)
```

6. GET `/api/v2/rhs/statuses?instance_id=https://connect.example.atlassian.net`
   with header user `connected_user` (or any string; connection store
   returns not-found).

Assert:

- **200**, `Content-Type: application/json`
- `len(statuses) == 2` (fixture)
- `len(categories) == 4`
- category keys include `done`
- first status id `"3"`, category key `indeterminate`

If the implementer used `getClient` for the admin path, this test **fails**
(`not_connected`). That is the point.

### `TestRHSHTTPListStatusesOAuth2NotConnectedJSON`

```go
oi := &cloudOAuthInstance{
	InstanceCommon: newInstanceCommon(p, CloudOAuthInstanceType, types.ID("https://oauth.example.atlassian.net")),
	JiraBaseURL:    "https://oauth.example.atlassian.net",
}
```

`p.userStore = rhsErrNotFoundUserStore{}`. `HasPermissionTo` true. GET
statuses.

Assert **401**, JSON `not_connected`. Proves OAuth2 does not invent a bot
client.

### `TestRHSHTTPListStatusesOAuth2IgnoresAdminAPIToken`

Same as the previous test, plus:

```go
p.updateConfig(func(conf *config) {
	conf.AdminEmail = "admin@example.com"
	conf.AdminAPIToken = "not-a-real-token"
	conf.maxAttachmentSize = defaultMaxAttachmentSize
})
```

Still **401** `not_connected`. Proves the global Basic-auth credential is
not a substitute.

### `TestRHSHTTPMissingUserStillPlainText`

GET `/api/v2/rhs/issues` **without** `Mattermost-User-Id`.

Assert:

- status 401
- body is the plain-text `"Not authorized"` (or contains it)
- `Content-Type` is **not** required to be JSON — this is `checkAuth` +
  `http.Error`. Do not “fix” this.

Same assertion on `/rhs/statuses`.

### Test imports

```go
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)
```

Expected `-run TestRHSHTTP` set: **14** tests (or 13 if BrowseURL is a
subtest of HappyPath — prefer a dedicated name).

---

## File-by-file change list (current line numbers at `285fe8d`)

| File | Action | Current lines | What |
|------|--------|---------------|------|
| `server/rhs_types.go` | **Modify (append)** | **1–97** unchanged. Append after `ErrInvalidRHSTab` (**97**) | `ErrRHSNotCloud`, `RHSIssueStatus`, `RHSIssue` |
| `server/rhs.go` | **Create** | n/a | `rhsCloudClient`, resolvers, normalize, `getRHSIssues`, `getRHSStatuses` |
| `server/rhs_http.go` | **Create** | n/a | JSON errors, `checkSystemAdmin`, both handlers |
| `server/http.go` | **Modify** | Insert constants after **70**. Insert registrations after **168**. Do not edit **217–220** (`respondErr`), **317–326** (`checkAuth`), **328–334** (`handleResponse`), **43** / **117** (`get-search-issues`) | Two routes |
| `server/rhs_http_test.go` | **Create** | n/a | `TestRHSHTTP*` including RE3 + Connect JWT no-connection |

**Do not modify:**

| File | Current lines (do not touch) | Why |
|------|------------------------------|-----|
| `server/issue.go` | `getClient` **1568–1582**; `httpGetSearchIssues` **903–916**; `GetSearchIssues` **918–973**; all 13 `p.getClient` sites | Gate 4 blast radius |
| `server/client.go` | `Client` **35–41**; `SearchIssues` **304–314**; `endpointURL` **497–509** | No widening; no `Issue.Search` |
| `server/autocomplete_search.go` | `getClient` **23**, **56** | Existing sites |
| `server/subscribe.go` | `getClient` **1185+**; `HasPermissionTo` **1076** (read-only precedent) | Existing sites |
| `server/client_cloud.go` | `newCloudClient` **25–31** | Call only |
| `server/instance_cloud.go` | `getClientForBot` **238–253**; `GetJiraBaseURL` **188–189** | Call only |
| `server/instance_cloud_oauth.go` | `GetURL` **224–226**; `GetJiraBaseURL` **228–229**; `GetClient` **126–132** | Call only; no AdminAPIToken |
| `server/rhs_jql.go` | `buildTabJQL` **52–84**; `validCategoryKeysFrom` **8–17**; `resolveTabs` **86–112** | Call only |
| `server/rhs_cache.go` | `getInstanceStatuses` **52–86**; `rhsStatusLister` **22–25** | Call only; do not mutate entries |
| `server/plugin.go` | cache fields **176–177**; invalidate **315** | Phase 3; no Phase 4 edit |
| `server/user.go` | settings-info | Phase 1 |
| `plugin.json` | | Phase 1 |
| `webapp/**` | | Milestone 2 |
| `respondErr` | `http.go:217-220` | Plain text forever |

---

## Commands

From the worktree root
`~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`:

```bash
# If compile fails on missing gitignored server/manifest.go:
make apply

# Phase 4 handler tests
cd server && go test ./... -run TestRHSHTTP -v

# Blast radius (must be empty / must not show RHS edits in these files)
git diff master -- server/issue.go
git diff master -- server/client.go
git diff master -- server/autocomplete_search.go
git diff master -- server/subscribe.go
git diff master -- server/client_cloud.go
git diff master -- server/instance_cloud.go
git diff master -- server/instance_cloud_oauth.go

# New JSON helper must not have replaced respondErr
git diff master -- server/http.go
# Reviewer reads that respondErr / checkAuth / get-search-issues registration
# are unchanged, and only two constants + two HandleFunc lines were added.

# Existing getClient sites still only those 21
git grep -n "p\.getClient(" server -- ':!*_test.go'

# No new Issue.Search call sites
git grep -n "Issue.Search" server/
```

`git diff master -- server/issue.go` and `server/client.go` must be
**empty**. Phase 1–3 did not touch them; Phase 4 must not either.

---

## Gate 4 checklist (reviewer)

From `IMPL_ORCHESTRATION_PLAN.md` Step 4. Execute these; they are not
optional.

### 1. Blast radius — existing HTTP / client behavior untouched

```bash
git diff --stat master -- server/
git diff master -- server/issue.go
git diff master -- server/client.go
```

- [ ] `server/issue.go` diff is **empty** (`getClient`,
      `httpGetSearchIssues`, `SearchIssues` call at `:955` untouched).
- [ ] `server/client.go` diff is **empty** (`SearchIssues` `:304-314`
      untouched; `Client` interface not widened).
- [ ] `git grep -n "p.getClient(" server -- ':!*_test.go'` still lists
      only the **21** pre-existing sites plus the **one** new call inside
      `resolveRHSUserClient` in `server/rhs.go`. No other new production
      site.
- [ ] `git grep -n "Issue.Search" server/` is still only
      `server/client.go`.
- [ ] `respondErr` (`http.go:217-220`) is unchanged — still `http.Error`.
- [ ] `checkAuth` still writes plain-text `"Not authorized"` on a missing
      header (`TestRHSHTTPMissingUserStillPlainText`).
- [ ] `routeAPIGetSearchIssues` registration is unchanged.

### 2. New JSON errors only on new routes

- [ ] `respondJSONErr` is defined in `server/rhs_http.go` and called only
      from the two new handlers / `respondRHSErr`.
- [ ] `cd server && go test ./... -run TestRHSHTTP -v` passes.
- [ ] `TestRHSHTTPGetIssuesNotConnectedJSON` asserts **JSON**
      `not_connected` and `Content-Type: application/json` (not just 401).
- [ ] `TestRHSHTTPGetIssuesRateLimitedJSON` asserts JSON `rate_limited`.
- [ ] `TestRHSHTTPGetIssuesNotCloudJSON` asserts JSON `not_cloud`.
- [ ] `TestRHSHTTPListStatusesNotAuthorizedJSON` asserts JSON
      `not_authorized` and 403.

### 3. Connect JWT admin path with **no personal connection**

- [ ] `TestRHSHTTPListStatusesConnectJWTNoPersonalConnection` exists
      (exact name).
- [ ] It uses a real `*cloudInstance` + `getClientForBot` (httptest
      BaseURL), **not** `getClient`.
- [ ] `userStore.LoadConnection` returns `kvstore.ErrNotFound`.
- [ ] The request still returns **200** with statuses + categories.
- [ ] `TestRHSHTTPListStatusesOAuth2NotConnectedJSON` exists and returns
      `not_connected` (OAuth2 has no bot).
- [ ] `TestRHSHTTPListStatusesOAuth2IgnoresAdminAPIToken` exists — setting
      `AdminEmail`/`AdminAPIToken` does not bypass `not_connected`.

### 4. RE3 — cached category keys (Gate 3 carry-forward)

- [ ] `rg -n "buildTabJQL" server/rhs.go` shows the call
      `buildTabJQL(..., validCategoryKeysFrom(entry.categories))`.
- [ ] `rg -n "statusCategoryKeyDone|\"done\"" server/rhs.go server/rhs_http.go`
      shows **no** hardcoded four-key map passed into `buildTabJQL`.
- [ ] `TestRHSHTTPGetIssuesUsesCachedCategoryKeys` exists, omits `done`
      from the fake/cached categories, requests Assigned, asserts JSON
      `invalid_request` and `searchCalls == 0`.
- [ ] Cache-returned status/category pointers are not mutated (no
      assignments into `entry.statuses[i]` / `entry.categories[i]`).

### 5. Mechanical

- [ ] `BrowseURL` tests assert the site URL from `GetJiraBaseURL()`, not
      `GetURL()`.
- [ ] Page size is 20 (`lastSearch.MaxResults == 20`); field list is the
      locked 11 fields.
- [ ] `/rhs/issues` returns `{issues, tabs, nextPageToken, isLast}`.
- [ ] `/rhs/statuses` is `PermissionManageSystem`-gated.
- [ ] `webapp/` diff is empty.
- [ ] File header year is 2017 on all new `server/*.go` files.

**Commit checkpoint (orchestration):** local commit of this phase only.
Do not push.

---

## What this phase hands to Phase 5 / Milestone 2

Phase 5 (quality gates) will run `make test` / `make check-style` and the
#36 audit. The HTTP layer’s #36 duty is the cached-keys call +
`TestRHSHTTPGetIssuesUsesCachedCategoryKeys`. Phase 5 must not “fix”
Assigned by hardcoding `done` at the handler.

Milestone 2 consumes:

| Item | Contract |
|------|----------|
| Issues route | `GET /api/v2/rhs/issues?instance_id=&tab_kind=&tab_key=&tab_id=&sort=&next_page_token=` |
| Issues body | `{issues, tabs, nextPageToken, isLast}` — `tabs` already resolved (W-D4) |
| Statuses route | `GET /api/v2/rhs/statuses?instance_id=` (system admin) |
| Statuses body | `{statuses, categories}` |
| Error body | `{"error":"<code>","message":"<text>"}` with the six codes above |
| DTO | `RHSIssue` fields in this plan; `status.categoryKey` for pill color |
| `rhs_enabled` | already on `/api/v2/settingsinfo` (Phase 1) |
| D2 picker | empty `value` → render Assigned + In Progress without `onChange` (not this phase) |

Phase 5 writes `.planning/PHASE1_HANDOFF.md`. **Do not write that file
in Phase 4.**

---

## Official / in-repo URLs cited

- Parent Phase 4: `.planning/PLAN.md` § Phase 4
- Orchestration Gate 4: `IMPL_ORCHESTRATION_PLAN.md` Step 4
- Context HTTP / admin / browse URL: `planner/projects/jira-plugin-rhs/context.md`
- Spec server API: `spec.md` § "Server API (net-new)"
- Cloud search GET: https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/#api-rest-api-3-search-jql-get
- Status list: https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-workflow-statuses/#api-rest-api-3-status-get

---

## Implementation Summary

**Implementer:** IE4 (new team member). No commit. No push. Phase 5 not started.

### Files created/modified

| File | Action |
|------|--------|
| `server/rhs_types.go` | Modified — appended `ErrRHSNotCloud`, `RHSIssueStatus`, `RHSIssue` after `ErrInvalidRHSTab`. Lines 1–97 unchanged. |
| `server/rhs.go` | Created — `rhsCloudClient`, user/admin resolvers, DTO normalize, `getRHSIssues`, `getRHSStatuses`. |
| `server/rhs_http.go` | Created — `respondJSONErr`, `respondRHSErr`, `checkSystemAdmin`, both handlers. |
| `server/http.go` | Modified — two route constants after `routeOAuth2Complete`; two GET registrations after subscription templates. `respondErr` / `checkAuth` / `handleResponse` / `get-search-issues` untouched. |
| `server/rhs_http_test.go` | Created — 14 `TestRHSHTTP*` tests. |

### Test results

```
cd server && go test ./... -run TestRHSHTTP -v
```

**14/14 PASS:** HappyPathDTO, CursorAndIsLast, NotConnectedJSON, RateLimitedJSON, NotCloudJSON, UsesCachedCategoryKeys (RE3), BrowseURLUsesJiraBaseURL, InvalidSortJSON, MissingInstanceIDJSON, ListStatusesNotAuthorizedJSON, ListStatusesConnectJWTNoPersonalConnection, ListStatusesOAuth2NotConnectedJSON, ListStatusesOAuth2IgnoresAdminAPIToken, MissingUserStillPlainText.

### RE3 / cached category keys

`validCategoryKeysFrom(entry.categories)` is the **only** key source passed to `buildTabJQL` in handlers (`server/rhs.go:208`). No hardcoded four-key map in `rhs.go` or `rhs_http.go`. Cache-returned `*JiraStatus` / `*JiraStatusCategory` are not mutated. `TestRHSHTTPGetIssuesUsesCachedCategoryKeys` omits `done`, requests Assigned, asserts JSON `invalid_request` and `searchCalls == 0`.

### Connect JWT admin path — no personal connection

`TestRHSHTTPListStatusesConnectJWTNoPersonalConnection` uses a real `*cloudInstance` + httptest `BaseURL` + `getClientForBot` / `newCloudClient`. `userStore` is `rhsErrNotFoundUserStore` (`kvstore.ErrNotFound`). `setupRHSHTTPPlugin` sets `maxAttachmentSize = defaultMaxAttachmentSize`. Request returns **200** with 2 statuses + 4 categories. OAuth2 tests still return `not_connected` even with `AdminEmail`/`AdminAPIToken` set.

### Blast radius

- `git diff master -- server/issue.go` — **empty**
- `git diff master -- server/client.go` — **empty**
- `Issue.Search` still only `server/client.go`
- Production `p.getClient(` sites: the original **21** plus the one prescribed call in `resolveRHSUserClient` (`server/rhs.go`). No other new production site.
- `server/http.go` diff is **+6 lines** only (2 constants + 2 HandleFunc).
- `webapp/` diff empty.

### Deviations

**Production: none.** Helpers, resolvers, handlers, routes, and error mapping match the plan.

**Test-only (compile/harness):**

1. `rhsTestCloudInstance.Common()` — value type must implement `Instance` so `storeRHSInstance(t, p, inst)` compiles (`Common()` is a pointer method on `InstanceCommon` and is not promoted onto the value).
2. `setupRHSHTTPPlugin` also mocks 9-arg `LogDebug` (`logResponse` Debug on non-OK). Without it, plugintest panics on error paths. Plan only called out 11-arg `LogWarn`.
3. Added `plugintest/mock` import (needed for `HasPermissionTo`) plus small helpers `doRHSHTTPGet` / `tes41Issue`.

### Blockers

None. Ready for Gate 4 review / local commit checkpoint (orchestration). Phase 5 not started.

## QA5 follow-up (2026-08-20)

Disconnected **cloud-oauth** users often do not get `kvstore.ErrNotFound` from
`LoadConnection` (zero `Connection`, `OAuth2Token == nil`). `GetClient` then
returns `errOAuthTokenMissing` (`no JWT instance found, and connection's OAuth
token is missing`), which `respondRHSErr` mapped to 500 `internal_error`. New
RHS routes now map that sentinel to JSON `not_connected` (401). Existing
`getClient` call sites and `get-search-issues` are unchanged. Tests:
`TestRHSHTTPGetIssuesCloudOAuthMissingTokenJSON`,
`TestRHSHTTPListStatusesCloudOAuthMissingTokenJSON`.

