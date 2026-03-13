#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

ROTATE_SECRET="false"
PRINT_URL="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rotate-secret)
      ROTATE_SECRET="true"
      ;;
    --print-url)
      PRINT_URL="true"
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

if [[ -z "${OWNER_SCOPE_ID:-}" ]]; then
  OWNER_SCOPE_ID="$(generate_uuid)"
  upsert_env "OWNER_SCOPE_ID" "${OWNER_SCOPE_ID}"
fi
if [[ -z "${OWNER_SCOPE_NAME:-}" ]]; then
  OWNER_SCOPE_NAME="Default_Owner_Scope"
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
if [[ -z "${ADMIN_ACCESS_SECRET:-}" || "${ROTATE_SECRET}" == "true" || "${ADMIN_ACCESS_SECRET}" == change_me* ]]; then
  ADMIN_ACCESS_SECRET="$(random_hex 24)"
  upsert_env "ADMIN_ACCESS_SECRET" "${ADMIN_ACCESS_SECRET}"
fi
if [[ -z "${ADMIN_ACCESS_URL_FILE:-}" ]]; then
  ADMIN_ACCESS_URL_FILE="./state/admin-access-url.txt"
  upsert_env "ADMIN_ACCESS_URL_FILE" "${ADMIN_ACCESS_URL_FILE}"
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
VALUES ('${OWNER_SCOPE_ID}', '${owner_email_sql}', 'magic-path-session', 'owner', FALSE)
ON CONFLICT (email) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  role = EXCLUDED.role;
SQL

state_dir="${DEPLOY_DIR}/state"
mkdir -p "${state_dir}"

if [[ "${ADMIN_ACCESS_URL_FILE}" = /* ]]; then
  url_file="${ADMIN_ACCESS_URL_FILE}"
else
  url_file="${DEPLOY_DIR}/${ADMIN_ACCESS_URL_FILE#./}"
fi

admin_url="${PUBLIC_BASE_URL}/${ADMIN_ACCESS_SECRET}/${OWNER_SCOPE_ID}/"
umask 077
printf '%s\n' "${admin_url}" >"${url_file}"

if [[ "${PRINT_URL}" == "true" ]]; then
  echo >&2
  warn "sensitive: admin access url is shown once" >&2
  echo "${admin_url}" >&2
  echo >&2
fi

printf 'OWNER_SCOPE_ID=%s\n' "${OWNER_SCOPE_ID}"
printf 'ADMIN_ACCESS_URL=%s\n' "${admin_url}"
printf 'ADMIN_ACCESS_URL_FILE=%s\n' "${url_file}"
