# Phase W6 Plan: Tests and quality gates

> Prescriptive verification plan for **Phase W6 only** (W6.1 audit + W6.2
> gates). An Implementation Engineer should be able to close Gate 9 without
> making design decisions. This phase does **not** add features.
>
> **Do not write production TS/JS until this plan is followed as written.**
> This document is the plan; it is not the code.
>
> **Default outcome of W6.1:** add **zero** tests. W1–W5 already landed the
> Gate 6–8 matrix. Only add a test if a named row in the gap table is
> actually missing.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase W6 (source of truth for WHAT)
- **Orchestration:** Step 9 / **Gate 9** of
  `IMPL_ORCHESTRATION_PLAN.md`
- **Prior phase plans (read Implementation Summaries):**
  `.planning/phase-w1/PLAN.md` … `.planning/phase-w5/PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `d6fbfdd` (W4 committed). W1–W5 are done:
  `6e6e073` W1, `cb5ec3a` W2, `b0ee5cd` W3, `1e50575` W5, `d6fbfdd` W4
- **Package:** `webapp/` (Jest 29, RTL 14, TypeScript 5.7 strict, ESLint 8)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer.** Sequential. Nothing in this phase is
  parallelizable. No production feature work.

## Scope

**In scope:**

- **W6.1** — audit existing Gate 6–8 tests by **exact name**. Add a test
  only if a gap-table row is missing. If every row is present, add
  **nothing**.
- **W6.2** — run `npm run lint`, the `check-types` **delta** (not
  `tsc` exit 0), and `npm run test`. Confirm no new `en.json`, no new
  `FormattedMessage`, no new suppressions.

**Out of scope (do not do these):**

- New features, UI copy, new endpoints, new Redux slices
- i18n adoption (`en.json`, `FormattedMessage`, `react-intl` messages)
  — tracked as milestone F1
- Fixing the **249** pre-existing `error TS` diagnostics that already
  exist on `master` (see Research §2)
- Deleting pre-existing `eslint-disable` or `FormattedMessage` in
  **old** files (channel subscriptions, confirm modal, webpack, …)
- `@ts-ignore` / `@ts-expect-error` / `eslint-disable` to “make
  check-types green”
- Raising `max-lines`, changing `.eslintrc.json` / `tsconfig.json`
- `make test` / `make check-style` (those also run Go). W6 is webapp
- Anything under `server/`
- Rewriting Gate 6–8 tests “for clarity” when they already pass
- Creating the parent File Change Map’s `webapp/src/**/*.test.tsx`
  bucket — those files already exist from W1–W5
- Committing or pushing

---

## Research (read this before running gates)

Line numbers and counts are as of **`d6fbfdd`**. If they have drifted,
search; do not invent a different pass/fail rule.

### 1. Parent W6 vs what W1–W5 already shipped

Parent `.planning/PLAN.md` W6.1 says “Create” component/integration
tests covering tab strip, sort reset, load more, five states, W4
gating, popout rehydrate, and the two W5.3 rules (D2 virtual seed; no
`onChange` while disconnected). Gate 9’s highest-value assertions are
the **negatives**.

Those tests were required by Gates 6–8 and **already exist** in the
files W1–W5 created. Parent File Change Map row
`webapp/src/**/*.test.tsx | W6 | Create` is stale. W6 is an audit +
gate run, not a second test-writing phase.

W4 Implementation Summary recorded **193 tests** / lint pass after W5
+ W4. Do not drop below that.

### 2. `check-types` — researched against `master` (LOAD-BEARING)

`webapp/package.json` script:

```
"check-types": "tsc"
```

`webapp/tsconfig.json`: `strict` + `strictNullChecks`, `noEmit: true`,
`skipLibCheck: true`, `include: ["./**/*"]` (production **and** tests).
`tsc` therefore typechecks `*.test.ts(x)`.

**`npm run check-types` is not, and has never been, exit 0 on this
plugin.** Treating Gate 9 as “`tsc` exits 0” would force a rewrite of
`actions/index.ts` (`getState: GlobalState` lie, 48 errors),
`edit_channel_subscription.test.tsx` (31 errors), react-bootstrap
`Modal` typings, and other pre-milestone files. W1–W5 all forbade
that. **Do not start.**

Fair comparison (PE11, 2026-08-19):

| Tree | How | `error TS` count | Files with errors |
|------|-----|------------------|-------------------|
| `master` `737f3cb` | `npx tsc --pretty false --noEmit` **after** copying gitignored `webapp/src/manifest.ts` | **249** | 48 |
| HEAD `d6fbfdd` | same, worktree already has `manifest.ts` | **249** | 48 |

Per-file counts are **identical**. Files only on current: **none**.
Files only on master: **none**.

**No new RHS file appears in the diagnostic list.** Confirmed empty:

```
src/actions/rhs.ts
src/actions/rhs.test.ts
src/client/index.test.ts
src/utils/rhs_*
src/components/rhs/**
src/components/admin_console/rhs_status_setting/**
src/plugin.test.ts
```

`plugin.tsx` still has the **same 6 pre-existing** diagnostics as
master (line numbers drifted because W4/W5 inserted lines). Messages:

1. `(dispatch, getState) => Promise<any>` not assignable to `Action<object>`
2. `() => object` not assignable to `GlobalState`
3. `Property 'ui_enabled' does not exist on type 'Action<object>'`
4. `Cannot find name 'PluginRegistry'`
5. `'React' refers to a UMD global`
6. `(() => Promise<void>) | undefined` not assignable to `(...args: any[]) => any`

Touched existing files that stay at the master count:

| File | `error TS` on master and HEAD |
|------|-------------------------------|
| `src/actions/index.ts` | 48 |
| `src/components/modals/channel_subscriptions/edit_channel_subscription.test.tsx` | 31 |
| `src/plugin.tsx` | 6 |
| `src/selectors/index.ts` | 0 |
| `src/reducers/index.ts` | 0 |
| `src/client/index.ts` | 0 |
| `src/types/model.ts` | 0 |
| `src/action_types/index.ts` | 0 |
| `src/testlib/test-utils.tsx` | 0 |

**`manifest.ts` trap:** that file is gitignored (`.gitignore:9`) and
produced by `make apply`. A fresh checkout / temp worktree **without**
it reports **254** errors — the extra 5 are all
`TS2307 Cannot find module './manifest'` (or `../manifest`) in
`plugin.tsx`, `index.ts`, `action_types/index.ts`, `selectors/index.ts`,
`manifest.test.tsx`. W1’s “249 pre-existing” assumed `manifest.ts` is
present (it is, in this worktree). If you see 254 and five are missing
`manifest`, run `make apply` from the repo root (or confirm
`webapp/src/manifest.ts` exists). **Do not treat those five as new W6
errors. Do not commit `manifest.ts`.**

**Gate 9 rule (locked):**

- Fail W6 **only** if a **new file** (not on `master`) appears in the
  `error TS` list, **or** an existing file **gains** a diagnostic that
  is not on `master` (same message text, ignoring line numbers).
- Do **not** fail W6 because `tsc` exits 1.
- Do **not** fail W6 because the total is still 249.
- Do **not** require fixing the 249 unless they are actually new.

W3 once reported 254 while untracked W5 files still had type errors;
those were fixed before W5 commit. HEAD is back to the 249 baseline.

### 3. Lint and i18n today

`npm run lint` is `eslint … --quiet --cache` (`package.json:11`).
`--quiet` hides warnings (`explicit-function-return-type`,
`no-magic-numbers`). Errors must be 0.

As of `d6fbfdd` (PE11 re-ran): **lint already passes**.

Gate 9 “zero suppressions” means **zero new** suppressions versus
`master`, not “delete every historical `eslint-disable`.”

PE11 vs `master` (`737f3cb`):

| Signal | Extra on HEAD vs master |
|--------|-------------------------|
| `eslint-disable` | **none** |
| `FormattedMessage` | **none** |
| `@ts-ignore` / `@ts-expect-error` / `@ts-nocheck` | **none** |
| `en.json` | **none on HEAD, none on master** |

Pre-existing (leave them; they are not W6 failures):

| File | What |
|------|------|
| `webapp/src/actions/index.ts:576` | `eslint-disable-line no-magic-numbers` |
| `webapp/src/components/modals/channel_subscriptions/edit_channel_subscription.test.tsx:4` | `eslint-disable max-lines` |
| `webapp/src/components/react_select_setting.tsx` | `eslint-disable-line react/no-did-update-set-state` |
| `webapp/src/components/data_selectors/backend_selector.tsx` | same |
| `webapp/src/components/jira_issue_selector/jira_issue_selector.jsx` | same |
| `webapp/src/components/input.jsx` | same |
| `webapp/webpack.config.js` | `no-process-env` / `no-console` |
| `webapp/src/components/confirm_modal.tsx` | `FormattedMessage` |
| `webapp/src/components/modals/full_screen_modal/close_icon.tsx` | `FormattedMessage` |
| `webapp/src/components/modals/full_screen_modal/back_icon.tsx` | `FormattedMessage` |

RHS / admin / W2–W4 helper trees have **no** `eslint-disable`, **no**
`FormattedMessage`, **no** `en.json`.

### 4. Existing test inventory (do not duplicate)

#### Gate 6 — Cloud-oauth gating (W1)

`webapp/src/selectors/index.test.ts`

- `isCloudInstance is true for CLOUD_OAUTH`
- `isCloudInstance is true for CLOUD`
- `isCloudInstance is false for SERVER`
- `hasCloudInstance is true for a cloud-oauth-only installation`
- `hasCloudInstance is true for a cloud-JWT-only installation`
- `hasCloudInstance is false when only Server/DC is installed`
- `hasCloudInstance is false when nothing is installed`
- `getConnectedCloudInstances returns only Cloud types`
- `getConnectedCloudInstances drops a connected instance that is not installed`
- `defaultMockState yields a non-empty getUserConnectedInstances`

#### Gate 7 — debounce + additive JSON helper (W2)

`webapp/src/actions/rhs.test.ts`

- `a second fetchRHSIssues for the same instance tab sort while in flight issues no second network call`
- `fetchRHSIssues for a different tab while in flight does call fetch again`
- `fetchRHSIssues after the in-flight request settles does call fetch again`
- `changing sort resets the list and cursor rather than appending`
- `loadMoreRHSIssues appends rather than replaces`
- `loadMoreRHSIssues does not fetch when isLast`
- `rhsLoading is true and issues are empty during a page-1 fetch`
- plus `not_connected` typed code, `fetchRHSStatuses` isolation, persist

`webapp/src/client/index.test.ts`

- `doFetchWithResponse still throws ClientError with the raw non-JSON body`
- plus JSON / text fallback / snake_case URL tests

#### Gate 8 — loading≠empty (W3)

`webapp/src/components/rhs/rhs.test.tsx`

- `loading state is not the empty state`
- `empty state renders after a successful zero-row fetch`
- `error state renders for internal_error`
- `not_connected error renders the connect state`
- `zero connected Cloud instances renders not-connected without fetching`
- `rate_limited renders the distinct rate-limit state`
- `instance picker is absent with one connected Cloud instance`
- `instance picker is present with two connected Cloud instances`
- `changing sort dispatches resolveAndFetchRHSIssues with created`
- `selecting a tab dispatches resolveAndFetchRHSIssues with that tab`
- `load more is shown when not last and click calls loadMoreRHSIssues`
- `refresh is disabled while loading so a click is not treated as a page-1 replace`
- `new ticket dispatches openCreateModalWithoutPost with empty description and the channel id`
- `on mount getConnected runs before restoreRHSViewState before resolveAndFetchRHSIssues`
- `booting shows loading not empty`

`webapp/src/selectors/index.test.ts`

- `getRHSLoading and getRHSIssues keep loading distinguishable from empty`

#### Gate 8 — registration + popout (W4)

`webapp/src/plugin.test.ts`

- `setupUILater does not call registerAppBarComponent when rhs_enabled is false`
- `setupUILater does not call registerAppBarComponent when only Server/DC is installed`
- `setupUILater calls registerAppBarComponent for a cloud-oauth-only installation when rhs_enabled is true`
- `setupUILater does not nest App Bar registration inside ui_enabled`

`webapp/src/utils/rhs_register.test.ts`

- `shouldRegisterJiraRHS is false when rhs_enabled is false even with a Cloud instance`
- `shouldRegisterJiraRHS is false when rhs_enabled is true but only Server/DC is installed`
- `shouldRegisterJiraRHS is true for a cloud-oauth-only installation when rhs_enabled is true`
- `shouldRegisterJiraRHS is true for a cloud-JWT-only installation when rhs_enabled is true`
- `shouldRegisterJiraRHS is false when settings is an error object`

`webapp/src/utils/rhs_popout.test.ts`

- `isRHSPopoutPathname is true for the 11.3 popout path shape`
- `isRHSPopoutPathname is true for the 11.6 popout path shape`
- plus two negative path tests

`webapp/src/actions/rhs.test.ts`

- `a simulated /_popout/ pathname rehydrates the persisted view state and triggers a fresh issues fetch`

`webapp/src/components/rhs/rhs.test.tsx`

- `popout pathname sets data-rhs-popout and still restores then fetches`

#### Gate 8 — D2 / W5.3 (W5)

`webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.test.tsx`

- `empty value renders Assigned and In Progress selected without calling onChange`
  (`test.each` for `null` / `{}` / `{[id]: []}`)
- `not_connected keeps stored chips disables the status control and does not call onChange`
- `selection follows props.value not a hardcoded config PluginSettings path`
- `a failed fetch with empty options does not call onChange`
- `an empty 200 fetch does not call onChange`
- plus Assigned lock, user-select `onChange` + `setSaveNeeded`, Cloud-only
  instances, silent `getConnected` failure

`webapp/src/components/admin_console/rhs_status_setting/rhs_status_options.test.ts`

- `displayTabsForInstance returns Assigned and In Progress when value is empty`
- `buildPersistedValue does not materialize the virtual seed when the instance is already empty`

### 5. Audited and **not** gaps (do not add)

These are real product behaviors, already covered at another layer, or
outside the Gate 9 matrix (gating / popout / D2 / loading≠empty /
debounce). **Do not write them in W6.**

| Idea | Why not a W6 add |
|------|------------------|
| `setupUILater` cloud-JWT-only | Helper test exists; Gate 8.3 required **oauth**-only at the registry |
| `setupUILater` with zero instances | `hasCloudInstance` false + Server/DC-only registry test |
| `fetchRHSIssues` persist omits `issues` | Gate 8.5 + `rhs_view_state` round-trip only writes `{instance,tab,sort}` |
| Instance switch resets tab to Assigned | W3 handler; not a Gate 9 negative |
| Load-more footer error keeps rows | W2 reducer + W3 list; not in the matrix |
| Connected-container `renderWithRedux` for `components/rhs` | Presentational tests + `plugin.test.ts` cover the gates. `renderWithRedux` uses `redux-mock-store` and does **not** run reducers |

---

## Surprises

### Surprise A — parent W6.1 “Create tests” is stale

W1–W5 already created every Gate 6–8 file. Adding a second suite that
repeats those names wastes review and can flake. Audit first.

### Surprise B — `check-types` cannot be “green” as exit 0

249 `error TS` on `master` and on HEAD. Fail only on **new**
diagnostics. Do not “fix the plugin.”

### Surprise C — missing `manifest.ts` fake-inflates the count by 5

Gitignored generated file. This worktree has it. A bare `master`
worktree does not. `make apply` if needed; never commit it.

### Surprise D — Gate 9 i18n / suppressions are a **delta vs master**

Old files already use `FormattedMessage` and `eslint-disable`. New
RHS/admin files do not. Compare to `master`, do not clean the repo.

### Surprise E — `plugin.tsx` will still fail `tsc`

Six pre-existing host-typing errors. Line numbers moved. Same
messages → pass. A **seventh** new message → fail W6.

---

## Gap list (add only if missing)

PE11 inventory at `d6fbfdd`: **every row is Present.** Expected
W6.1 result: **add zero tests.**

If a name is gone (deleted/renamed), restore it in the **existing**
file with the **exact** name below. Do not invent a new file. Do not
rename a passing test to match this table.

| # | Matrix cell | Exact test name (must exist) | File | ADD? |
|---|-------------|------------------------------|------|------|
| G1 | gating — oauth-only selector | `hasCloudInstance is true for a cloud-oauth-only installation` | `selectors/index.test.ts` | **no** |
| G2 | gating — off + Cloud | `setupUILater does not call registerAppBarComponent when rhs_enabled is false` | `plugin.test.ts` | **no** |
| G3 | gating — Server/DC only | `setupUILater does not call registerAppBarComponent when only Server/DC is installed` | `plugin.test.ts` | **no** |
| G4 | gating — oauth-only register | `setupUILater calls registerAppBarComponent for a cloud-oauth-only installation when rhs_enabled is true` | `plugin.test.ts` | **no** |
| G5 | gating — helper oauth-only | `shouldRegisterJiraRHS is true for a cloud-oauth-only installation when rhs_enabled is true` | `utils/rhs_register.test.ts` | **no** |
| P1 | popout — 11.3 path | `isRHSPopoutPathname is true for the 11.3 popout path shape` | `utils/rhs_popout.test.ts` | **no** |
| P2 | popout — 11.6 path | `isRHSPopoutPathname is true for the 11.6 popout path shape` | `utils/rhs_popout.test.ts` | **no** |
| P3 | popout — rehydrate + fetch | `a simulated /_popout/ pathname rehydrates the persisted view state and triggers a fresh issues fetch` | `actions/rhs.test.ts` | **no** |
| D1 | D2 — empty value, no `onChange` | `empty value renders Assigned and In Progress selected without calling onChange` | `rhs_status_setting.test.tsx` | **no** |
| D2 | D2 — `not_connected`, no `onChange` | `not_connected keeps stored chips disables the status control and does not call onChange` | `rhs_status_setting.test.tsx` | **no** |
| L1 | loading≠empty — visual | `loading state is not the empty state` | `components/rhs/rhs.test.tsx` | **no** |
| L2 | loading≠empty — booting | `booting shows loading not empty` | `components/rhs/rhs.test.tsx` | **no** |
| L3 | loading≠empty — selector | `getRHSLoading and getRHSIssues keep loading distinguishable from empty` | `selectors/index.test.ts` | **no** |
| L4 | loading≠empty — hung fetch | `rhsLoading is true and issues are empty during a page-1 fetch` | `actions/rhs.test.ts` | **no** |
| B1 | debounce — in-flight | `a second fetchRHSIssues for the same instance tab sort while in flight issues no second network call` | `actions/rhs.test.ts` | **no** |

Parent W6.1 also listed tab strip / sort reset / load more / five
states. Those names are already in `rhs.test.tsx` / `rhs.test.ts`
(Research §4). Not gaps.

**If and only if** a row is missing, add **that one test** to the
listed file. Copy the assertion recipe from the phase plan that first
required it (W1.4, W2 Gate 7, W3 Gate 8, W4 Gate 8, W5 Gate 8). Do not
redesign the production code to make a missing test easier.

---

## Tasks

### W6.1 — Audit existing tests; add only real gaps

**Files:** none, unless a gap-table name is missing.
**Action:** Verify. Create/append **only** on a missing name.

1. From `webapp/`, confirm every gap-table name exists:

```bash
cd webapp
python3 - <<'PY'
import pathlib, re, sys
root = pathlib.Path('src')
needles = [
    'hasCloudInstance is true for a cloud-oauth-only installation',
    'setupUILater does not call registerAppBarComponent when rhs_enabled is false',
    'setupUILater does not call registerAppBarComponent when only Server/DC is installed',
    'setupUILater calls registerAppBarComponent for a cloud-oauth-only installation when rhs_enabled is true',
    'shouldRegisterJiraRHS is true for a cloud-oauth-only installation when rhs_enabled is true',
    'isRHSPopoutPathname is true for the 11.3 popout path shape',
    'isRHSPopoutPathname is true for the 11.6 popout path shape',
    'a simulated /_popout/ pathname rehydrates the persisted view state and triggers a fresh issues fetch',
    'empty value renders Assigned and In Progress selected without calling onChange',
    'not_connected keeps stored chips disables the status control and does not call onChange',
    'loading state is not the empty state',
    'booting shows loading not empty',
    'getRHSLoading and getRHSIssues keep loading distinguishable from empty',
    'rhsLoading is true and issues are empty during a page-1 fetch',
    'a second fetchRHSIssues for the same instance tab sort while in flight issues no second network call',
]
text = '\n'.join(p.read_text() for p in root.rglob('*.test.ts')) + '\n' + '\n'.join(p.read_text() for p in root.rglob('*.test.tsx'))
missing = [n for n in needles if n not in text]
if missing:
    print('MISSING')
    for n in missing:
        print(' -', n)
    sys.exit(1)
print('all Gate 9 matrix names present')
PY
```

2. If the script prints `all Gate 9 matrix names present`: **stop
   W6.1.** Do not add tests. Do not create
   `webapp/src/components/rhs/*.test.tsx` or a new admin test file.
3. If it prints `MISSING`: add only those names, in the file from the
   gap table, using the original phase’s recipe. No `eslint-disable`.
   No `FormattedMessage`. No `en.json`.

### W6.2 — Gates

**Files:** none (verification only), unless lint/test/i18n/tsc-delta
fails for a reason you introduced in step 1.
**Action:** Verify.

Run the Commands section **in order**. Record outcomes in the
Implementation Summary at the bottom.

If lint fails on a **new** W6.1 test, fix that test. If lint fails on
unrelated pre-existing files, stop and report — do not “clean up”
master-era code.

If `check-types` delta fails (new file or new message), fix the **new**
diagnostic only. Do not touch the 249.

---

## Commands

From `webapp/` unless noted. `node_modules` is already installed. Do
**not** `npm install`. Do **not** run `make test`.

This worktree already has gitignored `webapp/src/manifest.ts`. Confirm
before `tsc`:

```bash
test -f webapp/src/manifest.ts || (echo 'missing manifest.ts — run make apply from repo root' && exit 1)
```

### 1. Lint

```bash
cd webapp
npm run lint
```

Must print nothing under `--quiet` (exit 0). No new `eslint-disable`.

### 2. Full Jest suite (no path filter)

```bash
cd webapp
npm run test
```

Must pass. Expect **≥ 193** tests (W4 baseline). Filtering while
iterating a restored gap test is fine; declaring Gate 9 done requires
the full suite.

### 3. `check-types` delta — do **not** require exit 0

```bash
cd webapp
npx tsc --pretty false --noEmit > /tmp/tsc-w6.txt || true
```

`tsc` **will** exit 1. That is expected. Then:

```bash
cd webapp
python3 - <<'PY'
import pathlib, re, sys

text = pathlib.Path('/tmp/tsc-w6.txt').read_text()
lines = [ln for ln in text.splitlines() if 'error TS' in ln]
print('error TS count:', len(lines))

new_file_re = re.compile(
    r'^src/(?:'
    r'actions/rhs|'
    r'client/index\.test|'
    r'utils/rhs_|'
    r'components/rhs/|'
    r'components/admin_console/rhs_status_setting/|'
    r'plugin\.test'
    r')'
)
new_hits = [ln for ln in lines if new_file_re.search(ln)]
if new_hits:
    print('FAIL: new RHS / W6 files have tsc errors')
    print('\n'.join(new_hits[:40]))
    sys.exit(1)

# plugin.tsx: allow the 6 master messages; fail on any other message
plugin = [ln for ln in lines if ln.startswith('src/plugin.tsx')]
allowed = [
    "Argument of type '(dispatch: Dispatch, getState: GlobalState) => Promise<any>' is not assignable to parameter of type 'Action<object>'",
    "Argument of type '() => object' is not assignable to parameter of type 'GlobalState'",
    "Property 'ui_enabled' does not exist on type 'Action<object>'",
    "Cannot find name 'PluginRegistry'",
    "'React' refers to a UMD global",
    "Type '(() => Promise<void>) | undefined' is not assignable to type '(...args: any[]) => any'",
]
unknown = []
for ln in plugin:
    if not any(a in ln for a in allowed):
        unknown.append(ln)
if unknown:
    print('FAIL: plugin.tsx gained a new diagnostic')
    print('\n'.join(unknown))
    sys.exit(1)
if len(plugin) > 6:
    print('FAIL: plugin.tsx has more than the 6 pre-existing errors')
    sys.exit(1)

# missing-manifest artifact — not a W6 failure, but do not proceed without applying
manifest_miss = [ln for ln in lines if "Cannot find module" in ln and 'manifest' in ln]
if manifest_miss:
    print('WARN: missing generated manifest.ts inflated tsc; run make apply and re-run')
    print('\n'.join(manifest_miss))
    sys.exit(1)

if len(lines) > 249:
    print('FAIL: error TS count rose above the 249 master/HEAD baseline')
    sys.exit(1)

print('check-types delta PASS')
print('  total error TS:', len(lines), '(249 expected; lower is fine)')
print('  plugin.tsx pre-existing:', len(plugin))
print('  new RHS files: 0')
PY
```

Optional re-compare vs `master` (needs `manifest.ts` on that tree).
PE11 already did this: **249 == 249**, identical per-file counts.
Re-run only if HEAD moved:

```bash
# from worktree root — do not leave this worktree lying around
git worktree add --detach /tmp/jira-master-tsc master
cp webapp/src/manifest.ts /tmp/jira-master-tsc/webapp/src/manifest.ts
ln -sfn "$(pwd)/webapp/node_modules" /tmp/jira-master-tsc/webapp/node_modules
(cd /tmp/jira-master-tsc/webapp && npx tsc --pretty false --noEmit > /tmp/tsc-master.txt || true)
echo "master=$(grep -c 'error TS' /tmp/tsc-master.txt) head=$(grep -c 'error TS' /tmp/tsc-w6.txt)"
git worktree remove --force /tmp/jira-master-tsc
```

A total **below** 249 is a pass (someone fixed an old error). A total
**above** 249 is a fail unless the extras are the five missing-manifest
`TS2307`s — then apply manifest and re-run.

### 4. i18n / eslint-disable / ts-suppression audit (delta vs `master`)

From the **worktree root**:

```bash
echo '=== new eslint-disable vs master ==='
comm -13 \
  <(git grep -n 'eslint-disable' master -- webapp | sed 's|^[^:]*:||' | sort) \
  <(git grep -n 'eslint-disable' HEAD -- webapp | sed 's|^[^:]*:||' | sort)

echo '=== new FormattedMessage vs master ==='
comm -13 \
  <(git grep -n 'FormattedMessage' master -- webapp | sed 's|^[^:]*:||' | sort) \
  <(git grep -n 'FormattedMessage' HEAD -- webapp | sed 's|^[^:]*:||' | sort)

echo '=== new @ts-ignore / @ts-expect-error / @ts-nocheck vs master ==='
comm -13 \
  <(git grep -nE '@ts-ignore|@ts-expect-error|@ts-nocheck' master -- webapp | sed 's|^[^:]*:||' | sort) \
  <(git grep -nE '@ts-ignore|@ts-expect-error|@ts-nocheck' HEAD -- webapp | sed 's|^[^:]*:||' | sort)

echo '=== en.json ==='
git ls-tree -r HEAD --name-only | grep -E 'en\.json$' || echo 'HEAD: none'
git ls-tree -r master --name-only | grep -E 'en\.json$' || echo 'master: none'
```

All four extras lists must be **empty**. `en.json` must stay **none**.

Untracked W6.1 files (only if you added a gap test) will not show up in
`git grep HEAD`. Also scan the working tree:

```bash
rg -n "eslint-disable|FormattedMessage|@ts-ignore|@ts-expect-error|@ts-nocheck" \
  webapp/src/actions/rhs.ts \
  webapp/src/actions/rhs.test.ts \
  webapp/src/client/index.ts \
  webapp/src/client/index.test.ts \
  webapp/src/utils/rhs_*.ts \
  webapp/src/components/rhs \
  webapp/src/components/admin_console/rhs_status_setting \
  webapp/src/plugin.tsx \
  webapp/src/plugin.test.ts \
  webapp/src/selectors/index.ts \
  webapp/src/selectors/index.test.ts

rg -n "from 'react-intl'|from \"react-intl\"" \
  webapp/src/components/rhs \
  webapp/src/components/admin_console/rhs_status_setting \
  webapp/src/utils/rhs_*.ts \
  webapp/src/actions/rhs.ts \
  webapp/src/plugin.tsx
```

RHS/admin/helper production + test files: **no matches**.
`plugin.tsx` must not import `react-intl`.

```bash
find webapp -name en.json -not -path '*/node_modules/*'
```

Must print nothing.

### 5. Confirm Gate 8 admin line stayed ungated

```bash
rg -n "registerAdminConsoleCustomSetting" webapp/src/plugin.tsx
```

Exactly one call, argument `'RHSStatusTabs'`, **not** inside
`if (settings.ui_enabled)` or `if (shouldRegisterJiraRHS`.

```bash
rg -n "isPopoutWindow|WebappUtils.popouts|registerRHSPluginPopoutListener" webapp/src
```

Must be empty.

---

## File-by-file change list

| File | Action | What to do |
|------|--------|------------|
| Existing Gate 6–8 `*.test.ts(x)` | **Do not touch** if names are present | Re-run only |
| A gap-table file | Append **only** if that name is missing | Restore the original recipe |
| `webapp/src/components/rhs/**` production | **Do not touch** | |
| `webapp/src/components/admin_console/**` production | **Do not touch** | |
| `webapp/src/plugin.tsx` | **Do not touch** | |
| `webapp/src/actions/index.ts` | **Do not touch** | 48 pre-existing `error TS` |
| `webapp/tsconfig.json` / `.eslintrc.json` | **Do not touch** | |
| `en.json` | **Do not create** | |
| `server/**` | **Do not touch** | |

Expected diff for a clean W6.1: **empty** (verification only).

---

## Definition of Done

- [ ] Gap-table script prints `all Gate 9 matrix names present`
- [ ] **No** new test files unless a table row was actually missing
- [ ] `npm run lint` exit 0
- [ ] `npm run test` (full suite) pass, **≥ 193** tests
- [ ] `check-types` **delta** PASS: no new RHS-file diagnostics; no new
      `plugin.tsx` message; count **≤ 249** with `manifest.ts` present
- [ ] `tsc` exit 1 is **accepted**; do not “fix” the 249
- [ ] No new `eslint-disable` / `@ts-*` vs `master`
- [ ] No new `FormattedMessage` vs `master`
- [ ] No `en.json` added
- [ ] Admin `registerAdminConsoleCustomSetting('RHSStatusTabs', …)` still
      ungated
- [ ] No `isPopoutWindow` / `WebappUtils.popouts` /
      `registerRHSPluginPopoutListener`
- [ ] No production feature changes

---

## Commit checkpoint (orchestration, not this engineer)

Do not commit. The lead commits Step 9 after Gate 9 review. Suggested
message when they do: Close milestone 2 quality gates without adding
duplicate tests or adopting i18n. Do not push.

If W6.1 added nothing, the Step 9 commit may be **empty of webapp
code** (planning-only or skipped). That is correct.

---

## Implementation Summary

**Engineer:** IE11  
**Date:** 2026-08-19  
**Status:** Gate 9 PASS (verification only; no production or test code added)

HEAD stayed `d6fbfdd`. `webapp/src/manifest.ts` present. Working tree: only this plan file (`.planning/phase-w6/`).

### W6.1

- Gap script result: `all Gate 9 matrix names present` (all 15 names)
- Tests added: **none**

### W6.2

| Command | Result |
|---------|--------|
| `npm run lint` | exit 0 (`--quiet`; no printed errors) |
| `npm run test` | **27** suites / **193** tests passed (0 failed); 7.811 s |
| `check-types` delta | count=**249** (== master baseline); new RHS files=**0**; plugin.tsx extras=**0** (still the 6 pre-existing messages). `tsc` exit 1 accepted. |
| i18n / eslint-disable vs master | all four extra lists **empty**; `en.json` none on HEAD and master; working-tree RHS/admin/helper scan: no `eslint-disable` / `FormattedMessage` / `@ts-*` / `react-intl` |
| admin ungated | `registerAdminConsoleCustomSetting('RHSStatusTabs', …)` at `plugin.tsx:122`, **outside** `if (shouldRegisterJiraRHS…)` (`117–119`) and **outside** `if (settings.ui_enabled)` |
| popout APIs | no `isPopoutWindow` / `WebappUtils.popouts` / `registerRHSPluginPopoutListener` |
| optional `go test` (not required) | `server`: `go test -count=1 -timeout 60s .` **ok** (6.003 s). Did **not** run `make test`. |

`plugin.tsx` diagnostics (same 6 messages as master; line numbers drifted):

1. `(dispatch, getState) => Promise<any>` not assignable to `Action<object>`
2. `() => object` not assignable to `GlobalState`
3. `Property 'ui_enabled' does not exist on type 'Action<object>'`
4. `Cannot find name 'PluginRegistry'`
5. `'React' refers to a UMD global`
6. `(() => Promise<void>) | undefined` not assignable to `(...args: any[]) => any`

No new `error TS` in `src/actions/rhs*`, `src/utils/rhs_*`, `src/components/rhs/**`, `src/components/admin_console/rhs_status_setting/**`, `src/client/index.test.ts`, or `src/plugin.test.ts`.

### Blockers

- None. Do not commit / push (orchestration owns Step 9).
