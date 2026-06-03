# Voltana — AI System Bootstrap Prompt

کپی کن و مستقیم به Claude Code بده.
Claude Code خودش همه فایل‌ها رو می‌سازه.

---

## PROMPT (کپی از اینجا تا END)

---

You are setting up the AI orchestration system for the **Voltana** project — a self-hosted EV companion app (Go API + PostgreSQL + Redis + React).

Your job is to create the entire `.ai/` folder structure and `CLAUDE.md` from scratch inside the current project root. Do not ask questions — just build it.

---

## Step 1 — Create CLAUDE.md in project root

Create `CLAUDE.md` with this exact content:

```markdown
# Voltana — Claude Code Master Context

> Read this file completely before doing anything. It is your source of truth.

## Project Overview

**Voltana** is a self-hosted, open-source smart EV companion app.
Stack: Go (Gin) · PostgreSQL 16 · Redis 7 · React 18 + TypeScript · Docker Compose · Capacitor (Android/iOS).

- API: `voltana-api/` (Go)
- Frontend: `voltana-web/` (React + Vite)
- AI system: `.ai/`

## How to Start Every Session

1. Read `.ai/context.md` — current state + last task + blockers
2. Read `.ai/PERSONA_ROUTER.md` — who does what
3. Check `.ai/workflows/` — find READY tasks
4. Announce your persona before acting: `# Acting as: <persona>`

## Persona System

| Persona | Role | Can write code? |
|---|---|---|
| `pm` | Product Manager — defines scope & acceptance criteria | ❌ |
| `architect` | Designs modules, APIs, ADRs | ❌ |
| `developer` | **Only persona that writes code/config** | ✅ |
| `dev_supervisor` | Reviews developer output | ❌ |
| `feature` | Designs feature specs | ❌ |
| `security` | Reviews auth, secrets, crypto | ❌ |
| `qa` | Runs checks, writes test plans | ❌ |
| `qa_supervisor` | Approves QA evidence, closes tasks | ❌ |
| `docs` | Writes documentation | ❌ |
| `release` | CI/CD, versioning, signing | ❌ |

**Golden rule**: Only `developer` touches files. All others output instructions that `developer` executes.

## Workflow Lifecycle

```
BACKLOG → READY → IN_PROGRESS → REVIEW → DONE
                                       ↘ BLOCKED
```

After every meaningful step: update the task `.md` file status.

## Core Architecture Rules

### Backend (Go)
- Dependencies inward only: `handler → service → repository → domain`
- No business logic in handlers. No SQL in services.
- Every repository function filters by `user_id` from JWT — no exceptions.
- All secrets via env vars. Never hardcode.

### Frontend (React)
- No component calls `fetch()` directly. All HTTP: `features/<name>/api.ts` → `hooks.ts`
- TanStack Query everywhere. No raw `useEffect` for data fetching.
- Access token in memory. Refresh token in httpOnly cookie.

### Database
- Every migration is a new file. Never edit existing migrations.
- Every new table needs `created_at TIMESTAMPTZ DEFAULT now()`.
- Mutable tables need `updated_at` with auto-trigger.

## Definition of Done

- [ ] Code compiles without errors
- [ ] Tests pass (`go test ./...` or `npm run test`)
- [ ] No secrets hardcoded
- [ ] Workflow task status updated
- [ ] `changelog.md` updated
- [ ] dev_supervisor approved
```

---

## Step 2 — Create .ai/context.md

```markdown
# Voltana — Project Context

> Update this file at the end of every session.

## Current State

- **Active Phase**: Phase 1 — Solid Foundation
- **Current Sprint**: Week 1
- **Last Completed Task**: None — bootstrap in progress

## Active Tasks

| Task | Persona | Status |
|---|---|---|
| TASK-0001 | developer | READY |
| TASK-0002 | developer | BACKLOG |
| TASK-0003 | developer | BACKLOG |
| TASK-0004 | developer | BACKLOG |
| TASK-0005 | developer | BACKLOG |
| TASK-0006 | developer | BACKLOG |
| TASK-0007 | developer | BACKLOG |
| TASK-0008 | developer | BACKLOG |

## Blockers
- None

## Key Decisions
- Backend: Go (Gin) — replaces Supabase
- Auth: Self-managed JWT (memory + httpOnly cookie)
- DB: PostgreSQL 16 self-hosted in Docker
- Frontend: Keep React, refactor to feature-based structure
- Mobile: Capacitor wraps React PWA

## Open Questions
- Email provider for verification (SMTP self-hosted or service?)
- Neshan map API key — needed for Phase 2
- OBD: ELM327 BLE vs USB — decide in Phase 3
```

---

## Step 3 — Create .ai/PERSONA_ROUTER.md

```markdown
# PERSONA_ROUTER — Who Should Act Now?

## Decision Tree

