# Cursor Cloud Agent Guide

This repository uses a Dockerfile-backed Cursor Cloud Agent environment with Docker-in-Docker. The image includes Go, Node.js, Docker, Docker Compose, AWS CLI, Chromium runtime libraries, and preloaded Mattermost Enterprise and Postgres images.

Browser automation and screenshots use Cursor's `computerUse` subagent (a Cursor-provided Chrome desktop). Nothing extra is installed in the image for it.

## Skip Flags

Set these environment variables to shorten boot when you already have dependencies or images:

- `CLOUD_AGENT_SKIP_GO_MOD=1` — skip `go mod download`
- `CLOUD_AGENT_SKIP_BUILD_TOOLS=1` — skip `make apply` (manifest/pluginctl build)
- `CLOUD_AGENT_SKIP_WEBAPP_DEPS=1` — skip `npm ci --prefix webapp`
- `CLOUD_AGENT_SKIP_PLAYWRIGHT=1` — skip Playwright browser install (no-op unless `e2e/` exists)
- `CLOUD_AGENT_SKIP_IMAGE_LOAD=1` — skip loading preloaded Docker image archives

When `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` are configured as Cloud Agent secrets, `cloud-agent-start.sh` logs in to Docker Hub so fallback `docker pull` operations avoid anonymous rate limits. Mark `DOCKERHUB_TOKEN` as **redacted** in the dashboard.

## Overview

**Mattermost Jira** is a two-way integration plugin for Jira Cloud, Server, and Data Center:

- **Go server plugin** (`server/`) — compiled with `CGO_ENABLED=0`
- **React/TypeScript webapp** (`webapp/`) — bundled with Webpack

The plugin id is `jira`. Bundles are named `jira-<version>.tar.gz` under `dist/`. `make apply` generates `server/manifest.go` and `webapp/src/manifest.ts` (both gitignored).

Minimum Mattermost server version is `10.7.0` (`plugin.json`). The preloaded image is Mattermost Enterprise `master`, which satisfies that requirement.

## Start Mattermost

After cloud-agent startup, Docker should be ready and Mattermost/Postgres images should be loaded. Start the stack:

```bash
export MM_IMAGE="${MATTERMOST_IMAGE:-mattermostdevelopment/mattermost-enterprise-edition}:${MATTERMOST_IMAGE_TAG:-master}"
export MM_PLATFORM="${MATTERMOST_PLATFORM:-linux/amd64}"
export POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres}:${POSTGRES_IMAGE_TAG:-16-alpine}"
export MM_DB_USER=mmuser
export MM_DB_PASSWORD=mostest
export MM_DB_NAME=mattermost_test
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=Password123

docker network create mattermost-dev || true
docker rm -f mattermost mm-postgres 2>/dev/null || true
docker volume create mm-postgres-data

docker run -d \
  --name mm-postgres \
  --network mattermost-dev \
  -e POSTGRES_USER="$MM_DB_USER" \
  -e POSTGRES_PASSWORD="$MM_DB_PASSWORD" \
  -e POSTGRES_DB="$MM_DB_NAME" \
  --health-cmd='pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  --health-interval=5s \
  --health-timeout=5s \
  --health-retries=24 \
  -v mm-postgres-data:/var/lib/postgresql/data \
  "$POSTGRES_IMAGE"

until [ "$(docker inspect -f '{{.State.Health.Status}}' mm-postgres)" = "healthy" ]; do
  sleep 2
done

mkdir -p /tmp/mattermost/{config,data,logs,plugins,client-plugins,bleve-indexes}
chmod -R 777 /tmp/mattermost

docker run -d \
  --name mattermost \
  --platform "$MM_PLATFORM" \
  --network mattermost-dev \
  -p 8065:8065 \
  -e MM_SQLSETTINGS_DRIVERNAME=postgres \
  -e "MM_SQLSETTINGS_DATASOURCE=postgres://$MM_DB_USER:$MM_DB_PASSWORD@mm-postgres:5432/$MM_DB_NAME?sslmode=disable&connect_timeout=10" \
  -e MM_SERVICESETTINGS_SITEURL=http://localhost:8065 \
  -e MM_SERVICESETTINGS_ENABLEDEVELOPER=true \
  -e MM_SERVICESETTINGS_ENABLELOCALMODE=true \
  -e MM_PLUGINSETTINGS_ENABLEUPLOADS=true \
  -e MM_PLUGINSETTINGS_ENABLEMARKETPLACE=false \
  -e MM_FILESETTINGS_MAXFILESIZE=256000000 \
  -v /tmp/mattermost/config:/mattermost/config \
  -v /tmp/mattermost/data:/mattermost/data \
  -v /tmp/mattermost/logs:/mattermost/logs \
  -v /tmp/mattermost/plugins:/mattermost/plugins \
  -v /tmp/mattermost/client-plugins:/mattermost/client/plugins \
  -v /tmp/mattermost/bleve-indexes:/mattermost/bleve-indexes \
  "$MM_IMAGE"
```

