#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/infra/compose/test/docker-compose.test.yml"
PROJECT_NAME="openernetdoor-stage2"
TEST_POSTGRES_PORT="${TEST_POSTGRES_PORT:-35432}"

cleanup() {
  docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" --profile stage2-test down -v
}
trap cleanup EXIT

export TEST_POSTGRES_PORT

docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" --profile stage2-test up -d

echo "Waiting for postgres-test to become healthy on port ${TEST_POSTGRES_PORT}..."
for i in {1..40}; do
  STATUS=$(docker inspect --format='{{json .State.Health.Status}}' "${PROJECT_NAME}-postgres-test-1" 2>/dev/null || true)
  if [[ "${STATUS}" == '"healthy"' ]]; then
    break
  fi
  sleep 2
  if [[ $i -eq 40 ]]; then
    echo "postgres-test did not become healthy"
    exit 1
  fi
done

export TEST_DATABASE_URL="postgresql://openernetdoor:openernetdoor@127.0.0.1:${TEST_POSTGRES_PORT}/openernetdoor_test?sslmode=disable"
export TEST_MIGRATIONS_DIR="${ROOT_DIR}/ops/migrations"

GOCACHE="${ROOT_DIR}/tmpcache/gobuild"
GOMODCACHE="${ROOT_DIR}/tmpcache/gomod"
mkdir -p "${GOCACHE}" "${GOMODCACHE}"

(
  cd "${ROOT_DIR}"
  GOCACHE="${GOCACHE}" GOMODCACHE="${GOMODCACHE}" go -C services/core-platform test -p 1 -tags=integration ./internal/store ./internal/service ./internal/http -count=1 -v
  GOCACHE="${GOCACHE}" GOMODCACHE="${GOMODCACHE}" go -C apps/api-gateway test -p 1 -tags=integration ./internal/http -count=1 -v
)
