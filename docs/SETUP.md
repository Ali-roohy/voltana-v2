# Voltana V2 — Setup Guide

This guide takes you from a fresh clone to a running stack, plus how to create the first admin user, test
email locally with MailHog, and run the frontend in development.

---

## 1. Prerequisites

| Tool | Version | Notes |
|---|---|---|
| **Docker Engine** | 24+ | Runs the whole backend stack |
| **Docker Compose** | **v2** (the `docker compose` plugin) | ⚠️ Compose **v1** (`docker-compose`) fails on rebuilt images (`KeyError: 'ContainerConfig'`) — use v2 |
| **Go** | 1.22.x | Only needed to run backend tests on the host (`go test ./...`) |
| **Node.js** | 20 LTS (18+ minimum) | For the Vite frontend |
| **npm** | 9+ | Ships with Node |

> You do **not** need Go or Node to *run* the stack — Docker builds the API image itself. They are only needed
> for local development (host tests, frontend dev server).

---

## 2. Clone & Configure

```bash
git clone git@github.com:Ali-roohy/voltana-v2.git
cd voltana-v2
cp .env.example .env
```

Edit `.env` and set **at least** these:

| Variable | What to set |
|---|---|
| `POSTGRES_PASSWORD` | A strong password |
| `JWT_SECRET` | A random string, **32+ characters** |
| `APP_ENV` | `development` locally; **`production` on the VPS** (so the refresh cookie gets `Secure`) |
| `APP_URL` | Public base URL used in verification links (`http://localhost` locally) |
| `SMTP_HOST` / `SMTP_PORT` | Leave blank for the dev log mailer, **or** `mailhog` / `1025` for local email testing (see §5) |

Full variable reference is in [`.env.example`](../.env.example).

> **Never commit `.env`** — it is git-ignored. Only `.env.example` (placeholders) is committed.

---

## 3. First Run (Docker Compose)

Bring up the full backend stack. Startup order is enforced by health checks:
**postgres (healthy) → redis (healthy) → migrate (runs all migrations, exits 0) → api (healthy) → nginx**
(plus `mailhog` in dev).

```bash
docker compose up -d --build
```

Verify:

```bash
docker compose ps                  # all services Up / healthy
curl http://localhost/health       # → {"status":"ok"}
docker compose logs migrate        # should show 000001…000008 applied
```

### Service endpoints

| Service | URL / Port | Purpose |
|---|---|---|
| **API (via nginx)** | http://localhost (`:80`) | All HTTP requests go through nginx → Go API on `:9090` |
| **MailHog UI** | http://localhost:8025 | Dev email inbox (verification links) |
| PostgreSQL | internal `:5432` | Not published to the host |
| Redis | internal `:6379` | Not published to the host |

### Redeploying after a code change

```bash
docker compose up -d --build api
```

nginx re-resolves the API container on its own (no reload needed). Only an **nginx config** change needs
`docker compose restart nginx`. On a heavily loaded host the in-container Go build can be slow; a host-compiled
fallback image (`voltana-api/Dockerfile.runtime`) is available for that case.

---

## 4. Create the First Admin User

Admin privileges gate the station-management write endpoints (`POST/PUT/DELETE /v1/stations`). There is **no
self-serve admin signup** — the `users.is_admin` flag defaults to `false` and is granted out-of-band by SQL.

**Step 1 — register a normal user** (via the app or curl):

```bash
curl -X POST http://localhost/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"YourStrongPassw0rd!"}'
```

**Step 2 — (local only) mark the email verified.** In production the user clicks the verification link
(captured by MailHog locally, see §5). For a quick local bootstrap you can flip the flag directly:

```bash
docker exec voltana-postgres sh -c \
  'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
   -c "UPDATE users SET is_email_verified = true WHERE email = '"'"'you@example.com'"'"';"'
```

**Step 3 — promote to admin:**

```bash
docker exec voltana-postgres sh -c \
  'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
   -c "UPDATE users SET is_admin = true WHERE email = '"'"'you@example.com'"'"';"'
```

The change takes effect **immediately** — the admin check is performed fresh against the database on every
write request (it is not baked into the JWT), so there's no need to log in again.

> **Note:** the psql commands above use `$POSTGRES_USER` / `$POSTGRES_DB` expanded *inside* the container so
> credentials are never echoed to your shell.

---

## 5. Local Email Testing with MailHog

