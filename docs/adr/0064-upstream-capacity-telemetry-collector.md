# ADR 0064: upstream capacity telemetry collector

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P5 人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0050 Ledger-backed supply capacity usage refresh；0058 Supply capacity telemetry evidence；0059 Capacity telemetry operating insights；0062 Ledger-backed supply action execution drawdown

## 背景

ADR 0058 已经提供 `SupplyCapacityTelemetry` 的 record/query API，并把遥测证据 upsert 到 `SupplyCapacity`。ADR 0059 也已经能把缺 telemetry、stale telemetry、高 GPU utilization 或低 headroom 变成 `OperatingInsight(category=capacity_risk)`。

剩余缺口是：这些 telemetry evidence 仍由 operator、测试或外部系统直接 POST 写入；系统没有证明能从真实上游 / 自持节点自动拉取容量、已用量、GPU utilization、质量分和单位成本。产品原则 P9 要求供给侧情报不能停在人工快照层，architecture 也把“真实硬件 / 上游 quota 自动采集”列为后续工作。

## 决策

新增显式 admin collect 能力，从已配置 channel 的 upstream base URL 拉取 capacity telemetry，并复用现有 `RecordSupplyCapacityTelemetry` 落库：

1. 新增 `POST /api/supply_capacity_telemetries/collect`。
2. collect input 第一版要求：
   - `channel_id`：用于读取 channel base URL 和 upstream key。
   - `supplier_id`：可选；不传时使用 channel 的 `supplier_id`，传入时必须与 channel supplier 一致。
   - `supply_node`、`model_name`、`period_start`、`period_end`：写入 telemetry / capacity snapshot 的业务维度。
3. collector 只访问 channel base URL 下的固定 path：`/token-router/telemetry/capacity`，并附带 query：
   - `supply_node`
   - `model`
   - `period_start`
   - `period_end`
4. upstream 返回固定 JSON：
   - `capacity_tokens`
   - `used_tokens`
   - `gpu_utilization_rate`
   - `quality_score`
   - `unit_cost_quota`
   - `observed_at`
   - 可选 `source_ref` / `notes`
5. collector 把返回值转换成 `SupplyCapacityTelemetryRecordInput`：
   - `source_type=node_report`
   - `source_ref` 优先使用 upstream response；缺省时使用 channel / endpoint / period 生成稳定来源。
   - `observed_at` 缺省时使用当前时间。
6. `token-router-sim mock-supply` 暴露该 telemetry endpoint，并基于 mock 已处理的请求累计 `used_tokens`，用于真实进程验证。

该能力是“显式采集”，不是后台定时器。后续 scheduler 或部署侧 cron 可以调用同一个 API；本 ADR 先证明系统能从已配置 upstream 自动采集并落库。

## 边界

1. 不实现后台定时任务、队列、告警、自动重试策略或 fleet agent。
2. 不扫描任意 URL；collector 只用数据库中已配置的 channel base URL 和固定 path。
3. 不自动创建 supplier/channel，不自动激活 routing policy，不自动调权，不自动禁用 supplier/channel。
4. 不触碰账单、结算、采购、付款、发票或资金状态。
5. 不把 upstream telemetry 当作 SLA 承诺；它只是供给侧事实证据，会被后续 profile/scorecard/insight 读取。

## 验收

1. model 测试证明 collector 会访问 channel upstream telemetry endpoint，把 response 转成 `SupplyCapacityTelemetry`，并 upsert 对应 `SupplyCapacity` source evidence。
2. HTTP / simulator 验证 gb10-4t mock supply 暴露 telemetry endpoint 后，`/api/supply_capacity_telemetries/collect` 能回填 capacity telemetry，且 `SupplyCapacity.last_telemetry_id` / source fields 指向采集证据。
3. 真实进程 simulator 输出 `capacity_telemetry_collect_verified=true`，并能回读 telemetry row 的 `source_ref=gb10-4t-mock-capacity`、`used_tokens=300`。
4. focused Go tests 在 `aima2` 通过；README / architecture / product principles / traffic-and-supply 记录本轮证据，并保留后台定时采集、真实 fleet agent、资金核销和自动执行边界。
