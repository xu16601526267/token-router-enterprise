# ADR 0059: capacity telemetry operating insights

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P3 先度量再承诺 SLA；P5 人审；P8 守住边界；P9 流量即情报
- 关联 ADR：0012 Supply capacity snapshots；0034 Operating insights；0050 Ledger-backed supply capacity usage refresh；0058 Supply capacity telemetry evidence

## 背景

ADR 0058 已经把 `SupplyCapacityTelemetry` 接入 capacity snapshot，让 operator 能看到容量、GPU utilization、quality、unit cost 的来源证据。

剩余缺口是：这些遥测只停留在 capacity 表展示里。若某个 supply node 的遥测缺失、过期、GPU utilization 已接近饱和，或 token headroom 很低，系统还不能把它提升成经营复盘面上的可审核信号。这样 operator 需要手动扫 capacity 表，P9 的供给侧情报还没有进入统一的 `OperatingInsight` 面。

## 决策

扩展 `/api/operating_insights/generate`，在既有 profile / SLA run insight 之外，读取同周期重叠的 enabled `SupplyCapacity`，并为以下情况生成 `OperatingInsight`：

1. 缺少最近 telemetry evidence：`last_telemetry_id = 0` 或 `telemetry_observed_at = 0`。
2. telemetry evidence 过期：以 generation period 的结束时间和当前时间中较早者为参考点，最近观测时间超过 freshness threshold。
3. GPU utilization 高：`gpu_utilization_rate >= 0.9`。
4. token headroom 很低：`capacity_tokens > 0` 且 `headroom_tokens <= 10% capacity_tokens`。

生成的 insight：

- `category = capacity_risk`
- `status = draft`
- 缺失 / 过期 telemetry 使用 `severity = watch`
- 高 GPU 或低 headroom 使用 `severity = action`
- key 按 supplier、node、model、period、risk reason 幂等 upsert
- `slice_key` 记录 capacity source，不绑定 user
- `supply_headroom_tokens`、`avg_unit_cost_quota`、summary、recommended action 携带可审计证据

`OperatingInsight` 继续保持 review 状态：重复 generate 不覆盖 acknowledged / dismissed 状态，只刷新事实字段。

## 边界

1. 不新增后台采集器或告警系统。
2. 不自动创建、修改或禁用 supplier/channel/capacity/routing policy。
3. 不自动调权、不改价、不审批 supply decision。
4. 不触碰账单、结算或资金动作。
5. 不把 stale/high-utilization insight 当成 SLA 通过或失败的证据；它只是 operator 复盘信号。

## 验收

1. model 测试覆盖 missing telemetry、stale telemetry、高 GPU / 低 headroom insight，并证明重复 generate 保留 review 状态。
2. HTTP E2E 在 gb10-4t 主链路外增加一条 hot node telemetry，证明 `/api/operating_insights/generate` 可生成并查询 `capacity_risk`。
3. 真实进程 simulator 输出 `capacity_telemetry_insight_verified=true`。
4. 在 `aima2` 通过 targeted Go tests 与真实进程 simulator，README / architecture / traffic docs / product principles 记录本轮证据。
