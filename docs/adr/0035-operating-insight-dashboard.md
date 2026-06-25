# ADR 0035: Operating insight dashboard

- 状态：Accepted
- 日期：2026-06-22
- 关联原则：P5 agent 出洞察、人审；P7 理念是假设、需数据复盘；P8 平台只做数据与信任中枢；P9 流量即情报
- 关联 ADR：0034 Operating insights；0033 Pricing recommendation dashboard；0016 Supply decision dashboard

## 背景

ADR 0034 已经把 `TrafficProfile`、`SupplyDecision` 与 `PricingRecommendation` 合成为可查询、可复盘的 `OperatingInsight`。后端和真实进程链路已经证明 generate / query / acknowledge / regenerate-preserve-review 可用，但 operator 仍缺一个默认 admin console 入口来查看洞察证据、筛选 severity/category/status，并记录 acknowledge / dismiss。

经营洞察比单一供给或定价建议更接近“agent hypothesis”：它解释某个流量切片为什么值得行动、观察或维持稳态。因此 dashboard 必须延续“agent/规则出洞察，人审留痕”的边界，只做证据面和 review 面。

## 决策

在默认 admin console `/token-router` 新增 `Operating Insights` tab：

1. 调用 `GET /api/operating_insights`，按全局 period 查询洞察。
2. 支持 status filter：All / draft / acknowledged / dismissed。
3. 支持 severity filter：All / action / watch / info。
4. 支持 category filter：All / cache_efficiency / capacity_risk / pricing_risk / quality_watch / steady_state。
5. 提供 `Generate Operating Insights` 操作，调用 `POST /api/operating_insights/generate`。
6. 对 draft 洞察提供 acknowledge / dismiss，调用对应 review API。
7. 表格展示切片、category/severity、review 状态、linked supply decision、linked pricing recommendation、traffic evidence、unit economics 与 recommended action。

该 tab 只展示和 review 经营假设，不修改 `SupplyDecision`、`PricingRecommendation`、supplier、capacity、routing policy、账单、结算或支付状态。

## 不做什么

1. 不引入真实 LLM prompt、agent workflow 或后台定时生成任务。
2. 不在 acknowledge / dismiss 后自动 approve/reject 下游建议。
3. 不提供调价、采购、扩容、调权、结算、付款或通知动作。
4. 不新增后端字段，不改变 ADR 0034 的规则和 review 保留语义。

## 验收

1. `/token-router` tab strip 出现 `Operating Insights`。
2. 前端类型/API 覆盖 `OperatingInsight` generate/query/acknowledge/dismiss。
3. i18n 覆盖 en / zh / fr / ja / ru / vi。
4. TypeScript typecheck、i18n sync、targeted lint、production build 通过。
5. Playwright WebKit 以 API mocks 验证 desktop/mobile 布局、filters、generate、acknowledge/dismiss，且 status/severity/category query params 正确。
