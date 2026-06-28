# CLAUDE.md

This branch is a backend-only development baseline.

Use `AGENTS.md` for the active engineering instructions. The important constraint is that `develop` must remain focused on the Go backend: API relay, multi-tenant management, API keys, routing, usage accounting, billing, settlement, operational governance and workers.

Do not restore the removed legacy interface code while working on this branch. Future interface work should consume the backend APIs from a fresh implementation.
