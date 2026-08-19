# Implementation Plan: Show your tickets in the RHS — Milestone 2 (Webapp + Admin UI)

> The user-visible half: App Bar entry, the RHS panel with tabs/sort/rows/load-more,
> the System Console status picker, and popout support. Consumes the Milestone 1 server
> API and ships the feature.

## Metadata

- **Spec:** `planner/projects/jira-plugin-rhs/ideas/001-show-tickets-rhs/spec.md` (§ "Phase 2 — Webapp and admin UI (milestone 2)")
- **Target repo:** https://github.com/mattermost/mattermost-plugin-jira
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Generated:** 2026-08-18
- **Status:** draft
- **Depends on:** `.planning/PLAN.md` (Milestone 1, server foundation) complete and merged
- **Phase ids:** `W1`–`W6` to avoid collision with Milestone 1's `1`–`5`

## What Milestone 1 hands over

This plan assumes the server already provides:

- `rhs_enabled` on the settings-info response, read the same way `ui_enabled` is today.
- `GET /api/v2/rhs/issues` — params `instance_id`, tab identity, `sort` (`updated`|`created`),
  optional `next_page_token`; page size 20. Returns `{issues, tabs, nextPageToken, isLast}`
  where `issues` is the normalized DTO and `tabs` is the **server-resolved** tab list with
  vanished entries already hidden.
- `GET /api/v2/rhs/statuses` — admin-gated instance-wide status list for the picker.
- A JSON error body on both: `{"error": "not_connected"|"rate_limited"|"not_authorized"|"not_cloud"|"invalid_request"|"internal_error", "message": "..."}`.
- `RHSStatusTabs` declared in `plugin.json` as a `type: "custom"` setting — **the key already
  exists**, so this milestone only registers the component against it.

**The one constraint Milestone 1 imposes:** seeding is *virtual*. The server stores nothing
for a never-configured instance and treats absent config as Assigned + In Progress. So the
picker will receive an **empty `value`** for such an instance and must render those two as
selected itself — see W5.3. This is load-bearing: if the component instead calls `onChange`
to materialize the seed, it violates the "never `onChange` while disconnected" rule and can
clobber config on an unrelated Save.

## Architecture Overview

`webapp/src/` organizes components as a directory per component with `index.ts` (the
`connect()` wiring) plus a `component.tsx`, following
`components/jira_instance_and_project_selector/` and
`components/modals/channel_subscriptions/`. The RHS follows the same shape.

New component tree under `webapp/src/components/rhs/`, plus one admin component under
`webapp/src/components/admin_console/`. Redux gains an RHS slice group; `client/index.ts`
gains one new fetch helper; `plugin.tsx` gains the registration calls.

**Decomposition is forced by lint, not chosen.** `max-lines` is 650 counting comments
(`.eslintrc.json:190-197`), and `react/jsx-max-props-per-line` is 1, which inflates JSX
line counts substantially. `react/no-multi-comp` is set with `ignoreStateless: true`
(`.eslintrc.json:520-525`), so small presentational **function** components may share a
file — only class components must be isolated. Writing the RHS as function components
therefore buys real freedom; this plan assumes function components throughout.

### Decisions this plan makes

**W-D1 — The popout store is `localStorage`.** This closes the last Phase 2 blocking
question in the spec. View state here is `{instance, tab, sort}` — ephemeral, per-browser
UI state, not a preference worth syncing across devices. A Mattermost user preference
would cost an API round trip on every tab switch and sort change to buy cross-device
continuity nobody asked for. Key the entry by user id so switching accounts in one browser
does not inherit the previous user's view.

**W-D2 — Keep persist-don't-transfer for popout, despite new evidence.** The spec chose
persistence partly because `sendToPopout` had no production consumers. That is still true
of the Mattermost webapp itself, but it is **no longer true of first-party plugins**: the
GitHub plugin ships the popout-initiated handshake at
`mattermost-plugin-github/webapp/src/index.js:74-91` — the popout sends `GET_RHS_STATE`,
the parent replies `SEND_RHS_STATE`, and the popout also re-fetches its own data. That is
exactly the "fallback" shape the spec described as correct-but-unproven, and it is now
proven. It does not change the decision, because persistence still avoids the 11.3 version
floor entirely and `min_server_version` is 10.7 — but it does mean the message-API route is
a validated upgrade path rather than a gamble, and it is worth recording as such.

