# Voltana — Project Context

> Update this file at the end of every session.

---

## Current State

- **Date**: 2026-06-12
- **Active Phase**: Phase 5 — TASK-0001–0039 all DONE ✅. **TASK-0041 Part A DONE ✅** (2026-06-13 — dev_supervisor ✅ + security ✅ + qa ✅ + qa_supervisor ✅ + PM ✅): added `voltana-api/cmd/genvapid/main.go` (`webpush.GenerateVAPIDKeys()` → prints paste-ready `VAPID_PUBLIC_KEY=`/`VAPID_PRIVATE_KEY=` to stdout only, never committed; verified 87/43-char keys); `go mod tidy` promoted `webpush-go` indirect→direct (go.sum unchanged); fixed the stale `go run ./cmd/genvapid` reference across `.env.example`/`.env.production.example`/`DEPLOY_PRODUCTION.md` (it 404'd before, now real). `go build`/`vet`/`gofmt`/`test` all green. **Parts B/C operator-gated**: Bale (primary)+optional Telegram bot tokens from **BotFather** (rotate if pre-0038 log-exposed); operator sets all six vars in VPS `/opt/voltana/.env` + verifies push-enabled log line, `/v1/push/vapid-key`, live Bale OTP — same operator-run pattern as the TASK-0040 deploy/smoke. No secret values committed. **TASK-0040 DONE ✅ CLOSED** (2026-06-13 — dev_supervisor ✅ + security ✅ + qa ✅ + qa_supervisor ✅ + PM ✅): Production Deployment — domain **voltanaev.ir** + **Poste.io** self-hosted mail (replaces the Resend suggestion; MailHog stays dev-only). Configs + docs only, **no Go/React source changed**. Compose `poste` service (`analogic/poste.io`, vol `/var/lib/voltana/mail`): public 25/465/587/993, **127.0.0.1-bound** admin 8443 + plaintext 110/143/995 (Docker bypasses UFW → localhost binds are the real guard), `DISABLE_CLAMAV=TRUE` for the 4 GB host, TLS reuses the certbot cert (mail SAN) — no port-80 fight. nginx.prod.conf gains `www→apex` 301 + `server_name voltanaev.ir www.voltanaev.ir` (envsubst still only `${DOMAIN}`). `.env.production.example`: Poste SMTP block + VAPID **prod** pair placeholders + DOMAIN. bootstrap UFW 25/465/587/993 + mail dir + 3-SAN certbot hint. DEPLOY_PRODUCTION.md: DNS table + PTR, Poste setup (SSH-tunnel admin, mailbox, cert-reuse TLS, DKIM→DNS), deliverability checklist (mail-tester ≥8/10), production smoke plan. **No-code findings documented**: no CORS middleware (same-origin; VITE_API_URL stays unset), manifest already domain-agnostic, deep links username-based. Verified: `compose config` exit 0 (localhost binds confirmed) · `nginx -t` test successful · `go test ✓` · `tsc 0` · `build ✓`. **Iranian VPS = Bale reachable → contact_share OTP finally testable; the standing no-browser caveat (since 0033) retires during the operator's production smoke.** Actual deploy is operator-run. **TASK-0036 (Bug Batch, 8 bugs) + TASK-0037 (Feature Batch, 6 features) created READY** from the operator field-testing report (2026-06-12). Operator labeled them 0035/0036 but those collided with the existing TASK-0035 → renumbered.
- **Current Sprint**: **TASK-0039 Web Push Notifications DONE ✅ CLOSED** (2026-06-12) — migration 000019 push_subscriptions; webpush-go + VAPID env keys; SSRF-guarded subscribe (https-only, no private hosts); SendToUser with 410-prune (delivery unit-proven against httptest: aes128gcm + vapid auth header); SOH 80%-cross alert via SetSOHAlertNotifier; sw.js push/notificationclick; Settings notifications card + admin test-push. Real-device display = operator caveat. Remaining Phase-5 candidates: Capacitor, catalog photos, OBD, i18n.
- (prior) **TASK-0036 + TASK-0037 both DONE ✅ CLOSED** (2026-06-12). TASK-0037 (6 features): FEAT-1 OTP-register name · FEAT-2 real PWA installability (192/512 icons were missing; SW now network-first — old SW pinned stale deploys) · FEAT-3 background designer (`voltana:bg` layer, composes with presets+dynamic themes) · FEAT-4 export/import (tx replace, fresh ids, 5 MB cap; snapshots carried) · FEAT-5 self-delete (last-admin guard) · FEAT-6 **rate snapshots** (migration 000018: `rate_*_at_time` + backfill; admin default rates in system_settings; registration seeds user_settings; edits re-price with FROZEN rates; frontend `ratesForSession`). Operator money scenario passed live (50000/90000/140000 → frozen-edit 100000/190000). qa found+fixed a `SetDefaultRates` pgx cast 500. **Security follow-up RESOLVED → TASK-0038 DONE ✅** (2026-06-12, commit 64aee3a): `internal/bot/sanitize.go` masks /bot<token> in all poller/sender/probe errors; 5 regression tests; live-verified bot*** + zero-token log grep. Operator action still recommended: rotate both bot tokens (pre-fix logs exposed the Bale token).
- (prior) **TASK-0036 DONE ✅ CLOSED** (2026-06-12) — all 8 field-report bugs qa-repro'd → fixed → re-verified live (BUG-3 root cause: TanStack `["me"]` cache survived identity changes → new `lib/query-client.ts` clears on JWT-sub change; BUG-5: new `awaiting_bot` status + 15-min `otp:dlpending` marker fixes the 3-second deep-link kick-out; BUG-8: `/v1/admin/test-bot-connection` getMe proven live against the real Bale/Telegram bots). Next: TASK-0037 (FEAT-6 rate-snapshot last). Operator's Android-intent Auth.tsx WIP adopted + committed with BUG-5 (note: hardcodes package `com.bale.app` — verify on-device).

## Post-close Commits (additional fixes/features committed after task close)
- **(2026-06-13) Deploy-doc fixes (TASK-0040/0041 follow-up)** — operator dry-run of the runbook surfaced issues. `DEPLOY_PRODUCTION.md` gained a canonical **"Quick Runbook (copy-paste)"** (14 steps, correct **DNS→certbot→deploy** ordering). Fixes: VAPID generator is now **`npx web-push generate-vapid-keys`** (VPS has Node not Go; `go run ./cmd/genvapid` kept as a dev-host note) across the doc + `.env.production.example`; certbot-before-deploy made explicit (nginx dies on boot without the cert); Poste first-run wizard + DKIM→DNS spelled out; `.env` filename warning (**NOT** `.env.production` — deploy.sh/compose read `.env`); APP_URL/APP_ENV/SMTP_PASSWORD reaffirmed. Stale `docs/DEPLOY.md` (pre-0040: Resend, bootstrap-vps.sh, no Poste/VAPID) got an **OUTDATED/superseded banner** → points to DEPLOY_PRODUCTION.md.

## Last Completed Tasks
- TASK-0032 — VPS Setup & Deployment Guide + Local WSL Testing (**DONE ✅ CLOSED** 2026-06-11 — bundled with 0029/0031 in commit `a4cde5d`). Part A: local WSL smoke test runbook; Part B: VPS provisioning and HTTPS deployment guide using the TASK-0031 scripts. End-to-end operator guide from bare Ubuntu 24.04 → live HTTPS app.
- TASK-0031 — VPS Production Deployment Phase 4 (**DONE ✅ CLOSED** 2026-06-11 — commit `a4cde5d`). Automated pg_dump backups, bind-mount postgres volume at `/var/lib/voltana/postgres`, Ubuntu 24.04 bootstrap update, `.env.prod.example` clean prod template, health check script, `DEPLOY_PRODUCTION.md`. dev_supervisor ✅ + security ✅ + qa ✅.
- TASK-0029 — Hybrid Auth Flow + Admin OTP Configuration (**DONE ✅ CLOSED** 2026-06-11 — commit `a4cde5d` + prior commits). `POST /auth/login/phone` (phone+password, `ErrNoPasswordSet`→400, constant-time bcrypt); `stay_logged_in` on all login paths (session vs 30-day cookie); `POST /v1/account/set-password` (authed, idempotent); `password_set` in `/v1/me`; migration 000013 `system_settings`; `GET/PUT /v1/admin/system-settings` (AdminOnly); `GET /auth/otp/config` (public); deep-link OTP mode (`requestOTPDeeplink`, `HandleDeepLinkOTP`); bot poller `/start phone_<E164>`. Frontend: phone+password login picker, stay-logged-in checkboxes on all paths, `SetPasswordStep` after OTP registration, deep-link button, `AdminSystemSettings` card in Settings. `go test ✓` (25 tests) · `tsc 0` · `npm build ✓`. ⚠️ UI not clicked (no browser).

## Post-close Commits (additional fixes/features committed after task close)
- `6dc8973 fix: login with unregistered phone no longer starts OTP countdown` — OTP request in login mode with unknown phone now returns `{status:"not_registered"}` instead of silently starting the timer. **⚠️ This commit is unpushed.**
- `07a5687 feat: user management — show name, email, phone as separate columns` — Admin Users table shows full_name/email/phone in separate columns instead of combined.
- `dd06c30 feat: OTP messages sent with protect_content` — Telegram/Bale OTP messages use `protect_content=true` (no forward/save).
- `c0297c3 fix: OTP 'wrong code' on registration` — mode param prevents login/register Redis key collision.
- `1664fd0 fix: normalizePhone — accept multi-digit country codes` — UAE +971, etc. now normalize correctly.
- `431f225 feat: fix OTP timing in contact_share` — poll-based pending OTP flow replaces the incorrect timer-start-on-202.
- `a72b592 fix: OTP register 500→409 on duplicate email` — contact_share UX + resend button fixes.

- TASK-0025 — VPS Production Deployment (**DONE ✅ CLOSED with caveat** by qa_supervisor, 2026-06-04 — dev_supervisor ✅ + security ✅ + qa ✅ 12/12 code-verified). `scripts/bootstrap-vps.sh` (idempotent: Docker, Node 20, certbot, UFW 22/80/443, voltana user) · `scripts/deploy.sh` (git pull → npm build → envsubst nginx → migrate → rebuild api → nginx reload; `set -euo pipefail`) · `infra/nginx/nginx.prod.conf` (HTTP→HTTPS, TLS 1.2+1.3 Mozilla Intermediate, 6 security headers, `/auth/` rate-limit, SPA fallback) · `infra/systemd/voltana.service` (oneshot+RemainAfterExit, Restart=on-failure) · `docker-compose.prod.yml` (port 443, cert mounts, mailhog dev-profile gate) · `docs/DEPLOY.md` complete guide. **Bonus (same commit):** `nginx/nginx.conf` updated to serve `voltana-web/dist` as SPA (no more Vite preview needed); `docker-compose.yml` mounts the dist. Live smoke: `GET /` → SPA HTML ✓ · `/health` → `{"status":"ok"}` ✓ · `/v1/cars` → 401 ✓. ⚠️ Caveat: live VPS acceptance requires operator-provisioned domain + server.

- TASK-0017 — OTP Login via Bale/Telegram Bot (**DONE / CLOSED with caveat** by qa_supervisor,
  2026-06-03 — dev_supervisor ✅ (6/6) + security ✅ + qa ✅ (13/13 live+code) + qa_supervisor ✅).
  Passwordless OTP login via Bale/Telegram alongside existing email/password auth. **Slice A (linking):**
  migration **000010** (phone E.164, bale_chat_id, telegram_chat_id, 3 partial-unique indexes);
  `POST /v1/account/bot-link` (JWT, mints `botlink:<token>` Redis, returns deep links); in-process
  `bot.Poller` long-poll goroutine (outbound HTTPS only, handles `/start <token>` + contact-share,
  writes to `UpdateBotLink`); `bot.LinkCallback` interface keeps service/bot layers decoupled.
  **Slice B (OTP):** `OTPSender` interface (BaleSender / TelegramSender / LogOTPSender dev);
  `POST /auth/otp/request` (anti-enum 202, phone+IP rate limits, cooldown, `LogOTPSender` when no token);
  `POST /auth/otp/verify` (single-use `CacheGetDel`, constant-time compare, 5-attempt lockout,
  reuses `issueTokenPair`). `/v1/me` now returns `phone`/`bale_linked`/`telegram_linked`. Frontend:
  3rd login tab (phone input → 6-digit InputOTP → verify → in-memory JWT); Settings "اتصال بله/تلگرام"
  card (linked status from `useMe`, deep links via `useBotLink` mutation). Live smoke: anti-enum 202 ✓,
  seeded OTP verify 200+JWT ✓, replay 401 ✓, rate limit 429 ✓, LogOTPSender logged ✓,
  migration round-trip clean ✓, email auth unaffected ✓. Host `go test` ✓ (+12 OTP tests) · `tsc` 0 · `npm build` ✓.
  **⚠️ Caveat:** full bot send + linking handshake blocked on real Bale bot token (operator must provision
  via Bale BotFather); `LogOTPSender` + seeded-Redis path covers all logic. Telegram API filtered in Iran —
  Bale is prod primary.
- TASK-0018 — Odometer Tracking for Efficiency Metrics (**DONE / CLOSED with caveat** by qa_supervisor,
  2026-06-03 — dev_supervisor ✅ (6/6) + qa ✅ (7/7 live) + qa_supervisor ✅; Phase-3 feature, no security
  stage). Optional `odometer_km` on the charging-session form → per-session **kWh/100km** (repo `LAG` prev
  reading → service `setSessionEfficiency`, `kwh/(Δkm/100)`, guard Δ>0) shown on the session card; dashboard
  `avg_kwh_per_100km` now derived from **session odometer deltas** (`EfficiencyAggregateByUser` CTE) instead of
  the cars'-odometer ratio. **Spec premise was wrong** — the column did NOT exist on `charging_sessions` (only
  `cars`); surfaced as a blocker, operator approved **migration 000009** (additive nullable + CHECK, reverses
  clean). **Shifts TASK-0017's migration to 000010.** Inline cost-save now round-trips `odometer_km` (PUT is
  full-replace). Live smoke green (eff=15 for the qualifying pair, dashboard avg=15, optional create 201,
  negative→400, up+down clean). Host `go test` ✓ (+3) · `tsc` 0 · `npm build` ✓. **⚠️ Caveat: UI not clicked
  (no browser).** Auto-Chain ran dev_supervisor→qa→qa_supervisor; paused once for the spec-error blocker.
