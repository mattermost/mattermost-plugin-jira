# Phase 5 Plan: Quality gates and the #36 audit

> Prescriptive verification + handoff plan for **Phase 5 only** (tasks 5.1–5.3).
> This phase does **not** add features. An Implementation Engineer should be able
> to run the gates, apply the three allowed lint fixes if needed, re-execute the
> #36 audit, and write the milestone 2 handoff without making design decisions.
>
> **Do not start Milestone 2. Do not implement the RHS webapp. Do not commit.**
> Orchestration owns the Step 5 commit checkpoint.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase 5 (source of truth for WHAT)
- **Orchestration:** Gate 5 / Step 5 of `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `df32f47` (Phase 4 HTTP layer — already committed)
- **Package:** `package main` under `server/` (module
  `github.com/mattermost/mattermost-plugin-jira`, `go.mod` at repo root)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer.** Sequential. Nothing in this phase is
  parallelizable.

## Scope

**In scope:**

- Run `make test` and `make check-style` (or the Go-only fallback documented
  below if webapp lint/install blocks)
- Re-execute the risk-#36 / Gate 3 audit (`rg` + stub validator + membership)
- Write `.planning/PHASE1_HANDOFF.md` with the exact contract Milestone 2 needs
- **Only if a gate fails on new RHS files:** the three lint fixes listed in
  “Allowed code changes”

**Out of scope (do not do these):**

- Any file under `webapp/` (Milestone 2 / `PLAN-MILESTONE2.md` W1–W6)
- Starting Milestone 2, registering App Bar / RHS / System Console components
- Changing JQL, cache, handlers, routes, DTO, or retry **behavior**
- Fixing `SearchIssues` / Cloud 410 on `rest/api/2/search`
- Editing `.golangci.yml` to exclude G404/G115
- Pushing, tagging, or opening a PR
- Writing production Go except the three allowed lint fixes below

---

## Research (read this before running gates)

Line numbers and commands are as of **`df32f47`** (Phase 4 committed). HEAD
must be that commit, or that commit plus planning-only untracked files.

### 1. What `make test` and `make check-style` actually run

From the worktree `Makefile`:

**`make test`** (`Makefile:339-346`) depends on `apply`,
`webapp/node_modules`, and `install-go-tools`, then:

1. `$(GOBIN)/gotestsum -- -v ./...` — Go tests for **every** module package
   (`server`, `server/utils…`, `build/manifest`, `build/pluginctl`). **Does
   not** pass `GO_TEST_FLAGS` (`-race` is used only by `make coverage`).
2. `cd webapp && npm run test` — full Jest suite (`package.json` script
   `jest --forceExit --detectOpenHandles --verbose`).

**`make check-style`** (`Makefile:188-205`) depends on the same three
prerequisites, then:

1. `cd webapp && npm run lint` — `eslint … --quiet --cache` (HAS_WEBAPP is
   **true**).
2. `go vet $$($(GO) list ./... | grep -v /webapp/)`
3. `$(GOBIN)/golangci-lint run ./...` (v2.6.0 per `install-go-tools`)
4. `go vet -vettool=$(GOBIN)/mattermost-govet -license -license.year=2017` on
   the same non-webapp package list.

`install-go-tools` (`Makefile:181-185`) installs into `./bin` (gitignored):

- `github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.0`
- `gotest.tools/gotestsum@v1.7.0`
- `github.com/mattermost/mattermost-govet/v2@7d8db289e508999dfcac47b97c9490a0fec12d66`

`apply` generates gitignored `server/manifest.go`. If `go test` fails on a
missing `manifest.go`, run `make apply` first.

Enabled golangci linters (`.golangci.yml:25-38`): `bodyclose`, `errcheck`,
`gocritic`, `gosec`, `govet`, `ineffassign`, `misspell`, `nakedret`, `revive`,
`staticcheck`, `unconvert`, `unused`, `whitespace`. Formatters: `gofmt` +
`goimports`. `bodyclose` is **off** for `_test.go`. gosec G104/G304/G301 are
excluded; **G404 and G115 are not**.

### 2. `webapp/node_modules` and whether full `check-style` is runnable

| Fact | As of `df32f47` on this machine |
|------|--------------------------------|
| `HAS_WEBAPP` | `true` (`./build/bin/manifest has_webapp`) |
| `webapp/node_modules` | **does not exist** |
| `webapp/package-lock.json` | present |
| `npm` | present (`nvm` Node v24) |
| `./bin/golangci-lint`, `gotestsum`, `mattermost-govet` | present (gitignored; `install-go-tools` recreates them) |
| `node_modules` in `.gitignore` | **not listed** (do not `git add` it) |

Both `make test` and `make check-style` **will `npm install`** via the
`webapp/node_modules` prerequisite (`Makefile:228-232`) before doing any
lint/test work. That is expected. Do not treat a missing `node_modules` as a
hard blocker.

Master CI (`.github/workflows/ci.yml`) uses `mattermost/actions-workflows`
`plugin-ci.yml`, which runs this Makefile on the untouched webapp. Milestone 1
did not change `webapp/`. So **full `make check-style` is runnable** after
that `npm install`, and eslint on the existing webapp is expected to pass.

**If** `npm install` or `npm run lint` / `npm run test` fails for environment
reasons (network, Node version, pre-existing webapp flake), **do not fix
webapp.** Use the Go-only Gate 5 fallback in “Commands”. Record that fallback
in the Implementation Summary. Milestone 1’s new code is 100% `server/` +
`plugin.json`; Go-only commands still satisfy Gate 5 for that code.

### 3. Known leftover lint — gosec G404 / G115, plus one revive

PE5 re-ran `./bin/golangci-lint run --timeout=5m ./...` from the worktree
root (the same command `make check-style` runs). **Exactly three issues,
all in new RHS files, none pre-existing:**

| File | Line | Linter | Finding |
|------|------|--------|---------|
| `server/client_cloud_rhs.go` | 41 | gosec G404 | `math/rand` in `rhsRandomJitter` |
| `server/client_cloud_rhs.go` | 55 | gosec G115 | `uint(failedAttempt-1)` shift |
| `server/rhs_cache_test.go` | 233 | revive | `max :=` redefines builtin `max` |

Phase 2 RE2 **accepted** G404/G115 because Phase 2 did not require
`make check-style`. Phase 5 **does**. Parent T5.1: fix findings in new code;
do not fix untouched files. There are no findings in untouched files.

`go vet` on non-webapp packages: clean.
`mattermost-govet -license -license.year=2017` on `./server`: clean.
RHS tests at `df32f47`:
`cd server && go test ./... -count=1 -run 'TestRHS|TestCloudRHS'` **PASS**.

**Decision (locked — do not reopen):**

1. **G115 — fix without `//nolint`.** Drop the `uint()` cast. `failedAttempt`
   is 1-indexed and capped at 4, so the shift is 0..3. Exact replacement is
   in “Allowed code changes”.