- Define requirements / acceptance criteria → `pm`
- Design module structure, API contracts, ADR → `architect`
- Design a specific feature (UI + state + hooks) → `feature`
- Write or change any code / config / file → `developer`
- Review code that developer wrote → `dev_supervisor`
- Review auth / secrets / crypto → `security`
- Run tests / build / check pipeline → `qa`
- Approve test evidence, close task → `qa_supervisor`
- Write or update docs → `docs`
- CI/CD / versioning / release → `release`

## Handoff Protocol

Every NO-CODE persona ends output with:

```
## Handoff → developer
- File to create/edit: ...
- Exact change: ...
- Acceptance: ...
```

## Voltana Persona Map

| Area | Primary | Reviewer |
|---|---|---|
| Go API | developer | dev_supervisor |
| DB migrations | developer (architect designs) | dev_supervisor |
| React features | developer (feature designs) | dev_supervisor |
| JWT / auth | developer | security + dev_supervisor |
| Docker / Nginx | developer (architect designs) | dev_supervisor |
| Battery algorithm | architect → developer | dev_supervisor + qa |
| OBD integration | architect + security → developer | security + qa_supervisor |
```

---

## Step 4 — Create .ai/workflows/TASK-0001.md

```markdown
# TASK-0001 — Docker Compose Stack Bootstrap

**Status**: READY
**Phase**: 1
**Persona**: developer
**Reviewer**: dev_supervisor

## Goal
Bring up Postgres, Redis, Nginx, Go API skeleton with one command.

## Scope In
- `docker-compose.yml` — services: postgres, redis, nginx, api, migrate
- `voltana-api/Dockerfile` — Go multi-stage build
- `voltana-api/cmd/server/main.go` — minimal server, health check only
- `.env.example` — all required env vars, no real values
- `nginx/nginx.conf` — reverse proxy to Go API port 8080
- `migrations/000001_init_schema.sql` — base schema (users, cars, charging_sessions, ev_models, user_settings)

## Scope Out
- No business logic yet
- No auth endpoints (TASK-0002)
- No frontend changes

## Acceptance Criteria
- [ ] `docker-compose up -d` starts all 5 services without errors
- [ ] `curl http://localhost/health` returns `{"status":"ok"}`
- [ ] Postgres volume persists between restarts
- [ ] Redis AOF persistence enabled
- [ ] migrate service exits 0 after running migrations
- [ ] No secrets in committed files — only in .env (gitignored)

## Architecture: Go project structure
```
voltana-api/
  cmd/server/main.go
  internal/
    domain/
    repository/
    service/
    handler/
      health_handler.go
    middleware/
  migrations/
  Dockerfile
  go.mod
```

## Docker Compose service startup order
1. postgres (healthcheck: pg_isready)
2. redis (healthcheck: redis-cli ping)
3. migrate (depends_on: postgres healthy)
4. api (depends_on: migrate complete + redis healthy)
5. nginx (depends_on: api healthy)

## Evidence Required
```bash
docker-compose ps              # all services Up/healthy
curl http://localhost/health   # {"status":"ok"}
docker logs voltana-migrate    # migration success
```

## Changelog (fill when DONE)
- Date:
- Files touched:
- Notes:
```

---

## Step 5 — Create .ai/workflows/TASK-0002.md

```markdown
# TASK-0002 — Go Auth API

**Status**: BACKLOG
**Phase**: 1
**Persona**: developer
**Reviewer**: dev_supervisor + security
**Depends on**: TASK-0001

## Goal
Full JWT auth flow — register, login, refresh, logout — replacing Supabase auth.

## Scope In
- `POST /auth/register` — create user, send verification email
- `POST /auth/login` — returns access token + sets httpOnly refresh cookie
- `POST /auth/refresh` — rotates refresh token, returns new access token
- `POST /auth/logout` — invalidates refresh token in Redis
- `migrations/000002_users_table.sql`
- `internal/domain/user.go`
- `internal/repository/user_repo.go`
- `internal/service/auth_service.go`
- `internal/handler/auth_handler.go`
- `internal/middleware/auth.go` — JWT validation

## Acceptance Criteria
- [ ] register creates user with bcrypt cost 12
- [ ] login returns 200 + access token in body + refresh in httpOnly cookie
- [ ] login returns 401 for wrong credentials
- [ ] refresh rotates + blacklists old token in Redis
- [ ] logout invalidates refresh token
- [ ] Access token TTL: 15 minutes
- [ ] Refresh token TTL: 30 days
- [ ] Rate limit: 10 login attempts / 15 min per IP
- [ ] No token stored in localStorage
- [ ] `go test ./internal/service/...` passes

