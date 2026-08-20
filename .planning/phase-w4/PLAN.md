# Phase W4 Plan: Registration, gating, and popout

> Prescriptive implementation plan for **Phase W4 only** (tasks W4.1–W4.2 plus
> the Gate 8 registration negatives the parent DoD requires). An Implementation
> Engineer should be able to land this without making design decisions. Do not
> implement later phases from this file.
>
> **Do not write production TS/JS until this plan is followed as written.**
> This document is the plan; it is not the code.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase W4 (source of truth for WHAT)
- **Handoff:** `.planning/PHASE1_HANDOFF.md` (`rhs_enabled`; App Bar / RHS
  register only when the flag is on **and** a Cloud instance exists; do **not**
  hide `RHSStatusTabs` behind this flag)
- **W3:** `.planning/phase-w3/PLAN.md` — connected default at
  `components/rhs` (presentational name is `Rhs`, not `RHS`)
- **W2:** `.planning/phase-w2/PLAN.md` — `utils/rhs_view_state.ts` already
  persists `{instance, tab, sort}` keyed by user id
- **W5:** `.planning/phase-w5/PLAN.md` — **already landed** the ungated
  `registerAdminConsoleCustomSetting('RHSStatusTabs', …)` line. Do not move
  it. Do not wrap it in `rhs_enabled`.
- **Orchestration:** Step 8 / **Gate 8 registration half** of
  `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `1e50575` (W5 commit; W3 is `b0ee5cd`). Do not rebase.
  Do not implement against an older SHA.
- **Package:** `webapp/` (Jest 29, TypeScript 5.7 strict, ESLint 8, RTL 14)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer.** Sequential. Nothing in this phase is
  parallelizable. W5 already owns the admin-register line in `plugin.tsx`;
  you add a **sibling** App Bar block. Do not take over W5’s line.

## Scope

**In scope:**

- `webapp/src/utils/rhs_register.ts` — **create** (pure gate + App Bar
  icon URL + title constants + `registerJiraAppBar`)
- `webapp/src/utils/rhs_register.test.ts` — **create** (gate matrix,
  including **cloud-oauth-only**)
- `webapp/src/utils/rhs_popout.ts` — **create** (`isRHSPopoutPathname`)
- `webapp/src/utils/rhs_popout.test.ts` — **create** (pathname shapes;
  **not** `isPopoutWindow`)
- `webapp/src/plugin.tsx` — **modify**: named-export `setupUILater`;
  import `Rhs` + helpers; **one gated** `registerAppBarComponent` block;
  remove dead `headerButtonId` scaffolding. **Do not move or gate** the
  existing `registerAdminConsoleCustomSetting` line.
- `webapp/src/plugin.test.ts` — **create** (**Gate 8 lives here** —
  registry-mock negatives)
- `webapp/src/components/rhs/rhs.tsx` — **modify by a few lines**: mark
  popout via `isRHSPopoutPathname(window.location.pathname)`; boot
  sequence stays `getConnected` → `restoreRHSViewState` →
  `resolveAndFetchRHSIssues`
- `webapp/src/components/rhs/rhs.test.tsx` — **append** one popout mount
  test (do not rename existing Gate 8 loading≠empty names)
- `webapp/src/actions/rhs.test.ts` — **append** the persist + rehydrate +
  re-fetch test that proves issues are **not** in localStorage

**Out of scope (do not do these):**

- Moving, deleting, or wrapping
  `registry.registerAdminConsoleCustomSetting('RHSStatusTabs', RHSStatusSetting, {showTitle: true})`
- Nesting App Bar registration inside `if (settings.ui_enabled)`
- `registerRightHandSidebarComponent` as a separate call (combined
  `registerAppBarComponent` is the primary). Split-call is **fallback
  only if the combined form is proven broken** — do not ship the
  fallback in this phase.
- `registerChannelHeaderButtonAction` / channel-header fallback
  (research §6.3; `min_server_version` is 10.7)
- `registerRHSPluginPopoutListener`
- `window.WebappUtils.popouts` / `isPopoutWindow()`
- Parsing team or channel out of the popout URL
- A second persist writer (W2 already persists on
  `fetchRHSIssues` / setters via `saveRHSViewState`)
- Carrying `rhsIssues` across the window boundary
- Editing `utils/rhs_view_state.ts` (W2 done — **call** it)
- Editing `actions/rhs.ts` persist helpers
- System Console picker (`components/admin_console/`)
- Anything under `server/`
- `en.json` / `FormattedMessage`
- `eslint-disable` comments
- New npm dependencies / webpack SVG loaders
- Committing or pushing

---

## Research (read this before coding)

Line numbers are as of **`1e50575`**. If they have drifted, search; do not
invent a different design.

### 1. `setupUILater` today — `webapp/src/plugin.tsx:42-127`

```42:127:webapp/src/plugin.tsx
const setupUILater = (registry: any, store: Store<object, Action<object>>): () => Promise<void> => async () => {
    registry.registerReducer(reducers);

    const settings = await store.dispatch(getSettings());
    const {id: PluginId} = manifest;

    try {
        await getConnected()(store.dispatch, store.getState);

        if (settings.ui_enabled) {
            registry.registerRootComponent(ConnectModal);
            ...
            registry.registerLinkTooltipComponent(LinkTooltip);
        }

        registry.registerRootComponent(ChannelSubscriptionsModal);
        registry.registerAdminConsoleCustomSetting('RHSStatusTabs', RHSStatusSetting, {showTitle: true});

        const hooks = new Hooks(store, settings);
        registry.registerSlashCommandWillBePostedHook(hooks.slashCommandWillBePostedHook);
    } finally {
        registry.registerWebSocketEventHandler(...);
    }
};
```

Load-bearing facts:

1. **`initialize()` only mounts `SetupUI`.** All real registration is in
   `setupUILater()`, which runs on login (`setup_ui.jsx:15-18`). Register
   App Bar **here**, not in `initialize()`.
2. **`getSettings()` returns the settings object itself** on success
   (`actions/index.ts:469-487` — `return data`, not `{data}`). That object
   already includes `rhs_enabled` (`types/model.ts:195-199`). Read
   `settings.rhs_enabled` the same way `:51` reads `settings.ui_enabled`.
   On fetch failure the thunk returns `{error}` — treat missing
   `rhs_enabled` as **false** (do not throw).
3. **`getConnected()` is invoked as**
   `getConnected()(store.dispatch, store.getState)` — **not**
   `store.dispatch(getConnected())`. That is the existing pattern. Keep
   it. It populates `installedInstances` via `RECEIVED_INSTANCE_STATUS`
   (`actions/index.ts:507-510`, `reducers/index.ts:30-38`) from
   `action.data.instances`.
4. **`hasCloudInstance` must run after `getConnected()`.** Before that
   call, `installedInstances` is `[]` and the gate would always be false.
5. **`hasCloudInstance` is installed Cloud, not connected Cloud**
   (`selectors/index.ts:99-106`). App Bar still registers when a Cloud
   instance is installed but the user is not connected — the panel then
   shows not-connected (W3). Do **not** gate on
   `getConnectedCloudInstances`.
6. **W5’s admin line is ungated on purpose** (handoff + parent W5.2). An
   admin must configure tabs *before* enabling the feature. It sits with
   `ChannelSubscriptionsModal`, outside `ui_enabled`. **Leave it exactly
   where it is.**

### 2. Exact `plugin.tsx` insertion (collision with W5)

**Do not touch these two existing lines:**

```typescript
import RHSStatusSetting from 'components/admin_console/rhs_status_setting';
```

```typescript
        registry.registerAdminConsoleCustomSetting('RHSStatusTabs', RHSStatusSetting, {showTitle: true});
