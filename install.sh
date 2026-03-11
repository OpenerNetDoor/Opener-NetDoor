#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/opener-netdoor/opener-netdoor.git"
TARGET_DIR="/opt/opener-netdoor"
FORWARDED_ARGS=()

usage() {
  cat <<'EOF'
Usage:
  bash <(curl -fsSL https://raw.githubusercontent.com/opener-netdoor/opener-netdoor/main/install.sh) [options]

Options:
  --domain <fqdn>            Enable HTTPS domain mode.
  --email <address>          Let's Encrypt email in domain mode.
  --ip-mode                  Force plain HTTP IP mode.
  --domain-mode              Force domain mode.
  --ip <address>             Force public IP in ip mode.
  --repo <git-url>           Override repository URL.
  --path <dir>               Override install directory (default /opt/opener-netdoor).
  --help                     Show help.

Any other flags are passed to deploy/install.sh.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO_URL="${2:-}"
      shift 2
      ;;
    --path)
      TARGET_DIR="${2:-}"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      FORWARDED_ARGS+=("$1")
      shift
      ;;
  esac
done

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "[opener-netdoor] this installer targets Linux hosts (Ubuntu/Debian)." >&2
  exit 1
fi

if [[ ! -f /etc/os-release ]]; then
  echo "[opener-netdoor] missing /etc/os-release." >&2
  exit 1
fi
# shellcheck disable=SC1091
source /etc/os-release

if [[ "${ID:-}" != "ubuntu" && "${ID:-}" != "debian" ]]; then
  echo "[opener-netdoor] warning: non-target distro '${ID:-unknown}', continuing best effort."
fi

SUDO=""
if [[ "${EUID}" -ne 0 ]]; then
  if command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
  else
    echo "[opener-netdoor] sudo is required when not running as root." >&2
    exit 1
  fi
fi

need_pkg_update="false"
install_pkg() {
  local pkg="$1"
  if ! dpkg -s "$pkg" >/dev/null 2>&1; then
    if [[ "$need_pkg_update" == "false" ]]; then
      $SUDO apt-get update -y
      need_pkg_update="true"
    fi
    $SUDO apt-get install -y "$pkg"
  fi
}

install_pkg curl
install_pkg ca-certificates
install_pkg git
install_pkg gnupg
install_pkg jq

if ! command -v docker >/dev/null 2>&1; then
  echo "[opener-netdoor] installing docker engine + compose plugin"
  $SUDO install -m 0755 -d /etc/apt/keyrings
  if [[ ! -f /etc/apt/keyrings/docker.gpg ]]; then
    curl -fsSL "https://download.docker.com/linux/${ID}/gpg" | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    $SUDO chmod a+r /etc/apt/keyrings/docker.gpg
  fi
  if [[ ! -f /etc/apt/sources.list.d/docker.list ]]; then
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${ID} ${VERSION_CODENAME} stable" | $SUDO tee /etc/apt/sources.list.d/docker.list >/dev/null
  fi
  $SUDO apt-get update -y
  $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "[opener-netdoor] docker compose plugin is required." >&2
  exit 1
fi

$SUDO mkdir -p "$TARGET_DIR"
if [[ ! -d "$TARGET_DIR/.git" ]]; then
  echo "[opener-netdoor] cloning ${REPO_URL} -> ${TARGET_DIR}"
  $SUDO git clone "$REPO_URL" "$TARGET_DIR"
else
  echo "[opener-netdoor] updating existing repository in ${TARGET_DIR}"
  $SUDO git -C "$TARGET_DIR" fetch --all --tags --prune
  $SUDO git -C "$TARGET_DIR" pull --ff-only || true
fi

if [[ ! -f "$TARGET_DIR/deploy/install.sh" ]]; then
  echo "[opener-netdoor] deploy/install.sh not found in ${TARGET_DIR}" >&2
  exit 1
fi

$SUDO chmod +x "$TARGET_DIR/deploy/install.sh"

echo "[opener-netdoor] running deploy/install.sh ${FORWARDED_ARGS[*]}"
$SUDO bash "$TARGET_DIR/deploy/install.sh" "${FORWARDED_ARGS[@]}"

env_file="$TARGET_DIR/deploy/.env"
panel_url=""
scope_id=""
admin_url_file=""
admin_url=""

if [[ -f "$env_file" ]]; then
  panel_url="$(grep -E '^PUBLIC_BASE_URL=' "$env_file" | tail -n1 | cut -d= -f2-)"
  scope_id="$(grep -E '^OWNER_SCOPE_ID=' "$env_file" | tail -n1 | cut -d= -f2-)"
  admin_url_file="$(grep -E '^ADMIN_ACCESS_URL_FILE=' "$env_file" | tail -n1 | cut -d= -f2-)"
  secret="$(grep -E '^ADMIN_ACCESS_SECRET=' "$env_file" | tail -n1 | cut -d= -f2-)"
  if [[ -n "$panel_url" && -n "$secret" && -n "$scope_id" ]]; then
    admin_url="${panel_url}/${secret}/${scope_id}/"
  fi
fi

if [[ -n "$admin_url_file" && "$admin_url_file" != /* ]]; then
  admin_url_file="$TARGET_DIR/deploy/${admin_url_file#./}"
fi

cat <<EOF

[opener-netdoor] one-click install finished.
Panel URL: ${panel_url:-unknown}
Admin access URL: ${admin_url:-unknown}
Hidden scope: ${scope_id:-unknown}
Admin URL file: ${admin_url_file:-unknown}

Open the admin access URL in browser to create secure cookie session.
EOF
