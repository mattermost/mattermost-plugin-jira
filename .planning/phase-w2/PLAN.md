# Phase W2 Plan: Data layer

> Prescriptive implementation plan for **Phase W2 only** (tasks W2.1–W2.4). An
> Implementation Engineer should be able to land this without making design
> decisions. Do not implement later phases from this file.
>
> **Do not write production TS/JS until this plan is followed as written.**
> This document is the plan; it is not the code.

## Metadata

- **Parent plan:** `.planning/PLAN.md` § Phase W2 (source of truth for WHAT)
- **Handoff:** `.planning/PHASE1_HANDOFF.md` (routes, DTO, JSON error codes)
- **W1:** `.planning/phase-w1/PLAN.md` (types/selectors already landed)
- **Orchestration:** Gate 7 / Step 7 of `IMPL_ORCHESTRATION_PLAN.md`
- **Worktree:** `~/workspace/worktrees/mattermost-plugin-jira-IDEA-001-show-tickets-rhs`
- **Branch:** `IDEA-001-show-tickets-rhs`
- **Starts from:** `6e6e073` (W1 commit). Do not rebase; do not implement against an older SHA.
- **Package:** `webapp/` (Jest 29, TypeScript 5.7 strict, ESLint 8)
- **Generated:** 2026-08-19
- **Status:** ready for implementation
- **Staffing:** **one implementer.** Sequential. Nothing in this phase is
  parallelizable.

## Scope

**In scope:**

- `webapp/src/types/model.ts` — RHS DTO, tab, sort, error-code types (W1
  deferred these)
- `webapp/src/client/index.ts` — **append** `doFetchJSON` + `getRHSIssues` +
  `getRHSStatuses` + `RHSFetchError`. **Do not edit** `doFetch` /
  `doFetchWithResponse`
- `webapp/src/client/index.test.ts` — **create** (JSON vs text fallback; old
  helper still raw-text)
- `webapp/src/action_types/index.ts` — RHS constants
- `webapp/src/reducers/index.ts` — RHS slices; relative `action_types` import
- `webapp/src/actions/rhs.ts` — **create** (thunks + in-flight map)
- `webapp/src/actions/index.ts` — **re-export only** from `./rhs`
- `webapp/src/actions/rhs.test.ts` — **create** (Gate 7 debounce call-count)
- `webapp/src/utils/rhs_view_state.ts` — **create**
- `webapp/src/utils/rhs_view_state.test.ts` — **create**
- `webapp/src/testlib/test-utils.tsx` — add RHS slice defaults to
  `defaultMockState` (keys must match reducer names, same class of bug W1
  fixed)

**Out of scope (do not do these):**

- Any RHS UI (`webapp/src/components/rhs/`)
- Admin status picker (`webapp/src/components/admin_console/`)
- `webapp/src/plugin.tsx` registration / App Bar / popout / `rhs_enabled` gating
- Changing `doFetch` or `doFetchWithResponse` bodies
- `en.json` / `FormattedMessage`
- `eslint-disable` comments
- Anything under `server/`
- D2 virtual-seed picker behavior (W5 / Gate 8)
- Selectors for RHS slices (W3 will add them; thunks read `pluginStateKey`)
- An `rhsStatuses` Redux slice (W5 keeps picker options in component state)
- Time-based debounce via `debounce-promise` (wrong tool — see Surprise B)
- Committing or pushing

---

## Research (read this before coding)

Line numbers are as of **`6e6e073`**. If they have drifted, search; do not
invent a different design.

### 1. `doFetch` / `doFetchWithResponse` — `webapp/src/client/index.ts:16-42`

**Do not change these two functions.** Gate 7 item 2 is that the JSON-error
helper is **additive**.

```16:42:webapp/src/client/index.ts
export const doFetch = async (url: string, options: FetchOptions) => {
    const {data} = await doFetchWithResponse(url, options);

    return data;
};

export const doFetchWithResponse = async (url: string, options = {}) => {
    const response = await fetch(url, Client4.getOptions(options));

    let data;
    if (response.ok) {
        data = await response.json();

        return {
            response,
            data,
        };
    }

    data = await response.text();

    throw new ClientError(Client4.url, {
        message: data || '',
        status_code: response.status,
        url,
    });
};
```

Load-bearing facts:

- Success: `response.json()`. Error: `response.text()`, thrown as
  `ClientError` whose `message` is the **raw string**. Against `/rhs/*` that
  would be `'{"error":"not_connected",...}'` with the code lost.
- Both go through `fetch(url, Client4.getOptions(options))`. The new helper
  **must** also use `Client4.getOptions` so Mattermost CSRF / cookie headers
  still attach. Do not call `fetch(url, options)` raw.
- `FetchOptions` (`:11-14`) requires `method`. Pass `{method: 'get'}`.
- `buildQueryString` (`:44-61`) prefixes `?`, `encodeURIComponent`s values,
  joins with `&`. Empty input → `''`. Value type is `string | number |
  boolean` — do not pass empty strings for omitted tab fields.

**Insert nothing between these functions.** Append the new helper and the two
RHS fetch functions **after** `buildQueryString` (after `:61`) so
`git diff 6e6e073 -- webapp/src/client/index.ts` shows a pure addition
below the existing helpers.

### 2. Action thunk idiom — `webapp/src/actions/index.ts`

Every server-calling thunk is `async (dispatch: Dispatch, getState: GlobalState) =>`
and returns `{data}` or `{error}` from a `try/catch` around `doFetch`. Canonical
example:

```91:110:webapp/src/actions/index.ts
export const fetchJiraIssueMetadataForProjects = (projectKeys: string[], instanceID: string) => {
    return async (dispatch: Dispatch, getState: GlobalState) => {
        const baseUrl = getPluginServerRoute(getState());
        ...
        try {
            data = await doFetch(`${baseUrl}/api/v2/get-create-issue-metadata-for-project?${params}`, {
                method: 'get',
            });
        } catch (error) {
            return {error};
        }
        ...
        return {data};
    };
};
```

`getSettings` (`:469-487`) is the same shape and already stores
`rhs_enabled` on `RECEIVED_PLUGIN_SETTINGS` because W1 typed
`PluginSettings`. **Do not gate RHS fetches on `rhs_enabled`.** Handoff:
the server routes do not read `EnableJiraRHS`; the picker must work while
the flag is off (W5.2).

**Do not copy the `getState: GlobalState` annotation.** That is a pre-existing
tsc error (`getState` is a function). New thunks use
`(dispatch: Dispatch, getState: () => GlobalState)`. See Surprise H.

URL construction is always `getPluginServerRoute(getState()) + '/api/v2/...'`.
`searchIssues` (`:161-165`) is the closest GET+query analogue:

```161:165:webapp/src/actions/index.ts
export const searchIssues = (params: SearchIssueParams) => {
    return async (dispatch: Dispatch, getState: GlobalState) => {
        const url = getPluginServerRoute(getState()) + '/api/v2/get-search-issues';
        return doFetchWithResponse(`${url}${buildQueryString(params)}`);
    };
};
```

RHS fetches live in `client/index.ts` (parent W2.1) and are **called** from
thunks, not inlined. Thunks still get `baseUrl` from `getPluginServerRoute`.

The file is **`max-lines` 591 / 650** (total 692, `skipBlankLines: true`,
`skipComments: false`). Adding six thunks + debounce here will fail lint.
See Surprise A — put them in `webapp/src/actions/rhs.ts` and re-export.

### 3. Reducer idiom — `webapp/src/reducers/index.ts`

Plain `combineReducers`, no Redux Toolkit. One function per slice, default
state in the argument, `SwitchCase: 0` (cases line up with `switch`).

```21:29:webapp/src/reducers/index.ts
function installedInstances(state = [], action = {} as AnyAction) {
    switch (action.type) {
    case ActionTypes.RECEIVED_INSTANCE_STATUS:
        return action.data.instances ? action.data.instances : [];
    default:
        return state;
    }
}
```

Boolean slices use `state = false`. Nullable slices already exist:
`pluginSettings(state: PluginSettings | null = null, ...)` (`:74`). Use
`null` for `rhsError`, never the identifier `undefined` (`no-undefined` is
error, `.eslintrc.json:350`).

`combineReducers` is `:269-286`. Append the new keys **before** the closing
`});`. `PluginState` is `ReturnType<typeof combinedReducers>`
(`webapp/src/types/store.ts:12`) — new keys appear automatically. Do not
edit `store.ts`.

Import today is the webpack bare specifier `action_types` (`:6`). Jest has
**no** `^action_types$` mapper (`package.json:100-112`). Any test that
imports the real reducer will fail to resolve it. Change that one import to
`../action_types` (Surprise D). Production webpack resolves relative paths
the same way `actions/index.ts:9` already does.

### 4. Action types file pattern — `webapp/src/action_types/index.ts`