```

**Exact body insertion:** after the `if (settings.ui_enabled) { … }`
block closes (currently after `registerLinkTooltipComponent(LinkTooltip);`
and its closing `}`), **before**
`registry.registerRootComponent(ChannelSubscriptionsModal);` insert:

```typescript
        if (shouldRegisterJiraRHS(settings, store.getState() as GlobalState)) {
            registerJiraAppBar(registry, store.getState() as GlobalState);
        }
```

Why this spot:

- **Sibling of `ui_enabled`, not nested in it.** RHS has its own flag.
  `ui_enabled: false` + `rhs_enabled: true` + a Cloud instance **must**
  still register App Bar.
- **After `getConnected()`**, so `hasCloudInstance` sees installed
  instances.
- **Before** the ungated `ChannelSubscriptionsModal` /
  `registerAdminConsoleCustomSetting` pair, so those two lines stay
  visually and semantically ungated.
- **Not in `finally`.** That block is websocket handlers only.

**Exact import edits in `plugin.tsx`:**

1. Leave the `RHSStatusSetting` import (`:17`) **exactly as it is**.
2. Change the selectors import (`:23`) from the webpack bare specifier
   `'selectors'` to **relative** `'./selectors'`. Same three names, single
   line. Jest has **no** `^selectors$` mapper (`package.json:100-112`);
   `plugin.test.ts` importing `setupUILater` will otherwise fail to
   resolve it (same class of bug W2 fixed for `'action_types'`).
   Do **not** add `hasCloudInstance` here — `rhs_register.ts` imports it.
   Do **not** import `Rhs` in `plugin.tsx`.
3. After the existing `utils/posts` import (`:24`), add:

```typescript
import {registerJiraAppBar, shouldRegisterJiraRHS} from 'utils/rhs_register';
```

   Two names → single line, ASCII (`registerJiraAppBar` before
   `shouldRegisterJiraRHS`).

**Named-export `setupUILater`** so Gate 8 tests can call it:

Change line 42 from `const setupUILater =` to
`export const setupUILater =`. The `Plugin` class keeps using it. Default
export remains the class.

### 3. Combined `registerAppBarComponent` — `rhsComponent` + `rhsTitle`

From `research.md` and `context.md` (verified byte-for-byte in
`release-10.7` and master):

```
registerAppBarComponent(iconUrl, action?, tooltipText, supportedProductIds?, rhsComponent?, rhsTitle?)
```

Passing `rhsComponent` internally calls
`registerRightHandSidebarComponent` and wires the icon to
`toggleRHSPlugin`. **Then `action` must be omitted** (pass `undefined`).
One call registers both surfaces.

**No first-party plugin uses this combined form.** Playbooks uses the
three-argument form plus a separate
`registerRightHandSidebarComponent`. GitHub never calls
`registerAppBarComponent`. The combined form is verified to exist and to
work at 10.7. Use it. Do **not** also call
`registerRightHandSidebarComponent`. Do **not** guard with
`if (registry.registerAppBarComponent)` — `min_server_version` is 10.7
and the method is identical on that floor.

`registerAppBarComponent` **never forwards `showPopout`**. On 11.3+ the
popout button always renders. That is fine: we want popout.

`supportedProductIds`: pass `undefined` (all products). Do not invent a
channels-only filter.

### 4. App Bar icon URL — no webpack SVG loader

`webapp/webpack.config.js:41-49` url-loads **png/jpg/gif only**. Jest
maps `*.svg` to `identity-obj-proxy`. **Do not** `import icon from
'…/icon.svg'`. **Do not** add an SVG loader.

`assets/icon.svg` is the plugin marketplace icon (`plugin.json`
`icon_path`). The Makefile copies `public/` into the bundle
(`Makefile:260-262`). Mattermost serves those files at
`{pluginRoute}/public/{file}`.

**Copy** `assets/icon.svg` to `public/icon.svg` (same bytes). Then:

```typescript
export function getJiraAppBarIconUrl(state: GlobalState): string {
    return getPluginServerRoute(state) + '/public/icon.svg';
}
```

`getPluginServerRoute` already exists (`selectors/index.ts:26-38`) and
is how every other plugin HTTP path is built. With SiteURL
`http://localhost:8065` this is `/plugins/jira/public/icon.svg`.

Do **not** use `components/icon.tsx` (it is a React class for menus, not
a URL). Do not use a data URI.

### 5. Gate — `rhs_enabled` AND installed Cloud

Handoff:

