# Voltana Changelog

## [Unreleased]

---

## [TASK-0013] — 2026-06-03 — Map + Real Station Data — **Phase 3**

Replaced the embedded-iframe map with an interactive **keyless Leaflet + OpenStreetMap** map rendering
DB-backed charging-station markers, plus an **admin-only** station CRUD API gated by a new `users.is_admin`
role. Closed by qa_supervisor: architect ✅ + dev_supervisor ✅ + security ✅ + qa ✅ (9/9 live).

### Added
- **Migrations** `000006_users_is_admin` (`is_admin BOOLEAN NOT NULL DEFAULT false`), `000007_charging_stations`
  (shared reference table — no `user_id`; DB-level lat/lng + `power_kw>0` CHECKs + `set_updated_at` trigger),
  `000008_seed_charging_stations` (5 Tehran demo stations). Applied live: schema **v5 → v8**.
- **`/v1/stations` API** (under the JWT group): `GET` list (markers — id/name/lat/lng/connector/power, with an
  optional `?min_lat&max_lat&min_lng&max_lng` bounding-box filter) + `GET /:id` detail — open to any authed
  user; `POST`/`PUT`/`DELETE` — **admin-only**. Full handler→service→repository slice mirroring the cars module.
- **`AdminOnly` middleware** + `AuthService.IsAdmin` — a **fresh DB check** per write (not baked into the JWT),
  so revoking admin takes effect immediately; denies non-admins with **403 before any station lookup** (no
  enumeration). Admin bootstrap is out-of-band SQL only (`UPDATE users SET is_admin=true …`).
- **Frontend** `features/stations/{api.ts,hooks.ts}` (TanStack Query, no direct `fetch`) + rewritten
  `pages/Map.tsx` (react-leaflet `MapContainer`/`TileLayer`/`Marker`/`Popup`, OSM tiles, click→detail panel).

### Changed
- `voltana-web/package.json` — **react-leaflet `^5.0.0` → `^4.2.1`** (v5 requires React 19; app is React 18.3.1)
  + added `@types/leaflet`. Same component API, no code change.
- `domain.User` / `user_repo` carry `is_admin`; `cmd/server/main.go` wires the station repo/service/handler + 5 routes.

### Fixed
- **`latitude:0` / `longitude:0` rejection** (caught in live smoke) — lat/lng request fields are now `*float64`
  with `binding:"required"` (pointer presence distinguishes omitted→400 from a valid 0); bounds validated in
  the service with descriptive messages. The equator now creates correctly.

### Outcome
- dev_supervisor ✅ + security ✅ (admin boundary) + qa ✅ (9/9). qa verified on a **clean `docker compose up -d
  --build api`** (in-container Go build 50.4s, no wedge): 5 markers, non-admin POST→403, admin POST→201
  (equator), PUT 200, DELETE 204→404, bbox subset + partial→400, seed intact; host `go test` ok, `tsc` 0 +
  `npm build` ✓. **First Phase-3 feature task done** (after the 0014 infra hardening).

---

## [TASK-0014] — 2026-06-02 — Release & Infra Hardening — **Phase 3**

Made deployment reproducible and removed the manual hand-deploy friction that trailed TASK-0009/0007/0008
(host-compile + `Dockerfile.runtime` swap + `nginx -s reload` + a DB `is_email_verified` flip).

### Changed
- `docker-compose.yml` — api service now passes `APP_URL` + `SMTP_HOST/PORT/USER/PASSWORD/FROM`
  (`${VAR:-default}` style); a clean `compose up` ships a working verify link.
- `nginx/nginx.conf` — replaced the static `upstream` (which cached the api IP at startup) with
  `resolver 127.0.0.11 valid=10s ipv6=off; set $upstream http://api:9090; proxy_pass $upstream$request_uri;`
  so nginx **re-resolves the api on restart without a reload**; all `proxy_set_header`s (incl. `X-Real-IP`) kept.
- `.env.example` — documented the MailHog dev option (`SMTP_HOST=mailhog`/`1025`) and the
  `APP_ENV=production` VPS note (closes carry-forward N1: `Secure` refresh cookie).

### Added
- `mailhog` service (dev-only) — verification emails land in the web UI at `http://localhost:8025`
  (SMTP `1025`, internal); removes the manual `is_email_verified` DB flip in smoke tests.

### Fixed
- SOH lower-bound floor (`analytics_service.go`): `if soh < 0.01 { soh = 0.01 }` so a sub-0.001-kWh estimate
  can't round `soh_pct` to 0.00 and trip the DB `CHECK (soh_pct > 0)` on Save (TASK-0007 carry-forward). +unit test.

### Outcome / runbook
- **Reproducible redeploy: `docker compose up -d --build api`** (Compose v2). api redeploys no longer touch
  nginx; an nginx *config* change still needs a one-time `docker compose restart nginx`. In-container build is
  the documented default with host-binary `Dockerfile.runtime` as the loaded-host fallback.
