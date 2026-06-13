# Voltana — Production Deployment Guide (OUTDATED)

> ⚠️ **Superseded — do not follow this file for a voltanaev.ir deploy.**
> This is the original Phase-3 quick start (third-party SMTP relay, `bootstrap-vps.sh`,
> single-domain cert) and predates the Poste.io mail server, web push / VAPID, the apex+www
> redirect, and the mail ports added in TASK-0040/0041.
> **Use [`docs/DEPLOY_PRODUCTION.md`](DEPLOY_PRODUCTION.md)** — its "Quick Runbook
> (copy-paste)" section is the canonical, current sequence. This file is kept only for
> historical reference.

Target: Ubuntu 22.04 LTS VPS, 2 vCPU / 4 GB RAM.

---

## Prerequisites

- A VPS with a public IPv4 address
- A domain name with an **A record pointing at the VPS IP** (propagation can take up to 1 hour; confirm with `dig +short yourdomain.com`)
- SSH access as root (or a user with sudo)

---

## 1. Bootstrap the server

Run once on a fresh VPS. Safe to re-run (idempotent).

```bash
git clone https://github.com/Ali-roohy/voltana-v2.git /opt/voltana
cd /opt/voltana
sudo bash scripts/bootstrap-vps.sh
```

This installs:
- Docker Engine + Compose v2 plugin
- `certbot` + `python3-certbot-nginx`
- UFW firewall (ports 22/80/443 open; all else denied)
- Creates the `voltana` system user (no login shell, in docker group)
- Creates `/opt/voltana/` deploy directory

---

## 2. Configure environment

```bash
cp /opt/voltana/.env.example /opt/voltana/.env
nano /opt/voltana/.env
```

Required fields for production:

| Variable | Example | Notes |
|---|---|---|
| `POSTGRES_PASSWORD` | `hunter2` | Strong random password |
| `JWT_SECRET` | 32+ chars | `openssl rand -hex 32` |
| `APP_ENV` | `production` | Enables Secure cookie flag |
| `APP_URL` | `https://voltana.example.com` | Used in email links |
| `DOMAIN` | `voltana.example.com` | Used by `deploy.sh` for nginx |
| `SMTP_HOST` | `smtp.resend.com` | Real relay (not MailHog) |
| `SMTP_PORT` | `587` | |
| `SMTP_USER` | `resend` | |
| `SMTP_PASSWORD` | `re_…` | API key |
| `SMTP_FROM` | `noreply@example.com` | |
| `BALE_BOT_TOKEN` | `1234:…` | Optional — OTP via Bale |

---

## 3. Obtain TLS certificate

Certbot needs port 80 free. Do this **before** starting the full stack.

```bash
# Replace with your actual domain
DOMAIN=voltana.example.com

certbot certonly --standalone -d "$DOMAIN"
```

Certificates land at `/etc/letsencrypt/live/$DOMAIN/`.

**Auto-renewal** — add a cron job:

```bash
crontab -e
# Add:
0 3 * * * certbot renew --quiet --deploy-hook "docker compose -f /opt/voltana/docker-compose.yml -f /opt/voltana/docker-compose.prod.yml exec nginx nginx -s reload"
```

---

## 4. Install the systemd service

```bash
cp /opt/voltana/infra/systemd/voltana.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable voltana
```

---

## 5. First deploy

```bash
cd /opt/voltana
bash scripts/deploy.sh
```

`deploy.sh` will:
1. `git pull origin main`
2. Build the React frontend (`voltana-web/dist`)
3. Generate `nginx/nginx.conf` from `infra/nginx/nginx.prod.conf` (substitutes `$DOMAIN`)
4. Run DB migrations via `docker compose run --rm migrate`
5. Rebuild the Go API container
6. Start `api` + `nginx` with the production overlay
7. Reload nginx

Verify:

```bash
# Health check
curl https://voltana.example.com/health
# → {"status":"ok"}

# HTTP redirect
curl -I http://voltana.example.com/
# → HTTP/1.1 301

# Security headers
curl -sI https://voltana.example.com/health | grep -E "Strict|X-Frame|X-Content"
```

---

## 6. Start the systemd service

```bash
systemctl start voltana
systemctl status voltana
# → active (exited)  ← oneshot + RemainAfterExit=yes is expected
```

After a server reboot the service will bring the stack back up automatically.

---

## Deploying updates

On any code change, just run:

```bash
cd /opt/voltana
bash scripts/deploy.sh
```

No manual steps — the script handles git pull, frontend build, nginx config regen, api rebuild, migrations, and nginx reload.

---

## Troubleshooting

| Symptom | Check |
|---|---|
| 502 Bad Gateway | `docker compose logs api` — api might still be starting |
| TLS cert error | `certbot certificates` — check expiry + domain match |
| Deploy fails | `deploy.sh` uses `set -euo pipefail`; check the line it exits on |
| DB migration error | `docker compose run --rm migrate` — run manually to see full error |
| Bale bot not linking | VPS must have direct internet access to `api.bale.ai`; check `docker compose logs api \| grep bot:` |
| nginx config invalid | `docker compose exec nginx nginx -t` before reloading |

---

## File map

| File | Purpose |
|---|---|
| `scripts/bootstrap-vps.sh` | Idempotent first-run server setup |
| `scripts/deploy.sh` | One-command deploy / update |
| `infra/nginx/nginx.prod.conf` | Production nginx template (`${DOMAIN}` substituted at deploy) |
| `infra/systemd/voltana.service` | systemd unit for auto-restart |
| `docker-compose.prod.yml` | Compose overlay: port 443, cert mounts, disables MailHog |
| `docs/DEPLOY.md` | This file |