```5:8:webapp/src/action_types/index.ts
const {id: PluginId} = manifest;

export default {
    OPEN_CONNECT_MODAL: `${PluginId}_open_connect_modal`,
```

`manifest.id` is `'jira'` (`webapp/src/manifest.ts:5`), so values are
`jira_...`. Default-export one object. Add new keys at the **end**, before
the closing `};` (`:43`). Match the existing `RECEIVED_*` / `OPEN_*` string
shape. Keep the historical typos `recevied_*` on old keys — do not "fix"
them.

### 5. How tests mock fetch — there is no `doFetch` mock

`webapp/tests/setup.js:6-11`:

```6:11:webapp/tests/setup.js
global.fetch = jest.fn(() =>
    Promise.resolve({
        json: () => Promise.resolve({}),
        ok: true,
    }),
);
```

`package.json:92` sets Jest `clearMocks: true` (clears call history, **not**
`mockImplementation`). There is **zero** production-test `jest.mock` of
`../client` or `doFetch`. Connected components that dispatch `getConnected`
succeed because this fetch mock returns `{ok: true, json: () => ({})}`.

Gate 7 requires asserting **the fetch mock's call count**. That means:

1. Thunks must actually call `fetch` (via `doFetchJSON` →
   `Client4.getOptions` → `fetch`). Do not `jest.mock('../client')` in the
   debounce test — that would assert on a fake and miss a double call.
2. Override `global.fetch` in the RHS action tests with a hanging Promise
   for the in-flight case; restore in `afterEach`.
3. The setup.js mock has **no `text()`**. Error-path tests must provide
   `text: () => Promise.resolve(...)`.
4. Module-scope in-flight state is **not** cleared by `clearMocks`. Export
   `resetRHSIssuesInFlight` and call it in `afterEach`. Do **not** name it
   with a leading underscore (`no-underscore-dangle` is error, `:351`).

`redux-mock-store` (`test-utils.tsx:8-13`) does **not** run reducers.
Debounce + load-more + persistence tests that read plugin state after
dispatch **must** use `createStore` + `redux-thunk` + the real plugin
reducer. `renderWithRedux` stays unused in W2 (no UI).

Jest `moduleNameMapper` has `^actions$` → `src/actions` (the folder index)
but **not** `^actions/(.*)$`, **not** `^selectors$`, **not** `^client$`.
New tests import relatively (`./rhs`, `./index`, `../reducers`). Production
files may keep webpack bare specifiers for `types/*` and `utils/*` (those
**are** mapped).

### 6. Plugin ID / API prefix

| Piece | Value | Source |
|-------|-------|--------|
| Plugin id | `jira` | `webapp/src/manifest.ts:5` |
| Redux key | `plugins-jira` | `webapp/src/types/store.ts:8-14` |
| `getPluginServerRoute` | `{sitePath}/plugins/jira` | `webapp/src/selectors/index.ts:16-28` |
| Server `routeAPI` | `/api/v2` | `server/http.go:38` |
| Issues route | `/rhs/issues` on `apiRouter` | `server/http.go:71,108,173` |
| Statuses route | `/rhs/statuses` on `apiRouter` | `server/http.go:72,174` |

Full browser URLs (SiteURL `http://localhost:8065` → empty basePath):

- `GET /plugins/jira/api/v2/rhs/issues?...`
- `GET /plugins/jira/api/v2/rhs/statuses?instance_id=...`

Handoff: *“Plugin HTTP prefix is `/api/v2` (`routeAPI` in `server/http.go`).
Do not add `/api/v2` again in client helpers.”* Meaning: concatenate
`baseUrl + '/api/v2/rhs/issues'`, **not** `baseUrl + '/api/v2/api/v2/...'`.
This matches every existing action (`/api/v2/settingsinfo`,
`/api/v2/get-search-issues`, …).

Query params are **snake_case** (`server/rhs_http.go:25-29`,
`server/constants.go:9`):

| Param | Required | Notes |
|-------|----------|-------|
| `instance_id` | yes | Cloud instance id |
| `tab_kind` | no (server default `assigned`) | `assigned` \| `category` \| `status` |
| `tab_key` | for `category` | e.g. `indeterminate` |
| `tab_id` | for `status` | numeric id as string, e.g. `"10001"` |
| `sort` | no (server default `updated`) | allowlist `updated` \| `created` |
| `next_page_token` | no | opaque cursor; omit on page 1 |

Page size is fixed at 20. **Do not** send `page_size` / `limit` / `startAt`.

JSON success (`server/rhs.go:176-181`):

```json
{"issues":[...],"tabs":[...],"nextPageToken":"...","isLast":true}
```

`tabs` is `[]RHSTabEntry` (`kind`, `key?`, `id?`, `name`) — camelCase nested
fields. Issues DTO fields are in the handoff table (`browseUrl`,
`categoryKey`, `issueType`, `dueDate`, `labels` never JSON `null`).

JSON error (`server/rhs_http.go:34-37, 17-23`):
`{"error":"<code>","message":"<text>"}` with `Content-Type: application/json`.

Codes: `not_connected` (401), `not_authorized` (403), `not_cloud` (400),
`rate_limited` (429), `invalid_request` (400), `internal_error` (500).

**Plain-text exception:** missing `Mattermost-User-Id` is still
`http.Error(w, "Not authorized", 401)` (`server/http.go:323-328`). The new
helper must treat that as the text fallback, not a JSON parse throw. Do not
try to "fix" `checkAuth`.

### 7. ESLint constraints

From `webapp/.eslintrc.json`:

| Rule | Setting | W2 impact |
|------|---------|-----------|
| `header/header` | `:130-135`, two-line copyright | every new file |
| `import/order` | `:138-152` | blank line between groups; internals consecutive (match `actions/index.ts:9-12`) |
| `import-newlines/enforce` | `:154-157`, wrap at **4+** named imports | 3 names stay single-line (W1 Surprise) |
| `sort-imports` | `ignoreDeclarationSort: true` | sort names inside `{…}` |
| `object-curly-spacing` | `"never"` | `{data}` not `{ data }` |
| `quotes` | single | |
| `comma-dangle` | always-multiline | |
| `@typescript-eslint/indent` | 4 spaces, `SwitchCase: 0` | |
| `react/jsx-no-literals` | error | **no JSX in W2**. Do not add strings inside JSX. |
| `max-lines` | 650, counts comments, skips blanks | **Surprise A** — `actions/index.ts` has 59 lines of headroom |
| `no-undefined` | error | `null`, not `undefined` |
| `no-underscore-dangle` | error | `resetRHSIssuesInFlight`, not `_reset...` |
| `no-void` | error | |
| `no-throw-literal` | error | throw `RHSFetchError` (an `Error`), not a plain object |
| `padded-blocks` | never | no blank line right after `{` |
| `lines-around-comment` | `beforeLineComment: true` | blank line before `//`, or avoid mid-body `//` |
| `arrow-parens` | always | `(dispatch) =>` |
| `space-before-function-paren` | `asyncArrow: always` | `async (dispatch)` |
| `consistent-return` | error | every thunk path returns `{data}` or `{error}` |
| `prefer-const` | error | |
| `no-magic-numbers` | **warning** (quiet) | HTTP 401/429 are fine; `--quiet` hides them |
| `@typescript-eslint/explicit-function-return-type` | warning | annotate new exports anyway |

Copyright header exactly:

```
// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
```

**Do not add `en.json` or `FormattedMessage`.** i18n is milestone F1.

### 8. W1 already landed `hasCloudInstance` / `PluginSettings`

Confirmed at `6e6e073`. W2 does **not** re-do this. W4 will read
`getPluginSettings(state)?.rhs_enabled` the way `plugin.tsx:50` reads
`settings.ui_enabled` today.

```184:199:webapp/src/types/model.ts
export enum InstanceType {
    CLOUD = 'cloud',
    CLOUD_OAUTH = 'cloud-oauth',
    SERVER = 'server',
}
export type Instance = {
    alias?: string;
    instance_id: string;
    type: InstanceType;
}

export type PluginSettings = {
    ui_enabled: boolean;
    rhs_enabled: boolean;
    security_level_empty_for_jira_subscriptions: boolean;
};
```

```89:106:webapp/src/selectors/index.ts
export const hasCloudInstance = (state: GlobalState): boolean => {
    const installed = getInstalledInstances(state);
    if (!installed) {
        return false;
    }

    return installed.some(isCloudInstance);
};
...
export const getPluginSettings = (state: GlobalState): PluginSettings | null => getPluginState(state).pluginSettings;
```

`isCloudInstance` is true for `CLOUD` and `CLOUD_OAUTH`. App Bar gating is
W4. **W2 fetches are not gated on Cloud or `rhs_enabled`.**

### 9. `node_modules` exists

`webapp/node_modules/.bin/jest` is present. Do not run `npm install` unless
a command fails with a missing binary. W1 `check-types` reported **249
pre-existing `error TS`**, none in W1 files — expect the same. New W2 files
must not add to that count.

