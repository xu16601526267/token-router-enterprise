# ADR 0068: prepaid supply lot drawdown evidence

- Status: Accepted
- Date: 2026-06-23
- Related: 0001 new-api baseline; 0020 supply action plans; 0024 supply action executions; 0060 self-hosted cost basis evidence; 0062 ledger-backed supply action execution drawdown

## Context

Product principle P8 says platform software must not become a payment, wallet, payout, invoice, or real funds system. Product principle P9 and the three-track supply model still require self-operated prepaid procurement to be auditable: operator needs to record that the company has purchased a batch of token capacity offline, track how much business usage has consumed it, and show remaining inventory / cost basis without treating the platform as the place where money moves.

`SupplyActionExecution` already records operator execution facts and can refresh execution-level drawdown from `UsageLedger`. `SupplyCostProfile` records self-hosted amortized cost basis. The remaining gap is a higher-level self-operated prepaid lot: a durable evidence row for an offline procurement batch, linked to supplier/node/model/period/source ref, whose token drawdown is refreshed from successful ledgers.

## Decision

Add `SupplyPrepaidLot` as a data-support read model for offline prepaid procurement:

1. `POST /api/supply_prepaid_lots/record` records or updates an offline prepaid lot by a stable `prepaid_lot_key`.
   - Required: `supplier_id`, `period_start`, `period_end`, `purchased_tokens`, `unit_cost_quota`, `source_ref`, `observed_at`.
   - Optional: `supply_node`, `model_name`, `channel_id`, `external_ref`, `notes`, `source_type`.
   - `total_cost_quota = purchased_tokens * unit_cost_quota`.
   - The supplier must be `self_operated`.
2. `GET /api/supply_prepaid_lots` returns lots with purchased, drawdown, remaining, drawdown rate, source evidence, and recorded metadata.
3. `POST /api/supply_prepaid_lots/refresh_usage` recomputes drawdown from successful `UsageLedger` rows matching:
   - `supplier_id`
   - optional `channel_id`
   - optional `supply_node`
   - optional `model_name`
   - lot period window
4. Refresh stores:
   - `drawdown_tokens`
   - `drawdown_request_count`
   - `remaining_tokens`
   - `drawdown_rate`
   - `drawdown_source_type=usage_ledger`
   - `drawdown_source_ref=usage_ledger:prepaid_lot:<id>:<period_start>:<period_end>`
   - `drawdown_refreshed_at`
5. Regenerating or refreshing evidence is idempotent. Re-recording a lot by the same key updates procurement evidence but preserves the same row identity.

## Non-goals

1. Do not create payments, payouts, invoices, purchase orders, approvals, bank fields, wallet balance, or cash status.
2. Do not automatically create supplier/channel/capacity rows.
3. Do not mutate routing policy, channel weights, pricing, billing, settlement statements, or user quota.
4. Do not interpret a prepaid lot as proof that funds were actually transferred. It is operator/accounting evidence only.
5. Do not implement automatic procurement, background scheduler, or fleet agent in this ADR.

## Consequences

- Self-operated supply now has a minimal procurement inventory / fund-drawdown evidence object without violating the no-funds-in-platform boundary.
- Operators and agents can compare purchased tokens, consumed tokens, remaining tokens, and unit cost basis for a prepaid batch.
- Ledger-backed drawdown reuses the same cache-aware business facts as settlement and execution drawdown, so cost/inventory evidence stays anchored to accepted traffic.
- Future dashboard work can expose prepaid lots alongside executions and cost profiles without changing the backend contract.

## Validation

1. Model tests cover record/update validation, self-operated supplier enforcement, query filters, and usage-ledger refresh matching.
2. HTTP e2e covers record, query, refresh, and rejection for non-self-operated suppliers.
3. `token-router-sim run` records a self-operated `gb10-4t` prepaid lot, refreshes it after demand traffic, and outputs `supply_prepaid_lot_verified=true`.
4. Focused Go tests, vet/build, and a real process run pass on `aima2`; docs record evidence and preserve payment/procurement boundary.
