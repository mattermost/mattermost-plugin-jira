# Cursor Cloud Agent Environment

This directory defines the checked-in environment for Cursor Cloud Agents.

- `environment.json` selects the Dockerfile build, declares the Mattermost (8065) and fake Jira (8090) ports, and defines the `jira-stack` terminal, which runs `make e2e-env-up` and then the fake Jira in the foreground.
- `Dockerfile` installs Go, Node.js, Docker-in-Docker support, Docker Compose, AWS CLI, Chromium runtime libraries, and cached Mattermost/Postgres image archives.
- `fake-jira.ts` runs the E2E suite's fake Jira (`e2e/playwright/fake-jira/`) on its own via `tsx`, so agents can use the plugin from a browser. `--reset` restores its seeded state.
- `scripts/cloud-agent-install.sh` hydrates Go, webapp and e2e dependencies, runs `make apply`, builds the e2e bundle (`make e2e-dist`, Atlassian URLs pointed at the fake) and caches `tsx`.
- `scripts/cloud-agent-start.sh` maps `fake-jira` and `e2e.atlassian.net` to `127.0.0.1` in `/etc/hosts`, starts `dockerd`, fixes socket permissions, logs in to Docker Hub when credentials are configured, loads cached images, and materializes `.cursor/AGENTS.md`.
- `cursor.md` contains cloud-only instructions for running Mattermost with the fake Jira and deploying this plugin.

`.cursor/AGENTS.md` is generated at cloud-agent startup from `cursor.md` and should not be committed.

## Validation

From the repository root:

```bash
python3 -m json.tool .cursor/environment.json >/dev/null
bash -n .cursor/scripts/cloud-agent-install.sh && bash -n .cursor/scripts/cloud-agent-start.sh
docker build --check -f .cursor/Dockerfile .cursor
docker build -f .cursor/Dockerfile .cursor/
```

The Dockerfile intentionally fetches `mattermostdevelopment/mattermost-enterprise-edition:master` and `postgres:16-alpine` during image build so cloud-agent startup can load them locally before running Mattermost. The Mattermost development image is pinned to `linux/amd64` because the `master` tag does not publish an arm64 image.

## Expected Secrets

Configure these in the [Cursor Cloud Agents dashboard](https://cursor.com/dashboard/cloud-agents) as environment-scoped secrets for the Jira Cloud Agent environment.

- AWS uploads use the standard AWS CLI environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and an artifact S3 destination. The image only supplies the `aws` binary.
- Docker Hub pulls use the same variable names as CI: `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`. The start hook runs `docker login` after `dockerd` is ready. Mark `DOCKERHUB_TOKEN` as **redacted** in the dashboard. When both are set, fallback registry pulls avoid anonymous Docker Hub rate limits.
