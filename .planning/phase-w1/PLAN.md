# Phase W1 Plan: Types, selectors, and fixtures

> Prescriptive implementation plan for **Phase W1 only** (tasks W1.1–W1.3). An
> Implementation Engineer should be able to land this without making design
> decisions. Do not implement later phases from this file.
>
> **Do not write production TS/JS until this plan is followed as written.**
> This document is the plan; it is not the code.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase W1 (source of truth for WHAT)
- **Handoff:** `.planning/PHASE1_HANDOFF.md` (API contract; **D2 is W5, not W1**)
- **Orchestration:** Gate 6 / Step 6 of `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `46fb3c5` (M2 plan promotion). Milestone 1 code through `bb5e8d1`.
- **Package:** `webapp/` (Jest 29, TypeScript 5.7 strict, ESLint 8)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer.** Sequential. Nothing in this phase is
  parallelizable.

## Scope

**In scope:**

- `webapp/src/types/model.ts` — add `InstanceType.CLOUD_OAUTH`; add `PluginSettings`
- `webapp/src/selectors/index.ts` — `isCloudInstance`, `hasCloudInstance`,
  `getConnectedCloudInstances`; type `getPluginSettings`
- `webapp/src/selectors/index.test.ts` — **create** (Gate 6 lives here)
- `webapp/src/reducers/index.ts` — type the `pluginSettings` reducer only (no
  new slices)
- `webapp/src/testlib/test-utils.tsx` — rename fixture key; add `cloud-oauth`
  instance
- `webapp/src/components/modals/channel_subscriptions/edit_channel_subscription.test.tsx`
  — rename the same wrong key in `editChannelMockState`

**Out of scope (do not do these):**

- Any RHS UI (`webapp/src/components/rhs/`)
- Admin status picker (`webapp/src/components/admin_console/`)
- `webapp/src/plugin.tsx` registration / App Bar / popout
- `webapp/src/actions/index.ts` (see Surprise A — there is no switch to fix)
- `webapp/src/client/index.ts` JSON-error helper (W2)
- RHS Redux slices, thunks, `rhs_view_state` (W2)
- RHS DTO / `RHSErrorCode` types (W2; the parent File Change Map bundled these
  onto `model.ts` under W1, but W1 tasks do not include them)
- Anything under `server/`
- `en.json` / `FormattedMessage`
- `eslint-disable` comments
- D2 virtual-seed picker behavior (W5 / Gate 8)
- Committing or pushing

---

## Research (read this before coding)

Line numbers are as of **`46fb3c5`**. If they have drifted, search; do not
invent a different design.

### 1. `InstanceType` is CLOUD + SERVER only — `webapp/src/types/model.ts:184-192`

```184:192:webapp/src/types/model.ts
export enum InstanceType {
    CLOUD = 'cloud',
    SERVER = 'server',
}
export type Instance = {
    alias?: string;
    instance_id: string;
    type: InstanceType;
}
```

The server serializes three types (`server/instance.go:14-18`):

```14:18:server/instance.go
const (
	CloudInstanceType      = InstanceType("cloud")
	ServerInstanceType     = InstanceType("server")
	CloudOAuthInstanceType = InstanceType("cloud-oauth")
)
```

`AsConfigMap` writes `"type": string(ic.Type)` (`server/instance.go:56-61`), so
the webapp receives `'cloud-oauth'` for every OAuth2 Cloud install. The
server-side predicate to **copy** is `IsCloudInstance` (`server/instance.go:72-74`):

```72:74:server/instance.go
func (ic InstanceCommon) IsCloudInstance() bool {
	return ic.Type == CloudInstanceType || ic.Type == CloudOAuthInstanceType
}
```

W1 adds the missing enum member and a webapp helper with the same truth table.

### 2. There are **zero** production `type === InstanceType.CLOUD` checks

A full-webapp search for `InstanceType` in `webapp/src` finds:

| File | Line | What it is |
|------|------|------------|
| `types/model.ts` | 184-191 | enum + `Instance.type` |
| `actions/index.ts` | 20, **576** | import; **`=== InstanceType.SERVER`** desktop-connect guard |
| `testlib/test-utils.tsx` | 11, 26-27 | fixtures |
| five `*.test.tsx` files | various | test data only |

**There is no `switch` on instance type anywhere in the webapp.** Parent
`.planning/PLAN.md` W1.1 claimed adding `CLOUD_OAUTH` would break a switch at
`actions/index.ts:576`. That line is:

```576:580:webapp/src/actions/index.ts
        if (instance && instance.type === InstanceType.SERVER && isDesktopApp() && !isMinimumDesktopAppVersion(4, 3, 0)) { // eslint-disable-line no-magic-numbers
            const errMsg = 'Your version of the Mattermost desktop client does not support authenticating between Jira and Mattermost directly. To connect your Jira account with Mattermost, please go to Mattermost via your web browser and type `/jira connect`, or [check the Mattermost download page](https://mattermost.com/download/#mattermostApps) to get the latest version of the desktop client.';
            dispatch(sendEphemeralPost(errMsg));
            return;
        }
