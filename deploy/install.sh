#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./scripts/lib.sh
source "${INSTALL_DIR}/scripts/lib.sh"

DOMAIN=""
LE_EMAIL=""
PUBLIC_IP=""
MODE=""
SKIP_DOCKER_INSTALL="false"
ROTATE_ADMIN_SECRET="false"

usage() {
  cat <<'EOF'
Usage:
  bash deploy/install.sh [options]

Options:
  --domain <fqdn>            Enable HTTPS with Caddy automatic TLS.
  --email <address>          Email for Let's Encrypt in domain mode.
  --ip <address>             Force IP host in ip mode.
  --ip-mode                  Force plain HTTP IP mode.
  --domain-mode              Force domain mode (requires --domain).
  --skip-docker-install      Do not install Docker automatically.
  --rotate-admin-secret      Regenerate admin access secret.
  -h, --help                 Show help.
EOF
}


run_install_helper() {
  local helper="$1"
  shift || true
  local helper_path="${INSTALL_DIR}/scripts/${helper}"
  [[ -f "${helper_path}" ]] || die "missing deploy helper script: ${helper_path}"
  bash "${helper_path}" "$@"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain)
      DOMAIN="${2:-}"
      shift
      ;;
    --email)
      LE_EMAIL="${2:-}"
      shift
      ;;
    --ip)
      PUBLIC_IP="${2:-}"
      shift
      ;;
    --ip-mode)
      MODE="ip"
      ;;
    --domain-mode)
      MODE="domain"
      ;;
    --skip-docker-install)
      SKIP_DOCKER_INSTALL="true"
      ;;
    --rotate-admin-secret)
      ROTATE_ADMIN_SECRET="true"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
  shift
done

if [[ "$(uname -s)" != "Linux" ]]; then
  die "this installer currently supports Linux hosts (Ubuntu/Debian target)"
fi

if [[ -f /etc/os-release ]]; then
  # shellcheck disable=SC1091
  source /etc/os-release
  case "${ID:-}" in
    ubuntu|debian)
      ;;
    *)
      warn "non-target distro detected (${ID:-unknown}); proceeding in best-effort mode"
      ;;
  esac
fi

require_cmd curl
require_cmd openssl
require_cmd sed
require_cmd awk

if ! command -v docker >/dev/null 2>&1; then
  if [[ "${SKIP_DOCKER_INSTALL}" == "true" ]]; then
    die "docker is required but not installed"
  fi

  if [[ "${EUID}" -ne 0 ]]; then
    die "docker is missing. re-run installer as root to install docker automatically"
  fi

  log "installing docker engine and compose plugin"
  apt-get update -y
  apt-get install -y ca-certificates curl gnupg lsb-release
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/"${ID}"/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${ID} ${VERSION_CODENAME} stable" \
    >/etc/apt/sources.list.d/docker.list
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

if ! docker compose version >/dev/null 2>&1; then
  die "docker compose plugin is required"
fi

if ! command -v jq >/dev/null 2>&1; then
  if [[ "${EUID}" -eq 0 ]]; then
    apt-get update -y
    apt-get install -y jq
  else
    die "jq is required for runtime bootstrap. install jq and retry"
  fi
fi

ensure_env_file
load_env_file

if is_placeholder "${POSTGRES_PASSWORD:-}"; then
  upsert_env "POSTGRES_PASSWORD" "$(random_hex 24)"
fi

