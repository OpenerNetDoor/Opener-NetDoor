#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

TIMEOUT_SECONDS="${1:-180}"

require_cmd docker

ensure_env_file
load_env_file

network_name="${COMPOSE_PROJECT_NAME:-openernetdoor}_backend"

net_curl() {
  docker run --rm --network "${network_name}" curlimages/curl:8.12.1 -fsS --max-time 5 "$1"
}

wait_for_url() {
  local url="$1"
  local timeout="$2"
  local start
  start="$(date +%s)"
  while true; do
    if net_curl "${url}" >/dev/null 2>&1; then
      return 0
    fi
    if (( $(date +%s) - start >= timeout )); then
      return 1
    fi
    sleep 2
  done
}

log "waiting for core-platform readiness via compose network"
if ! wait_for_url "http://core-platform:8081/internal/ready" "${TIMEOUT_SECONDS}"; then
  die "core-platform readiness check timed out"
fi

log "waiting for api-gateway readiness via compose network"
if ! wait_for_url "http://api-gateway:8080/v1/ready" "${TIMEOUT_SECONDS}"; then
  die "api-gateway readiness check timed out"
fi

log "internal control plane readiness checks passed"
