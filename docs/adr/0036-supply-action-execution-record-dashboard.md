# ADR 0036: Supply action execution record dashboard

- 状态：Accepted
- 日期：2026-06-23
- 关联原则：P5 agent 出建议、人审执行；P8 平台只做数据与信任中枢；P9 流量即情报
- 关联 ADR：0024 Supply action executions；0025 Supply action execution dashboard；0027 Supply routing policy dashboard

## 背景

ADR 0024 已经新增 `SupplyActionExecution` 与 `POST /api/supply_action_executions/record`，用于把 completed `SupplyActionPlan` 的线下执行结果登记成事实记录。ADR 0025 只把 execution records 做成只读 dashboard，ADR 0027 再从 recorded self-hosted execution 显式 activate routing policy。

这留下一个 operator workflow 缺口：dashboard 可以把 action plan 推进到 completed，也可以把 recorded execution 提升为 routing policy，但中间的“登记 execution record”仍需要离开 `/token-router` 调 API 或依赖模拟器。为了让供给行动计划 → 执行事实 → 路由策略这段控制面闭环可由同一个 dashboard 人工驱动，需要补上记录 execution 的表单。

## 决策

在默认 admin console `/token-router` 增加 execution record 操作：

1. 在 `Action Plans` tab 的 completed plan 行提供 `Record Execution` 操作。
2. 在 `Executions` tab 提供 `Record Execution` 操作，并列出当前周期 completed action plans 作为来源。
3. 调用既有 `POST /api/supply_action_executions/record`，提交：
   - `supply_action_plan_id`
   - `execution_status=recorded`
   - 可选 `supplier_id`、`channel_id`、`supply_capacity_id`
   - `actual_capacity_tokens`、`unit_cost_quota`
   - 可选 `effective_from`、`effective_to`
   - `external_ref`、`operator_note`
4. 成功后刷新 execution records、routing source executions 与 routing policies。

该表单只记录 operator 已在线下完成的事实；同一 action plan 重复提交继续复用后端幂等 upsert 语义。后续是否进入 routing policy，仍必须由 `Routing Policies` tab 显式 activate。

## 不做什么

1. 不自动创建或修改 supplier、channel、capacity。
2. 不自动采购、不自动扩容、不自动打款、不自动结算。
3. 不自动 activate routing policy。
4. 不改变 `SupplyActionExecution` 后端 schema 或校验规则。
5. 不把 completed action plan 解释为 execution 已发生；只有 record API 成功后才产生 execution fact。

## 验收

1. completed action plan 可以在 dashboard 打开 execution record dialog。
2. `Executions` tab 可以从 completed action plan 列表发起 record。
3. 表单提交调用 `/api/supply_action_executions/record`，并刷新 execution records。
4. i18n 覆盖 en / zh / fr / ja / ru / vi。
5. TypeScript typecheck、i18n sync、targeted lint、production build 通过。
6. Playwright WebKit 以 API mocks 验证 desktop/mobile dialog、record POST payload、recorded execution row 刷新。
