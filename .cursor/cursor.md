# Cursor Cloud Agent Guide

This repository uses a Dockerfile-backed Cursor Cloud Agent environment with Docker-in-Docker. The image includes Go, Node.js, Docker, Docker Compose, AWS CLI, Chromium runtime libraries, and preloaded Mattermost Enterprise and Postgres images.

Browser automation and screenshots use Cursor's `computerUse` subagent (a Cursor-provided Chrome desktop). Nothing extra is installed in the image for it.

There is no real Jira in this environment. The plugin runs against the E2E suite's fake Jira (`e2e/playwright/fake-jira/`), started by `.cursor/fake-jira.ts` in the `jira-stack` terminal.

## Skip Flags

Set these environment variables to shorten boot when you already have dependencies or images:

- `CLOUD_AGENT_SKIP_GO_MOD=1` — skip `go mod download`
- `CLOUD_AGENT_SKIP_BUILD_TOOLS=1` — skip `make apply` (manifest/pluginctl build)
- `CLOUD_AGENT_SKIP_WEBAPP_DEPS=1` — skip `npm ci --prefix webapp`
- `CLOUD_AGENT_SKIP_E2E_DIST=1` — skip building the e2e plugin bundle (`make e2e-dist`)
- `CLOUD_AGENT_SKIP_PLAYWRIGHT=1` — skip Playwright browser install (no-op unless `e2e/` exists)
- `CLOUD_AGENT_SKIP_IMAGE_LOAD=1` — skip loading preloaded Docker image archives

When `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured as Cloud Agent secrets, `cloud-agent-start.sh` logs in to Docker Hub so fallback `docker pull` operations avoid anonymous rate limits. Mark `DOCKERHUB_TOKEN` as **redacted** in the dashboard.

## Overview

**Mattermost Jira** is a two-way integration plugin for Jira Cloud, Server, and Data Center:

- **Go server plugin** (`server/`) — compiled with `CGO_ENABLED=0`
- **React/TypeScript webapp** (`webapp/`) — bundled with Webpack

The plugin id is `jira`. Release bundles are named `jira-<version>.tar.gz` under `dist/`; the e2e bundle used here is `dist/e2e/jira-<version>-e2e.tar.gz`. `make apply` generates `server/manifest.go` and `webapp/src/manifest.ts` (both gitignored).

Minimum Mattermost server version is `10.7.0` (`plugin.json`). The preloaded image is Mattermost Enterprise `master`, which satisfies that requirement.

## Start Mattermost And The Fake Jira

The `jira-stack` terminal (`.cursor/environment.json`) does this on boot. To run it by hand:

```bash
make e2e-env-up
cd e2e/playwright && npx --yes tsx@4.23.15 ../../.cursor/fake-jira.ts
```

`make e2e-env-up` starts Postgres and Mattermost from `e2e/playwright/docker-compose.yml`. Mattermost listens on `http://127.0.0.1:8065` and resolves `fake-jira` and `e2e.atlassian.net` to this machine via `extra_hosts`. The fake Jira listens on port `8090`; wait for `[fake-jira] Ready`. The start hook maps the same two hostnames in `/etc/hosts` so the agent's browser can follow the OAuth redirects.

Always open Mattermost at `http://127.0.0.1:8065`, not `localhost`: it is the SiteURL, and the plugin rejects a `localhost` SiteURL.

The server starts with no users. Create the system admin (the Playwright suite uses the same credentials):

```bash
cd e2e/playwright && docker compose exec mattermost /mattermost/bin/mmctl --local user create \
  --email sysadmin@example.com \
  --username sysadmin \
  --password 'Sys@dmin-sample1' \
  --system-admin \
  --email-verified
```

Then create a team from the browser.

## Deploy The Plugin

Build the e2e bundle. It compiles the Jira Cloud OAuth and API base URLs to `http://fake-jira:8090`. Do not use `make dist` or `make deploy` here: those bundles talk to the real Atlassian.

```bash
MM_SERVICESETTINGS_ENABLEDEVELOPER=1 GOARCH=amd64 make e2e-dist
MM_SERVICESETTINGS_SITEURL=http://127.0.0.1:8065 \
MM_ADMIN_USERNAME=sysadmin \
MM_ADMIN_PASSWORD='Sys@dmin-sample1' \
  ./build/bin/pluginctl deploy jira dist/e2e/jira-*-e2e.tar.gz
```

Rerun both commands after every change. The install hook builds the bundle the same way. Always set both variables together:

- `MM_SERVICESETTINGS_ENABLEDEVELOPER=1` builds a single platform instead of five.
- `GOARCH=amd64` makes that platform `linux-amd64`, the only one the Mattermost image runs. Without it, an arm64 agent builds `plugin-linux-arm64`. `server/dist` is never cleaned, so the bundle keeps the `plugin-linux-amd64` from an earlier build, and Mattermost silently loads that old code.

In **System Console > Plugins > Jira**, generate the Webhook Secret and the At Rest Encryption Key, then save.

## Connect Jira

Jira Cloud (OAuth 2.0):

