# ADR 0070: supply telemetry fleet agent

- Date: 2026-06-23
- Status: Accepted
- Related principles: P2 data before commitment; P5 human review; P8 platform boundary; P9 traffic as intelligence
- Related ADRs: 0058 supply capacity telemetry evidence; 0064 upstream capacity telemetry collector; 0065 supply capacity telemetry sweep; 0066 supply telemetry sweep runner

## Context

ADR 0066 added `token-router-supply telemetry sweep`, a repo-native one-shot runner for cron or systemd timers. That made capacity telemetry deployable, but it still left no durable fleet-agent evidence in the backend:

1. operators cannot see which resident telemetry process is alive;
2. sweep success / skipped / failure state is only visible from process logs unless telemetry rows are produced;
3. a dedicated server such as `aima2` cannot prove that a supply-side agent is running without an external process table or systemd state.

## Decision

Add a minimal supply telemetry fleet-agent control surface:

1. Add `SupplyTelemetryAgent`, keyed by `agent_key`, to store agent identity, hostname, runtime ref, version, last heartbeat, and last sweep summary.
2. Add admin APIs:
   - `GET /api/supply_telemetry_agents`
   - `POST /api/supply_telemetry_agents/heartbeat`
   - `POST /api/supply_telemetry_agents/sweep_result`
3. Add `token-router-supply telemetry agent [options]`, a resident loop that:
   - posts heartbeat evidence;
   - calls the existing `/api/supply_capacity_telemetries/sweep` API;
   - posts last sweep status and counts;
   - repeats by `--interval`, with `--once` for smoke tests and one-shot deployments.

## Boundaries

This increment must not:

1. start a hidden scheduler inside the API server;
2. create supplier, channel, capacity, pricing, billing, settlement, wallet, purchase order, or funds records;
3. dispatch remote commands to agents or implement task leasing;
4. auto-disable suppliers or channels, activate routing policies, change prices, or approve procurement;
5. treat heartbeat-only evidence as successful capacity telemetry collection.

## Acceptance

1. Agent heartbeat and sweep-result writes are idempotent by `agent_key`.
2. Agent list API exposes alive/stale evidence and last sweep summary for operators and automation.
3. `token-router-supply telemetry agent --once` posts heartbeat, performs sweep, records sweep result, and preserves `--fail-on-skip` / `--min-collected` semantics.
4. Go model and CLI tests cover upsert behavior, posted payloads, and failure accounting.
5. README / architecture / product principles / traffic docs distinguish the resident fleet agent from API-server internal workers and automatic execution.

## Implementation Record

- Added `SupplyTelemetryAgent`, heartbeat input, sweep-result input, filters, idempotent upsert by `agent_key`, and list query with `stale_before`.
- Added admin APIs: `GET /api/supply_telemetry_agents`, `POST /api/supply_telemetry_agents/heartbeat`, and `POST /api/supply_telemetry_agents/sweep_result`.
- Added `token-router-supply telemetry agent`; default identity is `<hostname>:telemetry`, `--once` runs one heartbeat+sweep+sweep_result cycle, and resident mode keeps looping by `--interval`.
- Preserved `--fail-on-skip` / `--min-collected` as non-zero gates in `--once` mode while still recording skipped / failed sweep summaries when the result endpoint is reachable.
- Validation passed on `aima2`: `go test ./model -run "SupplyTelemetryAgent|SupplyCapacity" -count=1`, `go test ./cmd/token-router-supply -run "TelemetryAgent|TelemetrySweep" -count=1`, `go test ./model ./controller ./router ./cmd/token-router-supply -count=1`, `go vet ./model ./controller ./router ./cmd/token-router-supply`, `go test ./model ./controller ./router ./cmd/token-router-sim ./cmd/token-router-supply ./tests/e2e -count=1`, and builds for `token-router-api`, `token-router-sim`, and `token-router-supply`. `go test ./... -count=1` still only fails at the root package because legacy `web/classic/dist` embed assets are absent; all other packages pass.
- Real process smoke passed on `aima2` at `/tmp/token-router-agent-once-20260623093517-2341857`: mock `gb10-4t` + seed + API server, then `token-router-supply telemetry agent --once --min-collected 1`. DB readback showed `agent_key=aima2:adr0070`, `last_sweep_status=ok`, `attempted=1`, `collected=1`, `skipped=0`, and telemetry/capacity evidence from `source_ref=gb10-4t-mock-capacity`.
