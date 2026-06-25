# ADR 0027: Supply routing policy dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P5 人在回路、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T3 自持算力接入、`SupplyRoutingPolicy`、默认 admin console

## 背景

ADR 0026 已经把 recorded self-hosted `SupplyActionExecution` 转成显式 `SupplyRoutingPolicy`，并让 channel selection 在匹配切片时优先读取 active policy。这个闭环已经能通过 API 和真实进程模拟器证明 self-hosted traffic routing。

但 P5 的当前模式是 agent / 系统给出洞察与建议，人在 dashboard 上批准后才生效。只有 API 入口还不够：operator 需要在默认 admin console 中看到哪些 self-hosted execution 可以转成 routing policy，哪些 policy 正在影响流量，以及如何人工禁用 policy。

## 决策

在 `/token-router` 默认 admin console 增加 `Routing Policies` tab：

1. 查询 `SupplyRoutingPolicy`，支持 `All / active / disabled` status filter。
2. 查询同周期 recorded self-hosted `SupplyActionExecution`，展示可激活来源 execution。
3. 对有 channel / supplier / capacity 引用的 self-hosted execution，提供 `Activate Policy` 操作，调用 `POST /api/supply_routing_policies/activate`。
4. 对 active policy 提供 `Disable Policy` 操作，调用 `POST /api/supply_routing_policies/:id/disable`，并用确认弹窗提示这是路由策略变更。
5. dashboard 展示 policy 的 slice、supplier/channel/capacity、priority、effective period、activated/disabled metadata、operator note 和 source execution/plan/decision。
6. 所有 UI 文案进入现有 i18n locale 覆盖，保持默认 console 的多语言一致性。

## 边界

本轮不做：

1. 不在 dashboard 中创建或修改 supplier / channel / supply capacity。
2. 不从未 recorded、非 self-hosted 或缺 channel 的 execution 激活 policy；后端校验仍是最终边界。
3. 不提供自动激活、批量激活、按权重调优或 policy DSL。
4. 不新增支付、采购、库存、发票或真实部署证明状态。
5. 不改变 `SupplyRoutingPolicy` 的选择语义和成本计算公式。

## 影响

正向影响：

- T3 自持算力路由从 API-only 变成 operator 可见、可操作、可禁用的 dashboard 流程。
- dashboard 明确显示 policy 是人审后的运营配置，而不是 execution fact 自动生效。
- 继续复用现有 period filter、table、stat card、toggle filter、toast 和 i18n 模式。

代价：

- 第一版只支持从已有 execution 激活，不提供复杂编辑；如果 execution 需要修正，仍应先修正执行事实或重新记录。
- policy miss / channel unavailable 告警仍待后续运营监控补齐。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `/token-router` 有 `Routing Policies` tab，能查询 active / disabled / all policies。
2. recorded self-hosted executions 在同页可见，并能触发 activate API。
3. active policy 能从 dashboard disable，并刷新 policy / execution 数据。
4. 新增 UI 文案在 en / zh / fr / ja / ru / vi locale 中覆盖，i18n sync report clean。
5. 前端 typecheck、targeted lint、build 通过，README 记录证据。