> App Bar / RHS register only when `rhs_enabled` is true **and** a Cloud
> instance exists.

```typescript
export function shouldRegisterJiraRHS(
    settings: {rhs_enabled?: boolean} | null | undefined,
    state: GlobalState,
): boolean {
    if (!settings || settings.rhs_enabled !== true) {
        return false;
    }
    return hasCloudInstance(state);
}
```

`hasCloudInstance` is true for `cloud` **and** `cloud-oauth`
(`isCloudInstance`, `selectors/index.ts:87-97`). A **cloud-oauth-only**
install must register — that is the case the old enum got wrong (W1 /
Gate 6). Server/DC-only must **not** register even when `rhs_enabled` is
true.

`rhs_enabled !== true` (not a truthy check on a random `{error}` object)
is required because a failed `getSettings()` returns `{error}`.

### 6. Persist is already in W2 — do not re-implement it

`persistCurrentRHSView` (`actions/rhs.ts:59-66`) writes
`{instance, tab, sort}` through `saveRHSViewState` on every
`fetchRHSIssues` (page-1, which also handles tab / sort / instance
changes) and on the setters. Key is `jira:rhs-view:${userId}`
(`utils/rhs_view_state.ts:12-16`). **Never persist issues.**

W3’s panel never calls the setters; it calls `resolveAndFetchRHSIssues`,
which calls `fetchRHSIssues`, which persists. On-change persist is
already live.

`restoreRHSViewState` (`actions/rhs.ts:110-122`) reads localStorage and
dispatches `HYDRATE_RHS_VIEW_STATE`. It does **not** write. W3 boot
already calls it before the first fetch.

**W4 does not add a second persist path.** W4 proves popout: a **fresh**
store + `/_popout/` pathname + seeded localStorage → hydrate view state
→ **new** issues fetch. Issues payload must not appear in localStorage.

### 7. Popout detect — pathname, not `isPopoutWindow()`

`window.WebappUtils.popouts` first appears in **11.3**. Floor is **10.7**.
`isPopoutWindow()` is unusable. Its internal check is
`window.location.pathname.startsWith('/_popout/')` (`context.md`).

URL shape changed at 11.6:

- 11.3–11.5: `/_popout/rhs/{team}/{channel}/plugin/{id}`
- 11.6+: `/_popout/rhs/{team}/plugin/{id}?channel={channel}`

**Never parse team or channel.** Prefix check is enough.

```typescript
export function isRHSPopoutPathname(pathname: string): boolean {
    return pathname.indexOf('/_popout/') === 0;
}
```

Use `indexOf === 0` rather than `startsWith` if you want to match the
W3 style; `startsWith` is also fine on this TS target. Pick one and
test both URL shapes.

The helper takes `pathname` as an argument so tests do not fight jsdom’s
`window.location`. Production `rhs.tsx` passes
`window.location.pathname`.

**Do not** call `registerRHSPluginPopoutListener`. **Do not** read
`window.WebappUtils`.

Popout is a **full app boot**: new Redux store, `initialize()` runs
once, `setupUILater` runs, `Rhs` mounts, W3 boot runs. Duplicate
registration is a non-issue (`loadedPlugins` is per-window).

W3 boot (`rhs.tsx:77-100`) is already:

1. `await getConnected()`
2. `restoreRHSViewState()`
3. `await resolveAndFetchRHSIssues()`

That **is** the popout rehydrate + re-fetch. W4 does not add a second
boot. W4 marks the popout on the root node so a test can prove the
pathname was seen, and adds an action-level test that a fresh store
rehydrates from localStorage and hits the network.

`rhs.tsx` change (root div, currently `:193-196`):

```tsx
    const isPopout = isRHSPopoutPathname(window.location.pathname);

    return (
        <div
            className='jira-rhs'
            data-testid='jira-rhs'
            data-rhs-popout={isPopout ? 'true' : 'false'}
        >
```

Keep `data-testid='jira-rhs'` — W3 tests use it. The extra attribute is
the popout handle. `'true'` / `'false'` are JS string expressions, not
JSX text literals (`jsx-no-literals` is fine).

Do **not** change the `useEffect` boot order. Do **not** skip
`getConnected` in a popout. Do **not** pass `booting` as a prop.

### 8. Dead `headerButtonId` scaffolding

`SetupUI` (`setup_ui.jsx:9-13`) propTypes are only `haveSetupUI`,
`finishedSetupUI`, `setupUI`. It **never reads** `headerButtonId`,
`setHeaderButtonId`, or `registry`. Parent W4.1: remove the dead
channel-header scaffolding.

In `plugin.tsx` **delete**:

- `private headerButtonId = '';` (`:131`)
- `private setHeaderButtonId = (id: string) => { … };` (`:138-140`)
- `this.headerButtonId = '';` inside `initialize` (`:145`)
- JSX props `headerButtonId={…}` and `setHeaderButtonId={…}` (`:156-157`)

Leave `SetupUI`’s remaining props (`registry`, `setupUI`, `haveSetupUI`,
`finishedSetupUI`). Do **not** edit `setup_ui.jsx`. Do **not** replace
`headerButtonId` with the App Bar id — we do not need to store it.

### 9. ESLint / i18n / test constraints (same as W2–W5)

| Rule | W4 impact |
|------|-----------|
| `header/header` | two-line copyright on every new file |
| `import/order` | blank line between groups |
| `import-newlines/enforce` | wrap at **4+** named imports |
| `object-curly-spacing` | `"never"` |
| `react/jsx-no-literals` | no bare JSX text. Attribute values and `const` strings are fine |
| `max-lines` | 650, counts comments. `plugin.tsx` is ~163 today |
| `no-undefined` | `null`, not `undefined`, in our types. Passing `undefined` as the
  **omitted `action` argument** to the host API is required — that is a
  positional skip, not an identifier we store. Do not write
  `const action = undefined`. Pass the literal in the call. |
| `no-underscore-dangle` | no `_popout` helper names. The path string `'/_popout/'` is fine |
| Function components | `rhs.tsx` stays a function; do not convert `Plugin` to a function |

**Do not add `en.json` or `FormattedMessage`.** i18n is milestone F1.

