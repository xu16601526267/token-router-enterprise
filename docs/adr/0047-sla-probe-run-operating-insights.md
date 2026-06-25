# ADR 0047: SLA probe run operating insights

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P3 先度量再承诺 SLA；P5 人审；P7 理念是假设；P8 守住边界
- 关联 ADR：0034 Operating insights；0038 SLA measurement automation；0040 SLA measurement evidence APIs；0042 token-router-sla probe runner；0045 Real-process SLA gated admission E2E；0046 Supplier evaluation SLA evidence dashboard

## 背景

`SlaProbeRun` 已经能记录 runner 产生的 SLA 证据，`SupplierEvaluation` 也已在 active contract 下把 passed admission run 作为 `admit` 的硬前置。前端也能在 Evaluations 表里核对 linked run。

但运行期 SLA probe 的失败、invalid、cancelled 仍只停留在 SLA Evidence tab。ADR 0034 的 `OperatingInsight` 只由 `TrafficProfile` 合成供给/定价洞察，没有引用 `SlaProbeRun`。这会让 P3 的“先度量再承诺 SLA”缺少经营复盘入口：runner 证明某个供应商/模型/SLA tier 不稳，operator 仍需要跳到另一个 tab 才能发现质量风险。

## 决策

扩展 `OperatingInsight`：

1. 增加只读证据字段：
   - `sla_contract_id`
   - `sla_probe_run_id`
   - `sla_probe_run_key`
   - `sla_probe_status`
   - `sla_hard_gate_passed`
   - `sla_failure_reasons`
   - `sla_artifact_sha256`
   - `sla_runtime_ref`
2. `GenerateOperatingInsights` 继续生成原有 profile-level insight，同时扫描同一周期内 status 为 `failed` / `invalid` / `cancelled` 或 hard gate 未通过的 `SlaProbeRun`，生成 run-level quality insight。
3. run-level insight 使用独立 key：`operating:sla_probe_run:<run_key>`，避免覆盖 profile-level insight。
4. run-level insight 的默认分类为 `quality_watch`：
   - `failed` / `invalid` / hard gate failed -> `severity=action`
   - `cancelled` -> `severity=watch`
5. 重复 generate 刷新 run 事实字段、summary 和 recommended action，但保留已 acknowledge / dismiss 的人审状态。
6. Operating Insights dashboard 增加 SLA evidence 展示列，只显示 evidence，不执行 probe，不改变 supplier、routing、pricing 或 settlement。

## 边界

1. 不把 failed runtime probe 自动转换为 supplier 禁用、routing policy disable 或 pricing raise。
2. 不在 API handler 或浏览器里执行 benchmark；probe execution 仍由 `token-router-sla` 或外部 runner 完成。
3. 不把缺少 profile 的 run 丢弃；SLA runner evidence 本身就是可复盘事实。
4. 不改变 supplier admission 的硬门禁；准入仍由 `SupplierEvaluation` 使用 passed admission run。

## 验收

1. `OperatingInsight` 可持久化并查询 SLA probe run evidence 字段。
2. `GenerateOperatingInsights` 在没有 `TrafficProfile` 但存在 failed/invalid/cancelled SLA run 时仍生成 quality insight。
3. 重复 generate 不覆盖已 review 状态。
4. `/token-router` Operating Insights tab 能展示 linked SLA run 证据。
5. focused Go tests、frontend i18n/typecheck、以及 route-mocked browser smoke 通过。
