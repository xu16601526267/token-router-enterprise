# ADR 0081: Self-hosted routing canary percent

## Status

Accepted

## Context

`SupplyRoutingPolicy` already lets an operator activate a self-hosted channel for a model / SLA / user slice after a recorded self-hosted execution and passed runtime SLA evidence. That closed the safety gate, but the live routing behavior is still binary: an active policy captures every matching request until the operator disables it.

For T3 self-hosted supply, the safer operational posture is a human-approved canary. Operators need to activate a policy at a bounded traffic share, watch ledger / latency / quality evidence, and then re-activate at a larger share only if the runtime evidence remains acceptable. This must stay deterministic per session so cache locality is not harmed by random per-request sampling.

## Decision

Add `traffic_percent` to `SupplyRoutingPolicy` and `SupplyRoutingPolicyActivateInput`.

- `0` in the activation payload means the existing default: `100`.
- Non-zero values must be `1..100`.
- Policy resolution hashes a stable route key into buckets `1..100`; a policy is eligible only when the bucket is within `traffic_percent`.
- The route key uses session headers first (`X-Session-Id`, `session_id`, prompt-cache keys), then falls back to the relay request id.
- Existing policies and existing callers keep hard-override behavior because missing / zero `traffic_percent` is normalized to `100`.

Traffic excluded by the percent gate is not a policy miss and must not create `OperatingInsight`. It falls through to the existing normal channel selection path. True policy misses, such as disabled channel / disabled supplier / cannot serve model, continue to generate miss insight.

## Boundaries

This ADR does not add automatic promotion, automatic rollback, a scheduler, or agent write access to activation. It does not modify `Channel.weight`, `Ability.weight`, `SupplierRoutePreference`, pricing, billing, settlement, wallet, payout, invoice, or funds state.

The runtime SLA activation gate remains mandatory and unchanged.

## Consequences

- Operators can route self-hosted supply as a deterministic canary without sacrificing session affinity.
- Normal routing remains the fallback for traffic outside the canary bucket.
- Later ADRs can add dashboard controls or scheduled promotion checks on top of the same field, but this ADR only adds the runtime primitive and proof.

## Verification

Completed on `aima2`:

1. Focused tests, vet, and simulator build:

   ```bash
   go test ./model -run "TestSupplyRoutingPolicy" -count=1
   go test ./service -count=1
   go vet ./model ./service ./cmd/token-router-sim
   go build -o bin/token-router-sim ./cmd/token-router-sim
   ```

   Output:

   ```text
   ok   github.com/QuantumNous/new-api/model   0.119s
   ok   github.com/QuantumNous/new-api/service 0.098s
   ```

2. Real-process proof with fresh SQLite, wildcard strict session mock, memory-cache API-only server, and `token-router-sim run`.

   Evidence directory: `/tmp/token-router-adr0081-canary-process-20260623132054-1553189`.

   The simulator activated the self-hosted policy at `traffic_percent=50`, selected one session inside the deterministic bucket and one outside it, and printed:

   ```text
   process e2e ok: ledgers=4 session=session-process-e2e assigned_session=trsess_202606230521099862964838268d9d63qZz2v3l cached_tokens_verified=true margin_verified=true settlement_verified=true capacity_verified=true capacity_usage_refresh_verified=true capacity_telemetry_verified=true capacity_telemetry_collect_verified=true capacity_telemetry_sweep_verified=true capacity_telemetry_insight_verified=true supplier_scorecard_verified=true supplier_evaluation_verified=true supplier_posture_verified=true supplier_route_preference_verified=true sla_evidence_verified=false traffic_profile_verified=true traffic_forecast_verified=true traffic_forecast_seasonal_anomaly_verified=true pricing_recommendation_verified=true supply_decision_verified=true self_hosted_cost_profile_verified=true supply_prepaid_lot_verified=true supply_expansion_opportunity_verified=true operating_insight_verified=true supply_action_plan_verified=true supply_action_execution_verified=true supply_action_execution_drawdown_verified=true supply_routing_policy_verified=true supply_routing_policy_canary_verified=true routing_sla_evidence_verified=true policy_miss_insight_verified=true assigned_session_verified=true
   ```

   DB evidence read back from the same SQLite file:

   ```text
   policy (1, 1, 'disabled', 2, 3, 50, 1)
   ledger ('process-e2e-request-policy-canary-in', 'session-process-e2e-canary-true-000', 2, 3, 'gb10-4t-self-hosted', 140, 49)
   ledger ('process-e2e-request-policy-canary-out', 'session-process-e2e-canary-false-000', 1, 2, 'gb10-4t', 140, 70)
   ```

   The included session used the self-hosted supplier/channel; the excluded session fell back to normal routing.
