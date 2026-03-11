#!/bin/sh
set -eu

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"

export PGPASSWORD="${POSTGRES_PASSWORD}"

psql_exec() {
  psql -v ON_ERROR_STOP=1 -h postgres -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" "$@"
}

echo "[opener-netdoor] waiting for postgres"
until psql_exec -c 'select 1' >/dev/null 2>&1; do
  sleep 1
done

echo "[opener-netdoor] postgres is ready"

psql_exec <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  checksum TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

for file in /migrations/*.sql; do
  [ -f "${file}" ] || continue
  version="$(basename "${file}")"
  checksum="$(sha256sum "${file}" | awk '{print $1}')"

  existing_checksum="$(psql_exec -tAc "SELECT checksum FROM schema_migrations WHERE version='${version}'")"
  existing_checksum="$(echo "${existing_checksum}" | tr -d '[:space:]')"

  if [ -n "${existing_checksum}" ]; then
    if [ "${existing_checksum}" != "${checksum}" ]; then
      echo "[opener-netdoor][error] checksum mismatch for already applied migration ${version}" >&2
      exit 1
    fi
    echo "[opener-netdoor] migration ${version} already applied"
    continue
  fi

  echo "[opener-netdoor] applying migration ${version}"
  psql_exec -f "${file}"
  psql_exec -c "INSERT INTO schema_migrations(version, checksum) VALUES ('${version}', '${checksum}')"
done

echo "[opener-netdoor] migrations completed"
