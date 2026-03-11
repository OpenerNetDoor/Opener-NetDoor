#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${DEPLOY_DIR}/.." && pwd)"
ENV_FILE="${DEPLOY_DIR}/.env"
ENV_TEMPLATE="${DEPLOY_DIR}/.env.example"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.yml"

log() {
  printf '[opener-netdoor] %s\n' "$*"
}

warn() {
  printf '[opener-netdoor][warn] %s\n' "$*" >&2
}

die() {
  printf '[opener-netdoor][error] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

ensure_env_file() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    cp "${ENV_TEMPLATE}" "${ENV_FILE}"
    log "created ${ENV_FILE} from template"
  fi
}

load_env_file() {
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
}

upsert_env() {
  local key="$1"
  local value="$2"
  if grep -q "^${key}=" "${ENV_FILE}"; then
    local escaped
    escaped="$(printf '%s' "${value}" | sed -e 's/[\\/&]/\\\\&/g')"
    sed -i "s|^${key}=.*|${key}=${escaped}|" "${ENV_FILE}"
  else
    printf '%s=%s\n' "${key}" "${value}" >>"${ENV_FILE}"
  fi
}

is_placeholder() {
  local value="$1"
  [[ -z "${value}" ]] && return 0
  [[ "${value}" == change_me* ]] && return 0
  [[ "${value}" == *dev-secret-change-me* ]] && return 0
  [[ "${value}" == *stage5-dev-signing-secret* ]] && return 0
  return 1
}

random_hex() {
  local bytes="$1"
  openssl rand -hex "${bytes}"
}

generate_uuid() {
  if [[ -r /proc/sys/kernel/random/uuid ]]; then
    cat /proc/sys/kernel/random/uuid
    return
  fi
  uuidgen
}

detect_public_ipv4() {
  local ip
  ip="$(curl -4fsS --max-time 5 https://api.ipify.org || true)"
  if [[ -n "${ip}" ]]; then
    printf '%s\n' "${ip}"
    return
  fi
  ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
  if [[ -n "${ip}" ]]; then
    printf '%s\n' "${ip}"
    return
  fi
  printf '127.0.0.1\n'
}

compose() {
  docker compose \
    --project-name "${COMPOSE_PROJECT_NAME:-openernetdoor}" \
    --env-file "${ENV_FILE}" \
    -f "${COMPOSE_FILE}" "$@"
}

configure_firewall_if_active() {
  if ! command -v ufw >/dev/null 2>&1; then
    return
  fi
  if ! ufw status 2>/dev/null | grep -qi 'Status: active'; then
    return
  fi
  log "detected active ufw; ensuring public ports are allowed"
  ufw allow 80/tcp >/dev/null 2>&1 || true
  if [[ "${HTTPS_ENABLED:-false}" == "true" ]]; then
    ufw allow 443/tcp >/dev/null 2>&1 || true
  fi
  if [[ "${RUNTIME_ENABLED:-true}" == "true" ]]; then
    ufw allow "${RUNTIME_VLESS_PORT:-8443}"/tcp >/dev/null 2>&1 || true
  fi
}

print_compose_hint() {
  cat <<'EOF'
Useful commands:
  docker compose --env-file deploy/.env -f deploy/docker-compose.yml ps
  docker compose --env-file deploy/.env -f deploy/docker-compose.yml logs -f --tail=200
  bash deploy/upgrade.sh
  bash deploy/upgrade.sh --rotate-admin-secret
  bash deploy/uninstall.sh
EOF
}
