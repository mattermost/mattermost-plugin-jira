# Jira plugin Playwright tests

Hermetic E2E suite: Mattermost runs in Docker, Jira is replaced by `fake-jira/` (started by `global_setup.ts`).
No Jira account or secrets are needed.

```sh
make e2e-env-up     # Postgres + Mattermost on :8065
make e2e            # builds the E2E bundle (make e2e-dist), installs the plugin, runs the suite
make e2e-env-down
```

`make e2e-dist` compiles the Jira Cloud OAuth endpoints to point at fake-jira (the plugin logs a warning on activation).
Never ship that bundle.

Useful overrides, set in the shell or in `e2e/playwright/.env` (copy `.env.example`; docker compose reads it too):

| Variable | Default | Purpose |
|---|---|---|
| `MM_IMAGE` | `mattermostdevelopment/mattermost-enterprise-edition:master` | Server image (amd64; use an arm64 tag on Apple Silicon) |
| `MM_PORT` | `8065` | Host port; must be the same for `e2e-env-up` and `e2e` |
| `JIRA_FLAVOR` | `cloud` | `cloud` (OAuth 2.0 3LO) or `server` (Data Center, OAuth 1.0a) |
| `FAKE_JIRA_PORT` | `8090` | Port fake-jira listens on |
| `PW_HEADLESS` | `true` | `false` to watch the browser |

## How Jira is faked

- The `setup` project installs one instance per run. Cloud goes through the real admin wizard against
  `http://e2e.atlassian.net:8090` with fake-jira acting as `auth.atlassian.com`/`api.atlassian.com`; server uses
  `/jira instance install server http://fake-jira:8090`. Mattermost resolves both hosts via `extra_hosts`, Chromium via
  `--host-resolver-rules`, so the OAuth redirects work.
- `fake-jira/server.ts` implements the endpoints the plugin calls. Unhandled routes return 404 and are logged as
  `[fake-jira] unhandled …` — add a route there when a new flow needs one.
- Tests drive it through `support/fake_jira.ts`: `reset()`, `requests()` (journal of every call the plugin made), and
  `fault(method, path, status)` for error paths.
- Inbound Jira webhooks replay the real payloads in `server/testdata/`; those tests are tagged `@synthetic-inbound`.
- Only `connect.spec.ts` drives the OAuth flow. Other specs connect users with `connectUserViaKV()`
  (`support/kv.ts`), which writes the plugin's connection records into Postgres — keep it in sync with `server/kv.go`.

## Keeping the fake honest

- Every Cloud REST request and response fake-jira handles is validated against Atlassian's published OpenAPI spec;
  the `jira` fixture fails the test on any violation, including Cloud routes not listed in
  `fake-jira/contract/operations.json`.
- New Cloud route: add it to `operations.json`, run `npm run contract:update`, commit `fake-jira/contract/specs/`.
- Nightly, `npm run contract:check` fails if Atlassian changed those operations upstream.
- Nightly (opt-in via repo variable `JIRA_REAL_CLOUD_ENABLED=true` and the secrets listed in `.env.example`, one admin
  and one regular user), `npm run test:real-cloud` checks fake-jira's payloads against a real Cloud site. Locally, copy
  `.env.example` to `.env`. Add a case
  to `real-cloud/parity.spec.ts` for each new fake GET route.

## Gotchas

- The plugin rejects a `localhost` SiteURL, so everything uses `127.0.0.1`.
- Don't use `pw.initSetup()`: it resets server config and disables the plugin. Use the `jira` fixture instead.