```

That `if` is SERVER-only on purpose. `cloud-oauth` must take the same path as
`cloud` (fall through to `redirectConnect`). **Do not convert this to a switch.
Do not touch `actions/index.ts` in W1.** Adding the enum member does **not**
fail `tsc` on this file.

The Cloud-gating bug is therefore **latent**: nothing in production currently
asks “is this Cloud?”. W4 will gate App Bar registration on `hasCloudInstance`.
If W1 shipped a helper that only checked `InstanceType.CLOUD`, W4 would
reproduce the exact production miss (OAuth2 Cloud-only installs). That is why
Gate 6 requires a **`cloud-oauth`-only** selector test, not a `cloud` test.

### 3. Selectors today — `webapp/src/selectors/index.ts`

`getUserConnectedInstances` (`:65-73`) already cross-filters
`userConnectedInstances` against `installedInstances` by `instance_id`. Keep
that behavior. `getConnectedCloudInstances` **must** filter that result, not
raw `userConnectedInstances`, so an uninstalled Cloud instance still self-heals
out of the RHS picker list.

```65:80:webapp/src/selectors/index.ts
export const getUserConnectedInstances = (state: GlobalState): Instance[] => {
    const installed = getPluginState(state).installedInstances as Instance[];
    const connected = getPluginState(state).userConnectedInstances as Instance[];
    if (!installed || !connected) {
        return [];
    }

    return connected.filter((instance1) => installed.find((instance2) => instance1.instance_id === instance2.instance_id));
};

export const getInstalledInstances = (state: GlobalState): Instance[] => getPluginState(state).installedInstances;
export const instanceIsInstalled = (state: GlobalState): boolean => getInstalledInstances(state).length > 0;

export const getDefaultUserInstanceID = (state: GlobalState) => getPluginState(state).defaultUserInstanceID;

export const getPluginSettings = (state: GlobalState) => getPluginState(state).pluginSettings;
```

`hasCloudInstance` looks at **installed** instances (App Bar gate = “a Cloud
instance is configured”), not connected ones. A user who has not connected
still needs the icon so the RHS can show `not_connected`. Spec: “at least one
Jira Cloud instance configured.”

`getConnectedCloudInstances` looks at **connected ∩ installed ∩ Cloud** — that
is the RHS instance picker list (W3.2).

No selector test file exists. Create `webapp/src/selectors/index.test.ts`.

### 4. Fixture key bug — `connectedInstances` vs `userConnectedInstances`

Reducer slice name is `userConnectedInstances` (`webapp/src/reducers/index.ts:62-72`,
exported at `:272`):

```62:72:webapp/src/reducers/index.ts
function userConnectedInstances(state = [], action = {} as AnyAction) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data.user_info) {
            return action.data.user_info.connected_instances ? action.data.user_info.connected_instances : [];
        }
        return state;
    default:
        return state;
    }
}
```

`defaultMockState` (`webapp/src/testlib/test-utils.tsx:24-41`) writes the wrong
key:

```24:33:webapp/src/testlib/test-utils.tsx
export const defaultMockState = {
    'plugins-jira': {
        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
        connectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}],
        defaultUserInstanceID: 'instance1',
        jiraProjectMetadata: null,
        jiraIssueMetadata: null,
        channelIdWithSettingsOpen: null,
        channelSubscriptions: {},
    },
