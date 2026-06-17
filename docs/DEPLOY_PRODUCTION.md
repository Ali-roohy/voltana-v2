# Voltana — Production Deployment Guide

Target: Ubuntu 24.04 LTS VPS, 2 vCPU / 4 GB RAM minimum.

This document covers the complete, hardened setup including backups, monitoring, rollback, and scaling.
The copy-paste runbook below is the canonical deploy sequence; the numbered phases further down
explain each step in detail.

> ⚠️ Order matters: **DNS → certbot → deploy**. nginx loads the TLS cert at startup, so it
> will not boot without a cert (`deploy.sh` does not issue one). And certbot's standalone
> challenge needs DNS already resolving to the VPS.

> ℹ️ **DuckDNS domain.** `voltanaev.duckdns.org` is a free DuckDNS subdomain. DuckDNS gives
> you a **single subdomain only** — there is no `www.`, no `mail.`, and you cannot set
> `MX`/`SPF`/`DKIM`/`DMARC`/`PTR` records. The app and TLS work fine; only outbound email
> deliverability is affected (see the Mail and Deliverability sections).

---

## Quick Runbook (copy-paste — voltanaev.duckdns.org)

```bash
# 1 — SSH in
ssh root@YOUR_VPS_IP

# 2 — Clone
git clone https://github.com/Ali-roohy/voltana-v2.git /opt/voltana
cd /opt/voltana

# 3 — Bootstrap (Docker, Node 20, certbot, UFW incl. mail ports, data dirs)
bash scripts/bootstrap-vps-prod.sh

# 4 — DNS (DuckDNS — point the subdomain at the VPS IP, BEFORE certbot)
#   At https://www.duckdns.org set the "current ip" of the `voltanaev` domain to VPS_IP,
#   or update it from the VPS itself:
#     curl "https://www.duckdns.org/update?domains=voltanaev&token=YOUR_DUCKDNS_TOKEN&ip="
#   (empty ip= lets DuckDNS auto-detect the caller's IP)
# Wait for propagation, then verify:
dig +short voltanaev.duckdns.org   # → VPS_IP

# 5 — TLS cert (BEFORE deploy; single domain — Poste reuses it)
certbot certonly --standalone -d voltanaev.duckdns.org

# 6 — Generate a fresh VAPID pair (the VPS has Node, not Go)
npx web-push generate-vapid-keys
#   → copy the Public + Private keys into .env in the next step

# 7 — Configure .env  (the file MUST be named .env — deploy.sh & compose read .env)
cp .env.production.example .env          # NOT .env.production
nano .env
#   Required:
#     DOMAIN=voltanaev.duckdns.org
#     APP_URL=https://voltanaev.duckdns.org
#     APP_ENV=production                 # enables the Secure cookie flag
#     POSTGRES_PASSWORD=<openssl rand -hex 32>
#     JWT_SECRET=<openssl rand -hex 32>
#     VAPID_PUBLIC_KEY=<from step 6>
#     VAPID_PRIVATE_KEY=<from step 6>
#     SMTP_HOST=voltanaev.duckdns.org
#     SMTP_PORT=587
#     SMTP_USER=noreply@voltanaev.duckdns.org
#     SMTP_FROM=noreply@voltanaev.duckdns.org
#     SMTP_PASSWORD=<filled in step 9, after the mailbox exists>
#     BALE_BOT_TOKEN=<from BotFather — ROTATED if ever log-exposed>
#     BALE_BOT_USERNAME=voltana_ev_bot

# 8 — First deploy
bash scripts/deploy.sh

# 9 — Poste.io first-run (admin UI is localhost-only — tunnel in)
ssh -L 8443:127.0.0.1:8443 root@YOUR_VPS_IP
#   Browse to https://localhost:8443 and complete the wizard:
#     1. Set the Poste admin password
#     2. Mail server hostname: voltanaev.duckdns.org
#     3. TLS → custom cert:
#          /etc/letsencrypt/live/voltanaev.duckdns.org/fullchain.pem
#          /etc/letsencrypt/live/voltanaev.duckdns.org/privkey.pem
#     4. Create mailbox noreply@voltanaev.duckdns.org → copy its password
#   (DKIM/SPF/DMARC can't be published on DuckDNS — skip them; expect mail to be
#    spam-foldered by external providers. Fine for in-app / testing use.)

# 10 — Put the mailbox password in .env and reload the API
nano .env                                 # SMTP_PASSWORD=<from step 9.4>
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d api

# 11 — Admin promotion (IF NEEDED)
#   The first registered user automatically becomes admin — normally you can skip this.
#   Use it only to promote a different/later account, or to restore admin if something
#   went wrong. Replace the email with the account to promote.
docker exec voltana-postgres sh -c \
  'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
   -c "UPDATE users SET is_admin=true WHERE email='\''ali.roohi.eng@gmail.com'\'';"'

# 12 — Install systemd (reboot survival)
cp infra/systemd/voltana.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now voltana

# 13 — Smoke test (see "Production Smoke Test" section for the full checklist)
curl https://voltanaev.duckdns.org/health  # → {"status":"ok"}
```

