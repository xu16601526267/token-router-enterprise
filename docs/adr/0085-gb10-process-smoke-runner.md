# ADR 0085: gb10 process smoke runner

## Status

Accepted

## Context

The project has repeatedly proven the gb10-4t supply + demand path with real
processes on `aima2`: build binaries, start the mock supply, seed SQLite, start
the API-only server, run demand through HTTP, and then run the operations review
agent once.

Those proofs are valuable, but the orchestration still lives mostly in per-run
shell history and temp directories. That makes later dedicated-server migration
and regression checks too dependent on remembering the exact sequence, ports,
environment variables, and log locations.

In particular, a reproducible proof must preserve the strict parts that caught
past mistakes:

- the mock supply must reject calls without an upstream session id
- seed and API must use the same `SQLITE_PATH`
- the API must start with memory channel cache enabled after seed data exists
- the demand simulator must verify ledger, cache, margin, capacity telemetry,
  posture, self-hosted canary, runtime SLA routing evidence, policy miss, and
  router-assigned-session markers
- the review agent must run against the live API with `--once` and a
  `--min-generated` gate

## Decision

Add `deploy/smoke/token-router-gb10-process-smoke.sh`.

The script will:

1. build `token-router-api`, `token-router-sim`, and `token-router-supply` into
   a timestamped run directory
2. choose free localhost ports unless explicit ports are supplied
3. start `token-router-sim mock-supply --require-session '*'`
4. seed a fresh SQLite database with the mock supply URL
5. start the API-only server with `SQLITE_PATH`, `MEMORY_CACHE_ENABLED=true`,
   `NODE_NAME`, and `GIN_MODE=release`
6. run `token-router-sim run` against the live API
7. run `token-router-supply review agent --once --min-generated 1`
8. grep for route-advancing proof markers and leave logs, binaries, SQLite, and
   summary in the evidence directory

Add `deploy/smoke/README.md` as the operator runbook.

## Boundaries

The smoke runner must not:

- install or enable systemd units
- mutate any non-temporary database
- require a real admin token
- approve, reject, apply, activate, disable, complete, acknowledge, or dismiss
  any operator workflow outside the simulator's own closed proof path
- create production suppliers, channels, routing policies, prices, bills,
  settlements, wallet records, payouts, invoices, or funds state
- claim production readiness beyond a local real-process smoke pass

This is a reproducibility artifact for the existing proof chain, not a new
scheduler and not an automatic operations agent.

## Consequences

- The current gb10 process proof is one command from a source checkout with Go.
- The evidence directory gives operators a concrete artifact to attach to route
  logs or release notes.
- Dedicated-server migration can first prove the same process chain before
  installing the systemd units from ADR 0084.

## Verification

Completed on `aima2` from `/tmp/token-router-adr0085-smoke-src`:

```bash
bash -n deploy/smoke/token-router-gb10-process-smoke.sh
deploy/smoke/token-router-gb10-process-smoke.sh --node-name aima2
```

The script exited `0` and preserved evidence at:

```text
/tmp/token-router-gb10-process-smoke-20260623142209-2472155
```

Artifacts:

- `bin/token-router-api` 65M
- `bin/token-router-sim` 40M
- `bin/token-router-supply` 7.9M
- `token-router.db` 2.2M
- `logs/demand-sim.log`
- `logs/review-agent.json`
- `summary.txt`

Demand proof:

```text
process e2e ok: ledgers=4 session=session-process-e2e ... capacity_telemetry_sweep_verified=true ... supplier_posture_verified=true ... supply_routing_policy_canary_verified=true routing_sla_evidence_verified=true policy_miss_insight_verified=true assigned_session_verified=true
```

Review agent proof:

```text
agent_key=aima2:gb10-process-smoke-review
status=ok
period_start=1782192157
period_end=1782195757
total_generated=12
supplier_scorecards=3
supplier_posture_recommendations=3
traffic_profiles=1
traffic_forecasts=1
pricing_recommendations=1
supply_decisions=1
supply_expansion_opportunities=1
operating_insights=1
```
