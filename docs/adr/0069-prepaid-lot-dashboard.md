# ADR 0069: prepaid lot dashboard

- Date: 2026-06-23
- Status: Accepted
- Related principles: P2 data before commitment; P5 human review; P8 platform boundary; P9 traffic as intelligence
- Related ADRs: 0024 supply action executions; 0025 supply action execution dashboard; 0061 self-hosted cost profile dashboard; 0068 prepaid supply lot drawdown

## Context

ADR 0068 added `SupplyPrepaidLot` and admin APIs for self-operated prepaid procurement evidence:

1. record or update an offline prepaid token lot;
2. query lots by supplier/node/model/source/period;
3. refresh drawdown from successful `UsageLedger` rows.

The backend is validated, but operators still need direct API calls to record or inspect prepaid lots. That leaves the self-operated supply track weaker than self-hosted cost profiles and execution drawdown in the default `/token-router` console.

## Decision

Add a `Prepaid Lots` work surface to the default token-router admin dashboard:

1. Query `GET /api/supply_prepaid_lots` with the global period filter.
2. Summarize visible lot count, purchased tokens, drawdown tokens, remaining tokens, and average drawdown rate.
3. Show lot rows with supplier, optional channel, node, model, period, purchased/remaining tokens, total/unit cost, source evidence, recorded metadata, and drawdown source.
4. Provide a `Record Prepaid Lot` dialog for self-operated suppliers only, posting to `/api/supply_prepaid_lots/record`.
5. Provide `Refresh Drawdown`, posting to `/api/supply_prepaid_lots/refresh_usage` with the global period filter, then refreshing the list.

## Boundaries

This dashboard must not:

1. create payments, payouts, invoices, purchase orders, bank records, wallet balances, or real-funds status;
2. approve procurement or imply that funds moved;
3. create or mutate suppliers, channels, capacity snapshots, pricing, user quota, billing, settlement, or routing policies;
4. treat prepaid lot evidence as SLA evidence or available-capacity proof;
5. run background refreshes or automatic procurement.

## Acceptance

1. TypeScript API/types match backend JSON field names.
2. `/token-router` exposes a `Prepaid Lots` tab that can query, record, and refresh drawdown with explicit operator action.
3. en/zh/fr/ja/ru/vi translations are complete after `i18n:sync`.
4. Frontend typecheck/build and a UI route-mock smoke pass verify the list, record payload, and refresh action.
5. README / architecture / product principles / traffic docs mention the dashboard and preserve the no-funds/no-auto-execution boundary.

## Implementation Record

- Added `SupplyPrepaidLot` frontend types and API wrappers for `GET /api/supply_prepaid_lots`, `POST /api/supply_prepaid_lots/record`, and `POST /api/supply_prepaid_lots/refresh_usage`.
- Added the default `/token-router` `Prepaid Lots` tab with summary cards, a self-operated-only `Record Prepaid Lot` dialog, and explicit `Refresh Drawdown`.
- Added en/zh/fr/ja/ru/vi translations and ran `npx --yes bun@1.3.14 run i18n:sync`; `_sync-report.json` shows missing / extras / untranslated all 0.
- Validation passed: `npx --yes bun@1.3.14 run typecheck`, targeted `oxfmt --check`, targeted `oxlint`, and `npx --yes bun@1.3.14 run build`.
- Playwright managed Chromium route mock verified the tab, record POST payload, refresh POST payload, desktop layout, and mobile layout. Artifacts: `output/playwright/adr0069-prepaid-lot-dashboard-check.js`, `output/playwright/adr0069-prepaid-lot-dashboard.png`, `output/playwright/adr0069-prepaid-lot-dashboard-mobile.png`, and `output/playwright/adr0069-prepaid-lot-dashboard-mobile-table.png`.
