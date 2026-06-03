# ⚡ Voltana V2

**Voltana** is a self-hosted EV (electric-vehicle) fleet & charging management platform. It tracks vehicles,
charging sessions and costs, estimates battery health (State-of-Health) from real charging data, and maps
charging stations — all on infrastructure you own.

V2 replaces the original Supabase MVP with a fully self-hosted **Go + PostgreSQL** stack that runs anywhere
Docker does. The development environment is WSL2; the deployment target is a small Ubuntu VPS
(2 vCPU / 4 GB RAM) via Docker Compose.

> **Status:** Phases 1 & 2 complete; Phase 3 in progress. See [`changelog.md`](changelog.md) for the full
> history and [`.ai/context.md`](.ai/context.md) for live project state.

---

## ✨ Features

- **Authentication** — self-managed JWT (access token in memory, refresh token in an httpOnly cookie),
  email verification gate, per-IP rate limiting, single-use refresh-token rotation.
- **Vehicles** — CRUD for user-owned cars, linked to a shared EV-model catalog (battery capacity, chemistry).
- **Charging sessions** — log sessions with energy, cost, and peak/mid/off-peak (TOU) cost breakdown.
- **Analytics** — lifetime fleet dashboard (kWh, cost, km, efficiency) + per-car **battery State-of-Health**
  estimated from charging history, with a chemistry-aware charging recommendation.
- **Charging-station map** — interactive, **keyless** Leaflet + OpenStreetMap map with database-backed station
  markers; **admin-only** station management API.
- **Self-hosted** — one `docker compose up` brings up the whole stack; no third-party SaaS dependency.

---

## 🧱 Tech Stack

| Layer | Technology |
|---|---|
| **API** | Go (Gin), layered `handler → service → repository → domain` |
| **Database** | PostgreSQL 16, migrations via `golang-migrate` |
| **Cache / Queue** | Redis 7 (refresh-token store, rate limiting, analytics cache) |
| **Auth** | Self-managed JWT — access token in React memory, refresh token in an httpOnly cookie |
| **Frontend** | React 18 + Vite, feature-based structure, TanStack Query, Leaflet, Recharts |
| **Mobile** | Capacitor wrapping the React PWA (no separate native codebase) |
| **Ingress** | nginx (reverse proxy for the API) |
| **Dev email** | MailHog (catches verification emails locally) |
| **Infra** | Docker Compose: `postgres → redis → migrate → api → nginx` (+ `mailhog` in dev) |

---

## 🚀 Quick Start

> Full instructions, including how to create the first admin user, are in **[docs/SETUP.md](docs/SETUP.md)**.

```bash
# 1. Clone
git clone git@github.com:Ali-roohy/voltana-v2.git
cd voltana-v2

# 2. Configure environment (fill in real secrets)
cp .env.example .env
#   edit .env → set POSTGRES_PASSWORD and a 32+ char JWT_SECRET
#   for local email testing, set: SMTP_HOST=mailhog  SMTP_PORT=1025

# 3. Bring up the full backend stack (postgres → redis → migrate → api → nginx [+ mailhog])
docker compose up -d --build

# 4. Verify the API is up
curl http://localhost/health        # → {"status":"ok"}
```

The API is now served by nginx on **http://localhost** (port 80). To run the **frontend** in development:

```bash
cd voltana-web
cp .env.example .env                 # VITE_API_URL=/ (same-origin via nginx)
npm install
npm run dev                          # Vite dev server (http://localhost:5173)
#   or: npm run build && npm run preview   # production build preview (http://localhost:4173)
```

---

## 📦 Repository Layout

```
voltana_V2/
├── voltana-api/          # Go backend (Gin) — handler/service/repository/domain
├── voltana-web/          # React + Vite frontend (feature-based)
├── migrations/           # golang-migrate SQL migrations (000001…000008)
├── nginx/                # nginx reverse-proxy config (API ingress)
├── docker-compose.yml    # full self-hosted stack
├── .env.example          # backend env template (copy → .env)
├── docs/                 # SETUP + ARCHITECTURE guides
├── changelog.md          # per-task changelog
└── .ai/                  # workflow specs, ADRs, live project context
```

---

## 🖼️ Screenshots

> _Placeholder — add screenshots here._

| Dashboard | Charging Sessions | Station Map |
|---|---|---|
| _`docs/img/dashboard.png`_ | _`docs/img/charging.png`_ | _`docs/img/map.png`_ |

---

## 📚 Documentation

| Doc | What's in it |
|---|---|
| **[docs/SETUP.md](docs/SETUP.md)** | Prerequisites, `.env` config, first run, creating the first admin, MailHog, frontend dev |
| **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** | Layer diagram, folder structure, full API reference, data model, auth & admin flows |
| **[changelog.md](changelog.md)** | Per-task changelog |
| **[.ai/spec/](.ai/spec/)** | Accepted Architecture Decision Records (ADR-001…003) |

---

## 🔐 Security Notes

- **Never commit secrets.** `.env`, `.env.production`, and `voltana-web/.env` are git-ignored; commit only the
  `.env.example` placeholders.
- Access tokens live only in React memory (never `localStorage`); the refresh token is an httpOnly cookie.
- On the production VPS set `APP_ENV=production` so the refresh cookie gets the `Secure` flag.
- Admin privileges are granted **out-of-band** via SQL (there is no self-serve admin signup) — see
  [docs/SETUP.md](docs/SETUP.md#create-the-first-admin-user).

---

## 📄 License

Proprietary / self-hosted. (Add a license here if you intend to open-source.)
