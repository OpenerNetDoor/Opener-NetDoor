#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

require_cmd curl
require_cmd jq

ensure_env_file
load_env_file

if [[ "${RUNTIME_ENABLED:-true}" != "true" ]]; then
  log "runtime is disabled; skipping xray bootstrap"
  exit 0
fi

core_base="http://127.0.0.1:8081"
owner_tenant="${OWNER_SCOPE_ID}"
node_hostname="${RUNTIME_PUBLIC_HOST:-${PUBLIC_HOST}}"
node_region="self-hosted"

mkdir -p "${DEPLOY_DIR}/state/xray"

log "resolving runtime node for tenant ${owner_tenant}"
nodes_json="$(curl -fsS "${core_base}/internal/v1/nodes?tenant_id=${owner_tenant}&limit=20&offset=0")"
node_id="$(printf '%s' "${nodes_json}" | jq -r '.items[0].id // empty')"

if [[ -z "${node_id}" ]]; then
  log "no node found; creating owner runtime node"
  create_payload="$(jq -n \
    --arg tenant_id "${owner_tenant}" \
    --arg region "${node_region}" \
    --arg hostname "${node_hostname}" \
    --arg agent_version "owner-runtime" \
    --arg contract_version "${NODE_CONTRACT_VERSION}" \
    --argjson capabilities '["heartbeat.v1","provisioning.v1","xray.runtime.v1"]' \
    '{tenant_id:$tenant_id, region:$region, hostname:$hostname, agent_version:$agent_version, contract_version:$contract_version, capabilities:$capabilities}')"
  created_json="$(curl -fsS -X POST -H "Content-Type: application/json" -d "${create_payload}" "${core_base}/internal/v1/nodes")"
  node_id="$(printf '%s' "${created_json}" | jq -r '.id')"
fi

if [[ -z "${node_id}" || "${node_id}" == "null" ]]; then
  die "failed to resolve node id for runtime bootstrap"
fi

log "generating runtime config for node ${node_id}"
config_resp="$(curl -fsS "${core_base}/internal/v1/nodes/runtime/config?tenant_id=${owner_tenant}&node_id=${node_id}")"
printf '%s' "${config_resp}" | jq -r '.config_json' > "${DEPLOY_DIR}/state/xray/config.json"

apply_payload="$(jq -n --arg tenant_id "${owner_tenant}" --arg node_id "${node_id}" '{tenant_id:$tenant_id,node_id:$node_id}')"
curl -fsS -X POST -H "Content-Type: application/json" -d "${apply_payload}" "${core_base}/internal/v1/nodes/runtime/apply" >/dev/null

log "runtime config applied for node ${node_id}"
printf 'RUNTIME_NODE_ID=%s\n' "${node_id}"
printf 'XRAY_CONFIG_FILE=%s\n' "${DEPLOY_DIR}/state/xray/config.json"
