#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SMOKE_JWT="${SMOKE_JWT:-}"

if [[ -z "${SMOKE_JWT}" ]]; then
  echo "SMOKE_JWT is required"
  exit 1
fi

echo "[stage1] GET ${BASE_URL}/v1/health"
curl -fsS "${BASE_URL}/v1/health" >/dev/null

echo "[stage1] GET ${BASE_URL}/v1/ready"
curl -fsS "${BASE_URL}/v1/ready" >/dev/null

echo "[stage1] tenant isolation deny check"
STATUS=$(curl -sS -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer ${SMOKE_JWT}" \
  "${BASE_URL}/v1/admin/users?tenant_id=tenant-deny")

if [[ "${STATUS}" != "403" ]]; then
  echo "expected 403 for tenant isolation deny path, got ${STATUS}"
  exit 1
fi

echo "[stage1] smoke passed"
