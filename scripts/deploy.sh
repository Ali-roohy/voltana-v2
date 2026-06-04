#!/usr/bin/env bash
# deploy.sh — Deploy or update Voltana on the VPS.
# Run from /opt/voltana: bash scripts/deploy.sh
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DEPLOY_DIR"

# ── Load .env ─────────────────────────────────────────────────────────────────
[[ -f .env ]] || { echo "ERROR: .env not found in $DEPLOY_DIR" >&2; exit 1; }
# Read the full .env for docker compose (needs DB/JWT/SMTP vars).
# Also export DOMAIN for envsubst to generate the nginx config.
set -o allexport; source .env; set +o allexport

[[ -n "${DOMAIN:-}" ]] || { echo "ERROR: DOMAIN not set in .env" >&2; exit 1; }

echo "[deploy] Starting deploy for $DOMAIN at $(date)"

# ── 1. Pull latest code ───────────────────────────────────────────────────────
echo "[deploy] Pulling latest code…"
git pull origin main

# ── 2. Build frontend ─────────────────────────────────────────────────────────
if [[ -d voltana-web ]]; then
    echo "[deploy] Building frontend…"
    (cd voltana-web && npm ci --silent && npm run build)
fi

# ── 3. Generate production nginx config from template ─────────────────────────
echo "[deploy] Generating nginx config for $DOMAIN…"
# envsubst with explicit var list so nginx's own $var syntax is untouched.
envsubst '${DOMAIN}' < infra/nginx/nginx.prod.conf > nginx/nginx.conf
echo "[deploy] nginx/nginx.conf written."

# ── 4. Run database migrations ────────────────────────────────────────────────
echo "[deploy] Running migrations…"
docker compose -f docker-compose.yml -f docker-compose.prod.yml \
    run --rm migrate

# ── 5. Rebuild and restart api + nginx ────────────────────────────────────────
echo "[deploy] Rebuilding api and restarting services…"
docker compose -f docker-compose.yml -f docker-compose.prod.yml \
    up -d --build api nginx

# ── 6. Reload nginx gracefully (picks up new config without dropping connections)
echo "[deploy] Reloading nginx config…"
sleep 3
docker compose exec nginx nginx -s reload 2>/dev/null || true

echo "[deploy] ✓ Deploy complete at $(date)"