Wait for Mattermost, then create a system admin:

```bash
until curl -fsS http://localhost:8065/api/v4/system/ping | jq -e '.status == "OK"' >/dev/null; do
  sleep 2
done

docker exec mattermost mmctl --local user search "$MM_ADMIN_USERNAME" | grep -q "$MM_ADMIN_USERNAME" || \
  docker exec mattermost mmctl --local user create \
    --email admin@example.com \
    --username "$MM_ADMIN_USERNAME" \
    --password "$MM_ADMIN_PASSWORD" \
    --system-admin
```

Mattermost will be available on port `8065`.

`MM_FILESETTINGS_MAXFILESIZE=256000000` raises the upload limit so plugin bundles can be deployed via `pluginctl`. The default limit is 100 MB.

## Deploy The Plugin

Use `MM_DEBUG=true` and `MM_SERVICESETTINGS_ENABLEDEVELOPER=1` for faster local-only builds (debug symbols + single platform binary):

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=Password123

MM_DEBUG=true MM_SERVICESETTINGS_ENABLEDEVELOPER=1 make dist
./build/bin/pluginctl deploy jira dist/jira-*.tar.gz
```

Or use the Makefile deploy target (rebuilds via `make dist`):

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=Password123

MM_DEBUG=true MM_SERVICESETTINGS_ENABLEDEVELOPER=1 make deploy
```

For iterative webapp work:

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=Password123

MM_DEBUG=true MM_SERVICESETTINGS_ENABLEDEVELOPER=1 make watch
```

Most plugin code, tests, and UI work do not need a live Jira. Connecting a Jira instance (`/jira install` / `/jira instance install`) is only required when verifying webhooks, issue create/attach, or OAuth flows. See `readme.md` for local Jira Server Docker setup if that is in scope.

## Lint, Test, and Type Check

| Task | Command |
|------|---------|
| Webapp lint | `cd webapp && npm run lint` |
| Webapp type check | `cd webapp && npm run check-types` |
| Webapp unit tests | `cd webapp && npm run test` |
| Server lint | `make check-style` |
| Server unit tests | `make test` or `go test ./...` |
| Full plugin build | `MM_DEBUG=true MM_SERVICESETTINGS_ENABLEDEVELOPER=1 make dist` |

`make apply` must run before compiling or testing so generated manifests exist.

## Drive The Mattermost UI

Use the `computerUse` subagent's Chrome desktop against the running local Mattermost instance:

1. Open `http://localhost:8065/login`
2. Sign in as `admin` / `Password123`
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
- Plugin deploy needs `MM_PLUGINSETTINGS_ENABLEUPLOADS=true` and `FileSettings.MaxFileSize` large enough for the bundle (see **Start Mattermost**). If Mattermost was started without `MM_FILESETTINGS_MAXFILESIZE`, raise it before deploy: `docker exec mattermost mmctl --local config set FileSettings.MaxFileSize 256000000 && docker exec mattermost mmctl --local config reload`.
- Without `MM_SERVICESETTINGS_ENABLEDEVELOPER=1`, `make dist` cross-compiles five architectures and is much slower.
- Connecting Jira to Mattermost requires a SiteURL that Jira can reach. `http://localhost:8065` is enough for in-Mattermost plugin UI and slash-command checks; it is not enough for Jira webhooks or Application Links. Use a tunnel or `host.docker.internal` only when that flow is in scope.
- Multiple Jira instances require Mattermost Professional, Enterprise, or Enterprise Advanced.

## Troubleshooting

- If Docker Hub rate limits block fallback pulls, configure `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` as Cloud Agent secrets and restart the agent.
- If Docker is unavailable, inspect `/tmp/docker-service-start.log` and `/tmp/dockerd.log`.
- If artifact uploads fail, run `aws sts get-caller-identity` and verify the target S3 URI before retrying.
- If the plugin upload fails with `Uploaded plugin size exceeds limit`, confirm `MM_FILESETTINGS_MAXFILESIZE=256000000` was set when starting Mattermost (or use the `mmctl` command in **Gotchas**).
- If the plugin upload fails for other reasons, confirm `MM_PLUGINSETTINGS_ENABLEUPLOADS=true` and the admin credentials are exported.
- If Mattermost is unhealthy, run `docker logs mattermost` and `docker logs mm-postgres`.
- To reset the local Mattermost stack, run `docker rm -f mattermost mm-postgres` and remove `/tmp/mattermost` or `mm-postgres-data` if persisted data is not needed.
