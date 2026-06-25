# ADR 0046: supplier evaluation SLA evidence dashboard

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P1 严选不做集市；P2 无数据不承诺；P3 先度量再承诺 SLA；P4 供应商优胜劣汰；P5 人审；P8 守住边界
- 关联 ADR：0031 Supplier evaluation dashboard；0038 SLA measurement automation；0041 SLA evidence dashboard；0043 SLA evidence gated supplier evaluations；0045 Real-process SLA gated admission E2E

## 背景

ADR 0043 已要求 active SLA contract 存在时，`SupplierEvaluation.admit` 必须引用 passed admission `SlaProbeRun`。ADR 0045 又用真实进程链路证明 `token-router-sla` 生成的 passed run 会被 supplier evaluation 采纳。

但 operator 在 Evaluations tab 里仍只能看到 scorecard runtime evidence、recommendation、review/apply 状态和 reason。`sla_contract_id`、`sla_probe_run_id` 和 `sla_gate_summary_json` 已经通过 API 返回，却没有被展示出来，容易让“admit 是否真的有 SLA 证据”退回到后端日志或数据库查询。

## 决策

在现有 Supplier Evaluations 表格里增加只读 SLA evidence 展示：

1. 保留原 scorecard 指标，并将表头明确为 Runtime Evidence。
2. 新增 SLA Evidence 列：
   - 有 `sla_probe_run_id` 时显示 linked badge、SLA contract id、probe run id。
   - 从 `sla_gate_summary_json` 解析并展示 run key、hard gate pass/fail、artifact SHA256 短码和 runtime ref。
   - 没有 SLA evidence 时显示 no-evidence badge，并明确这是 scorecard-only 状态。
3. 该列只显示后端事实，不创建 contract、plan 或 probe run，也不自动 approve/apply supplier evaluation。

## 边界

1. 不在浏览器里执行 SLA probe 或 benchmark。
2. 不把 runtime scorecard 均分替代为 SLA gate proof。
3. 不改变 supplier evaluation 的 recommendation、review、apply 或 routing 行为。
4. 不新增 dashboard-level admission policy；admit 门禁仍由后端 `SupplierEvaluationService` 执行。

## 验收

1. Evaluations tab 可以区分 Runtime Evidence 和 SLA Evidence。
2. linked SLA evidence 至少显示 contract/run id，并在 summary JSON 可解析时显示 run key、hard gate、artifact/runtime 元信息。
3. 缺少 SLA evidence 的 evaluation 不被误呈现为已通过 SLA，只显示 scorecard-only。
4. frontend i18n sync 和 typecheck 通过。
