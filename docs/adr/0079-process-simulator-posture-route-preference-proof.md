# ADR 0079: Process simulator posture route preference proof

## Status

Accepted

## Context

ADR 0072 through ADR 0078 added supplier posture recommendations, route preference overlays, operator controls, bounded manual boost, and posture-driven boost recommendations. Unit tests, HTTP E2E, and dashboard route mocks prove those surfaces in isolation.

The real-process `token-router-sim run` path is the user-facing end-to-end proof for the project goal: gb10-4t mock supply, demand simulator requests, usage ledger, capacity, scorecard, forecast, decisions, action plans, execution, and routing policy evidence. It currently prints many `*_verified=true` markers, but it does not verify supplier posture recommendations or the resulting `SupplierRoutePreference` overlay.

That leaves a proof gap: the main simulator can prove a supplier has a strong gb10 scorecard, but it does not prove that the same live process can generate a posture `boost`, require human-style approve/apply, and read back the active `150%` route preference.

## Decision

Extend `token-router-sim run` with a posture route preference verification step.

1. Reuse the generated gb10 supplier scorecard from the existing simulator scorecard verification.
2. Generate supplier posture recommendations for the scorecard period.
3. Require the gb10 supplier recommendation to be `boost` for the strong process scorecard.
4. Query `GET /api/supplier_posture_recommendations` with `recommended_action=boost` to prove the filter surface.
5. Approve and apply the recommendation through admin APIs.
6. Query active `SupplierRoutePreference` for the supplier and require:
   - `status=active`
   - `source_posture_recommendation_id=<applied recommendation id>`
   - `weight_percent=150`
   - reason includes `boost`
7. Print `supplier_posture_verified=true` and `supplier_route_preference_verified=true` in the process simulator success line.

This is a process proof only. It must not add automatic apply, background tuning, channel or ability weight mutation, pricing, billing, settlement, procurement, or funds behavior.

## Consequences

- The gb10 process simulator now covers the same human-reviewed posture overlay path operators use from the dashboard.
- ADR 0078's positive supplier lane is tied into the end-to-end simulator chain, not only unit/API/UI proofs.
- The simulator remains deterministic because the seeded gb10 supply path already produces a grade A scorecard with no open posture insights.

## Verification

Completed on `aima2`:

1. Focused formatter/tests/builds against the synced scratch copy `/tmp/token-router-adr0079`:

   ```bash
   gofmt -w cmd/token-router-sim/main.go
   go test ./cmd/token-router-sim -count=1
   go test ./model -run "SupplierPosture|SupplierRoutePreference|SupplierScorecard|SupplyDecision|SupplyExpansionOpportunity" -count=1
   go build -o bin/token-router-api ./cmd/token-router-api
   go build -o bin/token-router-sim ./cmd/token-router-sim
   ```

   Output:

   ```text
   ?    github.com/QuantumNous/new-api/cmd/token-router-sim [no test files]
   ok   github.com/QuantumNous/new-api/model 0.429s
   ```

2. Real-process simulator proof with strict session mock supply, fresh SQLite DB, seeded gb10 supply, memory-cache API-only server, and `token-router-sim run`.

   Evidence directory: `/tmp/token-router-adr0079-process-20260623125558-1174877`.

   Final simulator line:

   ```text
   process e2e ok: ledgers=5 session=session-process-e2e assigned_session=trsess_202606230456137349242898268d9d6owzy2oJh cached_tokens_verified=true margin_verified=true settlement_verified=true capacity_verified=true capacity_usage_refresh_verified=true capacity_telemetry_verified=true capacity_telemetry_collect_verified=true capacity_telemetry_sweep_verified=true capacity_telemetry_insight_verified=true supplier_scorecard_verified=true supplier_evaluation_verified=true supplier_posture_verified=true supplier_route_preference_verified=true sla_evidence_verified=false traffic_profile_verified=true traffic_forecast_verified=true traffic_forecast_seasonal_anomaly_verified=true pricing_recommendation_verified=true supply_decision_verified=true self_hosted_cost_profile_verified=true supply_prepaid_lot_verified=true supply_expansion_opportunity_verified=true operating_insight_verified=true supply_action_plan_verified=true supply_action_execution_verified=true supply_action_execution_drawdown_verified=true supply_routing_policy_verified=true routing_sla_evidence_verified=true policy_miss_insight_verified=true assigned_session_verified=true
   ```

3. Read-only evidence DB check:

   ```text
   supplier_scorecards: (id=1, supplier_id=1, total_requests=3, score=92.999, grade=A, cache_hit_rate=0.667, gross_profit_quota=162)
   supplier_posture_recommendations: (id=1, supplier_id=1, supplier_scorecard_id=1, status=applied, recommended_action=boost, score=92.999, grade=A, total_requests=3, supplier_status_before=1, supplier_status_after=1, reviewed_by=1, applied_by=1)
   supplier_route_preferences: (id=1, supplier_id=1, source_posture_recommendation_id=1, status=active, weight_percent=150, activated_by=1, reason="supplier_posture_recommendation #1 boost: grade=A score=92.999", operator_note="applied posture boost in process e2e")
   ```

During verification, the added third scorecard request also exercised downstream process assertions. The simulator assertions for pricing recommendation, supply decision, expansion opportunity, and operating insight now derive expected unit price / ROI / rank values from the actual ledgers, profile, forecast, decision, and recommendation instead of the old two-request constants. The posture apply step has a bounded retry only for transient SQLite `database is locked` / `SQLITE_BUSY` responses in this real-process smoke.