jwt_secret_value="${JWT_SECRET:-}"
jwt_secret_len=${#jwt_secret_value}
if is_placeholder "${JWT_SECRET:-}" || [[ ${jwt_secret_len} -lt 32 ]]; then
  upsert_env "JWT_SECRET" "$(random_hex 48)"
fi

session_secret_value="${SESSION_SECRET:-}"
session_secret_len=${#session_secret_value}
if is_placeholder "${SESSION_SECRET:-}" || [[ ${session_secret_len} -lt 32 ]]; then
  upsert_env "SESSION_SECRET" "$(random_hex 48)"
fi

node_signing_value="${NODE_SIGNING_SECRET:-}"
node_signing_len=${#node_signing_value}
if is_placeholder "${NODE_SIGNING_SECRET:-}" || [[ ${node_signing_len} -lt 32 ]]; then
  upsert_env "NODE_SIGNING_SECRET" "$(random_hex 48)"
fi

if [[ -z "${OWNER_SCOPE_ID:-}" ]]; then
  upsert_env "OWNER_SCOPE_ID" "$(generate_uuid)"
fi
if [[ -z "${OWNER_SUBJECT:-}" ]]; then
  upsert_env "OWNER_SUBJECT" "owner"
fi
if [[ -z "${OWNER_EMAIL:-}" ]]; then
  upsert_env "OWNER_EMAIL" "owner@example.com"
fi
if [[ -z "${OWNER_SCOPE_NAME:-}" ]]; then
  upsert_env "OWNER_SCOPE_NAME" "Default_Owner_Scope"
fi
if [[ -z "${SESSION_COOKIE_NAME:-}" ]]; then
  upsert_env "SESSION_COOKIE_NAME" "opener_netdoor_session"
fi
if [[ -z "${SESSION_TTL:-}" ]]; then
  upsert_env "SESSION_TTL" "168h"
fi
if [[ -z "${ADMIN_ACCESS_SECRET:-}" || "${ADMIN_ACCESS_SECRET}" == change_me* ]]; then
  upsert_env "ADMIN_ACCESS_SECRET" "$(random_hex 24)"
fi
if [[ -z "${SUBSCRIPTION_ACCESS_SECRET:-}" || "${SUBSCRIPTION_ACCESS_SECRET}" == change_me* ]]; then
  upsert_env "SUBSCRIPTION_ACCESS_SECRET" "$(random_hex 24)"
fi
if [[ -z "${ADMIN_ACCESS_URL_FILE:-}" ]]; then
  upsert_env "ADMIN_ACCESS_URL_FILE" "./state/admin-access-url.txt"
fi
if [[ -z "${XRAY_IMAGE:-}" ]]; then
  upsert_env "XRAY_IMAGE" "ghcr.io/xtls/xray-core:latest"
fi
if [[ -z "${RUNTIME_ENABLED:-}" ]]; then
  upsert_env "RUNTIME_ENABLED" "true"
fi
if [[ -z "${RUNTIME_VLESS_PORT:-}" ]]; then
  upsert_env "RUNTIME_VLESS_PORT" "8443"
fi
if [[ -z "${RUNTIME_REALITY_SHORT_ID:-}" ]]; then
  upsert_env "RUNTIME_REALITY_SHORT_ID" "$(random_hex 8)"
fi
if [[ -z "${RUNTIME_REALITY_SERVER_NAME:-}" ]]; then
  upsert_env "RUNTIME_REALITY_SERVER_NAME" "www.cloudflare.com"
fi
if [[ -z "${XRAY_DNS_PRIMARY:-}" ]]; then
  upsert_env "XRAY_DNS_PRIMARY" "1.1.1.1"
fi
if [[ -z "${XRAY_DNS_SECONDARY:-}" ]]; then
  upsert_env "XRAY_DNS_SECONDARY" "8.8.8.8"
fi

load_env_file

if [[ -n "${DOMAIN}" ]]; then
  MODE="domain"
fi
if [[ -z "${MODE}" ]]; then
  MODE="${DEPLOY_MODE:-ip}"
fi

if [[ "${MODE}" == "domain" ]]; then
  [[ -n "${DOMAIN}" ]] || DOMAIN="${PUBLIC_HOST:-}"
  [[ -n "${DOMAIN}" ]] || die "domain mode requires --domain <fqdn> or PUBLIC_HOST in deploy/.env"

  if [[ -n "${LE_EMAIL}" ]]; then
    upsert_env "LETSENCRYPT_EMAIL" "${LE_EMAIL}"
  elif [[ -n "${LETSENCRYPT_EMAIL:-}" ]]; then
    LE_EMAIL="${LETSENCRYPT_EMAIL}"
  fi
  [[ -n "${LE_EMAIL}" ]] || die "domain mode requires --email <address> or LETSENCRYPT_EMAIL in deploy/.env"

  upsert_env "DEPLOY_MODE" "domain"
  upsert_env "PUBLIC_HOST" "${DOMAIN}"
  upsert_env "HTTPS_ENABLED" "true"
  upsert_env "PUBLIC_BASE_URL" "https://${DOMAIN}"
  upsert_env "SESSION_SECURE" "true"
  upsert_env "RUNTIME_PUBLIC_HOST" "${DOMAIN}"
else
  if [[ -z "${PUBLIC_IP}" ]]; then
    PUBLIC_IP="$(detect_public_ipv4)"
  fi
  upsert_env "DEPLOY_MODE" "ip"
  upsert_env "PUBLIC_HOST" "${PUBLIC_IP}"
  upsert_env "HTTPS_ENABLED" "false"
  upsert_env "PUBLIC_BASE_URL" "http://${PUBLIC_IP}"
  upsert_env "SESSION_SECURE" "false"
  upsert_env "RUNTIME_PUBLIC_HOST" "${PUBLIC_IP}"
fi

load_env_file

run_install_helper "generate-reality-keys.sh"
load_env_file

run_install_helper "render-caddyfile.sh"
mkdir -p "${DEPLOY_DIR}/state/xray"

if [[ ! -f "${DEPLOY_DIR}/state/xray/config.json" ]]; then
  cat >"${DEPLOY_DIR}/state/xray/config.json" <<EOF
{
  "log": {"loglevel": "warning"},
  "inbounds": [],
  "outbounds": [{"protocol":"freedom"}]
}
EOF
fi

configure_firewall_if_active

log "starting infrastructure containers"
compose up -d postgres redis nats

log "running database migrations"
compose run --rm migrate

log "starting control plane services (core-platform + api-gateway)"
compose up -d --build core-platform api-gateway
run_install_helper "wait-control-plane.sh" "240"

owner_flags=(--print-url)
if [[ "${ROTATE_ADMIN_SECRET}" == "true" ]]; then
  owner_flags+=(--rotate-secret)
fi
owner_info="$(run_install_helper "bootstrap-owner.sh" "${owner_flags[@]}")"
owner_scope_id="$(echo "${owner_info}" | awk -F= '/^OWNER_SCOPE_ID=/{print $2}' | tail -n1)"
admin_access_url="$(echo "${owner_info}" | awk -F= '/^ADMIN_ACCESS_URL=/{print $2}' | tail -n1)"
admin_access_url_file="$(echo "${owner_info}" | awk -F= '/^ADMIN_ACCESS_URL_FILE=/{print $2}' | tail -n1)"

runtime_info="$(run_install_helper "bootstrap-runtime.sh")"
runtime_node_id="$(echo "${runtime_info}" | awk -F= '/^RUNTIME_NODE_ID=/{print $2}' | tail -n1)"

log "starting xray runtime service"
compose up -d --force-recreate xray
run_install_helper "check-runtime.sh" "180"

log "starting panel and reverse proxy"
compose up -d --build admin-web caddy
run_install_helper "wait-for-ready.sh" "${PUBLIC_BASE_URL}" "240"

log "running post-bootstrap health checks"
api_ready_status="failed"
if curl -fsS --max-time 5 "${PUBLIC_BASE_URL}/v1/ready" >/dev/null 2>&1; then
  api_ready_status="ok"
fi
panel_status="failed"
if curl -fsS --max-time 5 "${PUBLIC_BASE_URL}/login" >/dev/null 2>&1; then
  panel_status="ok"
fi
runtime_status="failed"
if compose ps xray 2>/dev/null | grep -qi "running"; then
  runtime_status="running"
fi
runtime_port_status="closed"
if timeout 2 bash -c "</dev/tcp/127.0.0.1/${RUNTIME_VLESS_PORT}" >/dev/null 2>&1; then
  runtime_port_status="open"
fi

compose_ps="$(compose ps --format json 2>/dev/null || true)"

cat <<EOF

Opener NetDoor install completed.

Panel URL: ${PUBLIC_BASE_URL}
Admin access URL: ${admin_access_url}
HTTPS enabled: ${HTTPS_ENABLED}
Hidden owner scope: ${owner_scope_id}

Runtime:
  - Node ID: ${runtime_node_id}
  - Xray service: ${runtime_status}
  - Xray port ${RUNTIME_VLESS_PORT}: ${runtime_port_status}

Service summary:
  - API readiness: ${api_ready_status}
  - Panel reachability: ${panel_status}

Docker services (json):
${compose_ps}

Sensitive files:
  - ${admin_access_url_file}

Subscription URL template:
  - ${PUBLIC_BASE_URL}/${SUBSCRIPTION_ACCESS_SECRET}/<USER_UUID>/#<USERNAME>

Config files:
  - deploy/.env
  - deploy/Caddyfile

EOF

print_compose_hint
