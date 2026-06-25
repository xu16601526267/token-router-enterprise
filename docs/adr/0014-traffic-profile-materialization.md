# ADR 0014: Traffic profile materialization

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P3 先度量再承诺 SLA、P5 人在 dashboard 决策、P9 流量即情报
- 关联架构：T1 画像 + dashboard、T2 供给决策建议、`UsageLedger` 事实层、`SupplyCapacity` 周期快照

## 背景

当前系统已经有：

1. `UsageLedger`：每次成功调用一行，包含 session、cache 拆分、双价、质量遥测和供给节点。
2. quality summary：按 supplier/channel/model/SLA/节点做只读质量聚合。
3. `SupplyCapacity`：按 supplier/node/model/period 记录供给容量、已用量、余量、利用率、质量分和单位成本。

这些能力仍偏“查询事实”。`traffic-and-supply.md` 的 L1 画像层需要把需求侧和供给侧合成可复用的周期画像：某个切片在一个周期里有多少需求、cache 局部性怎样、毛利怎样、质量是否达标、供给余量多少。没有这层物化画像，后续缺口检测、自营采购和自持算力 ROI 都会重复写临时聚合逻辑。

## 决策

新增 `TrafficProfile` 物化表与 admin API：

1. 切片维度先定为 `model_name + sla_tier + user_id`。
2. 空 `sla_tier` 归一为 `default`，表示尚未商业分层的默认质量档。
3. `POST /api/traffic_profiles/generate` 按周期从 `UsageLedger` 聚合并 upsert profiles。
4. `GET /api/traffic_profiles` 支持按周期、model、SLA、user 查询已物化 profiles。
5. profile 字段包含：
   - 需求：`request_count`、`demand_tokens`、`peak_tokens`、`peak_ratio`、`unique_sessions`
   - cache：`cache_hit_count`、`cache_hit_rate`、`total_cached_tokens`
   - 质量：`success_request_count`、`sla_met_rate`、`avg_latency_ms`、`max_latency_ms`
   - 经济：`total_sell_quota`、`total_cost_quota`、`gross_profit_quota`
   - 供给：`supply_capacity_tokens`、`supply_used_tokens`、`supply_headroom_tokens`、`avg_supply_quality_score`、`avg_unit_cost_quota`
6. `peak_tokens` 第一版按 profile 周期内单小时 token 需求峰值计算，`peak_ratio = peak_tokens / active_hour_avg_tokens`。

## 边界

本轮不做：

1. 不做自动供给决策或自动调权。
2. 不做 `SupplyDecision` 表或 agent 建议审批流。
3. 不从 profile 反写 supplier/channel 状态。
4. 不承诺 SLA；`sla_met_rate` 第一版等同于成功请求占比，用于后续 SLA 阈值设计前的事实占位。
5. 不做后台定时任务；先提供按需 generate API 和模拟器验证。

## 影响

正向影响：

- L1 画像层开始从“临时查询”变成“可复用事实表”。
- `gb10-4t` E2E 能同时证明需求、cache、毛利、质量和供给余量进入同一 profile。
- 后续缺口分析和自持 ROI 可以直接读取 `TrafficProfile`，不重复扫 `UsageLedger` 和 `SupplyCapacity`。

代价：

- 第一版 profile 是按需生成，不保证实时。
- `sla_met_rate` 尚未接入正式 SLA 阈值，只代表当前成功率。
- 峰值按小时桶计算，足够支撑起步画像，后续可扩展到更细粒度。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `TrafficProfile` 已进入普通迁移和 fast migration。
2. `/api/traffic_profiles/generate` 能基于 `gb10-4t` 两次请求生成 profile。
3. `/api/traffic_profiles` 能查询到该 profile，并校验 demand/cache/margin/SLA/supply headroom。
4. `token-router-sim run` 能在真实进程链路中核验 profile API。
5. `aima2` 上 focused Go 测试和 `go test ./...` 通过，README 记录证据。
