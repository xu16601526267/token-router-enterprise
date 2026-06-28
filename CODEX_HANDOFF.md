# Codex Handoff

## Branch

- Active branch: `develop`
- Purpose: backend-only baseline for a new independent interface rebuild.
- Remote: `origin` at `https://github.com/xu16601526267/token-router-enterprise.git`

## Current State

- Go backend routes are registered through `router.SetRouter`.
- API, relay, dashboard and video backends remain available.
- Old interface implementation has been removed from this branch.
- Docker and Compose files now build/run the Go backend only.
- `.env.example` no longer contains legacy client redirect configuration.

## Verified

```bash
go test ./...
go build -o /tmp/token-router-develop-api .
```

Local Docker verification was not run in the previous environment because Docker was unavailable there.

## Next Work

- Keep backend API contracts stable for the upcoming interface rebuild.
- Add or update backend tests when changing tenant, accounting, billing, settlement or routing logic.
- Treat old client references in archived historical notes as context only, not as active branch implementation.
