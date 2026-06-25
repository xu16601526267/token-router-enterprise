# ADR 0078: Supplier posture boost recommendations

## Status

Accepted

## Context

ADR 0077 allowed operators to manually set bounded supplier route preference boosts from `101` to `200`. That closed the manual control gap, but the agent-readable posture lane still only emits `observe`, `downgrade`, and `disable`.

Product principles P1 / P4 say high-performing suppliers should receive more traffic, while P5 keeps execution human-reviewed in the current operating mode. The next step should therefore be a recommendation, not automatic tuning: let the posture generator surface strong suppliers as boost candidates, then require explicit approve/apply before routing changes.

## Decision

Add a `boost` action to `SupplierPostureRecommendation`.

1. Generate `boost` only when the supplier is enabled, has grade `A`, score `>= 90`, nonzero request volume, and has no open quality / capacity / action posture insights for the period.
2. Keep existing negative thresholds first: severe evidence still generates `disable`, weak evidence still generates `downgrade`, and ordinary healthy evidence remains `observe`.
3. Applying an approved `boost` keeps `Supplier.status` unchanged and creates or updates an active `SupplierRoutePreference` with fixed `weight_percent=150` and `source_posture_recommendation_id=<recommendation id>`.
4. Applying `downgrade` remains fixed at `25`; applying `observe` or `disable` still clears an active route preference.
5. Extend the `Posture` dashboard action filter and action label so operators can find and apply boost recommendations from the same review surface.

This is still a human-reviewed control. Generate does not mutate routing. Approve/reject only records review. Apply is the only operation that can activate the route preference. The change must not mutate `Channel.weight`, `Ability.weight`, `SupplyRoutingPolicy`, pricing, billing, settlement, procurement, or funds state.

## Consequences

- P4 now has a positive scorecard-backed recommendation path as well as downgrade / disable paths.
- Strong suppliers can be promoted through the same auditable `SupplierRoutePreference` read model used by manual boost and posture downgrade.
- The boost threshold is intentionally stricter than grade `A` alone, keeping noisy scorecard rows in `observe`.
- Agent automatic tuning, dynamic boost percentages, and background apply remain future ADRs.

## Verification

Completed:

1. On `aima2`, ran `gofmt` and passed `go test ./model -run "TestApplySupplierPostureRecommendation|TestGenerateSupplierPostureRecommendations" -count=1`.
2. On `aima2`, passed `go test ./tests/e2e -run TestTokenRouterSupplierPostureRecommendationAPI -count=1 -timeout 2m`, covering generated `boost` recommendation approve/apply/query through admin APIs.
3. On `aima2`, passed `go test ./model -run "SupplierPosture|SupplierRoutePreference" -count=1`.
4. Locally passed frontend `i18n:sync` with missing / extras / untranslated all `0` for en / zh / fr / ja / ru / vi.
5. Locally passed frontend `typecheck`, targeted `oxfmt --check`, targeted `oxlint`, `build`, and `git diff --check`.
6. Playwright WebKit route mock `output/playwright/adr0078-supplier-posture-boost-recommendations-check.js` passed against local `rsbuild preview` on `127.0.0.1:4191`: it verified `Boost` action filtering sends `recommended_action=boost`, renders the boost row, and shows the active `150%` route preference badge. Screenshots: `output/playwright/adr0078-supplier-posture-boost-recommendations.png` and `output/playwright/adr0078-supplier-posture-boost-recommendations-mobile.png`.