- release ✅ · dev_supervisor ✅ (5/5) · qa ✅ (5/5 acceptance, **zero manual deploy steps**).

---

## [TASK-0008] — 2026-06-02 — Dashboard Analytics API + Battery Chart — **Phase 2 (analytics)**

Lifetime fleet stats endpoint (Redis-cached) + a battery-health trend chart and SOH card on the dashboard.
**Completes the Phase-2 analytics chain (0007→0008).** No new migration (reuses `battery_health_snapshots`).

### Added
- `GET /v1/analytics/dashboard` → `{total_kwh, total_cost, total_km, avg_kwh_per_100km, session_count}` —
  lifetime, all cars. Cache-aside in Redis (key `analytics:dashboard:<userID>`, **TTL 5m**); a charging
  create/update/delete busts the key via the TASK-0007 on-write hook. `total_cost` = persisted `SUM(cost)`;
  `total_km` = Σ `cars.odometer_km`; `avg_kwh_per_100km` = `total_kwh/(total_km/100)`, **null when `total_km==0`**.
- `GET /v1/analytics/battery/:car_id/history?limit=30` — the most recent `limit` (≤100) snapshots in
  chronological (ASC) order for the trend chart; ownership-gated (404 cross-user).
- `repository`: `ChargingRepository.AggregateByUser` (SQL SUM/COUNT, NULL→0, no list cap),
  `BatteryRepository.ListByCar` (newest-N via `ORDER BY computed_at DESC LIMIT`, reversed to ASC),
  `RedisTokenStore` cache helpers (`CacheGet/Set/Del`). `domain.DashboardStats`.
- Frontend `features/analytics/{api.ts,hooks.ts}`; dashboard fleet cards (total km, avg kWh/100km), SOH card
  (latest `soh_pct` + confidence / friendly empty state), and a Recharts battery-health trend (`soh_pct` vs
  `computed_at`, y 0–100%) with a car selector shown only for multi-car users.

### Changed
- `analytics_service.go` `RecomputeAsync` also `DEL`s the dashboard cache key on charging writes.
- `features/charging/hooks.ts` invalidates `["battery"]`/`["battery-history"]` (plus the existing `["dashboard"]`).

### Fixed (in review)
- Battery history initially returned the **oldest** N (would freeze the trend once a car exceeded the limit) —
  changed to newest-N + reverse to chronological; added a 40-snapshot ordering test.

### Notes
- `avg_kwh_per_100km` is a lifetime approximation (no per-session odometer); a windowed metric needs a schema
  add (future). `total_km` sums up to 100 cars.
- dev_supervisor ✅ (5/5, fix re-verified) · qa ✅ (5/5 live + isolation: dashboard 210/4200/15000/1.4/7,
  SOH 88%, history chronological, cache busts 210→240 on write).

---

## [TASK-0007] — 2026-06-02 — Battery Health Snapshots — **Phase 2 (analytics)**

Estimate per-car battery State of Health (SOH) from charging history via the delta-SOC method, persist a
snapshot history, and serve chemistry-aware care recommendations. First half of the Phase-2 analytics chain
(TASK-0008 consumes this). **No asynq** — recompute runs synchronously on charging-session writes (architect
ruling for the 2 vCPU / 4 GB host).

### Added
- `migrations/000005_battery_health_snapshots.{up,down}.sql` — append-only history table (CHECK
  `soh_pct ∈ (0,100]`, `confidence ∈ {low,medium,high}`; index `(car_id, computed_at DESC)`).
- `internal/domain/battery_health.go` — `BatteryHealthSnapshot` + `BatteryRecommendation`.
- `internal/repository/battery_repo.go` — `BatteryRepository.Save` / `GetLatest(userID, carID)`; reads scoped
  by userID (no unscoped accessor).
- `internal/service/analytics_service.go` — delta-SOC algorithm: qualifying session = `kwh>0` ∧ both SOC ∧
  `end>start` ∧ `Δsoc≥25`; `cap = (kwh·0.88)/(Δsoc/100)` (η=0.88 charging-efficiency constant); SOH =
  `100·weightedAvg(cap, weight=Δsoc)/nominal`, clamped to (0,100]; `<5` qualifying or no linked ev_model →
  insufficient-data. Coalescing async recompute (`RecomputeAsync`: per-car inflight+pending, detached context,
  30s timeout).
- `internal/service/health_advisor.go` — LFP→ceiling 100; NMC/NCA→80; null/unknown→80 generic.
- `internal/handler/analytics_handler.go` — `GET /v1/analytics/battery/:car_id` (200 snapshot / 200
  `{status:"insufficient_data",qualifying_sessions}` / 404 / 400) and `GET /v1/analytics/recommendations/:car_id`.