1. Run `/jira instance install cloud-oauth http://e2e.atlassian.net:8090`.
2. In the DM from the `jira` bot, click **Configure** and enter organization `http://e2e.atlassian.net:8090`, Client ID `e2e-oauth-client-id`, Client Secret `e2e-oauth-client-secret`.
3. Run `/jira connect`. The OAuth flow completes against the fake.

Jira Server / Data Center (OAuth 1.0a): run `/jira instance install server http://fake-jira:8090`, then `/jira connect`.

The fake is seeded with project `E2E` and issues `E2E-1` and `E2E-2`. Use its control API (see `e2e/playwright/support/fake_jira.ts`):

| Endpoint | Purpose |
|----------|---------|
| `GET http://localhost:8090/__control/requests` | Every Jira call the plugin made |
| `GET http://localhost:8090/__control/contract-violations` | Cloud requests/responses that break Atlassian's OpenAPI contract |
| `POST http://localhost:8090/__control/fault` | Inject an error, e.g. `{"method": "GET", "path": "/rest/api/2/issue/E2E-1", "status": 500}` |
| `POST http://localhost:8090/__control/reset` | Restore the seeded state (also `npx --yes tsx@4.23.15 ../../.cursor/fake-jira.ts --reset` from `e2e/playwright`) |

The fake does not send webhooks. To exercise inbound Jira events, replay payloads from `server/testdata/` the way `e2e/playwright/support/jira.ts` does.

Never use the real-Jira lanes (`make e2e-real`, `npm run test:real-cloud`, `npm run test:real-jira`) or Jira credentials.

## Lint, Test, and Type Check

| Task | Command |
|------|---------|
| Webapp lint | `cd webapp && npm run lint` |
| Webapp type check | `cd webapp && npm run check-types` |
| Webapp unit tests | `cd webapp && npm run test` |
| Server lint | `make check-style` |
| Server unit tests | `make test` or `go test ./...` |
| E2E type check | `cd e2e/playwright && npm run tsc` |
| E2E suite | `make e2e` or `cd e2e/playwright && npx playwright test tests/<spec>.spec.ts` |

`make apply` must run before compiling or testing so generated manifests exist.

The Playwright suite reuses the running fake Jira and resets it, which drops the tokens of users connected by hand. It also reinstalls the plugin with its own settings and reinstalls the Jira instance. Set `JIRA_FLAVOR=server` to run it against the Server flavor.

## Drive The Mattermost UI

Use the `computerUse` subagent's Chrome desktop against the running local Mattermost instance:

1. Open `http://127.0.0.1:8065/login`
2. Sign in as `sysadmin` / `Sys@dmin-sample1`
3. Confirm the Jira plugin is enabled (slash command `/jira` should be available)
4. Screenshot to `/tmp/artifacts/` when a visual record is needed

Do not try to install a separate browser CLI. Computer use is provided by Cursor.

## Upload Screenshot Artifacts

AWS CLI is installed so cloud agents can upload screenshots and other artifacts when AWS credentials and an artifact S3 destination are available.

```bash
mkdir -p /tmp/artifacts
aws sts get-caller-identity
aws s3 cp /tmp/artifacts/mattermost-jira.png <artifact-s3-uri>/mattermost-jira.png
```

Do not print AWS credentials or secret environment variables. If `aws sts get-caller-identity` fails, stop and report the missing AWS configuration instead of attempting to work around credentials.

## Gotchas

- Server tests use HTTP mocks (`httpmock`) and do not require a live Jira or PostgreSQL instance.
- `make apply` must run before `go test` / `make dist`; `server/manifest.go` and `webapp/src/manifest.ts` are generated and gitignored.
- The e2e bundle logs a warning on activation that the Atlassian URLs are overridden. That is expected; never ship it.
- Without `MM_SERVICESETTINGS_ENABLEDEVELOPER=1`, `make e2e-dist` cross-compiles five architectures and is much slower. With it, also set `GOARCH=amd64` (see **Deploy The Plugin**).
- If the browser cannot reach `fake-jira` or `e2e.atlassian.net`, check that `/etc/hosts` maps both to `127.0.0.1` (the start hook adds them).
- Multiple Jira instances require Mattermost Professional, Enterprise, or Enterprise Advanced.

## Troubleshooting

- If Docker Hub rate limits block fallback pulls, configure `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` as Cloud Agent secrets and restart the agent.
- If Docker is unavailable, inspect `/tmp/docker-service-start.log` and `/tmp/dockerd.log`.
- If artifact uploads fail, run `aws sts get-caller-identity` and verify the target S3 URI before retrying.
- If `pluginctl deploy` fails, confirm the admin credentials are exported and the stack is healthy (`cd e2e/playwright && docker compose ps`).
- If the plugin calls fail with `fake-jira has no route for ...`, the flow needs a new route in `e2e/playwright/fake-jira/server.ts`.
- Mattermost logs: `cd e2e/playwright && docker compose logs mattermost`.
- To reset all Mattermost data, run `make e2e-env-down && make e2e-env-up` (Postgres uses tmpfs).