2. **G404 — fix without `//nolint`.** Phase 2 already prescribed: keep
   `math/rand` unless `make check-style` requires a change; then prefer a
   small `crypto/rand` conversion over `//nolint`. Check-style **does**
   require a change. Convert `rhsRandomJitter` only. Tests inject
   `jitter: func() float64 { return 1.0 }` and never execute production
   jitter. Do **not** switch the whole retry helper to a new algorithm.
3. **revive `max` — fix without `//nolint`.** Rename the local in
   `TestRHSCacheOnConfigurationChangeEmptiesAllInstances` to `maxFileSize`.
4. **Do not** add `//nolint`, `#nosec`, or `.golangci.yml` exclude-rules.
5. **Do not** change retry tests, attempt cap, global-quota behavior, or
   jitter range `[0.7, 1.3]`.

### 4. `validateStatusCategoryOperand` location — Gate 3 still valid after Phase 4

Still exactly here:

```36:41:server/rhs_jql.go
func validateStatusCategoryOperand(operand string, validCategoryKeys map[string]bool) error {
	if operand == "" || validCategoryKeys == nil || !validCategoryKeys[operand] {
		return fmt.Errorf("%w: %q", ErrInvalidStatusCategory, operand)
	}
	return nil
}
```

Call sites in `buildTabJQL` (`server/rhs_jql.go:64` Assigned /
`statusCategoryKeyDone`; `:70` category / `tab.Key`). Assigned is **not**
special-cased around the check.

