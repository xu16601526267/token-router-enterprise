# ADR 0048: SLA evidence gated self-hosted routing policy

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P3 先度量再承诺 SLA；P5 人审；P8 守住边界；P9 流量即情报
- 关联 ADR：0026 Self-hosted routing policies；0038 SLA measurement automation；0040 SLA measurement evidence APIs；0042 token-router-sla probe runner；0047 SLA probe run operating insights

## 背景

ADR 0026 已经允许 operator 把 recorded self-hosted `SupplyActionExecution` 显式激活成 `SupplyRoutingPolicy`，并且 runtime channel selection 会优先命中该 policy。现有激活校验会检查 execution、supplier、channel、ability 与 supplier posture，但没有检查该 self-hosted channel 是否已经通过 SLA runtime probe。

这会留下一个 P3 缺口：自持算力的线下执行记录可以直接改变生产流量，而没有对应的 runtime SLA 测量证据。ADR 0047 已经能把失败的 runtime run 送入 Operating Insights；现在需要把通过的 runtime run 变成 routing policy 激活的前置证据。

## 决策

扩展 `SupplyRoutingPolicy`：

1. 增加只读 SLA evidence 字段：
   - `sla_contract_id`
   - `sla_probe_run_id`
   - `sla_probe_run_key`
   - `sla_artifact_sha256`
   - `sla_runtime_ref`
2. `ActivateSupplyRoutingPolicy` 在原有 execution / supplier / channel / ability 校验后，必须找到一条同 supplier、channel、model、SLA tier 的 passed runtime `SlaProbeRun`：
   - `status=passed`
   - `hard_gate_passed=true`
   - plan `probe_type` 为 `runtime_light` 或 `runtime_deep`
   - run 关联的 `SlaContract.status=active`
3. 找到的 latest run 会复制到 routing policy 的 evidence 字段；重复 activate 会刷新 evidence 字段并清空 disabled metadata。
4. 缺少 passed runtime evidence 时，激活失败，operator 应先用 `token-router-sla` 或外部 runner 生成并 record probe run。
5. 失败、invalid、cancelled 或 hard-gate failed run 不会激活 policy；它们继续通过 ADR 0047 进入 Operating Insights 复盘面。

## 边界

1. 不在 policy activation 中执行 benchmark。
2. 不自动创建 contract、plan、run、supplier、channel 或 capacity。
3. 不把 passed admission run 当作 runtime routing evidence。
4. 不自动 disable 已有 policy；如果后续 runtime probe 失败，只生成 insight，由 operator 决定是否 disable。
5. 不改变 pricing、billing、settlement 或 payment 语义。

## 验收

1. 缺少 passed runtime SLA run 的 self-hosted execution 不能 activate routing policy。
2. 有同 supplier/channel/model/SLA tier 的 passed runtime run 后，policy 可以 activate，并持久化 linked SLA evidence。
3. 非 passed / hard gate failed / admission-only run 不能作为 policy evidence。
4. HTTP e2e 证明 contract -> runtime plan -> passed run -> policy activation -> self-hosted demand route -> ledger 的链路。
5. focused Go tests 与必要前端类型检查通过，README / architecture 记录证据。