```

`getUserConnectedInstances` therefore returns `[]` in every test that feeds
this object to a mock store. `defaultUserInstanceID` **is** keyed correctly, so
code that falls back to the default instance id still “works” while the
connected list is empty.

The same wrong key is copied in
`edit_channel_subscription.test.tsx:32-35`.

`context.md` claimed exactly one consuming test
(`channel_subscription_filter.test.tsx`). That is stale. Parent W1.3 already
widened it; this plan widens it again (see § Existing tests).

### 5. `ui_enabled` / `rhs_enabled` — type now, do not gate now

Server already returns both on `GET /api/v2/settingsinfo`
(`server/user.go:219-229`):

```221:228:server/user.go
	return respondJSON(w, struct {
		UIEnabled                              bool `json:"ui_enabled"`
		RHSEnabled                             bool `json:"rhs_enabled"`
		SecurityLevelEmptyForJiraSubscriptions bool `json:"security_level_empty_for_jira_subscriptions"`
	}{
		UIEnabled:                              conf.EnableJiraUI,
		RHSEnabled:                             conf.EnableJiraRHS,
		SecurityLevelEmptyForJiraSubscriptions: conf.SecurityLevelEmptyForJiraSubscriptions,
	})
```

Webapp consumption today:

- `getSettings()` (`actions/index.ts:469-487`) stores `action.data` under
  `RECEIVED_PLUGIN_SETTINGS`. Return value is untyped.
- `plugin.tsx:50` gates UI on `settings.ui_enabled` from that dispatch result.
  **Do not add `rhs_enabled` gating here in W1.**
- `hooks.ts:49-55` reads `pluginSettings.ui_enabled` (or `this.settings.ui_enabled`).
- `channel_subscriptions/index.ts:56-57` reads
  `pluginSettings?.security_level_empty_for_jira_subscriptions`.
- Reducer `pluginSettings` (`reducers/index.ts:74-81`) is `state = null` with
  no type.

W1 adds a `PluginSettings` type matching those three JSON keys and annotates
the reducer + `getPluginSettings`. W4 reads `rhs_enabled`. W1 does not.

### 6. ESLint constraints that apply even to types-only work

From `webapp/.eslintrc.json`:

| Rule | Setting | W1 impact |
|------|---------|-----------|
| `header/header` | `:130-135`, two-line copyright | every new file |
| `import/order` | `:138-152`, newlines between groups | new test file |
| `import-newlines/enforce` | `:154-157`, wrap at **3+** named imports | selector test imports |
| `sort-imports` | `:582-587`, `ignoreDeclarationSort: true` | sort names inside `{…}` |
| `object-curly-spacing` | `:384-387`, `"never"` | `{Instance, InstanceType}` not `{ Instance }` |
| `quotes` | `:427-430`, single | |
| `comma-dangle` | `:66-69`, always-multiline | |
| `@typescript-eslint/indent` | `:665-670`, 4 spaces, **`SwitchCase: 0`** | cases line up with `switch` |
| `react/jsx-no-literals` | `:495`, error | W1 is types/selectors — **no JSX**. Do not add strings inside JSX. |
| `max-lines` | `:190-197`, 650, counts comments | new files are tiny |
| `react/no-multi-comp` | `:520-525` | N/A (no components) |
| `@typescript-eslint/explicit-function-return-type` | `:662`, **warning** | annotate new exports anyway |
| `no-underscore-dangle` | `:350` | do not name anything `_exhaustive` |
| `no-void` | `:376` | do not `void` a `never` |
| `no-undefined` | `:350` | write `if (!installed)`, not `=== undefined` |

Match existing selector style: `export const name = (state: GlobalState): T =>`.
Switch indent must match `reducers/index.ts:24-29` (`SwitchCase: 0`).

Do **not** add `en.json` or `FormattedMessage`.

### 7. `node_modules` exists

Confirmed on this worktree: `webapp/node_modules` is present (Phase 5 `npm
install` / `make test`). `webapp/node_modules/.bin/jest` exists. Do not run
`npm install` unless a command fails with a missing binary.

Jest is run from `webapp/` (`package.json:13`:
`jest --forceExit --detectOpenHandles --verbose`). There is **no**
`moduleNameMapper` entry for `^selectors$`. New tests in
`webapp/src/selectors/index.test.ts` must import from `./index` (relative),
not from `'selectors'`.

`webapp/tests/setup.js:6-11` mocks `global.fetch` as `{ok: true, json: () => ({})}`.
Connected children that dispatch real thunks (`getConnected`,
`fetchJiraProjectMetadata`) therefore **succeed with empty payloads**. That
matters for the fixture blast radius below.

### 8. Gate 6 (must be in this phase)

From `IMPL_ORCHESTRATION_PLAN.md` Step 6:

1. A selector test covers the **`cloud-oauth`-only** installation (the case
   broken today). A test that only uses `InstanceType.CLOUD` does **not**
   satisfy the gate.
2. Run the **full** existing Jest suite, not just new tests.

---

## Surprise A — parent W1.1 `actions/index.ts` switch does not exist

**Do not edit `webapp/src/actions/index.ts`.** Put the exhaustive Cloud check
in `isCloudInstance` (W1.2) instead. `tsc` will not fail when the enum grows
because there is no switch over `InstanceType` in actions.

A `default: never` helper is also a poor fit here: `no-underscore-dangle` and
`no-void` fight the usual `never` pattern. Use a `switch` with an explicit
`SERVER` → `false` and `default` → `false` (fail-closed for unknown future
types). Lock the truth table in tests so `CLOUD_OAUTH` cannot regress to
`false`.

---

## Surprise B — fixture blast radius is nine suites, not four

`renderWithRedux` (`test-utils.tsx:47-49`) defaults `initialState` to
`defaultMockState`. These files call it **without** overriding that state:

| File | Component under test | Reads `getUserConnectedInstances` via Redux? |
|------|----------------------|-----------------------------------------------|
| `channel_subscription_filter.test.tsx` | presentational `ChannelSubscriptionFilter`; nests **connected** `JiraEpicSelector` (`mapStateToProps` is `null`) | No (dispatch-only connect) |
| `jira_epic_selector.test.tsx` | presentational `./jira_epic_selector` | No |
| `connect_modal_form.test.tsx` | presentational; passes `connectedInstances` as **props** | No |
| `disconnect_modal_form.test.tsx` | presentational; `connectedInstances: []` as **props** | No |
| `create_issue_form.test.tsx` | presentational `CreateIssueForm` nests **connected** `JiraInstanceAndProjectSelector` (`create_issue_form.tsx:29,279`) | **Yes** |
| `jira_instance_and_project_selector.test.tsx` | presentational; instances as **props** | No |
| `jira_ticket_tooltip.test.tsx` | presentational; instances as **props** | No |
| `channel_subscription_filters.test.tsx` | presentational | No |
| `channel_subscriptions.test.tsx` | presentational `./channel_subscriptions` | No |

Plus the local clone:

| File | Notes |
|------|-------|
| `edit_channel_subscription.test.tsx:32-35` | own `editChannelMockState` with the same wrong key; nests the **connected** selector (`edit_channel_subscription.tsx:13,585`) |

Most suites will be behavior-unchanged because they pass instances as props.
The two that can change:

**`create_issue_form.test.tsx` (highest risk).** Nested connected selector
(`jira_instance_and_project_selector.tsx:57-88`) calls real `getConnected`
(fetch mock succeeds), then:

- Today: `userConnectedInstances` missing → connected list `[]`;
  `defaultUserInstanceID` is `'instance1'` → still auto-selects `'instance1'`;
  picker hidden (`length > 1` is false).
- After key rename + **two** connected Cloud instances (this plan): connected
  length is 2; auto-select still `'instance1'` via `defaultUserInstanceID`;
  **instance picker becomes visible** (`:137`). Existing assertions are
  `ref.current` / `create()` call counts, not picker absence. Expect them to
  still pass. If one fails, **do not shrink `defaultMockState`**. Pass a
  one-instance `initialState` into that test only.

**`edit_channel_subscription.test.tsx`.** `editChannelMockState` has no
`defaultUserInstanceID`. After the key rename, nested selector sees length 1
and auto-selects `https://something.atlassian.net`. Create-subscription tests
(`selectedSubscription: null`) then `setState(baseState)` which sets the same
`instanceID` (`:137-140, :359`). Expect them to still pass. If a create test
fails because `onInstanceChange` fired before `setState`, wait for the
existing `act` / set the local mock’s `defaultUserInstanceID` to that URL —
do not revert the key name.