Each step is expanded below; the mail-specific steps (9–10) are detailed under
"Mail Server (Poste.io)", and the deliverability + smoke checks under their own sections.

---

## Prerequisites

Before starting, ensure you have:

- A VPS with public IPv4, Ubuntu 24.04 LTS, ≥ 2 vCPU / 4 GB RAM
  - **Recommended location: Iran** — Bale (and Telegram) are directly reachable, which is
    required for OTP delivery and unblocks the contact-share OTP flow.
- The DuckDNS subdomain **voltanaev.duckdns.org** pointing at the VPS IP (see DNS section)
  - Verify propagation: `dig +short voltanaev.duckdns.org`
- A **DuckDNS account + token** (from <https://www.duckdns.org>) to manage the subdomain's IP
- SSH access as root (or a user with `sudo`)
- Bale bot token (get from BotFather on Bale) — required for OTP login

> Mail is self-hosted via **Poste.io** (a container in `docker-compose.prod.yml`) — no
> third-party SMTP relay account is needed. See "Mail Server (Poste.io)" below.

---

## DNS (voltanaev.duckdns.org)

DuckDNS only exposes the subdomain's **A record** (its "current ip"). There is no
registrar zone editor — you point the subdomain at the VPS from the DuckDNS dashboard or
its update URL. **No www, no mail.* subdomain, and no MX/SPF/DKIM/DMARC/PTR are possible.**

Set the IP (replace `VPS_IP` and `YOUR_DUCKDNS_TOKEN`):

```bash
# From the DuckDNS dashboard: set the "current ip" of `voltanaev` to VPS_IP.
# Or from the VPS / any host:
curl "https://www.duckdns.org/update?domains=voltanaev&token=YOUR_DUCKDNS_TOKEN&ip=VPS_IP"
#   → returns "OK". Omit the ip value (ip=) to let DuckDNS auto-detect the caller's IP.
```

> 💡 To keep the record fresh on a dynamic IP, add a cron entry on the VPS:
> ```
> */5 * * * * curl -s "https://www.duckdns.org/update?domains=voltanaev&token=YOUR_DUCKDNS_TOKEN&ip=" >/dev/null
> ```

Verify once propagated:
```bash
dig +short voltanaev.duckdns.org   # → VPS_IP
```

> ⚠️ **Email deliverability is limited on DuckDNS.** Because you cannot publish
> MX/SPF/DKIM/DMARC records or set a PTR for the IP, mail sent from
> `noreply@voltanaev.duckdns.org` will be spam-foldered or rejected by most external
> providers (Gmail etc.). The self-hosted Poste.io mailbox still works for in-app /
> testing use; use a real owned domain when deliverable verification email matters.

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
- UFW (22/80/443 + mail 25/465/587/993 open, all else denied)
- `voltana` system user (no login shell, in docker group)
- `/var/lib/voltana/postgres` — postgres data bind-mount
- `/var/lib/voltana/backups` — backup storage
- `/var/lib/voltana/mail` — Poste.io mail data bind-mount

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
| `DOMAIN` | The DuckDNS subdomain, no `https://`: `voltanaev.duckdns.org` |
| `APP_URL` | `https://voltanaev.duckdns.org` |
| `APP_ENV` | Must be `production` — enables Secure cookie flag |
| `POSTGRES_PASSWORD` | Strong random; `openssl rand -hex 32` |
| `JWT_SECRET` | Strong random; `openssl rand -hex 32` |
| `SMTP_HOST` | `voltanaev.duckdns.org` (the Poste.io container; no mail.* subdomain on DuckDNS) |
| `SMTP_USER` | `noreply@voltanaev.duckdns.org` (mailbox created in Poste admin) |
| `SMTP_PASSWORD` | Password of the `noreply@` Poste mailbox |
| `SMTP_FROM` | `noreply@voltanaev.duckdns.org` |
| `VAPID_PUBLIC_KEY` | Fresh prod pair: `npx web-push generate-vapid-keys` (don't reuse dev) |
| `VAPID_PRIVATE_KEY` | …the private half of that pair |

> The VPS only has Node (from bootstrap), not Go — so `npx web-push generate-vapid-keys` is the
> generator to use here. On a dev host with the Go toolchain you can instead run
> `cd voltana-api && go run ./cmd/genvapid`, which prints the same `VAPID_*` env block.
| `BALE_BOT_TOKEN` | From Bale BotFather |
| `BALE_BOT_USERNAME` | Your bot's username (without @) |

> **`VITE_API_URL` stays unset.** The SPA talks to the API same-origin through nginx;
> the API has no CORS middleware, so a cross-origin base URL would break it.

### Phase 3: TLS Certificate

Certbot needs ports 80/443 free. Do this **before** starting the full stack.
Issue **one** cert for the single DuckDNS subdomain — Poste.io reuses it (no separate
mail cert, no port-80 fight):

```bash
DOMAIN=voltanaev.duckdns.org
certbot certonly --standalone -d "$DOMAIN"
```

Certificates land at `/etc/letsencrypt/live/$DOMAIN/`, i.e.
`/etc/letsencrypt/live/voltanaev.duckdns.org/`.

Set up auto-renewal — reload **both** nginx and Poste so each picks up the renewed cert:

```bash
crontab -e
# Add this line:
0 3 * * * certbot renew --quiet --deploy-hook "docker compose -f /opt/voltana/docker-compose.yml -f /opt/voltana/docker-compose.prod.yml restart nginx poste"
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

### Phase 6: Admin User (if needed)

**The first registered user automatically becomes admin** (`users.is_admin` is set to
`NOT EXISTS (SELECT 1 FROM users)` on insert), so normally there is nothing to do here —
just register your account first.

Use the command below only to **promote a different/later account**, or to **restore admin**
if something went wrong:

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
curl https://voltanaev.duckdns.org/health
# → {"status":"ok"}

# HTTP → HTTPS redirect
curl -I http://voltanaev.duckdns.org/
# → HTTP/1.1 301  (Location: https://voltanaev.duckdns.org/)

# Security headers
curl -sI https://voltanaev.duckdns.org/health \
  | grep -E "Strict-Transport|X-Frame|X-Content"

# API auth is working
curl -s https://voltanaev.duckdns.org/v1/me
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

## Mail Server (Poste.io)

Voltana sends verification emails through a self-hosted **Poste.io** container
(`poste` in `docker-compose.prod.yml`). No third-party SMTP relay is involved.

### How it's wired

- The `poste` service publishes the TLS/MX mail ports (**25, 465, 587, 993**) on the host;
  the Voltana API connects to it at `SMTP_HOST=voltanaev.duckdns.org:587` (STARTTLS).
- The admin UI (**8443**) and plaintext POP3/IMAP (**110/143/995**) are bound to
  `127.0.0.1` only — they are **not** reachable from the internet. Reach the admin via an
  SSH tunnel (below).
- Mail data persists at `/var/lib/voltana/mail`.
- **TLS:** Poste reuses the certbot cert (issued for `voltanaev.duckdns.org`). The cert dir
  is mounted read-only; you point Poste at it in the admin (step 4).
- **DuckDNS limit:** no `mail.*` subdomain and no MX/SPF/DKIM/DMARC/PTR are possible, so
  mail to external providers will be spam-foldered or rejected. Usable for in-app / testing.
- **Memory:** ClamAV is disabled (`DISABLE_CLAMAV=TRUE`) to fit the 4 GB host. Remove that
  env var only on a larger VPS if you want attachment scanning.

> ⚠️ **Docker bypasses UFW.** Published container ports are inserted into Docker's own
> iptables chain *ahead* of UFW, so a UFW `deny` will **not** block a published port. That
> is why 110/143/995/8443 are bound to `127.0.0.1` in compose rather than merely left out
> of the UFW allow-list. Do not "fix" this by publishing them on `0.0.0.0`.

### First-run setup

1. **Start the stack** (Poste comes up with the rest):
   ```bash
   cd /opt/voltana
   docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
   docker compose -f docker-compose.yml -f docker-compose.prod.yml ps poste
   ```

2. **Open the admin UI over an SSH tunnel** (8443 is localhost-only on the VPS):
   ```bash
   # On your laptop:
   ssh -L 8443:127.0.0.1:8443 root@VPS_IP
   # then browse to:  https://localhost:8443
   ```
   Complete the Poste first-run wizard. Set the **mail server hostname** to
   `voltanaev.duckdns.org` and create the admin account.

3. **Create the sending mailbox.** In the admin: *Virtual domains → voltanaev.duckdns.org →
   Manage → Create box* → `noreply@voltanaev.duckdns.org` with a strong password. Put that
   password in `/opt/voltana/.env` as `SMTP_PASSWORD`, then redeploy the API so it picks it up:
   ```bash
   nano /opt/voltana/.env          # set SMTP_PASSWORD
   docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d api
   ```

4. **Point Poste at the certbot cert.** Admin → *System settings → TLS certificate* →
   choose the custom/existing option and set:
   - Certificate: `/etc/letsencrypt/live/voltanaev.duckdns.org/fullchain.pem`
   - Private key: `/etc/letsencrypt/live/voltanaev.duckdns.org/privkey.pem`

   Save. (These are mounted read-only into the container.)

5. **DKIM/SPF/DMARC — not available on DuckDNS.** These require publishing TXT/MX records on
   the domain, which DuckDNS does not allow. Skip them. Without them (and without a PTR for
   the IP), external providers like Gmail will spam-folder or reject the mail — this is
   expected on a DuckDNS domain. Use a real owned domain if deliverable email is required.

### Send a test from the API

After the mailbox exists and the API has restarted, register a new account — the
verification email is sent via Poste. On a DuckDNS domain expect external recipients
(Gmail etc.) to spam-folder or reject it; an internal test mailbox is the reliable check.

---

## Deliverability Checklist

> ⚠️ **DuckDNS cannot pass a real deliverability audit.** MX/SPF/DKIM/DMARC and PTR all
> require DNS/zone control that DuckDNS does not provide, so a mail-tester.com run will
> score low and Gmail/Outlook will spam-folder or reject mail from
> `noreply@voltanaev.duckdns.org`. This is expected and **not** a deploy bug. The Poste.io
> mailbox is still fine for in-app / internal testing. To get deliverable verification
> email, move to a **real owned domain** and follow the SPF/DKIM/DMARC/PTR steps from the
> mail provider's docs.

What you can still verify on DuckDNS:

- [ ] **A record**: `dig +short voltanaev.duckdns.org` → VPS IP.
- [ ] **Valid TLS on the mail host**: Poste uses the `voltanaev.duckdns.org` cert (the same
      one nginx serves).
- [ ] **SMTP reachable**: the API logs show mail sent without an SMTP error
      (`docker compose logs api | grep -i mail`).
- [ ] **Internal mailbox test**: mail sent to another mailbox on the same Poste server
      arrives (external delivery is the part DuckDNS can't satisfy).

---

## Production Smoke Test

Operator-run after the deploy completes. This is the pass that retires the standing
"UI not clicked / no browser on host" caveat carried since TASK-0033.

1. **App loads over HTTPS with a valid cert**
   ```bash
   curl -I https://voltanaev.duckdns.org/   # → 200, valid TLS
   curl -I http://voltanaev.duckdns.org/    # → 301 → https://voltanaev.duckdns.org/
   ```
   In a browser: `https://voltanaev.duckdns.org` loads, padlock valid.

2. **Email verification end-to-end** — register an account → the verification email is sent
   via Poste. On DuckDNS, external delivery (Gmail etc.) is unreliable; verify against an
   internal Poste mailbox, or check the API logs confirm the send.

3. **OTP via Bale** — request an OTP / link the bot. On the Iranian VPS the
   contact-share flow is reachable for the first time; confirm the code arrives in Bale and
   verifies.

4. **Web push** — in Settings → Notifications, enable push, accept the browser permission,
   and confirm the admin "test push" notification is received on the device.

5. **PWA install** — from `https://voltanaev.duckdns.org`, the browser offers "Install"; the
   installed app opens standalone at the domain root (manifest `scope`/`start_url` = `/`).

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
| Email not delivered | `docker compose logs poste`; test SMTP: `swaks --to you@gmail.com --from noreply@voltanaev.duckdns.org --server voltanaev.duckdns.org:587 -tls` (note: external delivery is unreliable on DuckDNS — no SPF/DKIM/MX/PTR) |
| Email lands in spam | Run the Deliverability Checklist — usually missing PTR or DKIM not published |
| Poste admin unreachable | It's bound to `127.0.0.1:8443` — tunnel first: `ssh -L 8443:127.0.0.1:8443 root@VPS_IP` |
| Mail port 25 blocked | Some VPS providers block outbound 25 by default — open a support ticket to unblock |
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
| `docker-compose.prod.yml` | Compose overlay: ports 80/443, certs, postgres bind-mount, **Poste.io mail** |
| `docs/DEPLOY.md` | Local/WSL dev setup + redirect to this guide for production |
| `docs/DEPLOY_PRODUCTION.md` | This file — canonical guide + copy-paste runbook |
