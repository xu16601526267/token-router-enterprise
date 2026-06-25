# ADR 0033: Pricing recommendation dashboard

- 状态：Accepted
- 日期：2026-06-22
- 关联原则：P3 先度量，再承诺 SLA；P5 agent 建议、人审；P6 平台利润来自效率；P8 平台只做数据与信任中枢
- 关联 ADR：0032 SLA pricing recommendations

## 背景

ADR 0032 已经把 `TrafficProfile` 转成可复算的 `PricingRecommendation`，并在真实进程链路里证明 generate / query / approve / regenerate-preserve-review 可用。后端现在有 SLA / 价格建议事实，但 operator 仍需要离开 `/token-router` dashboard 才能查看证据和审批。

现有 `/token-router` 已经覆盖流量画像、供给建议、供应商评分与评估。SLA/价格建议应放进同一个运营控制面，延续“agent/规则出建议，人审批准”的边界。

## 决策

在默认 admin console `/token-router` 新增 `Pricing` tab：

1. 调用 `GET /api/pricing_recommendations`，按全局 period 查询建议。
2. 支持 status filter：All / draft / approved / rejected。
3. 支持 action filter：All / raise_price / keep_price / share_savings。
4. 提供 `Generate Pricing Recommendations` 操作，调用 `POST /api/pricing_recommendations/generate`。
5. 对 draft 建议提供 approve / reject，调用对应 review API。
6. 表格展示切片、action、status/review、current unit price/cost/margin、recommended unit price/margin、demand/cache/SLA、supply evidence、reason。

该 tab 只展示和审批建议，不提供真实改价表单，不修改 `ModelPrice`、套餐、账单、结算或路由。

## 不做什么

1. 不接入真实价格配置变更。
2. 不展示外部合同或对外 SLA 文案。
3. 不把 approve/reject 绑定到自动通知、自动调价、自动账单。
4. 不新增后端字段或改变 ADR 0032 的规则。

## 验收

1. `/token-router` tab strip 出现 `Pricing`。
2. 前端类型/API 覆盖 `PricingRecommendation` generate/query/approve/reject。
3. i18n 覆盖 en / zh / fr / ja / ru / vi。
4. TypeScript typecheck、i18n sync、targeted lint、production build 通过。
5. Playwright WebKit 以 API mocks 验证 desktop/mobile 布局、filters、generate、approve/reject，不出现 tab/text overlap。
