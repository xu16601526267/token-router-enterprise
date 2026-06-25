# ADR 0074: Supplier route preference overlay

## Status

Accepted

## Context

ADR 0072 added `SupplierPostureRecommendation` and ADR 0073 exposed it in the dashboard. `disable` recommendations already enter the runtime supplier gate after explicit approve/apply. `downgrade` recommendations, however, still only append audit notes because normal channel selection has no supplier-level route-weight primitive.

That leaves a P1/P4 gap: the platform can identify a degraded supplier, but a human-approved downgrade cannot yet reduce normal traffic reliance without manually editing each channel weight.

## Decision

Add a `SupplierRoutePreference` overlay:

1. Store one current route preference per supplier.
2. `active` preferences carry a `weight_percent` multiplier applied during normal channel selection.
3. Applying an approved posture `downgrade` creates or updates an active preference with a conservative multiplier.
4. Applying `observe` or `disable` clears any active preference for that supplier:
   - `observe` means no route penalty is currently requested;
   - `disable` is handled by the existing supplier runtime gate, so a stale route preference must not reappear when the supplier is later re-enabled.
5. Normal channel selection applies the preference as an overlay to candidate weights for both memory-cache and DB fallback paths.
6. The overlay does not mutate `Channel.weight`, `Ability.weight`, routing policy rows, pricing, billing, settlement, or funds state.

The first multiplier is intentionally simple: `downgrade` uses `25%` of the supplier's normal selection weight. Future ADRs can add operator-configurable multipliers or dashboard controls if real traffic evidence demands it.

## Consequences

- A human-approved `downgrade` becomes a real routing signal without disabling the supplier.
- Channel configuration remains the source of baseline capacity intent; route preference is reversible evidence layered above it.
- Existing self-hosted `SupplyRoutingPolicy` hard overrides still take precedence over normal channel selection.
- This is still dashboard/human-driven. No recommendation auto-applies and no background worker changes routing weights.

## Verification

Completed on `aima2` after remote `gofmt` because the local workstation has no `go` / `gofmt` binaries:

1. `go test ./model -run 'SupplierRoutePreference|SupplierPosture|RuntimeChannelCache|RandomSatisfiedChannel' -count=1`
2. `go test ./tests/e2e -run TestTokenRouterSupplierPostureRecommendationAPI -count=1`
3. `go test ./model ./controller ./router ./service ./tests/e2e -run 'SupplierRoutePreference|SupplierPosture|RuntimeChannelCache|SupplyRoutingPolicy|TokenRouterSupplierPostureRecommendationAPI' -count=1`
4. `go build ./model ./controller ./router ./service ./cmd/token-router-api ./cmd/token-router-sim ./cmd/token-router-supply ./cmd/token-router-sla`
5. `go vet ./model ./controller ./router ./service ./cmd/token-router-api ./cmd/token-router-sim ./cmd/token-router-supply ./cmd/token-router-sla ./tests/e2e`
6. Full package tests for `./model ./controller ./service ./cmd/token-router-supply ./cmd/token-router-sla` passed. Full `./tests/e2e` as a suite still hit existing SQLite `disk I/O error` / `database is locked` concurrency behavior, so the ADR evidence uses the focused posture HTTP E2E above.

The focused tests prove downgrade apply creates an active 25% preference, observe/disable clear active preferences, and both memory-cache and DB fallback channel selection apply the overlay without disabling the supplier.
