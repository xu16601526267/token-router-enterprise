# ADR 0067: seasonal and anomaly-aware traffic forecast

## Status

Accepted

## Context

ADR 0063 已把默认 `TrafficForecast` 生成升级为 `weighted_moving_average`，并让 `SupplyDecision` / `SupplyExpansionOpportunity` 透传 forecast evidence。该方法能反映近期趋势，但仍把所有近期变化都当作可延续需求：周期性高低峰不会被单独识别，最新一期异常 spike / drop 也可能直接污染下一周期供给建议。

产品原则 P9 要求流量画像成为供给战略情报；原则 P5 / P8 又要求 agent 只出建议、人审，不自动执行。因此下一步需要让 forecast 层记录可复算的 seasonality / anomaly evidence，但不能把它变成黑盒 ML，也不能自动审批、调权、采购或触碰资金。

## Decision

在 `TrafficForecast` 上新增显式可选的 seasonal / anomaly-aware 生成能力：

1. `POST /api/traffic_forecasts/generate` 增加可选输入：
   - `seasonal_period_count`：source profiles 的周期长度，必须为 `0` 或 `>= 2`。
   - `anomaly_guard`：是否对最新 profile 做异常 spike/drop 防护。
   - `anomaly_threshold_rate`：异常阈值，默认 `2.0`；最新 demand 高于历史均值该倍率为 spike，低于其倒数为 drop。
2. 默认输入不变：未传 seasonal/anomaly 参数时继续生成 `method=weighted_moving_average`，保持现有 dashboard、decision、opportunity 兼容。
3. 传入 seasonal 或 anomaly 参数时生成 `method=seasonal_anomaly_adjusted`，并新增结构化 evidence：
   - `baseline_demand_tokens`：ADR 0063 的 recency-weighted demand。
   - `trend_demand_delta_tokens` / `trend_demand_delta_rate`：最新 source profile 相对最旧 source profile 的 demand 变化。
   - `seasonal_period_count` / `seasonal_index` / `seasonal_demand_tokens`：按 source profile 顺序取 `index % seasonal_period_count`，用目标 bucket 均值相对整体均值的 ratio 调整 baseline demand；历史不足时 index 为 `1`。
   - `anomaly_status` / `anomaly_profile_id` / `anomaly_demand_ratio`：最新 profile 相对前序 profile 均值的状态，取值 `not_evaluated` / `insufficient_history` / `normal` / `spike` / `drop`。
4. anomaly guard 只调整 `forecast_demand_tokens`，不修改 observed evidence。`forecast_peak_tokens` 仍取 source profiles 的 max observed peak，`forecast_headroom_tokens` 仍取 latest profile，`forecast_gap_tokens` 仍由 peak-headroom 计算，保持容量风险保守。
5. 下游 `SupplyDecision` 和 `SupplyExpansionOpportunity` 继续读取 final forecast demand / peak / headroom / gap 与 method；人审语义不变。

## Non-goals

1. 不引入外部数据仓库、实时流处理、机器学习服务、LLM 预测或后台调度。
2. 不新增 forecast 明细表；本轮只在 `TrafficForecast` 上保存必要 evidence。
3. 不绕过 `TrafficProfile` 事实层直接扫描 `UsageLedger`。
4. 不自动 approve/reject decision，不创建 action plan，不激活 routing policy，不采购、不付款、不结算。
5. 不把 forecast 解释为 SLA 或采购承诺。

## Consequences

- 默认 forecast 兼容既有链路；需要更强预测时，operator/agent 可以显式开启 seasonality / anomaly guard。
- Forecast reason 和结构化字段可以解释为什么 final demand 与 baseline demand 不同，方便后续 dashboard 和 agent 复盘。
- 这是可复算的启发式模型，后续若引入更复杂 seasonal model、异常分类或自动复盘，需要单独 ADR。

## Validation

1. model 单测覆盖默认 weighted forecast 兼容，以及 `seasonal_period_count + anomaly_guard` 下的 seasonal index、baseline demand、trend、spike/drop evidence 和 final demand 调整。
2. HTTP / simulator 验证 `/api/traffic_forecasts/generate` 可通过 admin API 生成 `method=seasonal_anomaly_adjusted` forecast，并能查询回读结构化 evidence。
3. `token-router-sim run` 增加专项输出 `traffic_forecast_seasonal_anomaly_verified=true`，但不改变主链路的 supply decision / opportunity 人审边界。
4. focused Go tests 与 aima2 真实进程验证通过；README / architecture / product principles / traffic-and-supply 记录 evidence，并保留自动执行、资金核销和复杂 ML forecast 边界。
