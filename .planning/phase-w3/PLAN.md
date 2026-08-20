# Phase W3 Plan: The RHS panel

> Prescriptive implementation plan for **Phase W3 only** (tasks W3.1–W3.6 plus
> the component tests the parent DoD requires). An Implementation Engineer
> should be able to land this without making design decisions. Do not implement
> later phases from this file.
>
> **Do not write production TS/JS until this plan is followed as written.**
> This document is the plan; it is not the code.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase W3 (source of truth for WHAT)
- **Handoff:** `.planning/PHASE1_HANDOFF.md` (DTO, error codes, W-D4 tabs)
- **W2:** `.planning/phase-w2/PLAN.md` (actions/selectors/slices already landed)
- **Orchestration:** Step 8 / Gate 8 (W3 half) of `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `cb5ec3a` (W2 commit). Do not rebase; do not implement against an older SHA.
- **Package:** `webapp/` (Jest 29, TypeScript 5.7 strict, ESLint 8, RTL 14)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer.** Sequential file creation is fine; do not
  split W3 across two agents. W5 is the parallel workstream and owns
  `plugin.tsx` collision risk with W4 — W3 stays out of `plugin.tsx`.

## Scope

**In scope:**

- `webapp/src/components/rhs/` — panel tree (create)
- `webapp/src/selectors/index.ts` — RHS slice selectors (W2 deferred these)
- `webapp/src/selectors/index.test.ts` — add selector tests
- `webapp/src/actions/rhs.ts` — append `resolveAndFetchRHSIssues` only
- `webapp/src/actions/index.ts` — add that name to the existing `./rhs` re-export
- `webapp/src/actions/rhs.test.ts` — append resolve/invalid-tab tests
- `webapp/src/utils/rhs_resolve.ts` + test — instance pick + tab equality
- `webapp/src/utils/rhs_time.ts` + test — relative updated time (no moment)

**Out of scope (do not do these):**

- **`webapp/src/plugin.tsx`** — registration is W4. W5 also edits this file.
  W3 staying out is the Step 8 collision avoidance.
- System Console picker (`webapp/src/components/admin_console/`) — W5
- Popout detect / `/_popout/` / `isPopoutWindow` / `WebappUtils.popouts` — W4
- Changing W2 fetch helpers, in-flight map, or reducer bodies
- `en.json` / `FormattedMessage`
- `eslint-disable` comments
- Anything under `server/`
- Adding npm dependencies (`@atlaskit/icon-object`, `compass-icons`, `moment`,
  `@testing-library/user-event`)
- D2 virtual-seed picker behavior (W5)
- Committing or pushing

---

## Research (read this before coding)

Line numbers are as of **`cb5ec3a`**. If they have drifted, search; do not
invent a different design.

### 1. Forced decomposition — lint, not taste

`.eslintrc.json:190-197` `max-lines` is 650, counts comments, skips blanks.
`.eslintrc.json:481-485` `react/jsx-max-props-per-line` is **1**.
`.eslintrc.json:520-525` `react/no-multi-comp` is on with
`ignoreStateless: true` — **function** components may share a file; class
components may not.

Parent W3 assumes function components throughout. Do that. Do not write a
class. Do not `eslint-disable` `max-lines` or `no-multi-comp`.

Other rules that will fail a first draft (same set W2 listed):

| Rule | Setting | W3 impact |
|------|---------|-----------|
| `header/header` | `:130-135` | every new file, two-line copyright |
| `import/order` | `:138-152` | blank line between groups |
| `import-newlines/enforce` | `:154-157` **`[2, 3]`** | 1–3 names **single line**; 4+ multiline |
| `sort-imports` | `ignoreDeclarationSort: true` | sort names inside `{…}` (ASCII: `RHSErrorCode` before `RHS_DEFAULT_*`) |
| `object-curly-spacing` | `"never"` | `{data}` not `{ data }` |
| `react/jsx-no-literals` | `:495` error | no bare JSX text. Use `{'Loading'}` or a const |
| `react/jsx-tag-spacing` | `:499-505` `beforeSelfClosing: "never"` | `<Foo/>` not `<Foo />` |
| `react/jsx-closing-bracket-location` | `:452-456` `tag-aligned` | |
| `react/jsx-first-prop-new-line` | `:467-469` `multiline` | |
| `react/self-closing-comp` | `:553` | empty tags self-close |
| `no-undefined` | `:350` | `null`, not `undefined` |
| `no-underscore-dangle` | `:351` | no `_booting` |
| `lines-around-comment` | `:181-188` | blank line before `//` |
| `padded-blocks` | never | no blank line right after `{` |
| `arrow-parens` | always | `(issue) =>` |
| `space-before-function-paren` | `asyncArrow: always` | `async () =>` |
| `no-await-in-loop` | `:220` | the invalid-tab retry is **one** follow-up, not a loop |

Copyright header exactly:

```
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
```

**Do not add `en.json` or `FormattedMessage`.** i18n is milestone F1.
`formatjs/*` rules (`:638-641`) are inert unless `FormattedMessage` appears.

### 2. `jsx-no-literals` — the established pattern

`webapp/src/components/loading.tsx:13`:

```13:13:webapp/src/components/loading.tsx
                <h3 style={{margin: '20px 0'}}>{'Loading'}</h3>
```

`jira_ticket_tooltip.tsx:32-33, 206` uses module-level consts (`unAssignedLabel`,
`isAssignedLabel`) and `{'Check your connection or try again later'}`.
Attributes (`label='Refresh'`, `aria-label='Refresh'`, `className='…'`,
`target='_blank'`) are **not** literals under this rule.

W3 puts every user-visible string in `rhs_strings.ts` and interpolates
`{RHS_STRINGS.empty}`. Tests import the same const so copy cannot drift.

### 3. Component tree precedent — `connect()` + directory

The plugin’s shape is a directory per component, `index.ts` for `connect()`,
sibling presentational file. Canonical:

```13:30:webapp/src/components/jira_instance_and_project_selector/index.ts
const mapStateToProps = (state) => {
    const installedInstances = getInstalledInstances(state);
    const connectedInstances = getUserConnectedInstances(state);
    const defaultUserInstanceID = getDefaultUserInstanceID(state);

    return {
        installedInstances,
        connectedInstances,
        defaultUserInstanceID,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    fetchJiraProjectMetadata,
    getConnected,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(JiraInstanceAndProjectSelector);
```

`channel_subscriptions/index.ts:42-89` is the typed variant
(`state: GlobalState`). Follow **that** typing. `jira_ticket_tooltip/index.ts:8-9`
uses bare `'selectors'` / `'actions'` (webpack `resolve.modules` includes
`src`, `webpack.config.js:20-22`). Jest maps `^actions$` but **not**
`^selectors$` (`package.json:100-112`). **W3 tests import presentational
files relatively and do not import the connected default** — same as
`jira_ticket_tooltip.test.tsx:10` and
`jira_instance_and_project_selector.test.tsx:12`. The connected `index.ts`
may use bare `'actions'` / `'selectors'` like `channel_subscriptions/index.ts`.

There is **no** `useSelector` / `useDispatch` in this plugin. Do not start.

### 4. `getConnected` on open — and the stale-props trap

Parent W3.1: on mount, `dispatch(getConnected())` before deciding what to
render, following `jira_instance_and_project_selector.tsx:57-76`.

```57:87:webapp/src/components/jira_instance_and_project_selector/jira_instance_and_project_selector.tsx
    componentDidMount() {
        this.fetchInstances();
    }

    fetchInstances = async () => {
        if (this.props.selectedInstanceID) {
            this.props.onInstanceChange(this.props.selectedInstanceID);
            ...
            return;
        }

        this.setState({fetchingInstances: true});
        const {error} = await this.props.getConnected();
        this.setState({fetchingInstances: false});
        if (error) {
            this.props.onError(error.message);
            return;
        }

        let instanceID = '';
        if (this.props.connectedInstances.length === 1) {
            instanceID = this.props.connectedInstances[0].instance_id;
        } else if (this.props.defaultUserInstanceID) {
            instanceID = this.props.defaultUserInstanceID;
        }
```

**Do not copy the post-await `this.props.connectedInstances` read.** After
`await getConnected()` the class still has **stale** props; Redux has updated
but React has not re-rendered. The existing selector “works” only because
`defaultUserInstanceID` was usually already in the pre-mount store.

W3’s fetch step must read **`getState()` after both `getConnected` and
`restoreRHSViewState`**. That is `resolveAndFetchRHSIssues` (W3.0). The
component’s `useEffect` only sequences the three calls; it does not pick an
instance from props.

`getConnected` (`actions/index.ts:490-513`) hits `/api/v2/userinfo` and
dispatches `RECEIVED_CONNECTED` + `RECEIVED_INSTANCE_STATUS`. Return is
`{data}` or `{error}`. The TS type `GetConnectedResponse`
(`types/model.ts:284-295`) says `data.user.connected_instances`. The server
actually serializes **`user_info`** (`server/info.go:68-78`,
`server/user.go:172-177`):

```json
{"can_connect":true,"is_connected":true,"instances":[...],"user_info":{"connected_instances":[...],"default_instance_id":"..."}}
```

The reducer already reads `action.data.user_info` (`reducers/index.ts:62-76`).
**Do not “fix” `GetConnectedResponse`.** Do not parse `getConnected()`’s
return body for instances. After the await, `resolveAndFetchRHSIssues` reads
`getConnectedCloudInstances(getState())`.