Note also that the GitHub plugin guards with `window.WebappUtils.popouts &&
window.WebappUtils.popouts.isPopoutWindow()`. We use a `/_popout/` pathname check instead
because `window.WebappUtils.popouts` does not exist below 11.3 and our floor is 10.7.

**W-D3 — The picker restores selection from `props.value`, not from `config`.** Two
precedents exist and they differ. `mattermost-plugin-custom-attributes` initializes from
`props.value` (`custom_attribute_settings.jsx`, constructor). `mattermost-plugin-user-survey`'s
`TeamFilter` instead reaches into the `config` prop and reads
`config.PluginSettings.Plugins['com.mattermost.user-survey']...` on mount
(`teamFilter.tsx:27-64`). `props.value` is the framework-intended path, is already scoped to
our setting, and does not hardcode the plugin id into a config path. Use it. The fetch
supplies *options only* and never selection.

**W-D4 — The server owns tab resolution; the client renders what it is given.** The
`/rhs/issues` response includes the resolved `tabs` array with vanished entries already
hidden and the virtual seed already applied. The webapp does not re-derive tabs from
config. This keeps the #36 validation and the status cache on one side of the wire.

## Reference implementation notes

The GitHub plugin RHS is the structural reference, but it is 2019-era and several of its
choices are actively worth avoiding. From the read of
`mattermost-plugin-github/webapp/src/components/sidebar_right/`:

**Borrow:** Redux-held view state with the active tab as a plain string slice; the
container/presentational split; theme-derived inline styles via `makeStyleFromTheme`;
websocket-push-instead-of-polling refresh (relevant to milestone F2, not this one).

**Do not borrow:**
- Its empty state doubles as its loading state — `'You have no active items'` renders
  while data is in flight (`github_items.tsx:311`). Our states must be distinct.
- It has **no error state at all**. A failed details fetch shows stale data with no signal.
- It has no loading indicator for its second-phase fetch; fields just appear.
- Long titles wrap unbounded with no truncation, which in a ~400px panel makes very tall
  rows.
- Its "tabs" are icon buttons in the left team sidebar, not an in-RHS tab strip, and it
  predates the App Bar. Our tab strip is genuinely net-new UI.
- `SidebarRight` is a class component; write function components.

---

## Phases

### Phase W1: Types, selectors, and fixtures

**Goal:** the webapp can correctly identify Cloud instances, and the test fixtures stop
silently lying. Nothing renders yet.

**Depends on:** Milestone 1 complete.

#### Tasks

- [ ] **W1.1 Add `CLOUD_OAUTH` to the instance type enum**
  - **Files:** `webapp/src/types/model.ts`, `webapp/src/actions/index.ts`
  - **Action:** Modify
  - **Details:** `InstanceType` (`webapp/src/types/model.ts:184-192`) declares only
    `CLOUD = 'cloud'` and `SERVER = 'server'`, while the server serializes
    `type: "cloud-oauth"` (`server/instance.go:62-68`). Every `type === InstanceType.CLOUD`
    check therefore silently excludes **every OAuth2 Cloud instance** — the modern Cloud
    auth path. Add `CLOUD_OAUTH = 'cloud-oauth'`.
    Adding the variant will break the existing switch over instance type at
    `webapp/src/actions/index.ts:576` — that is the point, and it is the compiler doing
    the work. Handle the new case, and give the `default` branch a `never` check so any
    future variant fails at compile time rather than falling through silently.

- [ ] **W1.2 Cloud-aware selectors**
  - **Files:** `webapp/src/selectors/index.ts`
  - **Action:** Extend
  - **Details:** Add `hasCloudInstance(state)` — true when any **installed** instance is
    `CLOUD` or `CLOUD_OAUTH` — and `getConnectedCloudInstances(state)`, which narrows the
    existing `getUserConnectedInstances` (`webapp/src/selectors/index.ts:65-73`) to Cloud
    types only. The former gates App Bar registration; the latter drives the RHS instance
    picker, which must list **connected Cloud** instances only.
    Keep the existing cross-filter behavior of `getUserConnectedInstances` — it intersects
    `userConnectedInstances` against `installedInstances`, which is what makes an
    uninstalled instance self-heal out of the list once the websocket lands.

