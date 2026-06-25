# ADR 0071: API server supply telemetry worker

- Date: 2026-06-23
- Status: Accepted
- Related principles: P2 data before commitment; P5 human review; P8 platform boundary; P9 traffic as intelligence
- Related ADRs: 0065 supply capacity telemetry sweep; 0066 supply telemetry sweep runner; 0070 supply telemetry fleet agent

## Context

ADR 0070 added a resident deployment-side `token-router-supply telemetry agent`. That proves a dedicated supply-side process can heartbeat, sweep upstream capacity telemetry, and record its latest result.

The remaining architecture gap is an API-server-owned telemetry worker for smaller deployments that do not want a separate process yet. This must not become hidden automation: the platform still cannot automatically change supplier posture, channel status, routing policy, pricing, procurement, billing, settlement, or funds state.

## Decision

Add an opt-in API server worker:

1. It starts only when `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED=true` and the process is the master node.
2. It runs the existing `model.SweepSupplyCapacityTelemetry` on a fixed interval.
3. It records the same `SupplyTelemetryAgent` heartbeat and sweep-result evidence as the external fleet agent, using an `agent_key` default of `api-server:<NODE_NAME or hostname>:supply-telemetry-worker`.
4. It supports narrow filters through environment variables:
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_INTERVAL`
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_AGENT_KEY`
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_RUNTIME_REF`
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_SUPPLIER_ID`
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_CHANNEL_ID`
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_SUPPLY_NODE`
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_MODEL`
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_PERIOD_START`
   - `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_PERIOD_END`
5. It runs once at startup, then repeats by interval.

## Boundaries

This worker must not:

1. start unless explicitly enabled;
2. create suppliers, channels, capacity targets, pricing, billing, settlement, wallet, purchase order, or funds records;
3. auto-disable suppliers or channels, activate routing policies, change prices, approve procurement, or acknowledge insights;
4. treat heartbeat-only evidence as successful capacity telemetry collection;
5. implement distributed task leasing or remote command dispatch.

## Acceptance

1. Disabled-by-default behavior is test covered.
2. Enabled worker cycles record heartbeat and sweep result through `SupplyTelemetryAgent`.
3. Sweep failures are recorded as `last_sweep_status=failed`; skipped sweeps are recorded as `skipped`.
4. API-only startup invokes the worker hook without changing default behavior.
5. Focused Go tests and a real-process smoke on `aima2` prove the worker can collect `gb10-4t` telemetry and persist both agent and capacity evidence.

## Implementation Record

- Added `service.StartSupplyTelemetryWorker`, disabled by default and gated by `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED=true` plus `common.IsMasterNode`.
- Added environment parsing for interval, agent key, runtime ref, supplier/channel/node/model filters, and optional period window.
- Added worker cycle logic that records `SupplyTelemetryAgent` heartbeat, runs `model.SweepSupplyCapacityTelemetry`, then records `ok` / `skipped` / `failed` sweep summaries.
- Wired the worker hook into both `cmd/token-router-api` and the main server entrypoint; default startup behavior is unchanged.
- Validation passed on `aima2`: `go test ./service -run "SupplyTelemetryWorker|TaskBilling" -count=1`, `go test ./model ./service ./controller ./router ./cmd/token-router-api ./cmd/token-router-supply -run "SupplyTelemetry|SupplyCapacity|TelemetryWorker|TelemetryAgent|TelemetrySweep" -count=1`, `go test ./model ./service ./controller ./router ./cmd/token-router-sim ./cmd/token-router-supply ./cmd/token-router-api ./tests/e2e -count=1`, `go vet ./model ./service ./controller ./router ./cmd/token-router-api ./cmd/token-router-sim ./cmd/token-router-supply ./tests/e2e`, and builds for `token-router-api`, `token-router-sim`, and `token-router-supply`.
- Real process smoke passed on `aima2` at `/tmp/token-router-api-worker-20260623094819-2544199`: mock `gb10-4t` + seed + API-only server with `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED=true` collected telemetry without an external agent. DB readback showed `agent_key=api-server:aima2:adr0071`, `last_sweep_status=ok`, `attempted=1`, `collected=1`, `skipped=0`, and telemetry/capacity evidence from `source_ref=gb10-4t-mock-capacity`.
- `go test ./... -count=1` still only fails at the root package because legacy `web/classic/dist` embed assets are absent; all other packages pass.
