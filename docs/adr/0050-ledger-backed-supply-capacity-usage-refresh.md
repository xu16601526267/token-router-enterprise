# ADR 0050: ledger-backed supply capacity usage refresh

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P5 人审；P8 守住边界；P9 流量即情报
- 关联 ADR：0012 Supply capacity snapshots；0014 Traffic profile materialization；0018 Supplier scorecards；0049 Real-process routing SLA evidence E2E

## 背景

`SupplyCapacity` 已经作为供给侧周期快照进入 T0 数据层，字段包含 capacity、used、headroom、utilization、quality score 和 unit cost。ADR 0012 的第一版明确 `used_tokens` 由写入方提供，不从 `UsageLedger` 回填。

现在真实业务链路已经能把 gb10-4t demand traffic 写入 `UsageLedger`，后续 `SupplierScorecard`、`TrafficProfile`、`SupplyDecision` 又会读取 `SupplyCapacity.used_tokens/headroom_tokens`。如果 used 仍靠 seed 或人工填写，供给余量就不是来自同一条事实台账，P9 的“流量即情报”会在供给侧断开。

## 决策

新增显式 admin refresh 能力：

1. 新增 `SupplyCapacityUsageRefreshInput` 和 `RefreshSupplyCapacityUsage`。
2. 新增 `POST /api/supply_capacities/refresh_usage`。
3. refresh 只扫描已有 `SupplyCapacity` rows，不创建 supplier、channel、capacity 或 policy。
4. 对每条 capacity，用同 supplier、supply node、model、period 的 successful `UsageLedger` 聚合：
   - `used_tokens = SUM(prompt_tokens + completion_tokens)`
   - 空 `supply_node` 表示该 supplier/model 的所有节点
   - 空 `model_name` 表示该 supplier/node 的所有模型
5. refresh 只回写 `used_tokens`、`headroom_tokens`、`utilization_rate`、`updated_time`。
6. `token-router-sim run` 在生成 scorecard/profile 前调用 refresh，并校验 capacity used/headroom 来自真实 ledger demand tokens。

## 边界

1. 不自动探测真实硬件容量、quota 水位或上游剩余额度。
2. 不改写 `capacity_tokens`、`quality_score`、`unit_cost_quota`。
3. 不自动调 channel 权重、supplier 状态、routing policy 或 pricing。
4. 不新增后台定时任务；第一版保持显式 admin API 和 simulator 验证。
5. 不触碰支付、库存采购、付款、发票或资金状态。

## 验收

1. 模型测试证明 stale capacity `used_tokens` 会被同周期 successful ledger usage 覆盖。
2. HTTP e2e 证明 `/api/supply_capacities/refresh_usage` 会回写 gb10-4t capacity，并且后续 scorecard/profile 读取刷新后的 supply headroom。
3. 真实进程 simulator 输出 `capacity_usage_refresh_verified=true`，证明 demand ledger -> capacity refresh -> scorecard/profile/supply decision 链路。
4. focused Go tests 与 aima2 真实进程验证通过，README / architecture / product principles 记录本轮证据。
