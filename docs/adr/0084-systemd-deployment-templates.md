# ADR 0084: Systemd deployment templates

## Status

Accepted

## Context

The project now has three deployable process surfaces:

- `token-router-api` for the API-only server used by real-process gb10 validation
- `token-router-supply telemetry agent` or `telemetry sweep` for supply telemetry collection
- `token-router-supply review agent` or `review once` for refreshing agent-readable review read models

The route docs repeatedly mention cron / systemd timer and later migration from `aima2` to a dedicated server, but the repository still only has the generic upstream `new-api.service`. That file does not describe the token-router binaries, the required environment variables, or the "choose resident agent or timer mode" deployment split.

Without repo-native templates, a dedicated server migration would rely on hand-written units and would be easy to misconfigure, especially around admin token handling, working directories, SQLite path, and accidentally enabling both resident and timer modes for the same runner.

## Decision

Add `deploy/systemd/` with:

1. `token-router-api.service`
2. `token-router-supply-telemetry-agent.service`
3. `token-router-supply-review-agent.service`
4. `token-router-supply-telemetry-sweep.service` + `.timer`
5. `token-router-supply-review-once.service` + `.timer`
6. `token-router.env.example`
7. `README.md` runbook

The templates will assume:

- binaries live under `/opt/token-router/bin`
- mutable state lives under `/var/lib/token-router`
- logs live under journald and `/var/log/token-router`
- secrets and deployment-specific values live in `/etc/token-router/token-router.env`
- systemd runs services as a dedicated `token-router` user / group

Operators must choose either resident agent mode or timer mode for each runner:

- telemetry: enable `token-router-supply-telemetry-agent.service` or `token-router-supply-telemetry-sweep.timer`
- review: enable `token-router-supply-review-agent.service` or `token-router-supply-review-once.timer`

## Boundaries

These templates must not:

- introduce a new scheduler inside the API server
- enable automatic approve / apply / activate / disable / complete / acknowledge behavior
- create suppliers, channels, capacity snapshots, cost profiles, prepaid lots, action plans, executions, or routing policies
- change `Channel.weight`, `Ability.weight`, `SupplierRoutePreference`, `SupplyRoutingPolicy`, pricing, billing, settlement, wallet, payout, invoice, or funds state
- store a real admin token in the repository

The templates only make existing process roles explicit and reproducible.

## Consequences

- `aima2` and the later dedicated server can use the same deployment shape.
- Operators get clear mode selection for resident processes versus timer jobs.
- Real-process validation remains separate from production rollout: passing `systemd-analyze verify` proves unit syntax, not that a live server has accepted traffic.

## Verification

Completed on `aima2`:

1. Unit syntax verification with temporary placeholder binaries and env file:

   ```bash
   rsync -az deploy/systemd/ aima2:/tmp/token-router-systemd-adr0084/
   ssh aima2 'bash -s' <<'EOF'
   set -euo pipefail
   mkdir -p /opt/token-router/bin /etc/token-router
   cleanup() {
     rm -f /opt/token-router/bin/token-router-api /opt/token-router/bin/token-router-supply /etc/token-router/token-router.env
     rmdir /opt/token-router/bin /opt/token-router /etc/token-router 2>/dev/null || true
   }
   trap cleanup EXIT
   touch /opt/token-router/bin/token-router-api /opt/token-router/bin/token-router-supply /etc/token-router/token-router.env
   chmod +x /opt/token-router/bin/token-router-api /opt/token-router/bin/token-router-supply
   systemd-analyze verify /tmp/token-router-systemd-adr0084/*.service /tmp/token-router-systemd-adr0084/*.timer
   EOF
   ```

   Output: command exited `0` with no diagnostics.

2. The initial verify pass caught an invalid `Documentation=README:...` URL. The templates now use `Documentation=file:/opt/token-router/deploy/systemd/README.md`.

3. README / architecture / traffic docs updated to reference the systemd bundle and preserve the no-auto-execution boundary.