### 5. W2 APIs the panel must call — `actions/rhs.ts`

Already landed. Do not change bodies except appending `resolveAndFetchRHSIssues`.

| Export | What W3 does with it |
|--------|----------------------|
| `restoreRHSViewState` (`:108-120`) | Mount, **before** first fetch. Returns `{data: saved \| null}`. Does **not** write localStorage. |
| `fetchRHSIssues(args)` (`:122-166`) | Page-1 replace. Dispatches `RHS_ISSUES_LOADING {reset: true}` which **clears** `rhsIssues`. |
| `loadMoreRHSIssues()` (`:168-209`) | Append. `RHS_ISSUES_LOADING {reset: false}` keeps rows. No-ops if `rhsIsLast` or empty token. |
| `refreshRHSIssues()` (`:211-219`) | `dispatch(fetchRHSIssues({current instance, tab, sort}))`. **Same in-flight key as load-more.** |
| `setRHSInstanceID` / `setRHSTab` / `setRHSSort` | **Do not call from the panel.** `fetchRHSIssues` already stamps view state + persists. Extra setter + fetch double-writes. |
| `openCreateModalWithoutPost` (`actions/index.ts:62-68`) | New ticket. `(description, channelId) => dispatch({type: OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST, data})`. |
| `handleConnectFlow` (`actions/index.ts:532-589`) | Not-connected Connect button. Existing production connect path. |
| `getConnected` | Mount only, via the sequence in W3.1. |

In-flight key (`rhs.ts:20-27`):

```20:27:webapp/src/actions/rhs.ts
export function rhsIssuesFlightKey(instanceID: string, tab: RHSTab, sort: RHSSort): string {
    return JSON.stringify({
        instance: instanceID,
        kind: tab.kind,
        key: tab.key || '',
        id: tab.id || '',
        sort,
    });
}
```

`name` is **not** in the key. Page 1, load-more, and refresh of the same
triple share **one** Map slot (`phase-w2/PLAN.md` Surprise B / `:1128-1129`).

### 6. RE7 carry-forward — refresh is not a guaranteed page-1 replace

`refreshRHSIssues` reuses `fetchRHSIssues`, therefore the **same**
`withRHSIssuesInFlight` slot as `loadMoreRHSIssues`.

If load-more is in flight and the user clicks Refresh:

1. `work()` does **not** run.
2. No `RHS_ISSUES_LOADING {reset: true}`.
3. The list is **not** cleared.
4. The returned Promise is the **load-more** Promise (append, not replace).

W3 **must not**:

- set local `refreshClicked` / `isRefreshing` and show the full-page loader
  because the button was pressed
- assume `rhsIssues` will become `[]` after Refresh
- dispatch `refreshRHSIssues` while `rhsLoading` is already true (disable the
  button instead — makes coalescing invisible)

Drive UI **only** from Redux:

| `rhsLoading` | `rhsIssues.length` | `rhsError` | Show |
|--------------|--------------------|------------|------|
| true | 0 | null | **loading** (page 1 in flight, including a refresh that actually started) |
| true | > 0 | null | **list + footer spinner** (load-more in flight, or a coalesced refresh) |
| false | 0 | null | **empty** |
| false | > 0 | null | **list** |
| false | 0 | `not_connected` / `rate_limited` / other | the matching full-panel state |
| false | > 0 | any | **list + footer error banner** (load-more failed; W2 does not clear rows on `RHS_ISSUES_ERROR`, `reducers/index.ts:380-390`) |

The GitHub plugin’s failure mode is `github_items.tsx:311`: when
`items.length === 0` it renders `{'You have no active items'}` — including
while the list is still in flight (the component has no loading prop). Our
empty copy is **never** `'Loading'` and the loading branch is checked
**before** `issues.length === 0`.

Disable Refresh and Load more when `booting || loading`. Tab / sort /
instance stay enabled (those change the flight key and **are** allowed to
issue a second request — W2 test
`fetchRHSIssues for a different tab while in flight does call fetch again`).

### 7. Selectors W2 left for W3

`selectors/index.ts` is 106 lines (plenty of headroom). W2 thunks read
`state[pluginStateKey]` directly. The panel uses selectors.

Already landed and **must** be reused:

```98:104:webapp/src/selectors/index.ts
export const getConnectedCloudInstances = (state: GlobalState): Instance[] => {
    return getUserConnectedInstances(state).filter(isCloudInstance);
};

export const instanceIsInstalled = (state: GlobalState): boolean => getInstalledInstances(state).length > 0;

export const getDefaultUserInstanceID = (state: GlobalState) => getPluginState(state).defaultUserInstanceID;
```

`getConnectedCloudInstances` is **connected ∩ installed ∩ Cloud** — the
picker list (W1.2 / parent W3.2). `hasCloudInstance` is **installed** Cloud
and is a W4 App Bar gate, not a W3 render gate.

`getPluginState` is file-private (`:14`). New selectors use it the same way.
Do not export it.

`getCurrentChannelId` lives in `mattermost-redux/selectors/entities/common`
(already imported by `actions/index.ts:5`). Map it in `index.ts` for New
ticket. `defaultMockState` has **no** `entities.users` / `entities.channels`
(`test-utils.tsx:43-49`). RHS tests that need a channel id pass
`initialState`; do **not** grow `defaultMockState` (W1 blast-radius lesson).

There is **no** `getTheme` usage in this plugin. Theme is a required prop on
modals (`create_issue_form.tsx:38`) but **no** `connect()` maps it — host
injection is inconsistent. Parent W3.6 allows `makeStyleFromTheme`
(`mattermost-redux/utils/theme_utils`, same module as `changeOpacity` in
`utils/styles.ts:4`). W3 does **not** need it: all chrome is SCSS +
`var(--center-channel-*)` like `ticketStyle.scss`. Do not thread `Theme`
through every child. Do not import `makeStyleFromTheme`.

### 8. Instance picker — do not reuse `JiraInstanceAndProjectSelector`

Research §6.1 (`research.md:521-533`): that component is modal-shaped
(`addValidate` / `removeValidate` / labeled `react-select`) and builds
options from **`installedInstances`** (`jira_instance_and_project_selector.tsx:131`)
while visibility uses `connectedInstances.length > 1` (`:137`). A user could
pick an instance they are not connected to.

RHS rules (parent W3.2 + spec):

- Options = `getConnectedCloudInstances` only
- Render the picker **only** when that list’s length is **> 1**
- Length 1: auto-select (via `resolveRHSInstanceID`), render **nothing**
- Length 0: do **not** fetch; show not-connected (App Bar still shows because
  `hasCloudInstance` is installed-not-connected — that is intentional)

Native `<select>`, not `ReactSelectSetting`. No new react-select theme work.

### 9. Styles, icons, time

**SCSS precedent to copy:** `components/jira_ticket_tooltip/ticketStyle.scss`
uses `var(--center-channel-color)`, `var(--center-channel-bg)`,
`var(--link-color)`, `rgba(var(--center-channel-color-rgb), 0.08)`
(`:9-11, :165`). Status pills there use hardcoded `#1C58D9` for
indeterminate (`:195-198`) — **do not copy that**. Parent W3.6: Mattermost
theme CSS variables only.

**SCSS precedent to avoid:** `channel_subscriptions_modal.scss` (layout only,
no theme vars, predates the convention).

Import pattern: `import './ticketStyle.scss'` (`jira_ticket_tooltip.tsx:13`).
Webpack `scss` rule is `style-loader` + `css-loader` + `sass-loader`
(`webpack.config.js:52-61`). Jest maps `^.+\\.(css|less|scss)$` to
`identity-obj-proxy` (`package.json:109`). Import `./rhs.scss` from `rhs.tsx`
only.

**Icons.** `plugin_constants/icons.tsx:6-15` has **one** icon
(`exclamationTriangle`). `icon.tsx` is the Jira logo (class component — do
not extend it). `svgWrapper/index.tsx:16-36` is the wrapper to reuse
(`viewBox` default `'0 0 16 16'`). Tooltip loading uses host Font Awesome
(`jira_ticket_tooltip.tsx:220` `fa fa-spin fa-spinner`). Mattermost ships FA;
chrome (refresh / plus) uses `fa fa-refresh` / `fa fa-plus`. **Do not** add
`compass-icons` or `@atlaskit/icon-object`.

DTO `RHSIssue.issueType` is a **string name only**
(`types/model.ts:231`, `server/rhs.go:118`, `server/rhs_types.go:117`).
There is **no** `hierarchyLevel`, **no** `subtask` bool, **no** `iconUrl`.
Spec/research fallbacks that mention `issuetype.hierarchyLevel` cannot be
implemented against this DTO. Map by lowercased name; `subtask` via
substring; default `task`. **Never** read a Jira `iconUrl`.

Priority is on the DTO (`priority: string`) but **not** on the W3.4 row.
Do not draw priority icons.

**Relative time.** No `moment` / `date-fns` in `package.json`. GitHub plugin
uses a local `formatTimeSince`. Write `formatRHSRelativeTime` (W3.0).
`updated` is RFC3339 or `""` (`PHASE1_HANDOFF.md` DTO table).

**Browse URL.** Use `issue.browseUrl` as the `<a href>`. Server built it from
`GetJiraBaseURL()` (`PHASE1_HANDOFF.md`). Reconstructing from `instance_id`
404s on `cloud-oauth` (`api.atlassian.com`).

### 10. Five states and error codes

Handoff codes (`PHASE1_HANDOFF.md`): `not_connected`, `rate_limited`,
`not_authorized`, `not_cloud`, `invalid_request`, `internal_error`.

