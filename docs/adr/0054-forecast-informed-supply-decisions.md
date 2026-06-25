# ADR 0054: forecast-informed supply decisions

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P5 agent 出洞察、人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0014 Traffic profile materialization；0015 Supply decision recommendations；0052 Traffic forecast materialization；0053 Traffic forecast dashboard

## 背景

ADR 0052/0053 已经让 `TrafficForecast` 从 `TrafficProfile` 物化，并在 `/token-router` dashboard 展示 forecast source window、target window、confidence、gap 和生成方法。

但 `SupplyDecision` 仍只读取历史 `TrafficProfile`。这样 operator 虽然能看到下一周期 forecast，却必须人工把 forecast gap / confidence 和 supply decision 对齐；P9 的“流量即情报”仍没有真正进入三轨供给建议。

## 决策

`GenerateSupplyDecisions` 在读取 source period 的 `TrafficProfile` 后，优先查找同 slice / source period 的 `TrafficForecast`：

1. 如果 forecast 存在，draft `SupplyDecision` 的 demand / peak / headroom / gap / cache / SLA / gross profit / unit cost / ROI / reason 使用 forecast evidence。
2. 保留原有 `decision_key = profile:<slice>|period:<source>`，并保留 `TrafficProfileId`，避免破坏既有 review、OperatingInsight、ActionPlan、Execution、RoutingPolicy 链路。
3. 在 `SupplyDecision` 上新增只读 forecast evidence 字段：`traffic_forecast_id`、`decision_source`、`forecast_target_period_start`、`forecast_target_period_end`、`forecast_confidence`、`forecast_method`。
4. 如果 forecast 不存在，继续使用 profile-only 逻辑，并将 `decision_source=profile`。
5. repeated generate 会刷新可复算 evidence，但不清空已 approve/reject 的 review 状态。

该改变只让供给建议引用 forecast 作为证据，不自动 approve/reject，不自动创建 action plan，不激活 routing policy，不改价，不修改 supplier/channel/capacity，不触碰账单、结算或资金动作。

## 不做什么

1. 不改变 ADR 0052 的 moving-average forecast 算法。
2. 不把 forecast decision period 切到 target period；当前为了保持下游链路稳定，decision 仍挂在 source profile period，但额外记录 forecast target window。
3. 不新增后台定时任务；operator 或 simulator 仍显式调用 generate。
4. 不允许 forecast 自动执行供应商招募、自营采购或自持算力路由。

## 验收

1. 有匹配 `TrafficForecast` 时，`SupplyDecision` 返回 forecast evidence 字段并使用 forecast demand / peak / headroom / gap。
2. 无 forecast 时，既有 profile-only decision 行为保持兼容。
3. approved/rejected decision repeated generate 不丢 review 状态。
4. model / controller / simulator / E2E 覆盖 forecast-informed decision。
5. 在 `aima2` 通过相关 Go tests；本机 docs/frontend schema 做静态校验。