Parent W1.3 listed only four `renderWithRedux` heirs. Re-run **all 12**
existing suites (Phase 5: 13 suites / 83 tests including `manifest.test.tsx`
and `jira_issue_metadata.test.tsx`, which do not use the fixture).

---

## Tasks

### W1.1 — Add `CLOUD_OAUTH` and `PluginSettings` to `model.ts`

**File:** `webapp/src/types/model.ts`
**Action:** Modify

Replace the enum at `:184-187` with:

```typescript
export enum InstanceType {
    CLOUD = 'cloud',
    CLOUD_OAUTH = 'cloud-oauth',
    SERVER = 'server',
}
```

Keep `CLOUD_OAUTH` between `CLOUD` and `SERVER`. Value **must** be the string
`'cloud-oauth'` (server `CloudOAuthInstanceType`).

Immediately after the existing `Instance` type (`:188-192`), add:

```typescript
export type PluginSettings = {
    ui_enabled: boolean;
    rhs_enabled: boolean;
    security_level_empty_for_jira_subscriptions: boolean;
};
```

JSON names must match `server/user.go:222-224`. Do not add RHS issue DTOs,
tab types, or error-code unions.

Do not change `GetConnectedResponse` / `Instance`. The `type: InstanceType`
field picks up the new member automatically.

