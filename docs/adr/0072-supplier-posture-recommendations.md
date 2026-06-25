# ADR 0072: Supplier posture recommendations

## Status

Accepted

## Context

Product principles P1 / P4 say low-quality upstream suppliers should be downgraded and eventually eliminated. The platform already has `SupplierScorecard`, SLA-gated `SupplierEvaluation`, `OperatingInsight`, and runtime supplier status gating, but the downgrade/elimination control surface is still incomplete:

- admission evaluation is focused on onboarding / periodic qualification;
- operating insights expose quality and capacity risks, but do not create supplier-level posture decisions;
- changing supplier runtime status still requires manual edits outside a dedicated evidence trail.

We still must preserve the P5 / P8 boundary: agent-generated evidence can recommend action, but the platform must not automatically mutate supplier posture, route weights, pricing, procurement, billing, settlement, or funds state.

## Decision

Add a `SupplierPostureRecommendation` read/review model.

1. Generation reads existing evidence for a supplier-period:
   - the latest matching `SupplierScorecard`;
   - open `OperatingInsight` rows in quality-watch / capacity-risk categories;
   - current `Supplier.status`.
2. Generation writes deterministic draft recommendations:
   - `observe`: supplier can remain under normal monitoring;
   - `downgrade`: evidence is weak enough that an operator should reduce reliance outside this table, but no automatic route-weight mutation exists yet;
   - `disable`: evidence is severe enough that an operator should consider runtime elimination.
3. Recommendations are keyed by scorecard and period, then upserted without overwriting review/apply fields.
4. Add admin APIs:
   - `GET /api/supplier_posture_recommendations`
   - `POST /api/supplier_posture_recommendations/generate`
   - `POST /api/supplier_posture_recommendations/:id/approve`
   - `POST /api/supplier_posture_recommendations/:id/reject`
   - `POST /api/supplier_posture_recommendations/:id/apply`
5. Applying is allowed only after approval:
   - `disable` sets `Supplier.status` to manually disabled and refreshes channel cache;
   - `downgrade` and `observe` append an audit note but keep supplier status unchanged.

This gives operators a supplier-level posture lane without introducing hidden automation.

## Consequences

- P1 / P4 now have a concrete downgrade/elimination recommendation surface tied to scorecard and insight evidence.
- Existing runtime routing gates immediately respect an approved `disable` apply because they already filter disabled suppliers.
- The first version does not implement supplier weights or automatic traffic redistribution. `downgrade` remains a reviewed audit action until a later ADR defines a routing-weight primitive and its safety checks.
- Automatic execution remains out of scope; future work must explicitly define admission triggers, rollback behavior, and audit requirements before any autonomous posture mutation.

## Verification

1. Model tests cover generate, preserve review/apply on regeneration, approve/reject validation, and approved disable apply.
2. API/router tests or E2E helpers cover generate/query/review/apply through admin endpoints.
3. Targeted `go test` / `go vet` / build checks run on `aima2`.
4. If feasible, a real-process smoke records a poor supplier scorecard / insight, generates a `disable` recommendation, approves it, applies it, and verifies supplier status plus route cache behavior.

## Implementation Notes

- Added `SupplierPostureRecommendation` with deterministic scorecard + open insight generation, review/apply audit fields, and status/action filters.
- Added admin APIs at `/api/supplier_posture_recommendations`.
- Added model tests for `disable`, `downgrade`, regeneration preservation, and runtime channel cache refresh after approved disable apply.
- Added `TestTokenRouterSupplierPostureRecommendationAPI` to cover the admin HTTP lifecycle.
- `aima2` validation passed:
  - `go test ./model -run "SupplierPosture|RuntimeChannelCache" -count=1`
  - `go test ./tests/e2e -run TestTokenRouterSupplierPostureRecommendationAPI -count=1`
  - `go test ./model ./controller ./router ./tests/e2e -run "SupplierPosture|SupplierScorecard|SupplierEvaluation|OperatingInsight|RuntimeChannelCache" -count=1`
  - `go test ./model ./service ./controller ./router ./cmd/token-router-api ./cmd/token-router-sim ./cmd/token-router-supply ./tests/e2e -count=1`
  - `go vet ./model ./service ./controller ./router ./cmd/token-router-api ./cmd/token-router-sim ./cmd/token-router-supply ./tests/e2e`
  - `go build -o bin/token-router-api ./cmd/token-router-api`
  - `go build -o bin/token-router-sim ./cmd/token-router-sim`
  - `go build -o bin/token-router-supply ./cmd/token-router-supply`
- Real-process smoke passed on `aima2` at `/tmp/token-router-posture-smoke-20260623100802-2847587`: seeded SQLite with `token-router-sim seed`, inserted a low `SupplierScorecard` and capacity-risk `OperatingInsight`, started `token-router-api`, generated `recommended_action=disable`, approved and applied recommendation `id=1`, and verified `/api/suppliers/1` returned manually disabled status with `supplier_posture_recommendation #` in notes.
