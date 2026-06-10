# Voltana — Production Deployment Guide

Target: Ubuntu 24.04 LTS VPS, 2 vCPU / 4 GB RAM minimum.

See `docs/DEPLOY.md` for the shorter "quick start" reference.
This document covers the complete, hardened setup including backups, monitoring, rollback, and scaling.

---

## Prerequisites

Before starting, ensure you have:

- A VPS with public IPv4, Ubuntu 24.04 LTS, ≥ 2 vCPU / 4 GB RAM
- A domain name with an A record pointing at the VPS IP
  - Verify propagation: `dig +short yourdomain.com`
- SSH access as root (or a user with `sudo`)
- An SMTP relay account (Resend or Mailgun recommended)
- Bale bot token (get from BotFather on Bale) — required for OTP login

---

## Full Deployment Checklist

### Phase 1: Server Bootstrap

```bash
# 1. Clone the repo to /opt/voltana
git clone https://github.com/Ali-roohy/voltana-v2.git /opt/voltana
cd /opt/voltana

# 2. Run the idempotent bootstrap (Docker, Node 20, certbot, UFW, data dirs)
sudo bash scripts/bootstrap-vps-prod.sh
```

Bootstrap installs:
- Docker Engine + Compose v2 plugin
- `certbot` + `python3-certbot-nginx`
- Node.js 20 LTS
- `awscli` (for optional S3 backups)
- UFW (22/80/443 open, all else denied)
- `voltana` system user (no login shell, in docker group)
- `/var/lib/voltana/postgres` — postgres data bind-mount
- `/var/lib/voltana/backups` — backup storage

### Phase 2: Configure Environment

```bash
cp /opt/voltana/.env.production.example /opt/voltana/.env
nano /opt/voltana/.env
```

**All values with `REPLACE_WITH_` must be set before deploying.**

Generate secrets:
```bash
openssl rand -hex 32   # for POSTGRES_PASSWORD
openssl rand -hex 32   # for JWT_SECRET
```

Required production variables:

| Variable | Notes |
|---|---|
| `DOMAIN` | Your domain (no https://) e.g. `voltana.example.com` |
| `APP_URL` | `https://voltana.example.com` |
| `APP_ENV` | Must be `production` — enables Secure cookie flag |
| `POSTGRES_PASSWORD` | Strong random; `openssl rand -hex 32` |
| `JWT_SECRET` | Strong random; `openssl rand -hex 32` |
| `SMTP_HOST` | `smtp.resend.com` (or Mailgun) |
| `SMTP_PASSWORD` | SMTP API key |
| `SMTP_FROM` | `noreply@yourdomain.com` |
| `BALE_BOT_TOKEN` | From Bale BotFather |
| `BALE_BOT_USERNAME` | Your bot's username (without @) |

### Phase 3: TLS Certificate

Certbot needs ports 80/443 free. Do this **before** starting the full stack.

```bash
DOMAIN=voltana.example.com
certbot certonly --standalone -d "$DOMAIN"
```

Certificates land at `/etc/letsencrypt/live/$DOMAIN/`.

Set up auto-renewal:

```bash
crontab -e
# Add this line:
0 3 * * * certbot renew --quiet --deploy-hook "docker compose -f /opt/voltana/docker-compose.yml -f /opt/voltana/docker-compose.prod.yml exec nginx nginx -s reload"
```

### Phase 4: Systemd Service

```bash
cp /opt/voltana/infra/systemd/voltana.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable voltana
```

### Phase 5: First Deploy

```bash
cd /opt/voltana
bash scripts/deploy.sh
```

`deploy.sh` performs:
1. `git pull origin main`
2. Frontend build (`npm ci && npm run build`)
3. Nginx config generation (`envsubst '${DOMAIN}' < nginx.prod.conf`)
4. DB migrations (`docker compose run --rm migrate`)
5. API rebuild (`docker compose up -d --build api nginx`)
6. Nginx reload

### Phase 6: Bootstrap Admin User

After the first deploy, promote your account to admin:

```bash
docker exec voltana-postgres sh -c \
  'psql -U "$POSTGRES_USER" "$POSTGRES_DB" -c "UPDATE users SET is_admin=true WHERE email='\''you@example.com'\'';"'
```

### Phase 7: Start the Systemd Service

```bash
systemctl start voltana
systemctl status voltana
# Expected: active (exited) — oneshot + RemainAfterExit=yes is correct
```

### Phase 8: Verification

```bash
# API health
curl https://voltana.example.com/health
# → {"status":"ok"}

# HTTP → HTTPS redirect
curl -I http://voltana.example.com/
# → HTTP/1.1 301

# Security headers
curl -sI https://voltana.example.com/health \
  | grep -E "Strict-Transport|X-Frame|X-Content"

# API auth is working
curl -s https://voltana.example.com/v1/me
# → {"code":"UNAUTHORIZED",...}  (401 is correct — not 502)
```

### Phase 9: Backup Timer

Re-run bootstrap (or manually install) to activate daily backups:

```bash
# If bootstrap has already run with the repo cloned:
cp /opt/voltana/infra/systemd/voltana-backup.service /etc/systemd/system/
cp /opt/voltana/infra/systemd/voltana-backup.timer   /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now voltana-backup.timer

# Verify timer
systemctl list-timers voltana-backup.timer
```

### Phase 10: Health Check Cron (Optional)

```bash
crontab -e -u voltana
# Add — checks every 5 minutes:
*/5 * * * * bash /opt/voltana/scripts/healthcheck.sh >> /var/log/voltana-health.log 2>&1
```

Set `ALERT_BOT_TOKEN`, `ALERT_CHAT_ID`, and `ALERT_PLATFORM` in `/opt/voltana/.env` to receive
Bale/Telegram alerts when the health check fails.

---

## Database Backups

### How It Works

`scripts/backup-db.sh` runs daily at 03:00 UTC via `voltana-backup.timer`:

1. `docker exec voltana-postgres pg_dump ...` → gzip → `/var/lib/voltana/backups/voltana_YYYYMMDD_HHMMSS.sql.gz`
2. Files older than `BACKUP_RETAIN_DAYS` (default 7) are pruned
3. If `AWS_S3_BACKUP_BUCKET` is set, the backup is also uploaded to S3-compatible storage

### Manual Backup

```bash
sudo -u voltana bash /opt/voltana/scripts/backup-db.sh
```

### Verify Backup

```bash
# List backups
ls -lh /var/lib/voltana/backups/

# Check backup log
tail -20 /var/lib/voltana/backups/backup.log

# Verify systemd timer ran
journalctl -u voltana-backup.service --since yesterday
```

### Restore from Backup

**Warning: this overwrites the running database.**

```bash
# 1. Stop the application
systemctl stop voltana

# 2. Drop and recreate the database
docker compose -f /opt/voltana/docker-compose.yml -f /opt/voltana/docker-compose.prod.yml up -d postgres
docker exec voltana-postgres sh -c '
  psql -U "$POSTGRES_USER" -c "DROP DATABASE IF EXISTS \"$POSTGRES_DB\";"
  psql -U "$POSTGRES_USER" -c "CREATE DATABASE \"$POSTGRES_DB\";"
'

# 3. Restore
BACKUP=/var/lib/voltana/backups/voltana_YYYYMMDD_HHMMSS.sql.gz
gunzip -c "$BACKUP" | docker exec -i voltana-postgres \
  psql -U "$POSTGRES_USER" "$POSTGRES_DB"

# 4. Restart
systemctl start voltana
```

### S3 / Backblaze B2 Setup

Add to `/opt/voltana/.env`:
```
AWS_S3_BACKUP_BUCKET=your-bucket-name
AWS_ACCESS_KEY_ID=your-key-id
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_DEFAULT_REGION=us-east-1
# For Backblaze B2:
# AWS_ENDPOINT_URL=https://s3.us-west-000.backblazeb2.com
```

List remote backups:
```bash
aws s3 ls s3://your-bucket-name/voltana/
```

---

## Deploying Updates

```bash
cd /opt/voltana
bash scripts/deploy.sh
```

The script is safe to run while the app is live — it rebuilds and replaces only `api` and `nginx`,
using `--build` so the latest code is compiled into the container. Nginx reloads gracefully
(no dropped connections).

---

## Rollback

If a deploy breaks the API:

```bash
cd /opt/voltana

# 1. Roll back git
git log --oneline -10      # find the last good commit
git checkout <commit-hash>

# 2. Re-deploy
bash scripts/deploy.sh

# 3. If the migration must also be reverted:
docker compose -f docker-compose.yml -f docker-compose.prod.yml \
  run --rm -e "COMMAND=down 1" migrate
# (adjust the migrate service to accept a COMMAND override if needed)
```

---

## Log Access

```bash
# All services
docker compose -f /opt/voltana/docker-compose.yml \
  -f /opt/voltana/docker-compose.prod.yml logs --follow

# API only
docker compose logs -f api

# nginx access/error log
docker compose logs -f nginx

# Backup log
tail -f /var/lib/voltana/backups/backup.log

# Health check log
tail -f /var/log/voltana-health.log

# Systemd service
journalctl -u voltana.service -f
journalctl -u voltana-backup.service --since "24 hours ago"
```

---

## Troubleshooting

| Symptom | Action |
|---|---|
| 502 Bad Gateway | `docker compose logs api` — api may still be starting (allow 30s) |
| TLS cert error / HTTPS broken | `certbot certificates` — check expiry + domain name match |
| Deploy fails at `npm ci` | Check disk space: `df -h`; Node modules need ~500 MB |
| Deploy fails at migrate | `docker compose run --rm migrate` manually to see full SQL error |
| API crashes in a loop | `journalctl -u voltana.service` + `docker compose logs api` — check for missing env vars |
| Bale bot not delivering OTPs | VPS needs direct internet to `tapi.bale.ai`; check `docker compose logs api \| grep -i bot` |
| Telegram blocked on VPS | Expected in Iran — Bale is the production primary |
| Backup not running | `systemctl status voltana-backup.timer`; check `POSTGRES_CONTAINER` matches running container name |
| Backup fails: container not found | `docker ps --format '{{.Names}}'` — find actual postgres container name, update `.env` |
| DB disk full | Postgres data at `/var/lib/voltana/postgres`; check `du -sh /var/lib/voltana/` |
| Email not delivered | Test SMTP: `swaks --to test@example.com --server $SMTP_HOST:$SMTP_PORT` |
| nginx config invalid after edit | `docker compose exec nginx nginx -t` before reloading |
| Port 80/443 blocked by UFW | `ufw status verbose` — ensure 80/tcp and 443/tcp are allowed |

---

## Monitoring

### Systemd Service Status

```bash
systemctl status voltana
systemctl status voltana-backup.timer
```

### Docker Stack Health

```bash
docker compose -f /opt/voltana/docker-compose.yml \
  -f /opt/voltana/docker-compose.prod.yml ps
```

### Disk Usage

```bash
df -h
du -sh /var/lib/voltana/postgres   # postgres data
du -sh /var/lib/voltana/backups    # local backup files
```

### Health Check Script

```bash
bash /opt/voltana/scripts/healthcheck.sh && echo "healthy" || echo "UNHEALTHY"
```

---

## Scaling Path

The current deployment runs all services on one VPS (postgres, redis, api, nginx as Docker containers).

When you need more capacity:

| Threshold | Action |
|---|---|
| CPU > 70% sustained | Vertical scale (more vCPUs) — `docker compose down && resize VPS && docker compose up -d` |
| RAM > 80% | Add swap: `fallocate -l 2G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile` |
| DB I/O bottleneck | Move postgres to a dedicated VPS with a managed disk; update `POSTGRES_URL` in `.env` |
| API throughput | Run multiple api replicas behind nginx upstream; see nginx `upstream {}` block |
| Zero-downtime deploys | Add a blue/green deploy step to `deploy.sh` (Phase 5 roadmap) |

---

## File Map

| File | Purpose |
|---|---|
| `scripts/bootstrap-vps-prod.sh` | Idempotent Ubuntu 24.04 server setup |
| `scripts/deploy.sh` | One-command deploy / update |
| `scripts/backup-db.sh` | Daily pg_dump backup + S3 upload |
| `scripts/healthcheck.sh` | /health check + optional bot alert |
| `.env.production.example` | Production env template (copy to `.env`) |
| `infra/nginx/nginx.prod.conf` | nginx template (HTTPS + security headers) |
| `infra/systemd/voltana.service` | Systemd unit for auto-start on boot |
| `infra/systemd/voltana-backup.service` | Systemd unit for backup job |
| `infra/systemd/voltana-backup.timer` | Daily backup timer (03:00 UTC) |
| `docker-compose.prod.yml` | Compose overlay: ports 80/443, certs, postgres bind-mount |
| `docs/DEPLOY.md` | Quick-start reference |
| `docs/DEPLOY_PRODUCTION.md` | This file — full production operations guide |