### W1.2 — Cloud-aware selectors and typed settings

**Files:** `webapp/src/selectors/index.ts`, `webapp/src/reducers/index.ts`
**Action:** Modify

#### `selectors/index.ts`

Change the model import (`:11`) from `{Instance}` to `{Instance, InstanceType}`.

Change `getPluginSettings` (`:80`) to:

```typescript
export const getPluginSettings = (state: GlobalState): PluginSettings | null => getPluginState(state).pluginSettings;
```

Add `PluginSettings` to the `types/model` import (named imports sorted:
`Instance`, `InstanceType`, `PluginSettings` — three names, so
`import-newlines/enforce` requires a multiline import):

```typescript
import {
    Instance,
    InstanceType,
    PluginSettings,
} from 'types/model';
```

Insert the following **after** `getInstalledInstances` (`:75`) and **before**
`instanceIsInstalled` (`:76`). `const` bindings are not hoisted; `hasCloudInstance`
must sit below `getInstalledInstances`.

```typescript
export const isCloudInstance = (instance: Instance): boolean => {
    switch (instance.type) {
    case InstanceType.CLOUD:
    case InstanceType.CLOUD_OAUTH:
        return true;
    case InstanceType.SERVER:
        return false;
    default:
        return false;
    }
};

export const hasCloudInstance = (state: GlobalState): boolean => {
    const installed = getInstalledInstances(state);
    if (!installed) {
        return false;
    }

    return installed.some(isCloudInstance);
};

export const getConnectedCloudInstances = (state: GlobalState): Instance[] => {
    return getUserConnectedInstances(state).filter(isCloudInstance);
};
```

Do **not** change `getUserConnectedInstances`. Do not filter installed Server
instances out of `getInstalledInstances` — that selector stays all types.

#### `reducers/index.ts`

Change `:7` from `{ChannelSubscription}` to `{ChannelSubscription, PluginSettings}`.

Change the `pluginSettings` function signature (`:74`) to:

```typescript
function pluginSettings(state: PluginSettings | null = null, action = {} as AnyAction): PluginSettings | null {
```

Leave the switch body unchanged. Do not add RHS slices.

### W1.3 — Fix fixtures

**Files:** `webapp/src/testlib/test-utils.tsx`,
`webapp/src/components/modals/channel_subscriptions/edit_channel_subscription.test.tsx`
**Action:** Modify

#### `test-utils.tsx`

Replace the two instance arrays (`:26-27`) with:

```typescript
        installedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.CLOUD_OAUTH}],
        userConnectedInstances: [{instance_id: 'instance1', type: InstanceType.CLOUD}, {instance_id: 'instance2', type: InstanceType.CLOUD_OAUTH}],
```

Keep `defaultUserInstanceID: 'instance1'` and the rest of the object. The
import of `InstanceType` (`:11`) is already correct; `CLOUD_OAUTH` needs no
import change.

Do **not** leave a `connectedInstances` key behind as a second copy.

#### `edit_channel_subscription.test.tsx`

At `:34-35`, rename only. Do **not** add a second instance (these tests model
one Cloud URL that matches the subscription):

```typescript
        installedInstances: [{instance_id: 'https://something.atlassian.net', type: InstanceType.CLOUD}],
        userConnectedInstances: [{instance_id: 'https://something.atlassian.net', type: InstanceType.CLOUD}],
```

