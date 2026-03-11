# Opener NetDoor Self-Hosted Deploy

This folder contains the production-oriented Docker Compose deployment flow for a single-owner VPS/VDS install.

## What gets deployed

- `postgres` (persistent volume)
- `redis`
- `nats`
- `core-platform`
- `api-gateway`
- `admin-web`
- `caddy` (public reverse proxy)

Only Caddy publishes public ports (`80`, `443`).
Internal services stay on an internal Docker network.

## Quick install

From repository root on Ubuntu/Debian VPS:

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
3. Generate strong secrets if placeholders are still present.
4. Render `deploy/Caddyfile` for domain or IP mode.
5. Start DB/infra containers.
6. Run SQL migrations idempotently.
7. Start app + proxy containers.
8. Bootstrap a hidden owner scope and create a bootstrap owner token.
9. Print panel URL, HTTPS status, token instructions, and service summary.

## Owner access

After install, open:

- `PANEL_URL/login`

Use:

- subject: value from `OWNER_SUBJECT` in `deploy/.env` (default `owner`)
- hidden scope: value from `OWNER_SCOPE_ID` in `deploy/.env`
- bootstrap token: printed once by installer and saved at `deploy/state/owner-bootstrap-token.txt`

Treat the bootstrap token as sensitive.

## Upgrade

```bash
bash deploy/upgrade.sh
```

Optional token rotation during upgrade:

```bash
bash deploy/upgrade.sh --rotate-owner-token
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
- `deploy/scripts/bootstrap-owner.sh` - hidden owner scope + bootstrap token flow
