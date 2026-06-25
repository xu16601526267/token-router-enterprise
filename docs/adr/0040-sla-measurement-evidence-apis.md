# ADR 0040: SLA measurement evidence APIs

- 状态：Accepted
- 日期：2026-06-23
- 关联原则：P1 严选不做集市；P2 无数据不承诺；P3 先度量再承诺 SLA；P4 供应商优胜劣汰；P5 人审；P8 守住边界
- 关联 ADR：0038 SLA measurement automation for supplier admission

## 背景

ADR 0038 已经规划 `SlaContract` / `SlaProbePlan` / `SlaProbeRun` / `token-router-sla` runner。当前系统已有 supplier scorecard / evaluation，但它们来自运行期业务流量聚合，不能替代官方 SLA 准入测量。

下一步需要先把 SLA 合同、探针计划和外部 runner 写回的证据落到平台数据层。这样可以先形成“合同 -> 计划 -> 运行证据 -> 查询回读”的闭环，再实现长时间 benchmark runner。

## 决策

1. 新增 `SlaContract`，保存模型 SLA 合同的版本化 JSON profile：
   - `contract_key`
   - `model_name`
   - `model_aliases`
   - `provider_family`
   - `source_name` / `source_ref` / `source_sha256`
   - `version`
   - `status`
   - `effective_from` / `effective_to`
   - `measurement_profile_json`
   - `hard_gate_json`
   - `soft_gate_json`
2. 新增 `SlaProbePlan`，从 active/draft contract 为 supplier/channel 生成 admission/runtime 探针计划。
3. 新增 `SlaProbeRun`，允许外部 runner 写回一次执行的 summary、hard gate 结果、artifact URI/hash 与失败原因。
4. 新增 admin API：
   - `POST /api/sla_contracts/import`
   - `GET /api/sla_contracts`
   - `GET /api/sla_contracts/:id`
   - `POST /api/sla_probe_plans/generate`
   - `GET /api/sla_probe_plans`
   - `GET /api/sla_probe_plans/:id`
   - `POST /api/sla_probe_runs/record`
   - `GET /api/sla_probe_runs`
   - `GET /api/sla_probe_runs/:id`
5. `SlaProbePlan` 复制 contract 的测量 profile，并把常用 profile 字段展开到 plan，方便 dashboard/runner 不必每次重新解析合同。
6. 该 slice 只做 API server 的数据与证据入口，不在 API handler 中执行长 benchmark。

## 不做什么

1. 不实现 `token-router-sla probe run` runner。
2. 不把 SLA probe run 自动接入 `SupplierEvaluation` 的 admit/reject 判定。
3. 不自动创建 supplier/channel/capacity。
4. 不自动调权、禁用 supplier/channel 或修改 routing policy。
5. 不触碰支付、打款、发票、结算状态。
6. 不承诺 Kimi/GLM SLA 是否通过；这里只记录可复核证据。

## 验收

1. 可以导入 contract 并回读 JSON profile。
2. 可以基于 contract + supplier + channel 生成 admission plan。
3. plan 中能回读 input/output/cache/stream/error profile 维度。
4. 可以 record probe run，并回读 status、summary、hard gate、artifact hash。
5. Go focused tests 与相关 backend/e2e tests 通过。
