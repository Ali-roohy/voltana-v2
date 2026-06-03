# Security Policy

## Supported versions
The latest `0.x` release on `main` receives security fixes. Older tags are not maintained.

## Reporting a vulnerability
**Do not open a public issue for security problems.**

Email **ali.roohi.eng@gmail.com** with:
- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- affected component/version (commit SHA or `VERSION`).

You'll get an acknowledgement within **72 hours**. Please allow a reasonable disclosure window before any
public disclosure; we'll coordinate a fix and credit you if desired.

## Scope
In scope: the Go API (auth, authorization, data isolation), JWT/refresh handling, the admin boundary, and the
Docker/nginx configuration in this repo. Out of scope: third-party dependencies' own advisories (report upstream)
and self-hosted deployments you operate.

## Handling notes
- Secrets live only in `.env` files (git-ignored); never commit credentials.
- Auth design: access token in memory, refresh token in an httpOnly cookie; admin via out-of-band SQL only.