### 10. Gate 7 (must be in this phase)

From `IMPL_ORCHESTRATION_PLAN.md` Step 7:

1. A test proves a second `fetchRHSIssues` for an identical
   `{instance, tab, sort}` while one is in flight issues **no** second
   network call — asserted on the **fetch mock's call count**.
2. The JSON-error helper is additive: `doFetch` / `doFetchWithResponse` are
   unchanged (diff + a regression test that a non-JSON 500 still throws
   `ClientError` with the raw text).

---

## Surprises

### Surprise A — `actions/index.ts` cannot hold the thunks

Parent File Change Map says “Modify `webapp/src/actions/index.ts`”. ESLint
`max-lines` count there is **591 / 650** (59 lines of headroom). The six
thunks + in-flight helper will not fit.

**Put production thunks in `webapp/src/actions/rhs.ts`.** Re-export from
`index.ts` so W3 can `import {fetchRHSIssues} from 'actions'` (Jest maps
`^actions$` to the folder index; there is no `^actions/(.*)$` mapper).

Do not raise `max-lines`. Do not `eslint-disable`.

### Surprise B — `debounce-promise` is the wrong debounce

`webapp/package.json:74` already depends on `debounce-promise`;
`jira_issue_selector.jsx` uses it for **time-based** typeahead. Gate 7 is
**in-flight** coalescing (same key, no second HTTP while outstanding), not
a wait-then-fire. Do not import `debounce-promise`. Use a module-scope
`Map<string, Promise<...>>`.

Parent allows “module scope or a reducer slice”. Module scope is required:
the set-if-absent must be **synchronous** before any `await`, or two
back-to-back dispatches both see “not in flight”. A reducer cannot do that
without a race. Popout isolation is already a separate module context
(spec); a module-scope map is per-window, which is what we want.

### Surprise C — no `doFetch` test mock exists

Gate 7 says “fetch mock”, not “client mock”. Drive `global.fetch`. Do not
`jest.mock` the client module in the debounce test.

### Surprise D — Jest cannot resolve `from 'action_types'`

`reducers/index.ts:6` uses the webpack bare specifier. Jest
`moduleDirectories` is `node_modules` + `non_npm_dependencies` only. Change
that import to `../action_types` in the same file edit as the new slices.
Otherwise `rhs.test.ts` cannot `import reducer from '../reducers'`.

### Surprise E — parent bundled RHS DTOs onto W1; W1 correctly deferred them

W1 plan out-of-scope: “RHS DTO / `RHSErrorCode` types (W2)”. Add them to
`model.ts` in this phase, after `PluginSettings`.

### Surprise F — tab identity is not a string

GitHub-plugin “active tab as a plain string” does not fit Assigned /
category / status. Debounce key and query params use `{kind, key?, id?}`.
`name` is display-only and is **not** part of the debounce key (two tabs
could theoretically share a name).

### Surprise G — no `rhsStatuses` slice

Parent W2.2 lists view + issues slices, not statuses. `fetchRHSStatuses`
returns `{data} | {error}` for W5 and does **not** dispatch into Redux.
Do not invent a statuses slice.

### Surprise H — existing `getState: GlobalState` is a tsc lie

New file uses `getState: () => GlobalState`. Do not “match the file” by
copying the wrong type — that would add tsc errors.

### Surprise I — `check-types` is not clean on this branch

249 pre-existing errors, unchanged by W1. Do not try to fix
`actions/index.ts` typing. Do not treat a dirty `tsc` as a W2 failure
unless the **new** files appear in the error list.

---

## Tasks

### W2.1 — Types on `model.ts`

**File:** `webapp/src/types/model.ts`
**Action:** Modify

Immediately **after** `PluginSettings` (`:195-199`) and **before**
`GetConnectedResponse` (`:201`), add exactly:

```typescript
export type RHSErrorCode =
    'not_connected' |
    'rate_limited' |
    'not_authorized' |
    'not_cloud' |
    'invalid_request' |
    'internal_error';

export type RHSTabKind = 'assigned' | 'category' | 'status';

export type RHSSort = 'updated' | 'created';

export type RHSTab = {
    kind: RHSTabKind;
    name: string;
    key?: string;
    id?: string;
};

export type RHSIssueStatus = {
    name: string;
    categoryKey: string;
};

export type RHSIssue = {
    key: string;
    summary: string;
    browseUrl: string;
    status: RHSIssueStatus;
    priority: string;
    issueType: string;
    project: string;
    assignee: string;
    reporter: string;
    created: string;
    updated: string;
    dueDate: string;
    labels: string[];
};

export type RHSIssuesResponse = {
    issues: RHSIssue[];
    tabs: RHSTab[];
    nextPageToken: string;
    isLast: boolean;
};

export type RHSStatusCategory = {
    id: number;
    key: string;
    name: string;
};

export type RHSStatus = {
    id: string;
    name: string;
    statusCategory: RHSStatusCategory;
};

export type RHSStatusesResponse = {
    statuses: RHSStatus[];
    categories: RHSStatusCategory[];
};

export type RHSViewState = {
    instance: string;
    tab: RHSTab;
    sort: RHSSort;
};

export const RHS_DEFAULT_SORT: RHSSort = 'updated';

export const RHS_DEFAULT_TAB: RHSTab = {
    kind: 'assigned',
    name: 'Assigned',
};

export type FetchRHSIssuesArgs = {
    instanceID: string;
    tab: RHSTab;
    sort: RHSSort;
};
```

JSON names must match `server/rhs_types.go` / `server/rhs.go:176-181` /
handoff DTO table. `labels` is `string[]`, never optional. Do not add JQL
fields. Do not add a `total`.

### W2.2 — JSON-error helper + RHS HTTP functions

**File:** `webapp/src/client/index.ts`
**Action:** Extend (append only)

Leave `:1-61` byte-identical, including `doFetch`, `doFetchWithResponse`,
and `buildQueryString`.

#### Imports

Current:

```typescript
import {Client4} from 'mattermost-redux/client';
import {ClientError} from '@mattermost/client';
```

Add a third import **after** those (same group = external vs internal:
`types/model` is internal, so **blank line** after `ClientError`):

```typescript
import {Client4} from 'mattermost-redux/client';
import {ClientError} from '@mattermost/client';

import {
    RHSErrorCode,
    RHSIssuesResponse,
    RHSSort,
    RHSStatusesResponse,
    RHSTab,
} from 'types/model';
```

Five names → multiline, alphabetized (`RHSErrorCode`, `RHSIssuesResponse`,
`RHSSort`, `RHSStatusesResponse`, `RHSTab`). `ClientError` stays — still
used by `doFetchWithResponse`.

#### Append after `buildQueryString` (after `:61`)

```typescript
export class RHSFetchError extends Error {
    errorCode: RHSErrorCode;
    statusCode: number;

    constructor(errorCode: RHSErrorCode, message: string, statusCode: number) {
        super(message);
        this.name = 'RHSFetchError';
        this.errorCode = errorCode;
        this.statusCode = statusCode;
    }
}

export function coerceRHSErrorCode(value: string): RHSErrorCode {
    switch (value) {
    case 'not_connected':
    case 'rate_limited':
    case 'not_authorized':
    case 'not_cloud':
    case 'invalid_request':
    case 'internal_error':
        return value;
    default:
        return 'internal_error';
    }
}

export function parseRHSErrorBody(text: string, statusCode: number): RHSFetchError {
    try {
        const parsed = JSON.parse(text);
        if (
            parsed &&
            typeof parsed === 'object' &&
            typeof parsed.error === 'string'
        ) {
            const message = typeof parsed.message === 'string' && parsed.message ?
                parsed.message :
                parsed.error;
            return new RHSFetchError(coerceRHSErrorCode(parsed.error), message, statusCode);
        }
    } catch {
        return new RHSFetchError('internal_error', text || '', statusCode);
    }

    return new RHSFetchError('internal_error', text || '', statusCode);
}

export const doFetchJSON = async <T>(url: string, options: FetchOptions): Promise<T> => {
    const response = await fetch(url, Client4.getOptions(options));
    if (response.ok) {
        return response.json();
    }

    const text = await response.text();
    throw parseRHSErrorBody(text, response.status);
};

export type GetRHSIssuesParams = {
    instanceID: string;
    tab: RHSTab;
    sort: RHSSort;
    nextPageToken?: string;
};

function buildRHSIssuesQuery(params: GetRHSIssuesParams): QueryParameters {
    const query: QueryParameters = {
        instance_id: params.instanceID,
        sort: params.sort,
        tab_kind: params.tab.kind,
    };
    if (params.tab.key) {
        query.tab_key = params.tab.key;
    }
    if (params.tab.id) {
        query.tab_id = params.tab.id;
    }
    if (params.nextPageToken) {
        query.next_page_token = params.nextPageToken;
    }

    return query;
}

export function getRHSIssues(baseUrl: string, params: GetRHSIssuesParams): Promise<RHSIssuesResponse> {
    return doFetchJSON<RHSIssuesResponse>(
        `${baseUrl}/api/v2/rhs/issues${buildQueryString(buildRHSIssuesQuery(params))}`,
        {method: 'get'},
    );
}

export function getRHSStatuses(baseUrl: string, instanceID: string): Promise<RHSStatusesResponse> {
    return doFetchJSON<RHSStatusesResponse>(
        `${baseUrl}/api/v2/rhs/statuses${buildQueryString({instance_id: instanceID})}`,
        {method: 'get'},
    );
}
```