Parent W3.5 five states:

| State | When | Copy (exact) |
|-------|------|--------------|
| loading | see §6 table | `Loading` (reuse `<Loading/>` from `components/loading.tsx` — already `{'Loading'}`) |
| empty | success, zero rows | `No tickets in this tab` |
| error | any code except the two below, and `issues.length === 0` | `Could not load tickets` + Try again |
| not connected | `rhsError === 'not_connected'` **or** zero connected Cloud instances after boot | `Connect your Jira account to see your tickets` + Connect Jira |
| rate limited | `rhsError === 'rate_limited'` | `Jira is rate limiting requests, try again shortly` (handoff stable message) + Try again |

Branch on **`rhsError`**, never `message` string-matching.

`not_authorized` / `not_cloud` / `invalid_request` / `internal_error` →
generic error. `not_cloud` should not happen if we only fetch Cloud ids;
still map it to generic.

Retry on error / rate-limited: `resolveAndFetchRHSIssues` (re-resolves
instance; safer than `refreshRHSIssues` when `rhsInstanceID` is `''`).

Connect button: `handleConnectFlow()` — `openConnectModal` only fires when
multiple installed instances need a picker (`actions/index.ts:582-587`);
single-instance goes straight to `redirectConnect`. That is the production
path. Do not invent a third connect flow.

### 11. Tab identity and W-D4

`RHSTab` is `{kind, name, key?, id?}` (`types/model.ts:213-218`). Assigned
is `{kind: 'assigned', name: 'Assigned'}`. Category needs `key`. Status
needs `id`. **Equality ignores `name`** (same as the debounce key). Server
returns resolved tabs with Assigned first and vanished entries already
hidden (`PHASE1_HANDOFF.md`, W-D4). Render **in array order**. Do not
re-sort. Do not re-derive from config.

If a persisted tab was removed by an admin, `/rhs/issues` returns
`invalid_request`. `resolveAndFetchRHSIssues` retries **once** with
`RHS_DEFAULT_TAB`. Not a loop (`no-await-in-loop`).

### 12. Tests, fetch mock, channel id

`tests/setup.js:6-11` — `global.fetch` returns `{ok: true, json: () => ({})}`.
`clearMocks: true` (history cleared, implementation kept). Connected children
that dispatch `getConnected` therefore “succeed” with an empty payload.

`renderWithRedux` (`test-utils.tsx:56-74`) uses **redux-mock-store** (does
**not** run reducers) + `IntlProvider` + `Provider`. Fine for presentational
tests. Do **not** use it to assert `rhsIssues` after dispatch.

`@testing-library/user-event` is **not** a dependency. Use `fireEvent` from
`@testing-library/react` (`edit_channel_subscription.test.tsx:7, 896`).
`@testing-library/jest-dom` is loaded in setup.

No Enzyme.

### 13. Host FA + webpack externals

`webpack.config.js:65-72` externalizes `react`, `react-dom`, `redux`,
`react-redux`, `react-bootstrap`, `react-router-dom`. Jest uses real
packages. Do not import `react-custom-scrollbars-2` (GitHub plugin
dependency; **we do not have it**). Native overflow on the list and tab
strip.

### 14. `actions/index.ts` headroom

Non-blank count at `cb5ec3a`: **603 / 650**. The re-export block is
`:694-705`. Adding `resolveAndFetchRHSIssues` is one name. **Do not** put
the thunk body in `index.ts`.

`actions/rhs.ts` is 231 lines. Appending one thunk fits.

Do **not** import `getConnected` from `./index` inside `rhs.ts` (cycle:
`index.ts` re-exports `./rhs` at EOF). `resolveAndFetchRHSIssues` only
reads `getState()`; the component calls `getConnected` then
`restoreRHSViewState` then `resolveAndFetchRHSIssues`.

### 15. Gate 8 (W3 half)

From `IMPL_ORCHESTRATION_PLAN.md` Step 8: **loading and empty are visually
distinct**. The other two Gate 8 items (registration off; picker `onChange`)
are W4 / W5. W3 owns the loading≠empty assertion by **test name**.

---

## Surprises

### Surprise A — DTO has no `hierarchyLevel` / `iconUrl`

Parent W3.4 and spec § Issue row mention `subtask` / `hierarchyLevel`
fallbacks. Milestone 1 stores only `issueType` string
(`server/rhs.go:118`). Map by name. Do not extend the DTO in W3.

### Surprise B — refresh and load-more share one in-flight slot (RE7)

See §6. UI is Redux-driven. Disable Refresh / Load more while `rhsLoading`.
No local “refresh clicked” loader.

### Surprise C — load-more error keeps the rows

`rhsError` is set; `rhsIssues` is unchanged (`reducers/index.ts:311-323,
380-390`). A full-panel error would hide tickets the user already has.
Footer banner when `issues.length > 0 && error`.

### Surprise D — first paint looks like empty unless `booting` exists

On first mount `rhsLoading` is `false`, `rhsIssues` is `[]`, `rhsError` is
`null` (`reducers` defaults / `defaultMockState:33-41`). That is the empty
state. Hold a local `booting` flag true until the mount sequence settles,
and treat `booting` like page-1 loading. This is **not** a refresh flag
(RE7). It flips false once and stays false.

### Surprise E — `GetConnectedResponse` lies; instance selector reads stale props

See §4. After `getConnected`, resolve from `getState()`, not props, not the
return body’s `user.connected_instances`.

### Surprise F — do not reuse the existing instance selector or add react-select

See §8. Native `<select>`. Options from connected Cloud only.

### Surprise G — no `moment`, no `getTheme` in this plugin, no compass-icons

Relative time is a 20-line helper. Styles are SCSS variables. Chrome icons
are Font Awesome classes the tooltip already uses.

### Surprise H — `defaultMockState` has two Cloud instances and no channel id

`test-utils.tsx:26-27` has `instance1` + `instance2`. A connected-container
test using the fixture will **show** the instance picker. Presentational
tests pass an explicit `connectedCloud` array. Do not shrink the fixture.
Do not add `entities.users` / `entities.channels` to `defaultMockState`.

### Surprise I — `check-types` is not clean

249 pre-existing `error TS` (W1/W2). Fail W3 only if a **new** file or a
W3-touched file appears in that list.

### Surprise J — GitHub RHS is a class, has no in-panel tabs, conflates empty/loading

`sidebar_right.jsx:66` class component. Tabs are left-sidebar icon buttons
(`research.md:49-52`). Empty = `'You have no active items'` at
`github_items.tsx:311`. Borrow Redux-held view state and
`target='_blank'` + `rel='noopener noreferrer'` on title links
(`github_items.tsx:94-103`). Do not borrow Scrollbars, octicons, or the
empty-as-loading pattern. `makeStyleFromTheme` is how GitHub themes
(`github_items.tsx:314`); we use SCSS instead (`ticketStyle.scss`).

---

## Tasks

### W3.0 — Helpers the panel needs

Do these first. No JSX.

#### `webapp/src/utils/rhs_resolve.ts` (create)

```typescript
import {Instance, RHSTab} from 'types/model';

export function rhsTabsEqual(a: RHSTab, b: RHSTab): boolean {
    return a.kind === b.kind &&
        (a.key || '') === (b.key || '') &&
        (a.id || '') === (b.id || '');
}

export function resolveRHSInstanceID(
    savedInstanceID: string,
    connectedCloud: Instance[],
    defaultUserInstanceID: string,
): string {
    if (savedInstanceID && connectedCloud.some((instance) => instance.instance_id === savedInstanceID)) {
        return savedInstanceID;
    }
    if (defaultUserInstanceID && connectedCloud.some((instance) => instance.instance_id === defaultUserInstanceID)) {
        return defaultUserInstanceID;
    }
    if (connectedCloud.length >= 1) {
        return connectedCloud[0].instance_id;
    }
    return '';
}
```

With 2+ connected Cloud and nothing saved: pick `[0]` so the list still
loads; the picker is visible and the user can switch. Do not leave
`instanceID` empty (that would be `invalid_request` or a blank panel).

#### `webapp/src/utils/rhs_time.ts` (create)

```typescript
const minuteMs = 60 * 1000;
const hourMs = 60 * minuteMs;
const dayMs = 24 * hourMs;
const weekMs = 7 * dayMs;

export function formatRHSRelativeTime(iso: string, nowMs: number): string {
    if (!iso) {
        return '';
    }
    const then = Date.parse(iso);
    if (Number.isNaN(then)) {
        return '';
    }
    const delta = Math.max(0, nowMs - then);
    if (delta < minuteMs) {
        return 'just now';
    }
    if (delta < hourMs) {
        return Math.floor(delta / minuteMs) + 'm ago';
    }
    if (delta < dayMs) {
        return Math.floor(delta / hourMs) + 'h ago';
    }
    if (delta < weekMs) {
        return Math.floor(delta / dayMs) + 'd ago';
    }
    return iso.slice(0, 10);
}
```

Pass `nowMs` from the caller (`Date.now()` in the row). Tests pass a frozen
number. No `locale` month names (i18n is F1). `no-magic-numbers` is a
**warning** and `--quiet` hides it; named consts still keep the file clear.

#### `webapp/src/components/rhs/rhs_strings.ts` (create)