`node_modules` exists. Do not `npm install`. `check-types` still reports
**pre-existing `error TS`**. Fail W4 only if a **new** file or a
W4-touched file appears in that list.

### 10. Gate 8 (registration half — must be in this phase’s tests)

From `IMPL_ORCHESTRATION_PLAN.md` Step 8 / parent W4 DoD / the user
brief. These failures are **silent** in production. Each needs an
explicit “did not happen” (or “did happen for cloud-oauth-only”)
assertion.

1. With `rhs_enabled` false, the registry mock received **no**
   `registerAppBarComponent` call — **even if** a Cloud instance is
   installed.
2. With `rhs_enabled` true but **only Server/DC** installed, **no**
   `registerAppBarComponent` call.
3. With `rhs_enabled` true and a **`cloud-oauth`-only** installation,
   registration **does** happen. This is the case that matters.
4. In cases (1)–(2), `registerAdminConsoleCustomSetting` **was still
   called** with `'RHSStatusTabs'`. Proves we did not wrap W5’s line.
5. A simulated `/_popout/` pathname rehydrates `{instance, tab, sort}`
   from `localStorage` and triggers a **fresh** issues fetch. localStorage
   JSON has **no** `issues` key.
6. No `isPopoutWindow` / `WebappUtils.popouts` /
   `registerRHSPluginPopoutListener` in new or edited production files.

---

## Surprises

### Surprise A — W5 already edited `plugin.tsx`

HEAD `1e50575` added the `RHSStatusSetting` import and the ungated
`registerAdminConsoleCustomSetting` line after
`ChannelSubscriptionsModal`. **Do not revert, move, or gate those
lines.** `git diff 1e50575 -- webapp/src/plugin.tsx` when you are done
must **not** show a minus on that register call.

### Surprise B — persist and rehydrate already exist

W2 writes localStorage; W3 boot reads it then fetches. W4’s job is
**registration + pathname detect + tests**, not a second persist
implementation. A popout is a remount of `Rhs` with a fresh store.

### Surprise C — `getConnected` must run before the Cloud gate

`hasCloudInstance` reads `installedInstances`. Those arrive from
`getConnected` → `RECEIVED_INSTANCE_STATUS`. Gating before that call
would never register.

### Surprise D — do not nest inside `ui_enabled`

`EnableJiraUI` and `EnableJiraRHS` are independent. Channel
subscriptions stay registered when UI is off; App Bar stays off when
RHS is off. A user with UI disabled and RHS enabled must still get the
App Bar if a Cloud instance exists.

### Surprise E — gate on **installed** Cloud, not connected Cloud

`hasCloudInstance` ≠ `getConnectedCloudInstances`. Zero personal
connections + one installed Cloud → **register**. The panel shows
Connect (W3). Using the connected selector would hide the App Bar from
the people who need it to connect.

### Surprise F — combined `registerAppBarComponent` has no in-repo precedent

Pass `undefined` for `action` and `supportedProductIds`. Fifth argument
is the connected `Rhs`. Sixth is the title string `'Jira'`. Do not also
call `registerRightHandSidebarComponent`.

### Surprise G — webpack cannot import SVG

Copy `assets/icon.svg` → `public/icon.svg`. URL via
`getPluginServerRoute` + `'/public/icon.svg'`.

### Surprise H — `SetupUI` never used `headerButtonId`

Safe to delete from `plugin.tsx`. Do not edit `setup_ui.jsx`.

### Surprise I — presentational name is `Rhs`

`react/jsx-pascal-case` rejected `RHS`. The connected default export
path is still `components/rhs`. Register that default.

### Surprise J — `check-types` is dirty on this branch

Pre-existing errors. Ignore unless a W4 file is in the list.

### Surprise K — Jest cannot resolve `from 'selectors'`

`plugin.tsx:23` uses the webpack bare specifier. Jest has no
`^selectors$` mapper. W4 is the first phase that imports `plugin.tsx`
from a test, so change that one import to `'./selectors'`. Do not add a
Jest mapper. Do not change other bare specifiers that **are** mapped
(`components/`, `utils/`, `types/`).

### Surprise L — jsdom `window.location` is painful

`isRHSPopoutPathname` takes a string. Unit tests pass path strings.
The one component test may stub `window.location.pathname`; if that
stub is flaky, the action-level rehydrate test (item 5) is the
load-bearing Gate 8 popout proof.

---

## Tasks

### W4.1 — Pure helpers (gate, icon, popout detect)

**Files:** `webapp/src/utils/rhs_register.ts`,
`webapp/src/utils/rhs_popout.ts`, `public/icon.svg`
**Action:** Create

#### `public/icon.svg`

Copy `assets/icon.svg` to `public/icon.svg` (same bytes). Do not edit
the SVG. `public/` is already bundled (`Makefile:260-262`).

#### `webapp/src/utils/rhs_popout.ts`

Copyright header, then:

```typescript
export function isRHSPopoutPathname(pathname: string): boolean {
    return pathname.indexOf('/_popout/') === 0;
}
```

No other exports. Do not import `WebappUtils`. Do not parse the rest of
the path.

#### `webapp/src/utils/rhs_register.ts`

Copyright header, then:

```typescript
import Rhs from 'components/rhs';
import {getPluginServerRoute, hasCloudInstance} from '../selectors';

import {GlobalState} from 'types/store';

export const JIRA_RHS_TITLE = 'Jira';

export function shouldRegisterJiraRHS(
    settings: {rhs_enabled?: boolean} | null | undefined,
    state: GlobalState,
): boolean {
    if (!settings || settings.rhs_enabled !== true) {
        return false;
    }
    return hasCloudInstance(state);
}

export function getJiraAppBarIconUrl(state: GlobalState): string {
    return getPluginServerRoute(state) + '/public/icon.svg';
}

export function registerJiraAppBar(registry: any, state: GlobalState): void {
    registry.registerAppBarComponent(
        getJiraAppBarIconUrl(state),
        undefined,
        JIRA_RHS_TITLE,
        undefined,
        Rhs,
        JIRA_RHS_TITLE,
    );
}
```