`QueryParameters` is already in this file (`:7-9`) and stays unexported.

**Fallback-to-text contract (exact):**

1. Non-2xx → `response.text()` (body can only be read once; do **not**
   also call `response.json()`).
2. `JSON.parse` that object. If it has a string `error` field, throw
   `RHSFetchError` with `coerceRHSErrorCode(error)` and `message` (or
   `error` if `message` is missing/empty).
3. If parse throws, or the value is not an object with string `error`
   (HTML 502, plain `Not authorized`, JSON array, …), throw
   `RHSFetchError` with `errorCode: 'internal_error'` and `message` equal
   to the raw text (or `''` if the body is empty). **Never** let
   `JSON.parse` escape to the caller.

Use optional-catch `catch {` (no binding) so `no-unused-vars` and
`no-underscore-dangle` stay quiet. Both `catch` and the trailing return
produce the same `internal_error` + raw text result — that duplication is
intentional so a `JSON.parse` throw cannot escape.

Do not wrap `doFetch` / `doFetchWithResponse`. They must remain the
plain-text path for every old route.

### W2.3 — Action types, reducers, `defaultMockState`

**Files:** `webapp/src/action_types/index.ts`, `webapp/src/reducers/index.ts`,
`webapp/src/testlib/test-utils.tsx`
**Action:** Modify

#### `action_types/index.ts`

Append **before** the closing `};` (`:43`):

```typescript
    SET_RHS_INSTANCE_ID: `${PluginId}_set_rhs_instance_id`,
    SET_RHS_TAB: `${PluginId}_set_rhs_tab`,
    SET_RHS_SORT: `${PluginId}_set_rhs_sort`,
    HYDRATE_RHS_VIEW_STATE: `${PluginId}_hydrate_rhs_view_state`,
    RHS_ISSUES_LOADING: `${PluginId}_rhs_issues_loading`,
    RECEIVED_RHS_ISSUES: `${PluginId}_received_rhs_issues`,
    RECEIVED_RHS_ISSUES_APPEND: `${PluginId}_received_rhs_issues_append`,
    RHS_ISSUES_ERROR: `${PluginId}_rhs_issues_error`,
```

Keep the trailing comma on the last key. Do not rename old `recevied_*`
keys.

`RHS_ISSUES_LOADING` payload: `{reset: boolean}`. `reset: true` is page-1
(`fetchRHSIssues` / `refreshRHSIssues`). `reset: false` is load-more
(keep current rows). This is how loading stays distinguishable from empty:
the issues slice becomes `[]` only when `reset: true`, and the UI (W3)
must check `rhsLoading` **before** `issues.length === 0`.

#### `reducers/index.ts`

1. Change `:6` from `import ActionTypes from 'action_types';` to:

```typescript
import ActionTypes from '../action_types';
```

2. Change `:7` from `{ChannelSubscription, PluginSettings}` to a
   3-or-fewer **or** 4+ import. Four names → multiline:

```typescript
import {
    ChannelSubscription,
    PluginSettings,
    RHS_DEFAULT_SORT,
    RHS_DEFAULT_TAB,
    RHSErrorCode,
    RHSIssue,
    RHSSort,
    RHSTab,
} from 'types/model';
```

Alphabetize: `ChannelSubscription`, `PluginSettings`, `RHS_DEFAULT_SORT`,
`RHS_DEFAULT_TAB`, `RHSErrorCode`, `RHSIssue`, `RHSSort`, `RHSTab`.

3. Add these functions **immediately before** `export default combineReducers`
   (`:269`). Copy the switch indent of `installedInstances`.

```typescript
function rhsInstanceID(state = '', action = {} as AnyAction): string {
    switch (action.type) {
    case ActionTypes.SET_RHS_INSTANCE_ID:
        return action.data;
    case ActionTypes.HYDRATE_RHS_VIEW_STATE:
        return action.data.instance;
    default:
        return state;
    }
}

function rhsTab(state: RHSTab = RHS_DEFAULT_TAB, action = {} as AnyAction): RHSTab {
    switch (action.type) {
    case ActionTypes.SET_RHS_TAB:
        return action.data;
    case ActionTypes.HYDRATE_RHS_VIEW_STATE:
        return action.data.tab;
    default:
        return state;
    }
}

function rhsSort(state: RHSSort = RHS_DEFAULT_SORT, action = {} as AnyAction): RHSSort {
    switch (action.type) {
    case ActionTypes.SET_RHS_SORT:
        return action.data;
    case ActionTypes.HYDRATE_RHS_VIEW_STATE:
        return action.data.sort;
    default:
        return state;
    }
}

function rhsIssues(state: RHSIssue[] = [], action = {} as AnyAction): RHSIssue[] {
    switch (action.type) {
    case ActionTypes.RHS_ISSUES_LOADING:
        if (action.data && action.data.reset) {
            return [];
        }
        return state;
    case ActionTypes.RECEIVED_RHS_ISSUES:
        return action.data.issues ? action.data.issues : [];
    case ActionTypes.RECEIVED_RHS_ISSUES_APPEND:
        return state.concat(action.data.issues ? action.data.issues : []);
    default:
        return state;
    }
}

function rhsTabs(state: RHSTab[] = [], action = {} as AnyAction): RHSTab[] {
    switch (action.type) {
    case ActionTypes.RECEIVED_RHS_ISSUES:
    case ActionTypes.RECEIVED_RHS_ISSUES_APPEND:
        return action.data.tabs ? action.data.tabs : state;
    default:
        return state;
    }
}

function rhsNextPageToken(state = '', action = {} as AnyAction): string {
    switch (action.type) {
    case ActionTypes.RHS_ISSUES_LOADING:
        if (action.data && action.data.reset) {
            return '';
        }
        return state;
    case ActionTypes.RECEIVED_RHS_ISSUES:
    case ActionTypes.RECEIVED_RHS_ISSUES_APPEND:
        return action.data.nextPageToken ? action.data.nextPageToken : '';
    default:
        return state;
    }
}

function rhsIsLast(state = true, action = {} as AnyAction): boolean {
    switch (action.type) {
    case ActionTypes.RHS_ISSUES_LOADING:
        if (action.data && action.data.reset) {
            return true;
        }
        return state;
    case ActionTypes.RECEIVED_RHS_ISSUES:
    case ActionTypes.RECEIVED_RHS_ISSUES_APPEND:
        return Boolean(action.data.isLast);
    default:
        return state;
    }
}

function rhsLoading(state = false, action = {} as AnyAction): boolean {
    switch (action.type) {
    case ActionTypes.RHS_ISSUES_LOADING:
        return true;
    case ActionTypes.RECEIVED_RHS_ISSUES:
    case ActionTypes.RECEIVED_RHS_ISSUES_APPEND:
    case ActionTypes.RHS_ISSUES_ERROR:
        return false;
    default:
        return state;
    }
}

function rhsError(state: RHSErrorCode | null = null, action = {} as AnyAction): RHSErrorCode | null {
    switch (action.type) {
    case ActionTypes.RHS_ISSUES_LOADING:
    case ActionTypes.RECEIVED_RHS_ISSUES:
    case ActionTypes.RECEIVED_RHS_ISSUES_APPEND:
        return null;
    case ActionTypes.RHS_ISSUES_ERROR:
        return action.data;
    default:
        return state;
    }
}
```

4. Append to `combineReducers` (`:269-286`), before `});`:

```typescript
    rhsInstanceID,
    rhsTab,
    rhsSort,
    rhsIssues,
    rhsTabs,
    rhsNextPageToken,
    rhsIsLast,
    rhsLoading,
    rhsError,
```

**Loading vs empty (do not regress this):**

| State | `rhsLoading` | `rhsIssues` | `rhsError` | W3 must show |
|-------|--------------|-------------|------------|--------------|
| In flight, page 1 | `true` | `[]` | `null` | loading, **not** empty |
| In flight, load more | `true` | previous rows | `null` | list + loading |
| Success, no rows | `false` | `[]` | `null` | empty |
| Success, rows | `false` | `[...]` | `null` | list |
| Failed page 1 | `false` | `[]` | code | error / not_connected / rate_limited |