```typescript
export const RHS_STRINGS = {
    loading: 'Loading',
    empty: 'No tickets in this tab',
    error: 'Could not load tickets',
    retry: 'Try again',
    notConnected: 'Connect your Jira account to see your tickets',
    connect: 'Connect Jira',
    rateLimited: 'Jira is rate limiting requests, try again shortly',
    refresh: 'Refresh',
    newTicket: 'New ticket',
    loadMore: 'Load more',
    sortLabel: 'Sort',
    sortUpdated: 'Updated',
    sortCreated: 'Created',
    instanceLabel: 'Jira instance',
    loadingMore: 'Loading more',
};
```

Every user-visible string in the panel comes from here.

#### Selectors — `webapp/src/selectors/index.ts`

Expand the `types/model` import. Three names today (`:11`) — adding RHS
types forces multiline, ASCII-sorted:

```typescript
import {
    Instance,
    InstanceType,
    PluginSettings,
    RHSErrorCode,
    RHSIssue,
    RHSSort,
    RHSTab,
    RHS_DEFAULT_SORT,
    RHS_DEFAULT_TAB,
} from 'types/model';
```

(`RHS_DEFAULT_*` after `RHSTab` — underscore sorts after letters. Match W2
Surprise / `reducers/index.ts`.)

Append after `getPluginSettings` (`:106`):

```typescript
export const getRHSInstanceID = (state: GlobalState): string => getPluginState(state).rhsInstanceID || '';

export const getRHSTab = (state: GlobalState): RHSTab => getPluginState(state).rhsTab || RHS_DEFAULT_TAB;

export const getRHSSort = (state: GlobalState): RHSSort => getPluginState(state).rhsSort || RHS_DEFAULT_SORT;

export const getRHSIssues = (state: GlobalState): RHSIssue[] => getPluginState(state).rhsIssues || [];

export const getRHSTabs = (state: GlobalState): RHSTab[] => getPluginState(state).rhsTabs || [];

export const getRHSIsLast = (state: GlobalState): boolean => Boolean(getPluginState(state).rhsIsLast);

export const getRHSLoading = (state: GlobalState): boolean => Boolean(getPluginState(state).rhsLoading);

export const getRHSError = (state: GlobalState): RHSErrorCode | null => getPluginState(state).rhsError || null;
```

Do **not** add `getRHSNextPageToken` unless a component reads it. Load-more
visibility is `!rhsIsLast` (the thunk already no-ops on empty token).

#### `resolveAndFetchRHSIssues` — append to `webapp/src/actions/rhs.ts`

Import `getConnectedCloudInstances` and `getDefaultUserInstanceID` from
`../selectors` (add to the existing `getPluginServerRoute` import — that
becomes 3 names, **single line**, ASCII:
`getConnectedCloudInstances, getDefaultUserInstanceID, getPluginServerRoute`).

Import `resolveRHSInstanceID` from `utils/rhs_resolve`.
Import `RHS_DEFAULT_TAB` into the existing model import (will go multiline;
ASCII-sort).

Optional overrides let tab / sort / instance clicks issue a page-1 fetch
without mapping `fetchRHSIssues` or the setters into the container. Empty
call = current resolved view (mount, retry, refresh).

```typescript
export type ResolveAndFetchArgs = {
    instanceID?: string;
    tab?: RHSTab;
    sort?: RHSSort;
};

export const resolveAndFetchRHSIssues = (overrides: ResolveAndFetchArgs = {}) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const state = getState();
        const plugin = state[pluginStateKey];
        const instanceID = overrides.instanceID || resolveRHSInstanceID(
            plugin.rhsInstanceID,
            getConnectedCloudInstances(state),
            getDefaultUserInstanceID(state) || '',
        );
        if (!instanceID) {
            return Promise.resolve({data: null});
        }

        const tab = overrides.tab || (plugin.rhsTab as RHSTab);
        const sort = overrides.sort || (plugin.rhsSort as RHSSort);

        return dispatch(fetchRHSIssues({instanceID, tab, sort}) as any).then((result: {data?: RHSIssuesResponse; error?: RHSFetchError}) => {
            if (
                result &&
                result.error &&
                result.error.errorCode === 'invalid_request' &&
                tab.kind !== 'assigned'
            ) {
                return dispatch(fetchRHSIssues({
                    instanceID,
                    tab: RHS_DEFAULT_TAB,
                    sort,
                }) as any);
            }
            return result;
        });
    };
};
```

`consistent-return`: both paths return Promises. The `.then` retry is one
follow-up, not a loop.

Add `resolveAndFetchRHSIssues` and the `ResolveAndFetchArgs` type to the
`actions/index.ts` re-export list, still alphabetized
(`loadMoreRHSIssues`, `refreshRHSIssues`, `resetRHSIssuesInFlight`,
`resolveAndFetchRHSIssues`, `restoreRHSViewState`, …). Type-only export:

```typescript
export type {ResolveAndFetchArgs} from './rhs';
```

Keep it adjacent to the value re-export. One extra line; headroom is 47.

### W3.1 — Panel shell and container

**Files:** `webapp/src/components/rhs/index.ts`, `webapp/src/components/rhs/rhs.tsx`
**Action:** Create

#### `index.ts` — `connect()` only

```typescript
import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';

import {
    getConnected,
    handleConnectFlow,
    loadMoreRHSIssues,
    openCreateModalWithoutPost,
    resolveAndFetchRHSIssues,
    restoreRHSViewState,
} from 'actions';
import {
    getConnectedCloudInstances,
    getDefaultUserInstanceID,
    getRHSError,
    getRHSInstanceID,
    getRHSIsLast,
    getRHSIssues,
    getRHSLoading,
    getRHSSort,
    getRHSTab,
    getRHSTabs,
} from 'selectors';

import {GlobalState} from 'types/store';

import RHS from './rhs';

const mapStateToProps = (state: GlobalState) => {
    return {
        connectedCloud: getConnectedCloudInstances(state),
        defaultUserInstanceID: getDefaultUserInstanceID(state) || '',
        channelId: getCurrentChannelId(state) || '',
        instanceID: getRHSInstanceID(state),
        tab: getRHSTab(state),
        sort: getRHSSort(state),
        issues: getRHSIssues(state),
        tabs: getRHSTabs(state),
        isLast: getRHSIsLast(state),
        loading: getRHSLoading(state),
        error: getRHSError(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    getConnected,
    handleConnectFlow,
    loadMoreRHSIssues,
    openCreateModalWithoutPost,
    resolveAndFetchRHSIssues,
    restoreRHSViewState,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(RHS);
```

`import-newlines`: 6 names from `actions` and 10 from `selectors` →
multiline, ASCII-sorted as shown. Do not import `fetchRHSIssues` /
`refreshRHSIssues` / setters here.

W4 will `import RHS from 'components/rhs'`. That default is this connected
component. W3 does not register it.

#### `rhs.tsx` — function component, owns boot + composition, no row chrome

Props (export the type for tests):

```typescript
export type Props = {
    connectedCloud: Instance[];
    defaultUserInstanceID: string;
    channelId: string;
    instanceID: string;
    tab: RHSTab;
    sort: RHSSort;
    issues: RHSIssue[];
    tabs: RHSTab[];
    isLast: boolean;
    loading: boolean;
    error: RHSErrorCode | null;
    getConnected: () => Promise<{data?: unknown; error?: unknown}>;
    restoreRHSViewState: () => {data: RHSViewState | null};
    resolveAndFetchRHSIssues: (overrides?: ResolveAndFetchArgs) => Promise<{data?: unknown; error?: unknown}>;
    loadMoreRHSIssues: () => Promise<unknown>;
    openCreateModalWithoutPost: (description: string, channelId: string) => void;
    handleConnectFlow: () => void;
};
```

Import `ResolveAndFetchArgs` from `actions` (or `../../actions/rhs` in
tests if the barrel type export is awkward). Re-export the type from
`actions/index.ts` next to the thunk if `import/named` complains.

Mount sequence — `useEffect` with `[]` deps, cancelled flag, **this order**:

1. `setBooting(true)` (initial state is already `true`)
2. `await getConnected()`
3. if cancelled, return
4. `restoreRHSViewState()` (sync)
5. if cancelled, return
6. `await resolveAndFetchRHSIssues()`
7. if !cancelled, `setBooting(false)`
8. cleanup sets `cancelled = true`

`react-hooks/exhaustive-deps` will warn about missing props in `[]`. That
is a **warning**; `--quiet` hides it. Keep `[]` — this is boot-once, like
`componentDidMount`. Do not re-boot when `connectedCloud` changes (websocket
`connect` / `instance_status` already refresh Redux; a remount is W4
popout).

`import './rhs.scss'`.

Body render — **this order, no extra branches**:

```
const showPage1Loading = booting || (loading && issues.length === 0 && error === null);
const showNotConnected = !showPage1Loading && (connectedCloud.length === 0 || error === 'not_connected');
const showRateLimited = !showPage1Loading && !showNotConnected && error === 'rate_limited' && issues.length === 0;
const showError = !showPage1Loading && !showNotConnected && !showRateLimited && error !== null && issues.length === 0;
const showEmpty = !showPage1Loading && !showNotConnected && !showError && !showRateLimited && issues.length === 0;
```

`error !== null` is required (`no-undefined`).

Shell:

```tsx
<div
    className='jira-rhs'
    data-testid='jira-rhs'
>
    <RHSHeader .../>
    {tabs.length > 0 && (
        <RHSTabStrip
            tabs={tabs}
            selectedTab={tab}
            onSelect={onSelectTab}
        />
    )}
    {body}
</div>
```