Phase 4’s only production `buildTabJQL` call (`server/rhs.go:208`):

```go
jql, err := buildTabJQL(selected, sort, validCategoryKeysFrom(entry.categories))
```

`server/rhs.go` contains **no** `statusCategory` string construction. That
must remain true. Gate 3’s stub procedure is unchanged: stub **this
function** to `return nil`, require
`TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys` to **FAIL**, restore.

HTTP-layer proof still present:
`TestRHSHTTPGetIssuesUsesCachedCategoryKeys` (omit `done` from cached
categories → JSON `invalid_request`, `searchCalls == 0`). Do not “fix”
Assigned by hardcoding a four-key map in the handler.

The omit-done test is still `server/rhs_jql_test.go:140-160`. Membership
test is still `TestRHSJQLAssignedExcludesDoneMembership` (`:193-213`) with
`assert.NotContains(..., "TES-DONE")` plus `TES-NEW` / `TES-IP` /
`TES-UNDEF` present.

### 5. Phase 4 API contract (for the handoff)

Captured in “Exact `PHASE1_HANDOFF.md` contents”. Do not invent fields.
Source: `server/rhs_http.go`, `server/rhs.go`, `server/rhs_types.go`,
`server/user.go`, `plugin.json`, `server/plugin.go` tags.

---

## Decisions this phase locks (do not reopen)

**P5-D1 — Verification + handoff, not features.** No new routes, no DTO
changes, no webapp.

**P5-D2 — Full Makefile first, Go-only fallback if webapp blocks.** Try
`make test` / `make check-style`. If webapp `npm install` / lint / Jest
blocks, run the Go-only commands and still treat Gate 5 as pass **for
Milestone 1 server code**. Do not modify `webapp/` to unblock.

**P5-D3 — Lint fixes only in the three new-file findings.** Exact diffs
below. No `//nolint`. No `.golangci.yml` edits.

**P5-D4 — Re-execute Gate 3; do not trust Phase 3’s self-run.** A skipped
`statusCategory` check is invisible at runtime. Stub, watch FAIL, restore.

**P5-D5 — Handoff is the Milestone 2 contract.** D2 (empty picker `value`
must show Assigned locked + In Progress selected, **no `onChange`**) is
the load-bearing item. If that paragraph is missing, this phase is not
done.

**P5-D6 — Do not commit.** Orchestration creates the Step 5 checkpoint
after review.

---

## Implementation order (single engineer)

1. Confirm `git rev-parse HEAD` is `df32f47` (or a descendant that only
   adds planning files). Working tree may contain untracked
   `.planning/PLAN.md` / `.planning/PLAN-MILESTONE2.md` / this file.
   Do not revert those.
2. `make apply` if `server/manifest.go` is missing.
3. **T5.1a** — apply the three allowed lint fixes; re-run golangci until
   `./bin/golangci-lint run --timeout=5m ./...` is clean.
4. **T5.1b** — `make test` (or Go-only fallback).
5. **T5.1c** — `make check-style` (or Go-only fallback). After the lint
   fixes, golangci in this step must report **0** issues.
6. **T5.2** — execute the #36 audit (rg + stub + membership). Restore the
   validator before leaving the tree.
7. **T5.3** — write `.planning/PHASE1_HANDOFF.md` using the exact template
   below. Fill the Verification box with the commands you actually ran.
8. Fill this file’s Implementation Summary. **Stop. Do not start W1.**