Never set `rhsLoading` false while leaving the caller to infer “empty”
from `[]` during flight. `rhsLoading` exists specifically because the
GitHub plugin conflates those states.

#### `test-utils.tsx`

Inside `defaultMockState['plugins-jira']` (`:25-33`), **after**
`channelSubscriptions: {}` add the new keys with reducer defaults. Do not
remove W1's `userConnectedInstances` / `instance2`. Do not reintroduce
`connectedInstances`.

```typescript
        channelSubscriptions: {},
        rhsInstanceID: '',
        rhsTab: {kind: 'assigned', name: 'Assigned'},
        rhsSort: 'updated',
        rhsIssues: [],
        rhsTabs: [],
        rhsNextPageToken: '',
        rhsIsLast: true,
        rhsLoading: false,
        rhsError: null,
```

`rhsError: null` is required (`no-undefined`). Importing `RHS_DEFAULT_TAB`
here is optional; a literal matching the constant is fine and avoids
growing the import.

### W2.4 — View-state persistence

**File:** `webapp/src/utils/rhs_view_state.ts` (create)

Copyright header, then:

```typescript
import {
    RHS_DEFAULT_TAB,
    RHSSort,
    RHSTab,
    RHSTabKind,
    RHSViewState,
} from 'types/model';

const storagePrefix = 'jira:rhs-view:';

export function rhsViewStorageKey(userId: string): string {
    return storagePrefix + userId;
}

function isRHSSort(value: unknown): value is RHSSort {
    return value === 'updated' || value === 'created';
}

function isRHSTabKind(value: unknown): value is RHSTabKind {
    return value === 'assigned' || value === 'category' || value === 'status';
}

export function validateRHSViewState(raw: unknown): RHSViewState | null {
    if (!raw || typeof raw !== 'object') {
        return null;
    }

    const candidate = raw as {instance?: unknown; tab?: unknown; sort?: unknown};
    if (typeof candidate.instance !== 'string' || !candidate.instance) {
        return null;
    }
    if (!isRHSSort(candidate.sort)) {
        return null;
    }
    if (!candidate.tab || typeof candidate.tab !== 'object') {
        return null;
    }

    const tabRaw = candidate.tab as {kind?: unknown; name?: unknown; key?: unknown; id?: unknown};
    if (!isRHSTabKind(tabRaw.kind)) {
        return null;
    }

    const tab: RHSTab = {
        kind: tabRaw.kind,
        name: typeof tabRaw.name === 'string' && tabRaw.name ? tabRaw.name : RHS_DEFAULT_TAB.name,
    };

    if (tab.kind === 'category') {
        if (typeof tabRaw.key !== 'string' || !tabRaw.key) {
            return null;
        }
        tab.key = tabRaw.key;
    }

    if (tab.kind === 'status') {
        if (typeof tabRaw.id !== 'string' || !tabRaw.id) {
            return null;
        }
        tab.id = tabRaw.id;
    }

    return {
        instance: candidate.instance,
        tab,
        sort: candidate.sort,
    };
}

export function loadRHSViewState(userId: string): RHSViewState | null {
    if (!userId) {
        return null;
    }

    try {
        const raw = localStorage.getItem(rhsViewStorageKey(userId));
        if (!raw) {
            return null;
        }
        return validateRHSViewState(JSON.parse(raw));
    } catch {
        return null;
    }
}

export function saveRHSViewState(userId: string, view: RHSViewState): void {
    if (!userId) {
        return;
    }

    try {
        const tab: RHSTab = {
            kind: view.tab.kind,
            name: view.tab.name,
        };
        if (view.tab.key) {
            tab.key = view.tab.key;
        }
        if (view.tab.id) {
            tab.id = view.tab.id;
        }
        const payload: RHSViewState = {
            instance: view.instance,
            tab,
            sort: view.sort,
        };
        localStorage.setItem(rhsViewStorageKey(userId), JSON.stringify(payload));
    } catch {
        return;
    }
}
```

Key: `jira:rhs-view:${userId}` (plugin id `jira`, per W-D1). Different users
in one browser must not share an entry.

Invalid stored tab kind, missing category `key`, missing status `id`, or
sort outside `updated`/`created` → `null` (caller uses defaults). Empty
`instance` → `null`. `JSON.parse` / `localStorage` throw → `null` / no-op.
**Never rethrow.** Use optional-catch `catch {` (no binding).

### W2.5 — Thunks and in-flight debounce

**Files:** `webapp/src/actions/rhs.ts` (create), `webapp/src/actions/index.ts`
(re-export only)

#### Debounce key (exact)

```typescript
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

Property order is load-bearing (insertion order). Always construct with
`instance`, `kind`, `key`, `id`, `sort` in that order. **Do not** include
`name` or `nextPageToken`. Page 1, load-more, and refresh of the same
triple share one in-flight slot — that is the rate-limit mitigation.

#### In-flight map

```typescript
const rhsIssuesInFlight: Map<string, Promise<{data: RHSIssuesResponse} | {error: RHSFetchError}>> = new Map();

export function resetRHSIssuesInFlight(): void {
    rhsIssuesInFlight.clear();
}

function withRHSIssuesInFlight(
    key: string,
    work: () => Promise<{data: RHSIssuesResponse} | {error: RHSFetchError}>,
): Promise<{data: RHSIssuesResponse} | {error: RHSFetchError}> {
    const existing = rhsIssuesInFlight.get(key);
    if (existing) {
        return existing;
    }

    const promise = (async () => {
        try {
            return await work();
        } finally {
            rhsIssuesInFlight.delete(key);
        }
    })();

    rhsIssuesInFlight.set(key, promise);
    return promise;
}
```

**`rhsIssuesInFlight.set` must run synchronously** in the thunk, before
any `await`. `withRHSIssuesInFlight` does that: it assigns the map and
returns, and only then does `work()` hit the network. Two
`store.dispatch(fetchRHSIssues(sameArgs))` on the same stack therefore
share one Promise and **one** `fetch` call.

`resetRHSIssuesInFlight` is for tests. Production never needs it (popout
= new module; `finally` deletes on settle).

Do **not** debounce `fetchRHSStatuses`. Statuses are not `/search/jql`
(21 points). Two admin mounts may both fetch.

#### `rhs.ts` thunks

Imports (match `actions/index.ts` grouping: redux external; then parent
`../…`; then `utils/`; blank; then `types/`):

```typescript
import {Dispatch} from 'redux';

import ActionTypes from '../action_types';
import {getRHSIssues, getRHSStatuses, RHSFetchError} from '../client';
import {getPluginServerRoute} from '../selectors';
import {loadRHSViewState, saveRHSViewState} from 'utils/rhs_view_state';

import {
    FetchRHSIssuesArgs,
    RHSErrorCode,
    RHSIssuesResponse,
    RHSSort,
    RHSTab,
} from 'types/model';
import {GlobalState, pluginStateKey} from 'types/store';
```

`getRHSIssues, getRHSStatuses, RHSFetchError` is three names — single line.
Five from `types/model` — multiline, alphabetized.

Helpers (not exported except as noted above):

```typescript
function persistCurrentRHSView(state: GlobalState): void {
    const plugin = state[pluginStateKey];
    saveRHSViewState(state.entities.users.currentUserId, {
        instance: plugin.rhsInstanceID,
        tab: plugin.rhsTab,
        sort: plugin.rhsSort,
    });
}

function toRHSFetchError(error: unknown): RHSFetchError {
    if (error instanceof RHSFetchError) {
        return error;
    }
    const message = error instanceof Error ? error.message : 'internal error';
    return new RHSFetchError('internal_error', message, 0);
}
```

`currentUserId` is how `sendEphemeralPost` already reads the user
(`actions/index.ts:659`). Do not import `getCurrentUser`.

**Setters** — update Redux + persist; **do not fetch**:

```typescript
export const setRHSInstanceID = (instanceID: string) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        dispatch({
            type: ActionTypes.SET_RHS_INSTANCE_ID,
            data: instanceID,
        });
        persistCurrentRHSView(getState());
        return {data: instanceID};
    };
};

export const setRHSTab = (tab: RHSTab) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        dispatch({
            type: ActionTypes.SET_RHS_TAB,
            data: tab,
        });
        persistCurrentRHSView(getState());
        return {data: tab};
    };
};

export const setRHSSort = (sort: RHSSort) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        dispatch({
            type: ActionTypes.SET_RHS_SORT,
            data: sort,
        });
        persistCurrentRHSView(getState());
        return {data: sort};
    };
};

