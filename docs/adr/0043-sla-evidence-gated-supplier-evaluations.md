# ADR 0043: SLA evidence gated supplier evaluations

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P1 严选不做集市；P2 无数据不承诺；P3 先度量再承诺 SLA；P4 供应商优胜劣汰；P5 人审；P8 守住边界
- 关联 ADR：0030 Supplier admission evaluations；0038 SLA measurement automation；0040 SLA measurement evidence APIs；0042 token-router-sla probe runner

## 背景

`SupplierEvaluation` 目前只基于 `SupplierScorecard` 生成。scorecard 来自运行期业务流量，能说明一段时间内的成功率、延迟、cache、毛利和供给余量，但它不是官方 SLA 准入测量。

ADR 0038 已明确 supplier admission 不能只看运营均分：没有 active `SlaContract` 时只能作为普通观察；有 active contract 但没有 passed admission `SlaProbeRun` 时不能生成 `admit`。ADR 0040/0041/0042 已经补齐合同、计划、证据 API、dashboard 和第一版 runner，现在应把这条证据链接入 supplier evaluation。

## 决策

扩展 `SupplierEvaluation`：

1. 新增 `sla_contract_id`：本次 evaluation 采用的 active SLA contract。
2. 新增 `sla_probe_run_id`：本次 evaluation 采用的 latest passed admission probe run。
3. 新增 `sla_gate_summary_json`：快照记录 run status、model、SLA tier、route mode、runner、artifact hash、summary 片段等，避免后续 run 被更新后 evaluation 失去当时证据。

生成规则：

1. `SupplierScorecard` 仍决定基础 recommendation：
   - score >= 85 -> `admit`
   - 70 <= score < 85 -> `observe`
   - score < 70 -> `reject`
2. 若系统内没有 active `SlaContract`，保持旧行为：scorecard 可生成 `admit`，但 reason 仍只代表运营观察。
3. 若存在 active `SlaContract`：
   - 为该 supplier 查找 latest passed `admission` `SlaProbeRun`，且 run 关联 active contract、`hard_gate_passed=true`。
   - 找到则复制 `sla_contract_id`、`sla_probe_run_id`、`sla_gate_summary_json`。
   - 找不到且基础 recommendation 为 `admit` 时，将 recommendation 降为 `observe`，reason 明确说明缺少 passed SLA admission evidence。
4. `approve/reject/apply` 语义不变：仍由 admin 人审，不自动创建 supplier/channel/capacity，不自动调权，不触碰资金动作。

当前 `SupplierScorecard` 是 supplier-period 粒度，不含 model/SLA tier。第一版按 supplier 查找最新 passed admission run；后续若 scorecard 升级到 model/SLA 粒度，再把匹配条件收紧到 model/SLA tier。

## 边界

1. 不把 probe run 自动审批为 supplier admission。
2. 不从 evaluation 自动运行 probe。
3. 不自动创建或修改 supplier/channel/capacity/routing policy。
4. 不把历史 scorecard 数据重写成 SLA 证明。
5. 不把 active contract 缺失当作已经通过 SLA；只保持当前运营观察逻辑。

## 验收

1. 无 active contract 时，现有 scorecard-only evaluation 逻辑保持兼容。
2. 有 active contract 但无 passed admission run 时，高分 scorecard 只能生成 `observe`，且 reason 说明缺少 SLA evidence。
3. 有 active contract 和 passed admission run 时，高分 scorecard 可生成 `admit`，并记录 `sla_contract_id`、`sla_probe_run_id` 和 gate summary。
4. HTTP e2e 证明 SLA contract -> plan -> run -> scorecard -> supplier evaluation 链路中，evaluation 引用了 probe run。
5. `aima2` focused Go tests 通过，README 记录证据。