---

## Allowed code changes (only if / because gates fail)

These three are **expected**. Apply them **before** declaring T5.1 done.
Touch **only** these two files. Do not reformat unrelated code.

### Fix 1 — G115 (`server/client_cloud_rhs.go` `rhsBackoffDelay`)

Replace the exponential line. Keep cap / Retry-After / jitter logic.

**Current (`:54-55`):**

```go
	exp := retry.base * time.Duration(1<<uint(failedAttempt-1)) // 2s, 4s, 8s, …
```

**Replace with:**

```go
	shift := failedAttempt - 1
	if shift < 0 {
		shift = 0
	}
	exp := retry.base * time.Duration(1<<shift) // 2s, 4s, 8s, …
```

Do not keep `uint(...)`. `TestCloudRHSSearchJQLAttemptsCapAt4` must still
see sleeps 2s / 4s / 8s (jitter 1.0).

### Fix 2 — G404 (`server/client_cloud_rhs.go` `rhsRandomJitter`)

Remove `"math/rand"` from imports. Add `"crypto/rand"`.

**Current (`:40-42`):**

```go
func rhsRandomJitter() float64 {
	return 0.7 + rand.Float64()*(1.3-0.7)
}
```

**Replace with:**

```go
func rhsRandomJitter() float64 {
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 1.0
	}
	return 0.7 + float64(b[0])*(1.3-0.7)/255.0
}
```

`crypto/rand.Read` is `rand.Read` after `import "crypto/rand"`. Range stays
`[0.7, 1.3]`. Fallback `1.0` is “no jitter” if the CSPRNG read fails — not
a retry policy change. Do **not** add a `rhsRandomJitter` unit test. Do
**not** change `testRetryNoSleep` / `defaultRHSRetry` wiring.

### Fix 3 — revive builtin `max` (`server/rhs_cache_test.go`)

In `TestRHSCacheOnConfigurationChangeEmptiesAllInstances` only:

```go
	maxFileSize := int64(100 * 1024 * 1024)
	api.On("GetConfig").Return(&model.Config{
		FileSettings: model.FileSettings{MaxFileSize: &maxFileSize},
	})
```

### After the three fixes, confirm

```bash
./bin/golangci-lint run --timeout=5m ./...
cd server && go test ./... -count=1 -run 'TestCloudRHS|TestRHSCacheOnConfigurationChangeEmptiesAllInstances' -v
```

Expected: golangci **0 issues**; those tests **PASS**.

If golangci reports anything else in **new** RHS files, fix that too
without `//nolint` if it is a one-line issue; stop and report if it
requires behavior change. If it reports an **old** file, do not fix it —
that would be a surprise (PE5 saw none).

---

## Task 5.1 — Full gate run

**Files:** none except the allowed lint fixes.
**Action:** Verify.

From the worktree root:

```bash
# 0. Tooling (safe to re-run; writes gitignored ./bin and server/manifest.go)
make apply
make install-go-tools

# 1. After lint fixes:
./bin/golangci-lint run --timeout=5m ./...
# expected: no issues

# 2. Preferred full gates (will npm install webapp/node_modules on first run)
make test
make check-style
```

**Expected `make test`:** gotestsum lists packages `ok`; Jest finishes
without failing. Duration may be several minutes because of Jest.

**Expected `make check-style`:** eslint quiet-pass; `go vet` silent;
golangci 0 issues; mattermost-govet license silent.

### Go-only fallback (only if webapp install/lint/Jest blocks)

Do **not** edit webapp. Run these from the worktree root, then continue
to T5.2 / T5.3:

```bash
make apply
make install-go-tools

go vet $(go list ./... | grep -v /webapp/)
./bin/golangci-lint run --timeout=5m ./...
go vet -vettool=./bin/mattermost-govet -license -license.year=2017 $(go list ./... | grep -v /webapp/)

./bin/gotestsum -- -v ./...
# equivalent if gotestsum misbehaves:
# go test $(go list ./... | grep -v /webapp/)
```