- [ ] **W1.3 Fix the test fixtures**
  - **Files:** `webapp/src/testlib/test-utils.tsx`, `webapp/src/components/modals/channel_subscriptions/edit_channel_subscription.test.tsx`
  - **Action:** Modify
  - **Details:** `defaultMockState` (`webapp/src/testlib/test-utils.tsx:27`) uses the key
    `connectedInstances`, but the reducer slice is `userConnectedInstances`
    (`webapp/src/reducers/index.ts:62`), so `getUserConnectedInstances()` returns `[]` in
    every test that uses the fixture. Rename the key. Add a second instance of type
    `cloud-oauth` so Cloud-gating tests exercise both Cloud types.
    **Blast radius is wider than previously recorded.** Besides
    `channel_subscription_filter.test.tsx`, which consumes `defaultMockState` directly,
    `edit_channel_subscription.test.tsx:35` declares its own `editChannelMockState` with
    the *same* wrong key and needs the same fix. Four further tests inherit the fixture
    through `renderWithRedux`: `jira_epic_selector.test.tsx`,
    `connect_modal_form.test.tsx`, `disconnect_modal_form.test.tsx`, and
    `create_issue_form.test.tsx`. Run the full Jest suite after the rename — some of those
    may have been passing *because* the selector returned empty.

#### Definition of Done

- [ ] `npm run check-types` passes, including the handled switch at `actions/index.ts:576`.
- [ ] A selector test proves `hasCloudInstance` is true for a `cloud-oauth`-only
      installation — the exact case that is broken today.
- [ ] `defaultMockState` yields a non-empty `getUserConnectedInstances()`.
- [ ] The full existing Jest suite passes after the fixture rename.

---

### Phase W2: Data layer

**Goal:** Redux state, actions, and a JSON-error-aware fetch path. No UI yet, but every
data path the UI needs is testable.

**Depends on:** W1.

#### Tasks

- [ ] **W2.1 A JSON-error-aware fetch helper**
  - **Files:** `webapp/src/client/index.ts`
  - **Action:** Extend
  - **Details:** `doFetchWithResponse` (`webapp/src/client/index.ts:22-42`) reads a non-2xx
    body with `response.text()` and throws a `ClientError` whose message is that raw
    string. Against the new routes that yields a JSON blob as a string and loses the error
    code entirely.
    Add a **new** helper — do not change `doFetch`/`doFetchWithResponse`, since every other
    route depends on the plain-text behavior. The new helper parses a non-2xx body as JSON,
    extracts `error` and `message`, and surfaces a typed
    `{errorCode: RHSErrorCode, message: string}`. Fall back to the text path when the body
    is not JSON, so an unexpected 502 from a proxy does not throw a parse error on top of
    the real failure.
    Add the two RHS fetch functions here alongside it.

- [ ] **W2.2 Action types and reducers**
  - **Files:** `webapp/src/action_types/index.ts`, `webapp/src/reducers/index.ts`
  - **Action:** Modify
  - **Details:** Add slices for view state — `rhsInstanceID`, `rhsTab`, `rhsSort` — and for
    data: `rhsIssues`, `rhsTabs` (server-resolved, per W-D4), `rhsNextPageToken`,
    `rhsIsLast`, `rhsLoading`, and `rhsError` holding the typed error code rather than a
    boolean.
    **Keep loading and empty distinguishable.** The GitHub plugin conflates them and its
    empty state renders during flight; `rhsLoading` exists specifically so ours does not.
    Follow the existing hand-rolled reducer idiom in `webapp/src/reducers/index.ts:269-286`
    — plain `combineReducers`, no Redux Toolkit.

- [ ] **W2.3 Actions, with in-flight debounce**
  - **Files:** `webapp/src/actions/index.ts`
  - **Action:** Extend
  - **Details:** Thunks following the file's existing
    `async (dispatch, getState) => ... return {data} | {error}` idiom:
    `fetchRHSIssues({instanceID, tab, sort})` (resets list and cursor),
    `loadMoreRHSIssues()` (appends using `nextPageToken`),
    `refreshRHSIssues()`, `setRHSInstance/Tab/Sort`, and `fetchRHSStatuses(instanceID)` for
    the admin picker.
    **Debounce in flight per `{instance, tab, sort}`** — never issue a second search for the
    same triple while one is outstanding. This is one of the three locked rate-limit
    mitigations and it is the cheapest one: tab-switch mashing and repeated refresh clicks
    are the realistic burst sources. Track the in-flight key in module scope or a reducer
    slice; do not rely on component state, since the popout is a separate window with a
    separate store and the RHS can unmount mid-flight.
    Changing sort resets list and cursor to page 1; it is not a client-side re-sort.

