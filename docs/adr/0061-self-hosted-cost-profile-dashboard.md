# ADR 0061: self-hosted cost profile dashboard

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P5 人审；P6 成本透明；P8 守住边界；P9 流量即情报
- 关联 ADR：0055 Supply expansion opportunities；0056 Supply expansion opportunity dashboard；0057 Action plan opportunity evidence；0060 Self-hosted cost basis evidence

## 背景

ADR 0060 已经新增 `SupplyCostProfile` 与 `/api/supply_cost_profiles/record|GET`，并让 self-hosted `SupplyExpansionOpportunity` 在存在匹配 cost profile 时保存摊销单位成本、单位节省、总节省，并把 savings 加入 rank score。

剩余缺口是：operator 仍需直接调用 API 才能录入或核对成本基准，默认 `/token-router` dashboard 只能看到 opportunity 的 rank signals，不能直接看到 rank 中的 self-hosted savings evidence。这样 P5 的人审工作面缺少关键成本来源，不利于 operator 判断是否推进自持算力。

## 决策

在默认 `/token-router` dashboard 增加成本基准工作面：

1. 新增 `Cost Profiles` tab，查询 `/api/supply_cost_profiles/`，展示 supplier、node、model、period、capacity、fixed cost、variable unit cost、amortized unit cost、source、observed/recorded evidence。
2. 在该 tab 提供 `Record Cost Profile` dialog，调用 `/api/supply_cost_profiles/record` 记录 self-hosted supplier/node/model/period 的成本证据。
3. dialog 默认使用当前全局 period；operator 需要显式填写 supplier id、supply node、model、capacity、fixed cost、variable unit cost、source ref 和 observed time。
4. `Opportunities` tab 展示 `self_hosted_cost_profile_id`、self-hosted unit cost、unit savings、total savings，让 rank score 中的成本证据可见。
5. generate opportunity 后刷新 cost profile 与 opportunity query，保证 operator 能立即核对 cost profile -> opportunity rank 的证据链。

## 边界

1. 不新增后台成本采集器。
2. 不自动创建 supplier/channel/capacity/routing policy。
3. 不自动 approve/reject `SupplyDecision`，不生成 action plan。
4. 不修改价格、账单、结算、资金或库存状态。
5. 不把 cost profile 当作 SLA 或容量可用证明；SLA 与 capacity 仍由对应 evidence 层负责。

## 验收

1. 前端 typecheck 通过，新增 API/type 与后端字段一致。
2. i18n sync 后 en/zh/fr/ja/ru/vi 无 missing / untranslated。
3. Playwright route mock 验证 `Cost Profiles` tab 的 query、record POST payload、record 成功后刷新、以及 `Opportunities` tab 展示 self-hosted savings evidence。
4. README / architecture / traffic docs / product principles 更新，明确 dashboard 已能记录和展示 cost profile evidence，同时保留不自动执行边界。

## 实施记录

- 已在 `/token-router` 增加 `Cost Profiles` tab、`Record Cost Profile` dialog、`SupplyCostProfile` 前端 type/API，以及 `Opportunities` 的 cost evidence 列与 self-hosted savings 汇总。
- 已完成 en/zh/fr/ja/ru/vi 翻译并运行 `i18n:sync`，report 显示 missing / extras / untranslated 均为 0。
- 本机验证：`npx --yes bun@1.3.14 run typecheck`、targeted `oxlint`、targeted `oxfmt --check`、`npx --yes bun@1.3.14 run build` 均通过；全量 `lint` 仍被导入基线既有 lint debt 阻塞。
- Playwright route mock 使用 managed Chromium 验证 tab、opportunity evidence 和 record POST payload，截图见 `output/playwright/adr0061-cost-profile-dashboard.png`，脚本见 `output/playwright/adr0061-cost-profile-dashboard-check.js`。
