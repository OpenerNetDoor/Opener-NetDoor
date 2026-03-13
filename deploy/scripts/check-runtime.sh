#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

TIMEOUT_SECONDS="${1:-120}"

require_cmd docker
require_cmd jq

ensure_env_file
load_env_file

if [[ "${RUNTIME_ENABLED:-true}" != "true" ]]; then
  log "runtime is disabled; skipping xray checks"
  exit 0
fi

config_file="${DEPLOY_DIR}/state/xray/config.json"
[[ -f "${config_file}" ]] || die "xray config file is missing: ${config_file}"

inbounds_count="$(jq '.inbounds | length' "${config_file}")"
if [[ "${inbounds_count}" -lt 1 ]]; then
  die "xray config has empty inbounds"
fi

client_count="$(jq '[.inbounds[]?.settings.clients // [] | length] | add // 0' "${config_file}")"

network_name="${COMPOSE_PROJECT_NAME:-openernetdoor}_backend"
net_curl() {
  docker run --rm --network "${network_name}" curlimages/curl:8.12.1 -fsS "$@"
}

owner_tenant="${OWNER_SCOPE_ID:-}"
active_vless_keys=0
if [[ -n "${owner_tenant}" ]]; then
  keys_json="$(net_curl "http://core-platform:8081/internal/v1/access-keys?tenant_id=${owner_tenant}&status=active&limit=10000&offset=0" || true)"
  if [[ -n "${keys_json}" ]]; then
    active_vless_keys="$(printf '%s' "${keys_json}" | jq '[.items[]? | select((.key_type | ascii_downcase) == "vless")] | length' 2>/dev/null || echo 0)"
  fi
fi

if [[ "${active_vless_keys}" -gt 0 && "${client_count}" -lt 1 ]]; then
  die "xray config has zero clients while active vless keys exist"
fi

log "xray config check: inbounds=${inbounds_count}, clients=${client_count}, active_vless_keys=${active_vless_keys}"

compose up -d --force-recreate xray >/dev/null

container_id="$(compose ps -q xray)"
[[ -n "${container_id}" ]] || die "xray container is not running"

port_mappings="$(docker inspect -f '{{json .NetworkSettings.Ports}}' "${container_id}" 2>/dev/null || true)"
if [[ "${port_mappings}" != *"${RUNTIME_VLESS_PORT:-8443}/tcp"* ]]; then
  die "xray port mapping for ${RUNTIME_VLESS_PORT:-8443}/tcp is missing"
fi

wait_tcp_local() {
  local host="$1"
  local port="$2"
  local timeout="$3"
  local start
  start="$(date +%s)"
  while true; do
    if timeout 2 bash -c "</dev/tcp/${host}/${port}" >/dev/null 2>&1; then
      return 0
    fi
    if (( $(date +%s) - start >= timeout )); then
      return 1
    fi
    sleep 2
  done
}

if ! wait_tcp_local "127.0.0.1" "${RUNTIME_VLESS_PORT:-8443}" "${TIMEOUT_SECONDS}"; then
  die "xray did not become reachable on 127.0.0.1:${RUNTIME_VLESS_PORT:-8443}"
fi

public_target="${PUBLIC_HOST:-127.0.0.1}"
if [[ "${public_target}" != "127.0.0.1" && "${public_target}" != "localhost" ]]; then
  if wait_tcp_local "${public_target}" "${RUNTIME_VLESS_PORT:-8443}" "10"; then
    log "public host probe succeeded: ${public_target}:${RUNTIME_VLESS_PORT:-8443}"
  else
    warn "public host probe failed: ${public_target}:${RUNTIME_VLESS_PORT:-8443}"
  fi
fi

log "xray runtime is listening on 0.0.0.0:${RUNTIME_VLESS_PORT:-8443} (host publish confirmed)"
