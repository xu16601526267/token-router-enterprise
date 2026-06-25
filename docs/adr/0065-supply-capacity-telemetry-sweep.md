# ADR 0065: supply capacity telemetry sweep

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P5 人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0058 Supply capacity telemetry evidence；0059 Capacity telemetry operating insights；0064 Upstream capacity telemetry collector

## 背景

ADR 0064 已经证明系统可以从一个已配置 channel upstream 的固定 telemetry endpoint 拉取 capacity telemetry，并复用 `RecordSupplyCapacityTelemetry` 写入 `SupplyCapacityTelemetry` / `SupplyCapacity`。

但单条 collect 仍要求 operator 或外部脚本逐条传 `channel_id`、supplier、node、model 和 period。要让后续专属服务器上的 cron、scheduler 或 fleet agent 低成本落地，需要一个批量 sweep 入口：它从现有 capacity snapshot 出发，自动找到同 supplier / model 的 enabled channel，逐条调用同一 collector，并返回 collected / skipped 的可审计结果。

## 决策

新增显式 admin sweep 能力：

1. 新增 `POST /api/supply_capacity_telemetries/sweep`。
2. sweep input 支持按 supplier、supply node、model、period、channel 过滤已有 `SupplyCapacity`。
3. 对每条候选 capacity：
   - 如果 input 指定 `channel_id`，使用该 channel，但要求 supplier 匹配。
   - 如果未指定，查找同 supplier、enabled、支持该 model、base URL 非空的 channel。
   - 调用 ADR 0064 的 `CollectSupplyCapacityTelemetry`，保持相同 upstream path、headers、record/upsert 语义。
4. sweep response 返回：
   - `attempted_count`
   - `collected_count`
   - `skipped_count`
   - `collected` telemetry rows
   - `skipped` entries（supplier/node/model/period/reason）
5. 单条 collect 失败不会让整个 sweep 失败；失败被记录到 `skipped`，用于 operator / cron 查看。
6. `token-router-sim run` 使用 sweep 验证 gb10-4t 主链路 telemetry 采集，并输出 `capacity_telemetry_sweep_verified=true`。

该能力是“可调度入口”，不是内置后台 worker。部署侧可以用 cron 调用它；系统本身仍不自动调权、不禁用 channel、不激活 policy。

## 边界

1. 不新增常驻后台 goroutine、分布式调度器、队列、重试退避或 fleet agent。
2. 不扫描任意 URL；只使用数据库已配置 channel base URL 和 ADR 0064 固定 path。
3. 不自动创建 supplier/channel/capacity；sweep 只扫描已有 capacity snapshot。
4. 不自动修改路由权重、不禁用 supplier/channel、不创建 action plan、不激活 routing policy。
5. 不触碰采购、付款、发票、账单、结算或资金状态。

## 验收

1. model 测试证明 sweep 能从已有 capacity 自动选择同 supplier/model channel，调用 upstream telemetry endpoint，并 upsert telemetry / capacity source evidence。
2. model 测试或 simulator 证明无法采集的 capacity 会进入 skipped，不阻断其他 capacity。
3. 真实进程 simulator 输出 `capacity_telemetry_sweep_verified=true`，DB 回读 `source_ref=gb10-4t-mock-capacity`、`used_tokens=300`、`last_telemetry_id` 指向 sweep 采集结果。
4. focused Go tests 在 `aima2` 通过；README / architecture / product principles / traffic-and-supply 记录本轮证据，并保留后台 worker、fleet agent、资金核销和自动执行边界。