## Security Rules (mandatory)
- bcrypt cost MUST be 12
- Refresh token stored in Redis with TTL
- Invalidate old token BEFORE issuing new one
- JWT_SECRET from env var only — never hardcoded
- Logs must NOT contain: password, token value, Authorization header
```

---

## Step 6 — Create .ai/workflows/TASK-0003.md through TASK-0008.md

Create one file per task:

**TASK-0003** — Cars & EV Models CRUD
- Status: BACKLOG / Phase 1 / Depends: TASK-0002
- Endpoints: GET/POST/PUT/DELETE /v1/cars, GET /v1/ev-models (bilingual search)
- Criteria: all CRUD with valid JWT, 401 without, every query filters by user_id, tests pass

**TASK-0004** — Charging Sessions CRUD
- Status: BACKLOG / Phase 1 / Depends: TASK-0003
- Endpoints: GET/POST/PUT/DELETE /v1/charging-sessions
- Criteria: server-side cost calc, SOC validated 0–100, user_id filter always, tests pass

**TASK-0005** — User Settings API
- Status: BACKLOG / Phase 1 / Depends: TASK-0003
- Endpoints: GET/PUT /v1/settings
- Criteria: auto-create on first GET, rates writable, default_car_id writable

**TASK-0006** — Frontend: Replace Supabase SDK
- Status: BACKLOG / Phase 1 / Depends: TASK-0005
- Remove @supabase/supabase-js, add src/lib/api.ts with JWT interceptor
- Fix all 8 known bugs:
  1. window.location.href → useNavigate (Charging.tsx:543)
  2. window.location.reload() → queryClient.invalidateQueries() (Header.tsx:96)
  3. Neshan key → VITE_NESHAN_API_KEY env var (Map.tsx:7)
  4. Mixed toast → sonner everywhere
  5. fetchSessions called 4x → single TanStack Query
  6. SOC bar reversed → fix startSoc/endSoc (SOCAnalysis.tsx:83-88)
  7. No email gate → confirmation screen before dashboard
  8. Map stub → keep stub, remove hardcoded key
- Criteria: npm run build passes, login/session persist, no Supabase imports, all 8 bugs fixed

**TASK-0007** — Battery Health Snapshots (Phase 2)
- Status: BACKLOG / Phase 2 / Depends: TASK-0004
- Background job via asynq, triggered after each session
- Delta-SOC algorithm, LFP vs NMC chemistry rules
- Endpoints: GET /v1/analytics/battery/:car_id, GET /v1/analytics/recommendations/:car_id
- Criteria: SOH% returned, LFP/NMC rules correct, tests pass

**TASK-0008** — Dashboard Analytics + Battery Chart (Phase 2)
- Status: BACKLOG / Phase 2 / Depends: TASK-0007
- GET /v1/analytics/dashboard — total kWh, cost, kWh/100km, km
- Redis cache 5 minutes
- Battery health Recharts chart in React dashboard
- Criteria: all stats correct, cache works, chart renders

---

## Step 7 — Create .ai/spec/ADR-001.md

```markdown
# ADR-001 — Replace Supabase with Self-Hosted Go + PostgreSQL

**Date**: May 2026 | **Status**: Accepted

## Decision
Replace Supabase with: Go (Gin) API · PostgreSQL 16 · Redis 7 · golang-migrate

## Why
- Full self-hosting, zero external dependencies
- Go is ideal for future OBD serial port / BLE integration
- JWT in memory (XSS-safe) vs Supabase localStorage pattern
- Single docker-compose up to run everything

## Migration
1. Port schema from Supabase migrations to standard SQL
2. Replace auth.users with self-managed users table
3. Remove auth.uid() RLS — enforce in Go repository layer (user_id filter)
4. Replace Supabase JS SDK with plain fetch + JWT interceptor

# ADR-002 — Feature-Based Frontend Structure

**Date**: May 2026 | **Status**: Accepted

## Decision
Each feature owns: api.ts, hooks.ts, components/, Page.tsx
No component calls fetch() directly — always through hooks.ts

## Why
Current Charging.tsx is 860 lines mixing everything.
Feature isolation lets one agent/developer work on one feature safely.

# ADR-003 — JWT Token Storage

**Date**: May 2026 | **Status**: Accepted

## Decision
- Access token: React state (memory) — lost on refresh, re-issued silently
- Refresh token: httpOnly cookie — not accessible to JavaScript

## Why
localStorage is XSS-vulnerable (current Supabase behavior).
httpOnly cookie eliminates token theft via XSS.
```

---

## Step 8 — Create changelog.md in project root

```markdown
# Voltana Changelog

## [Bootstrap] — May 2026
### Added
- .ai/ orchestration system (CLAUDE.md, PERSONA_ROUTER, context, workflows, ADRs)
- TASK-0001 through TASK-0008 defined
- ADR-001 (Supabase → Go), ADR-002 (feature structure), ADR-003 (JWT)
```

---

## Step 9 — Verify

After creating all files, run:

```bash
find .ai -name "*.md" | sort
cat CLAUDE.md | head -5
cat .ai/context.md | head -5
```

Confirm output shows all expected files. Then report:
- Files created: list
- Any errors: list
- Next step: "Run `# Acting as: developer` and start TASK-0001"

---
END OF PROMPT
