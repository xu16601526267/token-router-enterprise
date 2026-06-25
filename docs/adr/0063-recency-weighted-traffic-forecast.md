# ADR 0063: recency-weighted traffic forecast

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P5 人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0014 Traffic profile materialization；0052 Traffic forecast materialization；0054 Forecast-informed supply decisions；0055 Supply expansion opportunities

## 背景

ADR 0052 已经把 `TrafficProfile` 物化为下一周期 `TrafficForecast`，并用 `method=moving_average` 作为第一版算法。ADR 0054/0055 又把 forecast evidence 接入 `SupplyDecision` 与 `SupplyExpansionOpportunity`，让供给建议不只读取单周期历史画像。

剩余问题是：等权 moving average 对趋势变化反应太慢。连续 source profiles 中，最新 profile 往往更能代表下一周期的 cache、SLA、成本、毛利和需求强度；如果仍把旧周期和最新周期同权，供给建议会低估近期需求抬升，也会滞后反映成本改善或恶化。

## 决策

把 `GenerateTrafficForecasts` 的默认生成方法升级为 `method=weighted_moving_average`：

1. 保留 `TrafficForecastMethodMovingAverage = "moving_average"` 常量与查询兼容性，但新生成的 forecast 默认写入 `TrafficForecastMethodWeightedMovingAverage = "weighted_moving_average"`。
2. source profiles 仍按 `period_start ASC, id ASC` 读取，并在同 slice 内按时间顺序分配线性 recency weight：
   - 最旧 profile weight = 1
   - 后续 profile weight 逐个 +1
   - 最新 profile 权重最高
3. forecast demand、cache hit rate、SLA met rate、gross profit、avg unit cost 使用 weighted average：
   - 整数类 demand / gross profit 使用向上取整，避免低估需求和收益规模
   - 比率类指标保留 float average
4. peak 继续使用 source profiles 的最大 observed peak，而不是 weighted peak，保持容量规划保守。
5. headroom 继续使用最新 source profile 的 supply headroom，因为这是当前可见供给余量的最近事实。
6. confidence 继续使用 `min(source_profile_count / 3, 1)`，本轮不把 recency 权重误表述为统计置信度。
7. forecast reason 写明 recency-weighted 方法、profile 数、weight sum、latest weight、peak/headroom 规则。

该改变会被下游 `SupplyDecision.forecast_method` 与 `SupplyExpansionOpportunity.forecast_method` 透传为 evidence；下游无需新增字段或迁移。

## 边界

1. 不引入机器学习、季节性模型、异常检测、外部数据仓库或后台定时任务。
2. 不改 forecast table schema，不新增 weight 明细表。
3. 不改变 `TrafficProfile` 的生成口径，也不绕过 profile 直接读取 `UsageLedger`。
4. 不自动 approve/reject `SupplyDecision`，不创建 action plan，不激活 routing policy，不采购、不付款、不结算。
5. 不把 weighted forecast 称为 SLA 承诺；它仍是可复算的经营假设。

## 验收

1. model 测试证明多周期 profile 使用线性 recency weights，最新周期对 demand/cache/SLA/gross profit/unit cost 影响更大，同时 peak 仍取 max observed、headroom 仍取 latest profile。
2. simulator 与 HTTP e2e 证明单周期 gb10-4t 链路生成 `weighted_moving_average` forecast，并继续驱动 supply decision / expansion opportunity evidence。
3. focused Go tests 在 `aima2` 通过；真实进程 simulator 输出 `traffic_forecast_verified=true`，并能从 API 回读 `method=weighted_moving_average`。
4. README / architecture / product principles / traffic-and-supply 记录本轮证据，并保留季节性模型、后台遥测、真实硬件 quota、资金核销和自动执行边界。