export const restoreRHSViewState = () => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const saved = loadRHSViewState(getState().entities.users.currentUserId);
        if (!saved) {
            return {data: null};
        }
        dispatch({
            type: ActionTypes.HYDRATE_RHS_VIEW_STATE,
            data: saved,
        });
        return {data: saved};
    };
};
```

`restoreRHSViewState` must **not** call `saveRHSViewState` (no
write-on-read). W3 calls this on mount **before** the first
`fetchRHSIssues`.

**`fetchRHSIssues`** — stamps view state, resets list/cursor, page 1
(no `next_page_token`):

```typescript
export const fetchRHSIssues = (args: FetchRHSIssuesArgs) => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const key = rhsIssuesFlightKey(args.instanceID, args.tab, args.sort);

        return withRHSIssuesInFlight(key, async () => {
            dispatch({
                type: ActionTypes.SET_RHS_INSTANCE_ID,
                data: args.instanceID,
            });
            dispatch({
                type: ActionTypes.SET_RHS_TAB,
                data: args.tab,
            });
            dispatch({
                type: ActionTypes.SET_RHS_SORT,
                data: args.sort,
            });
            persistCurrentRHSView(getState());
            dispatch({
                type: ActionTypes.RHS_ISSUES_LOADING,
                data: {reset: true},
            });

            try {
                const data = await getRHSIssues(getPluginServerRoute(getState()), {
                    instanceID: args.instanceID,
                    tab: args.tab,
                    sort: args.sort,
                });
                dispatch({
                    type: ActionTypes.RECEIVED_RHS_ISSUES,
                    data,
                });
                return {data};
            } catch (error) {
                const rhsError = toRHSFetchError(error);
                dispatch({
                    type: ActionTypes.RHS_ISSUES_ERROR,
                    data: rhsError.errorCode,
                });
                return {error: rhsError};
            }
        });
    };
};
```

If the in-flight map already has `key`, `withRHSIssuesInFlight` returns
the existing Promise and **`work` does not run** — no second `dispatch`
of LOADING, no second `getRHSIssues`, no second `fetch`.

**`loadMoreRHSIssues`** — appends; skipped when `rhsIsLast` or empty
token (no network, no in-flight entry):

```typescript
export const loadMoreRHSIssues = () => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const plugin = getState()[pluginStateKey];
        if (plugin.rhsIsLast || !plugin.rhsNextPageToken) {
            return Promise.resolve({data: plugin.rhsIssues as RHSIssuesResponse['issues']});
        }

        const instanceID = plugin.rhsInstanceID as string;
        const tab = plugin.rhsTab as RHSTab;
        const sort = plugin.rhsSort as RHSSort;
        const nextPageToken = plugin.rhsNextPageToken as string;
        const key = rhsIssuesFlightKey(instanceID, tab, sort);

        return withRHSIssuesInFlight(key, async () => {
            dispatch({
                type: ActionTypes.RHS_ISSUES_LOADING,
                data: {reset: false},
            });

            try {
                const data = await getRHSIssues(getPluginServerRoute(getState()), {
                    instanceID,
                    tab,
                    sort,
                    nextPageToken,
                });
                dispatch({
                    type: ActionTypes.RECEIVED_RHS_ISSUES_APPEND,
                    data,
                });
                return {data};
            } catch (error) {
                const rhsError = toRHSFetchError(error);
                dispatch({
                    type: ActionTypes.RHS_ISSUES_ERROR,
                    data: rhsError.errorCode,
                });
                return {error: rhsError};
            }
        });
    };
};
```

`consistent-return`: the early path returns `Promise.resolve(...)` so both
paths are Promises.

**`refreshRHSIssues`** — page 1 of the *current* view state. Reuses
`fetchRHSIssues` (and therefore the same debounce key):

```typescript
export const refreshRHSIssues = () => {
    return (dispatch: Dispatch, getState: () => GlobalState) => {
        const plugin = getState()[pluginStateKey];
        return dispatch(fetchRHSIssues({
            instanceID: plugin.rhsInstanceID,
            tab: plugin.rhsTab,
            sort: plugin.rhsSort,
        }) as any);
    };
};
```

`as any` matches `plugin.tsx:67` / existing untyped thunk dispatch. Do not
introduce a new Redux Toolkit typing layer.

**`fetchRHSStatuses`** — no debounce, no Redux slice:

```typescript
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

`dispatch` is unused. Prefix is not allowed by `no-underscore-dangle` /
`args: after-used` may flag it. Keep the `(dispatch, getState)` signature
to match every other thunk (eslint `args: after-used` allows unused
`dispatch` because it is before a used arg). Confirm: `after-used` means
args before the last used arg may be unused. `dispatch` then `getState`
(used) → `dispatch` is allowed unused. Good.

#### `actions/index.ts` re-export

Add **at the end of the file** (`:692`), after `fetchIssueByKey`. This is
~12 non-blank lines; headroom is 59.

```typescript
export {
    fetchRHSIssues,
    fetchRHSStatuses,
    loadMoreRHSIssues,
    refreshRHSIssues,
    resetRHSIssuesInFlight,
    restoreRHSViewState,
    rhsIssuesFlightKey,
    setRHSInstanceID,
    setRHSSort,
    setRHSTab,
} from './rhs';
```

Alphabetize the export list. Do not add thunk bodies here. Do not touch
the SERVER `if` at `:576`.

---

## Tests

New files, copyright header, relative imports, no JSX, no `eslint-disable`,
no `en.json`.

Shared fixture for issues payloads (inline in tests; do not add
`webapp/src/testdata` unless you want — inline is enough):

```typescript
const assignedTab: RHSTab = {kind: 'assigned', name: 'Assigned'};

const page1: RHSIssuesResponse = {
    issues: [{
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
    }],
    tabs: [assignedTab],
    nextPageToken: 'tok-page-2',
    isLast: false,
};
```

### `webapp/src/client/index.test.ts` (create)

Import `ClientError` from `@mattermost/client`, helpers from `./index`.
Cast `global.fetch` to `jest.Mock`.

`beforeEach`: restore fetch to a controllable mock. `afterEach`:
`fetchMock.mockReset()`.

Required tests and **names**:

1. **`doFetchJSON returns JSON on a 2xx response`** — `ok: true`,
   `json: () => Promise.resolve({hello: 'world'})` → result `{hello:
   'world'}`.
2. **`doFetchJSON surfaces a typed error code from a JSON error body`** —
   `ok: false`, `status: 401`, `text: () => Promise.resolve(
   JSON.stringify({error: 'not_connected', message: 'Jira account is not connected'}))`.
   Rejects with `RHSFetchError`, `errorCode === 'not_connected'`,
   `message === 'Jira account is not connected'`, `statusCode === 401`.
3. **`doFetchJSON falls back to text when the body is not JSON`** —
   `ok: false`, `status: 502`, `text: () => Promise.resolve('<html>bad gateway</html>')`.
   Rejects with `RHSFetchError`, `errorCode === 'internal_error'`,
   `message === '<html>bad gateway</html>'`. **Must not throw
   `SyntaxError`.** This is the proxy-502 case from parent W2.1.
4. **`doFetchJSON falls back to text for checkAuth plain 401`** —
   body `'Not authorized\n'` (Go `http.Error` adds a newline).
   `errorCode === 'internal_error'`, message contains `Not authorized`.
5. **`doFetchWithResponse still throws ClientError with the raw non-JSON body`**
   — **Gate 7 item 2.** Same 502 HTML as test 3, but call
   `doFetchWithResponse(url, {method: 'get'})`. Rejects with
   `ClientError`, `error.message` containing `'<html>bad gateway</html>'`.
   `error` must **not** be an `RHSFetchError`. This fails if someone
   “upgraded” the old helper to parse JSON.
6. **`getRHSIssues calls /plugins path with snake_case query params`** —
   mock 200 with a minimal `RHSIssuesResponse`. Call
   `getRHSIssues('/plugins/jira', {instanceID: 'https://x.atlassian.net',
   tab: {kind: 'category', name: 'In Progress', key: 'indeterminate'},
   sort: 'created'})`.
   `fetchMock.mock.calls[0][0]` (the URL) contains
   `/plugins/jira/api/v2/rhs/issues`, `instance_id=`, `tab_kind=category`,
   `tab_key=indeterminate`, `sort=created`, and does **not** contain
   `next_page_token` or `/api/v2/api/v2`.
7. **`getRHSIssues omits empty tab_key and tab_id for assigned`** —
   tab `{kind: 'assigned', name: 'Assigned'}`. URL contains
   `tab_kind=assigned`, does not contain `tab_key=` or `tab_id=`.
8. **`getRHSStatuses calls /api/v2/rhs/statuses`** — URL contains
   `/plugins/jira/api/v2/rhs/statuses` and `instance_id=`.

For tests 6–8, `ok: true` and `json: () => Promise.resolve(...)`. Second
fetch argument is whatever `Client4.getOptions` produces — assert with
`expect.anything()`; do **not** expect the raw `{method: 'get'}`.

### `webapp/src/utils/rhs_view_state.test.ts` (create)

Use a real `localStorage` (jsdom provides one). `afterEach`:
`localStorage.clear()`.

Required tests:

1. **`rhs_view_state round-trips instance tab sort`** — `save` then `load`
   with user `'u1'` equals the saved `{instance, tab, sort}` including
   category `key`.
