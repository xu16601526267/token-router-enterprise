#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${TOKEN_ROUTER_BASE_URL:-}"
API_KEY="${TOKEN_ROUTER_API_KEY:-}"
MODEL="${TOKEN_ROUTER_MODEL:-gpt-4o-mini}"
CONCURRENCY="${TOKEN_ROUTER_CONCURRENCY:-6}"
MAX_TOKENS="${TOKEN_ROUTER_MAX_TOKENS:-128}"
LONG_MAX_TOKENS="${TOKEN_ROUTER_LONG_MAX_TOKENS:-512}"
TIMEOUT_SECONDS="${TOKEN_ROUTER_TIMEOUT_SECONDS:-120}"
SETTLE_SECONDS="${TOKEN_ROUTER_SETTLE_SECONDS:-3}"
STRICT_LEDGER="${TOKEN_ROUTER_STRICT_LEDGER:-0}"
RUN_DIR="${TOKEN_ROUTER_RUN_DIR:-}"

usage() {
  cat <<'USAGE'
usage: enterprise-downstream-customer-smoke.sh [options]

Downstream customer smoke test against a deployed Token Router service.
It only needs an API key, so it matches B/C customers or another relay station.

Environment variables:
  TOKEN_ROUTER_BASE_URL        Required. Example: https://approaching.aimaserver.com
  TOKEN_ROUTER_API_KEY         Required. Downstream sk-* key.
  TOKEN_ROUTER_MODEL           Model to call. Default: gpt-4o-mini
  TOKEN_ROUTER_CONCURRENCY     Parallel request count. Default: 6
  TOKEN_ROUTER_MAX_TOKENS      Normal request max_tokens. Default: 128
  TOKEN_ROUTER_LONG_MAX_TOKENS Long request max_tokens. Default: 512
  TOKEN_ROUTER_STRICT_LEDGER   Set 1 to fail when usage/log evidence does not increase.
  TOKEN_ROUTER_RUN_DIR         Evidence directory. Default: /tmp/token-router-downstream-smoke-*

Options override the same values:
  --base-url URL
  --api-key KEY
  --model NAME
  --concurrency N
  --max-tokens N
  --long-max-tokens N
  --timeout-seconds N
  --settle-seconds N
  --strict-ledger
  --run-dir DIR
  -h, --help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base-url)
      BASE_URL="$2"
      shift 2
      ;;
    --api-key)
      API_KEY="$2"
      shift 2
      ;;
    --model)
      MODEL="$2"
      shift 2
      ;;
    --concurrency)
      CONCURRENCY="$2"
      shift 2
      ;;
    --max-tokens)
      MAX_TOKENS="$2"
      shift 2
      ;;
    --long-max-tokens)
      LONG_MAX_TOKENS="$2"
      shift 2
      ;;
    --timeout-seconds)
      TIMEOUT_SECONDS="$2"
      shift 2
      ;;
    --settle-seconds)
      SETTLE_SECONDS="$2"
      shift 2
      ;;
    --strict-ledger)
      STRICT_LEDGER="1"
      shift
      ;;
    --run-dir)
      RUN_DIR="$2"
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

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 2
  fi
}

require_cmd curl
require_cmd python3
require_cmd date

if [[ -z "$BASE_URL" || -z "$API_KEY" ]]; then
  usage >&2
  exit 2
fi

BASE_URL="${BASE_URL%/}"

if [[ -z "$RUN_DIR" ]]; then
  RUN_DIR="/tmp/token-router-downstream-smoke-$(date +%Y%m%d%H%M%S)-$$"
fi

mkdir -p "$RUN_DIR/payloads" "$RUN_DIR/responses" "$RUN_DIR/meta"

mask_key() {
  python3 - "$1" <<'PY'
import sys

key = sys.argv[1]
if len(key) <= 10:
    print("***")
else:
    print(f"{key[:5]}...{key[-4:]}")
PY
}

MASKED_KEY="$(mask_key "$API_KEY")"

make_chat_payload() {
  local file="$1"
  local prompt="$2"
  local max_tokens="$3"
  python3 - "$MODEL" "$prompt" "$max_tokens" >"$file" <<'PY'
import json
import sys

model, prompt, max_tokens = sys.argv[1], sys.argv[2], int(sys.argv[3])
payload = {
    "model": model,
    "messages": [
        {"role": "system", "content": "You are a concise API smoke-test assistant."},
        {"role": "user", "content": prompt},
    ],
    "temperature": 0,
    "max_tokens": max_tokens,
}
print(json.dumps(payload, ensure_ascii=False))
PY
}

