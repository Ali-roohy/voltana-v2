# Deployment Guide

## Production Deployment (voltanaev.duckdns.org)

The complete production deployment guide is in:
→ [DEPLOY_PRODUCTION.md](./DEPLOY_PRODUCTION.md)

Covers: VPS bootstrap, DNS, TLS, Poste.io mail server, VAPID keys, bot tokens, systemd,
backups, and the copy-paste runbook.

---

## Development Setup (local / WSL)

### Prerequisites
- Docker + Docker Compose v2
- Node.js 20+

### Quick Start
```bash
git clone https://github.com/Ali-roohy/voltana-v2.git
cd voltana-v2

# Copy dev env (placeholders are fine for local)
cp .env.example .env

# Build the frontend — nginx serves voltana-web/dist (gitignored, so build it first)
(cd voltana-web && npm install && npm run build)

# Start the stack (postgres → redis → migrate → api → nginx, migrations run automatically)
docker compose up -d --build
```

Access:
- `http://localhost` — frontend (SPA + API proxied under `/v1`, `/auth`, `/health`)
- `http://localhost:8025` — MailHog (dev email catcher; set `SMTP_HOST=mailhog` / `SMTP_PORT=1025` in `.env` to route mail here)

### First User = Admin
The first registered user automatically becomes admin (the `users.is_admin` column is set to
`NOT EXISTS (SELECT 1 FROM users)` on insert). Every later signup is a normal user; promote
others from the admin UI or via SQL.

### Useful Commands
```bash
docker compose logs -f api          # API logs
docker compose up -d --build        # Rebuild after a code change
docker compose down -v              # Full reset (drops the database volume)
```

> After a frontend change, re-run the build (`cd voltana-web && npm run build`) — nginx serves
> the static `dist/`, so a rebuild is needed for the change to show.

---

For detailed architecture, see the main [README.md](../README.md).
