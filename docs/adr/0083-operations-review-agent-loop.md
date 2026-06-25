# ADR 0083: Operations review agent loop

## Status

Accepted

## Context

ADR 0080 added `token-router-supply review once`, which refreshes the agent-readable review set by calling the existing generate APIs in dependency order. That command is enough for cron or systemd timers, but long-running deployments already have a precedent in `token-router-supply telemetry agent`: operators can run a resident process, inspect JSON cycle output, and let the process repeat on an interval.

The product boundary is still P5/P7: agents produce current evidence and recommendations, while humans approve, apply, activate, disable, or acknowledge in the dashboard. A resident review loop must therefore be only a scheduler around the same read-model generation chain. It must not become an automatic optimizer.

## Decision

Add `token-router-supply review agent`.

The command will:

1. Reuse the same eight-step generation chain as `review once`.
2. Accept the same period and slice filters as `review once`.
3. Add deployable loop flags: `--agent-key`, `--hostname`, `--runtime-ref`, `--version`, `--interval`, and `--once`.
4. Print one JSON cycle summary per run, including agent identity, status, timestamps, the nested `review once` summary, and any error.
5. Recompute the default source period on every cycle when no explicit period is supplied, so a resident process reviews the latest rolling window instead of replaying the startup window forever.

This ADR does not add a review-agent table or heartbeat API. Review rows themselves remain the durable source of truth, and process managers can collect stdout/stderr logs. A dedicated heartbeat model can be added later only if operators need centralized liveness for review loops.

## Boundaries

`review agent` must not:

- approve, reject, acknowledge, dismiss, apply, activate, disable, complete, or cancel anything
- create action plans from unapproved decisions
- modify `Channel.weight`, `Ability.weight`, `SupplierRoutePreference`, `SupplyRoutingPolicy`, pricing tables, billing, settlement, wallet, payout, invoice, or funds state
- create suppliers, channels, capacity snapshots, cost profiles, prepaid lots, or executions
- treat a heartbeat or loop success as proof that any downstream recommendation was accepted by a human

It only schedules the same review read-model generation that `review once` already performs.

## Consequences

- Operators can choose either timer mode (`review once`) or resident mode (`review agent`) without writing shell loops.
- P5 becomes closer to an always-current evidence workflow while preserving the human dashboard approval boundary.
- The first implementation keeps state small: no new database table, no new admin endpoint, and no automatic remediation semantics.

## Verification

Completed on `aima2`:

1. Temporary module copy for focused CLI validation:

   ```bash
   ssh aima2 'rm -rf /tmp/token-router-adr0083 && mkdir -p /tmp/token-router-adr0083/cmd'
   rsync -az go.mod go.sum aima2:/tmp/token-router-adr0083/
   rsync -az cmd/token-router-supply aima2:/tmp/token-router-adr0083/cmd/
   ssh aima2 'cd /tmp/token-router-adr0083 && gofmt -w cmd/token-router-supply/main.go cmd/token-router-supply/main_test.go && go test ./cmd/token-router-supply -count=1 && go vet ./cmd/token-router-supply && go build -o bin/token-router-supply ./cmd/token-router-supply'
   ```

   Output:

   ```text
   ok   github.com/QuantumNous/new-api/cmd/token-router-supply 0.011s
   ```

2. Focused httptest coverage verifies:

   - `review agent` posts the same eight generate endpoints as `review once`, in dependency order.
   - when no explicit period is supplied, a cycle computes a current rolling one-hour source window instead of reusing a startup period.
   - `--min-generated` gate failures are reflected in the cycle JSON as `status=failed` with an error string.
   - `review agent --once` requires an admin token before running a cycle.

3. Real-process proof against strict gb10 mock supply + fresh SQLite + API-only server.

   Evidence directory: `/tmp/token-router-adr0083-agent-process-20260623135949-2137245`.

   The demand simulator first completed the existing process chain:

   ```text
   process e2e ok: ledgers=4 session=session-process-e2e assigned_session=trsess_202606230600057519350398268d9d6DzWdnOeD cached_tokens_verified=true margin_verified=true settlement_verified=true capacity_verified=true capacity_usage_refresh_verified=true capacity_telemetry_verified=true capacity_telemetry_collect_verified=true capacity_telemetry_sweep_verified=true capacity_telemetry_insight_verified=true supplier_scorecard_verified=true supplier_evaluation_verified=true supplier_posture_verified=true supplier_route_preference_verified=true sla_evidence_verified=false traffic_profile_verified=true traffic_forecast_verified=true traffic_forecast_seasonal_anomaly_verified=true pricing_recommendation_verified=true supply_decision_verified=true self_hosted_cost_profile_verified=true supply_prepaid_lot_verified=true supply_expansion_opportunity_verified=true operating_insight_verified=true supply_action_plan_verified=true supply_action_execution_verified=true supply_action_execution_drawdown_verified=true supply_routing_policy_verified=true supply_routing_policy_canary_verified=true routing_sla_evidence_verified=true policy_miss_insight_verified=true assigned_session_verified=true
   ```

   Then the resident-mode command executed one live API cycle without explicit period flags:

   ```bash
   token-router-supply review agent --once \
     --api http://127.0.0.1:<port> \
     --admin-token adminaccesstoken000000000001 \
     --model gpt-test \
     --user-id 2 \
     --min-generated 1 \
     --agent-key aima2:adr0083-review \
     --hostname aima2 \
     --runtime-ref process:/tmp/token-router-adr0083-agent-process-20260623135949-2137245 \
     --version adr0083-real-process
   ```

   Summary output:

   ```json
   {
     "agent_key": "aima2:adr0083-review",
     "hostname": "aima2",
     "runtime_ref": "process:/tmp/token-router-adr0083-agent-process-20260623135949-2137245",
     "version": "adr0083-real-process",
     "status": "ok",
     "started_at": 1782194405,
     "finished_at": 1782194406,
     "review": {
       "status": "ok",
       "period_start": 1782190805,
       "period_end": 1782194405,
       "model_name": "gpt-test",
       "user_id": 2,
       "total_generated": 12,
       "steps": [
         {"name": "supplier_scorecards", "path": "/api/supplier_scorecards/generate", "count": 3},
         {"name": "supplier_posture_recommendations", "path": "/api/supplier_posture_recommendations/generate", "count": 3},
         {"name": "traffic_profiles", "path": "/api/traffic_profiles/generate", "count": 1},
         {"name": "traffic_forecasts", "path": "/api/traffic_forecasts/generate", "count": 1},
         {"name": "pricing_recommendations", "path": "/api/pricing_recommendations/generate", "count": 1},
         {"name": "supply_decisions", "path": "/api/supply_decisions/generate", "count": 1},
         {"name": "supply_expansion_opportunities", "path": "/api/supply_expansion_opportunities/generate", "count": 1},
         {"name": "operating_insights", "path": "/api/operating_insights/generate", "count": 1}
       ]
     }
   }
   ```

4. Product, architecture, and traffic docs now distinguish timer mode (`review once`) from resident loop mode (`review agent`) while preserving the same non-execution boundary.