make_invalid_payload() {
  local file="$1"
  python3 - >"$file" <<'PY'
import json
import time

payload = {
    "model": f"token-router-smoke-invalid-model-{int(time.time())}",
    "messages": [{"role": "user", "content": "This request should be rejected."}],
    "max_tokens": 16,
}
print(json.dumps(payload, ensure_ascii=False))
PY
}

run_request() {
  local name="$1"
  local method="$2"
  local path="$3"
  local payload_file="${4:-}"
  local body="$RUN_DIR/responses/${name}.json"
  local meta="$RUN_DIR/meta/${name}.meta"
  local err="$RUN_DIR/meta/${name}.stderr"
  local args=(
    --silent
    --show-error
    --location
    --max-time "$TIMEOUT_SECONDS"
    --output "$body"
    --write-out "%{http_code} %{time_total}"
    --request "$method"
    --header "Authorization: Bearer ${API_KEY}"
    --header "Content-Type: application/json"
  )

  if [[ -n "$payload_file" ]]; then
    args+=(--data-binary "@${payload_file}")
  fi

  local result
  if result="$(curl "${args[@]}" "${BASE_URL}${path}" 2>"$err")"; then
    echo "$result" >"$meta"
  else
    local rc=$?
    echo "000 0" >"$meta"
    echo "curl_exit=${rc}" >>"$err"
  fi
}

echo "evidence: $RUN_DIR"
echo "base_url: $BASE_URL"
echo "api_key: $MASKED_KEY"
echo "model: $MODEL"
echo "concurrency: $CONCURRENCY"

make_chat_payload "$RUN_DIR/payloads/chat-single.json" "Return the exact text: token-router smoke ok" "$MAX_TOKENS"
make_chat_payload "$RUN_DIR/payloads/chat-long.json" "Write 8 short numbered lines about API relay billing verification." "$LONG_MAX_TOKENS"
make_invalid_payload "$RUN_DIR/payloads/chat-invalid-model.json"

run_request "subscription_before" "GET" "/v1/dashboard/billing/subscription"
run_request "usage_before" "GET" "/v1/dashboard/billing/usage"
run_request "chat_single" "POST" "/v1/chat/completions" "$RUN_DIR/payloads/chat-single.json"
run_request "chat_long" "POST" "/v1/chat/completions" "$RUN_DIR/payloads/chat-long.json"

for ((i = 1; i <= CONCURRENCY; i++)); do
  payload="$RUN_DIR/payloads/chat-concurrent-${i}.json"
  make_chat_payload "$payload" "Concurrent smoke request ${i}. Return only: ok-${i}" "$MAX_TOKENS"
  run_request "chat_concurrent_${i}" "POST" "/v1/chat/completions" "$payload" &
done
wait

run_request "chat_invalid_model" "POST" "/v1/chat/completions" "$RUN_DIR/payloads/chat-invalid-model.json"

if [[ "$SETTLE_SECONDS" != "0" ]]; then
  sleep "$SETTLE_SECONDS"
fi

run_request "usage_after" "GET" "/v1/dashboard/billing/usage"
run_request "subscription_after" "GET" "/v1/dashboard/billing/subscription"
run_request "token_logs" "GET" "/api/log/token"

summary_status=0
python3 - "$RUN_DIR" "$BASE_URL" "$MASKED_KEY" "$MODEL" "$CONCURRENCY" "$STRICT_LEDGER" >"$RUN_DIR/summary.md" <<'PY' || summary_status=$?
import json
import sys
from pathlib import Path

run_dir = Path(sys.argv[1])
base_url = sys.argv[2]
masked_key = sys.argv[3]
model = sys.argv[4]
concurrency = int(sys.argv[5])
strict_ledger = sys.argv[6] == "1"

failures: list[str] = []
warnings: list[str] = []


def meta(name: str) -> tuple[int, float]:
    path = run_dir / "meta" / f"{name}.meta"
    if not path.exists():
        return 0, 0.0
    parts = path.read_text(encoding="utf-8", errors="replace").strip().split()
    try:
        status = int(parts[0])
    except Exception:
        status = 0
    try:
        elapsed = float(parts[1])
    except Exception:
        elapsed = 0.0
    return status, elapsed


def body(name: str):
    path = run_dir / "responses" / f"{name}.json"
    if not path.exists():
        return None
    text = path.read_text(encoding="utf-8", errors="replace").strip()
    if not text:
        return None
    try:
        return json.loads(text)
    except Exception:
        return {"_raw": text[:500]}


def has_error(payload) -> bool:
    if not isinstance(payload, dict):
        return False
    if payload.get("error"):
        return True
    return payload.get("success") is False