Header always renders (New ticket must work even on error / not-connected).
Tab strip hidden when `tabs` is empty (first-load error / not-connected).

Handlers (in `rhs.tsx`). Overrides are defined on `resolveAndFetchRHSIssues`
in W3.0 — do not also map `fetchRHSIssues` / `refreshRHSIssues` / setters.

- tab click → if `rhsTabsEqual(next, tab)` return; else
  `resolveAndFetchRHSIssues({tab: next})`
- sort change → `resolveAndFetchRHSIssues({sort: next})`
- instance change → `resolveAndFetchRHSIssues({instanceID: next, tab: RHS_DEFAULT_TAB})`
  (parent W3.2: switching instance reloads tabs and resets the list; reset
  tab to Assigned because the previous tab may not exist on the new instance)
- retry → `resolveAndFetchRHSIssues()`
- load more → `loadMoreRHSIssues()` only if `!loading && !isLast`
- refresh → `resolveAndFetchRHSIssues()` only if `!loading` (same current
  view; we already disable while loading so RE7 coalescing should not trigger)
- new ticket → `openCreateModalWithoutPost('', channelId)`
- connect → `handleConnectFlow()`

Calling `resolveAndFetchRHSIssues()` without overrides is the page-1 refetch
of the resolved current view.

`onSelectTab` / `onSortChange` / `onInstanceChange` may run while loading
(different flight key). That is allowed.

### W3.2 — Instance picker and tab strip

**Files:** `rhs_instance_picker.tsx`, `rhs_tab_strip.tsx`
**Action:** Create

#### `RHSInstancePicker`

```typescript
export type Props = {
    instances: Instance[];
    selectedInstanceID: string;
    onChange: (instanceID: string) => void;
};
```

If `instances.length <= 1` return `null`.

Otherwise a native select:

```tsx
<label className='jira-rhs-instance'>
    <span className='jira-rhs-instance-label'>{RHS_STRINGS.instanceLabel}</span>
    <select
        aria-label={RHS_STRINGS.instanceLabel}
        data-testid='rhs-instance-picker'
        value={selectedInstanceID}
        onChange={(event) => onChange(event.target.value)}
    >
        {instances.map((instance) => (
            <option
                key={instance.instance_id}
                value={instance.instance_id}
            >
                {instance.alias || instance.instance_id}
            </option>
        ))}
    </select>
</label>
```

Option text is an expression (`{instance.alias || instance.instance_id}`),
not a JSX literal. `alias` is optional on `Instance` (`types/model.ts:190`).

`rhs.tsx` passes `connectedCloud` and hides the picker by not mounting it
when `connectedCloud.length <= 1` **or** by letting this component return
null — do **both** (parent DoD: “absent with one”). Passing `[]` / one
element must render nothing.

#### `RHSTabStrip`

```typescript
export type Props = {
    tabs: RHSTab[];
    selectedTab: RHSTab;
    onSelect: (tab: RHSTab) => void;
};
```

```tsx
<div
    className='jira-rhs-tab-strip-wrap'
    data-testid='rhs-tab-strip'
>
    <div
        className='jira-rhs-tab-strip'
        role='tablist'
    >
        {tabs.map((item) => {
            const selected = rhsTabsEqual(item, selectedTab);
            return (
                <button
                    key={item.kind + ':' + (item.key || '') + ':' + (item.id || '')}
                    className={selected ? 'jira-rhs-tab jira-rhs-tab--selected' : 'jira-rhs-tab'}
                    role='tab'
                    aria-selected={selected}
                    type='button'
                    onClick={() => onSelect(item)}
                >
                    {item.name}
                </button>
            );
        })}
    </div>
</div>
```

Key is identity, not `name`. Horizontal scroll is CSS (`overflow-x: auto`).
The wrap’s `::after` gradient is the **visible scroll affordance** (parent
W3.2). Do not cap tab count.

### W3.3 — Header: sort, refresh, New ticket

**File:** `rhs_header.tsx`
**Action:** Create

```typescript
export type Props = {
    sort: RHSSort;
    loading: boolean;
    onSortChange: (sort: RHSSort) => void;
    onRefresh: () => void;
    onNewTicket: () => void;
    instancePicker: React.ReactNode;
};
```

`instancePicker` is `null` or `<RHSInstancePicker .../>` — keeps
`rhs_header` from importing the picker and keeps `rhs.tsx` as the composer.

```tsx
<div
    className='jira-rhs-header'
    data-testid='rhs-header'
>
    {instancePicker}
    <div className='jira-rhs-header-actions'>
        <label className='jira-rhs-sort'>
            <span className='jira-rhs-sort-label'>{RHS_STRINGS.sortLabel}</span>
            <select
                aria-label={RHS_STRINGS.sortLabel}
                data-testid='rhs-sort'
                value={sort}
                onChange={(event) => onSortChange(event.target.value as RHSSort)}
            >
                <option value='updated'>{RHS_STRINGS.sortUpdated}</option>
                <option value='created'>{RHS_STRINGS.sortCreated}</option>
            </select>
        </label>
        <button
            type='button'
            className='jira-rhs-icon-button'
            aria-label={RHS_STRINGS.refresh}
            data-testid='rhs-refresh'
            disabled={loading}
            onClick={onRefresh}
        >
            <i
                className='fa fa-refresh'
                title={RHS_STRINGS.refresh}
            />
        </button>
        <button
            type='button'
            className='btn btn-primary jira-rhs-new-ticket'
            data-testid='rhs-new-ticket'
            onClick={onNewTicket}
        >
            <i className='fa fa-plus'/>
            <span>{RHS_STRINGS.newTicket}</span>
        </button>
    </div>
</div>
```

`disabled={loading}` is the RE7 mitigation. `rhs.tsx` passes
`loading={booting || loading}` into the header so boot also disables
refresh.

New ticket is never disabled.

Sort `<select>` stays enabled while loading (new flight key). Changing sort
calls `resolveAndFetchRHSIssues({sort})` which is page 1 (W2 DoD).

### W3.4 — Issue row + type icon + list

**Files:** `rhs_issue_type_icon.tsx`, `rhs_issue_row.tsx`, `rhs_issue_list.tsx`
**Action:** Create

#### Type icon

```typescript
export type RHSIssueTypeIconName = 'bug' | 'story' | 'epic' | 'subtask' | 'incident' | 'task';

export function rhsIssueTypeIconName(issueType: string): RHSIssueTypeIconName {
    const name = issueType.trim().toLowerCase();
    switch (name) {
    case 'bug':
    case 'fault':
        return 'bug';
    case 'story':
        return 'story';
    case 'epic':
        return 'epic';
    case 'sub-task':
    case 'subtask':
        return 'subtask';
    case 'incident':
        return 'incident';
    case 'task':
        return 'task';
    default:
        if (name.indexOf('sub-task') !== -1 || name.indexOf('subtask') !== -1) {
            return 'subtask';
        }
        return 'task';
    }
}
```

Use `indexOf`, not `includes`, if you prefer older lib targets — `includes`
is fine on this TS target. Exhaustive `switch` + `default`. No `never`
helper (`no-underscore-dangle` / `no-void` fought that in W1).

Render via `SVGWrapper` (`width={14}` `height={14}` `viewBox='0 0 16 16'`
`fill='currentColor'` `className={'jira-rhs-type-icon jira-rhs-type-icon--' + name}`).
Paths (original geometry — do **not** add `@atlaskit/icon-object`; do **not**
fetch Jira `iconUrl`):

| Name | Path `d` |
|------|----------|
| bug | `M8 1a3 3 0 0 1 3 3v1h1.5a.5.5 0 0 1 0 1H11v1h1.5a.5.5 0 0 1 0 1H11v1a3 3 0 0 1-6 0V8H3.5a.5.5 0 0 1 0-1H5V6H3.5a.5.5 0 0 1 0-1H5V4a3 3 0 0 1 3-3z` |
| story | `M4 2h8a1 1 0 0 1 1 1v11l-5-3-5 3V3a1 1 0 0 1 1-1z` |
| epic | `M9 1L3 9h4l-1 6 7-9H9l1-5z` |
| task | `M3 3h10v10H3V3zm1.5 5.2l2.2 2.3 4.8-5` (use two paths: rect + check `M5 8.2l2 2 4-4`) |
| subtask | same check, smaller: wrap with `transform='translate(3 3) scale(0.7)'` on a `<g>` |
| incident | `M8 2l6 11H2L8 2zm0 3.5v4M8 11.5h.01` |

`aria-hidden='true'` on the svg (via `className` + decorative). The row’s
link text is the accessible name.

#### Row

```typescript
export type Props = {
    issue: RHSIssue;
    nowMs: number;
};
```

```tsx
<article
    className='jira-rhs-issue'
    data-testid={'rhs-issue-' + issue.key}
>
    <div className='jira-rhs-issue-line1'>
        <RHSIssueTypeIcon issueType={issue.issueType}/>
        <a
            className='jira-rhs-issue-key'
            href={issue.browseUrl}
            target='_blank'
            rel='noopener noreferrer'
        >
            {issue.key}
        </a>
        <a
            className='jira-rhs-issue-summary'
            href={issue.browseUrl}
            target='_blank'
            rel='noopener noreferrer'
            title={issue.summary}
        >
            {issue.summary}
        </a>
    </div>
    <div className='jira-rhs-issue-line2'>
        <span className={statusClass(issue.status.categoryKey)}>
            {issue.status.name}
        </span>
        <span className='jira-rhs-issue-dot'>{'·'}</span>
        <span className='jira-rhs-issue-project'>{issue.project}</span>
        <span className='jira-rhs-issue-dot'>{'·'}</span>
        <span className='jira-rhs-issue-time'>{formatRHSRelativeTime(issue.updated, nowMs)}</span>
    </div>
</article>
```