### W1.4 — Selector tests (Gate 6)

**File:** `webapp/src/selectors/index.test.ts` (create)

Copyright header exactly:

```
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
```

Then a blank line, then imports. `types/*` and `testlib/*` are the same
`import/order` group (`internal`) — **no** blank line between them. `./index`
is `sibling` — one blank line before it. `Instance, InstanceType` is two
names (single-line import is fine). Four named imports from `./index` must
be multiline (`import-newlines/enforce` at 3) and alphabetized:

```typescript
import {Instance, InstanceType} from 'types/model';
import {GlobalState, pluginStateKey} from 'types/store';
import {defaultMockState} from 'testlib/test-utils';

import {
    getConnectedCloudInstances,
    getUserConnectedInstances,
    hasCloudInstance,
    isCloudInstance,
} from './index';
```

Helper (do not export). Cast through `unknown` because `GlobalState` includes
the full Mattermost store:

```typescript
function makeState(plugin: {
    installedInstances?: Instance[];
    userConnectedInstances?: Instance[];
}): GlobalState {
    return {
        [pluginStateKey]: plugin,
    } as unknown as GlobalState;
}

const cloud: Instance = {instance_id: 'https://cloud.atlassian.net', type: InstanceType.CLOUD};
const cloudOAuth: Instance = {instance_id: 'https://oauth.atlassian.net', type: InstanceType.CLOUD_OAUTH};
const server: Instance = {instance_id: 'http://jira.example.com', type: InstanceType.SERVER};
```

Required tests — names are load-bearing for Gate 6 review:

1. **`isCloudInstance is true for CLOUD_OAUTH`** — `expect(isCloudInstance(cloudOAuth)).toBe(true)`.
2. **`isCloudInstance is true for CLOUD`** — `cloud` → `true`.
3. **`isCloudInstance is false for SERVER`** — `server` → `false`.
4. **`hasCloudInstance is true for a cloud-oauth-only installation`** —
   `makeState({installedInstances: [cloudOAuth], userConnectedInstances: []})`
   → `hasCloudInstance` **true**. This is Gate 6 item 1. Installed, not
   connected. Do **not** implement this case by spreading `defaultMockState`
   (that fixture has both Cloud types after W1.3).
5. **`hasCloudInstance is true for a cloud-JWT-only installation`** —
   installed `[cloud]` → `true`.
6. **`hasCloudInstance is false when only Server/DC is installed`** —
   installed `[server]` → `false`.
7. **`hasCloudInstance is false when nothing is installed`** —
   `makeState({})` → `false` (guard on missing `installedInstances`).
8. **`getConnectedCloudInstances returns only Cloud types`** — installed
   `[cloud, cloudOAuth, server]`, connected the same three → result equals
   `[cloud, cloudOAuth]` (order preserved from `userConnectedInstances`).
   Server omitted.
9. **`getConnectedCloudInstances drops a connected instance that is not installed`**
   — installed `[server]`, connected `[cloudOAuth]` → `[]` (cross-filter of
   `getUserConnectedInstances` still applies).
10. **`defaultMockState yields a non-empty getUserConnectedInstances`** —
    `getUserConnectedInstances(defaultMockState as unknown as GlobalState)`
    `.length` to be `2`, and `getConnectedCloudInstances` of that state to
    include both `instance1` (`CLOUD`) and `instance2` (`CLOUD_OAUTH`). This
    is the fixture DoD. If this fails, the key rename is wrong.

Use `describe('selectors')` / `test('…')`. No JSX. No `eslint-disable`.

---

## File-by-file change list (current line numbers)

