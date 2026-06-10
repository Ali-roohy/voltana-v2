#!/usr/bin/env bash
# healthcheck.sh — Check Voltana API health and optionally alert via Bale/Telegram.
# Designed to run from cron (e.g. every 5 minutes) or systemd.
# Exit 0 = healthy; exit 1 = unhealthy.
#
# Optional env vars (or set in /opt/voltana/.env):
#   HEALTH_URL          — full URL to check (default: https://localhost/health)
#   ALERT_BOT_TOKEN     — Bale or Telegram bot token for failure alerts
#   ALERT_CHAT_ID       — chat/user ID to send the alert to
#   ALERT_PLATFORM      — "bale" (default) or "telegram"
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -f "$DEPLOY_DIR/.env" ]]; then
    set -o allexport
    # shellcheck disable=SC1091
    source "$DEPLOY_DIR/.env"
    set +o allexport
fi

HEALTH_URL="${HEALTH_URL:-https://localhost/health}"
LOG=/var/log/voltana-health.log
ALERT_PLATFORM="${ALERT_PLATFORM:-bale}"

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" | tee -a "$LOG"; }

send_alert() {
    local msg="$1"
    [[ -z "${ALERT_BOT_TOKEN:-}" ]] && return 0
    [[ -z "${ALERT_CHAT_ID:-}" ]] && return 0

    if [[ "$ALERT_PLATFORM" == "bale" ]]; then
        local api_url="https://tapi.bale.ai/bot${ALERT_BOT_TOKEN}/sendMessage"
    else
        local api_url="https://api.telegram.org/bot${ALERT_BOT_TOKEN}/sendMessage"
    fi

    curl -s -X POST "$api_url" \
        -H "Content-Type: application/json" \
        -d "{\"chat_id\":\"${ALERT_CHAT_ID}\",\"text\":\"${msg}\"}" \
        > /dev/null || true
}

# ── Health check ──────────────────────────────────────────────────────────────
HTTP_CODE=$(curl -sk -o /dev/null -w "%{http_code}" \
    --connect-timeout 5 --max-time 10 \
    "$HEALTH_URL" || echo "000")

if [[ "$HTTP_CODE" == "200" ]]; then
    log "OK (HTTP $HTTP_CODE) — $HEALTH_URL"
    exit 0
fi

HOSTNAME=$(hostname -f 2>/dev/null || hostname)
MSG="🔴 Voltana health check FAILED on ${HOSTNAME} — HTTP ${HTTP_CODE} (${HEALTH_URL})"
log "$MSG"
send_alert "$MSG"
exit 1
