# ADR 0056: supply expansion opportunity dashboard

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P5 agent 出洞察、人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0055 Supply expansion opportunities；0054 Forecast-informed supply decisions；0035 Operating insight dashboard

## 背景

ADR 0055 已经新增 `SupplyExpansionOpportunity` read model 和 `/api/supply_expansion_opportunities/generate` / query API，用来把 supply decision 进一步物化为 third-party gap、self-operated bulk、self-hosted cache 等可排序机会。

当前缺口是 operator 仍只能通过 API 或 simulator 查看这些 opportunity。按 P5，agent-readable 分析必须能在 dashboard 上被人看到；按 P9，cache locality、headroom risk 和 rank score 应该成为供给扩张方向的可视化证据。

## 决策

在默认 `/token-router` dashboard 新增 `Opportunities` tab：

1. 调用 `GET /api/supply_expansion_opportunities`，按全局 period 查询 opportunity read model。
2. 提供 `Generate Opportunities` 操作，调用 `POST /api/supply_expansion_opportunities/generate`，source window 使用全局 period。
3. 支持 opportunity type 和 priority 筛选，默认展示全部机会类型与全部优先级。
4. 汇总展示 visible opportunities、action opportunities、recommended capacity、rank score。
5. 表格展示 slice、opportunity type、priority/cluster、track/decision、source evidence、demand/headroom/gap/recommended capacity、locality/stability/risk/rank、status 和 reason。
6. 前端类型/API/i18n 补齐 `SupplyExpansionOpportunity`。

该 tab 只展示和生成 opportunity read model，不 approve/reject `SupplyDecision`，不创建 action plan，不记录 execution，不激活 routing policy，不改价，不修改 supplier/channel/capacity，不触碰账单、结算或资金动作。

## 不做什么

1. 不在 dashboard 内引入 LLM prompt、自动聚类调参或后台定时任务。
2. 不新增 opportunity 人审状态机；人审仍发生在 `SupplyDecision`、`SupplyActionPlan` 和 `SupplyRoutingPolicy` 链路。
3. 不从 opportunity 直接创建 supplier/channel/capacity/action plan。
4. 不把 opportunity 当成 SLA 或生产路由证据。

## 验收

1. 前端类型/API 覆盖 `SupplyExpansionOpportunity` generate/query。
2. `/token-router` 可展示 opportunity summary、filters、source evidence 和 rank signals。
3. i18n sync 无 missing / extras / untranslated。
4. typecheck、targeted lint 和 route-mocked Playwright 验证通过。