Also record:

```bash
cd server && go test ./... -count=1 -run 'TestRHS|TestCloudRHS' -v
```

Expected names still present and passing:

- Phase 1: `TestRHSConfig*`
- Phase 2: `TestCloudRHS*` (8)
- Phase 3: `TestRHSJQL*` (10) + `TestRHSCache*` (7)
- Phase 4: `TestRHSHTTP*` (14)

`make coverage` / `-race` are **not** required.

---

## Task 5.2 — Risk #36 audit ⚠️ — execute, do not inspect

**Files:** `server/rhs_jql.go`, `server/rhs_jql_test.go` (add tests **only
if** a step below fails with the real validator in place).
**Action:** Verify.

Run from the worktree root unless noted. Do this **after** lint fixes so
the stub is the only production diff during step 2.

### Step 1 — every `statusCategory` construction sits downstream of validation

```bash
rg -n "statusCategory" server --glob '*.go'
rg -n "validateStatusCategoryOperand" server/rhs_jql.go
rg -n "buildTabJQL" server --glob '*.go'
```

Classify every `statusCategory` hit:

| Kind of hit | Allowed locations |
|-------------|-------------------|
| JQL **construction** (`statusCategory =` / `statusCategory !=`) | **Only** `server/rhs_jql.go` inside `buildTabJQL`, each immediately after `validateStatusCategoryOperand` returns nil |
| Assigned operand | Must be `statusCategoryKeyDone` / emitted `done`. Must **not** skip the validator |
| JSON tag `json:"statusCategory"` | `server/rhs_types.go` — not a construction site |
| HTTP call site | `server/rhs.go` must call `buildTabJQL(..., validCategoryKeysFrom(entry.categories))` and must **not** concatenate `statusCategory` |
| Test fixtures / interpreter / JQL string assertions | `*_test.go` — expected (including `rhs_http_test.go`, `client_cloud_rhs_test.go`) |
| Comments | Fine |

Must **not** exist:

- `statusCategory` concatenation in `rhs.go`, `rhs_http.go`, `rhs_cache.go`,
  `plugin.go`, `client_cloud_rhs.go`
- a hardcoded four-key map passed into `buildTabJQL` from `rhs.go`
- `if kind != assigned { validate(...) }` (Assigned skipping the check)

`validateStatusCategoryOperand` must still be called on **both** the
Assigned branch (`statusCategoryKeyDone`) and the category branch
(`tab.Key`).

### Step 2 — temporary validation removal — suite must FAIL

In `validateStatusCategoryOperand`, **temporarily** replace the body with:

```go
func validateStatusCategoryOperand(operand string, validCategoryKeys map[string]bool) error {
	return nil
}
```

Keep the signature. Do not delete the function (call sites must still
compile). Then:

```bash
cd server && go test ./... -run TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys -v
```

**Expected: FAIL** — `An error is expected but got nil` (Phase 3 recorded
this at `rhs_jql_test.go:150`).

Also expected to FAIL under the same stub (not the required one, but
confirms the category path is not a hardcoded allowlist):

```bash
cd server && go test ./... -run TestRHSJQLCategoryErrorsWhenKeyMissingFromValidKeys -v
```

HTTP-layer extra (should FAIL or return 200-with-search if the handler
bypassed the map — either way, restore after; the **required** failure is
the JQL omit-`done` test):

```bash
cd server && go test ./... -run TestRHSHTTPGetIssuesUsesCachedCategoryKeys -v
```

If `TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys` still
**PASSES** under the stub, the mitigation is untested. Do not weaken the
test. Add or fix a `TestRHSJQL*` assertion so that stubbing the function
makes it fail, then restore and re-run. That is the only allowed test
addition in this phase.

**Restore the original function body immediately** (the 5-line version in
Research §4). Confirm `git diff -- server/rhs_jql.go` is empty (or only
your lint-unrelated accident — it must match `df32f47` for this file).

