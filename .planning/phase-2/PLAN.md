# Phase 2 Plan: Cloud client extension

> Prescriptive implementation plan for **Phase 2 only** (tasks 2.1–2.3). An
> Implementation Engineer should be able to land this without making design
> decisions. Do not implement later phases from this file.
>
> **Do not write production/test Go until this plan is followed as written.**
> This document is the plan; it is not the code.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase 2 (source of truth for WHAT)
- **Spec:** `planner/projects/jira-plugin-rhs/ideas/001-show-tickets-rhs/spec.md`
  (§ Cloud rate limits; search via `/search/jql`)
- **Orchestration:** Gate 2 / Step 2 of `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `b2bd53b` (Phase 1 config foundation — already committed)
- **Package:** `package main` under `server/` (module
  `github.com/mattermost/mattermost-plugin-jira`, `go.mod` at repo root)
- **Generated:** 2026-08-19
- **Status:** ready for implementation

## Scope

**In scope:**

- Append types to `server/rhs_types.go` (`JiraStatus`, `JiraStatusCategory`,
  `CloudSearchParams`, `CloudSearchResult`, `ErrRateLimited`)
- Create `server/client_cloud_rhs.go` (`ListStatuses`, `ListStatusCategories`,
  `SearchJQL` + retry)
- Create `server/client_cloud_rhs_test.go` (all names `TestCloudRHS*`)
- Create JSON fixtures under `server/testdata/rhs-*.json`

**Out of scope (do not do these):**

- Handlers, routes, JSON error convention (`rhs_http.go`, `http.go`)
- Status cache (`rhs_cache.go`, `Plugin` cache fields, `OnConfigurationChange`)
- JQL builder / tab resolution (`rhs_jql.go`)
- Issue DTO / `RHSIssue` / browse-link construction (`GetURL` vs
  `GetJiraBaseURL` is Phase 4)
- Any file under `webapp/`
- Adding methods to the shared `Client` / `SearchService` / `RESTService`
  interfaces
- Changing `SearchIssues` (`server/client.go:304-314`) or any existing
  `Issue.Search` call site
- `RESTPost` / POST `/search/jql` (GET is sufficient; spike-verified)
- Retry on 503 or on `jira-quota-tenant-based` as if it were global
- Server/DC search path

---

## Research (read this before coding)

### 1. `jiraCloudClient` and method attachment — `server/client_cloud.go`

Struct is `server/client_cloud.go:16-18`. Constructor is `newCloudClient` at
`:25-31`, which returns the shared `Client` interface:

```16:31:server/client_cloud.go
type jiraCloudClient struct {
	JiraClient
}

// ...

