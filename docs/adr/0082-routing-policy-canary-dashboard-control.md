# ADR 0082: Routing policy canary dashboard control

## Status

Accepted

## Context

ADR 0081 added the runtime primitive for self-hosted routing canaries: `SupplyRoutingPolicy.traffic_percent` and activation-time validation. The default remains `100`, and real-process proof shows deterministic included / excluded sessions.

The operator dashboard still activates a routing policy with a one-click button and no traffic-share input. That means the bounded rollout primitive exists in the API, but the human approval surface cannot choose or review the canary percent. This weakens the P5/P7 boundary: the operator can activate, but not with the same bounded control described by the runtime ADR.

## Decision

Extend the `/token-router` dashboard routing tab:

- Add `traffic_percent` to the frontend `SupplyRoutingPolicy` and activation input types.
- Replace one-click routing policy activation with a small dialog for recorded self-hosted executions.
- Default the dialog to `100`, allow only `1..100`, and submit `traffic_percent` to `/api/supply_routing_policies/activate`.
- Show the active policy traffic share in the policy table and in the execution source row.

## Boundaries

This ADR does not change backend routing semantics, does not add automatic promotion / rollback, and does not let the agent activate routing policies. It does not modify suppliers, channels, capacity, pricing, billing, settlement, wallet, payout, invoice, or funds state.

## Consequences

- Operators can use the ADR 0081 bounded rollout primitive from the existing dashboard.
- Policy rows remain auditable: a later reviewer can see whether the policy was hard override (`100%`) or canary (`1..99%`).
- The dashboard still requires explicit human activation from a recorded self-hosted execution.

## Verification

Completed locally:

- `cd web/default && node scripts/sync-i18n.mjs`
- i18n sync report: en / zh / fr / ja / ru / vi all have `missingCount=0`, `extrasCount=0`, `untranslatedCount=0`
- `cd web/default && npm run typecheck`
- `cd web/default && ../node_modules/.bin/oxfmt --check src/features/token-router/index.tsx src/features/token-router/types.ts src/i18n/locales/en.json src/i18n/locales/zh.json src/i18n/locales/fr.json src/i18n/locales/ja.json src/i18n/locales/ru.json src/i18n/locales/vi.json`
- `cd web/default && ../node_modules/.bin/oxlint -c .oxlintrc.json src/features/token-router/index.tsx src/features/token-router/types.ts`
- `cd web/default && npm run build`
- Playwright WebKit route mock against `http://127.0.0.1:31082/token-router`: opened `Routing Policies`, clicked a recorded self-hosted execution source, verified the dialog defaulted `Traffic Percent` to `100`, changed it to `25`, submitted, and confirmed POST #45 body was `{"supply_action_execution_id":801,"traffic_percent":25,"operator_note":"activated from dashboard"}`. The refreshed UI showed two `25%` labels and no remaining activate button for that source execution.

Known local tooling note: `npm run format:check` currently fails before checking code because `scripts/format-with-protected-headers.mjs` invokes `oxfmt --ignore-path .gitignore`, but this checkout has no `web/default/.gitignore`. The focused `oxfmt --check` command above passed for all touched frontend files.
