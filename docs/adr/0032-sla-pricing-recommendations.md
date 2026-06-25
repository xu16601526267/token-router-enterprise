# ADR 0032: SLA pricing recommendations

- 状态：Accepted
- 日期：2026-06-22
- 关联原则：P3 先度量，再承诺 SLA；P5 agent 建议、人审；P6 平台利润来自效率，不是压榨任一方；P8 平台只做数据与信任中枢
- 关联架构：`TrafficProfile` L1 画像、`UsageLedger` 双价事实、T2 决策闭环

## 背景

现有链路已经能把 `UsageLedger` 与 `SupplyCapacity` 聚合成 `TrafficProfile`，并进一步生成供给三轨建议。它能回答某个 model / SLA / user 切片的需求量、cache 命中、达成率、毛利、单位成本与供给余量。

但产品原则还缺一个独立闭环：SLA 与价格不能凭感觉承诺，也不能由 agent 自动改价。平台需要把已度量的质量、成本、cache 复用与毛利，转成可解释的 SLA / 价格建议，并让 operator 在 dashboard/API 里审批。审批结果先作为经营决策事实保存，不直接改 `ModelPrice`、用户套餐、token 余额或供应商结算。

## 决策

新增 `PricingRecommendation` 物化表与 admin API：

1. `POST /api/pricing_recommendations/generate`：基于已物化 `TrafficProfile` 生成 draft 建议。
2. `GET /api/pricing_recommendations`：按周期、model、SLA、user、action、status 查询建议。
3. `POST /api/pricing_recommendations/:id/approve`：把 draft/rejected 建议标记为 approved，并记录 operator。
4. `POST /api/pricing_recommendations/:id/reject`：把 draft/approved 建议标记为 rejected，并记录 operator。

第一版 recommendation 是确定性规则，不接真实 agent：

- `raise_price`：毛利非正、SLA 未达标、或高 SLA 低供给余量时建议提价/降承诺，用于覆盖真实成本和风险。
- `share_savings`：高毛利、高 cache 命中、SLA 达成稳定时建议让利，用于把 cache 效率收益回馈需求侧。
- `keep_price`：其余切片维持现价并继续观察。

建议记录会复制生成时的关键证据：`TrafficProfile` id、slice、period、request/demand/peak/cache/SLA/latency/price/cost/margin/supply facts、建议的单位价格、建议毛利率、reason、review fields。`recommendation_key` 以 profile slice + period 唯一；重复生成只刷新事实与建议字段，不覆盖已审批状态与 review note。

## 不做什么

1. 不自动修改真实价格配置、套餐、账单、供应商结算或路由权重。
2. 不对外发布 SLA 承诺；`sla_met_rate` 仍是事实指标，不是合同。
3. 不引入 LLM/agent prompt；规则引擎先保证可复算、可测。
4. 不新增支付、税务、发票、银行账号等 P8 外的字段。

## 影响

- `TrafficProfile` 不只驱动供给侧，也开始驱动需求侧的 SLA / 价格经营建议。
- operator 可以在改价前看到毛利、成本、SLA、cache 和供给余量证据。
- 审批结果形成后续 dashboard、真实 agent、人工改价流程的输入，但当前仍没有副作用。

## 验收

1. `PricingRecommendation` 进入普通迁移和 fast migration。
2. `/api/pricing_recommendations/generate` 能基于 `gb10-4t` 的 `TrafficProfile` 生成 draft 建议。
3. `/api/pricing_recommendations` 能查询到该建议，并校验 action、unit price、margin、status 和 evidence。
4. `/api/pricing_recommendations/:id/approve` 能把建议改为 approved 并记录 operator。
5. 重复 generate 不覆盖已审批状态。
6. `token-router-sim run` 能在真实进程链路中核验 recommendation API。
