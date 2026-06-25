# ADR 0058: supply capacity telemetry evidence

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P3 先度量再承诺 SLA；P8 守住边界；P9 流量即情报
- 关联 ADR：0012 Supply capacity snapshots；0050 Ledger-backed supply capacity usage refresh；0057 Action plan opportunity evidence

## 背景

`SupplyCapacity` 当前已经能记录 supplier / node / model / period 的 capacity、used、headroom、quality 和 unit cost，并且 `refresh_usage` 可以从 `UsageLedger` 回填业务已用 token。

剩余缺口是：这些 capacity snapshot 仍主要是人工或 seed 数据，没有记录“这条容量、GPU 利用率、质量分和单位成本来自哪一次节点遥测”。这会让自持 / 自营供给进入更真实运营时缺少可审计来源，也让 P9 的供给侧情报仍停留在人工快照层。

## 决策

新增 `SupplyCapacityTelemetry` 作为容量遥测证据表，并给 `SupplyCapacity` 增加最近一次遥测证据字段：

1. `gpu_utilization_rate`
2. `telemetry_source_type`
3. `telemetry_source_ref`
4. `telemetry_observed_at`
5. `last_telemetry_id`

新增 `POST /api/supply_capacity_telemetries/record`：

- 输入 supplier、supply node、model、period、capacity tokens、used tokens、GPU utilization、quality score、unit cost、source type、source ref、observed_at。
- `source_ref` 必填；如果未传 `telemetry_key`，由 source/type/ref + supplier/node/model/period 生成幂等 key。
- 记录 / upsert `SupplyCapacityTelemetry`，并在同一事务里 upsert exact-period `SupplyCapacity`。
- `SupplyCapacity` 继续用 `capacity_tokens - used_tokens` 计算 headroom，用 `used_tokens / capacity_tokens` 计算 utilization；`gpu_utilization_rate` 是独立硬件利用率证据，不替代 token utilization。

新增 `GET /api/supply_capacity_telemetries` 查询遥测证据，方便 operator/API 对照当前 capacity snapshot 的来源。

dashboard 的 `Supply Capacity` 表展示 GPU utilization 和最近遥测 source evidence。该表仍是事实展示，不是路由策略。

## 不做什么

1. 不从遥测自动创建 supplier/channel。
2. 不自动创建或修改 `SupplyRoutingPolicy`。
3. 不自动调权、不自动禁用供应商或 channel。
4. 不触碰价格、账单、结算或资金动作。
5. 不实现后台定时采集器；本 ADR 只提供可审计的 record/query API，后续采集器可以调用该 API。

## 验收

1. model 测试证明 telemetry record 会幂等写入遥测证据，并 upsert 对应 `SupplyCapacity` 的 capacity/used/headroom/utilization/GPU utilization/source fields。
2. HTTP E2E 和 `token-router-sim run` 证明 gb10-4t 主链路中可记录并查询 capacity telemetry，且后续 capacity/profile/scorecard 仍使用更新后的 capacity snapshot。
3. dashboard `Supply Capacity` 表展示 GPU utilization 和 telemetry source evidence，并通过 i18n/typecheck/lint/build 验证。
4. 在 `aima2` 通过相关 Go tests 与真实进程 simulator。