`Rhs` is the **connected default** from `components/rhs/index.ts`. Jest
maps `^components/(.*)$` (`package.json:107`). Webpack resolves relative
`../selectors` the same way. **Do not** `from 'selectors'` — Jest has
no `^selectors$` mapper (W2 Surprise D / W5 helpers duplicated Cloud
checks for the same reason). Here we **must** call the real
`hasCloudInstance`, so use a relative import.

`registry: any` matches `setupUILater`. Do not invent a
`PluginRegistry` interface.

`import-newlines`: `getPluginServerRoute, hasCloudInstance` is two
names — single line, ASCII.

### W4.2 — `plugin.tsx` registration

**File:** `webapp/src/plugin.tsx`
**Action:** Modify

1. Change `from 'selectors'` to `from './selectors'` (same three names).
   Jest has no `^selectors$` mapper; `plugin.test.ts` will import this file.
2. Add
   `import {registerJiraAppBar, shouldRegisterJiraRHS} from 'utils/rhs_register';`
   after `utils/posts`. Do **not** import `Rhs` here.
3. Change `const setupUILater` to `export const setupUILater`.
4. Insert the `if (shouldRegisterJiraRHS(…)) { registerJiraAppBar(registry, store.getState() as GlobalState); }`
   block per §2.
5. Remove `headerButtonId` scaffolding per §8.

`registerJiraAppBar` imports `Rhs` itself. `plugin.tsx` stays a thin
orchestrator. Tests can call `registerJiraAppBar` without going through
the class.

### W4.3 — `rhs.tsx` popout mark

**File:** `webapp/src/components/rhs/rhs.tsx`
**Action:** Modify

Add to the existing `utils/rhs_resolve` import group (blank line
between groups). `rhs_popout` is `utils/` — same internal group as
`rhs_resolve` if you add it to that import path:

```typescript
import {isRHSPopoutPathname} from 'utils/rhs_popout';
import {rhsTabsEqual} from 'utils/rhs_resolve';
```

`import/order`: consecutive internals, ASCII by module path
(`rhs_popout` before `rhs_resolve`).

In the function body, **after** the existing destructure and **before**
the `useEffect`, compute:

```typescript
    const isPopout = isRHSPopoutPathname(window.location.pathname);
```

Put `data-rhs-popout={isPopout ? 'true' : 'false'}` on the root
`.jira-rhs` div. **Do not** change the boot `useEffect`. **Do not** add
`isPopout` to the effect deps (boot-once stays `[]`).

---

## Tests

Copyright header, relative imports, no `eslint-disable`, no `en.json`.
`clearMocks: true` is already on. `afterEach`: `localStorage.clear()`
and `resetRHSIssuesInFlight()` where you dispatch fetches.

### Shared fixtures

```typescript
const cloudOAuth = {instance_id: 'https://oauth.example.atlassian.net', type: InstanceType.CLOUD_OAUTH};
const cloudJwt = {instance_id: 'https://cloud.example.atlassian.net', type: InstanceType.CLOUD};
const serverDc = {instance_id: 'http://jira.example.com', type: InstanceType.SERVER};

const assignedTab = {kind: 'assigned', name: 'Assigned'};
const inProgressTab = {kind: 'category', name: 'In Progress', key: 'indeterminate'};
```

Settings payloads (what `getSettings` returns on success):

```typescript
const settingsRhsOff = {
    ui_enabled: false,
    rhs_enabled: false,
    security_level_empty_for_jira_subscriptions: true,
};
const settingsRhsOn = {
    ...settingsRhsOff,
    rhs_enabled: true,
};
```

### `webapp/src/utils/rhs_popout.test.ts` (create)

1. **`isRHSPopoutPathname is true for the 11.3 popout path shape`** —
   `'/_popout/rhs/team/channel/plugin/jira'` → `true`.
2. **`isRHSPopoutPathname is true for the 11.6 popout path shape`** —
   `'/_popout/rhs/team/plugin/jira'` → `true`. Query string is not in
   `pathname`; do not pass `?channel=`.
3. **`isRHSPopoutPathname is false for a normal team channel path`** —
   `'/team/channel'` → `false`.
4. **`isRHSPopoutPathname is false when _popout appears later in the path`** —
   `'/team/_popout/nope'` → `false` (`indexOf === 0` / `startsWith`).

### `webapp/src/utils/rhs_register.test.ts` (create)

Use a tiny `makeState` like `selectors/index.test.ts` (`as unknown as GlobalState`).
SiteURL is **not** required for `shouldRegisterJiraRHS`. It **is**
required for `getJiraAppBarIconUrl`:

```typescript
function makeState(plugin: {installedInstances?: Instance[]}): GlobalState {
    return {
        entities: {
            general: {config: {SiteURL: 'http://localhost:8065'}},
        },
        [pluginStateKey]: plugin,
    } as unknown as GlobalState;
}
```

Required names:

1. **`shouldRegisterJiraRHS is false when rhs_enabled is false even with a Cloud instance`**
2. **`shouldRegisterJiraRHS is false when rhs_enabled is true but only Server/DC is installed`**
3. **`shouldRegisterJiraRHS is true for a cloud-oauth-only installation when rhs_enabled is true`**
   — **Gate 8 / W1 case.** `installedInstances: [cloudOAuth]`.
4. **`shouldRegisterJiraRHS is true for a cloud-JWT-only installation when rhs_enabled is true`**
5. **`shouldRegisterJiraRHS is false when settings is an error object`** —
   `shouldRegisterJiraRHS({error: true} as any, makeState({installedInstances: [cloudOAuth]}))`
   is `false`.
6. **`getJiraAppBarIconUrl uses the plugin public path`** — result
   contains `/plugins/jira/public/icon.svg` and does **not** contain
   `/api/v2`.
7. **`registerJiraAppBar calls registerAppBarComponent once with omitted action and Rhs`** —
   mock `{registerAppBarComponent: jest.fn()}`. After the call:
   `toHaveBeenCalledTimes(1)`;
   `mock.calls[0][1]` is `undefined`;
   `mock.calls[0][2]` and `[5]` are `JIRA_RHS_TITLE`;
   `mock.calls[0][4]` is the connected module’s default (import `Rhs from 'components/rhs'`
   in the test and `toBe(Rhs)`).