Then:

```bash
cd server && go test ./... -run 'TestRHSJQL|TestRHSCache|TestRHSHTTP' -v
```

Expected: PASS.

### Step 3 — negated-clause test asserts membership

```bash
rg -n "TES-DONE|NotContains|AssignedExcludesDoneMembership" server/rhs_jql_test.go
```

Confirm **all** of:

- [ ] `TestRHSJQLAssignedExcludesDoneMembership` exists (exact name).
- [ ] It asserts `assert.NotContains(..., "TES-DONE")`.
- [ ] It also asserts non-Done issues **are present** (`TES-NEW` / `TES-IP`).
- [ ] It does **not** treat success as HTTP 200, `err == nil` alone, or
      `assert.NotEmpty` / `len > 0` alone.
- [ ] `TestRHSJQLNegatedBogusOperandWidensToIncludeDone` exists and asserts
      `TES-DONE` **is present** for `statusCategory != bogusCategoryXYZ`.

Do not rewrite these tests “for clarity.” If they already match, leave
them.

---

## Task 5.3 — Write `.planning/PHASE1_HANDOFF.md`

**Files:** `.planning/PHASE1_HANDOFF.md` (create).
**Action:** Create.

Copy the template in the next section **verbatim**, then fill only the
`## Verification` box with the commands and outcomes from T5.1 / T5.2
(full Makefile vs Go-only fallback, lint fixes applied, stub FAIL
observed, validator restored).

Do **not** put implementation narrative in the handoff. Milestone 2
engineers will not read Phase 1–4 plans first; this file is the contract.

Path is the worktree `.planning/PHASE1_HANDOFF.md`, **not** the planner
repo.

---

## Exact `PHASE1_HANDOFF.md` contents

Create this file. After filling Verification, it must still contain every
section below, including the **D2** section in full.

````markdown
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
- **Phase 5 run date:** <YYYY-MM-DD>
- **`make test`:** <pass | not run — reason> <paste summary>
- **`make check-style`:** <pass | not run — reason> <paste summary>
- **Go-only fallback used?** <yes/no>
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
````

---

## File-by-file change list (as of `df32f47`)

| File | Action | What |
|------|--------|------|
| `server/client_cloud_rhs.go` | **Modify (lint only)** | G115 signed shift; G404 `crypto/rand` jitter |
| `server/rhs_cache_test.go` | **Modify (lint only)** | Rename `max` → `maxFileSize` |
| `server/rhs_jql.go` | Verify only (temporary stub during T5.2, then restore) | Must match `df32f47` when done |
| `server/rhs_jql_test.go` | Verify; extend **only** if stub does not FAIL | #36 omit-`done` + membership |
| `.planning/PHASE1_HANDOFF.md` | **Create** | Exact template above |

**Do not modify:** `webapp/**`, `.golangci.yml`, `server/issue.go`,
`server/client.go`, `server/http.go`, `plugin.json`, `server/rhs.go`,
`server/rhs_http.go`, `server/user.go` (unless a surprise lint finding
appears in a **new** RHS file not listed — then fix that file only).

---

## Commands (cheat sheet)

Worktree root:
`~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`

```bash
git rev-parse HEAD   # expect df32f47…

make apply
make install-go-tools

# After allowed lint fixes:
./bin/golangci-lint run --timeout=5m ./...
go vet $(go list ./... | grep -v /webapp/)
go vet -vettool=./bin/mattermost-govet -license -license.year=2017 $(go list ./... | grep -v /webapp/)

make test            # preferred
make check-style     # preferred; includes webapp eslint

# Go-only fallback if webapp blocks:
./bin/gotestsum -- -v ./...

# #36
rg -n "statusCategory" server --glob '*.go'
# stub validateStatusCategoryOperand to return nil
cd server && go test ./... -run TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys -v
# expect FAIL; restore; then:
cd server && go test ./... -run 'TestRHSJQL|TestRHSCache|TestRHSHTTP' -v
rg -n "TES-DONE|NotContains|AssignedExcludesDoneMembership" server/rhs_jql_test.go
```

