# ADR 0012: Supply capacity snapshots

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P4 供应商优胜劣汰、P5 人在 dashboard 决策、P9 流量即情报
- 关联架构：T0 遥测补齐、T1 画像 + dashboard、供给三轨

## 背景

M0-M3 已经跑通 `gb10-4t` 供给、需求端模拟器、cache-aware 双价台账、会话亲和、毛利报表和对账导出。ADR 0011 又把 `UsageLedger` 里的质量遥测聚合成只读 quality summary。

但 `UsageLedger` 只记录已经消耗的流量。`traffic-and-supply.md` 要求 T0 同时补供给侧周期快照：每个供应商 / 节点 / 模型在一个周期内的额定容量、已用量、余量、质量分和单位成本。没有这层数据，系统只能回答“用了多少、质量怎样”，不能回答“还剩多少供给、哪里有缺口、是否该招募第三方 / 自营采购 / 自持算力”。

## 决策

新增 `SupplyCapacity` 周期快照与 admin API：

1. 新表 `SupplyCapacity`，维度为 `supplier_id + supply_node + model_name + period_start + period_end`。
2. 字段包含 `capacity_tokens`、`used_tokens`、`headroom_tokens`、`utilization_rate`、`quality_score`、`unit_cost_quota`、`status`、`notes`。
3. `headroom_tokens` 与 `utilization_rate` 由写入数据归一化计算：
   - `headroom_tokens = capacity_tokens - used_tokens`
   - `utilization_rate = used_tokens / capacity_tokens`，容量为 0 时记 0
4. 新增 admin CRUD API：`/api/supply_capacities`，支持按 supplier、node、model、status、周期重叠过滤。
5. E2E 和 `token-router-sim seed/run` 为 `gb10-4t` 写入一条 capacity snapshot，并通过 API 核验。

## 边界

本轮不做：

1. 不自动采集真实硬件容量或上游 quota 水位。
2. 不做 `TrafficProfile` 物化表和 `SupplyDecision` 建议表。
3. 不自动调 channel 权重、supplier 状态或三轨供给策略。
4. 不新增支付、库存采购、付款、发票或资金状态。
5. 不承诺 SLA；capacity 与 quality 仍只是运营观测数据。

## 影响

正向影响：

- T0 的供给侧快照有了落库位置，后续 T1 dashboard 可以展示供给余量。
- `gb10-4t` 不再只是一个 channel/supplier 名称，还能被记录为有容量、余量和单位成本的供给节点。
- 后续缺口分析、自营采购和自持算力 ROI 可以复用同一 snapshot 事实层。

代价：

- 第一版 capacity 数据由 seed/API 写入，不代表已接入真实供应商遥测。
- `used_tokens` 先由写入方提供，不在本轮自动从 `UsageLedger` 回填；后续可增加周期聚合任务。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `SupplyCapacity` 已进入普通迁移和 fast migration。
2. admin API 可创建、查询、更新和删除 capacity snapshot。
3. 后端 E2E 能通过 `/api/supply_capacities` 读到 `gb10-4t` 的 capacity snapshot，并校验余量与利用率。
4. `token-router-sim run` 能在真实进程链路中核验 capacity API。
5. `aima2` 上 Go 测试通过，README 记录本轮证据。