- [ ] **W2.4 View-state persistence**
  - **Files:** `webapp/src/utils/rhs_view_state.ts`
  - **Action:** Create
  - **Details:** Read/write `{instance, tab, sort}` to `localStorage` per **W-D1**, keyed by
    the current user id so switching accounts in one browser does not inherit stale view
    state. Write on every view-state change; read at RHS boot.
    Wrap both in try/catch — `localStorage` throws in private-browsing modes and when the
    quota is exhausted, and a persistence failure must degrade to "RHS opens on the default
    tab", never to a broken panel.
    Validate on read: an unknown tab id or a sort outside `updated`/`created` falls back to
    defaults. Stored state can outlive an admin's tab-config change.

#### Definition of Done

- [ ] A JSON error body from a non-2xx response yields a typed error code; a non-JSON
      body falls back without throwing.
- [ ] A second `fetchRHSIssues` for an identical `{instance, tab, sort}` while one is in
      flight issues **no** second network call — asserted on the fetch mock's call count.
- [ ] Changing sort resets the cursor; load-more appends rather than replaces.
- [ ] `rhs_view_state` round-trips, tolerates a `localStorage` throw, and rejects an
      invalid stored tab or sort.

---

### Phase W3: The RHS panel

**Goal:** the panel renders correctly against mocked state, including all five terminal
states.

**Depends on:** W2. **Runs in parallel with W5.**

#### Tasks

- [ ] **W3.1 Panel shell and container**
  - **Files:** `webapp/src/components/rhs/index.ts`, `webapp/src/components/rhs/rhs.tsx`
  - **Action:** Create
  - **Details:** `index.ts` does the `connect()` wiring in the established style;
    `rhs.tsx` is the panel. On mount, `dispatch(getConnected())` before deciding what to
    render — following `jira_instance_and_project_selector.tsx:57-76`, the only precedent
    in the plugin for refreshing connection state on open. Nothing polls today, and this
    ~5-line fetch closes the stale-instance window that would otherwise let a ghost entry
    into the picker.
    Also on mount, rehydrate view state (W2.4) before the first fetch, so a popout or a
    reopened RHS lands on the right tab without a visible flip.
    Compose header, tab strip, and list; own no presentation itself.

- [ ] **W3.2 Instance picker and tab strip**
  - **Files:** `webapp/src/components/rhs/rhs_tab_strip.tsx`, `webapp/src/components/rhs/rhs_instance_picker.tsx`
  - **Action:** Create
  - **Details:** The instance picker renders **only** when the user has more than one
    connected Cloud instance (`getConnectedCloudInstances` from W1.2); with exactly one it
    auto-selects and renders nothing. Switching instance reloads the tab strip from that
    instance's resolved tabs and resets the list.
    The tab strip renders the server-resolved `tabs` array (W-D4) with Assigned always
    first. **No tab-count cap** — overflow scrolls horizontally. This deliberately
    overrules the research recommendation of capping at ~4 for a ~400px panel; the
    discoverability cost is accepted, and it is tracked as an open product assumption.
    Make the scroll affordance visible — a strip that scrolls with no visual hint is the
    failure mode here.

- [ ] **W3.3 Header: sort, refresh, New ticket**
  - **Files:** `webapp/src/components/rhs/rhs_header.tsx`
  - **Action:** Create
  - **Details:** Sort dropdown with **Updated** (default) and **Created**; changing it
    resets to page 1 per W2.3. A manual refresh button — refresh is on-open plus manual in
    this milestone; websocket push is future milestone F2. The **New ticket** button
    dispatches the existing `openCreateModalWithoutPost(description, channelId)`
    (`webapp/src/actions/index.ts:62-68`) with an empty description and the current channel
    id. No new modal and no server work for that path.

