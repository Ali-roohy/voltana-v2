## What & why
<!-- Summary of the change and the task it closes. -->

Closes: TASK-XXXX / #<issue>

## Type
- [ ] Feature (minor bump)  [ ] Fix (patch)  [ ] Infra/CI  [ ] Docs

## Checklist
- [ ] `cd voltana-api && go build ./... && go vet ./... && go test ./...` passes (host)
- [ ] `cd voltana-web && npx tsc --noEmit && npm run build` passes
- [ ] New/changed migrations are paired (`.up.sql` + `.down.sql`) and apply cleanly
- [ ] No secrets committed (`.env*` stay git-ignored)
- [ ] `CHANGELOG.md` updated under `Unreleased`
- [ ] `VERSION` bumped if this closes a feature/release

## Evidence
<!-- Test output, live smoke (curl), screenshots. -->
