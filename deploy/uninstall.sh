#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./scripts/lib.sh
source "${SCRIPT_DIR}/scripts/lib.sh"

PURGE_DATA="false"
PURGE_STATE="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --purge-data)
      PURGE_DATA="true"
      ;;
    --purge-state)
      PURGE_STATE="true"
      ;;
    -h|--help)
      cat <<'EOF'
Usage:
  bash deploy/uninstall.sh [--purge-data] [--purge-state]

Options:
  --purge-data   Remove postgres/caddy volumes.
  --purge-state  Remove deploy/state artifacts (including bootstrap token file).
EOF
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
  shift
done

ensure_env_file
load_env_file

if [[ "${PURGE_DATA}" == "true" ]]; then
  log "stopping stack and removing persistent volumes"
  compose down --remove-orphans --volumes
else
  log "stopping stack"
  compose down --remove-orphans
fi

if [[ "${PURGE_STATE}" == "true" ]]; then
  rm -rf "${DEPLOY_DIR}/state"
  log "removed ${DEPLOY_DIR}/state"
fi

cat <<'EOF'

Opener NetDoor uninstall completed.
If you kept volumes, data can be restored by running:
  bash deploy/install.sh

EOF