`statusClass`:

```typescript
export function rhsStatusClass(categoryKey: string): string {
    switch (categoryKey) {
    case 'new':
    case 'indeterminate':
    case 'done':
    case 'undefined':
        return 'jira-rhs-status jira-rhs-status--' + categoryKey;
    default:
        return 'jira-rhs-status jira-rhs-status--default';
    }
}
```

Truncation is **CSS** `-webkit-line-clamp: 2` on `.jira-rhs-issue-summary`
(parent W3.4 / GitHub unbounded-wrap anti-pattern). Do not slice the string
(the tooltip does that at 80 chars — we keep full text for `title=` and
accessibility).

Do not render `priority`, `assignee`, `labels`, `dueDate`.

#### List

```typescript
export type Props = {
    issues: RHSIssue[];
    loading: boolean;
    isLast: boolean;
    error: RHSErrorCode | null;
    nowMs: number;
    onLoadMore: () => void;
    onRetry: () => void;
};
```

```tsx
<div
    className='jira-rhs-list'
    data-testid='rhs-issue-list'
>
    {issues.map((issue) => (
        <RHSIssueRow
            key={issue.key}
            issue={issue}
            nowMs={nowMs}
        />
    ))}
    {loading && (
        <div
            className='jira-rhs-list-loading'
            data-testid='rhs-list-loading'
        >
            <span className='fa fa-spin fa-spinner'/>
            <span>{RHS_STRINGS.loadingMore}</span>
        </div>
    )}
    {error !== null && !loading && (
        <div
            className='jira-rhs-list-error'
            data-testid='rhs-list-error'
        >
            <span>{error === 'rate_limited' ? RHS_STRINGS.rateLimited : RHS_STRINGS.error}</span>
            <button
                type='button'
                onClick={onRetry}
            >
                {RHS_STRINGS.retry}
            </button>
        </div>
    )}
    {!isLast && !loading && (
        <button
            type='button'
            className='jira-rhs-load-more'
            data-testid='rhs-load-more'
            onClick={onLoadMore}
        >
            {RHS_STRINGS.loadMore}
        </button>
    )}
</div>
```

`rhs.tsx` only mounts `RHSIssueList` when `issues.length > 0`. Footer
spinner = load-more in flight (or coalesced refresh — list stays). Load
more hidden while `loading` (RE7).

`nowMs`: `rhs.tsx` captures `const nowMs = Date.now()` during render (not
in an effect). Good enough for a 20-row page; tests pass a fixed `nowMs`
into `RHSIssueRow` / `RHSIssueList` directly.

### W3.5 — The five states

**File:** `rhs_states.tsx`
**Action:** Create

Five **function** components in one file (`ignoreStateless: true`). No
classes.

```typescript
export function RHSLoadingState(): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid='rhs-state-loading'
        >
            <Loading/>
        </div>
    );
}

export function RHSEmptyState(): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid='rhs-state-empty'
        >
            <p>{RHS_STRINGS.empty}</p>
        </div>
    );
}

export function RHSErrorState(props: {onRetry: () => void}): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid='rhs-state-error'
        >
            <p>{RHS_STRINGS.error}</p>
            <button
                type='button'
                onClick={props.onRetry}
            >
                {RHS_STRINGS.retry}
            </button>
        </div>
    );
}

export function RHSNotConnectedState(props: {onConnect: () => void}): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid='rhs-state-not-connected'
        >
            <p>{RHS_STRINGS.notConnected}</p>
            <button
                type='button'
                className='btn btn-primary'
                data-testid='rhs-connect'
                onClick={props.onConnect}
            >
                {RHS_STRINGS.connect}
            </button>
        </div>
    );
}

export function RHSRateLimitedState(props: {onRetry: () => void}): JSX.Element {
    return (
        <div
            className='jira-rhs-state'
            data-testid='rhs-state-rate-limited'
        >
            <p>{RHS_STRINGS.rateLimited}</p>
            <button
                type='button'
                onClick={props.onRetry}
            >
                {RHS_STRINGS.retry}
            </button>
        </div>
    );
}
```

`Loading` already renders `{RHS_STRINGS.loading}`-equivalent `'Loading'`.
**Do not** also put `RHS_STRINGS.empty` inside `RHSLoadingState`.
`data-testid` values are the Gate 8 handles.

Empty vs loading assertion: `getByTestId('rhs-state-loading')` is in the
document XOR `getByTestId('rhs-state-empty')`. They never mount together.
Empty’s text is `No tickets in this tab`; loading’s text is `Loading`.

### W3.6 — Styles

**File:** `webapp/src/components/rhs/rhs.scss`
**Action:** Create

Root `.jira-rhs`: `display: flex; flex-direction: column; height: 100%;`
`color: var(--center-channel-color);` `background: var(--center-channel-bg);`

Required classes (match the JSX above):

- `.jira-rhs-header`, `.jira-rhs-header-actions` — row, gap, wrap
- `.jira-rhs-tab-strip-wrap` — `position: relative`
- `.jira-rhs-tab-strip-wrap::after` — 24px `linear-gradient(to right, transparent, var(--center-channel-bg))`, `pointer-events: none` (scroll hint)
- `.jira-rhs-tab-strip` — `display: flex; overflow-x: auto; scrollbar-width: thin;`
- `.jira-rhs-tab` / `--selected` — selected uses `var(--button-bg)` underline or `border-bottom`
- `.jira-rhs-issue-summary` — `-webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; display: -webkit-box; word-break: break-word;`
- `.jira-rhs-issue-key` / summary links — `color: var(--link-color); text-decoration: none;`
- `.jira-rhs-status--new` — `color: var(--away-indicator); background: rgba(var(--center-channel-color-rgb), 0.08);`
- `.jira-rhs-status--indeterminate` — `color: var(--button-bg);` same faint bg
- `.jira-rhs-status--done` — `color: var(--online-indicator);`
- `.jira-rhs-status--undefined`, `--default` — `color: var(--center-channel-color); background: rgba(var(--center-channel-color-rgb), 0.08);`
- `.jira-rhs-list` — `overflow-y: auto; flex: 1;`
- `.jira-rhs-state` — centered, padded

**No hex colors.** No `#1C58D9`. No `#FFFFFF` on pills (those fail in dark
theme — the tooltip’s mistake).

Do not create `styles.ts` helpers. Do not use `makeStyleFromTheme`.

---

## Tests

Copyright header, relative imports, no `eslint-disable`, no `en.json`, no
`user-event`. Prefer `screen` + `fireEvent` + `data-testid`.

Shared fixture (copy into the test files that need an issue; do not add
`webapp/src/testdata`):

```typescript
const assignedTab: RHSTab = {kind: 'assigned', name: 'Assigned'};
const inProgressTab: RHSTab = {kind: 'category', name: 'In Progress', key: 'indeterminate'};

const issueOne: RHSIssue = {
    key: 'TES-1',
    summary: 'One',
    browseUrl: 'https://example.atlassian.net/browse/TES-1',
    status: {name: 'In Progress', categoryKey: 'indeterminate'},
    priority: 'Medium',
    issueType: 'Task',
    project: 'TES',
    assignee: 'alice',
    reporter: 'bob',
    created: '2026-01-01T00:00:00Z',
    updated: '2026-01-02T00:00:00Z',
    dueDate: '',
    labels: [],
};
```

### `webapp/src/utils/rhs_resolve.test.ts` (create)

1. `rhsTabsEqual is true for the same category key and ignores name`
2. `rhsTabsEqual is false when status id differs`
3. `resolveRHSInstanceID prefers a saved id that is still connected`
4. `resolveRHSInstanceID falls back to defaultUserInstanceID when saved is gone`
5. `resolveRHSInstanceID picks the first connected Cloud instance when nothing saved`
6. `resolveRHSInstanceID returns empty string when there are no connected Cloud instances`

### `webapp/src/utils/rhs_time.test.ts` (create)

Freeze `nowMs = Date.parse('2026-01-02T01:00:00Z')`.

1. `formatRHSRelativeTime returns empty string for empty or invalid input`
2. `formatRHSRelativeTime returns just now under one minute`
3. `formatRHSRelativeTime returns Nm ago under one hour`
4. `formatRHSRelativeTime returns Nh ago under one day`
5. `formatRHSRelativeTime returns an ISO date after one week`

### `webapp/src/selectors/index.test.ts` (extend)

Add imports for the new selectors (multiline, ASCII). Tests:

1. `getRHSLoading and getRHSIssues keep loading distinguishable from empty` —
   `makeState` plugin `{rhsLoading: true, rhsIssues: [], rhsError: null}` →
   `getRHSLoading` true and `getRHSIssues` `[]`. This is the selector-level
   lock; the component test is the Gate 8 visual lock.
2. `getRHSError returns the typed code` — `rhsError: 'not_connected'` →
   `'not_connected'`.

### `webapp/src/actions/rhs.test.ts` (extend)

Keep existing Gate 7 names. Append:

1. `resolveAndFetchRHSIssues does not fetch when no connected Cloud instance` —
   store with empty connected/installed; call count stays 0.
2. `resolveAndFetchRHSIssues retries Assigned after invalid_request on a vanished tab` —
   first fetch 400 JSON `{error: 'invalid_request', message: '…'}` for a
   category tab; second fetch 200 Assigned page. `fetchMock` called **2**
   times; after settle `rhsTab.kind === 'assigned'` and `rhsError === null`.