### `webapp/src/plugin.test.ts` (create) — Gate 8 lives here

Import `{setupUILater}` from `./plugin` (named export). Do **not**
instantiate `Plugin`.

Reuse a real store like W2 `makeRHSStore`, plus `installedInstances`
already in plugin state **or** populated via mocked `userinfo`.
`setupUILater` always calls `getSettings` (fetch `/api/v2/settingsinfo`)
and `getConnected` (fetch `/api/v2/userinfo`). Drive `global.fetch` by
URL:

```typescript
function mockPluginFetches(opts: {settings: object; userinfo: object}) {
    const fetchMock = global.fetch as jest.Mock;
    fetchMock.mockImplementation((url: string) => {
        const href = String(url);
        if (href.indexOf('/api/v2/settingsinfo') !== -1) {
            return Promise.resolve({
                ok: true,
                json: () => Promise.resolve(opts.settings),
            });
        }
        if (href.indexOf('/api/v2/userinfo') !== -1) {
            return Promise.resolve({
                ok: true,
                json: () => Promise.resolve(opts.userinfo),
            });
        }
        return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({}),
        });
    });
    return fetchMock;
}
```

`userinfo` must include `instances` (installed — this is what
`RECEIVED_INSTANCE_STATUS` stores) **and** `user_info` so the connected
reducer does not explode:

```typescript
function userinfoWithInstances(instances: Instance[]) {
    return {
        can_connect: true,
        is_connected: instances.length > 0,
        instances,
        user_info: {
            connected_instances: instances,
            default_instance_id: instances[0] ? instances[0].instance_id : '',
        },
    };
}
```

Registry mock — **every method `setupUILater` calls**, as `jest.fn()`:

```typescript
function makeRegistry() {
    return {
        registerReducer: jest.fn(),
        registerRootComponent: jest.fn(),
        registerPostDropdownMenuAction: jest.fn(),
        registerLinkTooltipComponent: jest.fn(),
        registerAdminConsoleCustomSetting: jest.fn(),
        registerSlashCommandWillBePostedHook: jest.fn(),
        registerWebSocketEventHandler: jest.fn(),
        registerAppBarComponent: jest.fn(),
    };
}
```

Store: `createStore` + `redux-thunk` + real plugin reducer, with
`entities.general.config.SiteURL` and `entities.users.currentUserId`
(same as W2 `makeRHSStore`). `setupUILater` types the store as
`Store<object, Action<object>>` — pass it `as any` if needed.

`beforeEach` / `afterEach`: reset fetch; `localStorage.clear()`.

Helper:

```typescript
async function runSetup(registry: ReturnType<typeof makeRegistry>, store: ReturnType<typeof makeRHSStore>) {
    await setupUILater(registry, store as any)();
}
```

**Names are load-bearing for Gate 8 review:**

1. **`setupUILater does not call registerAppBarComponent when rhs_enabled is false`**
   — **Gate 8.1.** `settingsRhsOff`, `userinfoWithInstances([cloudOAuth])`.
   After `runSetup`:
   `expect(registry.registerAppBarComponent).not.toHaveBeenCalled()`.
   `expect(registry.registerAdminConsoleCustomSetting).toHaveBeenCalledWith('RHSStatusTabs', expect.anything(), {showTitle: true})`.

2. **`setupUILater does not call registerAppBarComponent when only Server/DC is installed`**
   — **Gate 8.2.** `settingsRhsOn`, `userinfoWithInstances([serverDc])`.
   `registerAppBarComponent` not called.
   Admin setting **still** called (same assertion as test 1).

3. **`setupUILater calls registerAppBarComponent for a cloud-oauth-only installation when rhs_enabled is true`**
   — **Gate 8.3.** `settingsRhsOn`, `userinfoWithInstances([cloudOAuth])`
   and **no** `cloud` JWT instance.
   `toHaveBeenCalledTimes(1)`.
   `mock.calls[0][1]` is `undefined`.
   `mock.calls[0][4]` is the connected `Rhs`.
   `mock.calls[0][5]` is `JIRA_RHS_TITLE`.
   Icon URL (`[0]`) contains `/public/icon.svg`.
   Admin setting still called.

Also (not a substitute for 1–3):

4. **`setupUILater does not nest App Bar registration inside ui_enabled`**
   — `ui_enabled: false`, `rhs_enabled: true`, `[cloudOAuth]`.
   `registerAppBarComponent` **is** called.
   `registerRootComponent` was **not** called with `ConnectModal` (or:
   `registerPostDropdownMenuAction` was not called). Proves the new `if`
   is a sibling, not a child, of `settings.ui_enabled`.

If importing `./plugin` pulls JSX and fails to parse, the file is
already `plugin.tsx` and babel-jest transforms it (`package.json:118`).
Do not rename the production file.

### `webapp/src/actions/rhs.test.ts` (extend) — popout persist / rehydrate

Keep existing Gate 7 names. Append:

