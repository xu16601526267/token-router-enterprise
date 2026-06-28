# AGENTS.md

## Scope

This `develop` branch is backend-only. It keeps the Go service, API relay, tenant/accounting domain, billing and settlement logic, provider routing, operational governance APIs, and background workers.

Do not reintroduce the removed legacy interface implementation in this branch. Future interface work should consume the backend API surface.

## Backend Stack

- Go 1.22+
- Gin router
- GORM with SQLite/MySQL/PostgreSQL support
- Redis where enabled by configuration
- Logrus for logging
- Viper-style environment/config loading used by the existing codebase

## Important Areas

- `main.go`: backend service bootstrap.
- `router/`: API, relay, dashboard and video route registration.
- `controller/`: HTTP handlers and request/response glue.
- `service/`: domain logic for tenant, billing, settlement, routing and governance.
- `model/`: GORM models and persistence contracts.
- `relay/`: upstream provider relay logic.
- `middleware/`: auth, logging, request handling and cross-cutting middleware.
- `worker/`: background usage, telemetry and async jobs.

## Development Commands

```bash
go test ./...
go build -o /tmp/token-router-develop-api .
docker compose -f docker-compose.dev.yml up -d
```

If Docker is not installed locally, verify Docker builds on the target machine or CI runner.

## Code Rules

- Follow the repository's existing Go style and package boundaries.
- Keep API response wrappers and error shapes consistent with existing handlers.
- Preserve backwards-compatible database migrations unless a breaking migration is explicitly requested.
- Avoid changing unrelated domain behavior while preparing backend API support for the interface rebuild.
- Keep tenant, usage, cost and settlement changes covered by focused tests.
- Do not remove protected upstream attribution or legal identifiers from notices and license files.

## Branch Intent

`develop` is the clean backend base for the upcoming independent interface rebuild. It should stay focused on backend service code and backend delivery.