- TASK-0016 — Admin UI for Charging Stations (**DONE / CLOSED with caveat** by qa_supervisor, 2026-06-03 —
  dev_supervisor ✅ (5/5) + security ✅ + qa ✅ (9/9 live) + qa_supervisor ✅; Phase-3 feature). Admin-only
  `/admin/stations` page: station table + add/edit dialog with a shared `StationMapPicker` (Leaflet/OSM
  click+drag marker) + delete `AlertDialog` confirm + `AdminRoute` guard + admin-only Header nav, all driving
  the existing `/v1/stations` CRUD through extended `features/stations/{api,hooks}` (TanStack mutations,
  ADR-002 — no `fetch()` in components). **Scope grew (operator-approved): added backend `GET /v1/me`**
  (`{id,email,is_admin}`, authed-only, no sensitive data) because `is_admin` is deliberately absent from the
  JWT (TASK-0013 instant-revocation) and no `/me` existed — `AdminOnly` left unchanged as the real boundary.
  Deviations: API field is **`power_kw`** (not `max_power_kw`); the table hydrates operator/address via
  per-row detail (list returns markers only). qa redeployed the api (in-container build, no wedge) + live
  smoke 9/9: /v1/me admin=true / non-admin=false / 401; non-admin POST→403; admin POST 201 / PUT 200 / DELETE
  204→404; lat=99→400. Host `go test` ✓ (+2 `GetUser` tests) · `tsc` 0 · `npm build` ✓. **⚠️ Caveat
  (operator-accepted):** UI guard not clicked (no browser on host) — redirect logic code-verified + the
  `/v1/me` it depends on proven live; the API admin boundary is fully proven live. **First time the Auto-Chain
  Rule (dev_supervisor→security→qa→qa_supervisor→commit) ran end-to-end.**
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
| TASK-0001–0028 | various | **ALL DONE ✅ CLOSED** — see phase summaries below |
| TASK-0029 | developer | **DONE ✅ CLOSED** (2026-06-11) — Hybrid Auth Flow + Admin OTP Config. `POST /auth/login/phone`, stay_logged_in on all paths, `POST /v1/account/set-password`, `password_set` in `/v1/me`, migration 000013 `system_settings`, `GET/PUT /v1/admin/system-settings`, `GET /auth/otp/config`, deep-link OTP, bot `/start phone_<E164>`, admin OTP config UI. `go test ✓` (25 tests) · `tsc 0` · `npm build ✓` |
| TASK-0031 | developer | **DONE ✅ CLOSED** (2026-06-11) — VPS Production Deployment (Phase 4). pg_dump backups, bind-mount postgres volume, Ubuntu 24.04 bootstrap, `.env.prod.example`, health check script, `DEPLOY_PRODUCTION.md`. |
| TASK-0032 | developer | **DONE ✅ CLOSED** (2026-06-11) — WSL local smoke test guide + VPS provisioning guide. End-to-end operator runbook from bare Ubuntu 24.04 → HTTPS live app. |
| TASK-0033 | developer | **DONE ✅ CLOSED with caveat** (2026-06-12 — dev_supervisor ✅ (1 finding fixed) + qa ✅ 9/9 live + qa_supervisor ✅). EV Car Catalog UI + Dynamic Theming: migration **000016** `ev_catalog` (23 cars × 43 cols from xlsx via stdlib converter `scripts/seed-ev-catalog.py`, round-trip clean); `GET /v1/cars/catalog` (authed, Redis `catalog:cars` TTL 1h, +3 service tests); `features/catalog/` (grid/list toggle, brand/body/segment/battery filters, range/accel/tier sort, detail Sheet w/ 6 spec sections + tabs, ColorPicker, ≤3-car comparison w/ best/worst highlight); `lib/dynamic-theme.ts` (Persian color name → warm/cool/neutral palette, persists `dynamic:<color>` under `voltana:theme`, ThemeContext restores, presets coexist); nav (BottomNav/Header/swipe) + lazy `/catalog` chunk (25 kB). `go test ✓` · `tsc 0` · `build ✓` · live: 401/200+23 cars/47 fields/TTL 3600. ⚠️ UI not clicked (no browser). |

