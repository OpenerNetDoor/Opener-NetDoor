#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./scripts/lib.sh
source "${SCRIPT_DIR}/scripts/lib.sh"

ROTATE_TOKEN="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rotate-owner-token)
      ROTATE_TOKEN="true"
      ;;
    -h|--help)
      cat <<'EOF'
Usage:
  bash deploy/upgrade.sh [--rotate-owner-token]
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
ensure_env_file
load_env_file

bash "${SCRIPT_DIR}/scripts/render-caddyfile.sh"

log "pulling latest base images"
compose pull postgres redis nats caddy || warn "compose pull completed with warnings"

log "updating infrastructure"
compose up -d postgres redis nats

log "running migrations"
compose run --rm migrate

log "rebuilding application services"
compose up -d --build core-platform api-gateway admin-web caddy

bash "${SCRIPT_DIR}/scripts/wait-for-ready.sh" "${PUBLIC_BASE_URL}" "240"

bootstrap_args=()
if [[ "${ROTATE_TOKEN}" == "true" ]]; then
  bootstrap_args+=(--rotate-token)
fi
bash "${SCRIPT_DIR}/scripts/bootstrap-owner.sh" "${bootstrap_args[@]}" >/dev/null

cat <<EOF

Opener NetDoor upgrade completed.
Panel URL: ${PUBLIC_BASE_URL}
HTTPS enabled: ${HTTPS_ENABLED}

EOF

print_compose_hint