1. **`a simulated /_popout/ pathname rehydrates the persisted view state and triggers a fresh issues fetch`**
   — **Gate 8.5 / parent W4 DoD.** Exact recipe:

   ```typescript
   expect(isRHSPopoutPathname('/_popout/rhs/team/plugin/jira')).toBe(true);

   const saved = {
       instance: 'https://oauth.example.atlassian.net',
       tab: inProgressTab,
       sort: 'created' as const,
   };
   saveRHSViewState('user-1', saved);

   const stored = JSON.parse(localStorage.getItem('jira:rhs-view:user-1') as string);
   expect(stored.issues).toBeUndefined();
   expect(stored.instance).toBe(saved.instance);

   const store = makeRHSStore();
   store.dispatch({
       type: ActionTypes.RECEIVED_INSTANCE_STATUS,
       data: {instances: [cloudOAuth]},
   });
   store.dispatch({
       type: ActionTypes.RECEIVED_CONNECTED,
       data: userinfoWithInstances([cloudOAuth]),
   });

   store.dispatch(restoreRHSViewState());
   expect(pluginFrom(store).rhsInstanceID).toBe(saved.instance);
   expect(pluginFrom(store).rhsSort).toBe('created');
   expect(pluginFrom(store).rhsIssues).toEqual([]);

   const fetchMock = mockFetchOk(page1);
   await store.dispatch(resolveAndFetchRHSIssues() as any);
   expect(fetchMock).toHaveBeenCalledTimes(1);
   const url = String(fetchMock.mock.calls[0][0]);
   expect(url).toContain('/api/v2/rhs/issues');
   expect(url).toContain('sort=created');
   expect(url).toContain('tab_kind=category');
   expect(url).toContain('tab_key=indeterminate');
   expect(pluginFrom(store).rhsIssues.length).toBeGreaterThan(0);
   ```

   Import `saveRHSViewState` from `utils/rhs_view_state`,
   `restoreRHSViewState` / `resolveAndFetchRHSIssues` from `./rhs`,
   `isRHSPopoutPathname` from `utils/rhs_popout`.

   This is a **fresh store** (popout isolation) + persist + re-fetch.
   Do **not** put issues in the saved blob.

### `webapp/src/components/rhs/rhs.test.tsx` (extend)

Keep all existing names, especially
`loading state is not the empty state`. Append:

1. **`popout pathname sets data-rhs-popout and still restores then fetches`**
   — Stub pathname so `window.location.pathname` is
   `'/_popout/rhs/team/plugin/jira'` (if jsdom blocks `delete
   window.location`, assign via `Object.defineProperty` on a copy, or
   spy `window.location` with `pathname` only). Mount presentational
   `Rhs` like the other tests. After `waitFor`:
   `expect(screen.getByTestId('jira-rhs')).toHaveAttribute('data-rhs-popout', 'true')`.
   Order array still `['connected', 'restore', 'fetch']` (copy test 14’s
   mock style). Restore in `afterEach`.

   A **control**: default pathname → `data-rhs-popout='false'`.

If the location stub is flaky, keep the attribute test best-effort and
**do not drop Gate 8.5** in `rhs.test.ts` (actions). The attribute is
nice-to-have; the action test is the gate.

---

## File-by-file change list (current line numbers)

Line numbers as of **`1e50575`**.

| File | Action | Current lines | What to do |
|------|--------|---------------|------------|
| `public/icon.svg` | Create | n/a (copy of `assets/icon.svg`) | App Bar URL target |
| `webapp/src/utils/rhs_popout.ts` | Create | n/a | `isRHSPopoutPathname` |
| `webapp/src/utils/rhs_popout.test.ts` | Create | n/a | 11.3 / 11.6 / negative paths |
| `webapp/src/utils/rhs_register.ts` | Create | n/a | gate + icon URL + `registerJiraAppBar` |
| `webapp/src/utils/rhs_register.test.ts` | Create | n/a | gate matrix including cloud-oauth-only |
| `webapp/src/plugin.tsx` | Modify | `setupUILater` `:42-127`; `ui_enabled` `:51-114`; ChannelSubscriptionsModal `:116`; **admin register `:117` DO NOT TOUCH**; hooks `:119`; `headerButtonId` `:131,138-140,145,156-157`; import `RHSStatusSetting` `:17`; selectors `'selectors'` `:23` | Export `setupUILater`; `'./selectors'`; add helper import; insert gated App Bar **between** `:114` and `:116`; delete headerButtonId scaffolding |
| `webapp/src/plugin.test.ts` | Create | n/a | **Gate 8.1–8.3** registry mock |
| `webapp/src/components/rhs/rhs.tsx` | Modify | boot `:77-100`; root `:193-196` | Import popout helper; `data-rhs-popout`; **do not change boot order** |
| `webapp/src/components/rhs/rhs.test.tsx` | Extend | Gate 8 loading≠empty stays | Append popout attribute + boot-order test |
| `webapp/src/actions/rhs.test.ts` | Extend | Gate 7 names stay | Append Gate 8.5 rehydrate + fetch |
| `webapp/src/utils/rhs_view_state.ts` | **Do not touch** | persist API | W2 done |
| `webapp/src/actions/rhs.ts` | **Do not touch** | `persistCurrentRHSView` `:59-66`; `restoreRHSViewState` `:110-122` | W2 done |
| `webapp/src/components/admin_console/**` | **Do not touch** | W5 | |
| `plugin.json` | **Do not touch** | `EnableJiraRHS` already declared | |
| `server/**` | **Do not touch** | Milestone 1 | |

Every new file stays under 650 non-blank lines. `plugin.tsx` stays far
under.

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
- `test` is the **full** Jest suite when declaring W4 done. Filtering
  `plugin.test.ts` / `rhs_register.test.ts` while iterating is fine.
- `check-types`: fail only if a W4 file appears.

Do not run `make test` (that also runs Go). W4 does not touch Go.

Gate 8 manual checks:

```bash
rg -n "registerAdminConsoleCustomSetting" webapp/src/plugin.tsx
```

Exactly one call, argument `'RHSStatusTabs'`, **not** inside
`if (settings.ui_enabled)` or `if (shouldRegisterJiraRHS` /
`settings.rhs_enabled`.

```bash
rg -n "registerAppBarComponent" webapp/src
```

Production call sites: `utils/rhs_register.ts` and the `plugin.tsx`
block that calls `registerJiraAppBar`. No second RHS/App Bar register.

```bash
rg -n "isPopoutWindow|WebappUtils.popouts|registerRHSPluginPopoutListener" webapp/src
```

Must be **empty**.

```bash
rg -n "FormattedMessage|en\\.json|eslint-disable" webapp/src/utils/rhs_register.ts webapp/src/utils/rhs_popout.ts webapp/src/plugin.tsx webapp/src/components/rhs/rhs.tsx
```

Must be empty in production lines.

```bash
git diff 1e50575 -- webapp/src/plugin.tsx
```

Must **keep** the `RHSStatusSetting` import and the ungated
`registerAdminConsoleCustomSetting` line. Must **add** the App Bar `if`
**above** `ChannelSubscriptionsModal`, not around the admin line. Must
**remove** `headerButtonId` scaffolding.