---

## Gate 5 checklist (reviewer)

From `IMPL_ORCHESTRATION_PLAN.md` Step 5. Execute; not inspect-only.

### 1. Tests

- [ ] `make test` passes, **or** Go-only fallback is documented and
      `gotestsum -- -v ./...` (or `go test` non-webapp) passes.
- [ ] `cd server && go test ./... -run 'TestRHS|TestCloudRHS'` passes
      (Phases 1–4 names still present).

### 2. Style

- [ ] `make check-style` passes, **or** Go-only `go vet` +
      `golangci-lint run ./...` + mattermost-govet license pass.
- [ ] `golangci-lint run ./...` reports **0** issues (G404/G115/revive
      `max` are gone).
- [ ] No `//nolint` / `#nosec` added. No `.golangci.yml` exclude-rule
      added for G404/G115.
- [ ] New `server/*.go` files still have the 2017 Mattermost license
      header.

### 3. #36 audit executed in this phase

- [ ] `rg` classification completed; only `buildTabJQL` constructs
      `statusCategory` JQL; Assigned goes through
      `validateStatusCategoryOperand(statusCategoryKeyDone, …)`.
- [ ] `server/rhs.go` still uses
      `validCategoryKeysFrom(entry.categories)` — no hardcoded map.
- [ ] Stub → `TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys`
      **FAIL** observed (not assumed).
- [ ] Validator **restored**; `git diff -- server/rhs_jql.go` empty
      versus the post-lint-fix tree (no leftover `return nil`).
- [ ] Membership test still asserts `TES-DONE` **absent** and a non-Done
      issue **present**.

### 4. Handoff

- [ ] `.planning/PHASE1_HANDOFF.md` exists.
- [ ] It lists both routes, query params, DTO fields, the six JSON error
      codes, `rhs_enabled`, and `RHSStatusTabs` shape.
- [ ] It contains the **D2** constraint in full: empty `value` → render
      Assigned (locked) + In Progress selected, **no `onChange`**.
- [ ] It states `/rhs/statuses` is not gated on `rhs_enabled`.

### 5. Blast radius / process

- [ ] `webapp/` diff is empty (aside from an untracked `node_modules`
      after `npm install` — **do not add it**).
- [ ] `git diff --stat df32f47 -- server/issue.go server/client.go` is
      empty.
- [ ] Milestone 2 was **not** started.
- [ ] **No commit / no push** by the Phase 5 implementer.

**Commit checkpoint (orchestration):** local commit of lint fixes (if
any) + `PHASE1_HANDOFF.md` after review. Do not push unless explicitly
asked.

---

## What this phase hands to Milestone 2

`.planning/PHASE1_HANDOFF.md` is the boundary. W1 can theoretically start
without the server (enum/fixture bugs), but W2+ need this contract.
Orchestration Step 6 is the next execution step — **not this implementer.**

---

## Implementation Summary

Filled by IE5 on 2026-08-19. HEAD started at `df32f47`. **No commit. No push.** Milestone 2 / webapp not started.

### Commands run and results

| Command | Result |
|---------|--------|
| `git rev-parse HEAD` | `df32f4702c8433998e856c39b13a9383348e0446` |
| `make apply` + `make install-go-tools` | pass |
| `./bin/golangci-lint run --timeout=5m ./...` (after lint fixes) | **0 issues** |
| `cd server && go test ./... -count=1 -run 'TestCloudRHS\|TestRHSCacheOnConfigurationChangeEmptiesAllInstances' -v` | **PASS** (8 `TestCloudRHS*` + cache test). Gate 2 still holds: `TestCloudRHSSearchJQLGlobalQuotaNoRetry`, `TestCloudRHSSearchJQLAttemptsCapAt4` (sleeps 2s/4s/8s with jitter 1.0), `TestCloudRHSSearchJQLRetryAfterHonored`. |
| `make test` | **pass** (exit 0). gotestsum packages `ok` (`server` 8.313s). Jest 13 suites / 83 tests. See deviation below. |
| `make check-style` | **pass**. eslint `--quiet` clean; `go vet` silent; golangci **0 issues**; mattermost-govet license silent. |
| `cd server && go test ./... -count=1 -run 'TestRHS\|TestCloudRHS'` | **PASS** (Phases 1–4 names still present). |
| Go-only fallback | **not used** |

