#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

ROTATE_TOKEN="false"
PRINT_TOKEN="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rotate-token)
      ROTATE_TOKEN="true"
      ;;
    --print-token)
      PRINT_TOKEN="true"
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
  shift
done

require_cmd openssl
require_cmd docker

ensure_env_file
load_env_file

if [[ -z "${OWNER_SCOPE_ID:-}" ]]; then
  OWNER_SCOPE_ID="$(generate_uuid)"
  upsert_env "OWNER_SCOPE_ID" "${OWNER_SCOPE_ID}"
fi
if [[ -z "${OWNER_SCOPE_NAME:-}" ]]; then
  OWNER_SCOPE_NAME="Default Owner Scope"
  upsert_env "OWNER_SCOPE_NAME" "${OWNER_SCOPE_NAME}"
fi
if [[ -z "${OWNER_SUBJECT:-}" ]]; then
  OWNER_SUBJECT="owner"
  upsert_env "OWNER_SUBJECT" "${OWNER_SUBJECT}"
fi
if [[ -z "${OWNER_EMAIL:-}" ]]; then
  OWNER_EMAIL="owner@example.com"
  upsert_env "OWNER_EMAIL" "${OWNER_EMAIL}"
fi
if [[ -z "${OWNER_TOKEN_TTL_HOURS:-}" || ! "${OWNER_TOKEN_TTL_HOURS}" =~ ^[0-9]+$ ]]; then
  OWNER_TOKEN_TTL_HOURS="720"
  upsert_env "OWNER_TOKEN_TTL_HOURS" "${OWNER_TOKEN_TTL_HOURS}"
fi

load_env_file

owner_scope_sql_name="${OWNER_SCOPE_NAME//\'/''}"
owner_email_sql="${OWNER_EMAIL//\'/''}"

compose exec -T postgres psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" <<SQL
INSERT INTO tenants(id, name, status)
VALUES ('${OWNER_SCOPE_ID}', '${owner_scope_sql_name}', 'active')
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  status = 'active';

INSERT INTO admins(tenant_id, email, password_hash, role, is_mfa_enabled)
VALUES ('${OWNER_SCOPE_ID}', '${owner_email_sql}', 'bootstrap-token-only', 'owner', FALSE)
ON CONFLICT (email) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  role = EXCLUDED.role;
SQL

state_dir="${DEPLOY_DIR}/state"
mkdir -p "${state_dir}"

if [[ -z "${OWNER_BOOTSTRAP_TOKEN_FILE:-}" ]]; then
  OWNER_BOOTSTRAP_TOKEN_FILE="./state/owner-bootstrap-token.txt"
  upsert_env "OWNER_BOOTSTRAP_TOKEN_FILE" "${OWNER_BOOTSTRAP_TOKEN_FILE}"
  load_env_file
fi

if [[ "${OWNER_BOOTSTRAP_TOKEN_FILE}" = /* ]]; then
  token_file="${OWNER_BOOTSTRAP_TOKEN_FILE}"
else
  token_file="${DEPLOY_DIR}/${OWNER_BOOTSTRAP_TOKEN_FILE#./}"
fi

b64url() {
  openssl base64 -A | tr '+/' '-_' | tr -d '='
}

escape_json() {
  local raw="$1"
  raw="${raw//\\/\\\\}"
  raw="${raw//\"/\\\"}"
  printf '%s' "${raw}"
}

generate_token() {
  local now exp subject payload header signing_input signature
  now="$(date +%s)"
  exp="$((now + OWNER_TOKEN_TTL_HOURS * 3600))"
  subject="$(escape_json "${OWNER_SUBJECT}")"
  header='{"alg":"HS256","typ":"JWT"}'
  payload=$(printf '{"sub":"%s","iss":"%s","aud":"%s","exp":%s,"tenant_id":"%s","scopes":["admin:read","admin:write","platform:admin"]}' \
    "${subject}" "$(escape_json "${JWT_ISSUER}")" "$(escape_json "${JWT_AUDIENCE}")" "${exp}" "$(escape_json "${OWNER_SCOPE_ID}")")

  signing_input="$(printf '%s' "${header}" | b64url).$(printf '%s' "${payload}" | b64url)"
  signature="$(printf '%s' "${signing_input}" | openssl dgst -binary -sha256 -hmac "${JWT_SECRET}" | b64url)"
  printf '%s.%s\n' "${signing_input}" "${signature}"
}

if [[ ! -f "${token_file}" || "${ROTATE_TOKEN}" == "true" ]]; then
  umask 077
  generate_token >"${token_file}"
  log "owner bootstrap token generated at ${token_file}"
else
  log "owner bootstrap token already exists at ${token_file}"
fi

if [[ "${PRINT_TOKEN}" == "true" ]]; then
  echo >&2
  warn "sensitive: bootstrap token is shown once" >&2
  cat "${token_file}" >&2
  echo >&2
fi

printf 'OWNER_SCOPE_ID=%s\n' "${OWNER_SCOPE_ID}"
printf 'OWNER_BOOTSTRAP_TOKEN_FILE=%s\n' "${token_file}"