## Current Focus
- **TASK-0035 — DONE ✅ CLOSED with caveat** (2026-06-12 — dev_supervisor ✅ + qa ✅ + qa_supervisor ✅). Unify Car Creation on the EV Catalog: `/cars` add dialog now searches the cached 23-car catalog (fa/en/brand, sets `catalog_car_id`, name defaults `name_fa`, ✕ unlink, hint → `/catalog`); ev_models is display-only legacy (deprecation header in `features/ev-models/api.ts`; endpoints + analytics fallback untouched). Live: legacy PUT round-trip keeps `ev_model_id`; quick-add 201 with name defaulted. Frontend-only, no Go changes. ⚠️ UI not clicked — same operator browser pass as 0033/0034. Resolves the ev_models↔ev_catalog open question at the UX level (table removal still gated on a data migration).
- **TASK-0034 — DONE ✅ CLOSED with caveat** (2026-06-12 — dev_supervisor ✅ + qa ✅ 10/10 live + qa_supervisor ✅). Add EV from Catalog to My Cars: migration **000017** (`cars.catalog_car_id` FK SET NULL + `spec_overrides JSONB` diff-only, round-trip clean); POST/PUT `/v1/cars` extended (name defaults to catalog `name_fa`; 422 `INVALID_CATALOG_CAR`; 400 `INVALID_OVERRIDE_KEY`; whitelist 43 keys + colors, typed, capacity/range > 0); `CatalogRepository.GetByID`; **analytics fallback chain** override→catalog→ev_model for capacity AND chemistry (+9 tests); `CustomizeCarModal` (43 fields/6 sections via `features/catalog/fields.ts`, constrained ColorPickers, diff-only save), CarDetail button «اضافه کردن به خودروهای من», `/cars` catalog badge + effective battery + PUT round-trip guard. Live: SOH nominal=60 from override (catalog 84.8), chemistry NMC from catalog, legacy path unchanged. ⚠️ UI not clicked (no browser) — same operator browser pass as TASK-0033.
- TASK-0033 (EV catalog + dynamic theming) closed 2026-06-12 with the standard no-browser caveat — retire with an operator browser pass (grid/list, filters, drawer, color→theme live + reload persistence, compare highlighting, mobile).
- **Working-tree note (2026-06-12):** uncommitted operator WIP in `voltana-web/src/pages/Auth.tsx` (Android intent deep-link for Bale) was deliberately **excluded** from the TASK-0033 commit; stray empty files `qa`/`super_dev` at repo root left untracked; `.claude/` added to `.gitignore`.
- **Phase 5 planning** (researcher/pm pass) is next. Likely candidates: OBD/ELM327 BLE integration, Capacitor mobile packaging, CDN/asset caching, multi-tenant/fleet sharing, push notifications, internationalization (full i18n). Catalog follow-ups: real car photos for `img_url`, public (unauthed) catalog decision, ev_models↔ev_catalog relationship.