3. `resolveAndFetchRHSIssues does not retry Assigned when the first error is invalid_request on Assigned` —
   call count **1**; `rhsError === 'invalid_request'`.

Reuse `makeRHSStore`. For (1) the default reducer state has no
`userConnectedInstances` — `getConnectedCloudInstances` is `[]`. Good.

### `webapp/src/components/rhs/rhs.test.tsx` (create) — Gate 8 lives here

Import presentational `RHS` from `./rhs`, **not** `./index`. Helper
`renderRHS(overrides)` fills no-op jest fns for all callbacks.

**Names are load-bearing for Gate 8 review:**

1. **`loading state is not the empty state`** — Gate 8. Props:
   `booting={false}`, `loading={true}`, `issues={[]}`, `error={null}`,
   `connectedCloud={[one Cloud instance]}`.
   `getByTestId('rhs-state-loading')` present.
   `queryByTestId('rhs-state-empty')` **null**.
   `queryByText(RHS_STRINGS.empty)` **null**.
   `getByText('Loading')` present (from `<Loading/>`).
2. **`empty state renders after a successful zero-row fetch`** —
   `loading={false}`, `issues={[]}`, `error={null}`, one connected Cloud.
   `getByTestId('rhs-state-empty')` present.
   `queryByTestId('rhs-state-loading')` null.
   `getByText(RHS_STRINGS.empty)` present.
3. **`error state renders for internal_error`** — `error='internal_error'`,
   `issues={[]}`, `loading={false}`. `getByTestId('rhs-state-error')`.
   Click Try again → `resolveAndFetchRHSIssues` called.
4. **`not_connected error renders the connect state`** — `error='not_connected'`.
   `getByTestId('rhs-state-not-connected')`. Click Connect →
   `handleConnectFlow` called. **Not** `openConnectModal`.
5. **`zero connected Cloud instances renders not-connected without fetching`**
   — `connectedCloud={[]}`, `loading={false}`, `error={null}`,
   `issues={[]}`. Same testid. Do not expect `resolveAndFetchRHSIssues` from
   this render (no click).
6. **`rate_limited renders the distinct rate-limit state`** —
   `getByTestId('rhs-state-rate-limited')` and
   `getByText(RHS_STRINGS.rateLimited)`. Must **not** be
   `rhs-state-error`.
7. **`instance picker is absent with one connected Cloud instance`** —
   `connectedCloud` length 1. `queryByTestId('rhs-instance-picker')` null.
8. **`instance picker is present with two connected Cloud instances`** —
   length 2. `getByTestId('rhs-instance-picker')` present.
9. **`changing sort dispatches resolveAndFetchRHSIssues with created`** —
   `fireEvent.change` on `rhs-sort` to `'created'`.
   `resolveAndFetchRHSIssues` called with `{sort: 'created'}`.
10. **`selecting a tab dispatches resolveAndFetchRHSIssues with that tab`** —
    `tabs={[assignedTab, inProgressTab]}`, `tab={assignedTab}`. Click
    In Progress. Called with `{tab: inProgressTab}`.
11. **`load more is shown when not last and click calls loadMoreRHSIssues`** —
    `issues={[issueOne]}`, `isLast={false}`, `loading={false}`. Click
    `rhs-load-more`. `loadMoreRHSIssues` called.
12. **`refresh is disabled while loading so a click is not treated as a page-1 replace`**
    — RE7. `issues={[issueOne]}`, `loading={true}`. `rhs-refresh` has
    `disabled`. `queryByTestId('rhs-state-loading')` **null** (list is
    showing). `getByTestId('rhs-issue-list')` present.
    `getByTestId('rhs-list-loading')` present.
    `queryByTestId('rhs-load-more')` null (hidden while loading).
    `fireEvent.click` on refresh does **not** increment
    `resolveAndFetchRHSIssues` (disabled buttons still fire in JSDOM —
    **assert `disabled` and do not click**, or click and expect 0 calls if
    the handler is not bound when disabled). Safest: `expect(refresh).toBeDisabled()`
    and do not click.
13. **`new ticket dispatches openCreateModalWithoutPost with empty description and the channel id`**
    — `channelId='channel-1'`. Click `rhs-new-ticket`.
    `openCreateModalWithoutPost` called with `('', 'channel-1')`.
14. **`on mount getConnected runs before restoreRHSViewState before resolveAndFetchRHSIssues`**
    — mock fns that push to an `order` array. `await waitFor`.
    `order` equals `['connected', 'restore', 'fetch']`. Use
    `booting={true}` initially — the component owns booting internally, so
    this test just mounts with default mocks and waits. Do **not** pass
    `booting` as a prop if it is internal state (it is internal). The
    `Props` type above does **not** include `booting` — it is `useState`
    inside `rhs.tsx`.

For (14), make `getConnected` async (`jest.fn(async () => { order.push('connected'); return {}; })`)
so the sequence is observable.

15. **`booting shows loading not empty`** — default mount, do **not**
    resolve `getConnected` (leave it hanging). `getByTestId('rhs-state-loading')`.
    `queryByTestId('rhs-state-empty')` null.

### `webapp/src/components/rhs/rhs_issue_row.test.tsx` (create)

1. **`row links use the DTO browseUrl and do not reconstruct a host`** —
   `href` on key and summary is
   `https://example.atlassian.net/browse/TES-1`. The rendered HTML does
   **not** contain `api.atlassian.com`.
2. **`long summary keeps full text and uses the clamp class`** — summary of
   400 chars; `getByText` finds it (or `expect(summaryEl).toHaveClass('jira-rhs-issue-summary')`
   and `expect(summaryEl.textContent).toHaveLength(400)`).
3. **`status pill class follows statusCategory key`** —
   `categoryKey: 'done'` → class contains `jira-rhs-status--done`.

### `webapp/src/components/rhs/rhs_issue_type_icon.test.tsx` (create)

1. `rhsIssueTypeIconName maps Bug to bug`
2. `rhsIssueTypeIconName maps Sub-task and custom subtask names to subtask`
3. `rhsIssueTypeIconName maps an unknown type to task`
4. Render `<RHSIssueTypeIcon issueType='Bug'/>` — container class includes
   `jira-rhs-type-icon--bug`.

### Existing suites

Run the **full** Jest suite. W3 must not change W1/W2 behavior. Do not
update snapshots that do not exist. If a fixture snapshot of
`'plugins-jira'` appears (none today), include the already-present RHS
keys — do not drop them.

W2 baseline: **17 suites / 119 tests**. W3 adds suites listed above.
Missing Gate 8 names fail the phase. Extra tests are fine.

---

## File-by-file change list (current line numbers)

Line numbers as of `cb5ec3a`.

| File | Action | Current lines | What to do |
|------|--------|---------------|------------|
| `webapp/src/selectors/index.ts` | Extend | EOF `:106`; model import `:11`; `getConnectedCloudInstances` `:98-100` | Expand model import; append 8 getters |
| `webapp/src/selectors/index.test.ts` | Extend | 10 existing tests | Add loading≠empty + error-code selector tests |
| `webapp/src/actions/rhs.ts` | Extend | EOF `:231`; `refreshRHSIssues` `:211-219`; flight key `:20-27` | Append `resolveAndFetchRHSIssues`. **Do not change** in-flight map or existing thunks |
| `webapp/src/actions/index.ts` | Re-export | export list `:694-705`; **603 / 650** non-blank | Add `resolveAndFetchRHSIssues` to the list only |
| `webapp/src/actions/rhs.test.ts` | Extend | Gate 7 names stay | Append 3 resolve/retry tests |
| `webapp/src/utils/rhs_resolve.ts` | Create | n/a | `rhsTabsEqual`, `resolveRHSInstanceID` |
| `webapp/src/utils/rhs_resolve.test.ts` | Create | n/a | equality + instance fallbacks |
| `webapp/src/utils/rhs_time.ts` | Create | n/a | `formatRHSRelativeTime` |
| `webapp/src/utils/rhs_time.test.ts` | Create | n/a | buckets + empty/invalid |
| `webapp/src/components/rhs/rhs_strings.ts` | Create | n/a | all user-visible copy |
| `webapp/src/components/rhs/index.ts` | Create | n/a | `connect()` only |
| `webapp/src/components/rhs/rhs.tsx` | Create | n/a | boot sequence + compose; import `./rhs.scss` |
| `webapp/src/components/rhs/rhs_header.tsx` | Create | n/a | sort, refresh, New ticket |
| `webapp/src/components/rhs/rhs_instance_picker.tsx` | Create | n/a | native select; null if ≤1 |
| `webapp/src/components/rhs/rhs_tab_strip.tsx` | Create | n/a | server order; identity key |
| `webapp/src/components/rhs/rhs_issue_type_icon.tsx` | Create | n/a | name map + local SVG |
| `webapp/src/components/rhs/rhs_issue_row.tsx` | Create | n/a | two-line row; `browseUrl` |
| `webapp/src/components/rhs/rhs_issue_list.tsx` | Create | n/a | rows + load more + footer states |
| `webapp/src/components/rhs/rhs_states.tsx` | Create | n/a | five function components |
| `webapp/src/components/rhs/rhs.scss` | Create | n/a | CSS variables; clamp; scroll hint |
| `webapp/src/components/rhs/rhs.test.tsx` | Create | n/a | **Gate 8** + picker/tab/sort/load-more/new-ticket/RE7 |
| `webapp/src/components/rhs/rhs_issue_row.test.tsx` | Create | n/a | browseUrl, clamp class, pill |
| `webapp/src/components/rhs/rhs_issue_type_icon.test.tsx` | Create | n/a | name map |
| `webapp/src/plugin.tsx` | **Do not touch** | `ui_enabled` `:50` | W4 + W5 |
| `webapp/src/client/index.ts` | **Do not touch** | | W2 done |
| `webapp/src/utils/rhs_view_state.ts` | **Do not touch** | | W2 done; W3 only *calls* `restoreRHSViewState` |
| `webapp/src/components/admin_console/**` | **Do not touch** | | W5 |
| `server/**` | **Do not touch** | | Milestone 1 |

