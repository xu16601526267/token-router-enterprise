# ADR 0073: Supplier posture dashboard

## Status

Accepted

## Context

ADR 0072 added the backend control plane for supplier runtime posture recommendations:

- `SupplierPostureRecommendation` stores scorecard + quality/capacity insight evidence.
- Admin APIs can generate, query, approve, reject, and apply recommendations.
- Applying an approved `disable` recommendation writes `Supplier.status` and enters the existing runtime supplier gate.
- Applying `downgrade` only appends an audit note because there is no supplier route-weight primitive yet.

Operators still need to leave the default `/token-router` dashboard to use that control plane. That weakens P5/P7: agent-readable recommendations exist, but the human review surface is incomplete.

## Decision

Add a `Posture` tab to the default `/token-router` dashboard beside `Scorecards` and `Evaluations`.

The tab will:

1. Query `GET /api/supplier_posture_recommendations` for the current global period.
2. Filter by status, recommended action, and scorecard grade.
3. Show summary cards for visible recommendations, draft reviews, disable recommendations, and applied recommendations.
4. Provide a `Generate Posture` action that calls `/api/supplier_posture_recommendations/generate` for the selected period.
5. For draft rows, provide explicit approve / reject buttons.
6. For approved rows, provide explicit apply.
7. Display review and apply audit fields, scorecard evidence, open insight counts, supplier status before/after, and reason text.

The dashboard must not:

- auto-generate recommendations on page load;
- auto-approve, auto-apply, or auto-disable suppliers;
- create supplier route weights;
- mutate channels, routing policies, pricing, billing, settlement, procurement, or funds state.

## Consequences

- P1/P4 runtime posture recommendations become operator-visible in the main dashboard.
- P5/P7 stays intact: agent output remains inert until a human clicks approve/apply.
- `disable` uses the existing supplier runtime gate; `downgrade` remains an audit-only action until a future ADR defines route weights.
- This is a frontend/API-client increment; it should not change backend recommendation semantics from ADR 0072.

## Verification

1. Frontend typecheck passes.
2. i18n sync reports no missing / extra / untranslated keys across supported locales.
3. Targeted lint/format checks cover the touched token-router frontend files.
4. Playwright route mocks verify:
   - the `Posture` tab renders recommendations and filters;
   - generate uses the current period payload;
   - approve / reject / apply call the expected endpoints;
   - applied audit and supplier status before/after are visible.

## Implementation Notes

- Implemented in `web/default/src/features/token-router/index.tsx`, `api.ts`, `types.ts`, and locale files under `web/default/src/i18n/locales/`.
- `Posture` queries `GET /api/supplier_posture_recommendations` with global period filters, status, recommended action, and grade. It calls `POST /api/supplier_posture_recommendations/generate`, `/:id/approve`, `/:id/reject`, and `/:id/apply` only from explicit operator buttons.
- Local validation passed:
  - `npx --yes bun@1.3.14 run i18n:sync`
  - `npx --yes bun@1.3.14 run typecheck`
  - `npx --yes bun@1.3.14 x oxfmt --check src/features/token-router/index.tsx src/features/token-router/api.ts src/features/token-router/types.ts`
  - `npx --yes bun@1.3.14 x oxlint -c .oxlintrc.json src/features/token-router/index.tsx src/features/token-router/api.ts src/features/token-router/types.ts`
  - `npx --yes bun@1.3.14 run build`
  - `git diff --check -- output/playwright/adr0073-supplier-posture-dashboard-check.js web/default/src/features/token-router/index.tsx web/default/src/features/token-router/api.ts web/default/src/features/token-router/types.ts web/default/src/i18n/locales docs/adr/0073-supplier-posture-dashboard.md`
- Playwright CLI route mock artifact: `output/playwright/adr0073-supplier-posture-dashboard-check.js`.
- Screenshots:
  - `output/playwright/adr0073-supplier-posture-dashboard.png`
  - `output/playwright/adr0073-supplier-posture-dashboard-mobile.png`
  - `output/playwright/adr0073-supplier-posture-dashboard-mobile-table.png`
