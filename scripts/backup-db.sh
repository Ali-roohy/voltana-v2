#!/usr/bin/env bash
# backup-db.sh — Daily PostgreSQL backup with local retention + optional S3 upload.
# Run as voltana user (or root). Called by voltana-backup.service.
# Env vars (sourced from /opt/voltana/.env if present):
#   POSTGRES_USER, POSTGRES_DB — container credentials
#   BACKUP_DIR              — override backup location (default: /var/lib/voltana/backups)
#   BACKUP_RETAIN_DAYS      — days to keep local backups (default: 7)
#   AWS_S3_BACKUP_BUCKET    — optional; if set, uploads to s3://<bucket>/voltana/<file>
set -euo pipefail

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Load production env for POSTGRES_USER / POSTGRES_DB / AWS_* vars
if [[ -f "$DEPLOY_DIR/.env" ]]; then
    set -o allexport
    # shellcheck disable=SC1091
    source "$DEPLOY_DIR/.env"
    set +o allexport
fi

BACKUP_DIR="${BACKUP_DIR:-/var/lib/voltana/backups}"
RETAIN_DAYS="${BACKUP_RETAIN_DAYS:-7}"
POSTGRES_USER="${POSTGRES_USER:-voltana_user}"
POSTGRES_DB="${POSTGRES_DB:-voltana}"
CONTAINER="${POSTGRES_CONTAINER:-voltana-postgres}"

TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)
FILENAME="voltana_${TIMESTAMP}.sql.gz"
DEST="$BACKUP_DIR/$FILENAME"
LOG="$BACKUP_DIR/backup.log"

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" | tee -a "$LOG"; }

log "Starting backup → $DEST"

# ── 1. Ensure backup directory exists ────────────────────────────────────────
mkdir -p "$BACKUP_DIR"

# ── 2. pg_dump via docker exec ───────────────────────────────────────────────
docker exec "$CONTAINER" \
    pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" \
    | gzip -9 > "$DEST"

BYTES=$(stat -c%s "$DEST")
log "Backup written: $FILENAME (${BYTES} bytes)"

# ── 3. Local 7-day retention ─────────────────────────────────────────────────
PRUNED=$(find "$BACKUP_DIR" -maxdepth 1 -name "voltana_*.sql.gz" -mtime +"$RETAIN_DAYS" -print)
if [[ -n "$PRUNED" ]]; then
    log "Pruning backups older than ${RETAIN_DAYS} days:"
    echo "$PRUNED" | while read -r f; do
        rm -f "$f"
        log "  deleted: $(basename "$f")"
    done
fi

# ── 4. Optional S3 upload ─────────────────────────────────────────────────────
if [[ -n "${AWS_S3_BACKUP_BUCKET:-}" ]]; then
    S3_KEY="voltana/$FILENAME"
    log "Uploading to s3://${AWS_S3_BACKUP_BUCKET}/${S3_KEY}…"
    aws s3 cp "$DEST" "s3://${AWS_S3_BACKUP_BUCKET}/${S3_KEY}" \
        --storage-class STANDARD_IA \
        --no-progress
    log "S3 upload complete."
fi

log "Backup done."
