#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

ensure_env_file
load_env_file

if [[ -z "${PUBLIC_HOST:-}" ]]; then
  die "PUBLIC_HOST is required"
fi

if [[ "${HTTPS_ENABLED:-false}" == "true" ]]; then
  if [[ -z "${LETSENCRYPT_EMAIL:-}" ]]; then
    die "LETSENCRYPT_EMAIL is required when HTTPS_ENABLED=true"
  fi
  cat >"${DEPLOY_DIR}/Caddyfile" <<EOF
{
  email ${LETSENCRYPT_EMAIL}
}

${PUBLIC_HOST} {
  encode zstd gzip

  @api path /v1*
  handle @api {
    reverse_proxy api-gateway:8080
  }

  handle {
    reverse_proxy admin-web:3000
  }
}
EOF
else
  cat >"${DEPLOY_DIR}/Caddyfile" <<EOF
http://${PUBLIC_HOST} {
  encode zstd gzip

  @api path /v1*
  handle @api {
    reverse_proxy api-gateway:8080
  }

  handle {
    reverse_proxy admin-web:3000
  }
}
EOF
fi

log "rendered ${DEPLOY_DIR}/Caddyfile"
