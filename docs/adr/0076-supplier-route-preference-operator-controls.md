# ADR 0076: Supplier route preference operator controls

## Status

Accepted

## Context

ADR 0074 made approved supplier posture `downgrade` apply create an active `SupplierRoutePreference` overlay, and ADR 0075 exposed active overlays in the `Posture` dashboard. That closes the recommendation-driven path, but the route preference itself is still not operator-configurable.

The product principles explicitly keep execution human-reviewed, and the architecture still lists operator-configurable route preference strategy as future work. The missing control is not automatic agent tuning; it is a bounded manual override for cases where an operator has evidence outside the current posture recommendation window and wants to temporarily reduce normal traffic reliance on one supplier without disabling the supplier or editing every channel.

## Decision

Add explicit admin controls for `SupplierRoutePreference`:

1. Add `POST /api/supplier_route_preferences/activate` to create or update the one current preference for a supplier.
2. Add `POST /api/supplier_route_preferences/:supplier_id/disable` to clear the active preference for that supplier.
3. Require an enabled supplier id, a route penalty `weight_percent` between `1` and `100`, a non-empty reason, and optional effective window / operator note on activate.
4. Treat manual preferences as source recommendation `0`; posture-driven preferences keep their source recommendation id.
5. Preserve audit fields: activated / disabled user, timestamps, reason, operator note, effective window.
6. Refresh the runtime channel cache after activate or disable so both memory-cache and DB fallback routing see the same current overlay.
7. Extend the existing `Posture` tab `Active Route Preferences` panel with `Set Route Preference` and `Disable` controls.

This is still a human-reviewed operational control. It must not mutate `Channel.weight`, `Ability.weight`, `SupplyRoutingPolicy`, pricing, billing, settlement, procurement, or funds state. It must not allow promotion above baseline weight in this increment; weight `100` means no penalty and `1..99` means reduce normal selection weight. Fully draining traffic remains a supplier disable / posture gate decision, not a manual route preference shortcut.

## Consequences

- Operators can apply a temporary supplier-level route penalty from the same surface where posture evidence is reviewed.
- Manual route preference changes remain auditable and reversible without losing posture-driven evidence semantics.
- Agent-suggested automatic tuning, boost multipliers above baseline, and policy-level strategy rules remain future ADRs because they need stronger admission rules and rollback semantics.

## Verification

Completed:

1. On `aima2`, ran `gofmt` and passed `go test ./model -run "TestActivateSupplierRoutePreference|TestManualSupplierRoutePreferenceRefreshesRuntimeChannelCache" -count=1`.
2. On `aima2`, passed `go test ./tests/e2e -run TestTokenRouterSupplierPostureRecommendationAPI -count=1 -timeout 2m`, covering posture-driven preference creation, manual activate with `source_posture_recommendation_id=0`, active query, and disable.
3. Locally passed frontend `i18n:sync`, `typecheck`, targeted `oxfmt`, targeted `oxlint`, and `build`. Full frontend lint still has unrelated pre-existing lint debt outside the token-router surface.
4. Playwright route mock passed against the `Posture` UI, proving manual activate payload, active preference rendering, disable payload, refetch behavior, and desktop/mobile layout. Screenshots: `output/playwright/adr0076-supplier-route-preference-operator-controls.png` and `output/playwright/adr0076-supplier-route-preference-operator-controls-mobile.png`.