def expect_ok(name: str, label: str):
    status, _ = meta(name)
    payload = body(name)
    if status < 200 or status >= 300:
        failures.append(f"{label} HTTP 状态异常：{status}")
        return payload
    if has_error(payload):
        failures.append(f"{label} 返回业务错误：{payload.get('error') or payload.get('message')}")
    return payload


def expect_chat(name: str, label: str):
    payload = expect_ok(name, label)
    if not isinstance(payload, dict):
        failures.append(f"{label} 响应不是 JSON 对象")
        return
    choices = payload.get("choices")
    if not isinstance(choices, list) or not choices:
        failures.append(f"{label} 缺少 OpenAI 兼容 choices")
    if "usage" not in payload:
        warnings.append(f"{label} 缺少 usage 字段，无法直接核对 token 消耗")


subscription_before = expect_ok("subscription_before", "前置余额查询")
usage_before = expect_ok("usage_before", "前置用量查询")
expect_chat("chat_single", "单次补全")
expect_chat("chat_long", "长输出补全")

for index in range(1, concurrency + 1):
    expect_chat(f"chat_concurrent_{index}", f"并发补全 #{index}")

invalid_status, _ = meta("chat_invalid_model")
invalid_body = body("chat_invalid_model")
if 200 <= invalid_status < 300 and not has_error(invalid_body):
    failures.append("错误模型请求被成功放行，模型权限/路由约束需要复查")

usage_after = expect_ok("usage_after", "后置用量查询")
subscription_after = expect_ok("subscription_after", "后置余额查询")
token_logs = expect_ok("token_logs", "API Key 日志查询")

before_usage = None
after_usage = None
if isinstance(usage_before, dict):
    before_usage = usage_before.get("total_usage")
if isinstance(usage_after, dict):
    after_usage = usage_after.get("total_usage")
if isinstance(before_usage, (int, float)) and isinstance(after_usage, (int, float)):
    if after_usage < before_usage:
        failures.append(f"后置用量小于前置用量：before={before_usage}, after={after_usage}")
    elif after_usage == before_usage:
        message = f"用量没有增长：before={before_usage}, after={after_usage}"
        (failures if strict_ledger else warnings).append(message)

before_limit = None
after_limit = None
if isinstance(subscription_before, dict):
    before_limit = subscription_before.get("hard_limit_usd")
if isinstance(subscription_after, dict):
    after_limit = subscription_after.get("hard_limit_usd")
if isinstance(before_limit, (int, float)) and isinstance(after_limit, (int, float)):
    if after_limit > before_limit:
        warnings.append(f"后置余额高于前置余额：before={before_limit}, after={after_limit}")

log_count = None
if isinstance(token_logs, dict) and isinstance(token_logs.get("data"), list):
    log_count = len(token_logs["data"])
    expected_min = 2 + concurrency
    if log_count < expected_min:
        message = f"API Key 日志数量少于本轮请求数：logs={log_count}, expected_min={expected_min}"
        (failures if strict_ledger else warnings).append(message)

print("# 下游客户真实环境 Smoke 结果")
print()
print(f"- 服务地址：`{base_url}`")
print(f"- API Key：`{masked_key}`")
print(f"- 模型：`{model}`")
print(f"- 并发数：`{concurrency}`")
print(f"- 严格账本模式：`{strict_ledger}`")
print()
print("## 请求状态")
print()
names = [
    ("subscription_before", "前置余额"),
    ("usage_before", "前置用量"),
    ("chat_single", "单次补全"),
    ("chat_long", "长输出补全"),
]
names.extend((f"chat_concurrent_{i}", f"并发补全 #{i}") for i in range(1, concurrency + 1))
names.extend([
    ("chat_invalid_model", "错误模型"),
    ("usage_after", "后置用量"),
    ("subscription_after", "后置余额"),
    ("token_logs", "Key 日志"),
])
print("| 项目 | HTTP | 耗时秒 |")
print("| --- | ---: | ---: |")
for name, label in names:
    status, elapsed = meta(name)
    print(f"| {label} | {status} | {elapsed:.3f} |")

print()
print("## 对账观察")
print()
print(f"- 用量：before=`{before_usage}` after=`{after_usage}`")
print(f"- 余额：before=`{before_limit}` after=`{after_limit}`")
print(f"- Key 日志条数：`{log_count}`")

if warnings:
    print()
    print("## 警告")
    print()
    for item in warnings:
        print(f"- {item}")

if failures:
    print()
    print("## 失败项")
    print()
    for item in failures:
        print(f"- {item}")
    sys.exit(1)

print()
print("## 结论")
print()
print("下游 API Key 的余额查询、用量查询、单次请求、长输出、并发请求、错误模型拦截和 Key 日志查询均通过。")
PY

cat "$RUN_DIR/summary.md"
exit "$summary_status"
