# ADR 0062: ledger-backed supply action execution drawdown

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P5 人审；P6 成本透明；P8 守住边界；P9 流量即情报
- 关联 ADR：0024 Supply action executions；0026 Self-hosted routing policies；0050 Ledger-backed supply capacity usage refresh；0058 Supply capacity telemetry evidence；0060 Self-hosted cost basis evidence

## 背景

`SupplyActionExecution` 已经记录 completed `SupplyActionPlan` 的线下执行结果，并能作为 self-hosted routing policy 的人工激活来源。`SupplyCapacity.used_tokens` 也已经能通过 ADR 0050 从 `UsageLedger` 回填。

剩余缺口是：operator 能看到执行记录的 actual capacity / unit cost / capacity snapshot，但看不到该 execution 对应的真实业务账本消耗，也就无法把自营库存或自持算力从"已上线"推进到"已核销多少、剩余多少"。如果这部分仍靠人工备注，P9 的需求账本与供给执行事实层会断开。

## 决策

新增显式 admin refresh 能力，把 successful `UsageLedger` 聚合成 `SupplyActionExecution` 的 drawdown/read-model：

1. 在 `SupplyActionExecution` 增加 `drawdown_tokens`、`drawdown_request_count`、`remaining_tokens`、`drawdown_rate`、`drawdown_source_type`、`drawdown_source_ref`、`drawdown_refreshed_at`。
2. 新增 `POST /api/supply_action_executions/refresh_usage`，支持按 execution / plan / supplier / channel / capacity / track / period 过滤已有 execution。
3. refresh 只扫描已有 `SupplyActionExecution` rows，不创建 action plan、supplier、channel、capacity、routing policy 或 settlement。
4. 对每条 execution，用同 supplier、可选 channel、可选 capacity supply node、model、SLA tier、user、effective window 的 successful `UsageLedger` 聚合；账本未携带 SLA tier（空字符串）或 user 维度（`user_id=0`）时视为未分维度，仍可匹配该 execution：
   - `drawdown_tokens = SUM(prompt_tokens + completion_tokens)`
   - `drawdown_request_count = COUNT(*)`
   - `remaining_tokens = actual_capacity_tokens - drawdown_tokens`
   - `drawdown_rate = drawdown_tokens / actual_capacity_tokens`
5. effective window 优先使用 `effective_from/effective_to`；为空时回退到 action plan copied `period_start/period_end`。
6. `record` execution 会替换执行事实并清空旧 drawdown evidence，避免 supplier/channel/capacity/actual capacity 变更后沿用旧核销结果。
7. 默认 `/token-router` dashboard 的 `Executions` tab 展示 drawdown evidence，并提供显式刷新按钮。

## 边界

1. 不实现支付、钱包、打款、采购、发票或资金状态。
2. 不把 drawdown 当成 settlement statement，不生成应收应付。
3. 不自动修改 `actual_capacity_tokens`、`unit_cost_quota`、capacity snapshot、channel 权重或 routing policy。
4. 不新增后台定时任务；第一版保持显式 admin API、dashboard 按钮和 simulator 验证。
5. 不自动探测真实硬件 quota 或上游剩余额度。

## 验收

1. 模型测试证明 execution drawdown 只聚合同 supplier/channel/node/model/SLA/user/window 的 successful ledger usage，并回写剩余量、使用率和 source evidence。
2. HTTP / simulator 验证 self-hosted routing 产生 ledger 后，`/api/supply_action_executions/refresh_usage` 能回填对应 execution drawdown。
3. dashboard 能刷新并展示 execution drawdown，不自动触发路由、结算、采购或资金动作。
4. focused Go tests 在 `aima2` 通过；前端 typecheck/i18n/targeted lint/build 通过；README / architecture / product principles 记录本轮证据与仍未覆盖的真实硬件自动采集边界。