Every new file stays under 650 non-blank lines. If `rhs.tsx` or
`rhs.test.tsx` approaches 650, split tests, not production, first.
`rhs.tsx` should stay under ~250.

---

## Commands

From `webapp/` (`node_modules` is already installed). Do not `npm install`.

```bash
cd webapp
npm run lint
npm run test
npm run check-types
```

- `lint` is `eslint … --quiet --cache`. Errors must be zero. No
  `eslint-disable`. Confirm no file reports `max-lines`.
- `test` is the **full** Jest suite when declaring W3 done. Filtering
  `rhs.test.tsx` while iterating is fine.
- `check-types`: still **249** pre-existing errors. Fail only if a W3 file
  appears.

Do not run `make test` (that also runs Go). W3 does not touch Go.

Gate 8 / RE7 manual checks:

```bash
rg -n "eslint-disable" webapp/src/components/rhs webapp/src/utils/rhs_resolve.ts webapp/src/utils/rhs_time.ts
rg -n "FormattedMessage|en.json" webapp/src/components/rhs
rg -n "isPopoutWindow|WebappUtils.popouts|/_popout/" webapp/src/components/rhs webapp/src/actions/rhs.ts
rg -n "plugin.tsx" webapp/src/components/rhs
git diff cb5ec3a -- webapp/src/plugin.tsx
```

First four must be empty / no matches. `plugin.tsx` diff must be empty.

```bash
rg -n "You have no active items|No tickets in this tab" webapp/src/components/rhs
```

Empty copy is only `No tickets in this tab`. Loading copy is only `Loading`
(from `<Loading/>`). They must not share a string.

---

## Definition of Done

- [ ] All five states render distinctly; **Gate 8** test
      `loading state is not the empty state` proves loading ≠ empty by
      testid **and** by text
- [ ] Zero connected Cloud after boot → not-connected, **no** issues fetch
- [ ] Instance picker absent with 1 connected Cloud, present with 2
- [ ] Tab click / sort change call `resolveAndFetchRHSIssues` with the
      override (sort is page 1 because that thunk uses `fetchRHSIssues`)
- [ ] Load more visible when `!isLast && !loading`; click calls
      `loadMoreRHSIssues`
- [ ] Refresh disabled while `loading`; list + footer spinner, not
      full-page loading, when `loading && issues.length > 0` (RE7)
- [ ] New ticket → `openCreateModalWithoutPost('', channelId)`
- [ ] Connect → `handleConnectFlow()`
- [ ] Row `href` is `browseUrl`; long summary uses the clamp class
- [ ] Type icons from local SVG name map; no `iconUrl`; no new npm deps
- [ ] Mount order: `getConnected` → `restoreRHSViewState` →
      `resolveAndFetchRHSIssues`
- [ ] Vanished persisted tab: one `invalid_request` retry as Assigned
- [ ] Function components only; no class; no `eslint-disable`; no file
      over 650 non-blank lines
- [ ] No `en.json` / `FormattedMessage`
- [ ] **`webapp/src/plugin.tsx` byte-identical to `cb5ec3a`**
- [ ] No popout pathname / `isPopoutWindow` code
- [ ] `npm run lint` and `npm run test` (full suite) pass
- [ ] `check-types` adds no new-file errors

---

## Commit checkpoint (orchestration, not this engineer)

Do not commit. The lead commits the W3 workstream after review (Step 8,
alongside W5). Suggested message when they do: Add the Jira tickets RHS
panel with distinct loading and empty states. Do not push.

W4 starts from that commit and is the first phase allowed to edit
`plugin.tsx` for App Bar / RHS registration. W3 leaves a default export at
`components/rhs` for W4 to import. No extra popout hook is required — a
popout is a new mount and runs the same boot sequence.

---

## Implementation Summary

**Engineer:** IE8 (new; not PE8 / IE1–IE7)
**Started from:** `cb5ec3a`
**Commit/push:** none
**W4 / W5 implementation:** none

### What landed

RHS panel tree under `webapp/src/components/rhs/`, plus W3.0 helpers and the
deferred selectors / `resolveAndFetchRHSIssues` thunk.

Mount sequence is `getConnected` → `restoreRHSViewState` →
`resolveAndFetchRHSIssues`. A local `booting` flag keeps first paint on the
loading state. Refresh and Load more are disabled/hidden while `rhsLoading`.
UI is Redux-driven (no refresh-clicked loader). Load-more errors stay in the
footer; existing rows are kept. Instance picker is a native `<select>` over
`getConnectedCloudInstances` only.

### Files created

- `webapp/src/utils/rhs_resolve.ts` + `rhs_resolve.test.ts`
- `webapp/src/utils/rhs_time.ts` + `rhs_time.test.ts`
- `webapp/src/components/rhs/rhs_strings.ts`
- `webapp/src/components/rhs/index.ts` (connected default for W4)
- `webapp/src/components/rhs/rhs.tsx`
- `webapp/src/components/rhs/rhs_header.tsx`
- `webapp/src/components/rhs/rhs_instance_picker.tsx`
- `webapp/src/components/rhs/rhs_tab_strip.tsx`
- `webapp/src/components/rhs/rhs_issue_type_icon.tsx` + test
- `webapp/src/components/rhs/rhs_issue_row.tsx` + test
- `webapp/src/components/rhs/rhs_issue_list.tsx`
- `webapp/src/components/rhs/rhs_states.tsx`
- `webapp/src/components/rhs/rhs.scss`
- `webapp/src/components/rhs/rhs.test.tsx` (Gate 8 names)

### Files extended

- `webapp/src/selectors/index.ts` — 8 RHS getters
- `webapp/src/selectors/index.test.ts` — loading≠empty + typed error
- `webapp/src/actions/rhs.ts` — `resolveAndFetchRHSIssues` only (W2 thunks unchanged)
- `webapp/src/actions/index.ts` — re-export + `type ResolveAndFetchArgs`
- `webapp/src/actions/rhs.test.ts` — 3 resolve/retry tests

### Test results

W3-related and pre-existing suites excluding parallel W5:

```
Test Suites: 22 passed, 22 total
Tests:       157 passed, 157 total
```

Gate 8 `loading state is not the empty state` passed (testid **and** text).
All 15 named panel tests passed, including RE7 refresh-disabled, boot order,
and `booting shows loading not empty`.

W2 Gate 7 names unchanged and still passing.

### `plugin.tsx`

**W3 did not edit `webapp/src/plugin.tsx`.**
`git diff cb5ec3a -- webapp/src/plugin.tsx` is **not** empty in this worktree:
W5 already added `RHSStatusSetting` import +
`registerAdminConsoleCustomSetting('RHSStatusTabs', ...)`. Those two lines
are not from this engineer and were not reverted.

### Lint / types

- W3 paths: `eslint --quiet` clean. No `eslint-disable`. No file over 650
  non-blank lines (`rhs.tsx` 215, `rhs.test.tsx` 278).
- W3 files add **zero** `error TS` lines.
- Repo-wide `npm run lint` and full `npm run test` currently fail on
  **untracked W5** `webapp/src/components/admin_console/**` (lint + one test
  `only Cloud instances are offered`). Not a W3 change.
- `check-types` count is 254 (249 baseline + W5 admin files). No W3 file in
  that list.

### Gate 8 rg checks (W3 trees)

- No `eslint-disable` under `components/rhs` or the new utils
- No `FormattedMessage` / `en.json`
- No popout / `isPopoutWindow` / `WebappUtils.popouts`
- Empty copy is only `No tickets in this tab`; loading is `<Loading/>` /
  `Loading`. They do not share a string.

### Deviations (lint / tsc only; no design change)

1. Presentational component is `Rhs`, not `RHS`.
   `react/jsx-pascal-case` rejects all-caps JSX tags. File path and connected
   default export remain `components/rhs`.
2. `ResolveAndFetchArgs` is re-exported as `type ResolveAndFetchArgs` inside
   the existing `export {…} from './rhs'` block (`no-duplicate-imports`).
3. `connect(...)(Rhs as any)` — `bindActionCreators` + untyped thunk
   `Dispatch` makes `getConnected()` look like a thunk, not a `Promise`.
4. `instancePicker?` is optional so babel-plugin-typescript-to-proptypes does
   not mark it required when the composer passes `null`.
5. `mapDispatchToProps` types `dispatch: Dispatch`; `defaultUserInstanceID`
   is coerced `as string`. Needed so the new `index.ts` has no tsc errors.

### Blockers

- Parallel W5 files in this worktree make repo-wide `npm run lint` and
  unfiltered `npm run test` fail. W3-only lint/tests are green.
- `plugin.tsx` is no longer byte-identical to `cb5ec3a` because of W5, not W3.

W4/W5 were not started. No commit. No push.
