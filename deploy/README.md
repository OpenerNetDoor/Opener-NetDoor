# Opener NetDoor Self-Hosted Deploy

This folder contains the production-oriented Docker Compose deployment flow for a single-owner VPS/VDS install.

## What gets deployed

- `postgres` (persistent volume)
- `redis`
- `nats`
- `core-platform`
- `api-gateway`
- `admin-web`
- `xray` (VLESS + REALITY runtime)
- `caddy` (public reverse proxy for panel/API)

Public ports:

- `80/443` via Caddy
- `RUNTIME_VLESS_PORT` (default `8443`) via Xray for client traffic

Runtime networking requirement:
- `xray` must be attached to both `backend` and `public` compose networks.
- `backend` is used for internal control-plane access, `public` is required for external client ingress on `RUNTIME_VLESS_PORT`.

## One-click bootstrap

Run directly on a clean VPS:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/opener-netdoor/opener-netdoor/main/install.sh) --ip-mode
```

Domain mode:

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/opener-netdoor/opener-netdoor/main/install.sh) --domain panel.example.com --email ops@example.com
```

The bootstrap script installs prerequisites, clones to `/opt/opener-netdoor`, and runs `deploy/install.sh`.

## Quick install from cloned repo

```bash
bash deploy/install.sh --ip-mode
```

Domain + HTTPS mode:

```bash
bash deploy/install.sh --domain panel.example.com --email ops@example.com
```

The installer will:

1. Ensure Docker + Compose plugin are available.
2. Create `deploy/.env` from `deploy/.env.example` (if missing).
3. Generate strong secrets and REALITY keypair if missing.
4. Render `deploy/Caddyfile` for domain or IP mode.
5. Start DB/infra containers.
6. Run SQL migrations idempotently.
7. Start control plane services.
8. Bootstrap hidden owner scope and magic admin access URL.
9. Bootstrap runtime node, generate/apply Xray config, and start Xray.
10. Print panel URL, admin URL, HTTPS/runtime status, and service summary.

## Owner access

After install, open the **admin access URL** printed by installer:

- `https://HOST/ADMIN_ACCESS_SECRET/OWNER_SCOPE_ID/`

This URL sets an HttpOnly session cookie and redirects to dashboard.

Also written to file:

- `deploy/state/admin-access-url.txt`

Treat it as sensitive.

## Upgrade

```bash
bash deploy/upgrade.sh
```

Rotate admin secret on upgrade:

```bash
bash deploy/upgrade.sh --rotate-admin-secret
```

## Uninstall

Stop services but keep data:

```bash
bash deploy/uninstall.sh
```

Full data removal:

```bash
bash deploy/uninstall.sh --purge-data --purge-state
```

## Main files

- `deploy/docker-compose.yml` - production stack
- `deploy/.env.example` - env template
- `deploy/Caddyfile` - generated reverse-proxy config
- `deploy/scripts/run-migrations.sh` - idempotent migration runner
- `deploy/scripts/bootstrap-owner.sh` - owner scope + admin magic URL flow
- `deploy/scripts/bootstrap-runtime.sh` - node runtime config generation/apply
- `deploy/scripts/generate-reality-keys.sh` - REALITY x25519 keypair generation

