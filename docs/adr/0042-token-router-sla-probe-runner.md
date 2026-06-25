# ADR 0042: token-router-sla probe runner

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P1 严选不做集市；P2 无数据不承诺；P3 先度量再承诺 SLA；P5 人审；P8 守住边界
- 关联 ADR：0038 SLA measurement automation；0040 SLA measurement evidence APIs；0041 SLA evidence dashboard

## 背景

ADR 0040 已经提供 SLA contract / probe plan / probe run 的证据 API，ADR 0041 已经让 operator 可以在 dashboard 手工导入、生成和记录证据。但当前证据仍主要靠人工写回；这不能证明“合同 -> 计划 -> 实测 -> artifact -> API 回写”的自动化链条。

ADR 0038 明确要求 `token-router-sla` runner 独立于 API handler，负责长时间或可重复测量。下一步应先实现一个可实测的最小 runner，用 `gb10-4t` mock 或 token-router `/v1/chat/completions` 端点采集 OpenAI-compatible chat 样本，再把摘要和 artifact hash 写回证据 API。

## 决策

新增 `cmd/token-router-sla`，第一版提供：

```text
token-router-sla probe run \
  --api http://127.0.0.1:19090 \
  --admin-token <root access token> \
  --plan-id <sla_probe_plan_id> \
  --endpoint http://127.0.0.1:19090/v1/chat/completions \
  --demand-token <token or sk-token> \
  --out output/sla/<run>.json \
  --record
```

runner 行为：

1. 从 `GET /api/sla_probe_plans/:id` 获取计划，从 `GET /api/sla_contracts/:id` 获取 hard gate。
2. 按 `sample_size * repeat_count` 执行样本；为空时至少执行 1 个样本。
3. 对 `cold_no_cache` 样本使用唯一 `X-Session-Id` / OpenAI `user`，避免 warm cache 混入 cold SLA 证明；`warm_same_session` 使用同一 session。
4. 默认用 streaming chat completion，发送 `stream_options.include_usage=true`，记录 first byte、first event、first token、total latency、usage 和失败分类。
5. 非 streaming 只记录 full response latency，并设置 `ttft_observed=false`；不能把 full latency 冒充 TTFT。
6. 写出 artifact JSON，计算 artifact SHA256。
7. `--record` 时调用 `POST /api/sla_probe_runs/record`，写回 status、summary、hard gate 结果、artifact URI/hash。

第一版 hard gate 只实现可明确解释的简单门槛：

- `{"ttft_ms":{"p90_lte":8000}}`
- `{"ttft_ms":{"p95_lte":8000}}`
- `{"ttft_ms":{"p99_lte":8000}}`
- `{"ttft_p90_ms":8000}` / `{"ttft_p95_ms":8000}` / `{"ttft_p99_ms":8000}`

若合同包含 TTFT gate 但本次没有 streaming TTFT 样本，runner 将把 run 判为 failed/invalid，而不是制造不存在的 TTFT 证据。

## 边界

1. 不在 API handler 内执行 probe。
2. 不自动创建 supplier / channel / capacity。
3. 不自动生成或审批 supplier evaluation。
4. 不修改 routing policy、supplier posture、billing 或 settlement。
5. 不声称覆盖 Kimi/GLM 全量官方验收；这是第一版可执行 smoke/admission runner。
6. 不替代 `token-router-sim` 的需求侧账务闭环验证；两者分别证明 SLA 证据链和业务流量台账链。

## 验收

1. `token-router-sla probe run` 能从 mock admin API 读取 plan / contract。
2. streaming mock endpoint 下能记录 TTFT、usage、sample sessions、artifact SHA256。
3. `cold_no_cache` 两个样本使用不同 session。
4. `--record` 能向 `/api/sla_probe_runs/record` 提交 passed run，summary 内含 TTFT 与 usage。
5. focused Go tests 与 `go test ./cmd/token-router-sla` 通过。
