#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

require_cmd docker
require_cmd jq

ensure_env_file
load_env_file

if [[ "${RUNTIME_ENABLED:-true}" != "true" ]]; then
  log "runtime is disabled; skipping xray bootstrap"
  exit 0
fi

if [[ -z "${OWNER_SCOPE_ID:-}" ]]; then
  die "OWNER_SCOPE_ID is required for runtime bootstrap"
fi

network_name="${COMPOSE_PROJECT_NAME:-openernetdoor}_backend"
owner_tenant="${OWNER_SCOPE_ID}"
node_hostname="${RUNTIME_PUBLIC_HOST:-${PUBLIC_HOST}}"
node_region="self-hosted"
core_base="http://core-platform:8081"

mkdir -p "${DEPLOY_DIR}/state/xray"

net_curl() {
  docker run --rm --network "${network_name}" curlimages/curl:8.12.1 -fsS "$@"
}

wait_core_ready() {
  local attempt=0
  local max_attempts=120
  while (( attempt < max_attempts )); do
    if net_curl "${core_base}/internal/ready" >/dev/null 2>&1; then
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 2
  done
  return 1
}

log "waiting for core-platform readiness via compose network"
if ! wait_core_ready; then
  die "core-platform is not ready inside compose network"
fi

log "resolving runtime node for tenant ${owner_tenant}"
nodes_json="$(net_curl "${core_base}/internal/v1/nodes?tenant_id=${owner_tenant}&limit=200&offset=0")"
node_id="$(printf '%s' "${nodes_json}" | jq -r --arg host "${node_hostname}" '([.items[]? | select(((.capabilities // []) | index("local.default.v1"))) | .id][0] // [.items[]? | select(.hostname == $host) | .id][0] // empty)')"

if [[ -z "${node_id}" ]]; then
  log "no node found; creating owner runtime node"
  create_payload="$(jq -n \
    --arg tenant_id "${owner_tenant}" \
    --arg region "${node_region}" \
    --arg hostname "${node_hostname}" \
    --arg agent_version "owner-runtime" \
    --arg contract_version "${NODE_CONTRACT_VERSION}" \
    --argjson capabilities '["heartbeat.v1","provisioning.v1","xray.runtime.v1","protocol.vless_reality.v1","local.default.v1"]' \
    '{tenant_id:$tenant_id, region:$region, hostname:$hostname, agent_version:$agent_version, contract_version:$contract_version, capabilities:$capabilities}')"
  created_json="$(net_curl -X POST -H "Content-Type: application/json" -d "${create_payload}" "${core_base}/internal/v1/nodes")"
  node_id="$(printf '%s' "${created_json}" | jq -r '.id // empty')"
fi

if [[ -z "${node_id}" ]]; then
  die "failed to resolve node id for runtime bootstrap"
fi

keys_json="$(net_curl "${core_base}/internal/v1/access-keys?tenant_id=${owner_tenant}&status=active&limit=10000&offset=0" || true)"
active_vless_keys=0
if [[ -n "${keys_json}" ]]; then
  active_vless_keys="$(printf '%s' "${keys_json}" | jq '[.items[]? | select((.key_type | ascii_downcase) == "vless")] | length' 2>/dev/null || echo 0)"
fi

log "generating runtime config for node ${node_id}"
config_resp="$(net_curl "${core_base}/internal/v1/nodes/runtime/config?tenant_id=${owner_tenant}&node_id=${node_id}")"
printf '%s' "${config_resp}" | jq -r '.config_json' > "${DEPLOY_DIR}/state/xray/config.json"

if ! jq -e '.inbounds | length > 0' "${DEPLOY_DIR}/state/xray/config.json" >/dev/null 2>&1; then
  die "generated xray config has empty inbounds"
fi

clients_count="$(jq '[.inbounds[]?.settings.clients // [] | length] | add // 0' "${DEPLOY_DIR}/state/xray/config.json")"
if [[ "${active_vless_keys}" -gt 0 && "${clients_count}" -lt 1 ]]; then
  die "generated xray config has zero clients while active vless keys exist"
fi

apply_payload="$(jq -n --arg tenant_id "${owner_tenant}" --arg node_id "${node_id}" '{tenant_id:$tenant_id,node_id:$node_id}')"
net_curl -X POST -H "Content-Type: application/json" -d "${apply_payload}" "${core_base}/internal/v1/nodes/runtime/apply" >/dev/null

log "runtime config applied for node ${node_id}"
printf 'RUNTIME_NODE_ID=%s\n' "${node_id}"
printf 'XRAY_CONFIG_FILE=%s\n' "${DEPLOY_DIR}/state/xray/config.json"
printf 'XRAY_CONFIG_CLIENTS=%s\n' "${clients_count}"
printf 'ACTIVE_VLESS_KEYS=%s\n' "${active_vless_keys}"



