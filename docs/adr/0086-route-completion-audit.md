# ADR 0086: Route completion audit

## Status

Accepted

## Context

The token-router route has accumulated many ADRs and proofs. The current docs
now describe a broad implemented surface: cache-aware ledger, session affinity,
settlement data, supply telemetry, scorecards, posture recommendations, route
preference overlays, self-hosted routing policy canaries, review runners,
systemd templates, and a gb10 process smoke runner.

At this point the highest-risk failure is no longer one missing endpoint. It is
claiming the wrong kind of completion: treating smoke-only, syntax-only,
dashboard-only, or temp-dir-only evidence as a production runtime guarantee, or
accidentally pulling future autonomous execution work into the current accepted
scope.

The route needs one repo-native audit that maps product / architecture
obligations to concrete evidence and names the remaining intentional boundaries.

## Decision

Add `docs/completion-audit.md`.

The audit will:

- define the current accepted scope
- map each major obligation to concrete implementation and evidence
- distinguish accepted runtime proof from dashboard-only, syntax-only, or
  deployment-template proof
- list explicit non-goals and future work
- point at the latest ADR0085 smoke evidence snippets under
  `docs/evidence/adr0085-gb10-process-smoke/`

Add the latest small smoke outputs as durable evidence snippets:

- `docs/evidence/adr0085-gb10-process-smoke/summary.txt`
- `docs/evidence/adr0085-gb10-process-smoke/demand-sim.log`
- `docs/evidence/adr0085-gb10-process-smoke/review-agent.json`

## Boundaries

This ADR does not implement product behavior. It must not:

- mark production rollout complete
- treat systemd unit syntax verification as live service success
- treat review generation as automatic approval / apply / activate / disable
- turn future automatic tuning, promotion, rollback, or execution into current
  scope
- claim long-run real-hardware cache ownership or cost amortization beyond the
  current gb10 process proof

## Consequences

- Future work can start from one evidence matrix instead of re-reading dozens of
  ADRs.
- The current completion claim is precise: accepted for the repo-native
  data/control-plane proof and gb10 process smoke; not accepted for autonomous
  operations or production rollout.
- Follow-up ADRs must update the audit when they change an accepted boundary.

## Verification

Completed locally:

1. `bash -n deploy/smoke/token-router-gb10-process-smoke.sh`
2. Path check for:
   - `docs/completion-audit.md`
   - `docs/adr/0086-route-completion-audit.md`
   - `docs/evidence/adr0085-gb10-process-smoke/summary.txt`
   - `docs/evidence/adr0085-gb10-process-smoke/demand-sim.log`
   - `docs/evidence/adr0085-gb10-process-smoke/review-agent.json`
3. `jq empty docs/evidence/adr0085-gb10-process-smoke/review-agent.json`
4. Grep checks for `total_generated=12`, `Current accepted scope`, and
   `status=ok`
5. `git diff --check`
