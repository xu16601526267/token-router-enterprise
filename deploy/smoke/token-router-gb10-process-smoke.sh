#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUN_DIR=""
API_PORT=""
SUPPLY_PORT=""
MODEL="gpt-test"
SESSION_ID="session-process-e2e"
ADMIN_TOKEN="adminaccesstoken000000000001"
DEMAND_TOKEN="demandtoken"
NODE_NAME="$(hostname 2>/dev/null || echo local)"
GO_BIN="${GO:-go}"
PIDS=()

usage() {
  cat <<'USAGE'
usage: token-router-gb10-process-smoke.sh [options]

Options:
  --run-dir DIR       Evidence directory. Defaults to /tmp/token-router-gb10-process-smoke-<timestamp>-<pid>.
  --api-port PORT     API-only server port. Defaults to a free localhost port.
  --supply-port PORT  Mock supply port. Defaults to a free localhost port.
  --model NAME        Model name. Defaults to gpt-test.
  --session-id ID     Demand session id. Defaults to session-process-e2e.
  --node-name NAME    NODE_NAME for seed/API process evidence. Defaults to hostname.
  --go BIN            Go executable. Defaults to $GO or go.
  -h, --help          Show this help.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --run-dir)
      RUN_DIR="$2"
      shift 2
      ;;
    --api-port)
      API_PORT="$2"
      shift 2
      ;;
    --supply-port)
      SUPPLY_PORT="$2"
      shift 2
      ;;
    --model)
      MODEL="$2"
      shift 2
      ;;
    --session-id)
      SESSION_ID="$2"
      shift 2
      ;;
    --node-name)
      NODE_NAME="$2"
      shift 2
      ;;
    --go)
      GO_BIN="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$RUN_DIR" ]]; then
  RUN_DIR="/tmp/token-router-gb10-process-smoke-$(date +%Y%m%d%H%M%S)-$$"
fi

port_is_open() {
  local port="$1"
  (echo >/dev/tcp/127.0.0.1/"$port") >/dev/null 2>&1
}

pick_port() {
  local start="$1"
  local end=$((start + 999))
  local port
  for ((port = start; port <= end; port++)); do
    if ! port_is_open "$port"; then
      echo "$port"
      return 0
    fi
  done
  echo "no free localhost port found in ${start}-${end}" >&2
  return 1
}

wait_for_port() {
  local port="$1"
  local label="$2"
  local pid="${3:-}"
  local deadline=$((SECONDS + 30))
  while ((SECONDS < deadline)); do
    if port_is_open "$port"; then
      return 0
    fi
    if [[ -n "$pid" ]] && ! kill -0 "$pid" >/dev/null 2>&1; then
      echo "$label exited before opening port $port" >&2
      return 1
    fi
    sleep 0.2
  done
  echo "timeout waiting for $label on port $port" >&2
  return 1
}

cleanup() {
  local status=$?
  for pid in "${PIDS[@]}"; do
    if kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
  for pid in "${PIDS[@]}"; do
    wait "$pid" >/dev/null 2>&1 || true
  done
  if [[ "$status" -ne 0 ]]; then
    echo "smoke failed; evidence preserved at $RUN_DIR" >&2
    echo "logs: $RUN_DIR/logs" >&2
  fi
}
trap cleanup EXIT

if [[ -z "$API_PORT" ]]; then
  API_PORT="$(pick_port 43000)"
fi
if [[ -z "$SUPPLY_PORT" ]]; then
  SUPPLY_PORT="$(pick_port 44000)"
fi

mkdir -p "$RUN_DIR/bin" "$RUN_DIR/logs"
DB_PATH="$RUN_DIR/token-router.db"
API_BASE="http://127.0.0.1:${API_PORT}"
SUPPLY_BASE="http://127.0.0.1:${SUPPLY_PORT}"

echo "building binaries into $RUN_DIR/bin"
{
  cd "$ROOT_DIR"
  "$GO_BIN" build -o "$RUN_DIR/bin/token-router-api" ./cmd/token-router-api
  "$GO_BIN" build -o "$RUN_DIR/bin/token-router-sim" ./cmd/token-router-sim
  "$GO_BIN" build -o "$RUN_DIR/bin/token-router-supply" ./cmd/token-router-supply
} >"$RUN_DIR/logs/build.log" 2>&1

