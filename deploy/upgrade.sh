#!/usr/bin/env bash
set -euo pipefail

UPGRADE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./scripts/lib.sh
source "${UPGRADE_DIR}/scripts/lib.sh"

ROTATE_ADMIN_SECRET="false"


run_upgrade_helper() {
  local helper="$1"
  shift || true
  local helper_path="${UPGRADE_DIR}/scripts/${helper}"
  [[ -f "${helper_path}" ]] || die "missing deploy helper script: ${helper_path}"
  bash "${helper_path}" "$@"
}

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

run_upgrade_helper "generate-reality-keys.sh"
load_env_file
run_upgrade_helper "render-caddyfile.sh"

log "pulling latest base images"
compose pull postgres redis nats caddy xray || warn "compose pull completed with warnings"

log "updating infrastructure"
compose up -d postgres redis nats

log "running migrations"
compose run --rm migrate

log "rebuilding control plane services (core-platform + api-gateway)"
compose up -d --build core-platform api-gateway
run_upgrade_helper "wait-control-plane.sh" "240"

owner_args=()
if [[ "${ROTATE_ADMIN_SECRET}" == "true" ]]; then
  owner_args+=(--rotate-secret)
fi
owner_info="$(run_upgrade_helper "bootstrap-owner.sh" "${owner_args[@]}")"
admin_access_url="$(echo "${owner_info}" | awk -F= '/^ADMIN_ACCESS_URL=/{print $2}' | tail -n1)"

runtime_info="$(run_upgrade_helper "bootstrap-runtime.sh")"
runtime_node_id="$(echo "${runtime_info}" | awk -F= '/^RUNTIME_NODE_ID=/{print $2}' | tail -n1)"

log "restarting xray runtime service"
compose up -d --force-recreate xray
run_upgrade_helper "check-runtime.sh" "180"

log "rebuilding panel and reverse proxy"
compose up -d --build admin-web caddy
run_upgrade_helper "wait-for-ready.sh" "${PUBLIC_BASE_URL}" "240"

cat <<EOF

Opener NetDoor upgrade completed.
Panel URL: ${PUBLIC_BASE_URL}
Admin access URL: ${admin_access_url}
HTTPS enabled: ${HTTPS_ENABLED}
Runtime node: ${runtime_node_id}
Subscription URL template: ${PUBLIC_BASE_URL}/${SUBSCRIPTION_ACCESS_SECRET}/<USER_UUID>/#<USERNAME>

EOF

print_compose_hint