### Changed
- `internal/service/charging_service.go` — `HealthRecomputer` interface + `SetHealthRecomputer`; create/update/
  delete trigger a recompute (delete fetches the session first to learn its car). Nil-safe.
- `cmd/server/main.go` — wired `batteryRepo` + `analyticsSvc`, the recompute hook, and the two `/v1/analytics/*`
  routes under the JWT group.

### Security / correctness
- user_id isolation in the repo + `ownedCar` (cross-user/unknown car → 404). Charging losses modeled via η so
  SOH cannot read >100%. dev_supervisor ✅ (5/5) · qa ✅ (6/6 live: SOH 88%, LFP advice, insufficient 200,
  isolation 404). 10 new analytics test functions on host.

### Decision
- Dropped the spec's `asynq` worker (architect): synchronous coalesced recompute on write — no new container.

### Known follow-ups (not blockers)
- SOH lower bound unclamped — a sub-0.001-kWh session could round `soh_pct` to 0.00 and trip the DB CHECK on
  Save (not reproducible with real data); add a one-line floor or relax CHECK to `>= 0`.
- Recurring stale-redeploy + nginx upstream-IP cache + duplicate container → release ticket for a reproducible
  redeploy path; dev SMTP catcher (MailHog) to avoid the manual `is_email_verified` flip in smoke tests.

---

## [TASK-0009] — 2026-06-02 — Email Verification Gate — **Phase 2 (backend)**

Closed the email-verification gap left open by TASK-0002: registration now issues a verification token and
login refuses unverified accounts. Backend (Go API) + bug **#7** verify/resend UI (frontend). Email sending
sits behind a `service.Mailer` interface so SMTP is never reached in unit tests. **No new migration** — the
`email_verification_tokens` table (`000002`) already fits (`token_hash VARCHAR(64)` = SHA-256 hex).

