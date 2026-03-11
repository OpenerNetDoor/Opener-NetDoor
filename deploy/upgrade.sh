#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./scripts/lib.sh
source "${SCRIPT_DIR}/scripts/lib.sh"

ROTATE_ADMIN_SECRET="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rotate-admin-secret)
      ROTATE_ADMIN_SECRET="true"
      ;;
    -h|--help)
      cat <<'EOF'
Usage:
  bash deploy/upgrade.sh [--rotate-admin-secret]
EOF
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
  shift
done

require_cmd docker
require_cmd jq
ensure_env_file
load_env_file

bash "${SCRIPT_DIR}/scripts/generate-reality-keys.sh"
load_env_file
bash "${SCRIPT_DIR}/scripts/render-caddyfile.sh"

log "pulling latest base images"
compose pull postgres redis nats caddy xray || warn "compose pull completed with warnings"

log "updating infrastructure"
compose up -d postgres redis nats

log "running migrations"
compose run --rm migrate

log "rebuilding control plane services (core-platform + api-gateway)"
compose up -d --build core-platform api-gateway
bash "${SCRIPT_DIR}/scripts/wait-control-plane.sh" "240"

owner_args=()
if [[ "${ROTATE_ADMIN_SECRET}" == "true" ]]; then
  owner_args+=(--rotate-secret)
fi
owner_info="$(bash "${SCRIPT_DIR}/scripts/bootstrap-owner.sh" "${owner_args[@]}")"
admin_access_url="$(echo "${owner_info}" | awk -F= '/^ADMIN_ACCESS_URL=/{print $2}' | tail -n1)"

runtime_info="$(bash "${SCRIPT_DIR}/scripts/bootstrap-runtime.sh")"
runtime_node_id="$(echo "${runtime_info}" | awk -F= '/^RUNTIME_NODE_ID=/{print $2}' | tail -n1)"

log "restarting xray runtime service"
compose up -d xray

log "rebuilding panel and reverse proxy"
compose up -d --build admin-web caddy
bash "${SCRIPT_DIR}/scripts/wait-for-ready.sh" "${PUBLIC_BASE_URL}" "240"

cat <<EOF

Opener NetDoor upgrade completed.
Panel URL: ${PUBLIC_BASE_URL}
Admin access URL: ${admin_access_url}
HTTPS enabled: ${HTTPS_ENABLED}
Runtime node: ${runtime_node_id}

EOF

print_compose_hint

