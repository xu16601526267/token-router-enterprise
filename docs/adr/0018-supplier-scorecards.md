# ADR 0018: Supplier scorecards

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P1 严选不做集市、P3 先度量再承诺 SLA、P4 供应商优胜劣汰、P8 平台只做数据与信任中枢
- 关联架构：`UsageLedger` 事实层、`SupplyCapacity` 供给遥测、quality summary、后续供应商评级与分流建议

## 背景

当前系统已经能从 `UsageLedger` 聚合 quality summary，也能记录 `SupplyCapacity` 周期快照；这些数据足以回答某个供应商在一个周期内的请求量、成功率、延迟、cache 命中、毛利、容量余量和单位成本。

但 `Supplier` 仍主要是业务身份与协议成本归集主体，缺少一个稳定的、可回读的周期性质量评分记录。产品原则 P1 / P4 要求“严入选 + 持续评级 + 优胜劣汰”，而路线约束要求“度量先于承诺”。因此需要先把供应商评级的事实层落库，供 dashboard / agent / 人审使用。

## 决策

新增 `SupplierScorecard` 物化表与 admin API：

1. `POST /api/supplier_scorecards/generate`：按周期从 `UsageLedger` 与 `SupplyCapacity` 聚合供应商 scorecard。
2. `GET /api/supplier_scorecards`：按周期、supplier、grade 查询已物化 scorecards。
3. scorecard 记录周期内请求量、成功率、平均/最大延迟、cache hit rate、毛利、容量/余量、平均质量分、平均单位成本。
4. 计算一个 `score`（0-100）和 `grade`（A/B/C/D），作为供应商持续评级的最小事实层。
5. upsert 以 supplier + period 为唯一键；重复 generate 更新事实字段，不追加重复记录。

## 评分规则

第一版使用可解释的简单公式：

```
score =
  success_rate * 40
  + cache_hit_rate * 20
  + min(avg_supply_quality_score, 100) * 0.2
  + latency_score * 10
  + margin_score * 10
```

- `latency_score = max(0, 1 - avg_latency_ms / 5000)`；平均延迟 0ms 得 1，5000ms 及以上得 0。
- `margin_score = 1` 当 `gross_profit_quota > 0`，否则 0。
- `grade`: `A >= 85`、`B >= 70`、`C >= 55`、否则 `D`。

该公式只是可解释的第一版事实评分，不是 SLA 承诺，也不自动改变路由。

## 边界

本轮不做：

1. 不自动调权、不自动禁用 supplier / channel。
2. 不把 grade 暴露成对外 SLA 承诺。
3. 不引入 AI agent prompt / workflow；本轮只准备 agent 可读的事实层。
4. 不新建资金、支付、赔付或发票状态。

## 影响

正向影响：

- `Supplier` 从业务身份推进到有周期性质量评级事实。
- 后续 agent 可以读取 scorecards 生成供应商淘汰、扩容或调权建议。
- 人审 dashboard 可以在批准供给策略前查看供应商质量证据。

代价：

- 第一版 score 是启发式公式，需要后续用真实运营数据校准。
- 没有 `UsageLedger` 的供应商不会生成 scorecard；没有 `SupplyCapacity` 时容量相关字段为 0。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `SupplierScorecard` 进入普通迁移和 fast migration。
2. `/api/supplier_scorecards/generate` 能基于 `gb10-4t` E2E 数据生成 scorecard。
3. `/api/supplier_scorecards` 能查询回读，并校验 requests、success rate、cache hit rate、gross profit、headroom、score、grade。
4. 真实进程模拟器输出 `supplier_scorecard_verified=true`。
5. `go test` 覆盖 model / controller / router / simulator / e2e，并记录 README 证据。