### Added
- `internal/repository/verification_repo.go` — `VerificationTokenRepository`: `ReplaceVerificationToken`
  (delete-then-insert in a txn → one outstanding token per user) and `ConsumeVerificationToken` (single txn:
  `SELECT … FOR UPDATE` by hash + unexpired → delete user's tokens → flip `users.is_email_verified`).
- `internal/mailer/mailer.go` — `SMTPMailer` (`net/smtp`, `SMTP_*`) + `LogMailer` (dev; never logs the
  token/URL or full recipient). Satisfies `service.Mailer` structurally (no import of service).
- `POST /auth/verify-email` (200 verified / 200 already-verified / 400 INVALID_REQUEST / 400
  INVALID_VERIFICATION_TOKEN / 429) and `POST /auth/resend-verification` (always 202 / 400 / 429),
  `{error,code}` envelope. Frontend `pages/VerifyEmail.tsx` + `/verify-email` route.

### Changed
- `internal/service/auth_service.go` — `Mailer` interface; errors `ErrEmailNotVerified` /
  `ErrInvalidVerificationToken`. **Register** mints a 256-bit base64url token (only SHA-256 hex stored) and
  emails the link — **best-effort** (failures logged by user ID, registration still succeeds). **Login**
  returns `ErrEmailNotVerified` **only after** a successful password check (wrong password still
  `ErrInvalidCredentials` — no enumeration). `VerifyEmail` (per-IP 20/15m) + `ResendVerification` (per-IP 5/h
  + per-email 3/h on `sha256(lowercased email)` + 60s cooldown; always nil for anti-enumeration).
- `internal/handler/auth_handler.go` — login 403 `EMAIL_NOT_VERIFIED`; two new public routes.
- `cmd/server/main.go` — wired `verifRepo` + mailer (SMTP when `SMTP_HOST` set, else log mailer) + `APP_URL`.
- `voltana-web` — `features/auth/api.ts` (`verifyEmail`/`resendVerification`); `pages/Auth.tsx` **register no
  longer auto-logs-in** → "check your email" screen (resend + back), login 403 routes to the same screen;
  `App.tsx` route; `i18n/locales/{en,fa}.json` new `auth.*` keys (en↔fa parity).

### Security
- SHA-256-hash-only token storage (raw never persisted/logged); resend always 202 (anti-enumeration);
  rate limits backed by forge-proof ClientIP (`X-Real-IP` only + trusted proxies); 403 gate only after a
  passing credential check. dev_supervisor ✅ (6/6) · security ✅ (5/5) · qa ✅ (5/5 live smoke).

### Decision
- Email send is **best-effort**: registration succeeds even if SMTP fails; the user can resend.

### Known follow-ups (→ release/infra, not blockers)
- `docker-compose.yml` `api` service does not pass `APP_URL` / `SMTP_*` and still builds the wedge-prone
  in-container `Dockerfile` — add those env vars + a host-binary runtime-image path for a reproducible deploy.
- verify→login end-to-end is unit-covered only (no dev SMTP catcher on host to capture the raw token).
- N1: set `APP_ENV=production` on the VPS so the refresh cookie gets `Secure`.

---

## [TASK-0012] — 2026-06-01 — Session History Filters + Detail View — **Phase 2 #3**

Made the charging history browsable: a server-side **date-range filter** (the API already supported
`?from`/`?to`) plus a **tap-to-expand** detail accordion. Frontend-only — **no API/DB change**.

### Changed
- `voltana-web/src/features/charging/api.ts` — added `ChargingListFilter { car_id?, from?: Date, to?: Date }`;
  `listChargingSessions(filter?)` serializes to the query (`limit=100` + only set params; `from`=start-of-day,
  **`to`=end-of-day `23:59:59.999` inclusive**, RFC3339). Return type unchanged (`ChargingSession[]`).
- `voltana-web/src/features/charging/hooks.ts` — `useChargingSessions(filter?)` with a **filter-aware query
  key** (base key when no filter → dashboard unaffected) + `placeholderData: keepPreviousData`. Mutation
  invalidation prefix-matches the filtered keys.
- `voltana-web/src/pages/Charging.tsx` — from/to `JalaliDatePicker`s + Clear; car `<Select>` shown **only for
  multi-car users**; removed the client-side filter slice (all filtering now server-side); newest-first sort;
  `invalidRange` guard (from > to → message, filter omitted); **tap-to-expand accordion**: collapsed summary
  (car · date · kWh · cost) → expanded detail (start time + duration · `TOUBreakdown` · location · **`notes`**,
  newly surfaced · `SOCAnalysis` · inline cost-override); loading/error/empty-in-range/invalid-range states.
- `voltana-web/src/i18n/locales/{en,fa}.json` — `charging.{from,to,clearFilters,noSessionsInRange,invalidRange,notes}`.

### Evidence
- Reviews: feature ✅ · dev_supervisor ✅ (5/5) · qa ✅ (API-verified) · qa_supervisor ✅ (with caveat)
- `npx tsc --noEmit` exit 0 · `npm run build` ✓ (clean) · preview (0.0.0.0:4173) HTTP 200
- qa proved server-side `?from`/`?to` filtering + **inclusive end-of-day** via browser-equivalent curl
  (seeded 3 sessions; May range returned only the May-31T20:00 boundary session)

### Caveat (operator-accepted)
- Browser UI scenarios **expand-detail** and **clear-filters** were code-/data-verified only — Playwright's
  CDN is geo-blocked and no system browser is available. Retire with a UI smoke when a browser is obtainable.

---

## [TASK-0011] — 2026-06-01 — Monthly Cost Trend Chart — **Phase 2 #2**

Added the money dimension to the dashboard: a monthly **cost** trend chart beside the existing energy
trend, plus **total spend** and **avg cost / session** headline figures. Frontend-only — **no API/DB
change**; cost derived via the shared `lib/cost.ts` helper from TASK-0010.

### Changed
- `voltana-web/src/pages/Index.tsx`:
  - `stats` memo — the month-bucket loop now accumulates **both** energy and cost per month
    (cost = `s.cost ?? calcCost(s, rates).total`) into a **single shared `trend: [{month, energy, cost}]`**
    (renamed from `energyTrend`); both charts read it so they share the x-domain. Added scoped
    `sessionCount` and `avgCost` (`= totalCost / count`, `null` at 0 sessions).
  - **Repurposed the dead `avgEfficiency` stat card** (`— kWh/100km`) → **avg cost / session** (تومان, or
    "—" when no sessions).
  - Added a **Monthly Cost `BarChart`** (`dataKey="cost"`, تومان tooltip/axis via `formatNumber`) beside
    the energy line chart; moved the SOC chart to its own full-width row.
- `voltana-web/src/i18n/locales/{en,fa}.json` — added `dashboard.avgCostPerSession` + `dashboard.monthlyCost`.

### Decisions
- **Two separate single-unit charts (energy line + cost bar), not dual-axis** — kWh and Toman are unrelated
  scales; a shared axis would mislead. Currency = Toman, no ÷10 (consistent with TASK-0010).

### Evidence
- Reviews: feature ✅ · dev_supervisor ✅ (5/5) · qa ✅ · qa_supervisor ✅
- `npx tsc --noEmit` exit 0 · `npm run build` ✓ (clean) · preview (0.0.0.0:4173) HTTP 200; operator approved
  skipping the full browser click-through (preview verified working)

### Follow-ups (non-blocking)
- The "Sessions" stat card still uses unscoped `sessions.length` — optional cleanup for a future dashboard touch.

---

## [TASK-0010] — 2026-06-01 — TOU Cost Breakdown Card — **Phase 2 #1**

Surfaced the time-of-use split (peak/mid/off-peak energy + cost) as a reusable stacked breakdown,
rendered per charging session and as a dashboard "This month" summary. Frontend-only — **no API/DB
change**; all data already existed. First Phase-2 task.

### Added
- `voltana-web/src/lib/cost.ts` — single source of truth for TOU cost: `Rates`/`TouCost` types,
  `ratesFromSettings(settings)`, and `calcCost(session, rates) → {peak, mid, offpeak, total}` where
  `total = sum(segments)`. Pure module (type-only imports; no React/fetch). Manual override stays at the
  call site (`session.cost ?? calcCost(...).total`).
- `voltana-web/src/components/TOUBreakdown.tsx` — presentational CSS stacked bar (`variant: inline|summary`),
  props `{peak, mid, offpeak, total?}` of `{kwh, cost}`; peak=red / mid=amber / off-peak=green; zero buckets
  omitted; degraded total-only state; تومان labels; rows read `label: [kwh] kWh · [cost] تومان` (RTL-safe via
  `dir="ltr"` value spans).
- i18n `tou` group in `src/i18n/locales/{en,fa}.json` (thisMonth/peak/mid/offpeak/total/toman/noBreakdown).

### Changed
- `voltana-web/src/pages/Charging.tsx` — `getSessionCost` refactored onto the shared helper (inline rate
  math removed); `<TOUBreakdown variant="inline">` mounted per session card; removed the `$` (`DollarSign`)
  icon from the cost row and relabeled `ریال` → `تومان`.
- `voltana-web/src/pages/Index.tsx` — `stats` memo now derives a current-month `touMonth` aggregate and
  **fixes `totalCost`** to `Σ (s.cost ?? calcCost(s, rates).total)` (was `Σ (s.cost ?? 0)`, which undercounted
  rate-computed sessions); rendered a "This month" `<TOUBreakdown variant="summary">` card.

### Decisions
- **Currency = Toman, treat-as-is** (operator): no ÷10 conversion; the existing `ریال` label flipped to
  `تومان` for a single app-wide unit.

### Evidence
- Reviews: feature ✅ · dev_supervisor ✅ (5/5) · qa ✅ + re-check ✅ · qa_supervisor ✅
- `npx tsc --noEmit` exit 0 · `npm run build` ✓ (clean) · `vite preview` (0.0.0.0:4173) HTTP 200
- Operator browser-confirmed: formatting correct, `$` removed, RTL fixed

### Follow-ups (non-blocking)
- **TASK-0011** reuses `lib/cost.ts` for the monthly cost trend.
- Dashboard "This month" aggregates rate-based costs (ignores rare per-session manual overrides) — documented.

---

## [TASK-0006] — 2026-06-01 — Frontend: Replace Supabase SDK with Go API — **Phase 1 COMPLETE**

Refactored the React MVP off the Supabase JS SDK onto the self-hosted Go API, restructured to a
feature-based data layer (ADR-002) with in-memory JWT auth + silent refresh (ADR-003), and fixed the
8 known bugs. Imported the MVP app into this repo as `voltana-web/`. **Last open Phase-1 task — closes
Phase 1.**

### Added
- `voltana-web/` — the React app brought in-repo (Vite 5 / React 18 / TanStack Query / sonner)
- `src/lib/api.ts` — single `fetch` wrapper: base URL, `Authorization: Bearer` from memory, `credentials:include`, **single in-flight `/auth/refresh` on 401 + one retry** (dedup so refresh rotation can't invalidate parallel callers), `{error,code}` → `ApiError`. **No component calls `fetch()` directly.**
- `src/lib/auth-store.ts` — access token in an **in-memory module var only** (never localStorage/sessionStorage); restored on reload via `/auth/refresh`; JWT `sub` decoded (display-only, unverified) for `user.id`
- `src/features/{auth,cars,ev-models,charging,settings}/{api,hooks}.ts` — feature-based `api.ts` → TanStack `useQuery`/`useMutation` hooks (mutations `invalidateQueries`)
- `voltana-web/.env.example` — `VITE_API_URL=/` and `VITE_NESHAN_API_KEY=` (no real key)

### Changed
- Frontend adapted to the Go schema (Go API unchanged, operator decision): `date`→`started_at`, `energy_kwh`→`kwh_charged`, `*_soc_percent`→`*_soc`, settings rate field-flip; per-session odometer **dropped** (odometer lives on the car) — dashboard distance/efficiency show "—" pending a later source
- Charging form: **default car pre-selected** from settings; **required-field validation** (car · date · total energy >0 · duration >0) blocks submit with red border/label + a single toast

### Removed
- `src/integrations/supabase/` deleted; `@supabase/supabase-js` uninstalled (absent from `package.json`); old Supabase `useAuth` replaced. `grep -r "@supabase" src` → none.

### 8 known bugs
- Fixed (7): #1 `useNavigate` (no `window.location.href`) · #2 Header `invalidateQueries` (no `reload()`) · #3 `VITE_NESHAN_API_KEY` env var · #4 sonner-only (radix toast removed) · #5 single `useChargingSessions` query · #6 `SOCAnalysis` start→end order · #8 Map stub keyed from env
- **Deferred (1): #7 email confirmation gate → TASK-0009** (recorded sequencing decision; register auto-logs-in for now). Not a defect.

### Security (ADR-003)
- Access token in memory only; refresh token is the httpOnly cookie (JS-unreadable); 401→refresh→retry can't loop/leak; client JWT decode is display-only (authorization enforced server-side, repo-layer `user_id` scoping). **Deployment invariant:** `VITE_API_URL` must stay same-origin (nginx).

### Evidence
- Reviews: dev_supervisor ✅ (6/6 checks; initial + 2026-06-01 re-review) · security ✅ (ADR-003 token storage) · qa ✅ · qa_supervisor ✅
- `npm run build` ✓ (built ~13.8s) · `npx tsc --noEmit` exit 0 · `vite preview :4173` HTTP 200
- Operator manual browser test: register/login, default car pre-selected, required fields go red + block on empty submit, cost calc correct, no Supabase console errors

### Follow-ups (non-blocking)
- **TASK-0009** — email verification gate (incl. bug #7 UI)
- **N1** — set `APP_ENV=production` on the VPS so the refresh cookie gets `Secure`
- Optional: strict ADR-002 `features/<name>/Page.tsx` relocation (pages still in `src/pages/`); delete orphaned radix toast files

---

## [TASK-0005] — 2026-05-31 — User Settings API

`GET`/`PUT /v1/settings` for electricity rates + default car, with auto-create-on-first-GET.
No migration — `user_settings` already existed (000001); this adds the Go layers and extends the
`settings_repo` that TASK-0004 introduced.

### Added
- `voltana-api/internal/domain/user_settings.go` — `UserSettings` + `SettingsInput` (input in **domain** so the handler imports only domain+service — D1)
- `voltana-api/internal/service/settings_service.go` (+ `_test.go`) — rate validation (≥0), default-car ownership via reused `CarRepository`, error translation
- `voltana-api/internal/handler/settings_handler.go` — `GET`/`PUT /v1/settings`, `{error,code}` envelope
- `voltana-api/cmd/server/main.go` — DI + settings routes

### Changed
- `voltana-api/internal/repository/settings_repo.go` — extended from read-only `GetRates` (TASK-0004) with `GetOrCreate` (auto-create via `INSERT … ON CONFLICT (user_id) DO NOTHING` + SELECT — a read does not bump `updated_at`) and `Update` (upsert; PUT works whether or not a row exists)

### Behavior
- **GET** auto-creates a default row (rates 0, no default car) on first call (Supabase parity)
- **PUT** is full-replace: omitted rates default to 0; omitted/null `default_car_id` clears it
- `default_car_id` must reference one of the caller's own cars (else `422 INVALID_CAR`)

### Security
- All `user_settings` access keyed by `user_id` from the JWT; `SettingsInput` has no `user_id` field; upsert conflicts on `user_id` so a caller can only ever read/write their own row; `ID`/`UserID` are `json:"-"`

### Evidence
- Reviews: dev_supervisor ✅ · security ✅ (4 controls) · qa ✅ (8 checks) · qa_supervisor ✅
- Host Go `go test ./...` → `ok internal/service ~10s`; schema unchanged at **v4**
- Live smoke: auto-create defaults, PUT persist (20/11.5/5), owned-car 200 / unowned 422, rate -1 → 400, per-user isolation (B sees own zeros), D2 401 envelope

---

## [TASK-0004] — 2026-05-31 — Charging Sessions CRUD API

Authenticated CRUD for user-owned charging sessions under `/v1/charging-sessions`, with server-side
time-of-use cost calculation. The `charging_sessions` table already existed (000001); this adds the
Go layers + routes + a per-period energy migration.

### Added
- `migrations/000004_charging_session_energy_split.{up,down}.sql` — adds `energy_peak_kwh`/`energy_mid_kwh`/`energy_offpeak_kwh` (`kwh_charged` retained as the grand total) for the time-of-use cost model
- `voltana-api/internal/domain/charging_session.go` — `ChargingSession` (+ `ChargingInput`/`ChargingFilter`; input lives in **domain** so the handler imports only domain+service — D1 lesson applied)
- `voltana-api/internal/repository/charging_repo.go` — `user_id`-scoped CRUD + `car_id`/date-range list filter
- `voltana-api/internal/repository/settings_repo.go` — read-only `GetRates` (TASK-0005 extends to full settings CRUD)
- `voltana-api/internal/service/charging_service.go` (+ `_test.go`) — validation (SOC 0–100, time order, non-negative), TOU cost (`peak·peak_rate + mid·mid_rate + offpeak·offpeak_rate`, client cost wins, no energy → NULL), car-ownership via reused `CarRepository`, error translation
- `voltana-api/internal/handler/charging_handler.go` — imports `domain`+`service` only (D1)
- `voltana-api/cmd/server/main.go` — DI + `/v1/charging-sessions` routes
- `voltana-api/Dockerfile.runtime` — dev-only host-build deploy helper (avoids the wedge-prone in-container compile on this host)

### Changed
- `voltana-api/internal/middleware/auth.go` — **D2 fix**: both 401 responses now include `code:"UNAUTHORIZED"` so the whole `/v1` surface returns a uniform `{error,code}` envelope
- `CLAUDE.md` — added "Dev Environment Notes": run Go tests with host Go, never the `golang:1.22-alpine` container (it wedges on this host); don't echo DB secrets

### Security
- Ownership isolation enforced in the repository (`WHERE … AND user_id = $`); cross-user access → **404**; `Update` checks session-ownership before car-ownership so cross-user PUT is 404 (not 422); a session can only reference the caller's own car; cost rates read scoped to caller

### Evidence
- Reviews: dev_supervisor ✅ · security ✅ (7 controls) · qa ✅ · qa_supervisor ✅
- Host Go `go test ./...` → `ok internal/service` (~10s); migration version **4**
- Live smoke: computed cost 54, provided-cost override 123.45, invalid car 422 INVALID_CAR, cross-user 404/404/404, D2 401 envelope

### Follow-ups (non-blocking)
- **Release:** production must build via the canonical multi-stage `Dockerfile`; `Dockerfile.runtime` is dev-only; gitignore the produced `server` binary
- **TASK-0005:** extend `settings_repo.go` to full settings CRUD + auto-create-on-first-GET (rates default to 0 until a settings row exists)

---

## [TASK-0003] — 2026-05-31 — Cars & EV Models CRUD API

Authenticated CRUD for user-owned `cars` + read-only search over the shared `ev_models` catalog,
under the JWT-protected `/v1` group. `cars`/`ev_models` tables already existed (000001), so this
added the Go layers + routes + an `ev_models` seed (no new tables).

### Added
- `voltana-api/internal/domain/{car,ev_model}.go` — `Car` (`UserID` is `json:"-"`, never serialized) and `EVModel` response models
- `voltana-api/internal/repository/{car_repo,ev_model_repo}.go` — pgx repos behind interfaces; every `cars` statement scoped by `user_id`; FTS search on `ev_models` (name_fa OR name_en)
- `voltana-api/internal/service/{car_service,ev_model_service}.go` (+ `_test.go`) — validation, pagination clamping (default 20, max 100), repository→service error translation; unit tests with mock repos
- `voltana-api/internal/handler/{response,car_handler,ev_model_handler}.go` — `{items,limit,offset,total}` list envelope + `{error,code}` error envelope
- `voltana-api/cmd/server/main.go` — wired `GET/POST/PUT/DELETE /v1/cars`, `GET /v1/ev-models[/:id]` on the existing `middleware.Auth` group; DI of car/ev-model repos+services+handlers
- `migrations/000003_seed_ev_models.{up,down}.sql` — `name_en` UNIQUE constraint + 12-model starter seed via `ON CONFLICT (name_en) DO NOTHING` (idempotent)

### Security
- Ownership isolation enforced in the repository (`WHERE … AND user_id = $`); cross-user access returns **404** (not 403) to avoid existence enumeration; `ev_models` exposes zero write endpoints

### Evidence
- Reviews: dev_supervisor ✅ (5/5 layering checks) · security ✅ (5/5 isolation controls) · qa ✅ · qa_supervisor ✅
- Live smoke via nginx `:80`, two users A/B — 9/9 acceptance criteria (incl. cross-user 404/404/404, B list total 0, 401 no-token, 422 INVALID_EV_MODEL, 400 validation, limit clamp 100)
- Migration: `schema_migrations` version 3; `ev_models` = 12 rows; re-seed `INSERT 0 0` (idempotent); duplicate `name_en` rejected by constraint
- `go build`/`go vet`/`go test ./...` → all green (operator host run + developer in-image run `ok internal/service 10.143s`)

### Follow-ups (non-blocking)
- **D1** — move `repository.CarInput` to a service/domain input type so the handler depends only on `service`
- **D2** — add a `code` to the shared `middleware.Auth` 401 envelope (pairs with TASK-0002 F1)
- Full Supabase `ev_models` import (12-model starter set shipped) — data/docs follow-up
- QA runbook: pre-warm `voltana-gomod`/`voltana-gocache` volumes for reliable containerized test reruns on the 2 vCPU dev host

---

## [Hotfix] — 2026-05-30 — Pin go.mod to Go 1.22 (toolchain conflict)

The earlier `go mod tidy` had bumped `go.mod`'s `go` directive to `1.25`, pulled in by the test-only transitive dependency `github.com/rogpeppe/go-internal@v1.15.0` (which declares `go >= 1.25`, reached via `pgx → kr/pretty`). This conflicted with the `golang:1.22-alpine` builder image in `voltana-api/Dockerfile` and would have broken `docker compose build`.

### Changed
- `voltana-api/go.mod` — `go` directive back to `1.22`; pinned `github.com/rogpeppe/go-internal` to `v1.12.0` (declares `go 1.20`)
- `voltana-api/go.sum` — regenerated via `GOTOOLCHAIN=local go mod tidy`
- `voltana-api/Dockerfile` — `EXPOSE 8080` → `EXPOSE 9090` (stale after the port hotfix)

### Evidence
- `GOTOOLCHAIN=local go build ./...` → exit 0
- `GOTOOLCHAIN=local go vet ./...` → exit 0
- `GOTOOLCHAIN=local go test ./internal/service/...` → `ok  voltana-api/internal/service  10.502s`

---

## [Tooling] — 2026-05-29 — Dashboard sync script

### Added
- `voltana-dashboard-sync.js` — Node.js script that reads all `.ai/workflows/TASK-*.md` files (including multi-task files like `TASK-0003-0008.md`), parses `**Status**:` values, and patches `voltana-dashboard.html` in-place: TASKS array statuses, hero stat card counts, and kanban column counts. Zero dependencies (Node stdlib only). Run with `node voltana-dashboard-sync.js` after every workflow status change.
- `CLAUDE.md` — Dashboard Sync Rule section updated to operational prose (script now exists)

### Notes
- Handles both single-task files (`TASK-0001.md`) and multi-task files (`TASK-0003-0008.md`) via `# TASK-XXXX` header scanning
- TESTING status counts alongside REVIEW in the dashboard's Review column
- First sync: corrected dashboard from stale initial state → DONE:1 READY:1 BACKLOG:6

---

## [Hotfix] — 2026-05-29 — Internal API port 8080 → 9090

Port 8080 was already bound on the host machine. Changed the internal Docker-network port used by the Go API. Nginx still listens on 80 externally; only the nginx→api upstream and the Go server's fallback default changed.

### Changed
- `docker-compose.yml` — `PORT: 9090`, health check URL updated to `:9090`
- `nginx/nginx.conf` — upstream `server api:9090`
- `voltana-api/cmd/server/main.go` — default fallback `port = "9090"`
- `.env.example` — added `PORT=9090`

---

## [TASK-0001] — 2026-05-29 — Docker Compose Stack Bootstrap

### Added
- `docker-compose.yml` — 5-service stack: postgres, redis, migrate, api, nginx with proper health checks and startup ordering
- `voltana-api/Dockerfile` — Go 1.22 multi-stage build (golang:1.22-alpine → alpine:3.19)
- `voltana-api/go.mod` — Go module, gin v1.9.1
- `voltana-api/cmd/server/main.go` — minimal Gin server, health endpoint only
- `voltana-api/internal/handler/health_handler.go` — `GET /health → {"status":"ok"}`
- `voltana-api/internal/{domain,repository,service,middleware}/` — empty package directories (ready for TASK-0002)
- `migrations/000001_init_schema.up.sql` — base schema: users, ev_models, cars, charging_sessions, user_settings + set_updated_at trigger
- `migrations/000001_init_schema.down.sql` — reverse migration
- `nginx/nginx.conf` — reverse proxy to api:8080
- `.env.example` — all required env vars, no real secrets
- `.gitignore` — excludes .env

### Fixed (dev_supervisor review pass)
- `voltana-api/go.sum` — generated via `go mod tidy` (86 entries); `go build ./...` passes cleanly
- `voltana-api/Dockerfile` — removed `GONOSUMDB=*` and `GOFLAGS=-mod=mod`; now uses `COPY go.mod go.sum` + `RUN go mod download` (reproducible, verified builds)
- `voltana-api/go.mod` — updated with all indirect deps explicitly listed (Go 1.22 lazy-loading standard)
- `docker-compose.yml` api service — removed `env_file: .env`; added `JWT_SECRET: ${JWT_SECRET}` and `APP_ENV: ${APP_ENV:-development}` to `environment:` block
- `migrations/000001_init_schema.down.sql` — `DROP FUNCTION IF EXISTS set_updated_at()` (added explicit parameter list)

### Notes
- Postgres volume persists at `postgres_data`; Redis AOF enabled via `--appendonly yes`
- `migrate` service (golang-migrate v4.17.0) runs `up` and exits; api depends on `service_completed_successfully`
- `user` must be in the `docker` group: `sudo usermod -aG docker $USER && newgrp docker`

---

## [Bootstrap] — 2026-05-29

### Added
- `.ai/` orchestration system: CLAUDE.md, PERSONA_ROUTER, context, workflows TASK-0001–0008, ADR-001–003
