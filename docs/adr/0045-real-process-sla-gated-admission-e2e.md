# ADR 0045: real-process SLA gated admission E2E

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P1 严选不做集市；P2 无数据不承诺；P3 先度量再承诺 SLA；P4 供应商优胜劣汰；P5 人审；P8 守住边界
- 关联 ADR：0030 Supplier admission evaluations；0038 SLA measurement automation；0042 token-router-sla probe runner；0043 SLA evidence gated supplier evaluations；0044 token-router-sla workflow CLI

## 背景

`token-router-sim run` 已经能在真实进程环境里验证 `gb10-4t` mock supply、需求请求、usage ledger、scorecard、supplier evaluation、pricing、decision、action、routing policy 等链路。

ADR 0043 又要求 active SLA contract 存在时，`SupplierEvaluation.admit` 必须引用 passed admission `SlaProbeRun`。ADR 0044 已让 operator 可以用 `token-router-sla contract import -> plan generate -> probe run --record` 编排证据链。但目前真实进程 simulator 仍默认不要求 SLA evidence，因此不能证明“CLI 生成的 passed run 被 supplier evaluation 采纳”。

## 决策

扩展 `cmd/token-router-sim run`：

```text
token-router-sim run --expect-sla-evidence --expected-sla-run-key process-sla-admission
```

行为：

1. 默认行为不变：没有 active SLA contract 的普通 business process E2E 仍可继续跑。
2. 当 `--expect-sla-evidence` 打开时，supplier evaluation 必须：
   - `recommendation=admit`
   - `sla_contract_id > 0`
   - `sla_probe_run_id > 0`
   - `sla_gate_summary_json` 非空
   - gate summary 内含指定 `--expected-sla-run-key`
   - reason 内含 SLA admission evidence。
3. `gb10-4t` mock supply 在收到 `stream=true` 的 OpenAI-compatible chat request 时返回 SSE chunk、usage 和 `[DONE]`，让 `token-router-sla probe run` 可以采集真实 TTFT，而不是用 non-streaming full latency 冒充。
4. 真实进程验证顺序由 operator/script 执行：
   - 启动 `gb10-4t` mock supply 和 API-only server。
   - `token-router-sim seed` 写入 supplier/channel/capacity/token。
   - `token-router-sla contract import` 导入 active contract。
   - `token-router-sla plan generate` 生成 admission plan。
   - `token-router-sla probe run --record` 执行并回写 passed run。
   - `token-router-sim run --expect-sla-evidence` 发送业务流量并验证 evaluation 引用该 run。

## 边界

1. simulator 只验证证据是否被采纳，不在自身内部导入 contract 或执行 probe。
2. 不自动 approve/apply supplier evaluation 之外的新 runtime 策略。
3. 不把 scorecard 运营均分当成 SLA 证明。
4. 不要求所有 process E2E 都先导入 SLA；该门禁只在显式 flag 下启用。

## 验收

1. `token-router-sim run --expect-sla-evidence` 在缺少 SLA evidence 时会失败。
2. mock `gb10-4t` streaming response 可让 `token-router-sla probe run` 采集 TTFT 并通过 simple hard gate。
3. 完整真实进程链路通过：CLI contract import -> CLI plan generate -> CLI probe run --record -> simulator demand run -> supplier evaluation 引用 `sla_probe_run_id`。
4. focused Go build/test 通过，aima2 真实进程验证输出 `sla_evidence_verified=true`。
