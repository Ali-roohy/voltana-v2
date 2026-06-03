# Voltana — Project Context

> Update this file at the end of every session.

---

## Current State

- **Date**: 2026-06-03
- **Active Phase**: **Phase 3 — in progress** (Phases 1 & 2 complete)
- **Current Sprint**: Phase 3 — infra hardening ✅ (0014) + **TASK-0013 (Map + Real Station Data) ✅ CLOSED 2026-06-03** (Leaflet+OSM map, `/v1/stations` CRUD, `users.is_admin` + `AdminOnly`; full architect→dev_supervisor→security→qa chain green) + **TASK-0015 (GitHub governance) ✅ CLOSED 2026-06-03** (SemVer/`VERSION`, CI, templates, CODEOWNERS, SECURITY.md, CHANGELOG, labels/milestones; branch protection + `v0.3.0` tag applied at closure). First Phase-3 **feature** task done; release/governance baseline in place.

## Last Completed Task
- TASK-0015 — GitHub Repository Setup / governance (**DONE / CLOSED with caveat** by qa_supervisor,
  2026-06-03 — dev_supervisor ✅ (5/5) + qa ✅ (4/4); Phase-3 release/infra). SemVer (`VERSION`=0.3.0 + tags),
  `.github/` issue+PR templates + CODEOWNERS + **`ci.yml`** (Go build/vet/test + frontend tsc/build, push & PR
  to main, no deploy), `SECURITY.md`, promoted `changelog.md`→`CHANGELOG.md` (Keep a Changelog), labels +
  milestones (v0.3.0/v0.4.0/v1.0.0). CI first run GREEN (#26872969711 @ `2777c47`, both jobs success).
  **At-closure operator actions applied:** branch protection on `main` (both CI checks required, PR required,
  no force-push/delete) + annotated **`v0.3.0` tag pushed**. Closes the Phase-3 governance gap.
- TASK-0013 — Map + Real Station Data (**DONE / CLOSED** by qa_supervisor, 2026-06-03 — architect ✅ +
  dev_supervisor ✅ + security ✅ (admin boundary) + qa ✅ (9/9 live); **first Phase-3 feature task**).
  Replaced the iframe map with **keyless Leaflet + OpenStreetMap** rendering DB-backed station markers +
  click→detail; added **`/v1/stations`** (GET list w/ optional bbox filter + GET `:id` open to any authed
  user; POST/PUT/DELETE **admin-only**) behind a new **`users.is_admin`** role + **`AdminOnly` middleware**
  doing a **fresh DB check** (not in JWT → instant revocation; 403 before lookup, no enumeration; bootstrap is
  out-of-band SQL only). Migrations **000006** (is_admin) · **000007** (charging_stations: no user_id, DB
  lat/lng + power CHECKs + `set_updated_at` trigger) · **000008** (5 Tehran seed) — applied live, schema
  **v5→v8**. Frontend `features/stations/{api.ts,hooks.ts}` + react-leaflet (pinned **v4**; v5 needs React 19).
  **Fixed in smoke:** `latitude:0`/`longitude:0` rejection → lat/lng now `*float64`+`required`, bounds in the
  service. qa on a **clean `docker compose up -d --build api`** (in-container build 50.4s, no wedge): 5 markers,
  non-admin POST→403, admin POST→201 (equator), PUT 200, DELETE 204→404, bbox subset + partial→400, seed
  intact; host `go test` ok, `tsc` 0 + `npm build` ✓. **⚠️ Pending: dev_supervisor `git commit`+`push`** per the
  new DoD Git Commit Rule before the next task.
- TASK-0014 — Release & Infra Hardening (**DONE / CLOSED** by qa_supervisor, 2026-06-02 — release ✅ +
  dev_supervisor ✅ (5/5) + qa ✅ (5/5 acceptance, **zero manual deploy steps**) + qa_supervisor ✅; first
  Phase-3 task). Reproducible redeploy **`docker compose up -d --build api`** (Compose v2); nginx re-resolves
  the api via `resolver 127.0.0.11` + variable `proxy_pass $upstream$request_uri` (no reload on api restart);
  **MailHog** dev SMTP catcher (`:8025`, `SMTP_HOST=mailhog`/`1025`) — no more `is_email_verified` DB flip;
  `APP_URL`/`SMTP_*` flow through compose + `APP_ENV=production` note (N1); SOH lower floor
  `if soh < 0.01 { soh = 0.01 }` + test (0007 carry-forward). **Clears the deploy debt that trailed
  0009/0007/0008.** Accepted limits: in-container compose build reliable only on an unloaded host
  (`Dockerfile.runtime` fallback); nginx *config* changes still need a one-time `compose restart nginx`.
- TASK-0008 — Dashboard Analytics API + Battery Chart (**DONE / CLOSED** by qa_supervisor, 2026-06-02 —
  architect ✅ + dev_supervisor ✅ (5/5, history fix re-verified) + qa ✅ (5/5 live + isolation) +
  qa_supervisor ✅; **completes the Phase-2 analytics chain 0007→0008**). `GET /v1/analytics/dashboard`
  (`total_kwh/total_cost/total_km/avg_kwh_per_100km/session_count`; lifetime all-cars; Redis cache-aside key
  `analytics:dashboard:<userID>` TTL 5m, busted on charging write via the 0007 hook; `avg`=`null` when
  `total_km==0`) + `GET /v1/analytics/battery/:car_id/history` (newest-N, ASC, 404 cross-user) + frontend
  fleet cards + SOH card + Recharts SOH trend (multi-car selector). New `AggregateByUser`/`ListByCar`/cache
  helpers; no new migration (reuses 000005). **Review caught + fixed** a history-window bug (was oldest-N →
  newest-N reversed to ASC). Host `go test` ok; live smoke green (dashboard 210/4200/15000/1.4/7, SOH 88%,
  history chronological, cache 210→240 on write). Clean redeploy (orphan reaped, nginx reloaded).
- TASK-0007 — Battery Health Snapshots (**DONE / CLOSED with caveat** by qa_supervisor, 2026-06-02 —
  architect ✅ + dev_supervisor ✅ (5/5) + qa ✅ (6/6 live smoke) + qa_supervisor ✅; Phase-2 analytics
  foundation). delta-SOC SOH estimate (η=0.88 charging-efficiency, Δsoc≥25 qualifying filter, Δsoc-weighted,
  clamp (0,100], min-5 qualifying → else insufficient-data) behind `analytics_service`; **no asynq** —
  synchronous per-car *coalescing* recompute on charging-session create/update/delete; `GET
  /v1/analytics/battery/:car_id` + `/recommendations/:car_id` (LFP→100 / NMC·NCA→80 / null→generic);
  migration **000005** `battery_health_snapshots` (history table). user_id isolation → 404. Host `go test`
  ok (10 new analytics fns). Live smoke green: **SOH 88%** (52.8/60 kWh, medium), LFP advice, insufficient
  `200 {qualifying:2}`, unknown car 404. **⚠️ Caveats (operator-accepted):** (1) **SOH lower-bound floor** —
  sub-0.001-kWh session could round `soh_pct`→0.00 and trip DB `CHECK (>0)` on Save (not reproducible w/
  real data) → 1-line follow-up; (2) **release follow-up** — recurring stale-redeploy + **nginx upstream-IP
  cache** (fixed live via `nginx -s reload`) + orphan `voltana-api-new` container; want reproducible
  compose-v2 redeploy + MailHog. qa redeployed the api itself (running binary was stale task0009).
- TASK-0009 — Email Verification Gate (**DONE / CLOSED with caveat** by qa_supervisor, 2026-06-02 —
  dev_supervisor ✅ (6/6) + security ✅ (5/5 controls) + qa ✅ (5/5 live smoke) + qa_supervisor ✅; **first
  Phase-2 backend task**, carried bug **#7** verify/resend UI). Login 403 `EMAIL_NOT_VERIFIED` (only after a
  passing password check — wrong pw still 401, no enumeration); `/auth/verify-email` + `/auth/resend-verification`
  (rate-limited: verify 20/15m, resend 5/h IP + 3/h email + 60s cooldown; resend always 202 anti-enum);
  SHA-256-hash-only single-use 24h tokens behind a `service.Mailer` interface (SMTP + dev log mailer);
  register no longer auto-logs-in → "check email" screen; `/verify-email` page. No new migration (`000002`
  table fit). Host `go test ./...` ok (uncached 16.6s); **qa hand-redeployed the api** (host-compile +
  `Dockerfile.runtime`) — running container was stale. **⚠️ Caveats (operator-accepted):** (1) verify→login
  E2E unit-covered only (no dev SMTP catcher to capture the raw token) — retire with a MailHog smoke;
  (2) **release follow-up** — `docker-compose.yml` `api` lacks `APP_URL`/`SMTP_*` + still builds the
  wedge-prone in-container `Dockerfile` (deploy not reproducible without the manual swap). **Closes the
  long-deferred N2/bug-#7.**
- TASK-0012 — Session History Filters + Detail View (**DONE / CLOSED with caveat** by qa_supervisor, 2026-06-01 — feature ✅ + dev_supervisor ✅ (5/5) + qa ✅ (API-verified) + qa_supervisor ✅; frontend-only, no API/DB change. Server-side date-range filter (`?from`/`?to`, **inclusive end-of-day**) + filter-aware TanStack key with `keepPreviousData`; car filter only for multi-car users; tap-to-expand accordion (TOUBreakdown + SOCAnalysis + location + **notes** + start time/duration). `tsc` 0 · build ✓ · preview 200. **⚠️ Caveat (operator-accepted):** Playwright CDN geo-blocked + no system browser → date-filter + inclusive-end-of-day **proven via browser-equivalent curl**, but UI scenarios **expand-detail & clear-filters code-/data-verified only**, not clicked. Retire with a UI smoke when a browser is obtainable. **Completes the Phase-2 UX trio.**)
- TASK-0011 — Monthly Cost Trend Chart (**DONE / CLOSED** by qa_supervisor, 2026-06-01 — feature ✅ + dev_supervisor ✅ (5/5) + qa ✅ + qa_supervisor ✅; frontend-only, no API/DB change. Dashboard: shared `trend` (energy+cost) reusing `lib/cost.ts`; new Monthly Cost bar chart beside the energy line; repurposed the dead avg-efficiency card → **avg cost/session** (null-safe); SOC chart moved to its own row. Two single-unit charts (not dual-axis); Toman, no ÷10. `tsc` 0 · build ✓ · preview 200; operator approved skipping the full browser test. Carried: "Sessions" card still unscoped `sessions.length` (optional).)
- TASK-0010 — TOU Cost Breakdown Card (**DONE / CLOSED** by qa_supervisor, 2026-06-01 — feature ✅ + dev_supervisor ✅ (5/5) + qa ✅ (+ re-check after `$`/RTL browser fixes) + qa_supervisor ✅; frontend-only, no API/DB change. Added shared `lib/cost.ts` (`calcCost`/`ratesFromSettings`) + presentational `TOUBreakdown` stacked bar; per-session inline + dashboard "This month" summary; fixed dashboard `totalCost` undercount; currency = **Toman, no ÷10** (`ریال→تومان`). `tsc` 0 · build ✓ · operator browser-confirmed (formatting, `$` removed, RTL). **First Phase-2 task done; `lib/cost.ts` now reused by TASK-0011.**)
- TASK-0006 — Frontend: Replace Supabase SDK with Go API (**DONE / CLOSED** by qa_supervisor, 2026-06-01 — dev_supervisor ✅ (6/6, incl. re-review) + security ✅ (ADR-003 token storage) + qa ✅ + qa_supervisor ✅; React MVP refactored off Supabase onto the Go API, feature-based data layer, in-memory JWT + silent refresh, 7/8 bugs fixed. `npm run build` ✓ · `tsc --noEmit` 0 · preview :4173 200 · operator manual browser test green (register/login, default-car pre-select, required-field validation, cost calc, no Supabase console errors). **#7 email gate deferred → TASK-0009.** **This was the last open Phase-1 task → Phase 1 COMPLETE.**)
- TASK-0005 — User Settings API (**DONE / CLOSED** by qa_supervisor, 2026-05-31 — dev_supervisor ✅ + security ✅ + qa ✅ + qa_supervisor ✅; `GET/PUT /v1/settings`, auto-create-on-first-GET, extended `settings_repo` GetOrCreate/Update; no migration. Host `go test` ok, schema v4, live smoke incl. per-user isolation + 422 unowned default car. **Closed TASK-0004's settings_repo carry-forward.** Backend API surface for Phase 1 now complete.)
- TASK-0004 — Charging Sessions CRUD API (**DONE / CLOSED** by qa_supervisor, 2026-05-31 — dev_supervisor ✅ + security ✅ + qa ✅ + qa_supervisor ✅; host `go test` ok, migration v4, live smoke green incl. computed cost 54 / override 123.45 / 422 invalid car / cross-user 404. **D1 applied** (input in `domain`) + **D2 fixed** (401 `code:"UNAUTHORIZED"`). Carry-forwards: `Dockerfile.runtime` dev-only→release.)
- TASK-0003 — Cars & EV Models CRUD API (**DONE / CLOSED** by qa_supervisor, 2026-05-31 — dev_supervisor ✅ + security ✅ + qa ✅ + qa_supervisor ✅; live smoke 9/9, migration v3 + idempotency green, `go test ./...` ok via operator host run + developer in-image run. Carry-forwards: D1 `repository.CarInput` coupling, D2 401 envelope `code`, full Supabase ev_models import, QA-runbook Go cache-volume pre-warm.)
- TASK-0002 — Go Auth API (**DONE / CLOSED** by qa_supervisor, 2026-05-30 — security ✅ + dev_supervisor ✅ + qa ✅ + qa_supervisor ✅; full FAIL→fix→PASS chain traceable in the task file; `go test ./...` exit 0, live flow green, 10/10 criteria)
- TASK-0001 — Docker Compose stack bootstrap (DONE, 2026-05-30)

## Active Tasks

| Task | Persona | Status |
|---|---|---|
| TASK-0001 | developer | DONE |
| TASK-0002 | developer | DONE ✅ CLOSED (qa_supervisor signed off) |
| TASK-0003 | developer | DONE ✅ CLOSED (qa_supervisor signed off 2026-05-31) |
| TASK-0004 | developer | DONE ✅ CLOSED (qa_supervisor signed off 2026-05-31) |
| TASK-0005 | developer | DONE ✅ CLOSED (qa_supervisor signed off 2026-05-31) |
| TASK-0006 | developer | DONE ✅ CLOSED (qa_supervisor signed off 2026-06-01) — **closes Phase 1** |
| TASK-0009 | developer | DONE ✅ CLOSED with caveat (qa_supervisor signed off 2026-06-02) — **first Phase-2 backend task; email gate + bug #7** |
| TASK-0010 | feature → developer | DONE ✅ CLOSED (qa_supervisor signed off 2026-06-01) — **first Phase-2 task** |
| TASK-0011 | feature → developer | DONE ✅ CLOSED (qa_supervisor signed off 2026-06-01) — **second Phase-2 task** |
| TASK-0012 | feature → developer | DONE ✅ CLOSED with caveat (qa_supervisor signed off 2026-06-01) — **third Phase-2 task** |
| TASK-0007 | developer | DONE ✅ CLOSED with caveat (qa_supervisor signed off 2026-06-02) — **Phase-2 analytics foundation; battery SOH + recommendations** |
| TASK-0008 | developer | DONE ✅ CLOSED (qa_supervisor signed off 2026-06-02) — **completes Phase-2 analytics chain (0007→0008)** |
| TASK-0014 | release (+ developer) | DONE ✅ CLOSED (qa_supervisor signed off 2026-06-02) — **reproducible compose redeploy + nginx re-resolve + MailHog + SOH floor; clears the 0009/0007/0008 deploy debt** |
| TASK-0015 | release (+ developer) | **DONE ✅ CLOSED with caveat** (qa_supervisor signed off 2026-06-03) — **Phase-3 release/infra governance**. dev_supervisor ✅ (5/5) + qa ✅ (4/4 — CI green, labels, milestones, governance files). Created `VERSION`=0.3.0, `.github/` (issue+PR templates, CODEOWNERS, **`ci.yml`**), `SECURITY.md`, promoted `changelog.md`→`CHANGELOG.md`; labels + milestones (v0.3.0/v0.4.0/v1.0.0). CI first run GREEN @ `2777c47`. **At-closure operator actions applied:** branch protection on `main` + **`v0.3.0` tag pushed**. |
| TASK-0013 | developer (git commit) | **DONE ✅ CLOSED** (qa_supervisor signed off 2026-06-03) — **first Phase-3 feature task**; Leaflet+OSM map [keyless] + `/v1/stations` CRUD + `users.is_admin`/`AdminOnly`. architect ✅ + dev_supervisor ✅ + security ✅ + qa ✅ (9/9 live). **Pending: dev_supervisor git commit+push per new DoD Git Commit Rule.** |

## Current Focus
- **🎉 Phase 1 — Solid Foundation: COMPLETE (2026-06-01).** All Phase-1 tasks closed by qa_supervisor:
  TASK-0001 (compose stack) · 0002 (auth) · 0003 (cars/ev-models) · 0004 (charging) · 0005 (settings) ·
  **0006 (frontend off Supabase → Go API)**. Deliverable: a fully self-hosted Go + Postgres backend
  (auth · cars · ev-models · charging · settings on the `/v1` JWT group) **and** the React frontend
  refactored onto it (feature-based data layer, in-memory JWT + httpOnly refresh, sonner, 7/8 bugs fixed).
- **Phase 2 in progress (sequence: TASK-0010 → 0011 → 0012 → 0009 → 0007 → 0008; see PM Decision below).**
  **UX trio done:** **0010 ✅** (`lib/cost.ts` + `TOUBreakdown`) · **0011 ✅** (monthly cost trend +
  avg-cost/session) · **0012 ✅** (history date-range filter + tap-to-expand detail, *browser caveat*).
  **Email gate done:** **0009 ✅ CLOSED with caveat (2026-06-02)** — login 403 gate + verify/resend endpoints
  (rate-limited, anti-enum) + SHA-256 single-use tokens behind a `Mailer` interface + bug #7 UI.
  **🎉 Analytics chain done (0007→0008):** **0007 ✅ CLOSED with caveat** — delta-SOC SOH (η=0.88, Δsoc≥25,
  weighted, clamp, min-5) + chemistry recommendations + migration 000005, **synchronous coalescing recompute
  (no asynq)**. **0008 ✅ CLOSED** — `GET /v1/analytics/dashboard` (lifetime totals + avg kWh/100km, Redis
  cache-aside busted on write) + `/battery/:car_id/history` (newest-N, ASC) + Recharts SOH trend & fleet cards;
  review caught/fixed a history-window bug. Full architect/dev_supervisor/qa chain green (live SOH 88%,
  dashboard 210/4200/15000/1.4/7, cache 210→240 on write).
- **🎉 Phase 2 — Intelligence Layer COMPLETE.** All Phase-2 tasks (0009 · 0010 · 0011 · 0012 · 0007 · 0008) CLOSED.
- **➡️ Phase 3 kicked off (PM, 2026-06-02).** Two specs created:
  - **TASK-0014 — Release & Infra Hardening ✅ CLOSED (2026-06-02)** — reproducible `docker compose up -d
    --build api`, nginx re-resolve (no reload on api restart), MailHog dev SMTP, compose `APP_URL`/`SMTP_*`/
    `APP_ENV`, SOH floor. **The 0009/0007/0008 hand-deploy tax is gone** — qa verified with zero manual steps.
  - **➡️ TASK-0013 — Map + Real Station Data (READY, architect contract FINALIZED 2026-06-03)** — replace the
    iframe map; keyless **Leaflet + OpenStreetMap** + **`users.is_admin`** admin CRUD. **Architect pass done**
    — see the `# Architect Contract — FINALIZED` section in `TASK-0013.md`. Pinned: migrations renumbered to
    **000006** (`users.is_admin`) · **000007** (`charging_stations`, with DB lat/lng CHECKs + `set_updated_at`
    trigger) · **000008** (demo seed so the map renders pre-admin); **`AdminOnly` middleware does a fresh DB
    `IsAdmin` check** (not baked in the 15-min JWT → immediate revocation); `/v1/stations` GET open to any
    authed user, POST/PUT/DELETE admin-only **403 before lookup** (no enumeration); marker-vs-detail field
    split; full handler→service→repo slice mirroring the cars module; admin bootstrap is **out-of-band SQL**
    (`UPDATE users SET is_admin=true …`). `leaflet`+`react-leaflet` already in `package.json`.
    **Next: developer implements → dev_supervisor → security (admin boundary) → qa.**
  - Phase-3 **OBD/ELM327** (original roadmap) remains unscoped — a later researcher/pm pass.
- **Release/infra follow-ups (track, non-blocking):**
  - **(0009)** `docker-compose.yml` `api` must pass `APP_URL` + `SMTP_*` and move off the wedge-prone
    in-container `Dockerfile` (host-binary + `Dockerfile.runtime`) so a clean `compose up` works.
  - **(0007, recurring)** stale-redeploy pattern + **nginx upstream-IP cache** (nginx caches `api`'s IP at
    startup; after an api container swap it routes to a stale instance — fixed live via `nginx -s reload`) +
    orphan `voltana-api-new` container to reap. Want a reproducible compose-v2 redeploy path (nginx
    `resolver`/variable `proxy_pass`) + a dev SMTP catcher (MailHog) to drop the manual `is_email_verified`
    flip in smokes.
  - **(0007)** SOH `soh_pct` lower-bound floor (1-line guard or relax DB CHECK to `>= 0`) → developer backlog.
- **Tooling note (2026-06-01):** no headless browser on this host — **Playwright's CDN is geo-blocked**
  (`403 … not available in your location`) and no system Chromium/Chrome. UI verification has relied on
  build/tsc + browser-equivalent curl + operator manual checks. Flag for release/infra: provide a browser
  (system Chromium or an unblocked Playwright mirror) to enable real UI smoke tests.
- **Phase-1 carry-forwards to track (non-blocking):** **TASK-0009** (email gate incl. bug #7); **N1** —
  set `APP_ENV=production` on the VPS so the refresh cookie gets `Secure`; **deployment invariant** —
  `VITE_API_URL` must stay same-origin (nginx); optional strict ADR-002 `features/<name>/Page.tsx`
  relocation (pages still in `src/pages/`) + delete orphaned radix toast files.

### PM Decision (2026-06-01) — Phase 2 sequencing / kickoff
**Phase 1 closed; Phase 2 ordered.** Locked start sequence (each predecessor closes before the next
is marked READY):

**TASK-0010 → TASK-0011 → TASK-0012 → TASK-0009 → TASK-0007 → TASK-0008**

| # | Task | Why here |
|---|------|----------|
| 1 | **TASK-0010** — TOU cost breakdown card | **READY.** UI win, **zero backend**, highest impact/effort ratio; introduces the **shared cost helper** that 0011 reuses → must lead. |
| 2 | **TASK-0011** — Monthly cost trend chart | UI win, no backend. **Hard dep on 0010** (shared cost helper + currency unit) → directly after. |
| 3 | **TASK-0012** — History filters + detail view | UI win, no backend (wires existing `?from/?to`). Optional reuse of 0010's breakdown in the detail view. |
| 4 | **TASK-0009** — Email verification gate | First **backend** task of the phase; carries bug **#7** UI (Phase-1 carry-forward). Sequenced after the quick UI wins so users see value sooner, but before the heavier analytics engine. |
| 5 | **TASK-0007** — Battery health snapshots | Analytics engine (migration + `asynq` job + endpoints). Larger backend lift; foundation for 0008. |
| 6 | **TASK-0008** — Dashboard analytics API + chart | **Hard dep on 0007** (consumes its health data) → last. |

**Rationale:** front-load the three **no-backend UI wins** (0010–0012) to ship visible value fast on the
now-complete frontend, then the **auth-hardening** gate (0009), then the **analytics engine** (0007→0008)
whose dependency chain (0008 needs 0007) fixes their relative order. All six deps are satisfied
(0010/0012→0006 DONE, 0011→0010, 0009→0002 DONE, 0007→0004 DONE, 0008→0007). **Persona note:** 0010–0012
are `feature → developer`; 0009/0007/0008 are developer-led backend (0009 also needs security review).

### PM Decision (2026-06-01) — Phase 2 specs from researcher report
Created specs for the researcher's **top-3** proposals (all derive from data the Phase-1 API
already returns — **no backend/DB/migration work** in any of the three):
- **TASK-0010 — TOU cost breakdown card** (High impact / Low effort): stacked peak/mid/off-peak
  kWh + cost on the dashboard *and* per-session card. Introduces a **shared cost helper** that
  `getSessionCost` (currently inline in `pages/Charging.tsx`) refactors onto.
- **TASK-0011 — Monthly cost trend chart** (High impact / Low effort): adds a monthly **cost**
  series beside the existing energy trend in `pages/Index.tsx`, plus total spend + avg
  cost/session. **Sequence after 0010** to share the cost helper + currency unit.
- **TASK-0012 — Session history filters + detail view** (Med impact / Low effort): date-range
  filter wired to the existing `?from`/`?to` API params (frontend `api.ts`/`hooks.ts` must
  start passing them + key the query on the filter) + tap-to-expand detail card.
- **Cross-cutting open question flagged in 0010/0011:** currency unit — existing Charging page
  shows **ریال/Rial** via `formatCost`; proposals said "Toman". Recommendation: keep Rial
  app-wide; treat a Rial→Toman switch as a separate decision. Do **not** mix units across cards.
- **Persona note:** all three are frontend → routed `feature → developer` (UI/state/hook design
  hands off before developer implements), reviewer `dev_supervisor`. They build on TASK-0006
  (frontend baseline, currently TESTING) so they unblock once 0006 closes.

### PM Decision (2026-05-30) — next-task planning
1. **Next READY task → TASK-0003 (Cars & EV Models CRUD API).** Critical-path; dep TASK-0002
   satisfied. (TASK-0009 also unblocked but sequenced later — see #3.)
2. **Blockers before TASK-0003 can start: NONE remaining.** Both READY prerequisites are now
   DONE (architect, 2026-05-30):
   - (a) ✅ Split into its own `.ai/workflows/TASK-0003.md` (bundled section stubbed out).
   - (b) ✅ API contract added: `/v1/cars` CRUD + `/v1/ev-models` search shapes, validation,
     pagination (`{items,limit,offset,total}`, limit≤100), error envelope, and the
     **user_id-from-JWT isolation enforced in the repository layer** (cross-user → 404).
   - **Architect scope correction:** the `cars` + `ev_models` tables ALREADY exist in
     `000001_init_schema.up.sql` — TASK-0003 adds Go layers + `/v1` routes + an `ev_models`
     **seed** migration (`000003`, with a `name_en` unique constraint for idempotency), NOT
     new tables. TASK-0003 is now fully workable by the developer.
   - Non-blocking ops items (Docker Compose v2, node) do not affect TASK-0003 development.
3. **TASK-0009 (Email Verification Gate) → AFTER TASK-0003**, scheduled late in Phase 1 just
   before TASK-0006. Rationale: the CRUD chain (0003 → 0004 → 0005) is the product critical
   path; email verification is auth-hardening whose verify/resend UX lands naturally with the
   frontend task (0006); gating login now would add friction to building/testing the CRUD
   endpoints. **Phase-1 order: 0003 → 0004 → 0005 → 0009 → 0006.**

- Carry-forwards from TASK-0002 close (non-blocking):
  - **N1** — set `APP_ENV=production` on VPS so refresh cookie gets `Secure` (dev runs `development`).
  - **N2** — tracked as TASK-0009.
  - **F1/F2/F3** — optional dev recs: translate `repository.ErrEmailTaken`→`service.ErrEmailTaken`; generic bind-error message; single source for 30d refresh TTL.
  - **S2 deployment invariant** — nginx must remain sole ingress and always set `X-Real-IP`.

## Blockers / Ops Notes
- (RESOLVED 2026-05-30) WSL `docker.service` had failed mid-session; daemon restarted, TASK-0003 verification completed, migration 000003 applied, api redeployed.
- (RESOLVED 2026-05-31) **node** now available → `voltana-dashboard-sync.js` runs; dashboard reconciled (DONE:3 incl. TASK-0003).
- (2026-05-31) **QA Go-test runbook:** dev host has no local Go toolchain and the 2 vCPU / 4 GB host starves cold `golang:1.22-alpine` compiles when co-located stacks run. For containerized test reruns, pre-warm cache volumes (`-v voltana-gomod:/go/pkg/mod -v voltana-gocache:/root/.cache/go-build`); operator can also run host Go directly.
- (2026-05-31) During TASK-0003 QA, the **unrelated stacks `synapse`, `element`, `nextcloud_{app,redis,db}_1` were stopped** to free resources — restart when needed: `docker start synapse element nextcloud_db_1 nextcloud_redis_1 nextcloud_app_1`.
- Dev host **docker-compose v1.29.2** + Docker Engine 29 → `up` of a *rebuilt* image fails (`KeyError: 'ContainerConfig'`). Worked around with `docker run` on the compose network. Install **Docker Compose v2 plugin** on dev + VPS (flag to release).
- (2026-06-02) **TASK-0014 resolved the redeploy friction (DONE / CLOSED).** Reproducible redeploy runbook:
  **`docker compose up -d --build api`** (Compose v2 plugin `v5.1.4` is present — use `docker compose`, NOT
  `docker-compose` v1). nginx now re-resolves the api via a `resolver` + variable `proxy_pass`, so **api
  redeploys no longer need `nginx -s reload`** (only an nginx *config* change does: `docker compose restart
  nginx`). Dev email: **MailHog** at `http://localhost:8025` (`SMTP_HOST=mailhog`/`SMTP_PORT=1025` in `.env`) —
  no more manual `is_email_verified` DB flip to read a verify link. The hand-deploy
  (host-compile + `Dockerfile.runtime` swap) is now only a fallback for a loaded host.

## Key Decisions Made
- Backend: Go (Gin) instead of .NET — better OBD serial port support, lower VPS footprint
- Auth: Self-managed JWT (access token in memory, refresh in httpOnly cookie) — replaces Supabase auth
- DB: PostgreSQL 16 self-hosted — replaces Supabase Postgres
- Frontend: Keep existing React codebase, refactor to feature-based structure
- Mobile: Capacitor wraps React PWA — no separate native codebase

## Open Questions
- Neshan map API key — obtain before Phase 2 map work
- OBD ELM327 BLE vs USB — decide in Phase 3 planning
- Email provider for verification emails (Phase 1) — SMTP or service?

## Environment
- Dev machine: WSL2 / Linux
- Target server: Ubuntu VPS, 2 vCPU / 4 GB RAM
- Docker Compose for all services
