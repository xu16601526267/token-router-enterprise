# ADR 0052: traffic forecast materialization

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P5 人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0014 Traffic profile materialization；0015 Supply decision recommendations；0034 Operating insights；0050 Ledger-backed supply capacity usage refresh

## 背景

`TrafficProfile` 已经把历史业务流量、cache、毛利、SLA attainment 和 supply headroom 物化到周期画像。`SupplyDecision` 能读取单周期 profile 生成三轨建议，`OperatingInsight` 能把建议合成可复盘假设。

但产品原则 P9 还要求“流量即情报”：平台不仅要解释已经发生的流量，还要把近期趋势变成下一周期的经营判断输入。当前缺口是没有一个稳定、可查询、可测试的 forecast 事实层；如果直接把预测逻辑塞进 `SupplyDecision`，后续很难区分“历史事实”“预测假设”和“人审决策”。

## 决策

新增 `TrafficForecast` 物化表与 admin API：

1. `TrafficForecast` 以 model / SLA / user slice 为粒度，记录：
   - source period：用于计算预测的历史窗口
   - target period：预测要覆盖的下一周期
   - source profile count、observed demand / peak / request count
   - forecast demand / peak / supply headroom / gap
   - cache hit rate、SLA met rate、gross profit、avg unit cost、confidence、method、reason
2. `POST /api/traffic_forecasts/generate` 从已有 `TrafficProfile` 生成 forecast：
   - 第一版 `method=moving_average`
   - demand 用 source profiles 的平均值
   - peak 用 source profiles 的最大 peak，保持容量风险保守
   - headroom 用最新 source profile 的 supply headroom
   - gap = `max(forecast_peak_tokens - forecast_headroom_tokens, 0)`
   - confidence = `min(source_profile_count / 3, 1)`
3. `GET /api/traffic_forecasts` 支持 model、SLA、user、source/target 周期过滤。
4. `token-router-sim run` 在 traffic profile 后生成 forecast，并验证 gb10-4t 真实进程链路产生可查询 forecast。

## 边界

1. 不引入 LLM、机器学习模型、外部数据仓库或定时任务。
2. 不自动创建或修改 `SupplyDecision`、`SupplyActionPlan`、supplier、channel、capacity、routing policy、pricing、billing 或 settlement。
3. forecast 是经营假设，不是 SLA 承诺或采购承诺。
4. 第一版只使用 `TrafficProfile`，不直接扫 `UsageLedger`，避免预测层绕过已验证的画像事实层。
5. 后续如果要把 forecast 纳入 supply decision 权重或自动复盘，需要单独 ADR。

## 验收

1. model 测试证明多周期 profile 可以生成幂等 forecast，并正确计算 moving average demand、max peak、latest headroom、gap 和 confidence。
2. HTTP e2e 证明 `/api/traffic_forecasts/generate` 与 query API 可用，且 forecast 读取 gb10-4t profile 数据。
3. 真实进程 simulator 输出 `traffic_forecast_verified=true`。
4. focused Go tests 与 aima2 真实进程验证通过，README / architecture / product principles 记录本轮证据。
