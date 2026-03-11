#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./scripts/lib.sh
source "${SCRIPT_DIR}/scripts/lib.sh"

DOMAIN=""
LE_EMAIL=""
PUBLIC_IP=""
MODE=""
SKIP_DOCKER_INSTALL="false"
ROTATE_TOKEN="false"

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
  --rotate-owner-token       Regenerate owner bootstrap token.
  -h, --help                 Show help.
EOF
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
    --rotate-owner-token)
      ROTATE_TOKEN="true"
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
  upsert_env "OWNER_SCOPE_NAME" "Default Owner Scope"
fi
if [[ -z "${OWNER_TOKEN_TTL_HOURS:-}" ]]; then
  upsert_env "OWNER_TOKEN_TTL_HOURS" "720"
fi
if [[ -z "${OWNER_BOOTSTRAP_TOKEN_FILE:-}" ]]; then
  upsert_env "OWNER_BOOTSTRAP_TOKEN_FILE" "./state/owner-bootstrap-token.txt"
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
else
  if [[ -z "${PUBLIC_IP}" ]]; then
    PUBLIC_IP="$(detect_public_ipv4)"
  fi
  upsert_env "DEPLOY_MODE" "ip"
  upsert_env "PUBLIC_HOST" "${PUBLIC_IP}"
  upsert_env "HTTPS_ENABLED" "false"
  upsert_env "PUBLIC_BASE_URL" "http://${PUBLIC_IP}"
fi

load_env_file

bash "${SCRIPT_DIR}/scripts/render-caddyfile.sh"
mkdir -p "${DEPLOY_DIR}/state"

configure_firewall_if_active

log "starting infrastructure containers"
compose up -d postgres redis nats

log "running database migrations"
compose run --rm migrate

log "starting application services"
compose up -d --build core-platform api-gateway admin-web caddy

bash "${SCRIPT_DIR}/scripts/wait-for-ready.sh" "${PUBLIC_BASE_URL}" "240"

bootstrap_flags=(--print-token)
if [[ "${ROTATE_TOKEN}" == "true" ]]; then
  bootstrap_flags+=(--rotate-token)
fi
owner_info="$(bash "${SCRIPT_DIR}/scripts/bootstrap-owner.sh" "${bootstrap_flags[@]}")"
owner_scope_id="$(echo "${owner_info}" | awk -F= '/^OWNER_SCOPE_ID=/{print $2}' | tail -n1)"
owner_token_file="$(echo "${owner_info}" | awk -F= '/^OWNER_BOOTSTRAP_TOKEN_FILE=/{print $2}' | tail -n1)"

log "running post-bootstrap health checks"
api_ready_status="failed"
if curl -fsS --max-time 5 "${PUBLIC_BASE_URL}/v1/ready" >/dev/null 2>&1; then
  api_ready_status="ok"
fi
panel_status="failed"
if curl -fsS --max-time 5 "${PUBLIC_BASE_URL}/login" >/dev/null 2>&1; then
  panel_status="ok"
fi

compose_ps="$(compose ps --format json 2>/dev/null || true)"

cat <<EOF

Opener NetDoor install completed.

Panel URL: ${PUBLIC_BASE_URL}
HTTPS enabled: ${HTTPS_ENABLED}
Hidden owner scope: ${owner_scope_id}

Owner access:
  - Open ${PUBLIC_BASE_URL}/login in your browser.
  - Use subject: ${OWNER_SUBJECT}
  - Use scope: ${owner_scope_id}
  - Paste bootstrap token shown above.
  - Token file: ${owner_token_file}

Service summary:
  - API readiness: ${api_ready_status}
  - Panel reachability: ${panel_status}

Docker services (json):
${compose_ps}

Config files:
  - deploy/.env
  - deploy/Caddyfile
  - ${owner_token_file}

EOF

print_compose_hint
