# token-router completion audit

Date: 2026-06-23

## Current accepted scope

Accepted scope is the repo-native token-router data and control plane described
by `README.md`, `docs/architecture.md`, `docs/product-principles.md`, and
`docs/traffic-and-supply.md`:

- cache-aware usage ledger and settlement data
- session affinity and upstream session propagation
- gb10-4t supply + demand process simulator
- supply telemetry evidence and deployment-side telemetry/review runners
- supplier scorecards, evaluations, posture recommendations, and route
  preference overlays
- self-operated prepaid lot and self-hosted cost / execution drawdown read
  models
- self-hosted routing policy with passed runtime SLA evidence and deterministic
  `traffic_percent` canary
- default `/token-router` dashboard controls where humans approve, apply,
  activate, disable, acknowledge, dismiss, or record evidence
- systemd templates plus one-command gb10 process smoke runner

This audit accepts the current scope as proven for local real-process gb10 smoke
and repo-native control-plane behavior. It does not claim production rollout,
autonomous operations, or long-run real-hardware economics.

## Evidence matrix

| Obligation | Current implementation | Accepted proof |
|---|---|---|
| Platform is data/control plane, not funds platform | `Supplier`, `SupplierAgreement`, `UsageLedger`, `SettlementStatement`, prepaid lot and execution drawdown read models avoid payment / wallet / payout / invoice state | Product and architecture docs preserve no-funds boundary; smoke proof exercises ledger, settlement, prepaid lot, execution drawdown without funds mutation |
| Cache-aware usage and session affinity | `RecordUsage`, `UsageLedger`, `service/channel_affinity.go`, upstream `X-Session-Id` / `session_id` propagation, router-assigned `trsess_...` fallback | ADR0085 smoke demand log: `process e2e ok: ledgers=4`, `cached_tokens_verified=true`, `margin_verified=true`, `assigned_session_verified=true` |
| gb10 supply + demand real process path | `cmd/token-router-sim mock-supply|seed|run`, `cmd/token-router-api` | `deploy/smoke/token-router-gb10-process-smoke.sh` on `aima2`, evidence path `/tmp/token-router-gb10-process-smoke-20260623142209-2472155`, durable snippets in `docs/evidence/adr0085-gb10-process-smoke/` |
| Supply telemetry and capacity risk evidence | `SupplyCapacity`, `SupplyCapacityTelemetry`, collect/sweep APIs, `token-router-supply telemetry sweep|agent`, API worker | ADR0085 smoke demand log verifies `capacity_telemetry_sweep_verified=true` and `capacity_telemetry_insight_verified=true`; earlier ADR0070/0071 evidence covers agent/API-worker surfaces |
| Review agent refreshes agent-readable read models | `token-router-supply review once|agent` calls scorecard, posture, profile, forecast, pricing, decision, opportunity, insight generate APIs | ADR0085 review log: `status=ok`, `total_generated=12`, step counts `3/3/1/1/1/1/1/1` |
| Supplier posture remains human-gated | `SupplierPostureRecommendation` generate/query/approve/reject/apply and `SupplierRoutePreference` overlay | ADR0085 smoke demand log verifies `supplier_posture_verified=true` and `supplier_route_preference_verified=true`; product docs keep automatic tuning out of scope |
| Self-hosted policy is runtime-SLA-gated and canaried | `SupplyRoutingPolicy` activation requires passed runtime `SlaProbeRun`; `traffic_percent=1..100` deterministic session canary | ADR0085 smoke demand log verifies `routing_sla_evidence_verified=true`, `supply_routing_policy_canary_verified=true`, and `policy_miss_insight_verified=true` |
| Dashboard is a human operations surface | `/token-router` tabs expose generate/review/apply/activate/disable/record controls and evidence views | Frontend validations are recorded in README and relevant ADRs; audit accepts dashboard proof as UI/control proof, not as autonomous execution proof |
| Deployment shape is reproducible | `deploy/systemd/` unit/timer templates and `deploy/smoke/token-router-gb10-process-smoke.sh` | ADR0084: `systemd-analyze verify` syntax pass. ADR0085: live local process smoke pass. These are distinct proof levels |

## Latest smoke evidence

Source checkout on `aima2`:

```text
/tmp/token-router-adr0085-smoke-src
```

Run command:

```bash
bash -n deploy/smoke/token-router-gb10-process-smoke.sh
deploy/smoke/token-router-gb10-process-smoke.sh --node-name aima2
```

Runtime evidence directory:

```text
/tmp/token-router-gb10-process-smoke-20260623142209-2472155
```

Durable snippets:

- `docs/evidence/adr0085-gb10-process-smoke/summary.txt`
- `docs/evidence/adr0085-gb10-process-smoke/demand-sim.log`
- `docs/evidence/adr0085-gb10-process-smoke/review-agent.json`

## Explicit non-goals

These are not accepted as complete by this audit:

- automatic agent approval, apply, activate, disable, tune, promote, rollback,
  complete, acknowledge, or dismiss behavior
- production systemd installation on a dedicated server with real secrets
- long-run real hardware validation of cache ownership, capacity saturation,
  cost amortization, or SLA economics
- complex ML / external-data forecasting beyond the current deterministic and
  seasonal/anomaly evidence paths
- replacing dashboard human review with autonomous execution
- treating root-package `go test ./...` as required in source-only smoke copies
  that intentionally omit generated frontend dist artifacts

## Completion statement

The current route is complete for the accepted repo-native control-plane scope:
gb10 supply + demand, cache-aware ledger, telemetry, review, posture, canary
routing, policy miss, assigned-session behavior, and deployment smoke are all
covered by ADR-backed implementation and real-process evidence.

The route remains intentionally open for production rollout and autonomous
operations. Any change to those boundaries requires a new ADR with admission
trigger, rollback semantics, audit surface, and a proof that is stronger than
smoke-only evidence.