---

## Definition of Done

- [ ] `rhs_enabled` false → **no** `registerAppBarComponent` (Gate 8.1
      test name above)
- [ ] `rhs_enabled` true + Server/DC only → **no**
      `registerAppBarComponent` (Gate 8.2)
- [ ] `rhs_enabled` true + **cloud-oauth-only** → **does**
      `registerAppBarComponent` with combined `rhsComponent` +
      `rhsTitle`, `action` omitted (Gate 8.3)
- [ ] W5’s `registerAdminConsoleCustomSetting('RHSStatusTabs', …)` still
      runs when the App Bar gate is off
- [ ] App Bar is **not** inside `if (settings.ui_enabled)`
- [ ] Gate runs **after** `getConnected()` and uses `hasCloudInstance`
      (installed), not connected-only
- [ ] Combined `registerAppBarComponent`; no extra
      `registerRightHandSidebarComponent`; no channel-header fallback
- [ ] Icon URL is `getPluginServerRoute` + `/public/icon.svg`;
      `public/icon.svg` exists
- [ ] `{instance, tab, sort}` persist remains W2 `saveRHSViewState`
      (no second writer; no issues in the blob)
- [ ] Popout detect is `pathname` prefix `/_popout/` only; both 11.3 and
      11.6 shapes; no team/channel parse
- [ ] Simulated `/_popout/` rehydrates view state and **re-fetches**
      issues (Gate 8.5)
- [ ] No `isPopoutWindow` / `WebappUtils.popouts` /
      `registerRHSPluginPopoutListener`
- [ ] Dead `headerButtonId` scaffolding removed from `plugin.tsx`
- [ ] `plugin.tsx` selectors import is `'./selectors'` so `plugin.test.ts` can load `setupUILater`
- [ ] Function `Rhs` still registered via `components/rhs` default
- [ ] No `en.json`, no `FormattedMessage`, no `eslint-disable`
- [ ] `npm run lint` and `npm run test` (full suite) pass
- [ ] `check-types` adds no new-file errors

---

## Commit checkpoint (orchestration, not this engineer)

Do not commit. The lead commits the W4 workstream after Gate 8 review
and after confirming `plugin.tsx` still has W5’s ungated admin
register. Suggested message when they do: Register the Jira tickets RHS
on the App Bar behind rhs_enabled and Cloud, with popout rehydrate via
localStorage. Do not push.

---

## Implementation Summary

Implemented by IE10 on 2026-08-19. No commit. No push. W6 not started.

### Files

**Created**
- `public/icon.svg` — byte-for-byte copy of `assets/icon.svg`
- `webapp/src/utils/rhs_popout.ts` + `rhs_popout.test.ts`
- `webapp/src/utils/rhs_register.ts` + `rhs_register.test.ts`
- `webapp/src/plugin.test.ts` — Gate 8.1–8.3 registry mocks

**Modified**
- `webapp/src/plugin.tsx` — named-export `setupUILater`; selectors import is `'./selectors'`; gated `registerJiraAppBar` sibling of `ui_enabled`, after `getConnected()`, before `ChannelSubscriptionsModal`; deleted `headerButtonId` scaffolding
- `webapp/src/components/rhs/rhs.tsx` — `data-rhs-popout`; boot order unchanged
- `webapp/src/components/rhs/rhs.test.tsx` — popout attribute + boot-order test
- `webapp/src/actions/rhs.test.ts` — Gate 8.5 persist/rehydrate/re-fetch
- `webapp/package.json` — `^selectors$` Jest mapper (see deviations)

**Not touched:** W5 admin setting, `actions/rhs.ts`, `utils/rhs_view_state.ts`, `server/**`, `en.json`

### W5 admin line

Unchanged and ungated. `git diff 1e50575 -- webapp/src/plugin.tsx` keeps:

```
registry.registerAdminConsoleCustomSetting('RHSStatusTabs', RHSStatusSetting, {showTitle: true});
```

It is not inside `if (settings.ui_enabled)` or `if (shouldRegisterJiraRHS…)`. Gate 8 tests assert it still runs when App Bar does not register.

### Registration

- Combined `registerAppBarComponent(iconUrl, omittedAction, 'Jira', omittedSupportedProductIds, Rhs, 'Jira')`
- Gate: `rhs_enabled === true` AND `hasCloudInstance` (Cloud JWT **and** cloud-oauth-only)
- Gate runs after `getConnected()`
- Not nested in `ui_enabled`
- No `registerRightHandSidebarComponent`, no channel-header fallback

### Tests

`cd webapp && npm run lint && npm run test` — pass (193 tests).

Gate 8 names present:
- `setupUILater does not call registerAppBarComponent when rhs_enabled is false`
- `setupUILater does not call registerAppBarComponent when only Server/DC is installed`
- `setupUILater calls registerAppBarComponent for a cloud-oauth-only installation when rhs_enabled is true`
- Admin setting still registered in the negatives
- `a simulated /_popout/ pathname rehydrates the persisted view state and triggers a fresh issues fetch`

`check-types`: no new W4 files. Remaining `plugin.tsx` errors are the pre-existing dispatch/`PluginRegistry`/React ones. The new `shouldRegisterJiraRHS(settings, …)` call is cast so it does not add a diagnostic.

### Deviations

1. **Omitted `action` / `supportedProductIds`:** the plan’s `undefined` literal fails `no-undefined`; `void 0` fails `no-void`; no `eslint-disable`. Passed `([] as any[])[0]`, which is runtime `undefined`. Tests assert `toBeUndefined()`.
2. **Jest `^selectors$` mapper:** required so `plugin.test.ts` / `rhs_register.test.ts` can load connected `components/rhs` and other files that still use the webpack bare specifier. `plugin.tsx` still imports `'./selectors'` as specified.
3. **`settings as {rhs_enabled?: boolean}`** at the gate call so `check-types` does not add a new error (`dispatch` is typed as `Action<object>`).
4. **Popout pathname stub** uses `history.pushState` — jsdom cannot redefine `window.location`.

### Blockers

None.
