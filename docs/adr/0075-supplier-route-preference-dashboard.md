# ADR 0075: Supplier route preference dashboard visibility

## Status

Accepted

## Context

ADR 0074 made approved supplier posture `downgrade` apply create a real `SupplierRoutePreference` overlay. The route preference is now queryable through `/api/supplier_route_preferences` and affects normal channel selection, but the default `/token-router` dashboard still only shows the posture recommendation rows.

That leaves an operator visibility gap: after clicking `Apply` on a downgrade, the human review surface cannot show which supplier currently has an active route penalty, which recommendation created it, or whether `observe` / `disable` cleared it.

## Decision

Extend the existing `Posture` tab with read-only route preference visibility:

1. Query `GET /api/supplier_route_preferences` for active preferences.
2. Add a summary metric for active route preferences.
3. Add a compact `Active Route Preferences` panel in the `Posture` tab showing supplier id, source recommendation id, weight percent, status, effective window, operator note, and reason.
4. Mark posture rows whose recommendation has an active route preference, so the recommendation evidence and the live routing overlay can be correlated.
5. Refresh route preferences after posture apply, approve, reject, or generate operations using normal React Query invalidation.

This increment is read-only. It must not add manual weight editing, auto-apply route preferences, mutate channel or ability weights, change `SupplyRoutingPolicy`, or touch pricing, billing, settlement, procurement, or funds state.

## Consequences

- Operators can verify the runtime consequence of an approved downgrade from the same review surface.
- The dashboard distinguishes `disable` runtime gate evidence from `downgrade` route preference evidence.
- Future operator-configurable multipliers remain a separate ADR because they need admission rules, audit policy, and rollback semantics.

## Verification

1. `npx --yes bun@1.3.14 run i18n:sync` passed; `src/i18n/locales/_reports/_sync-report.json` reports missing / extras / untranslated = 0 for en / zh / fr / ja / ru / vi.
2. Direct token-router i18n key scan passed: `All t() keys found in en.json!`.
3. `npx --yes bun@1.3.14 run typecheck` passed.
4. `npx --yes bun@1.3.14 x oxfmt --check src/features/token-router/index.tsx src/features/token-router/api.ts src/features/token-router/types.ts` passed.
5. `npx --yes bun@1.3.14 x oxlint -c .oxlintrc.json src/features/token-router/index.tsx src/features/token-router/api.ts src/features/token-router/types.ts` passed.
6. `npx --yes bun@1.3.14 run build` passed.
7. Playwright managed Chromium route mock `output/playwright/adr0075-supplier-route-preference-dashboard-check.js` passed against the local `rsbuild preview` server on `127.0.0.1:4190`: it verified initial empty active preferences, `GET /api/supplier_route_preferences?status=active`, `POST /api/supplier_posture_recommendations/72/apply`, route preference refetch after apply, row badge, summary metric, active preference panel, and desktop/mobile layout. Screenshots: `output/playwright/adr0075-supplier-route-preference-dashboard.png`, `output/playwright/adr0075-supplier-route-preference-dashboard-mobile.png`.