echo "starting strict gb10-4t mock supply on $SUPPLY_BASE"
"$RUN_DIR/bin/token-router-sim" mock-supply \
  --addr "127.0.0.1:${SUPPLY_PORT}" \
  --model "$MODEL" \
  --require-session "*" \
  >"$RUN_DIR/logs/mock-supply.log" 2>&1 &
mock_pid="$!"
PIDS+=("$mock_pid")
wait_for_port "$SUPPLY_PORT" "mock supply" "$mock_pid"

echo "seeding SQLite database at $DB_PATH"
(
  cd "$RUN_DIR"
  SQLITE_PATH="$DB_PATH" \
  MEMORY_CACHE_ENABLED=true \
  NODE_NAME="$NODE_NAME" \
  "$RUN_DIR/bin/token-router-sim" seed \
    --supply-url "$SUPPLY_BASE" \
    --model "$MODEL" \
    --admin-token "$ADMIN_TOKEN" \
    --demand-token "$DEMAND_TOKEN"
) >"$RUN_DIR/logs/seed.log" 2>&1

echo "starting API-only server on $API_BASE"
(
  cd "$RUN_DIR"
  PORT="$API_PORT" \
  SQLITE_PATH="$DB_PATH" \
  MEMORY_CACHE_ENABLED=true \
  NODE_NAME="$NODE_NAME" \
  GIN_MODE=release \
  "$RUN_DIR/bin/token-router-api"
) >"$RUN_DIR/logs/api.log" 2>&1 &
api_pid="$!"
PIDS+=("$api_pid")
wait_for_port "$API_PORT" "API-only server" "$api_pid"

echo "running demand simulator"
(
  cd "$RUN_DIR"
  SQLITE_PATH="$DB_PATH" \
  MEMORY_CACHE_ENABLED=true \
  NODE_NAME="$NODE_NAME" \
  "$RUN_DIR/bin/token-router-sim" run \
    --base-url "$API_BASE" \
    --model "$MODEL" \
    --session-id "$SESSION_ID" \
    --admin-token "$ADMIN_TOKEN" \
    --demand-token "$DEMAND_TOKEN"
) >"$RUN_DIR/logs/demand-sim.log" 2>&1

grep -q "process e2e ok:" "$RUN_DIR/logs/demand-sim.log"
grep -q "capacity_telemetry_sweep_verified=true" "$RUN_DIR/logs/demand-sim.log"
grep -q "supplier_posture_verified=true" "$RUN_DIR/logs/demand-sim.log"
grep -q "routing_sla_evidence_verified=true" "$RUN_DIR/logs/demand-sim.log"
grep -q "supply_routing_policy_canary_verified=true" "$RUN_DIR/logs/demand-sim.log"
grep -q "policy_miss_insight_verified=true" "$RUN_DIR/logs/demand-sim.log"
grep -q "assigned_session_verified=true" "$RUN_DIR/logs/demand-sim.log"

echo "running review agent once"
"$RUN_DIR/bin/token-router-supply" review agent \
  --once \
  --api "$API_BASE" \
  --admin-token "$ADMIN_TOKEN" \
  --model "$MODEL" \
  --user-id 2 \
  --min-generated 1 \
  --agent-key "${NODE_NAME}:gb10-process-smoke-review" \
  --hostname "$NODE_NAME" \
  --runtime-ref "process-smoke:${RUN_DIR}" \
  --version "dev" \
  >"$RUN_DIR/logs/review-agent.json" 2>&1

grep -q '"status": "ok"' "$RUN_DIR/logs/review-agent.json"
grep -q '"total_generated":' "$RUN_DIR/logs/review-agent.json"
grep -q '"operating_insights"' "$RUN_DIR/logs/review-agent.json"

cat >"$RUN_DIR/summary.txt" <<SUMMARY
status=ok
root_dir=$ROOT_DIR
run_dir=$RUN_DIR
db_path=$DB_PATH
api_base=$API_BASE
supply_base=$SUPPLY_BASE
model=$MODEL
session_id=$SESSION_ID
node_name=$NODE_NAME
demand_log=$RUN_DIR/logs/demand-sim.log
review_log=$RUN_DIR/logs/review-agent.json
SUMMARY

echo "smoke ok"
cat "$RUN_DIR/summary.txt"
