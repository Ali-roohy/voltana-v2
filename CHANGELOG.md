# Changelog

All notable changes to Voltana V2 are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Granular per-task records (specs, reviews, acceptance evidence) live in `.ai/workflows/TASK-*.md`.

## [Unreleased]

### Added
- GitHub repository governance: SemVer (`VERSION` + tags), issue/PR templates, `CODEOWNERS`, `SECURITY.md`,
  CI pipeline (Go build/vet/test + frontend typecheck/build), and this `CHANGELOG.md` (TASK-0015).
- Project documentation: `README.md`, `docs/SETUP.md`, `docs/ARCHITECTURE.md`.

---

## [0.3.0] — 2026-06-03 — Phase 3 (partial)

### Added
- **Charging-station map & API** (TASK-0013): interactive keyless **Leaflet + OpenStreetMap** map with
  database-backed station markers and a click-to-detail panel. New `/v1/stations` endpoints — `GET` list
  (with optional bounding-box filter) and `GET /:id` open to any authenticated user; `POST`/`PUT`/`DELETE`
  restricted to admins. New `users.is_admin` role + `AdminOnly` middleware (fresh per-request DB check; 403
  before any lookup, no enumeration). Migrations `000006` (is_admin), `000007` (charging_stations),
  `000008` (Tehran seed).

### Changed
- **Release & infra hardening** (TASK-0014): reproducible redeploy via `docker compose up -d --build api`;
  nginx re-resolves the API container without a reload; MailHog dev SMTP catcher; compose now threads
  `APP_URL`/`SMTP_*`/`APP_ENV`.
- Pinned `react-leaflet` to v4 (React 18 compatibility); added `@types/leaflet`.

### Fixed
- Station create rejecting `latitude:0` / `longitude:0` (the equator/prime meridian) — request lat/lng are now
  `*float64` with presence + service-side bounds validation.
- Battery SOH lower-bound floor so a sub-0.001-kWh estimate can't trip the `soh_pct > 0` DB check (TASK-0014).

---

## [0.2.0] — 2026-06-02 — Phase 2: Intelligence Layer

### Added
- **Battery health analytics** (TASK-0007 / TASK-0008): delta-SOC State-of-Health estimation with
  `battery_health_snapshots` history, chemistry-aware charging recommendations, the `/v1/analytics/dashboard`
  lifetime-totals endpoint (Redis cache-aside, busted on write), the SOH history endpoint, and the dashboard
  fleet cards + Recharts SOH trend.
- **Email verification gate** (TASK-0009): login `403 EMAIL_NOT_VERIFIED`, `/auth/verify-email` and
  `/auth/resend-verification` (rate-limited, anti-enumeration), SHA-256 single-use 24h tokens behind a
  `Mailer` interface.
- **Cost & history UX** (TASK-0010 / 0011 / 0012): TOU cost breakdown card (shared `lib/cost.ts`), monthly
  cost-trend chart + avg cost/session, and session-history date-range filters with a tap-to-expand detail view.

---

## [0.1.0] — 2026-06-01 — Phase 1: Solid Foundation

### Added
- **Self-hosted stack** (TASK-0001): Docker Compose bringing up postgres → redis → migrate → api → nginx.
- **Auth API** (TASK-0002): self-managed JWT — access token in memory, refresh token in an httpOnly cookie,
  per-IP rate limiting, single-use refresh-token rotation.
- **Cars & EV models** (TASK-0003): user-owned car CRUD + a seeded shared EV-model catalog.
- **Charging sessions** (TASK-0004): session CRUD with computed/override cost and user isolation.
- **User settings** (TASK-0005): `GET/PUT /v1/settings`, auto-created on first GET (TOU rates, default car).
- **Frontend off Supabase** (TASK-0006): React app refactored onto the Go API — feature-based data layer,
  in-memory JWT with silent refresh.

[Unreleased]: https://github.com/Ali-roohy/voltana-v2/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/Ali-roohy/voltana-v2/releases/tag/v0.3.0
[0.2.0]: https://github.com/Ali-roohy/voltana-v2/releases/tag/v0.2.0
[0.1.0]: https://github.com/Ali-roohy/voltana-v2/releases/tag/v0.1.0