## Phase 3 Summary (2026-06-02 → 2026-06-04)
Phase 3 delivered: TASK-0013 (map+stations) · 0014 (infra hardening) · 0015 (GitHub governance) · 0016 (admin UI) · 0017 (OTP/Bale bot) · 0018 (odometer) · 0019 (themes) · 0020 (fonts) · 0021 (currency) · 0022 (efficiency chart) · 0023 (total_km fix) · 0024 (session close button) · 0025 (VPS deployment). Also: bot poller exponential backoff + Telegram IPv4 fix + nginx static SPA serving + post-commit dashboard auto-sync hook.

## Previously Current Focus (archived)
- **➡️ Phase 3 active tasks (2026-06-03):**
  - **TASK-0023 ✅ DONE** — `total_km` now derived from session odometer deltas. Removed `ListByUser` loop in `GetDashboard`; uses `effKM` directly. Regression test added. Live smoke: 200→350 km tracking correctly; no-odometer sessions don't change it.
  - **TASK-0024 ✅ DONE** — Close button ("بستن" + X icon) added at bottom of expanded session detail. Chevron was already present.
  - **TASK-0019 ✅ DONE** — Theme system: 8 CSS-variable presets, ThemeProvider, Settings 4-col swatch grid.
  - **TASK-0020 ✅ DONE with caveat** — Font selection: 5 fonts (Vazirmatn default + Inter/Noto/Samim/System), FontProvider, Settings card. ⚠️ CDN (same as existing); self-hosted WOFF2 is Phase-4.
  - **TASK-0021 ✅ DONE** — Currency: migration `000011`, `formatCost` in `lib/cost.ts`, Settings toggle (Toman/Rial/USD), all cost displays updated. USD static rate 500k.
  - **TASK-0022 ✅ DONE** — Efficiency chart: `EfficiencyChart` component (Recharts ComposedChart, avg line, min/max band, tooltip), placed on Dashboard below SOH chart.
- **Recommended next:** TASK-0023 and TASK-0024 (bugs, READY) before the BACKLOG features.

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
  - **(0015) Branch protection on `main` — ✅ APPLIED 2026-06-03.** Repo made **PUBLIC** (operator ran
    `gh repo edit --visibility public`; secret-scan of full history was clean — no tracked `.env`, no secret
    diffs) to unblock the feature on the free plan. Protection now active: both CI checks required
    (`Go API — build · vet · test` + `Frontend — typecheck · build`, strict/up-to-date), **1 approving review**,
    no force-push, no deletions, conversation resolution required.
    - **`enforce_admins=false` (operator decision 2026-06-03).** Required PR + 1 approval + CI checks apply to
      non-admins/automation, but the **admin owner bypasses**, so the DoD "Git Commit Rule" (direct
      `git add . && commit && push` to `main`) still works for the persona workflow. Chosen over relaxing to
      0 approvals / a PR-only flow because a solo maintainer can't approve their own PR (would deadlock merges).
      If contributors are added later, switch to a real PR-based flow + re-enable `enforce_admins`.
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