2. **`rhs_view_state isolates keys by user id`** — save as `'u1'`, load as
   `'u2'` returns `null`.
3. **`loadRHSViewState returns null when localStorage throws`** — spy
   `localStorage.getItem` to throw. `loadRHSViewState('u1')` is `null`
   (does not throw).
4. **`saveRHSViewState swallows a localStorage throw`** — spy `setItem` to
   throw. Calling save does not throw.
5. **`validateRHSViewState rejects an unknown tab kind`** — `{instance:
   'x', tab: {kind: 'nope', name: 'X'}, sort: 'updated'}` → `null`.
6. **`validateRHSViewState rejects a sort outside updated or created`** —
   `sort: 'priority'` → `null`.
7. **`validateRHSViewState rejects a category tab without key`** —
   `{kind: 'category', name: 'In Progress'}` → `null`.
8. **`validateRHSViewState rejects a status tab without id`** → `null`.

### `webapp/src/actions/rhs.test.ts` (create) — Gate 7 lives here

Imports: `createStore`, `applyMiddleware` from `redux`; `thunk` from
`redux-thunk`; plugin `reducer` from `../reducers`; thunks from `./rhs`;
`pluginStateKey` from `types/store`.

```typescript
function makeRHSStore() {
    const pluginState = reducer(undefined, {type: '@@INIT'} as any);
    const initialState = {
        entities: {
            general: {
                config: {
                    SiteURL: 'http://localhost:8065',
                },
            },
            users: {
                currentUserId: 'user-1',
            },
        },
        [pluginStateKey]: pluginState,
    };
    const root = (state = initialState, action: any) => {
        return {
            entities: state.entities,
            [pluginStateKey]: reducer(state[pluginStateKey], action),
        };
    };
    return createStore(root, applyMiddleware(thunk));
}

function mockFetchOk(body: RHSIssuesResponse) {
    const fetchMock = global.fetch as jest.Mock;
    fetchMock.mockImplementation(() => Promise.resolve({
        ok: true,
        json: () => Promise.resolve(body),
    }));
    return fetchMock;
}
```

`beforeEach` / `afterEach`: `resetRHSIssuesInFlight()`; reset `global.fetch`.

Required tests — **names are load-bearing for Gate 7 review**:

1. **`a second fetchRHSIssues for the same instance tab sort while in flight issues no second network call`**
   — **Gate 7 item 1.** Exact recipe:

   ```typescript
   const fetchMock = global.fetch as jest.Mock;
   let resolveFetch: (value: unknown) => void = () => {
       // filled in by the mock
   };
   fetchMock.mockImplementation(() => new Promise((resolve) => {
       resolveFetch = resolve;
   }));

   const store = makeRHSStore();
   const args = {instanceID: 'https://example.atlassian.net', tab: assignedTab, sort: 'updated' as const};

   const p1 = store.dispatch(fetchRHSIssues(args) as any);
   const p2 = store.dispatch(fetchRHSIssues(args) as any);

   expect(fetchMock).toHaveBeenCalledTimes(1);

   resolveFetch({
       ok: true,
       json: () => Promise.resolve({
           issues: [],
           tabs: [assignedTab],
           nextPageToken: '',
           isLast: true,
       }),
   });

   await p1;
   await p2;
   expect(fetchMock).toHaveBeenCalledTimes(1);
   ```

   Both dispatches are **synchronous** on the same stack so the map is
   populated before either `work()` runs. Do **not** `await` between
   `p1` and `p2`. After resolve, both promises settle. Call count stays
   **1**. This is the whole gate.

2. **`fetchRHSIssues for a different tab while in flight does call fetch again`**
   — hang the first call; dispatch a second with `tab: {kind: 'category',
   name: 'In Progress', key: 'indeterminate'}`. `toHaveBeenCalledTimes(2)`.
   Then resolve so the test does not leak.

3. **`fetchRHSIssues after the in-flight request settles does call fetch again`**
   — complete one request, then dispatch the same args again.
   `toHaveBeenCalledTimes(2)`. Proves `finally` deleted the map entry
   (otherwise Gate 7 would pass by never fetching twice, ever).

4. **`changing sort resets the list and cursor rather than appending`** —
   `mockFetchOk(page1)`; `fetchRHSIssues` sort `updated`; state issues
   length 1, `rhsNextPageToken === 'tok-page-2'`, `rhsIsLast === false`,
   `rhsLoading === false`. Then mock a page-1 created response
   `{issues: [{... key: 'TES-9' ...}], tabs: [...], nextPageToken: '',
   isLast: true}`; `fetchRHSIssues` with `sort: 'created'`. Issues
   length is **1** (not 2), key is `TES-9`, token is `''`. This is parent
   DoD “changing sort resets the cursor”.

5. **`loadMoreRHSIssues appends rather than replaces`** — after page1
   landed, mock page2 `{issues: [{... key: 'TES-2' ...}], tabs: [...],
   nextPageToken: '', isLast: true}`. Dispatch `loadMoreRHSIssues()`.
   Issues keys are `['TES-1', 'TES-2']`. Fetch URL (second call) contains
   `next_page_token=tok-page-2`. `rhsLoading` is false after settle.

6. **`loadMoreRHSIssues does not fetch when isLast`** — after a fetch that
   returned `isLast: true`, `nextPageToken: ''`, dispatch load-more.
   Fetch call count stays 1.

7. **`rhsLoading is true and issues are empty during a page-1 fetch`** —
   hang fetch; dispatch `fetchRHSIssues`; **before** resolving,
   `getState()[pluginStateKey].rhsLoading === true` and
   `.rhsIssues.length === 0` and `.rhsError === null`. This is the
   loading-vs-empty lock. Then resolve.

8. **`a JSON not_connected error stores the typed code`** — mock
   `ok: false`, `status: 401`, `text: () => Promise.resolve(
   JSON.stringify({error: 'not_connected', message: 'Jira account is not connected'}))`.
   After dispatch, `rhsError === 'not_connected'`, `rhsLoading === false`.
   Result `{error: RHSFetchError}`.

9. **`fetchRHSStatuses returns data without writing issue slices`** — mock
   `{statuses: [], categories: []}`. After dispatch, `rhsIssues` still
   `[]` and `rhsLoading` still `false`. Return value `{data: ...}`.

10. **`setRHSSort persists view state keyed by user id`** — dispatch
    `setRHSSort('created')`; `JSON.parse(localStorage.getItem('jira:rhs-view:user-1'))`
    has `sort === 'created'`. Needs the real store so `getState()` after
    dispatch sees the new sort.

Also assert test 1’s fetch URL contains `/api/v2/rhs/issues` once you
resolve, if convenient — not required for the gate.

Do **not** use `redux-mock-store` here. It will not update `rhsNextPageToken`
for load-more.

---

## File-by-file change list (current line numbers)

Line numbers as of `6e6e073`.

| File | Action | Current lines | What to do |
|------|--------|---------------|------------|
| `webapp/src/types/model.ts` | Modify | `PluginSettings` `:195-199`; `GetConnectedResponse` `:201` | Insert RHS types after `PluginSettings` |
| `webapp/src/client/index.ts` | Extend | `doFetch` `:16-20`; `doFetchWithResponse` `:22-42`; `buildQueryString` `:44-61`; EOF `:61` | **Do not edit 16–61.** Append helper + fetches after `:61`. Expand imports. |
| `webapp/src/client/index.test.ts` | Create | n/a | JSON / text / Gate 7.2 / URL shape |
| `webapp/src/action_types/index.ts` | Modify | object `:8-43` | Append 8 RHS keys before `};` |
| `webapp/src/reducers/index.ts` | Modify | import `:6-7`; `pluginSettings` `:74-81`; combine `:269-286` | Relative `action_types`; add 9 slices; append to `combineReducers` |
| `webapp/src/actions/rhs.ts` | Create | n/a | Thunks + module-scope in-flight map |
| `webapp/src/actions/index.ts` | Re-export | EOF `:692`; SERVER `if` `:576`; thunk idiom `:91-110` | Append `export {…} from './rhs'` only. **Do not add thunk bodies.** Headroom 59 lines. |
| `webapp/src/actions/rhs.test.ts` | Create | n/a | **Gate 7.1** call-count test + sort reset + append + loading-vs-empty |
| `webapp/src/utils/rhs_view_state.ts` | Create | n/a | `localStorage` per user id, try/catch, validate |
| `webapp/src/utils/rhs_view_state.test.ts` | Create | n/a | Round-trip, throw, invalid tab/sort |
| `webapp/src/testlib/test-utils.tsx` | Modify | `defaultMockState` `:24-41` | Add RHS slice defaults |
| `webapp/src/selectors/index.ts` | **Do not touch** | `hasCloudInstance` `:89-96`; `getPluginSettings` `:106` | W1 done; W3 adds RHS selectors |
| `webapp/src/plugin.tsx` | **Do not touch** | `ui_enabled` `:50` | W4 reads `rhs_enabled` |
| `webapp/src/components/**` | **Do not touch** | — | W3 / W5 |
| `server/**` | **Do not touch** | — | Milestone 1 already ships the routes |

