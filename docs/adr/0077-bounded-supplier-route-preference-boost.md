# ADR 0077: Bounded supplier route preference boost

## Status

Accepted

## Context

ADR 0074 introduced supplier-level route preference overlays for posture-driven downgrades. ADR 0076 added manual activate / disable controls, but intentionally capped manual `weight_percent` at `100`, so operators could only reduce or restore normal channel selection weight.

That leaves the positive half of P4 incomplete: good suppliers should be able to receive more traffic after human review. Jumping straight to agent automatic tuning would be too broad for the current human-reviewed operating model, but a bounded manual boost is a small, auditable step.

## Decision

Allow manual `SupplierRoutePreference.weight_percent` from `1` to `200`.

1. Keep `100` as baseline, no preference effect.
2. Keep `1..99` as a penalty that reduces normal channel selection weight.
3. Add `101..200` as a bounded boost that increases normal channel selection weight.
4. Apply the same multiplier in memory-cache channel selection and DB fallback channel selection.
5. Keep posture-driven downgrade fixed at `25`; posture recommendations do not generate boosts in this increment.
6. Keep manual activate constraints from ADR 0076: enabled supplier only, non-empty reason, optional effective window / operator note, audit fields, and runtime channel cache refresh.
7. Update the `Posture` dashboard route preference form copy and input cap to make the `1..200` range explicit.

This remains a human-reviewed operational control. It still must not mutate `Channel.weight`, `Ability.weight`, `SupplyRoutingPolicy`, pricing, billing, settlement, procurement, or funds state. Agent automatic boosts, scorecard-to-boost policy, and caps above `200` require a separate ADR.

## Consequences

- Operators can temporarily reward a supplier with stronger normal-routing preference without editing every channel.
- The existing route preference read model remains the single audit trail for penalties, baseline restores, and boosts.
- Because this is still a multiplier over existing channel / ability weights, existing hard gates and self-hosted routing policy precedence remain unchanged.

## Verification

Completed:

1. On `aima2`, ran `gofmt` and passed `go test ./model -run "TestSupplierRoutePreferenceSelectionWeight|TestActivateSupplierRoutePreference|TestGetRandomSatisfiedChannelAppliesSupplierRoutePreferenceBoost|TestManualSupplierRoutePreferenceRefreshesRuntimeChannelCache" -count=1`.
2. On `aima2`, passed `go test ./tests/e2e -run TestTokenRouterSupplierPostureRecommendationAPI -count=1 -timeout 2m`, covering posture-driven preference creation, manual `150%` boost activate/query, and disable.
3. Locally passed frontend `i18n:sync`, `typecheck`, targeted `oxlint`, and `build`; i18n sync report shows missing / extras / untranslated are all `0` for en / zh / fr / ja / ru / vi.
4. Playwright route mock passed against the `Posture` UI, proving the `200` input cap, `150` boost payload, rendered active preference, disable payload, refetch behavior, and desktop/mobile layout. Screenshots: `output/playwright/adr0077-bounded-supplier-route-preference-boost.png` and `output/playwright/adr0077-bounded-supplier-route-preference-boost-mobile.png`.