func newCloudClient(jiraClient *jira.Client) Client {
	return &jiraCloudClient{
		JiraClient: JiraClient{
			Jira: jiraClient,
		},
	}
}
```

Existing Cloud methods use a **value receiver** (`func (client jiraCloudClient)
ListProjectStatuses(...)`). Match that. Do **not** add fields to
`jiraCloudClient` and do **not** change `newCloudClient`.

`JiraClient` (`server/client.go:91-95`) holds `Jira *jira.Client`. That is how
you reach `client.Jira.NewRequest` / `client.Jira.Do`.

The shared `Client` tree (`server/client.go:35-50`) composes `RESTService`,
`IssueService`, `ProjectService`, `SearchService`, `UserService`. RHS methods
are Cloud-only. Adding them to `SearchService` would force stubs onto
`jiraServerClient`. Phase 4 type-asserts to a local `rhsCloudClient` interface.
**Do not widen `Client`.**

### 2. Why `ListProjectStatuses` is the wrong precedent

Interface: `server/client.go:64`. Cloud impl: `server/client_cloud.go:143-150`.

```143:150:server/client_cloud.go
func (client jiraCloudClient) ListProjectStatuses(projectID string) ([]*IssueTypeWithStatuses, error) {
	var result []*IssueTypeWithStatuses
	if err := client.RESTGet(fmt.Sprintf("3/project/%s/statuses", projectID), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
```

That is **per-project** and **grouped by issue type**
(`IssueTypeWithStatuses` at `server/client_cloud.go:20-23`, embedding
`*jira.IssueType` plus `Statuses []*jira.Status`). Its only consumer is create
metadata. There is no instance-wide listing today.

`ListStatuses` calls `GET /rest/api/3/status` (flat instance-wide list). Copy
the *call style* (`RESTGet` + dest pointer) but not the path or return type.

### 3. `RESTGet`, `RESTGetRaw`, `endpointURL` — `server/client.go`

```99:119:server/client.go
func (client JiraClient) RESTGet(endpoint string, params map[string]string, dest interface{}) error {
	endpointURL, err := endpointURL(endpoint)
	if err != nil {
		return err
	}
	req, err := client.Jira.NewRequest("GET", endpointURL, nil)
	// ... query params ...
	resp, err := client.Jira.Do(req, dest)
	if err != nil {
		err = userFriendlyJiraError(resp, err)
	}
	return err
}
```

`RESTGetRaw` (`:121-152`) skips the `/rest/api/` prefix. Do not use it here.

`endpointURL` is **`server/client.go:497-509`** (parent PLAN.md is correct;
`context.md` / `assumptions.md` still cite stale `:556-574`):

```497:509:server/client.go
func endpointURL(endpoint string) (string, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if parsedURL.Scheme != "" {
		return endpoint, nil
	}
	if parsedURL.Host != "" {
		return "", fmt.Errorf("endpoint must not contain a host without a scheme, got %q", endpoint)
	}
	return path.Join("/rest/api", endpoint), nil
}
```

`endpointURL("3/status")` → `/rest/api/3/status`.
`endpointURL("3/statuscategory")` → `/rest/api/3/statuscategory`.
`endpointURL("3/search/jql")` → `/rest/api/3/search/jql`.

**No URL branching.** go-jira `NewRequest` resolves that relative path against
the client's `baseURL`:

| Auth mode | `jira.NewClient(..., baseURL)` | Resulting GET |
|-----------|--------------------------------|---------------|
| Connect JWT | `cloudInstance.GetURL()` = site (`server/instance_cloud.go:184-186`) | `{site}.atlassian.net/rest/api/3/...` |
| Cloud OAuth2 | `cloudOAuthInstance.GetURL()` = `https://api.atlassian.com/ex/jira/{cloudid}` (`server/instance_cloud_oauth.go:224-226`) | `api.atlassian.com/ex/jira/{cloudid}/rest/api/3/...` |

`GetJiraBaseURL()` (`instance_cloud_oauth.go:228-229`) is the **user-facing
site**. It is **not** this phase. DTO browse links are Phase 4.

`RESTGet` / `RESTGetRaw` never return `*http.Response`, so they cannot read
`Retry-After` or `RateLimit-Reason`. Status listing does not need those
headers — use `RESTGet`. Search must not.

There is **no generic `RESTPost`**. GET `/search/jql` is spike-verified 200
(`spikes/status-tab-jql` probe 9). Do not add POST.

### 4. `SearchIssues` / `Issue.Search` — must not be used

```304:314:server/client.go
func (client JiraClient) SearchIssues(jql string, options *jira.SearchOptions) ([]jira.Issue, error) {
	found, resp, err := client.Jira.Issue.Search(jql, options)
	// ...
}
```

go-jira v1.16.0 `Issue.Search` → `SearchWithContext` hardcodes
`rest/api/2/search`:

```1090:1120:/Users/marianunez/go/pkg/mod/github.com/andygrunwald/go-jira@v1.16.0/issue.go
func (s *IssueService) SearchWithContext(...) ([]Issue, *Response, error) {
	u := url.URL{
		Path: "rest/api/2/search",
	}
	// ...
	req, err := s.client.NewRequestWithContext(ctx, "GET", u.String(), nil)
```

Live Cloud probe (2026-08-08): **410 Gone** —
`The requested API has been removed. Please migrate to the /rest/api/3/search/jql API.`
([CHANGE-2046](https://developer.atlassian.com/changelog/#CHANGE-2046)).

The same class of trap exists on go-jira's status helpers — do **not** call
them either:

| go-jira method | Hardcoded path | Use instead |
|----------------|----------------|-------------|
| `Issue.Search` | `rest/api/2/search` | `NewRequest` + `Do` on `3/search/jql` |
| `Status.GetAllStatuses` | `rest/api/2/status` | `RESTGet("3/status", …)` |
| `StatusCategory.GetList` | `rest/api/2/statuscategory` | `RESTGet("3/statuscategory", …)` |

`git grep -n "Issue.Search" server/` after this phase must still show **only**
`server/client.go:306`.

### 5. go-jira `NewRequest` / `Do` / `Response` — the only new raw-response code

Module: `github.com/andygrunwald/go-jira v1.16.0`
(`go.mod:6`). Local source:
`/Users/marianunez/go/pkg/mod/github.com/andygrunwald/go-jira@v1.16.0/`.

**`NewRequest`** (`jira.go:208-209` → `NewRequestWithContext` `:163-205`):

- Relative `urlStr` is resolved against `Client.baseURL`.
- Leading `/` is trimmed (`:169`).
- `body == nil` → no JSON body (correct for GET).
- Sets `Content-Type: application/json` even on GET (harmless).

**`Do`** (`jira.go:278-301`) — **this is the surprise vs a naive reading of
the parent plan:**

```278:301:/Users/marianunez/go/pkg/mod/github.com/andygrunwald/go-jira@v1.16.0/jira.go
func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	httpResp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	err = CheckResponse(httpResp)
	if err != nil {
		// Even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		return newResponse(httpResp, nil), err
	}

	if v != nil {
		defer httpResp.Body.Close()
		err = json.NewDecoder(httpResp.Body).Decode(v)
	}

	resp := newResponse(httpResp, v)
	return resp, err
}
```

`CheckResponse` (`:303-314`) treats any status outside 200–299 as an error
**without reading or closing the body**.

Consequences for `Do(req, nil)` (what this phase must use):

| Outcome | Return | Body closed by `Do`? | Body decoded by `Do`? |
|---------|--------|----------------------|------------------------|
| Network error | `(nil, err)` | n/a | n/a |
| 429 / 4xx / 5xx | `(resp, err)` | **No** | **No** |
| 2xx, `v == nil` | `(resp, nil)` | **No** | **No** |
| 2xx, `v != nil` | `(resp, err)` | Yes (`defer Close`) | Yes |

So:

1. On 429 you **do** get `*jira.Response` (headers are readable).
2. You **must** inspect `resp.StatusCode` **before** treating `err != nil` as
   terminal — a 429 is an error from `Do` *and* the retry signal.
3. You **must** decode the 200 body yourself (`v` is nil).
4. You **must** close `resp.Body` on every path. `Do` will not.

**`jira.Response`** (`jira.go:322-330`) embeds `*http.Response`:

```go
type Response struct {
	*http.Response
	StartAt    int
	MaxResults int
	Total      int
}
```

`resp.Header` / `resp.StatusCode` / `resp.Body` are the embedded
`*http.Response` fields. `StartAt` / `Total` are populated only when `Do`
decodes a legacy `searchResult`; they stay 0 here and **must not** be used as
a substitute for Cloud's missing `total`.

**Do not pass `&result` into `Do`.** That would close-on-success only, hide
429 headers behind a decode that never runs, and split body ownership.
Always `Do(req, nil)`, then branch on status, then decode or retry, then
close.

**`jira.NewJiraError`** (`error.go:22-27`) **reads and closes** the body.
`userFriendlyJiraError` (`server/client.go:585-613`) calls it on non-`*jira.Error`
values when `resp != nil`. On the non-429 error path, call
`userFriendlyJiraError` first (it closes), then call `closeJiraResp` again
(idempotent; satisfies `bodyclose` in *this* function).

Never `defer resp.Body.Close()` around the whole retry loop — that would hold
bodies open across attempts. Close before `continue` and before every
`return`.

### 6. Current `server/rhs_types.go` — APPEND only

File is 40 lines (Phase 1). It already has `RHSTabKind` (including
`RHSTabKindAssigned` — a Phase 1 addition the parent T1.1 draft omitted),
`RHSTabEntry`, category-key constants, and `rhsDefaultTabs()`.

**Do not redefine or move those.** Append the Phase 2 types after
`rhsDefaultTabs`. Do **not** add `RHSIssue` (Phase 4).

### 7. `.golangci.yml` — `bodyclose` + `errcheck`

Enabled linters (`.golangci.yml:25-38`): `bodyclose`, `errcheck`, `gocritic`,
`gosec`, `govet`, `ineffassign`, `misspell`, `nakedret`, `revive`,
`staticcheck`, `unconvert`, `unused`, `whitespace`.

`bodyclose` is the real lint risk. This phase is the only new production code
that holds a raw `*http.Response`. Close on every path, including retry and
error. Prefer ` _ = resp.Body.Close()` (or `closeJiraResp`) so `errcheck`
cannot fire even if the exclusion below is later removed.

`errcheck` is enabled, but `.golangci.yml:44-46` excludes findings matching
`Error return value` — that is the entire errcheck message shape, so
errcheck is effectively a no-op today. Do not rely on that. Still check
`json.NewDecoder(...).Decode` errors.

`issues.exclude-rules` (`.golangci.yml:118-122`) turns **`bodyclose` off for
`_test.go`**. Production `client_cloud_rhs.go` is **not** excluded.

`revive` `exported` / `var-naming` / `package-comments` are excluded
(`.golangci.yml:41-43`). Unexported `searchJQL`, `rhsRetry`, `closeJiraResp`
are fine.

Formatters: `gofmt` + `goimports` with
`local-prefixes: github.com/mattermost/mattermost-plugin-jira`.

File header (mattermost-govet `-license.year=2017`), same as Phase 1:

```go
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
```

### 8. Existing test / testdata patterns

`server/testdata/` holds **55 webhook fixtures only** (`webhook-*.json`). No
search or status fixtures. Webhook tests load them with
`os.Open("testdata/...")` from the `server/` working directory
(`server/webhook_parser_misc_test.go:18`). New files: `rhs-*.json`.

There is **no** existing `httptest.NewServer` + `jira.NewClient` +
`newCloudClient` test for REST methods. Closest precedents:

- `server/utils/wrap_http_client_test.go:19` — `httptest.NewServer` +
  `http.Client` (simplest)
- `server/instance_cloud_oauth_migration_test.go:444+` — `httptest.NewServer`
  as a fake Jira, but it tests instance install, not `RESTGet`

`github.com/jarcoal/httpmock` is used in `command_test.go` / `utils_test.go`
for `/status` health checks. **Do not use it here.** Parent T2.3 requires
`httptest.NewServer` so retry and headers are real.

Package: `package main`. Assertions: `testify/assert` + `require`. Table
tests: `for name, tc := range map[string]struct{ ... }` + `t.Run`.

These tests do **not** need `setupTestPlugin` / `plugintest`. They construct
`jiraCloudClient` directly. If `go test` fails on a missing
`server/manifest.go`, run `make apply` from the worktree root (gitignored;
Phase 1 already needed this).

### 9. Official Cloud search — `GET /rest/api/3/search/jql`

Docs:
https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/#api-rest-api-3-search-jql-get

Query parameters this phase sends (only these):

| Param | Type | When |
|-------|------|------|
| `jql` | string | always |
| `fields` | array (send as one comma-joined value) | when `len(Fields) > 0` |
| `maxResults` | integer | when `MaxResults > 0` |
| `nextPageToken` | string | when non-empty |

Do **not** send `expand`, `properties`, `fieldsByKeys`, `failFast`,
`reconcileIssues`.

`fields` encoding: official type is array; the spike used
`fields=key,status` (comma-joined, one query key) and got 200
(`spikes/cloud-search-sunset/README.md`). go-jira's old `Search` does the
same (`issue.go:1109-1110`). Use `q.Set("fields", strings.Join(params.Fields, ","))`.

Response (`SearchAndReconcileResults`):

```json
{
  "issues": [ { "id": "...", "key": "...", "fields": { ... } } ],
  "nextPageToken": "<opaque>",
  "isLast": false
}
```

**No `total`.** Spike with `maxResults=2` returned `nextPageToken` (opaque
base64) + `isLast: false` and no `total` at any point
(`spikes/cloud-search-sunset/README.md`). That is why the UI is Load more,
not "N of M". Do not invent a `Total` field.

Default fields when `fields` is omitted: **`id` only**. Tests must pass
fields. Unbounded JQL (`ORDER BY` with no restriction) is **400** — not this
phase (JQL is built in Phase 3).

GET vs POST: POST exists for long JQL. Assignee-scoped tab JQL is short. GET
only.

Old `/rest/api/2/search` and `/rest/api/3/search` are gone (410). The docs
page still lists them as "Currently being removed"; live Cloud is already
410. Do not call them.

### 10. Official status + status-category shapes

**Statuses** —
https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-workflow-statuses/#api-rest-api-3-status-get

`GET /rest/api/3/status` → JSON **array**. Each element:

```json
{
  "description": "...",
  "iconUrl": "...",
  "id": "10000",
  "name": "In Progress",
  "self": "https://…/rest/api/3/status/10000",
  "statusCategory": {
    "colorName": "yellow",
    "id": 4,
    "key": "indeterminate",
    "name": "In Progress",
    "self": "https://…/rest/api/3/statuscategory/4"
  }
}
```

Status `id` is a **string**. Category `id` is an **integer**. Extra fields
(`self`, `description`, `iconUrl`, `colorName`) are ignored if our structs
omit them.

**Status categories** —
https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-workflow-status-categories/#api-rest-api-3-statuscategory-get

`GET /rest/api/3/statuscategory` → JSON **array**. Official examples use dummy
keys (`in-flight`, `completed`). **Those are not the real Cloud keys.** Live
list (spike probe 12 +
[developer community](https://community.developer.atlassian.com/t/bad-documentation-for-rest-api-3-statuscategory/78565)):

| id | key | English name |
|----|-----|--------------|
| 1 | `undefined` | No Category |
| 2 | `new` | To Do |
| 4 | `indeterminate` | In Progress |
| 3 | `done` | Done |

Fixtures **must** use these real keys. Phase 3 validates JQL operands against
them.

### 11. Official rate-limit policy

https://developer.atlassian.com/cloud/jira/platform/rate-limiting/
(page last updated 2026-08-19)

429 headers (enforcement-active table on that page):

| Header | Role |
|--------|------|
| `Retry-After` | seconds to wait (integer in every official example) |
| `RateLimit-Reason` | which limiter fired |
| `X-RateLimit-Limit` / `Remaining` / `Reset` | informational; do not branch on these |

`RateLimit-Reason` values:

- `jira-burst-based` — per-endpoint per-tenant burst (~100 GET/s default)
- `jira-quota-global-based` — Tier-1 hourly points pool (65k pts/hr, shared
  across all tenants of the app)
- `jira-quota-tenant-based` — Tier-2 per-tenant hourly pool
- `jira-per-issue-on-write` — write path; irrelevant to GET search

`/search/jql` costs `1 + issues returned` → **21 points** for a page of 20.
Field count does not add cost.

Official retry guidance (same page, "Implementing retry logic"):

- Base delay **2s**, double per 429, cap **~30s**
- Jitter **×0.7–1.3**
- Cap retries (example: **4 attempts**)
- Prefer `Retry-After` when present (as the delay)
- On `jira-quota-global-based`, **stop calling** — the hourly pool is spent;
  `Retry-After` can be ~1800s (official example `Retry-After: 1847`)

Official extras we **do not** implement (parent/spec win):

- Official prose also says pause on `jira-quota-tenant-based`. Spec + parent
  T2.2 only special-case **`jira-quota-global-based`**. Tenant and burst
  **are** retried.
- Official pseudocode does `delay += delay * random(0.7, 1.3)` (≈1.7–2.3×).
  Spec + parent say **multiply** by 0.7–1.3. Multiply.
- Official "only retry if `Retry-After` is present." Parent retries any
  non-global 429, using exponential when the header is missing.
- Official 503-with-`Retry-After` handling. Not this phase. Only 429.

### 12. Pitfall confirmation vs parent `.planning/PLAN.md`

| Claim in parent PLAN.md | Current worktree / docs | Drift? |
|-------------------------|-------------------------|--------|
| `jiraCloudClient` at `client_cloud.go:16-31` | struct `:16-18`, `newCloudClient` `:25-31` | Cosmetic |
| `RESTGet` `:99-119`, `RESTGetRaw` `:121-152` | matches | **No** |
| `endpointURL` `:497-509` | matches | **No** (`context.md` 556-574 is stale) |
| `SearchIssues` `:304-314` | matches | **No** |
| `ListProjectStatuses` `:143-150` | matches | **No** |
| `Do(req, nil)` returns `*jira.Response` embedding `*http.Response` | yes; **and** `Do` returns `err` on 429 **without** closing | **Yes — must handle.** See §5 |
| Types already include `JiraStatus` / `ErrRateLimited` | **not present**; Phase 1 left them out on purpose | **No** (append) |
| testdata has no search fixtures | 55 webhook files only | **No** |
| `bodyclose` + `errcheck` on raw response | `bodyclose` is the real one; errcheck excluded by text | Cosmetic |
| Official dummy category keys in REST examples | live keys are `new`/`indeterminate`/`done`/`undefined` | **Yes — use real keys in fixtures** |

---

## Decisions this phase locks (do not reopen)

**P2-D1 — Methods live on `jiraCloudClient` only.** Value receivers. No
`Client` / `SearchService` change. Tests type-assert
`newCloudClient(jc).(*jiraCloudClient)`.

**P2-D2 — `Do(req, nil)` then we own the body.** Never pass a dest into `Do`.
Inspect `StatusCode` before treating `err` as fatal. Decode 200 ourselves.

**P2-D3 — Close before continue/return, never loop-scoped defer.** Helper
`closeJiraResp`. Double-close is OK.

**P2-D4 — Four HTTP attempts, not four retries.** Attempt 1 + at most 3
retries = **4** upstream requests. After the 4th 429, return `ErrRateLimited`
without sleeping.

**P2-D5 — Injectable retry via unexported `searchJQL` + `rhsRetry`.** No
package-level `time.Sleep` override (not parallel-safe). Exported `SearchJQL`
calls `searchJQL(params, defaultRHSRetry())`. Tests that measure sleep call
`searchJQL` with a recording `sleep` and `jitter` of `1.0`. Do **not**
`t.Parallel()` on those tests anyway.

**P2-D6 — Prefer `Retry-After` as the delay** (replace exponential, do not
`max()`). Then multiply by jitter. The ~30s cap applies to the **exponential**
value only. Invalid / empty `Retry-After` → exponential. Parse integer
seconds only (`strconv.Atoi`); do not parse HTTP-dates.

**P2-D7 — Do not retry `RateLimit-Reason: jira-quota-global-based`.** Zero
sleeps, exactly one request, return `ErrRateLimited`. Burst and tenant quota
retry.

**P2-D8 — Append types; do not reuse `jira.Status` for the instance list.**
`jira.Status` is what `ListProjectStatuses` already returns (wrong grouping).
Declare slim `JiraStatus` / `JiraStatusCategory`. `CloudSearchResult.Issues`
**does** use `[]jira.Issue`.

**P2-D9 — `GetURL` vs `GetJiraBaseURL` is not this phase.**

---

## Exact type additions (`server/rhs_types.go`) — Tasks 2.1 / 2.2

Keep the Phase 1 contents (lines 1–40) unchanged. After `rhsDefaultTabs`,
append:

```go
import (
	"errors"

	jira "github.com/andygrunwald/go-jira"
)
```

Go requires imports at the top of the file. Add that import block **under
`package main`**, above the existing `RHSTabKind` declarations. The file
currently has **no** import block.

Then append these types **below** `rhsDefaultTabs`:

```go
// JiraStatus is one instance-wide workflow status from GET /rest/api/3/status.
// Extra JSON fields (self, description, iconUrl) are ignored.
type JiraStatus struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	StatusCategory JiraStatusCategory `json:"statusCategory"`
}

// JiraStatusCategory is a Cloud status category.
// ID is numeric (official schema int64). Key is the JQL operand
// (new / indeterminate / done / undefined).
type JiraStatusCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// CloudSearchParams is the input to jiraCloudClient.SearchJQL.
type CloudSearchParams struct {
	JQL           string
	Fields        []string
	MaxResults    int
	NextPageToken string
}

// CloudSearchResult is GET /rest/api/3/search/jql.
// Cloud returns no total — do not add one.
type CloudSearchResult struct {
	Issues        []jira.Issue `json:"issues"`
	NextPageToken string       `json:"nextPageToken"`
	IsLast        bool         `json:"isLast"`
}

// ErrRateLimited is returned when a 429 is not retried (global quota) or
// when retries are exhausted. Phase 4 maps this to JSON error rate_limited.
var ErrRateLimited = errors.New("jira rate limited")
```

Field types are locked: status `ID` is `string`; category `ID` is `int`. That
matches the official JSON and go-jira's `Status` / `StatusCategory`.

Sentinel style matches `kvstore.ErrNotFound` (`server/utils/kvstore/kvstore.go:21`).
Phase 4 will `errors.Is(err, ErrRateLimited)`.

---

## Exact methods (`server/client_cloud_rhs.go`) — Tasks 2.1 / 2.2

Create the file. `package main`. Header as in §7.

### Constants and retry plumbing (top of file)

```go
const (
	rateLimitReasonGlobalQuota = "jira-quota-global-based"
	rhsSearchMaxAttempts       = 4
)

type rhsRetry struct {
	base        time.Duration
	cap         time.Duration
	maxAttempts int
	sleep       func(time.Duration)
	jitter      func() float64 // factor in [0.7, 1.3]
}

func defaultRHSRetry() rhsRetry {
	return rhsRetry{
		base:        2 * time.Second,
		cap:         30 * time.Second,
		maxAttempts: rhsSearchMaxAttempts,
		sleep:       time.Sleep,
		jitter:      rhsRandomJitter,
	}
}

func rhsRandomJitter() float64 {
	return 0.7 + rand.Float64()*(1.3-0.7)
}

func closeJiraResp(resp *jira.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_ = resp.Body.Close()
}

// rhsBackoffDelay is the sleep after a retryable 429 on failedAttempt (1-indexed).
// Prefer parseable Retry-After (seconds) over exponential. Cap applies to
// exponential only. Then multiply by jitter.
func rhsBackoffDelay(retry rhsRetry, failedAttempt int, retryAfterHeader string) time.Duration {
	exp := retry.base * time.Duration(1<<uint(failedAttempt-1)) // 2s, 4s, 8s, …
	if exp > retry.cap {
		exp = retry.cap
	}
	delay := exp
	if secs, err := strconv.Atoi(strings.TrimSpace(retryAfterHeader)); err == nil && secs >= 0 {
		delay = time.Duration(secs) * time.Second
	}
	factor := 1.0
	if retry.jitter != nil {
		factor = retry.jitter()
	}
	return time.Duration(float64(delay) * factor)
}
```

`rhsRandomJitter` is not a security source. If `gosec` G404 flags
`math/rand`, keep `math/rand` and do not add a `//nolint` unless lint fails —
prefer a one-line conversion from `crypto/rand` only if `make check-style`
requires it.

### Task 2.1 — signatures and bodies

```go
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

Paths are **`3/status`** and **`3/statuscategory`** (no leading slash, no
`/rest/api/` prefix). `endpointURL` adds that. Do not call
`3/project/{id}/statuses`.

### Task 2.2 — `SearchJQL`

```go
func (client jiraCloudClient) SearchJQL(params CloudSearchParams) (*CloudSearchResult, error) {
	return client.searchJQL(params, defaultRHSRetry())
}

func (client jiraCloudClient) searchJQL(params CloudSearchParams, retry rhsRetry) (*CloudSearchResult, error) {
	if retry.maxAttempts < 1 {
		retry.maxAttempts = rhsSearchMaxAttempts
	}
	if retry.sleep == nil {
		retry.sleep = time.Sleep
	}

	endpoint, err := endpointURL("3/search/jql")
	if err != nil {
		return nil, err
	}

	for attempt := 1; attempt <= retry.maxAttempts; attempt++ {
		req, err := client.Jira.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		q := req.URL.Query()
		q.Set("jql", params.JQL)
		if len(params.Fields) > 0 {
			q.Set("fields", strings.Join(params.Fields, ","))
		}
		if params.MaxResults > 0 {
			q.Set("maxResults", strconv.Itoa(params.MaxResults))
		}
		if params.NextPageToken != "" {
			q.Set("nextPageToken", params.NextPageToken)
		}
		req.URL.RawQuery = q.Encode()

		resp, doErr := client.Jira.Do(req, nil)

		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			reason := resp.Header.Get("RateLimit-Reason")
			retryAfter := resp.Header.Get("Retry-After")
			closeJiraResp(resp)

			if reason == rateLimitReasonGlobalQuota {
				return nil, ErrRateLimited
			}
			if attempt == retry.maxAttempts {
				return nil, ErrRateLimited
			}
			retry.sleep(rhsBackoffDelay(retry, attempt, retryAfter))
			continue
		}

		if doErr != nil {
			if resp != nil {
				wrapped := userFriendlyJiraError(resp, doErr)
				closeJiraResp(resp)
				return nil, wrapped
			}
			return nil, doErr
		}

		var result CloudSearchResult
		decErr := json.NewDecoder(resp.Body).Decode(&result)
		closeJiraResp(resp)
		if decErr != nil {
			return nil, decErr
		}
		return &result, nil
	}

	return nil, ErrRateLimited
}
```

URL construction **must** go through `endpointURL` then `NewRequest` then
query-encode — the same sequence as `RESTGet` (`server/client.go:99-112`).
Do **not** hand-build `https://…` URLs.

Required imports for this file:

```go
import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
)
```

`goimports` will group them.

### Retry algorithm (locked)

After retryable 429 on attempt `n` (1-indexed), if `n < 4`:

1. Exponential = `min(2s * 2^(n-1), 30s)` → 2s, 4s, 8s, then cap 30s
   (the 4th attempt never sleeps).
2. If `Retry-After` parses as `>= 0` integer seconds, **replace** exponential
   with that duration.
3. Multiply by jitter in `[0.7, 1.3]`.
4. `sleep(delay)`, then next attempt.

If `RateLimit-Reason` is exactly `jira-quota-global-based`: return
`ErrRateLimited` immediately (even when `Retry-After` is present, even on
attempt 1).

Non-429 failures: no retry. `userFriendlyJiraError` when `resp != nil`.

---

## Exact tests (`server/client_cloud_rhs_test.go`) — Task 2.3

Create the file. `package main`. Header as in §7. **Every test name starts
with `TestCloudRHS`** so `go test ./... -run TestCloudRHS` runs them.

### Helpers (same file, unexported)

```go
func newTestCloudRHSClient(t *testing.T, handler http.HandlerFunc) *jiraCloudClient {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	jc, err := jira.NewClient(ts.Client(), ts.URL)
	require.NoError(t, err)
	client, ok := newCloudClient(jc).(*jiraCloudClient)
	require.True(t, ok)
	return client
}

func loadRHSTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return b
}

func testRetryNoSleep(sleeps *[]time.Duration) rhsRetry {
	return rhsRetry{
		base:        2 * time.Second,
		cap:         30 * time.Second,
		maxAttempts: rhsSearchMaxAttempts,
		sleep: func(d time.Duration) {
			*sleeps = append(*sleeps, d)
		},
		jitter: func() float64 { return 1.0 },
	}
}

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
```

Handler paths that go-jira will hit (leading slash):

- `/rest/api/3/search/jql`
- `/rest/api/3/status`
- `/rest/api/3/statuscategory`

Assert `r.Method == http.MethodGet`. Count with a `int` closed over by the
handler (these tests are sequential; do not `t.Parallel()`).

### `TestCloudRHSSearchJQLHappyPath`

Drive **exported** `SearchJQL` (no sleep). Handler returns
`testdata/rhs-search-jql-page1.json` (200).

Assert:

- `err == nil`
- `len(result.Issues) == 1`
- `result.Issues[0].Key == "TES-41"`
- `result.NextPageToken == "opaque-token-page-2"`
- `result.IsLast == false`
- incoming query: `jql` is the params JQL, `maxResults=20`,
  `fields` contains `summary` and `status` (comma-joined is fine),
  `nextPageToken` absent
- `r.URL.Path == "/rest/api/3/search/jql"`
- request count == 1

Params to send:

```go
CloudSearchParams{
	JQL:        "assignee = currentUser() AND statusCategory != done ORDER BY updated DESC, key ASC",
	Fields:     []string{"summary", "status"},
	MaxResults: 20,
}
```

### `TestCloudRHSSearchJQLCursorPassthrough`

One `SearchJQL` call with `NextPageToken: "opaque-token-page-2"`. Handler
returns `testdata/rhs-search-jql-last.json`.

Assert:

- query `nextPageToken` **exactly** `opaque-token-page-2`
- `result.IsLast == true`
- `result.NextPageToken == ""`
- request count == 1

### `TestCloudRHSSearchJQLRetryAfterHonored`

Use `searchJQL` + `testRetryNoSleep`. Handler:

1. First request: **429**, headers `Retry-After: 7` and
   `RateLimit-Reason: jira-burst-based`, empty body.
2. Second request: **200** + `rhs-search-jql-last.json`.

Assert:

- `err == nil`
- request count == **2**
- `len(sleeps) == 1` and `sleeps[0] == 7*time.Second`
  (jitter is 1.0; Retry-After **replaced** the 2s exponential — if the
  implementation uses exponential 2s instead, this fails)

### `TestCloudRHSSearchJQLGlobalQuotaNoRetry`

Use `searchJQL` + `testRetryNoSleep`. Handler **always** returns 429 with:

- `RateLimit-Reason: jira-quota-global-based`
- `Retry-After: 1847` (official example value — must still not retry)

Assert:

- `errors.Is(err, ErrRateLimited)`
- request count == **1** (Gate 2)
- `len(sleeps) == 0`

This is the assertion that proves the no-retry rule. A reviewer who only
reads the code has not executed Gate 2.

### `TestCloudRHSSearchJQLAttemptsCapAt4`

Use `searchJQL` + `testRetryNoSleep`. Handler **always** returns 429 with
`RateLimit-Reason: jira-burst-based` and **no** `Retry-After` (forces
exponential).

Assert:

- `errors.Is(err, ErrRateLimited)`
- request count == **4**
- `len(sleeps) == 3`
- `sleeps[0] == 2*time.Second`
- `sleeps[1] == 4*time.Second`
- `sleeps[2] == 8*time.Second`
  (jitter 1.0; 30s cap is not hit in 3 sleeps)

### `TestCloudRHSListStatuses`

`ListStatuses()` against a handler that serves `testdata/rhs-status.json` on
`/rest/api/3/status`.

Assert:

- path is `/rest/api/3/status`, method GET
- `len(got) == 2`
- `got[0].ID == "3"`, `got[0].Name == "In Progress"`,
  `got[0].StatusCategory.Key == "indeterminate"`,
  `got[0].StatusCategory.ID == 4`
- `got[1].ID == "10001"`, `got[1].StatusCategory.Key == "new"`

### `TestCloudRHSListStatusCategories`

`ListStatusCategories()` against `testdata/rhs-statuscategory.json` on
`/rest/api/3/statuscategory`.

Assert:

- four entries
- keys (order as in fixture): `undefined`, `new`, `indeterminate`, `done`
- ids: 1, 2, 4, 3
- names: `No Category`, `To Do`, `In Progress`, `Done`

### `TestCloudRHSEndpointURL`

No HTTP. Locks the "no URL branching" claim:

```go
func TestCloudRHSEndpointURL(t *testing.T) {
	status, err := endpointURL("3/status")
	require.NoError(t, err)
	assert.Equal(t, "/rest/api/3/status", status)

	cats, err := endpointURL("3/statuscategory")
	require.NoError(t, err)
	assert.Equal(t, "/rest/api/3/statuscategory", cats)

	search, err := endpointURL("3/search/jql")
	require.NoError(t, err)
	assert.Equal(t, "/rest/api/3/search/jql", search)
}
```

### Test imports

```go
import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

Expected `-run TestCloudRHS` set: 8 tests.

---

## Fixture files (`server/testdata/`)

Create four files. Minimal but valid Cloud shapes. Real category keys.

### `server/testdata/rhs-search-jql-page1.json`

```json
{
  "issues": [
    {
      "id": "10040",
      "self": "https://example.atlassian.net/rest/api/3/issue/10040",
      "key": "TES-41",
      "fields": {
        "summary": "Fix login redirect",
        "status": {
          "id": "3",
          "name": "In Progress",
          "statusCategory": {
            "id": 4,
            "key": "indeterminate",
            "name": "In Progress"
          }
        }
      }
    }
  ],
  "nextPageToken": "opaque-token-page-2",
  "isLast": false
}
```

### `server/testdata/rhs-search-jql-last.json`

```json
{
  "issues": [
    {
      "id": "10041",
      "key": "TES-42",
      "fields": {
        "summary": "Last page issue",
        "status": {
          "id": "1",
          "name": "Open",
          "statusCategory": {
            "id": 2,
            "key": "new",
            "name": "To Do"
          }
        }
      }
    }
  ],
  "isLast": true
}
```

No `nextPageToken` key (empty string after unmarshal). No `total`.

### `server/testdata/rhs-status.json`

```json
[
  {
    "description": "This issue is being actively worked on.",
    "iconUrl": "https://example.atlassian.net/images/icons/statuses/inprogress.png",
    "id": "3",
    "name": "In Progress",
    "self": "https://example.atlassian.net/rest/api/3/status/3",
    "statusCategory": {
      "self": "https://example.atlassian.net/rest/api/3/statuscategory/4",
      "id": 4,
      "key": "indeterminate",
      "colorName": "yellow",
      "name": "In Progress"
    }
  },
  {
    "description": "The issue is open.",
    "id": "10001",
    "name": "Backlog",
    "statusCategory": {
      "id": 2,
      "key": "new",
      "colorName": "blue-gray",
      "name": "To Do"
    }
  }
]
```

### `server/testdata/rhs-statuscategory.json`

```json
[
  {
    "self": "https://example.atlassian.net/rest/api/3/statuscategory/1",
    "id": 1,
    "key": "undefined",
    "colorName": "medium-gray",
    "name": "No Category"
  },
  {
    "self": "https://example.atlassian.net/rest/api/3/statuscategory/2",
    "id": 2,
    "key": "new",
    "colorName": "blue-gray",
    "name": "To Do"
  },
  {
    "self": "https://example.atlassian.net/rest/api/3/statuscategory/4",
    "id": 4,
    "key": "indeterminate",
    "colorName": "yellow",
    "name": "In Progress"
  },
  {
    "self": "https://example.atlassian.net/rest/api/3/statuscategory/3",
    "id": 3,
    "key": "done",
    "colorName": "green",
    "name": "Done"
  }
]
```

---

## File-by-file change list (current line numbers)

Line numbers are as of `b2bd53b` (Phase 1 committed). Phase 2 does not edit
`plugin.go` / `user.go` / `plugin.json` / `client.go` / `client_cloud.go`.

| File | Action | Current lines | What |
|------|--------|---------------|------|
| `server/rhs_types.go` | **Modify (append)** | **1–40** (do not change). Add import block after `package main` (currently none). Append types after `rhsDefaultTabs` (ends **line 40**) | `JiraStatus`, `JiraStatusCategory`, `CloudSearchParams`, `CloudSearchResult`, `ErrRateLimited` |
| `server/client_cloud_rhs.go` | **Create** | n/a | `ListStatuses`, `ListStatusCategories`, `SearchJQL` / `searchJQL`, retry helpers |
| `server/client_cloud_rhs_test.go` | **Create** | n/a | Eight `TestCloudRHS*` tests |
| `server/testdata/rhs-search-jql-page1.json` | **Create** | n/a | Happy-path search (`nextPageToken`, `isLast: false`) |
| `server/testdata/rhs-search-jql-last.json` | **Create** | n/a | Last page (`isLast: true`, no token) |
| `server/testdata/rhs-status.json` | **Create** | n/a | Two statuses |
| `server/testdata/rhs-statuscategory.json` | **Create** | n/a | Four real categories |

**Do not modify:**

| File | Current lines (do not touch) | Why |
|------|------------------------------|-----|
| `server/client.go` | `Client` `:35-50`; `RESTGet` `:99-119`; `SearchIssues` `:304-314`; `endpointURL` `:497-509` | No interface widening; no `Issue.Search` change |
| `server/client_cloud.go` | struct `:16-18`; `newCloudClient` `:25-31`; `ListProjectStatuses` `:143-150` | Methods go in the new file |
| `server/rhs_types_test.go` | existing `TestRHSConfig*` | Phase 1 only |
| `server/plugin.go` | `externalConfig` now `:52+` (Phase 1 added fields at `:56-60`); `Plugin` struct; `OnConfigurationChange` | Phase 3 edits cache regions |
| `plugin.json` | Phase 1 settings already present | |
| `server/user.go` | `httpGetSettingsInfo` | |
| `server/http.go` | routes | Phase 4 |
| `webapp/**` | | Milestone 2 |
| `.golangci.yml` | `bodyclose` `:26`; test exclude `:118-122` | |

---

## Commands

From the worktree root
`~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`:

```bash
# If compile fails on missing gitignored server/manifest.go:
make apply

# Phase 2 unit tests (all TestCloudRHS* names)
cd server && go test ./... -run TestCloudRHS -v

# No new Issue.Search call sites (must remain only client.go)
git grep -n "Issue.Search" server/

# bodyclose on the new raw-response file (from worktree root)
./bin/golangci-lint run ./server/client_cloud_rhs.go
# or, if the binary is not present:
golangci-lint run ./server/client_cloud_rhs.go
```

Parent DoD also asks `golangci-lint run ./...` — run it from the worktree
root if the toolchain is available. Fix findings in **new** code only.

Do not run `make check-style` as a Phase 2 requirement if it pulls in
untouched `webapp/` lint; the Gate 2 lint bar is `bodyclose` on the new
raw-response file plus `go test`.

---

## Gate 2 checklist (reviewer)

From `IMPL_ORCHESTRATION_PLAN.md` Step 2. A reviewer **executes** these; they
are not optional.

- [ ] `cd server && go test ./... -run TestCloudRHS -v` passes.
- [ ] `TestCloudRHSSearchJQLGlobalQuotaNoRetry` exists and asserts the fake
      server saw **exactly one** request when
      `RateLimit-Reason: jira-quota-global-based`.
- [ ] `TestCloudRHSSearchJQLRetryAfterHonored` exists and asserts the recorded
      sleep equals the `Retry-After` value (not the 2s exponential).
- [ ] `TestCloudRHSSearchJQLAttemptsCapAt4` exists and asserts **4** upstream
      requests (and 3 sleeps).
- [ ] Happy-path test decodes `nextPageToken` and `isLast`.
- [ ] Cursor test asserts `nextPageToken` is forwarded on the query string.
- [ ] Status and status-category tests decode the fixtures (real keys
      `new` / `indeterminate` / `done` / `undefined`).
- [ ] `git grep -n "Issue.Search" server/` shows **no new call sites**
      (only `server/client.go`).
- [ ] `client_cloud_rhs.go` never calls `client.Jira.Issue.Search`,
      `Status.GetAllStatuses`, or `StatusCategory.GetList`.
- [ ] `client_cloud_rhs.go` uses `Do(req, nil)` (or equivalent with `v == nil`)
      and closes `resp.Body` on success, 429-retry, 429-return, and non-429
      error paths. `golangci-lint` reports **no `bodyclose`** finding on that
      file.
- [ ] `Client` / `SearchService` interfaces are unchanged
      (`git diff b2bd53b -- server/client.go` is empty).
- [ ] `webapp/` diff is empty. `plugin.go` / `http.go` diffs are empty.

**Commit checkpoint (orchestration):** local commit of this phase only. Do
not push.

---

## What this phase hands to later phases

- `jiraCloudClient.ListStatuses()` / `ListStatusCategories()` — Phase 3 cache
  (`getInstanceStatuses`) is the first caller.
- `jiraCloudClient.SearchJQL(CloudSearchParams) (*CloudSearchResult, error)` —
  Phase 4 handlers pass built JQL, the fixed field list, `MaxResults: 20`,
  and the cursor.
- `ErrRateLimited` — Phase 4 maps it to JSON `{error: "rate_limited"}`.
- `JiraStatus` / `JiraStatusCategory` — Phase 3 membership checks and Phase 4
  DTO `statusCategory.key`.
- `CloudSearchResult` has `NextPageToken` + `IsLast` and **no** `total`.
- Methods are **not** on `Client`. Phase 4 declares
  `type rhsCloudClient interface { ListStatuses() …; ListStatusCategories() …; SearchJQL(…) … }`
  and type-asserts after `getClient` / `newCloudClient(getClientForBot())`.
- Browse URLs still unbuilt. Phase 4 must use `instance.GetJiraBaseURL()`,
  never `GetURL()`.

---

## Official URLs cited

- Search GET: https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/#api-rest-api-3-search-jql-get
- Statuses: https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-workflow-statuses/#api-rest-api-3-status-get
- Status categories: https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-workflow-status-categories/#api-rest-api-3-statuscategory-get
- Rate limiting: https://developer.atlassian.com/cloud/jira/platform/rate-limiting/
- Search sunset: https://developer.atlassian.com/changelog/#CHANGE-2046
- Migration KB: https://confluence.atlassian.com/jirakb/run-jql-search-query-using-jira-cloud-rest-api-1289424308.html
- Real category list (docs examples are dummy): https://community.developer.atlassian.com/t/bad-documentation-for-rest-api-3-statuscategory/78565

---

## Implementation Summary

Implemented by IE2 (2026-08-19). Phase 1 left at `b2bd53b`. No commit. No push. Phase 3 not started.

### Files created/modified

| File | Action |
|------|--------|
| `server/rhs_types.go` | Modified — import block + append `JiraStatus`, `JiraStatusCategory`, `CloudSearchParams`, `CloudSearchResult`, `ErrRateLimited`. Phase 1 types unchanged. |
| `server/client_cloud_rhs.go` | Created — `ListStatuses`, `ListStatusCategories`, `SearchJQL` / `searchJQL`, retry helpers. Value receivers on `jiraCloudClient`. |
| `server/client_cloud_rhs_test.go` | Created — eight `TestCloudRHS*` tests |
| `server/testdata/rhs-search-jql-page1.json` | Created |
| `server/testdata/rhs-search-jql-last.json` | Created |
| `server/testdata/rhs-status.json` | Created |
| `server/testdata/rhs-statuscategory.json` | Created |

Unchanged (as required): `server/client.go`, `server/client_cloud.go`, `server/plugin.go`, `server/http.go`, `server/user.go`, `plugin.json`, `webapp/`.

### Test results

`cd server && go test ./... -run TestCloudRHS -v` — **PASS** (8/8):

- `TestCloudRHSSearchJQLHappyPath`
- `TestCloudRHSSearchJQLCursorPassthrough`
- `TestCloudRHSSearchJQLRetryAfterHonored`
- `TestCloudRHSSearchJQLGlobalQuotaNoRetry`
- `TestCloudRHSSearchJQLAttemptsCapAt4`
- `TestCloudRHSListStatuses`
- `TestCloudRHSListStatusCategories`
- `TestCloudRHSEndpointURL`

`git grep -n "Issue.Search" server/` — only `server/client.go:306`. No new call sites. `client_cloud_rhs.go` does not call `Issue.Search`, `Status.GetAllStatuses`, or `StatusCategory.GetList`.

### Lint

- `golangci-lint run ./server/client_cloud_rhs.go` reports typecheck false positives (single-file; package types live in other files).
- Package lint (`--new-from-rev=b2bd53b` / `bodyclose`+`errcheck` on `./server`): **no `bodyclose`**, no `errcheck` on the new file.
- `gosec` G404 (`math/rand` jitter) and G115 (`uint(failedAttempt-1)`) fire on the plan’s exact retry helpers. Left as specified (plan §7: keep `math/rand`; no `//nolint` unless `make check-style` requires a change).

### Deviations

None. Types, method bodies, retry algorithm, fixtures, and test names match the plan.

### Gate 2 notes for the reviewer

- Global-quota 429 (`RateLimit-Reason: jira-quota-global-based`, `Retry-After: 1847`) → **exactly one** upstream request, zero sleeps, `errors.Is(err, ErrRateLimited)`.
- Burst 429 with `Retry-After: 7` → sleep is **7s** (not the 2s exponential); second request succeeds.
- Burst 429 with no `Retry-After` → **4** HTTP attempts, 3 sleeps of 2s / 4s / 8s, then `ErrRateLimited`.
- Tenant quota is **not** treated as global (only `jira-quota-global-based` short-circuits).
- `SearchJQL` uses `Do(req, nil)` and `closeJiraResp` on 429, non-429 error, and 200 decode paths.
- `git diff b2bd53b -- server/client.go` is empty. `webapp/` / `plugin.go` / `http.go` diffs are empty.
