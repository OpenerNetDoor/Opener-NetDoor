#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

BASE_URL="${1:-${PUBLIC_BASE_URL:-http://127.0.0.1}}"
TIMEOUT_SECONDS="${2:-180}"

require_cmd curl

wait_for_http() {
  local url="$1"
  local timeout="$2"
  local start
  start="$(date +%s)"
  while true; do
    if curl -fsS --max-time 5 "${url}" >/dev/null 2>&1; then
      return 0
    fi
    if (( $(date +%s) - start >= timeout )); then
      return 1
    fi
    sleep 2
  done
}

load_env_file

log "waiting for gateway readiness at ${BASE_URL}/v1/ready"
if ! wait_for_http "${BASE_URL}/v1/ready" "${TIMEOUT_SECONDS}"; then
  die "gateway readiness check timed out"
fi

log "waiting for admin panel at ${BASE_URL}/login"
if ! wait_for_http "${BASE_URL}/login" "${TIMEOUT_SECONDS}"; then
  die "admin panel check timed out"
fi

log "readiness checks passed"