---

## Existing tests that must be re-run

Run the **full** Jest suite (`npm run test`, no path filter), same as W1.

W2 should not change behavior of existing suites: new reducer keys default
in `combineReducers` only when the **real** reducer runs, and existing
tests use `redux-mock-store` with `defaultMockState`. Adding keys to that
fixture is additive. If a test snapshots the whole `'plugins-jira'`
object, update the snapshot / assertion to include the new keys — do not
omit the keys from the fixture.

W1 baseline after its commit: **14 suites / 93 tests**. W2 adds 3 suites.
Expect **17 suites**. Test count = 93 + the names listed above (client 8 +
view-state 8 + actions 10 = 26) → **119 tests** if no extras and no
skips. Extra tests are fine; missing Gate 7 names are not.

---

## Commands

From `webapp/` (or `cd webapp && …` from the worktree root). `node_modules`
is already installed.

```bash
cd webapp
npm run lint
npm run test
npm run check-types
```

- `lint` is `eslint … --quiet --cache` (`package.json:11`). Errors must be
  zero. No `eslint-disable`. Confirm `actions/index.ts` stays under 650
  non-blank lines.
- `test` is the **full** Jest suite (`package.json:13`). Do not pass a
  filename when declaring Gate 7 done. While iterating, filtering
  `rhs.test.ts` / `client/index.test.ts` is fine.
- `check-types` will still report the **249 pre-existing** errors. Fail
  W2 only if a **new** file or a W2-touched file appears in that list.

Do not run `make test` (that also runs Go). W2 does not touch Go.
Do not run `npm install` unless binaries are missing.

Gate 7 manual checks after tests:

```bash
git diff 6e6e073 -- webapp/src/client/index.ts
```

`doFetch` / `doFetchWithResponse` hunks must be empty (those lines
unchanged). Only imports + appended helpers should appear.

```bash
rg -n "doFetchWithResponse|doFetchJSON" webapp/src/client/index.ts
```

Old helper still `response.text()` + `ClientError`. New helper still
`parseRHSErrorBody`.

```bash
rg -n "debounce-promise" webapp/src/actions webapp/src/client
```

Must be empty.

```bash
rg -n "FormattedMessage|en.json" webapp/src/actions/rhs.ts webapp/src/client/index.ts webapp/src/utils/rhs_view_state.ts
```

Must be empty.

---

## Definition of Done

- [ ] `doFetch` (`:16-20`) and `doFetchWithResponse` (`:22-42`) are
      byte-identical to `6e6e073`
- [ ] `doFetchJSON` on JSON `{"error":"not_connected",...}` yields
      `RHSFetchError` with that code; on HTML/text 502 yields
      `internal_error` and does not throw `SyntaxError`
- [ ] **Gate 7.2:** `doFetchWithResponse` on that same HTML 502 still
      throws `ClientError` with the raw body (test name in client tests
      item 5)
- [ ] `getRHSIssues` / `getRHSStatuses` hit
      `/plugins/jira/api/v2/rhs/issues` and `/plugins/jira/api/v2/rhs/statuses`
      with snake_case params; no double `/api/v2`
- [ ] **Gate 7.1:** test
      `a second fetchRHSIssues for the same instance tab sort while in flight issues no second network call`
      asserts `fetchMock` call count **1** (and still 1 after both
      promises settle)
- [ ] A different `{instance, tab, sort}` while in flight **does** fetch
      again; the same triple **after** settle also fetches again
- [ ] Changing sort via `fetchRHSIssues` replaces the list and clears the
      cursor; `loadMoreRHSIssues` concatenates and sends `next_page_token`
- [ ] During a hung page-1 fetch, `rhsLoading === true` and
      `rhsIssues.length === 0` (loading ≠ empty)
- [ ] `rhsError` stores `RHSErrorCode` (e.g. `not_connected`), not a boolean
- [ ] In-flight map is module-scope, keyed by
      `JSON.stringify({instance, kind, key, id, sort})`, set synchronously,
      cleared in `finally`
- [ ] `rhs_view_state` round-trips, swallows `localStorage` throws, rejects
      unknown tab / bad sort / category-without-key
- [ ] `actions/index.ts` only re-exports; thunks live in `actions/rhs.ts`
- [ ] No `rhsStatuses` slice; `fetchRHSStatuses` returns `{data}|{error}`
- [ ] Fetches are **not** gated on `rhs_enabled` or `hasCloudInstance`
- [ ] `webapp/src/plugin.tsx`, `server/**`, RHS UI, admin picker, `en.json`
      untouched
- [ ] `npm run lint` and `npm run test` (full suite) pass
- [ ] No `eslint-disable`, no `FormattedMessage`, no `debounce-promise` on
      the new path

---

## Commit checkpoint (orchestration, not this engineer)

Do not commit. The lead commits Step 7 after review. Suggested message
when they do: RHS data layer with JSON-error fetch helper and in-flight
debounce. Do not push.

---

## Implementation Summary

**Implementer:** IE7 (new team member). Phase W2 only. Not committed. Not pushed. W3 not started.

### Files

**Created**

- `webapp/src/actions/rhs.ts` — setters, restore, fetch/load-more/refresh thunks, module-scope in-flight `Map`, `resetRHSIssuesInFlight`
- `webapp/src/actions/rhs.test.ts` — Gate 7.1 debounce call-count + sort reset + append + loading-vs-empty
- `webapp/src/client/index.test.ts` — JSON helper, text fallback, Gate 7.2 `ClientError` regression, snake_case URLs
- `webapp/src/utils/rhs_view_state.ts` — per-user `localStorage` persist/validate
- `webapp/src/utils/rhs_view_state.test.ts` — round-trip, isolation, throw swallow, invalid tab/sort

**Modified**

- `webapp/src/types/model.ts` — RHS DTO / tab / sort / error-code types after `PluginSettings`
- `webapp/src/client/index.ts` — additive `doFetchJSON` + `getRHSIssues` + `getRHSStatuses` + `RHSFetchError` (imports + append after `buildQueryString` only)
- `webapp/src/action_types/index.ts` — 8 RHS constants
- `webapp/src/reducers/index.ts` — relative `../action_types`; 9 RHS slices; no `rhsStatuses`
- `webapp/src/actions/index.ts` — re-export only from `./rhs` (603 / 650 non-blank)
- `webapp/src/testlib/test-utils.tsx` — RHS defaults on `defaultMockState`

**Untouched (as required):** `doFetch` / `doFetchWithResponse` bodies, `plugin.tsx`, `server/**`, RHS UI, admin picker, `en.json`, selectors, `debounce-promise`.

### Gate 7 — `doFetch` unchanged

`git diff 6e6e073 -- webapp/src/client/index.ts` is import additions + appended helpers only. No hunks on `doFetch` or `doFetchWithResponse`. Old helper still `response.text()` + `ClientError`. New helper uses `parseRHSErrorBody`.

### Gate 7 — debounce test

**Name:** `a second fetchRHSIssues for the same instance tab sort while in flight issues no second network call`

Hanging `global.fetch` (unresolved Promise). Two sync `dispatch(fetchRHSIssues(sameArgs))`. Asserts `fetchMock` call count **1** before resolve and **1** after both promises settle. Key is `{instance, kind, key, id, sort}` (`name` not included).

### Test results

| Command | Result |
|---------|--------|
| `npm run lint` | pass (0 errors) |
| `npm run test` (full suite) | **17 suites / 119 tests** passed (W1 14/93 + 3 suites / 26 tests) |
| `npm run check-types` | **249** pre-existing `error TS` (unchanged). No W2-created files in the error list. `actions/index.ts` errors are the pre-existing `getState: GlobalState` lie. |

### Deviations (lint-required only; no design changes)

1. `sort-imports` ASCII order: `RHS_*` constants after `RHSErrorCode` / `RHSTab*` in `reducers/index.ts` and `utils/rhs_view_state.ts`. Plan listed CONST-first “alphabetize”.
2. Client import is `{RHSFetchError, getRHSIssues, getRHSStatuses}` (uppercase first) instead of the plan’s visual order.
3. `toRHSFetchError` assigns `const errorCode: RHSErrorCode = 'internal_error'` so the planned `RHSErrorCode` import is used (`unused-imports`).
4. `saveRHSViewState` empty `catch` uses a comment instead of `return` (`no-useless-return`).
5. `makeRHSStore` uses `reducer({} as any, {type: '@@INIT'} as any)` instead of `reducer(undefined, …)` (`no-undefined`).

### Blockers

None. Ready for lead review / Step 7 commit. Do not start W3.