| File | Action | Current lines | What to do |
|------|--------|---------------|------------|
| `webapp/src/types/model.ts` | Modify | enum `:184-187`; `Instance` `:188-192` | Add `CLOUD_OAUTH = 'cloud-oauth'`; add `PluginSettings` after `Instance` |
| `webapp/src/selectors/index.ts` | Modify | imports `:11-12`; `getUserConnectedInstances` `:65-73`; `getInstalledInstances` `:75`; `getPluginSettings` `:80`; EOF `:81` | Expand model import; insert three functions after `:75`; type `getPluginSettings` |
| `webapp/src/selectors/index.test.ts` | Create | n/a | Gate 6 tests as specified |
| `webapp/src/reducers/index.ts` | Modify | import `:7`; `pluginSettings` `:74-81`; combine `:269-286` | Import `PluginSettings`; annotate reducer. Do not add keys to `combineReducers` |
| `webapp/src/testlib/test-utils.tsx` | Modify | `defaultMockState` `:24-41`; key `:27` | Rename key; add `instance2` / `CLOUD_OAUTH` to **both** arrays |
| `…/edit_channel_subscription.test.tsx` | Modify | `editChannelMockState` `:32-44`; key `:35` | Rename key only |
| `webapp/src/actions/index.ts` | **Do not touch** | SERVER `if` `:576` | Parent W1.1 was wrong; see Surprise A |
| `webapp/src/plugin.tsx` | **Do not touch** | `ui_enabled` `:50` | W4 |
| `server/**` | **Do not touch** | — | Milestone 1 already ships `cloud-oauth` and `rhs_enabled` |

---

## Existing tests that must be re-run / possibly updated

Run **all** of them via `npm run test` (Gate 6 item 2). Do not run a path
filter that excludes old suites.

### Must pass unchanged (fixture is unused or instances come from props)

- `webapp/src/manifest.test.tsx`
- `webapp/src/utils/jira_issue_metadata.test.tsx`
- `webapp/src/components/modals/connect_modal/connect_modal_form.test.tsx`
- `webapp/src/components/modals/disconnect_modal/disconnect_modal_form.test.tsx`
- `webapp/src/components/data_selectors/jira_epic_selector/jira_epic_selector.test.tsx`
- `webapp/src/components/jira_instance_and_project_selector/jira_instance_and_project_selector.test.tsx`
- `webapp/src/components/jira_ticket_tooltip/jira_ticket_tooltip.test.tsx`
- `webapp/src/components/modals/channel_subscriptions/channel_subscription_filter.test.tsx`
- `webapp/src/components/modals/channel_subscriptions/channel_subscription_filters.test.tsx`
- `webapp/src/components/modals/channel_subscriptions/channel_subscriptions.test.tsx`

### Likely pass, but this is where behavior actually changes

- `webapp/src/components/modals/create_issue/create_issue_form.test.tsx` —
  nested connected selector + `defaultMockState` now has **two** connected
  Cloud instances → instance picker visible. Auto-select target stays
  `instance1`. If a test fails, override `initialState` in **that test**, not
  in `defaultMockState`.
- `webapp/src/components/modals/channel_subscriptions/edit_channel_subscription.test.tsx`
  — local key rename (W1.3). Create-flow tests with `selectedSubscription: null`
  (`:164`, `:346`, `:404`, `:448`, `:503`, `:855`) are the ones that can see
  auto-select from Redux.

If a failure looks like “expected empty connected list”: that test was passing
**because of the bug**. Update the assertion to the post-fix truth, or give
that test a dedicated `initialState`. Never restore `connectedInstances`.

---

## Commands

From `webapp/` (or `cd webapp && …` from the worktree root). `node_modules` is
already installed.

```bash
cd webapp
npm run check-types
npm run lint
npm run test
```

- `check-types` is `tsc` (`package.json:16`). Must be clean. There is no
  actions switch to satisfy; the new `isCloudInstance` switch plus tests are
  the stand-in.
- `lint` is `eslint … --quiet --cache` (`package.json:11`). `--quiet` hides
  warnings (`explicit-function-return-type` is a warning). Errors must be
  zero. No `eslint-disable`.
- `test` is the **full** Jest suite (`package.json:13`). Do not pass a
  filename. Phase 5 baseline: 13 suites / 83 tests. W1 adds one suite
  (10 tests above) → **14 suites / 93 tests** if no existing tests are added
  or removed.

Do not run `make test` (that also runs Go). W1 does not touch Go.
Do not run `npm install` unless the binaries are missing.

---

## Definition of Done

- [ ] `InstanceType.CLOUD_OAUTH === 'cloud-oauth'`
- [ ] `isCloudInstance` is true for both `CLOUD` and `CLOUD_OAUTH`, false for `SERVER`
- [ ] **Gate 6.1:** `hasCloudInstance` is true for a state whose **only**
      installed instance is `cloud-oauth` (test name in W1.4 item 4)
- [ ] `getConnectedCloudInstances` uses `getUserConnectedInstances` (cross-filter
      preserved) then drops Server/DC