- [ ] **W3.4 Issue row**
  - **Files:** `webapp/src/components/rhs/rhs_issue_row.tsx`
  - **Action:** Create
  - **Details:** Compact two-line row: type icon, key, summary as link on line one; status
    pill, project, and relative updated time on line two.
    Build the browse URL from the DTO's server-provided value — the server already derives
    it from `GetJiraBaseURL()`, which is the only correct source for cloud-oauth. Do not
    reconstruct it client-side from an instance URL.
    Icons are **local SVGs**; do not use Jira's `iconUrl`, which breaks for custom icons
    under OAuth2 in the browser. Map type by `issuetype.name` with `subtask` /
    `hierarchyLevel` fallbacks.
    Status pill colors derive from the DTO's `statusCategory.key` mapped to Mattermost
    theme CSS variables.
    **Truncate the summary** — clamp to two lines with ellipsis. The GitHub plugin lets
    titles wrap unbounded and long Jira summaries would make rows very tall in a narrow
    panel.

- [ ] **W3.5 The five states**
  - **Files:** `webapp/src/components/rhs/rhs_states.tsx`
  - **Action:** Create
  - **Details:** Five visually distinct states, as stateless function components sharing
    one file (allowed by `ignoreStateless: true`): **loading** (never conflated with empty
    — this is the specific GitHub-plugin mistake we are avoiding), **empty** ("No tickets
    in this tab"), **error** (generic failure with a retry affordance), **not connected**
    (prompting the user to connect their Jira account, driven by the `not_connected` error
    code), and **rate limited** ("Jira is rate limiting requests, try again shortly",
    driven by `rate_limited`). The last two exist only because the Milestone 1 endpoints
    return distinct codes rather than a generic 500; render them from the code, never by
    string-matching a message.

- [ ] **W3.6 Styles**
  - **Files:** `webapp/src/components/rhs/rhs.scss`
  - **Action:** Create
  - **Details:** Follow `components/jira_ticket_tooltip/ticketStyle.scss`, which uses
    Mattermost CSS custom properties (`var(--center-channel-color)`,
    `var(--center-channel-bg)`, `var(--link-color)`, and
    `rgba(var(--center-channel-color-rgb), 0.08)`). Do **not** follow
    `channel_subscriptions_modal.scss`, which hardcodes colors and predates the variable
    convention. Where inline styles are needed, use `makeStyleFromTheme` from
    `mattermost-redux/utils/theme_utils`, which memoizes per theme.

#### Definition of Done

- [ ] All five states render distinctly; loading is never the empty state.
- [ ] The instance picker is absent with one connected Cloud instance and present with two.
- [ ] Sort change resets to page 1; load more appends; refresh refetches.
- [ ] New ticket dispatches `openCreateModalWithoutPost` with the current channel id.
- [ ] A long summary truncates rather than growing the row.
- [ ] `npm run lint` passes with **no** `max-lines` or `no-multi-comp` suppressions.

---

### Phase W4: Registration, gating, and popout

**Goal:** the feature is reachable, correctly gated, and survives being popped out.

**Depends on:** W3.

#### Tasks

- [ ] **W4.1 App Bar and RHS registration**
  - **Files:** `webapp/src/plugin.tsx`
  - **Action:** Modify
  - **Details:** Register inside `setupUILater()` (`webapp/src/plugin.tsx:41-125`), where
    every other real registration happens — not in `initialize()`, which only mounts the
    `SetupUI` probe. Gate on `settings.rhs_enabled` **and** `hasCloudInstance` (W1.2),
    mirroring how the existing block gates on `settings.ui_enabled` at :51.
    Use the single `registerAppBarComponent` call with `rhsComponent` + `rhsTitle`: that
    variant internally calls `registerRightHandSidebarComponent` and wires the icon to
    `toggleRHSPlugin`, so one call registers both surfaces. It is byte-for-byte identical
    in `release-10.7`, so it is safe at the current `min_server_version`.
    **Be aware there is no first-party precedent for the combined form.** Playbooks uses
    the three-argument form plus a separate `registerRightHandSidebarComponent`
    (`mattermost-plugin-playbooks/webapp/src/index.tsx:200,221-222`), and the GitHub plugin
    never calls `registerAppBarComponent` at all. The combined form is verified to exist
    and to work; if it misbehaves, the split-call form is the drop-in fallback and has the
    side benefit of allowing `showPopout` control on 11.9+.
    Remove the dead `headerButtonId` scaffolding (`webapp/src/plugin.tsx:129,138,143,154-155`)
    — it is threaded into `SetupUI` as props that `SetupUI` never reads, and it was
    scaffolded for a channel-header button, a different API we are not using.
    Because the gate lives inside `setupUILater()`, a disabled feature is never registered
    at all rather than registered-and-hidden. That is the existing convention and it is the
    behavior we want.

- [ ] **W4.2 Popout detection and rehydrate**
  - **Files:** `webapp/src/plugin.tsx`, `webapp/src/components/rhs/rhs.tsx`
  - **Action:** Extend
  - **Details:** Detect the popout with
    `window.location.pathname.startsWith('/_popout/')`. Do **not** use `isPopoutWindow()`:
    `window.WebappUtils.popouts` first appears in 11.3 and our floor is 10.7, so the key is
    simply absent on 10.7-11.2. The pathname check is exactly what `isPopoutWindow()` does
    internally and works on every supported version.
    Do not parse team or channel out of that path — the URL shape changed at 11.6, from
    `/_popout/rhs/{team}/{channel}/plugin/{id}` to
    `/_popout/rhs/{team}/plugin/{id}?channel={channel}`.
    On detecting a popout, rehydrate `{instance, tab, sort}` from the store (W2.4) and
    **re-fetch issues**. Never carry an issues payload across the window boundary; it would
    be stale on arrival.
    The popout is a complete new app boot with a fresh Redux store and its own module
    context, so `plugin.initialize()` runs exactly once there and the `loadedPlugins` dedupe
    map is per-window — there is no duplicate-registration hazard.
    Accept that the popout button always renders on 11.3+ and cannot be suppressed under
    App Bar registration on any version. That is fine: we want popout.

#### Definition of Done

- [ ] With `rhs_enabled` false, nothing registers — assert the registry mock received no
      `registerAppBarComponent` call.
- [ ] With `rhs_enabled` true but only Server/DC instances installed, nothing registers.
- [ ] With `rhs_enabled` true and a **`cloud-oauth`-only** installation, registration
      happens. This is the case today's enum gets wrong, so it is the one that matters.
- [ ] A simulated `/_popout/` pathname rehydrates the persisted view state and triggers a
      fresh issues fetch.
- [ ] No reference to `isPopoutWindow` or `WebappUtils.popouts` exists in the new code.

---

### Phase W5: System Console status picker

**Goal:** admins can configure per-instance status tabs, and a disconnected admin cannot
destroy the config by saving.

**Depends on:** W1 and W2. **Runs in parallel with W3 and W4.**

#### Tasks

- [ ] **W5.1 The multi-select component**
  - **Files:** `webapp/src/components/admin_console/rhs_status_setting/rhs_status_setting.tsx`, `.../index.ts`
  - **Action:** Create
  - **Details:** A Cloud-instance selector plus a status multi-select. Closest analogue is
    `mattermost-plugin-user-survey`'s `TeamFilter`
    (`webapp/src/components/systemConsole/teamFilter/teamFilter.tsx`, ~102 lines plus a
    38-line `react-select` wrapper) — an async-loaded multi-select in the System Console.
    Prefer the plugin's own `ReactSelectSetting`
    (`webapp/src/components/react_select_setting.tsx`) with `isMulti`, since it already
    themes react-select through `getStyleForReactSelect(theme)`; `TeamFilter` uses plain
    SCSS and consequently looks slightly out of place in a dark System Console.
    Options come from `fetchRHSStatuses` (W2.3) hitting the admin endpoint, deduplicated by
    id — reuse the dedupe idiom in `getStatusField`
    (`webapp/src/utils/jira_issue_metadata.tsx:209-248`), which already collapses statuses
    by id across issue types into `{label, value}` options.
    Per **W-D3**, initialize selection from `props.value`, not from the `config` prop.
    Only Cloud instances appear in the instance selector; Server/DC is omitted entirely.
    Help text must note that choosing a **category** name (To Do, In Progress, Done) matches
    `statusCategory` and is broader than a single status.

- [ ] **W5.2 Registration**
  - **Files:** `webapp/src/plugin.tsx`
  - **Action:** Modify
  - **Details:** `registry.registerAdminConsoleCustomSetting('RHSStatusTabs', RHSStatusSetting)`
    inside `setupUILater()`. The key is already declared in `plugin.json` by Milestone 1;
    without that declaration the console would not render the component at all.
    Register it unconditionally rather than behind the `rhs_enabled` gate — an admin needs
    to configure tabs *before* enabling the feature, and the setting is invisible to
    non-admins regardless.

- [ ] **W5.3 Seed display and the not-connected rule** ⚠️
  - **Files:** `.../rhs_status_setting.tsx`
  - **Action:** Extend
  - **Details:** Two behaviors that are easy to get subtly wrong and that carry real
    consequences.
    **The virtual seed.** Per Milestone 1's **D2**, a never-configured instance has no
    stored value, so `props.value` arrives empty. The component must render **Assigned
    (locked, not deselectable)** and **In Progress** as selected anyway, purely as display.
    It must **not** call `onChange` to materialize them — doing so marks the form dirty on
    mere page load and writes config the server does not need.
    **Not connected.** When the admin has no Jira connection for the selected instance, the
    status fetch returns `not_connected`. Keep the chips from stored config visible, disable
    the control, and show "Connect Jira to change status tabs." The component **must not**
    call `onChange` in this state. If it did, a System Console Save of any unrelated
    setting would write an empty list over the admin's configured tabs. Note that
    `TeamFilter` has no error handling at all on its fetch — do not copy that.
    Connect JWT instances resolve server-side through the bot client, so the picker works
    there with no personal connection at all; only OAuth2 instances can hit this state.

#### Definition of Done

- [ ] With an empty `props.value`, Assigned and In Progress render selected and **no**
      `onChange` fires on mount — asserted on the mock.
- [ ] With `not_connected`, the control is disabled, stored chips remain, and no `onChange`
      fires.
- [ ] Assigned cannot be deselected.
- [ ] Selecting a status calls `onChange` and `setSaveNeeded` with the correct structured
      value.
- [ ] Only Cloud instances are offered.

---

### Phase W6: Tests and quality gates

**Goal:** the suite covers the gating and state matrix, and CI is green.

**Depends on:** W3, W4, W5.

#### Tasks

- [ ] **W6.1 Component and integration tests**
  - **Files:** `webapp/src/components/rhs/*.test.tsx`, `.../rhs_status_setting.test.tsx`
  - **Action:** Create
  - **Details:** React Testing Library with `renderWithRedux` from
    `webapp/src/testlib/test-utils.tsx` — the repo has RTL 14 and Jest 29, and **no
    Enzyme**. Cover the tab strip, sort reset, load more, all five states, the gating
    matrix from W4, popout rehydrate, and the two W5.3 rules.
    The highest-value assertions are the negative ones: no registration when the gate is
    off, no second fetch while one is in flight, no `onChange` while disconnected. Each of
    those failures is silent in production.

- [ ] **W6.2 Gates**
  - **Files:** none (verification only)
  - **Action:** Verify
  - **Details:** `npm run lint`, `npm run check-types`, `npm run test` in `webapp/`. Lint
    must pass with **zero** suppressions: `react/jsx-no-literals` is an error and the
    established pattern is the expression container — `{'Loading'}`, `label='Refresh'`, or
    a `const` for repeated strings — which passes repo-wide today with no overrides.
    **Do not add an `en.json` and do not introduce `FormattedMessage`.** i18n is
    deliberately out of scope for this milestone and is tracked as future milestone F1;
    adding it piecemeal here would leave the plugin half-translated.

#### Definition of Done

- [ ] `npm run lint`, `npm run check-types`, and `npm run test` all pass.
- [ ] No new `eslint-disable` anywhere in the new code.
- [ ] No `en.json` was added and no `FormattedMessage` was introduced.
- [ ] The Cloud-gating matrix is covered including the `cloud-oauth`-only case.

---

## File Change Map

| File | Phase(s) | Action | Summary |
|------|----------|--------|---------|
| `webapp/src/types/model.ts` | W1 | Modify | `CLOUD_OAUTH` enum value; RHS DTO and error-code types |
| `webapp/src/actions/index.ts` | **W1, W2** | Modify | Exhaustive switch fix (W1); RHS thunks + debounce (W2) |
| `webapp/src/selectors/index.ts` | W1 | Extend | `hasCloudInstance`, `getConnectedCloudInstances` |
| `webapp/src/testlib/test-utils.tsx` | W1 | Modify | Fixture key rename + `cloud-oauth` entry |
| `.../channel_subscriptions/edit_channel_subscription.test.tsx` | W1 | Modify | Same key bug in its local mock state |
| `webapp/src/client/index.ts` | W2 | Extend | JSON-error fetch helper + RHS fetches |
| `webapp/src/action_types/index.ts` | W2 | Modify | RHS action type constants |
| `webapp/src/reducers/index.ts` | W2 | Modify | RHS view-state and data slices |
| `webapp/src/utils/rhs_view_state.ts` | W2 | Create | `localStorage` persistence per W-D1 |
| `webapp/src/components/rhs/*` | W3 | Create | Panel, picker, tab strip, header, row, states, styles |
| `webapp/src/plugin.tsx` | **W4, W5** | Modify | App Bar + RHS registration and popout (W4); admin setting registration (W5) |
| `webapp/src/components/admin_console/rhs_status_setting/*` | W5 | Create | System Console multi-select |
| `webapp/src/**/*.test.tsx` | W6 | Create | Component and gating tests |

**Conflict risk — two files are written by more than one phase.**
`webapp/src/actions/index.ts` is touched by W1 (switch fix) and W2 (thunks); these are
sequential, so a commit between them is sufficient. `webapp/src/plugin.tsx` is touched by
W4 and W5, and **those two phases are meant to run in parallel** — both add registration
calls inside `setupUILater()`, within a few lines of each other. Either serialize the
`plugin.tsx` edit into whichever phase finishes first, or accept a small manual merge.
This is the one real collision in the plan.

## Parallelization

The meaningful fork is **W5 alongside W3+W4**. The admin picker depends only on W1 and W2
and shares no component code with the RHS panel, so two agents can run:

- Agent A: W3 → W4 (the RHS panel, then registration and popout)
- Agent B: W5 (the System Console picker)

Both then converge on W6. Watch `plugin.tsx` per the conflict note above.

Within W3, the row (W3.4), the states (W3.5), and the styles (W3.6) are independent of
each other once the shell (W3.1) exists.

## Testing Strategy

Jest 29 with React Testing Library 14 and `@testing-library/jest-dom`, plus
`redux-mock-store`. No Enzyme. `renderWithRedux` in
`webapp/src/testlib/test-utils.tsx` wraps components in both a Redux `Provider` and
react-intl's `IntlProvider`.

Note that W1.3 fixes a fixture bug that has been suppressing real behavior in existing
tests — run the whole suite after that change, not just the new tests, because some
existing assertions may have been passing only because `getUserConnectedInstances()`
returned empty.

Weight the suite toward negative assertions. The failures that matter in this milestone are
all silent: a feature that registers when it should not, a duplicate in-flight request that
burns Cloud quota, an `onChange` from a disconnected picker that wipes an admin's config.
Each needs an explicit "did not happen" assertion, because none of them throws.

```bash
cd webapp && npm run lint
cd webapp && npm run check-types
cd webapp && npm run test
```

## Definition of Done (Overall)

- [ ] The App Bar icon appears only when `rhs_enabled` is on **and** at least one Cloud
      instance is configured, including the `cloud-oauth`-only case.
- [ ] The RHS lists the user's assigned tickets for the selected Cloud instance across
      Assigned plus the configured status tabs.
- [ ] Sort (Updated / Created), Load more, manual refresh, New ticket, and the instance
      picker all work.
- [ ] Loading, empty, error, not-connected, and rate-limited are five distinct states.
- [ ] A repeated request for the same `{instance, tab, sort}` is debounced in flight.
- [ ] Popping the RHS out rehydrates `{instance, tab, sort}` and re-fetches, with no
      dependency on any platform popout API.
- [ ] The System Console picker restores from `props.value`, shows the virtual seed with no
      `onChange`, and is inert while disconnected.
- [ ] `npm run lint`, `npm run check-types`, and `npm run test` pass with no suppressions
      and no `en.json`.