Registration sends an email-verification link. To catch it locally instead of sending real mail:

1. In `.env` set:
   ```
   SMTP_HOST=mailhog
   SMTP_PORT=1025
   ```
2. `docker compose up -d` (MailHog is already in the compose file).
3. Register a user, then open **http://localhost:8025** — the verification email (with the link) appears there.

If `SMTP_HOST` is left blank, the API uses a **dev log mailer** that sends nothing (and never prints the token);
use the SQL flip in §4 to verify in that case.

---

## 6. Frontend Development

The backend (via nginx) serves the API on `:80`. The React frontend is run separately with Vite and talks to
that same origin.

```bash
cd voltana-web
cp .env.example .env          # VITE_API_URL=/  → same-origin API through nginx
npm install
```

| Command | Result |
|---|---|
| `npm run dev` | Vite dev server with HMR → http://localhost:5173 |
| `npm run build` | Production build into `dist/` |
| `npm run preview` | Serve the production build → http://localhost:4173 |
| `npx tsc --noEmit` | Type-check only (must be 0 errors) |
| `npm run lint` | ESLint |

> **Deprecated env:** `VITE_NESHAN_API_KEY` is no longer used — the station map switched to keyless
> **Leaflet + OpenStreetMap** (no API key required). You can leave it blank or remove it.

### Same-origin requirement

`VITE_API_URL` must remain same-origin (served by nginx). The access token lives only in React memory and the
refresh token is an httpOnly cookie, so cross-origin setups break the silent-refresh flow.

---

## 7. Running Backend Tests

Run Go tests on the **host** (not in the `golang:1.22-alpine` container — a cold container compile starves the
small dev host):

```bash
cd voltana-api
go test ./...        # service-layer unit tests
go vet ./...
```

---

## 8. Troubleshooting

| Symptom | Fix |
|---|---|
| `docker-compose up` errors with `KeyError: 'ContainerConfig'` | You're on Compose **v1**. Use `docker compose` (v2 plugin). |
| API unhealthy after start | `docker compose logs api` — usually a bad `DATABASE_URL`/`JWT_SECRET` or migrations not applied. |
| Verification link never arrives | Set `SMTP_HOST=mailhog`/`SMTP_PORT=1025` and check http://localhost:8025, or use the SQL flip in §4. |
| `403 admin privileges required` on station writes | Your user isn't admin — run the §4 promote SQL. |
| Frontend can't reach API | Ensure `VITE_API_URL=/` and the stack is up (`curl http://localhost/health`). |

---

## 9. Contributing & Branch Protection

The `main` branch is protected — all changes land via pull request, and CI must pass before merge.

### Branch-protection rules (configured on `main`)

- **No direct pushes to `main`** — every change goes through a PR.
- **Require a pull request before merging** (≥ 1 approving review; CODEOWNERS auto-requested).
- **Require status checks to pass** before merging, with **"branches up to date"** enforced. Required checks:
  - `Go API — build · vet · test`
  - `Frontend — typecheck · build`
- **Require conversation resolution** before merging.
- **No force pushes**, **no branch deletion** on `main`.

### Applying the rules (GitHub CLI)

> Status checks can only be marked *required* after they've run at least once — open the first PR, let CI run,
> then apply this. The check contexts must exactly match the CI job `name:` values in `.github/workflows/ci.yml`.

```bash
gh api -X PUT repos/Ali-roohy/voltana-v2/branches/main/protection \
  -H "Accept: application/vnd.github+json" \
  -f 'required_status_checks[strict]=true' \
  -f 'required_status_checks[checks][][context]=Go API — build · vet · test' \
  -f 'required_status_checks[checks][][context]=Frontend — typecheck · build' \
  -F 'enforce_admins=true' \
  -F 'required_pull_request_reviews[required_approving_review_count]=1' \
  -F 'restrictions=' \
  -F 'allow_force_pushes=false' \
  -F 'allow_deletions=false'
```

### Release flow (Semantic Versioning)

1. Update [`../VERSION`](../VERSION) and move `CHANGELOG.md`'s `Unreleased` entries into a new version section
   with the date.
2. Merge the PR to `main` (CI green).
3. Tag and push: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`.

Bump rules (pre-1.0): **MINOR** when a feature task closes (new endpoints / backward-compatible migration),
**PATCH** for fixes/docs/infra, **MAJOR** reserved for the v1.0.0 commercial release.