- [ ] `defaultMockState` uses `userConnectedInstances`;
      `getUserConnectedInstances(defaultMockState)` is non-empty and includes
      both Cloud types
- [ ] `editChannelMockState` uses `userConnectedInstances`
- [ ] `PluginSettings` includes `rhs_enabled`; `getPluginSettings` is typed;
      no UI reads it yet
- [ ] `webapp/src/actions/index.ts` and `webapp/src/plugin.tsx` and `server/`
      are untouched
- [ ] `npm run check-types`, `npm run lint`, `npm run test` all pass
- [ ] **Gate 6.2:** full Jest suite was run (not a path filter)
- [ ] No `eslint-disable`, no `en.json`, no `FormattedMessage`, no RHS UI

---

## Commit checkpoint (orchestration, not this engineer)

Do not commit. The lead commits Step 6 after review. Suggested message when
they do: types/selectors/fixtures for Cloud OAuth2 + `rhs_enabled` typing.
Do not push.

---

## Implementation Summary

Implemented by IE6 on 2026-08-19. Not committed. W2 not started.

### Files changed

| File | Action |
|------|--------|
| `webapp/src/types/model.ts` | `InstanceType.CLOUD_OAUTH = 'cloud-oauth'`; `PluginSettings` (`ui_enabled`, `rhs_enabled`, `security_level_empty_for_jira_subscriptions`) |
| `webapp/src/selectors/index.ts` | `isCloudInstance`, `hasCloudInstance`, `getConnectedCloudInstances`; typed `getPluginSettings` |
| `webapp/src/selectors/index.test.ts` | **Created** — 10 tests including Gate 6 cloud-oauth-only |
| `webapp/src/reducers/index.ts` | Typed `pluginSettings` as `PluginSettings \| null` (no new slices) |
| `webapp/src/testlib/test-utils.tsx` | `connectedInstances` → `userConnectedInstances`; added `instance2` / `CLOUD_OAUTH` to both arrays |
| `webapp/src/components/modals/channel_subscriptions/edit_channel_subscription.test.tsx` | Key rename only (still one Cloud URL) |

**Untouched (as required):** `webapp/src/actions/index.ts`, `webapp/src/plugin.tsx`, `server/**`, RHS UI, admin picker, `en.json`.

### Existing tests updated (and why)

- `edit_channel_subscription.test.tsx` — fixture key rename only (`connectedInstances` → `userConnectedInstances`). No assertion changes.
- **No other existing suites needed updates.** `create_issue_form.test.tsx` still passed with two connected Cloud instances (picker visible; auto-select remains `instance1`). Did not restore `connectedInstances`. Did not shrink `defaultMockState`.

### Full Jest / lint / types results

From `webapp/`:

| Command | Result |
|---------|--------|
| `npm run test` (full suite, no path filter) | **14 suites / 93 tests passed**, 0 failed. Baseline was 13/83; W1 added 1 suite / 10 tests. |
| `npm run lint` | **pass** (0 errors, `--quiet`) |
| `npm run check-types` | **fail, 249 `error TS`** — **identical count on HEAD before W1**. Zero of those errors are in W1 files. Pre-existing (actions `getState` typing, react-bootstrap `Modal`, jira metadata tests, etc.). W1 did not add or remove tsc errors. |

### Gate 6

- **Gate 6.1 confirmed:** `hasCloudInstance is true for a cloud-oauth-only installation` in `webapp/src/selectors/index.test.ts` uses `makeState({installedInstances: [cloudOAuth], userConnectedInstances: []})` — **cloud-oauth only**, no `cloud` instance, installed not connected.
- **Gate 6.2 confirmed:** full `npm run test` (not a path filter).

### Deviations

1. **`import-newlines/enforce`:** Plan W1.2 said three named imports must be multiline. Repo rule `[2, 3]` errors on multiline when there are **3 or fewer** elements. Used a single-line `import {Instance, InstanceType, PluginSettings} from 'types/model'`. Four named imports in the test file remain multiline as specified.
2. **`npm run check-types` is not clean on this branch** (249 pre-existing errors, unchanged by W1). Not fixed: would require touching `actions/index.ts` and unrelated files, which W1 forbids.

No `eslint-disable` added. No `en.json` / `FormattedMessage`. No RHS UI. No commit / push.
