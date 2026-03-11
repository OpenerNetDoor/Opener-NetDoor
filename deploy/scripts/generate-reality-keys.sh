#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib.sh
source "${SCRIPT_DIR}/lib.sh"

ensure_env_file
load_env_file

if [[ -n "${RUNTIME_REALITY_PRIVATE_KEY:-}" && -n "${RUNTIME_REALITY_PUBLIC_KEY:-}" ]]; then
  log "reality keypair already configured"
  exit 0
fi

if ! command -v docker >/dev/null 2>&1; then
  die "docker is required to generate reality keypair"
fi

log "generating Xray REALITY x25519 keypair"
key_output="$(docker run --rm "${XRAY_IMAGE:-ghcr.io/xtls/xray-core:latest}" x25519 2>/dev/null || true)"
if [[ -z "${key_output}" ]]; then
  die "failed to generate xray reality keys using image ${XRAY_IMAGE:-ghcr.io/xtls/xray-core:latest}"
fi

private_key="$(printf '%s\n' "${key_output}" | awk -F': ' '/Private key:/{print $2}' | tr -d '\r' | tail -n1)"
public_key="$(printf '%s\n' "${key_output}" | awk -F': ' '/Public key:/{print $2}' | tr -d '\r' | tail -n1)"

if [[ -z "${private_key}" || -z "${public_key}" ]]; then
  die "unable to parse xray x25519 output"
fi

upsert_env "RUNTIME_REALITY_PRIVATE_KEY" "${private_key}"
upsert_env "RUNTIME_REALITY_PUBLIC_KEY" "${public_key}"
log "saved REALITY keypair to deploy/.env"