### Lint fixes made

Matched this plan exactly. No `//nolint` / `#nosec`. No `.golangci.yml` edits.

1. **G115** `server/client_cloud_rhs.go` `rhsBackoffDelay`: dropped `uint()`; `shift := failedAttempt - 1` with `shift < 0` clamp; `1<<shift`.
2. **G404** `server/client_cloud_rhs.go` `rhsRandomJitter`: `math/rand` → `crypto/rand`; one-byte read, range `[0.7, 1.3]`, CSPRNG-fail fallback `1.0`.
3. **revive** `server/rhs_cache_test.go`: `max` → `maxFileSize` in `TestRHSCacheOnConfigurationChangeEmptiesAllInstances`.

### Gate 3 / #36 audit re-run

`rg -n "statusCategory" server --glob '*.go'`: JQL construction (`statusCategory =` / `!=`) **only** in `buildTabJQL` (`server/rhs_jql.go`), each immediately after `validateStatusCategoryOperand`. Assigned uses `statusCategoryKeyDone` and does **not** skip the validator. `server/rhs.go` has **no** `statusCategory` string; it calls `buildTabJQL(..., validCategoryKeysFrom(entry.categories))`. Other hits: JSON tag, constants, tests.

Stub `validateStatusCategoryOperand` to `return nil`. Tests that **FAILED** (required):

- `TestRHSJQLAssignedErrorsWhenDoneOmittedFromValidKeys` — `rhs_jql_test.go:150` `An error is expected but got nil`
- `TestRHSJQLCategoryErrorsWhenKeyMissingFromValidKeys` — `rhs_jql_test.go:167` `An error is expected but got nil` (bogus-key / category path)
- `TestRHSHTTPGetIssuesUsesCachedCategoryKeys` — expected HTTP 400, actual 200

Validator **restored**. `git diff -- server/rhs_jql.go` empty vs `df32f47`. Then `cd server && go test ./... -run 'TestRHSJQL|TestRHSCache|TestRHSHTTP' -v` **PASS**.

Membership: `TestRHSJQLAssignedExcludesDoneMembership` still asserts `assert.NotContains(..., "TES-DONE")` plus `TES-NEW` / `TES-IP` / `TES-UNDEF` present. `TestRHSJQLNegatedBogusOperandWidensToIncludeDone` still asserts `TES-DONE` present for `statusCategory != bogusCategoryXYZ`. Tests not rewritten.

### Handoff path

`.planning/PHASE1_HANDOFF.md` (worktree). Includes D2 in full: empty picker `value` must render Assigned (locked) + In Progress selected and **must not** call `onChange` to materialize them.

### Deviations

- `make test` / gotestsum printed 5 `(unknown)` FAIL lines for **pre-existing** `TestEditSubscriptionTemplate` / `TestGetSubscriptionTemplate`. Those tests `--- PASS`; stdout contains `/n httpGetSubscriptionTemplates`, which confuses gotestsum’s parser. Overall `make test` exit 0; Jest passed. **Did not fix** untouched files.
- `npm install` mutated `webapp/package-lock.json`. **Restored** with `git checkout -- webapp/package-lock.json`. `webapp/` tracked diff vs `df32f47` is empty. Did not add `node_modules`.
- Temporary stub of `validateStatusCategoryOperand` during T5.2; restored before leaving the tree.

### Blockers

None. Phase 5 complete. Do not start W1 / Milestone 2. Orchestration owns the Step 5 commit checkpoint.
