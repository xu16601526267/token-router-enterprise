# ADR 0057: action plan opportunity evidence

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P5 agent 出洞察、人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0020 Supply action plans；0055 Supply expansion opportunities；0056 Supply expansion opportunity dashboard

## 背景

ADR 0055 / 0056 已经把 `SupplyDecision` 进一步物化为 `SupplyExpansionOpportunity`，并在 dashboard 展示 opportunity type、priority、cluster、locality/stability/headroom risk 和 rank score。

当前缺口是后续 `SupplyActionPlan` 仍只复制 `SupplyDecision` 的字段。operator 可以在 `Opportunities` tab 看到“为什么这是高优先级机会”，但生成 action plan 后，这些 L2 排序证据不会随工作项进入 handoff。这样会让 “L2 分析 → 人审 → 运营工作项” 之间丢失 opportunity 证据，后续 execution / routing 审计也只能回查 decision，不能直接看到当时的机会排序来源。

## 决策

在 `SupplyActionPlan` 上新增只读 opportunity evidence 字段：

1. `supply_expansion_opportunity_id`
2. `opportunity_key`
3. `opportunity_type`
4. `opportunity_priority`
5. `opportunity_cluster_key`
6. `opportunity_rank_score`

`GenerateSupplyActionPlans` 仍只从 approved `SupplyDecision` 生成 action plan；如果同一 `supply_decision_id` 已存在 `SupplyExpansionOpportunity`，则复制上述 opportunity evidence。若不存在 opportunity，仍按旧行为生成 action plan，opportunity 字段为空 / 0，保持向后兼容。

重复 generate 继续以 `supply_decision_id` 幂等 upsert，并刷新可复算的 opportunity evidence，但不修改已进入 lifecycle 的 status、started/completed/cancelled timestamps 或 operator note。

dashboard 的 `Action Plans` 表展示 opportunity evidence，使 operator 在工作项视图中能看到对应 opportunity type、priority、cluster 和 rank score。

## 不做什么

1. 不允许 opportunity 直接创建 action plan；approved `SupplyDecision` 仍是唯一入口。
2. 不在 action plan generate 时自动生成 `SupplyExpansionOpportunity`；机会排序仍由显式 generate/query 链路产生。
3. 不新增 opportunity review 状态机。
4. 不从 opportunity 或 action plan 自动创建 supplier/channel/capacity。
5. 不自动激活 routing policy，不修改价格，不触碰账单、结算或资金动作。

## 验收

1. model 测试证明已有 opportunity 时，action plan 会复制 opportunity id/key/type/priority/cluster/rank。
2. HTTP E2E 和 `token-router-sim run` 证明 gb10-4t 主链路中 action plan 回读包含 opportunity evidence。
3. dashboard `Action Plans` 表展示 opportunity evidence，并通过 i18n/typecheck/lint/build 验证。
4. 在 `aima2` 通过相关 Go tests 与真实进程 simulator。
