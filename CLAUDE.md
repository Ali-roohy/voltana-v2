# Voltana V2 — CLAUDE.md

## Project Overview

Voltana is a self-hosted EV fleet/vehicle management platform. V2 replaces the Supabase MVP with a fully self-hosted Go + PostgreSQL stack. Dev environment is WSL2; deployment target is Ubuntu VPS (2 vCPU / 4 GB RAM) via Docker Compose.

---

## Tech Stack

| Layer | Technology |
|---|---|
| API | Go (Gin), layered architecture (handler → service → repository) |
| Database | PostgreSQL 16 (self-hosted Docker), migrations via `golang-migrate` |
| Cache / Queue | Redis 7 (JWT blacklist, rate limiting, background jobs) |
| Auth | Self-managed JWT — access token in React memory, refresh token in httpOnly cookie |
| Frontend | React (feature-based structure), TanStack Query, plain fetch + JWT interceptor |
| Mobile | Capacitor wrapping React PWA — no separate native codebase |
| Infrastructure | Docker Compose (postgres → redis → migrate → api → nginx) |

---

## Persona System

Only `developer` writes code. All other personas produce specs, designs, or reviews — never code directly.

| Persona | Role |
|---|---|
| `researcher` | Competitive analysis, UX research, feature discovery (hands off to pm/feature) |
| `pm` | Requirements, scope, acceptance criteria |
| `architect` | Module design, API contracts, ADRs |
| `feature` | UI + state + hook design (hands off to developer) |
| `developer` | **Sole code author** — implements handoffs |
| `dev_supervisor` | Reviews all developer output |
| `security` | Reviews auth / secrets / crypto / security boundaries |
| `qa` | Runs tests, build, pipeline checks |
| `qa_supervisor` | Approves test evidence and closes tasks |
| `docs` | Writes/updates documentation |
| `release` | CI/CD pipeline, versioning, releases |

### Handoff Protocol

Non-code personas end their output with:
```
## Handoff → developer
- File to create/edit: ...
- Exact change: ...
- Acceptance criteria: ...
```
After implementation, developer hands to `dev_supervisor`. After review passes, `qa` runs evidence checks, `qa_supervisor` closes the task.

---

## Workflow Lifecycle

```
BACKLOG → READY → IN_PROGRESS → REVIEW → TESTING → DONE
```

- Tasks live in `.ai/workflows/TASK-XXXX.md`
- Update `.ai/context.md` at the end of every session
- Each task has explicit **Scope In / Scope Out** and **Acceptance Criteria**
- Evidence (curl output, docker ps, test results) is required before a task is closed

---

## Dashboard Sync Rule

After every task status change (any `.ai/workflows/TASK-*.md` update), run:

```
node voltana-dashboard-sync.js
```

This updates `voltana-dashboard.html` with live task statuses and stat counts from the workflow files. No arguments needed — it finds everything automatically.

---

## Core Architecture Rules

### Go API
- Structure: `cmd/server/main.go` → `internal/handler/` → `internal/service/` → `internal/repository/` → `internal/domain/`
- No business logic in handlers; no DB calls in services directly — always through the repository layer
- User isolation: enforce `user_id` filtering in the repository layer (no RLS — that was Supabase)

### Auth (ADR-003)
- Access token: React state only (memory) — never localStorage, never sessionStorage
- Refresh token: httpOnly cookie — never readable by JS
- `lib/api.ts` intercepts 401s, silently calls `POST /auth/refresh`, retries the original request

### Frontend (ADR-002)
- Feature-based structure: `features/<name>/api.ts`, `hooks.ts`, `components/`, `Page.tsx`
- **No component calls `fetch()` directly** — all HTTP goes through `features/<name>/api.ts` then `hooks.ts`
- Shared UI only in `src/components/`; shared logic only in `src/lib/`
- TanStack Query everywhere — no raw `useEffect` for data fetching

### Infrastructure
- `docker-compose up -d` must bring up the full stack
- Service startup order: postgres (healthy) → redis (healthy) → migrate (exits 0) → api (healthy) → nginx
- No secrets in committed files — use `.env.example` with placeholders

---

## Dev Environment Notes

- **Run Go tests with host Go, never the `golang:1.22-alpine` container.** The container wedges on
  this 2 vCPU / 4 GB dev host (cold compile starves on CPU/IO). Always:
  ```
  cd voltana-api && go test ./...
  ```
- Do not echo secrets (e.g. `$POSTGRES_PASSWORD`) into the terminal. Run `psql` inside the postgres
  container with env-var expansion (`docker exec voltana-postgres sh -c 'psql -U "$POSTGRES_USER" …'`)
  or apply migrations via `docker compose run --rm migrate` (reads `.env` itself).

---

## Active State (May 2026)

- **Phase**: 1 — Solid Foundation, Week 1
- **TASK-0001** (Docker Compose stack bootstrap) — READY / developer
- **TASK-0002** (Auth endpoints) — BACKLOG
- **TASK-0003** (Architecture) — BACKLOG / architect

---

## Key Files

| Path | Purpose |
|---|---|
| `.ai/context.md` | Live project state — update each session |
| `.ai/PERSONA_ROUTER.md` | Persona decision tree and handoff rules |
| `.ai/spec/ADR-001-003.md` | Accepted architecture decisions |
| `.ai/workflows/TASK-XXXX.md` | Per-task specs, acceptance criteria, evidence |
