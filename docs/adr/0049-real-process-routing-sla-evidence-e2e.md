# ADR 0049: real-process routing SLA evidence E2E

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P3 先度量再承诺 SLA；P5 人审；P8 守住边界；P9 流量即情报
- 关联 ADR：0040 SLA measurement evidence APIs；0045 Real-process SLA gated admission E2E；0048 SLA evidence gated self-hosted routing policy

## 背景

ADR 0048 已要求 self-hosted `SupplyRoutingPolicy` 激活必须引用同 supplier、channel、model、SLA tier 的 passed runtime `SlaProbeRun`。HTTP e2e 已覆盖 contract -> runtime plan -> passed run -> policy activation -> demand route -> ledger，但真实进程 `token-router-sim run` 仍是在 recorded self-hosted execution 后直接 activate policy。

这会让真实进程 supply + demand simulator 不能证明 ADR 0048 的生产式链路：缺少 runtime evidence 时应该先拒绝，记录 passed runtime run 后才允许 policy 接管后续流量。

## 决策

扩展 `cmd/token-router-sim run` 的 supply decision 阶段：

1. 对 recorded self-hosted execution 先调用 `/api/supply_routing_policies/activate` 并断言失败消息包含 `passed runtime SLA probe run is required`。
2. 通过现有 SLA evidence API 在真实 API 进程中写入一条 runtime evidence：
   - `/api/sla_contracts/import` 导入 active self-hosted routing contract。
   - `/api/sla_probe_plans/generate` 生成 `probe_type=runtime_light`、`route_mode=direct_upstream` 的 plan。
   - `/api/sla_probe_runs/record` 记录 `status=passed`、`hard_gate_passed=true` 的 run。
3. 再次 activate routing policy，并断言 policy 持久化 `sla_contract_id`、`sla_probe_run_id`、`sla_probe_run_key`、`sla_artifact_sha256`、`sla_runtime_ref`。
4. simulator 输出增加 `routing_sla_evidence_verified=true`，表示真实进程链路已经证明 routing policy 使用的是 passed runtime run，而不是 admission run 或 scorecard。

## 边界

1. simulator 只用 API 写入确定性 probe record；不在自身内部执行 benchmark，也不替代 `token-router-sla` 的真实 probe runner。
2. 该 runtime contract 在 supply decision 阶段才导入，避免影响本轮 supplier evaluation 的 admission gate 判定。
3. 不自动 disable 已有 policy；runtime evidence 只作为 activation 前置证据。
4. 不改变 pricing、settlement、billing、supplier posture 或 channel status。

## 验收

1. `token-router-sim run` 在 self-hosted policy activation 前先验证缺 runtime evidence 会被拒绝。
2. simulator 通过 API 写入 active contract、runtime_light direct_upstream plan、passed run 后，policy activate 成功并持久化 evidence 字段。
3. 后续 demand request 命中 self-hosted supplier/channel，usage ledger 仍记录供应侧路径与 cache-aware 成本。
4. focused Go validation 与 aima2 真实进程验证通过，README / architecture 记录实测证据。
