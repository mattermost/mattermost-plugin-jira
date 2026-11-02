#!/usr/bin/env bash
set -euo pipefail

log() { printf '[cloud-agent-start] %s\n' "$*" >&2; }

is_true() {
    case "${1:-}" in
        1|true|TRUE|yes|YES) return 0 ;;
        *) return 1 ;;
    esac
}

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

if [[ -f .cursor/cursor.md ]]; then
    cp .cursor/cursor.md .cursor/AGENTS.md
    log "Materialized Cloud Agent instructions at .cursor/AGENTS.md"
fi

if [[ -f /proc/sys/kernel/apparmor_restrict_unprivileged_userns ]]; then
    sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0 >/dev/null 2>&1 || \
        log "Could not relax AppArmor userns restriction; userns-based tests may fail"
fi

apply_docker_socket_acl() {
    if [[ -S /var/run/docker.sock ]]; then
        sudo groupadd -f docker
        sudo usermod -aG docker "$(id -un)" || true
        sudo chgrp docker /var/run/docker.sock >/dev/null 2>&1 || true
        sudo chmod g+rw /var/run/docker.sock >/dev/null 2>&1 || true
        if command -v setfacl >/dev/null 2>&1; then
            sudo setfacl -m "u:$(id -un):rw" /var/run/docker.sock >/dev/null 2>&1 || true
        fi
    fi
}

docker_login_if_configured() {
    if [[ -z "${DOCKERHUB_USERNAME:-}" || -z "${DOCKERHUB_TOKEN:-}" ]]; then
        log "Docker Hub credentials not configured; anonymous pulls may hit rate limits."
        return 0
    fi

    log "Logging in to Docker Hub as ${DOCKERHUB_USERNAME}."
    if echo "${DOCKERHUB_TOKEN}" | docker login -u "${DOCKERHUB_USERNAME}" --password-stdin >/tmp/docker-login.log 2>&1; then
        log "Docker Hub login succeeded."
    else
        log "Docker Hub login failed; see /tmp/docker-login.log."
        tail -n 20 /tmp/docker-login.log >&2 || true
    fi
}

load_image_archive() {
    local image_ref="$1"
    local archive="$2"

    if docker image inspect "$image_ref" >/dev/null 2>&1; then
        return
    fi

    if [[ -f "$archive" ]]; then
        log "Loading $image_ref from $archive"
        docker load -i "$archive"
        return
    fi

    log "Preloaded archive not found for $image_ref; pulling from registry."
    docker pull "$image_ref"
}

if ! command -v docker >/dev/null 2>&1; then
    log "Docker CLI is missing — Cloud Agent image did not build from .cursor/Dockerfile"
    exit 1
fi

apply_docker_socket_acl

if ! docker info >/dev/null 2>&1; then
    log "Starting Docker daemon"
    if command -v service >/dev/null 2>&1; then
        sudo sh -c 'service docker start >/tmp/docker-service-start.log 2>&1' || \
            log "service docker start failed; falling back to direct dockerd"
    fi

    if ! pgrep -x dockerd >/dev/null 2>&1; then
        sudo rm -f /var/run/docker.pid
        sudo sh -c 'nohup dockerd --host=unix:///var/run/docker.sock >/tmp/dockerd.log 2>&1 &'
    fi
fi

for _ in $(seq 1 60); do
    apply_docker_socket_acl
    if docker info >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

if ! docker info >/dev/null 2>&1; then
    log "Docker did not become ready within 60 seconds."
    log "docker service log:"
    tail -n 80 /tmp/docker-service-start.log 2>/dev/null || true
    log "dockerd log:"
    tail -n 120 /tmp/dockerd.log 2>/dev/null || true
    exit 1
fi

log "Docker is ready"
docker version >/dev/null || true
docker compose version >/dev/null || true

docker_login_if_configured

if ! is_true "${CLOUD_AGENT_SKIP_IMAGE_LOAD:-}"; then
    load_image_archive "${MATTERMOST_IMAGE:-mattermostdevelopment/mattermost-enterprise-edition}:${MATTERMOST_IMAGE_TAG:-master}" /opt/cursor-prepulled/mattermost-enterprise-edition.tar
    load_image_archive "${POSTGRES_IMAGE:-postgres}:${POSTGRES_IMAGE_TAG:-16-alpine}" /opt/cursor-prepulled/postgres.tar
fi

log "Cloud agent start complete. Docker is ready."
