#!/usr/bin/env bash
set -euo pipefail

log() { printf '[cloud-agent-install] %s\n' "$*" >&2; }

is_true() {
    case "${1:-}" in
        1|true|TRUE|yes|YES) return 0 ;;
        *) return 1 ;;
    esac
}

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

export GOPATH="${GOPATH:-$HOME/go}"
export PATH="/usr/local/go/bin:$GOPATH/bin:/usr/local/bin:$PATH"

require() {
    command -v "$1" >/dev/null 2>&1 || {
        log "Missing required tool: $1"
        exit 1
    }
}

require go
require node
require npm
require docker
require aws

if ! is_true "${CLOUD_AGENT_SKIP_GO_MOD:-}"; then
    log "Hydrating Go modules"
    go mod download
fi

if ! is_true "${CLOUD_AGENT_SKIP_BUILD_TOOLS:-}"; then
    log "Running make apply"
    mkdir -p build/bin
    make apply
fi

if ! is_true "${CLOUD_AGENT_SKIP_WEBAPP_DEPS:-}" && [[ -f webapp/package-lock.json ]]; then
    log "Hydrating webapp dependencies"
    npm ci --prefix webapp
fi

if ! is_true "${CLOUD_AGENT_SKIP_E2E_DIST:-}"; then
    log "Building the e2e plugin bundle (Atlassian URLs point at the fake Jira)"
    MM_SERVICESETTINGS_ENABLEDEVELOPER=1 GOARCH=amd64 make e2e-dist \
        || log "make e2e-dist failed; run it manually before deploying"
fi

if ! is_true "${CLOUD_AGENT_SKIP_PLAYWRIGHT:-}" && [[ -f e2e/playwright/package.json ]]; then
    log "Installing Playwright browsers for e2e/"
    if [[ -f e2e/playwright/package-lock.json ]]; then
        npm ci --prefix e2e/playwright
    else
        npm install --prefix e2e/playwright
    fi
    # Satisfies the Makefile's e2e/playwright/node_modules target so `make e2e` skips a second npm ci.
    touch e2e/playwright/node_modules
    npx --prefix e2e/playwright playwright install chromium --with-deps \
        || log "Playwright browser install failed; will retry next boot"
    log "Caching tsx for the fake Jira launcher"
    (cd e2e/playwright && npx --yes tsx@4.23.15 --version >/dev/null) \
        || log "tsx download failed; the jira-stack terminal will retry"
fi

log "Tool versions:"
node --version 2>&1 | sed 's/^/  node /' >&2 || true
go version 2>&1 | sed 's/^/  /' >&2 || true
docker --version 2>&1 | sed 's/^/  /' >&2 || true
aws --version 2>&1 | sed 's/^/  /' >&2 || true

log "Cloud agent install complete."
