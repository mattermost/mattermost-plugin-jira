# Phase W5 Plan: System Console status picker

> Prescriptive implementation plan for **Phase W5 only** (tasks W5.1–W5.3). An
> Implementation Engineer should be able to land this without making design
> decisions. Do not implement later phases from this file.
>
> **Do not write production TS/JS until this plan is followed as written.**
> This document is the plan; it is not the code.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase W5 (source of truth for WHAT)
- **Handoff:** `.planning/PHASE1_HANDOFF.md` — **D2 is load-bearing. Read it twice.**
- **W1 / W2:** already landed. Start from **`cb5ec3a`** (W2 commit). Depends on
  W1/W2 only. Do not wait for W3/W4.
- **Orchestration:** Step 8 (W5 half) / **Gate 8 picker negatives** of
  `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `cb5ec3a` (W2). Do not rebase onto an older SHA. Do not
  implement against uncommitted W3/W4 work.
- **Package:** `webapp/` (Jest 29, React Testing Library 14, TypeScript 5.7
  strict, ESLint 8)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer.** Sequential. Nothing in this phase is
  parallelizable. Runs **in parallel with W3+W4** at the orchestration level —
  those agents own other files. Your only shared file is `plugin.tsx`.

## Scope

**In scope:**

- `webapp/src/components/admin_console/rhs_status_setting/rhs_status_options.ts`
  — **create** (pure helpers: empty-value detection, virtual seed display,
  option encoding, persist-map builder). This is where D2 lives.
- `webapp/src/components/admin_console/rhs_status_setting/rhs_status_options.test.ts`
  — **create** (seed / persist / option-build unit tests)
- `webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.tsx`
  — **create** (function component: Cloud instance selector + status
  multi-select)
- `webapp/src/components/admin_console/rhs_status_setting/index.ts` — **create**
  (`connect()` wiring)
- `webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.test.tsx`
  — **create** (**Gate 8 lives here**)
- `webapp/src/plugin.tsx` — **modify by a few lines only**: one import + one
  `registerAdminConsoleCustomSetting` call. See W5.3 and the collision note.

**Out of scope (do not do these):**

- Any RHS panel UI (`webapp/src/components/rhs/`)
- App Bar / popout / `rhs_enabled` gating (W4)
- An `rhsStatuses` Redux slice (W2 Surprise G: `fetchRHSStatuses` returns
  `{data}|{error}` and does **not** dispatch)
- Debouncing `fetchRHSStatuses` (W2: statuses are not `/search/jql`)
- Editing `doFetch` / `doFetchWithResponse` / `getRHSStatuses` / `actions/rhs.ts`
- Editing `plugin.json` (`RHSStatusTabs` is already declared)
- Anything under `server/`
- `en.json` / `FormattedMessage`
- `eslint-disable` comments
- Copying TeamFilter’s `config.PluginSettings.Plugins[...]` read
- Copying TeamFilter’s uncaught fetch
- Calling `onChange` / `setSaveNeeded` to materialize the virtual seed
- Committing or pushing

---

## Research (read this before coding)

Line numbers are as of **`cb5ec3a`**. If they have drifted, search; do not
invent a different design.

### 1. D2 — virtual seed (LOAD-BEARING)

From `.planning/PHASE1_HANDOFF.md` **D2** and parent W5.3 / W-D3.

Milestone 1 **does not write** plugin config on instance install. Seeding is
read-time only (`server/rhs_types.go:41-46`, `server/rhs_jql.go:86-92`):

```41:46:server/rhs_types.go
func rhsDefaultTabs() []RHSTabEntry {
	return []RHSTabEntry{
		{Kind: RHSTabKindAssigned, Name: "Assigned"},
		{Kind: RHSTabKindCategory, Key: statusCategoryKeyIndeterminate, Name: "In Progress"},
	}
}
```

```86:92:server/rhs_jql.go
func resolveTabs(configured []RHSTabEntry, statuses []*JiraStatus, categories []*JiraStatusCategory) []RHSTabEntry {
	out := []RHSTabEntry{{Kind: RHSTabKindAssigned, Name: "Assigned"}}

	extras := configured
	if len(configured) == 0 {
		extras = rhsDefaultTabs()[1:]
	}
```

Facts that the picker must honor:

1. **Assigned is never stored.** `resolveTabs` prepends it and `continue`s any
   saved `kind: assigned`. The persisted map holds **only extra**
   category/status tabs.
2. `len(configured) == 0` (nil **or** empty slice) re-applies In Progress.
   There is **no** way to persist “Assigned only.” Do not invent a sentinel
   tab to work around that — it is a Milestone 1 constraint (Surprise C).
3. A never-configured instance therefore arrives as an **empty `props.value`**:
   `undefined` / `null` / `{}` / missing key / empty array for that instance.
4. The picker **renders** Assigned (locked) + In Progress as selected when
   that instance’s stored extras are empty.
5. The picker **must not** call `onChange` or `setSaveNeeded` to write that
   seed. Doing so dirties the System Console form on mere page load and can
   write config the server does not need.
6. While `not_connected`: keep stored (or seed) chips visible, disable the
   status control, **no `onChange`**. A Save of an unrelated setting would
   otherwise write an empty list over the admin’s tabs.
7. Connect JWT (`cloud`) statuses resolve through the bot client — no personal
   connection. Only `cloud-oauth` can return `not_connected`.

Gate 8 (orchestration Step 8, W5 half) is these **negatives**. They are silent
in production. Each needs an explicit “did not happen” assertion.

### 2. `props.value` vs TeamFilter’s `config` path — W-D3

Two shipping precedents disagree.

**Copy this (custom-attributes):** initialize from `props.value`. Constructor
at `mattermost-plugin-custom-attributes` `custom_attribute_settings.jsx:36-43`
(fetched 2026-08-19 from `master`):

```javascript
constructor(props) {
    super(props);
    this.state = {
        attributes: this.initAttributes(props.value),
        ...
    };
}
initAttributes(attributes) {
    if (!attributes) {
        return new Map();
    }
    return new Map(attributes.map((a, index) => [index, a]));
}
```

`onChange(this.props.id, …)` + `setSaveNeeded()` run only from user
`handleChange` / `handleDelete` — **never** from the constructor. `config` is
declared on propTypes and **never read for selection**.

**Do not copy this (user-survey TeamFilter):**

```27:64:~/git/mattermost-plugin-user-survey/webapp/src/components/systemConsole/teamFilter/teamFilter.tsx
    useEffect(() => {
        const task = async () => {
            const teams: Team[] = await Client4.getTeams(0, 10000, false) as Team[];
            ...
            const savedSetting = config.PluginSettings.Plugins['com.mattermost.user-survey']?.systemconsolesetting.TeamFilter;
            ...
            setSelectedTeams(initialOptions);
            setInitialSetting(id, optionsToConfig(initialOptions));
        };
        task();
    }, [config.PluginSettings.Plugins, id, setInitialSetting]);
```

Three TeamFilter mistakes, all load-bearing for W5:

| Mistake | File:line | Why it is fatal here |
|---------|-----------|----------------------|
| Selection from a **hardcoded plugin-id config path** | `teamFilter.tsx:46` | Parent W-D3. Our `id` is already scoped. Hardcoding `jira` / `com.mattermost.jira` / `PluginSettings.Plugins[...]` is a bug even if the path happens to work today. |
| **Uncaught fetch** | `teamFilter.tsx:28-61` — `task()` has no `try/catch` | A failed load leaves options **and** selection empty. If we then `onChange` that empty list, Save clobbers stored tabs. Context.md:239-241. |
| `setInitialSetting` on mount | `teamFilter.tsx:60` | A user-survey parent invention (`systemConsole/index.tsx:48-53`). It exists to materialize defaults so a later sibling save has a full object. **That is exactly the D2 violation.** We have no sibling sub-settings and must not grow a wrapper that writes defaults on mount. |

TeamFilter **does** call `onChange(id, value)` **and** `setSaveNeeded()`
together on every **user** selection (`teamFilter.tsx:71-74`). Do that — but
only from the status multi-select’s user handler.

`rg -n "PluginSettings\\.Plugins" webapp/src/components/admin_console` must be
**empty** when W5 is done.

### 3. What `props.value` / `props.id` actually are

Verified against `~/git/mattermost` `schema_admin_settings.tsx` +
`custom_plugin_settings/index.ts` (as of local master).

Plugin schema keys are rewritten before they reach the component
(`custom_plugin_settings/index.ts:50-76`):

```
setting.key.toLowerCase()  →  'rhsstatustabs'
full key                   →  'PluginSettings.Plugins.' + escapedPluginId + '.' + key
```

`manifest.id` is `'jira'` (`webapp/src/manifest.ts:5`). `jira` has no dots, so
no escaping. The component therefore receives:

| Prop | Value |
|------|--------|
| `id` | `'PluginSettings.Plugins.jira.rhsstatustabs'` |
| `value` | the **native JSON object** `Record<instanceId, RHSTab[]>` or `undefined` if never saved |
| `onChange` | `(id, nativeObject) => void` — **no stringify** (context.md:105) |

`buildCustomSetting` (`schema_admin_settings.tsx:963-978`) passes
`value={this.state[setting.key]}` and `onChange={this.handleChange}`.
`handleChange` (`:798-826`) **already sets `saveNeeded: 'config'`** and writes
`state[id] = value`. Calling `onChange` **is** marking the form dirty.

`getConfigFromState` (`:1525-1542`) writes `state[setting.key]` back onto the
cloned config for **every** schema key on Save, including untouched ones.
Structured values survive an unrelated Save **if and only if**
`state[setting.key]` still holds the original map. If we `onChange` an empty
object because a fetch failed, Save writes that empty object. That is the
TeamFilter clobber.

**Always call `onChange(props.id, nextMap)`.** Never
`onChange('RHSStatusTabs', …)` and never a hardcoded
`PluginSettings.Plugins[...]` path.

Registration key stays `'RHSStatusTabs'` (platform lowercases it:
`reducers/plugins/index.ts:375`). The manifest entry already exists
(`plugin.json:169-172`). Without that key the console would not render the
component at all — do not re-declare it.

### 4. This plugin’s `ReactSelectSetting` — `webapp/src/components/react_select_setting.tsx`

Prefer this over TeamFilter’s unthemed SCSS wrapper (context.md:242-245). It
already themes via `getStyleForReactSelect(theme)` (`utils/styles.ts:38-135`).

Load-bearing API (`react_select_setting.tsx:55-67, 134-142`):

- `isMulti` works. Existing caller: `edit_channel_subscription.tsx:502-522`
  (`events` / `issue_types`).
- **`value` must be option object(s), not raw strings.** The subscription
  modal does `eventOptions.filter((option) => this.state.filters.events.includes(option.value))`.
- `onChange` received by the **parent** is `(name: string, values: string[])`
  for multi — `handleChange` maps `value.map((x) => x.value)` and **drops
  `actionMeta`**. We cannot see `remove-value` / `pop-value`. Lock Assigned in
  **our** handler by re-inserting it if missing (Surprise F).
- `{...this.props}` is spread onto `<ReactSelect>` **before** the wrapper
  overrides `onChange` / `styles`. We **can** pass `isDisabled`, `isClearable`,
  `isLoading`, `components`, `isMulti`.
- Do **not** pass `limitOptions` — that path assumes a flat `options` array
  (`filterOptions` at `:70-76`) and would break grouped category/status
  options.
- Do **not** edit `ReactSelectSetting`. It is shared.

`theme` is required. Existing `connect()` files in this plugin **do not** pass
it (pre-existing gap; `getStyleForReactSelect` no-ops on a missing theme at
`styles.ts:39-41`). W5 **does** pass it: `getTheme` from
`mattermost-redux/selectors/entities/preferences` (`~/git/mattermost`
`preferences.ts:184`). Tests may use `mockTheme` from `testlib/test-utils.tsx`.

### 5. Status option shape and `getStatusField` dedupe

Statuses endpoint (handoff + W2 `getRHSStatuses`):

```
GET {baseUrl}/api/v2/rhs/statuses?instance_id=...
{ statuses: [{id, name, statusCategory:{id,key,name}}], categories: [{id,key,name}] }
```

Category `id` is a JSON **number**; status `id` is a JSON **string**. Real
Cloud keys: `undefined` / `new` / `indeterminate` / `done`. Spec: **never
offer “No Category”** (`key === 'undefined'`) as a tab.

Dedupe statuses by id, same idiom as `getStatusField`
(`webapp/src/utils/jira_issue_metadata.tsx:209-248`):

```209:225:webapp/src/utils/jira_issue_metadata.tsx
export function getStatusField(...) {
    const keys = new Set<string>();
    const statuses: Status[] = [];
    ...
                    if (!keys.has(status.id)) {
                        keys.add(status.id);
                        statuses.push(status);
                    }
```

Reuse the *idea*, not the function — `getStatusField` reads
`issue_types_with_statuses` (per-project). Our list is already instance-wide.

Option `value` strings **must** encode kind, because a status can be named
“In Progress” too:

| Kind | `option.value` | Stored extra |
|------|----------------|--------------|
| Assigned (display only) | `assigned` | **never stored** |
| Category | `category:<key>` e.g. `category:indeterminate` | `{kind:'category', key, name}` |
| Status | `status:<id>` e.g. `status:10001` | `{kind:'status', id, name}` |

Grouped `react-select` options: `Categories` then `Statuses`. Assigned is
always in the **selected** value (locked), not a persistable extra.

Vanished stored extras (id/key not in the fetch) still render as chips —
synthesize an option from the stored `RHSTab`, same idea as TeamFilter’s
`Archived Team: ${teamId}` (`teamFilter.tsx:52`) but **without** reading
`config`.

### 6. `fetchRHSStatuses` from W2 — `webapp/src/actions/rhs.ts:222-230`

```222:230:webapp/src/actions/rhs.ts
export const fetchRHSStatuses = (instanceID: string) => {
    return async (dispatch: Dispatch, getState: () => GlobalState) => {
        try {
            const data = await getRHSStatuses(getPluginServerRoute(getState()), instanceID);
            return {data};
        } catch (error) {
            return {error: toRHSFetchError(error)};
        }
    };
};
```

Already re-exported from `actions/index.ts:694-705`. Call it via `connect` +
`bindActionCreators`. It **does not throw** on HTTP failure — it returns
`{error: RHSFetchError}` with `errorCode`. Still wrap the dispatch in
`try/catch` (TeamFilter pitfall). Do **not** dispatch into Redux. Do **not**
add a statuses slice.

`RHSFetchError` is `webapp/src/client/index.ts:71-81`. Compare
`error.errorCode === 'not_connected'`. Parse `error`, never `message`.

The route is **admin-gated** (`server/rhs_http.go:90-94`) and is **not** gated
on `rhs_enabled` (handoff). The picker must work while the feature flag is
off.

### 7. Instance list — installed Cloud, not connected Cloud

W1’s `getConnectedCloudInstances` (`selectors/index.ts:98-100`) is for the
**RHS user** picker (connected ∩ installed ∩ Cloud). Wrong here.

The admin configures tabs for the **site**. Connect JWT works with **no**
personal connection (`server/rhs.go:54-65`). Filter
`getInstalledInstances` with existing `isCloudInstance`
(`selectors/index.ts:77-87`). Include both `cloud` and `cloud-oauth`. Omit
`server`.

Refresh on mount with `getConnected()` — same precedent as
`jira_instance_and_project_selector.tsx:57-76`. `setupUILater` already called
it once (`plugin.tsx:48`); calling again closes the stale-instance window.
A failed refresh must **not** `onChange`.

Instance option label: `instance.alias || instance.instance_id`
(`jira_instance_and_project_selector.tsx:131-132`).

Always show the instance selector when there is ≥1 Cloud instance (the admin
needs to see which instance they are editing). Auto-select the first Cloud
instance. Zero Cloud instances: empty copy, no status control, no `onChange`.

### 8. `plugin.tsx` `setupUILater` and the W4 collision

```41:125:webapp/src/plugin.tsx
const setupUILater = (registry: any, store: Store<object, Action<object>>): () => Promise<void> => async () => {
    registry.registerReducer(reducers);
    const settings = await store.dispatch(getSettings());
    ...
    try {
        await getConnected()(store.dispatch, store.getState);

        if (settings.ui_enabled) {
            // W4 will likely add a *sibling* `if (settings.rhs_enabled && hasCloudInstance)`
            // next to this block (parent W4.1 mirrors :50).
            ...
        }

        registry.registerRootComponent(ChannelSubscriptionsModal);  // :115  UNGATED

        const hooks = new Hooks(store, settings);                   // :117
        registry.registerSlashCommandWillBePostedHook(...);
    } finally {
        // websocket only — do not register UI here
    }
};
```

**Exact W5 insertion (only legal `plugin.tsx` body edit):**

After `:115` `registry.registerRootComponent(ChannelSubscriptionsModal);`
and **before** `:117` `const hooks = new Hooks(store, settings);` insert
exactly:

```typescript
        registry.registerAdminConsoleCustomSetting('RHSStatusTabs', RHSStatusSetting, {showTitle: true});
```

Why this spot:

- **Ungated.** Sits with `ChannelSubscriptionsModal`, which is already
  registered outside `ui_enabled`. Parent W5.2: register unconditionally so
  an admin can configure tabs **before** enabling the feature.
- **Not in `finally`.** That block is websocket handlers only.
- **Not next to `:50`.** That is where W4 will add the App Bar / RHS
  `if (settings.rhs_enabled && hasCloudInstance)` block. Keep W5 out of that
  neighborhood so a rebase is one-line, not a semantic merge.

**Exact W5 import edit:** after `:16`
`import ChannelSubscriptionsModal from 'components/modals/channel_subscriptions';`
add:

```typescript
import RHSStatusSetting from 'components/admin_console/rhs_status_setting';
```

Import the **connected default** from `index.ts`, not the raw function
component.

Do **not** touch `headerButtonId`, `initialize()`, the `ui_enabled` block, or
`finally`. W4 owns those.

**Collision note for implementers / rebase:** W4 (not started) will also edit
`setupUILater()` to register App Bar + RHS. If W4 lands first and inserts near
`:115`, keep this one `registerAdminConsoleCustomSetting` line **outside** any
`rhs_enabled` / `ui_enabled` / `hasCloudInstance` gate. If both insert after
`:115`, order is: ChannelSubscriptionsModal, **W5 admin setting**, (optional
W4 block), then `const hooks`. Accept one small manual merge. This is the
only milestone-2 collision (parent File Change Map).

`{showTitle: true}` makes the console wrap the component in `Setting` using
`plugin.json` `display_name` (“Jira RHS status tabs”)
(`schema_admin_settings.tsx:983-992`). Do **not** also render `props.label`
or the title will duplicate.

### 9. ESLint / i18n / test constraints (same as W2)

| Rule | W5 impact |
|------|-----------|
| `header/header` | two-line copyright on every new file |
| `import/order` | blank line between groups |
| `import-newlines/enforce` | wrap at **4+** named imports |
| `object-curly-spacing` | `"never"` |
| `react/jsx-no-literals` | **error**. Every visible string is `{'…'}` or a `const` |
| `react/jsx-max-props-per-line` | 1 — inflates JSX. Split files rather than suppress |
| `react/no-multi-comp` | `ignoreStateless: true` — small **function** helpers may share a file |
| `max-lines` | 650, counts comments | keep helpers in `rhs_status_options.ts` |
| `no-undefined` | `null`, not `undefined` |
| `no-underscore-dangle` | no `_fetchGen` — use `fetchGeneration` |
| Function components | parent plan; do not write a class component |

**Do not add `en.json` or `FormattedMessage`.** i18n is milestone F1.

`node_modules` exists. Do not `npm install` unless a binary is missing.
`check-types` still reports **249 pre-existing `error TS`**. Fail W5 only if a
**new** file appears in that list.

### 10. Gate 8 (must be in this phase’s tests)

From `IMPL_ORCHESTRATION_PLAN.md` Step 8 / parent W5 DoD / the user brief:

1. When `value` is empty, render Assigned (locked) + In Progress as selected
   **without** calling `onChange` (or `setSaveNeeded`).
2. While disconnected: chips stay, control disabled, **no `onChange`**.
3. Restore selection from `props.value`, not a hardcoded
   `config.PluginSettings.Plugins[...]` path — poison-`config` test.
4. A failed / empty fetch must not clobber stored config on Save (no
   `onChange` from the fetch path).

Also required by parent DoD (user-positive, still this phase):

5. Assigned cannot be deselected (handler no-ops / re-inserts; chip has no X).
6. Selecting a status calls `onChange` **and** `setSaveNeeded` with the
   correct structured map (extras only; Assigned omitted).
7. Only Cloud instances are offered.

---

## Surprises

### Surprise A — `props.id` is not `'RHSStatusTabs'`

The framework rewrites the key to
`PluginSettings.Plugins.jira.rhsstatustabs`. Always use `props.id` as the
first `onChange` argument. Register with `'RHSStatusTabs'` (platform
lowercases).

### Surprise B — `onChange` already dirties the form

`schema_admin_settings.handleChange` sets `saveNeeded`. Calling `onChange` on
mount **is** the D2 bug, even if you skip `setSaveNeeded`. Still call **both**
on a real user edit (parent DoD + TeamFilter / custom-attributes).

### Surprise C — Assigned-only cannot be persisted

`resolveTabs` treats `[]` and missing as the virtual seed. If the user
deselects In Progress and leaves only Assigned, write `{[instanceId]: []}`
(that **is** a user edit — `onChange` is correct) and accept that a reload
will show In Progress again. Do not invent a dummy stored tab.

When the computed extras **equal the virtual seed** (only In Progress
category) **and** that instance is already empty in `props.value`, do **not**
`onChange` (re-selecting the seed must stay silent).

### Surprise D — do not use `getConnectedCloudInstances`

JWT admins may have zero personal connections and must still configure that
instance. Filter `getInstalledInstances` with `isCloudInstance`.

### Surprise E — do not copy `setInitialSetting`

It is a user-survey wrapper that materializes defaults on mount. D2 forbids
that. We are a single custom setting; `props.value` is sufficient.

### Surprise F — `ReactSelectSetting` drops `actionMeta`

Lock Assigned by (1) hiding `MultiValueRemove` when `data.isFixed` and
(2) re-inserting `assigned` in our `(name, values)` handler. `isClearable={false}`.

### Surprise G — `plugin.tsx` is the W4 collision

W5 edit is **one import + one register line** after
`ChannelSubscriptionsModal`. W4 owns the `rhs_enabled` gate near `:50`. If
you rebase onto W4, do not slide the admin registration inside that `if`.

### Surprise H — W2 already shipped the fetch

`fetchRHSStatuses` / `getRHSStatuses` / `RHSFetchError` exist. Do not
re-implement HTTP. Do not add an `rhsStatuses` slice.

### Surprise I — `check-types` is dirty on this branch

249 pre-existing errors. Ignore unless a W5 file is in the list.

### Surprise J — selection is a pure function of `props.value`

Do **not** keep selected tabs in `useState`. Derive chips with
`displayTabsForInstance(props.value, instanceID)`. Fetch populates **options
and error only**. This is the mechanical guarantee that a failed fetch cannot
clobber selection or call `onChange`.

---

## Tasks

### W5.1 — Pure helpers

**File:** `webapp/src/components/admin_console/rhs_status_setting/rhs_status_options.ts`
**Action:** Create

Copyright header, then exactly these exports. Keep this file free of React.

```typescript
import {
    Instance,
    InstanceType,
    RHSErrorCode,
    RHSStatus,
    RHSStatusesResponse,
    RHSTab,
} from 'types/model';

export type RHSStatusTabsValue = Record<string, RHSTab[]>;

export type StatusTabOption = {
    label: string;
    value: string;
    isFixed?: boolean;
    tab: RHSTab;
};

export const ASSIGNED_TAB: RHSTab = {
    kind: 'assigned',
    name: 'Assigned',
};

export const IN_PROGRESS_TAB: RHSTab = {
    kind: 'category',
    key: 'indeterminate',
    name: 'In Progress',
};

export const ASSIGNED_OPTION_VALUE = 'assigned';

export const ASSIGNED_OPTION: StatusTabOption = {
    label: ASSIGNED_TAB.name,
    value: ASSIGNED_OPTION_VALUE,
    isFixed: true,
    tab: ASSIGNED_TAB,
};

export const CATEGORY_OPTION_PREFIX = 'category:';
export const STATUS_OPTION_PREFIX = 'status:';

export const NOT_CONNECTED_MESSAGE = 'Connect Jira to change status tabs.';
export const STATUS_TABS_HELP = 'Choosing a category name (To Do, In Progress, Done) matches statusCategory and is broader than a single status.';
export const INSTANCE_LABEL = 'Instance';
export const STATUS_TABS_LABEL = 'Status tabs';
export const NO_CLOUD_INSTANCE_MESSAGE = 'Install a Jira Cloud instance to configure RHS status tabs.';
export const UNABLE_TO_LOAD_STATUSES_MESSAGE = 'Unable to load statuses.';

export function isCloudInstalledInstance(instance: Instance): boolean {
    switch (instance.type) {
    case InstanceType.CLOUD:
    case InstanceType.CLOUD_OAUTH:
        return true;
    case InstanceType.SERVER:
        return false;
    default: {
        const exhaustive: never = instance.type;
        return exhaustive;
    }
    }
}

export function filterInstalledCloudInstances(instances: Instance[] | null): Instance[] {
    if (!instances) {
        return [];
    }
    return instances.filter(isCloudInstalledInstance);
}

export function isTabsValueEmpty(value: RHSStatusTabsValue | null, instanceID: string): boolean {
    if (!value || typeof value !== 'object') {
        return true;
    }
    const extras = value[instanceID];
    return !extras || extras.length === 0;
}

export function storedExtrasForInstance(value: RHSStatusTabsValue | null, instanceID: string): RHSTab[] {
    if (isTabsValueEmpty(value, instanceID)) {
        return [];
    }
    return value[instanceID].filter((tab) => tab.kind !== 'assigned');
}

export function displayTabsForInstance(value: RHSStatusTabsValue | null, instanceID: string): RHSTab[] {
    if (isTabsValueEmpty(value, instanceID)) {
        return [ASSIGNED_TAB, IN_PROGRESS_TAB];
    }
    return [ASSIGNED_TAB, ...storedExtrasForInstance(value, instanceID)];
}

export function optionValueForTab(tab: RHSTab): string {
    switch (tab.kind) {
    case 'assigned':
        return ASSIGNED_OPTION_VALUE;
    case 'category':
        return CATEGORY_OPTION_PREFIX + (tab.key || '');
    case 'status':
        return STATUS_OPTION_PREFIX + (tab.id || '');
    default: {
        const exhaustive: never = tab.kind;
        return exhaustive;
    }
    }
}

export function tabFromOptionValue(value: string, options: StatusTabOption[]): RHSTab | null {
    const match = options.find((option) => option.value === value);
    if (match) {
        return match.tab;
    }
    if (value === ASSIGNED_OPTION_VALUE) {
        return ASSIGNED_TAB;
    }
    if (value.indexOf(CATEGORY_OPTION_PREFIX) === 0) {
        const key = value.slice(CATEGORY_OPTION_PREFIX.length);
        if (!key) {
            return null;
        }
        return {
            kind: 'category',
            key,
            name: key,
        };
    }
    if (value.indexOf(STATUS_OPTION_PREFIX) === 0) {
        const id = value.slice(STATUS_OPTION_PREFIX.length);
        if (!id) {
            return null;
        }
        return {
            kind: 'status',
            id,
            name: id,
        };
    }
    return null;
}

export function optionsFromTabs(tabs: RHSTab[]): StatusTabOption[] {
    return tabs.map((tab) => {
        if (tab.kind === 'assigned') {
            return ASSIGNED_OPTION;
        }
        return {
            label: tab.name,
            value: optionValueForTab(tab),
            tab,
        };
    });
}

export function extrasFromOptionValues(values: string[], options: StatusTabOption[]): RHSTab[] {
    const extras: RHSTab[] = [];
    for (let i = 0; i < values.length; i++) {
        const value = values[i];
        if (value === ASSIGNED_OPTION_VALUE) {
            continue;
        }
        const tab = tabFromOptionValue(value, options);
        if (tab && tab.kind !== 'assigned') {
            extras.push(tab);
        }
    }
    return extras;
}

export function isVirtualSeedExtras(extras: RHSTab[]): boolean {
    if (extras.length !== 1) {
        return false;
    }
    const tab = extras[0];
    return tab.kind === 'category' && tab.key === IN_PROGRESS_TAB.key;
}

export function buildPersistedValue(
    current: RHSStatusTabsValue | null,
    instanceID: string,
    extras: RHSTab[],
): RHSStatusTabsValue {
    const next: RHSStatusTabsValue = current && typeof current === 'object' ? {...current} : {};
    if (isVirtualSeedExtras(extras) && isTabsValueEmpty(current, instanceID)) {
        delete next[instanceID];
        return next;
    }
    next[instanceID] = extras;
    return next;
}

export function persistedValuesEqual(left: RHSStatusTabsValue | null, right: RHSStatusTabsValue): boolean {
    return JSON.stringify(left || {}) === JSON.stringify(right);
}

export function dedupeStatusesById(statuses: RHSStatus[]): RHSStatus[] {
    const keys = new Set<string>();
    const out: RHSStatus[] = [];
    for (let i = 0; i < statuses.length; i++) {
        const status = statuses[i];
        if (!keys.has(status.id)) {
            keys.add(status.id);
            out.push(status);
        }
    }
    return out;
}

export type StatusOptionGroup = {
    label: string;
    options: StatusTabOption[];
};

export function buildStatusOptionGroups(data: RHSStatusesResponse): StatusOptionGroup[] {
    const categories = data.categories.
        filter((category) => category.key !== 'undefined').
        map((category): StatusTabOption => {
            const tab: RHSTab = {
                kind: 'category',
                key: category.key,
                name: category.name,
            };
            return {
                label: category.name,
                value: optionValueForTab(tab),
                tab,
            };
        });

    const statuses = dedupeStatusesById(data.statuses).map((status): StatusTabOption => {
        const tab: RHSTab = {
            kind: 'status',
            id: status.id,
            name: status.name,
        };
        return {
            label: status.name,
            value: optionValueForTab(tab),
            tab,
        };
    });

    return [
        {label: 'Categories', options: categories},
        {label: 'Statuses', options: statuses},
    ];
}

export function flattenOptionGroups(groups: StatusOptionGroup[]): StatusTabOption[] {
    const out: StatusTabOption[] = [ASSIGNED_OPTION];
    for (let i = 0; i < groups.length; i++) {
        out.push(...groups[i].options);
    }
    return out;
}

export function statusFetchMessage(code: RHSErrorCode): string {
    switch (code) {
    case 'not_connected':
        return NOT_CONNECTED_MESSAGE;
    case 'rate_limited':
        return 'Jira is rate limiting requests, try again shortly';
    case 'not_authorized':
        return 'not authorized';
    case 'not_cloud':
        return 'Jira RHS is available for Jira Cloud only';
    case 'invalid_request':
    case 'internal_error':
        return UNABLE_TO_LOAD_STATUSES_MESSAGE;
    default: {
        const exhaustive: never = code;
        return exhaustive;
    }
    }
}
```

Do **not** import `selectors` from this file — Jest has no `^selectors$`
mapper, and W1 already taught that lesson. Duplicate the Cloud check with an
exhaustive `switch` over `InstanceType` (same cases as
`selectors/index.ts:77-87`). `filterInstalledCloudInstances` uses this local
helper.

### W5.2 — Presentational component + `connect`

**Files:**
`webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.tsx`
(create),
`webapp/src/components/admin_console/rhs_status_setting/index.ts` (create)

#### Props (exact)

```typescript
export type Props = {
    id: string;
    value: RHSStatusTabsValue | null;
    disabled: boolean;
    config: unknown;
    onChange: (id: string, value: RHSStatusTabsValue) => void;
    setSaveNeeded: () => void;
    theme: Theme;
    installedInstances: Instance[];
    fetchRHSStatuses: (instanceID: string) => Promise<{data?: RHSStatusesResponse; error?: RHSFetchError}>;
    getConnected: () => Promise<{data?: unknown; error?: unknown}>;
};
```

`config` is accepted **and never read**. That is the W-D3 lock. Extra
System Console props (`label`, `helpText`, `registerSaveAction`, `license`,
…) may arrive at runtime; do not put them on `Props` unless you use them.
`{showTitle: true}` consumes `label` / `helpText` in the wrapper. Do **not**
call `registerSaveAction` (side effects at save, not config writes —
context.md:225-226).

#### `rhs_status_setting.tsx` behavior (function component)

Local state **only** for:

- `instanceID: string` (empty string default; `no-undefined`)
- `optionGroups: StatusOptionGroup[]` (default `[]`)
- `fetchError: RHSErrorCode | null` (default `null`)
- `loadingStatuses: boolean`
- `fetchGeneration: number` (stale-response guard; increment on each fetch
  start, ignore results whose generation does not match)

**Do not** store selected tabs in state. Chips =

```typescript
const cloudInstances = filterInstalledCloudInstances(installedInstances);
const selectedTabs = instanceID ? displayTabsForInstance(value, instanceID) : [ASSIGNED_TAB];
const selectedOptions = optionsFromTabs(selectedTabs);
const flatOptions = flattenOptionGroups(optionGroups);
```

Mount:

1. `getConnected()` inside `try/catch`. Ignore the error. **No `onChange`.**
2. After instances are available, if `instanceID` is empty and
   `cloudInstances[0]` exists, `setInstanceID(cloudInstances[0].instance_id)`.

When `instanceID` changes (including the auto-select):

3. Increment generation, set `loadingStatuses` true, `fetchError` null
   (do **not** clear chips — they derive from `value`).
4. `try { result = await fetchRHSStatuses(instanceID) } catch { treat as
   internal_error }`.
5. If generation mismatch, return.
6. If `result.error`: set `fetchError` to `result.error.errorCode`, leave
   `optionGroups` as-is or `[]`. **Do not call `onChange` / `setSaveNeeded`.**
   **Do not** replace `selectedTabs` (they are not in state).
7. If `result.data`: `setOptionGroups(buildStatusOptionGroups(result.data))`,
   `setFetchError(null)`.
8. `setLoadingStatuses(false)` in a `finally` that still respects generation.

Instance `<ReactSelectSetting>`:

- `name='rhs-status-instance'`
- `label={INSTANCE_LABEL}`
- `options={cloudInstances.map(... alias || instance_id ...)}`
- `value={that list find current}`
- `isMulti={false}`
- `isClearable={false}`
- `isDisabled={disabled}`
- `theme={theme}`
- `onChange` sets `instanceID` only. **No `props.onChange`.**

Status `<ReactSelectSetting>`:

- `name='rhs-status-tabs'`
- `label={STATUS_TABS_LABEL}`
- `helpText={STATUS_TABS_HELP}`
- `isMulti={true}`
- `isClearable={false}`
- `isLoading={loadingStatuses}`
- `isDisabled={disabled || !instanceID || fetchError !== null}`
- `options={optionGroups}` (grouped)
- `value={selectedOptions}`
- `theme={theme}`
- `components={{MultiValueRemove: RHSFixedMultiValueRemove}}` where
  `RHSFixedMultiValueRemove` is a function component in this file
  (`ignoreStateless: true`): if `props.data.isFixed` return `null`; else
  return `<components.MultiValueRemove {...props} />` (`components` imported
  from `react-select`).
- `onChange` → `handleStatusValuesChange(name, values: string[])`.

`handleStatusValuesChange`:

1. If `disabled` or `fetchError !== null` or `!instanceID`: **return**.
2. Ensure `ASSIGNED_OPTION_VALUE` is present in `values` (re-unshift if the
   user removed it). If after that the extras equal
   `storedExtrasForInstance(value, instanceID)` when value is non-empty, or
   equal the virtual seed when value is empty: **return** (Assigned lock /
   seed no-op).
3. `extras = extrasFromOptionValues(values, [...flatOptions, ...selectedOptions])`.
4. `next = buildPersistedValue(value, instanceID, extras)`.
5. If `persistedValuesEqual(value, next)`: **return**.
6. `onChange(id, next)` then `setSaveNeeded()`.

Render the fetch error as help text under the status control using
`statusFetchMessage(fetchError)` when `fetchError !== null`.
`jsx-no-literals`: `{'…'}` or the consts from `rhs_status_options.ts`.

Zero Cloud instances: render `NO_CLOUD_INSTANCE_MESSAGE` only. No
`onChange`.

One JSX prop per line. No `eslint-disable`. No `FormattedMessage`.

#### `index.ts` (`connect`, copy `jira_instance_and_project_selector/index.ts`)

```typescript
import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getTheme} from 'mattermost-redux/selectors/entities/preferences';

import {fetchRHSStatuses, getConnected} from '../../../actions';
import {getInstalledInstances} from '../../../selectors';

import {GlobalState} from 'types/store';

import RHSStatusSetting from './rhs_status_setting';

const mapStateToProps = (state: GlobalState) => {
    return {
        installedInstances: getInstalledInstances(state),
        theme: getTheme(state),
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    fetchRHSStatuses,
    getConnected,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(RHSStatusSetting);
```

`getTheme` exists in current Mattermost
(`mattermost-redux/.../preferences.ts:184`) and is the right source for
`ReactSelectSetting`. If `check-types` flags the import in this **new** file
only, fall back to `Preferences.THEMES.denim` from
`mattermost-redux/constants/preferences` (already used in
`create_issue_form.test.tsx:30`) and leave a one-line comment that theme
theming then matches tests, not the admin’s theme. Do not expand the fallback
into a custom theme loader.

Jest maps `^actions$` and `^selectors$` is **not** mapped — `jira_instance_and_project_selector/index.ts`
uses relative `../../selectors`. Use **relative** `../../../selectors` and
`../../../actions` from
`components/admin_console/rhs_status_setting/index.ts` so Jest can load the
connected module if a test imports it. Production webpack resolves both.

Depth: `webapp/src/components/admin_console/rhs_status_setting/index.ts` →
`../../../actions`, `../../../selectors`, `../../../types/store`.

### W5.3 — `plugin.tsx` registration (few lines)

**File:** `webapp/src/plugin.tsx`
**Action:** Modify — **only** the import + the one register call.

1. After the `ChannelSubscriptionsModal` import (`:16`), add the
   `RHSStatusSetting` import (see §8).
2. After `registry.registerRootComponent(ChannelSubscriptionsModal);`
   (`:115`), add the `registerAdminConsoleCustomSetting` line (see §8).

`git diff cb5ec3a -- webapp/src/plugin.tsx` must show **only** those two
hunks. If you see `headerButtonId`, `ui_enabled`, websocket, or App Bar
edits, you have drifted into W4.

---

## Tests

New files, copyright header, relative imports, no `eslint-disable`, no
`en.json`. Test the **presentational** component (`./rhs_status_setting`),
not the connected default — same as
`jira_instance_and_project_selector.test.tsx`.

Shared fixtures (inline):

```typescript
const SETTING_ID = 'PluginSettings.Plugins.jira.rhsstatustabs';
const CLOUD_ID = 'https://cloud.example.atlassian.net';
const OAUTH_ID = 'https://oauth.example.atlassian.net';
const SERVER_ID = 'http://jira.example.com';

const cloudInstance = {instance_id: CLOUD_ID, type: InstanceType.CLOUD};
const oauthInstance = {instance_id: OAUTH_ID, type: InstanceType.CLOUD_OAUTH};
const serverInstance = {instance_id: SERVER_ID, type: InstanceType.SERVER};

const poisonConfig = {
    PluginSettings: {
        Plugins: {
            jira: {
                rhsstatustabs: {
                    [CLOUD_ID]: [{kind: 'status', id: '999', name: 'FromConfig'}],
                },
            },
            'com.mattermost.user-survey': {
                systemconsolesetting: {
                    TeamFilter: {filteredTeamIDs: ['nope']},
                },
            },
        },
    },
};

const statusesData: RHSStatusesResponse = {
    categories: [
        {id: 1, key: 'undefined', name: 'No Category'},
        {id: 2, key: 'new', name: 'To Do'},
        {id: 4, key: 'indeterminate', name: 'In Progress'},
        {id: 3, key: 'done', name: 'Done'},
    ],
    statuses: [
        {id: '3', name: 'In Progress', statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'}},
        {id: '3', name: 'In Progress', statusCategory: {id: 4, key: 'indeterminate', name: 'In Progress'}},
        {id: '10001', name: 'Submitted', statusCategory: {id: 2, key: 'new', name: 'To Do'}},
    ],
};
```

`baseProps`:

```typescript
const baseProps: Props = {
    id: SETTING_ID,
    value: null,
    disabled: false,
    config: poisonConfig,
    onChange: jest.fn(),
    setSaveNeeded: jest.fn(),
    theme: mockTheme as Theme,
    installedInstances: [cloudInstance, oauthInstance, serverInstance],
    fetchRHSStatuses: jest.fn().mockResolvedValue({data: statusesData}),
    getConnected: jest.fn().mockResolvedValue({data: {}}),
};
```

Default `config` **is the poison object** so every test that forgets W-D3
fails if the component reads it.

`beforeEach`: `jest.clearAllMocks()`.

Use `renderWithRedux` + `act`/`waitFor` like the instance-selector tests.
After mount, `waitFor` `fetchRHSStatuses` **or** `getConnected` so effects
flush before asserting `onChange`.

### `rhs_status_options.test.ts` (create)

Required names:

1. **`isTabsValueEmpty treats null undefined empty object missing key and empty array as empty`**
2. **`displayTabsForInstance returns Assigned and In Progress when value is empty`**
3. **`displayTabsForInstance prepends Assigned to stored extras and drops a stored assigned entry`**
4. **`buildPersistedValue does not materialize the virtual seed when the instance is already empty`**
   — `buildPersistedValue(null, CLOUD_ID, [IN_PROGRESS_TAB])` equals `{}`
   (no `CLOUD_ID` key).
5. **`buildPersistedValue writes extras without Assigned`**
   — extras `[IN_PROGRESS_TAB, {kind:'status', id:'10001', name:'Submitted'}]`
   → `{[CLOUD_ID]: those two}` and no `kind: 'assigned'`.
6. **`buildStatusOptionGroups omits No Category and dedupes statuses by id`**
   — no option whose `tab.key === 'undefined'`; only one `status:3`.
7. **`filterInstalledCloudInstances drops SERVER and keeps cloud and cloud-oauth`**
8. **`statusFetchMessage returns the connect copy for not_connected`**

### `rhs_status_setting.test.tsx` (create) — Gate 8 lives here

**Names are load-bearing for Gate 8 review.**

1. **`empty value renders Assigned and In Progress selected without calling onChange`**
   — **Gate 8.1 / D2.** `value={null}` (also run a second `it` or
   `it.each` for `undefined` **skipped — `no-undefined`**, `{}`, and
   `{[CLOUD_ID]: []}`). After `waitFor` fetch settle:
   `getByText('Assigned')`, `getByText('In Progress')`,
   `expect(onChange).not.toHaveBeenCalled()`,
   `expect(setSaveNeeded).not.toHaveBeenCalled()`.
   Must **not** render `FromConfig` (poison `config`).

2. **`not_connected keeps stored chips disables the status control and does not call onChange`**
   — **Gate 8.2.**
   `value={{[CLOUD_ID]: [{kind:'category', key:'new', name:'To Do'}]}}`.
   `fetchRHSStatuses.mockResolvedValue({error: new RHSFetchError('not_connected', 'Jira account is not connected', 401)})`.
   After settle: `getByText('Assigned')`, `getByText('To Do')`,
   `getByText(NOT_CONNECTED_MESSAGE)`,
   status combobox disabled (`getByLabelText(STATUS_TABS_LABEL)` /
   the react-select input has `isDisabled` — assert
   `expect(screen.getByText(NOT_CONNECTED_MESSAGE)).toBeInTheDocument()`
   **and** that the instance control is **not** the thing showing the
   connect copy).
   `expect(onChange).not.toHaveBeenCalled()`.
   `expect(setSaveNeeded).not.toHaveBeenCalled()`.
   Chips must still be the **stored** To Do, not a cleared list and not
   `FromConfig`.

3. **`selection follows props.value not a hardcoded config PluginSettings path`**
   — **Gate 8.3.**
   `value={{[CLOUD_ID]: [{kind:'status', id:'10001', name:'Submitted'}]}}`
   with the same `poisonConfig`. After settle: `getByText('Assigned')`,
   `getByText('Submitted')`, **no** `FromConfig`.
   `expect(onChange).not.toHaveBeenCalled()`.
   This fails if anyone copied TeamFilter’s
   `config.PluginSettings.Plugins['jira']` (or user-survey) read.

4. **`a failed fetch with empty options does not call onChange`**
   — **Gate 8.4 / TeamFilter pitfall.**
   `value={{[CLOUD_ID]: [{kind:'category', key:'done', name:'Done'}]}}`.
   `fetchRHSStatuses.mockRejectedValue(new Error('network down'))`
   **or** `mockResolvedValue({data: {statuses: [], categories: []}})` — run
   **both** as two tests if cheap; at minimum the **throw** path (TeamFilter
   has no try/catch) and the **empty 200** path.
   After settle: chips still `Assigned` + `Done`,
   `expect(onChange).not.toHaveBeenCalled()`.

5. **`Assigned cannot be deselected`**
   — Call `handleStatusValuesChange` the way ReactSelectSetting would:
   render, then use `userEvent` to click the Assigned remove control if it
   exists (`queryByLabelText` / `queryByRole('button')` near Assigned).
   Prefer: **Assigned’s `MultiValueRemove` is absent**
   (`queryByText` / container query — `isFixed` render returns `null`).
   Also invoke the change path by exporting nothing extra: click a selected
   chip’s X on In Progress (allowed) and assert `onChange` **was** called;
   there is no X on Assigned. If RTL cannot find the hidden remove, add
   `data-testid='rhs-assigned-chip'` on a wrapper only if needed — prefer
   asserting `MultiValueRemove` is not in the Assigned chip.

6. **`selecting a status calls onChange and setSaveNeeded with extras only`**
   — Open the status select (`userEvent.click` on the Status tabs
   combobox) and click `Done`.
   `expect(onChange).toHaveBeenCalledTimes(1)`.
   First arg is `SETTING_ID` (the full PluginSettings path), **not**
   `'RHSStatusTabs'`.
   Second arg is
   `{[CLOUD_ID]: [IN_PROGRESS_TAB, {kind:'category', key:'done', name:'Done'}]}`
   when `value` was empty (virtual seed + Done). No `kind: 'assigned'` in
   the payload. `setSaveNeeded` called.

7. **`only Cloud instances are offered`**
   — After mount, the instance control shows Cloud + OAuth labels and
   **does not** show `SERVER_ID` / `jira.example.com`.

8. **`getConnected is called on mount and a getConnected failure does not call onChange`**
   — `getConnected.mockRejectedValue(new Error('nope'))`.
   After settle: `expect(getConnected).toHaveBeenCalled()`,
   `expect(onChange).not.toHaveBeenCalled()`.

If `userEvent` + react-select portal makes test 6 flaky, you may drive it
by finding the `ReactSelectSetting` `onChange` through a tiny test-only
prop **do not add**. Instead, keep test 6 as RTL if possible; the payload
shape is already locked by `rhs_status_options.test.ts` item 5. Gate 8
review cares most about tests **1–4**. Do not drop 1–4 to save time.

Do **not** `jest.mock` `../client`. `fetchRHSStatuses` is a **prop**.

---

## File-by-file change list (current line numbers)

Line numbers as of `cb5ec3a`.

| File | Action | Current lines | What to do |
|------|--------|---------------|------------|
| `webapp/src/components/admin_console/rhs_status_setting/rhs_status_options.ts` | Create | n/a | D2 helpers, option encoding, persist map |
| `webapp/src/components/admin_console/rhs_status_setting/rhs_status_options.test.ts` | Create | n/a | Seed / persist / dedupe / Cloud filter |
| `webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.tsx` | Create | n/a | Function component; derive chips from `props.value` |
| `webapp/src/components/admin_console/rhs_status_setting/index.ts` | Create | n/a | `connect`: instances + `getTheme` + thunks |
| `webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.test.tsx` | Create | n/a | **Gate 8.1–8.4** + Assigned lock + Cloud filter |
| `webapp/src/plugin.tsx` | Modify | import `:16`; `setupUILater` `:41-125`; `ui_enabled` `:50`; ChannelSubscriptionsModal `:115`; hooks `:117`; `headerButtonId` `:129` | **Import after `:16`. Register after `:115`.** Nothing else. |
| `webapp/src/actions/rhs.ts` | **Do not touch** | `fetchRHSStatuses` `:222-230` | W2 done |
| `webapp/src/client/index.ts` | **Do not touch** | `getRHSStatuses` `:160-165` | W2 done |
| `webapp/src/selectors/index.ts` | **Do not touch** | `isCloudInstance` `:77-87`; `getConnectedCloudInstances` `:98-100` | Filter in the helper; do not add a selector |
| `webapp/src/plugin.json` / `plugin.json` | **Do not touch** | `RHSStatusTabs` `:169-172` | Already declared |
| `webapp/src/components/rhs/**` | **Do not touch** | — | W3 |
| `server/**` | **Do not touch** | D2 already shipped | |

---

## Existing tests that must be re-run

Run the **full** Jest suite (`npm run test`, no path filter).

W5 is additive UI + `plugin.tsx` registration. Existing suites should not
change. W2 baseline after `cb5ec3a`: **17 suites / 119 tests**. W5 adds 2
suites (`rhs_status_options` + `rhs_status_setting`). Extra tests are fine;
missing Gate 8 names are not.

---

## Commands

From `webapp/` (or `cd webapp && …` from the worktree root).

```bash
cd webapp
npm run lint
npm run test
npm run check-types
```

- `lint` must be 0 errors. No `eslint-disable`.
- `test` is the **full** suite when declaring Gate 8 done. Filtering
  `rhs_status_setting.test.tsx` while iterating is fine.
- `check-types`: still **249** pre-existing errors. Fail only if a W5 file
  appears.

Do not run `make test` (that also runs Go). W5 does not touch Go.

Gate 8 manual checks after tests:

```bash
git diff cb5ec3a -- webapp/src/plugin.tsx
```

Only the import + `registerAdminConsoleCustomSetting` hunks.

```bash
rg -n "PluginSettings\\.Plugins|com\\.mattermost\\.user-survey|setInitialSetting" webapp/src/components/admin_console
```

Must be empty (the **test** file may mention `PluginSettings` inside
`poisonConfig` / `SETTING_ID` — that is the assertion input, not a production
read). Production `.ts`/`.tsx` except `*.test.*` must be empty:

```bash
rg -n "PluginSettings\\.Plugins|com\\.mattermost\\.user-survey|setInitialSetting" webapp/src/components/admin_console --glob '!*test.*'
```

```bash
rg -n "FormattedMessage|en\\.json|eslint-disable" webapp/src/components/admin_console webapp/src/plugin.tsx
```

Must be empty in the new/edited production lines.

```bash
rg -n "registerAdminConsoleCustomSetting" webapp/src/plugin.tsx
```

Exactly one call, argument `'RHSStatusTabs'`, **not** inside
`if (settings.ui_enabled)` or a `rhs_enabled` gate.

---

## Definition of Done

- [ ] Empty `props.value` (`null` / `{}` / missing instance key / `[]`)
      renders Assigned (locked) + In Progress and **`onChange` /
      `setSaveNeeded` are not called** — Gate 8.1 test name above
- [ ] `not_connected` keeps stored chips, disables the status control, shows
      `Connect Jira to change status tabs.`, **no `onChange`** — Gate 8.2
- [ ] Poison `config.PluginSettings.Plugins[...]` is ignored; chips follow
      `props.value` — Gate 8.3
- [ ] Fetch throw **and** empty 200 do not `onChange` — Gate 8.4
- [ ] Assigned cannot be deselected (no remove control / handler re-inserts)
- [ ] User-selecting a status calls `onChange(props.id, map)` **and**
      `setSaveNeeded`; payload extras omit Assigned; `id` is the full
      `PluginSettings.Plugins.jira.rhsstatustabs` string
- [ ] Instance selector lists `cloud` + `cloud-oauth` only
- [ ] Categories omit `undefined` / No Category; statuses deduped by id
- [ ] Help text notes that a category name matches `statusCategory`
- [ ] `plugin.tsx` diff is import + one ungated
      `registerAdminConsoleCustomSetting('RHSStatusTabs', …, {showTitle: true})`
      after `ChannelSubscriptionsModal`
- [ ] No `rhsStatuses` slice; no `plugin.json` / `server/**` / RHS UI edits
- [ ] No `PluginSettings.Plugins` read in production admin files
- [ ] No `en.json`, no `FormattedMessage`, no `eslint-disable`
- [ ] `npm run lint` and `npm run test` (full suite) pass

---

## Commit checkpoint (orchestration, not this engineer)

Do not commit. The lead commits the W5 workstream after Gate 8 review and
after reconciling `plugin.tsx` with W4. Suggested message when they do:
System Console RHS status picker with silent virtual-seed and
not-connected guards. Do not push.

---

## Implementation Summary

**Engineer:** IE9  
**Date:** 2026-08-19  
**Status:** complete — not committed, not pushed

### What landed

System Console status picker for `RHSStatusTabs`. Chips are derived from
`props.value` only (D2 virtual seed is display-only). Fetch populates options
and error. Empty value, `not_connected`, poison `config`, and failed/empty
fetch never call `onChange` / `setSaveNeeded`.

### Files

| File | Action |
|------|--------|
| `webapp/src/components/admin_console/rhs_status_setting/rhs_status_options.ts` | Create |
| `webapp/src/components/admin_console/rhs_status_setting/rhs_status_options.test.ts` | Create |
| `webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.tsx` | Create |
| `webapp/src/components/admin_console/rhs_status_setting/index.ts` | Create |
| `webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.test.tsx` | Create |
| `webapp/src/plugin.tsx` | Modify — one import + one ungated register |

Did **not** touch `actions/rhs.ts`, `client/index.ts`, `plugin.json`,
`server/**`, or `webapp/src/components/rhs/**`.

### `plugin.tsx` diff vs `cb5ec3a`

Exactly two hunks. No `headerButtonId` / `ui_enabled` / App Bar / RHS edits.
No unexpected local W4 edits were present; no conflict to report.

```diff
+import RHSStatusSetting from 'components/admin_console/rhs_status_setting';
```

```diff
         registry.registerRootComponent(ChannelSubscriptionsModal);
+        registry.registerAdminConsoleCustomSetting('RHSStatusTabs', RHSStatusSetting, {showTitle: true});
```

Registration sits with `ChannelSubscriptionsModal` (outside `ui_enabled` /
any `rhs_enabled` gate).

### Gate 8 tests (exact names)

All in `rhs_status_setting.test.tsx`:

1. `empty value renders Assigned and In Progress selected without calling onChange` — `null` / `{}` / `{[id]: []}` via `test.each`
2. `not_connected keeps stored chips disables the status control and does not call onChange`
3. `selection follows props.value not a hardcoded config PluginSettings path`
4. `a failed fetch with empty options does not call onChange` (throw) plus `an empty 200 fetch does not call onChange`

Also: Assigned lock, user-select `onChange(SETTING_ID, extras)` +
`setSaveNeeded`, Cloud-only instances, `getConnected` failure silent.

### Commands

- `cd webapp && npm run lint` — 0 errors
- `cd webapp && npm run test` — **24 suites / 176 tests** passed
- W5 files do not appear in `tsc` output after local type adapters
- Production admin path has no `PluginSettings.Plugins` / `setInitialSetting` read
- No `FormattedMessage` / `en.json` / `eslint-disable` in W5 production lines

### Deviations (no behavior redesign)

- Leading `.filter` / `.map` dots (`dot-location: property`) instead of the plan’s trailing-dot style
- `storedExtrasForInstance` adds a `!value` guard so TS can narrow; same empty behavior
- `ConsoleSelect` is a type-only alias of this plugin’s `ReactSelectSetting` because its declared Props omit `label` / `helpText` (still the same runtime component)
- `index.ts` types `dispatch` and casts `bindActionCreators` through `unknown` so the new file stays off `check-types`
- `inputId` matches `name` so Setting’s `htmlFor` labels the react-select input
- `fetchGeneration` state plus a ref so incrementing generation cannot retrigger the fetch effect
- Extra empty-200 test name besides the required throw-path Gate 8.4 name

### Worktree note

W3 RHS test files already exist in this worktree from other agents. IE9 did
not implement or edit W3/W4 UI. `plugin.tsx` had no W4 App Bar / RHS
registration when W5 landed.

### Blockers

None.
