# ADR 0053: traffic forecast dashboard

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P5 agent 出洞察、人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0014 Traffic profile materialization；0052 Traffic forecast materialization；0015 Supply decision recommendations；0035 Operating insight dashboard

## 背景

ADR 0052 已经新增 `TrafficForecast` 物化表和 `/api/traffic_forecasts/generate` / query API，并通过 gb10-4t 真实进程链路证明 forecast 可以从 `TrafficProfile` 生成。

但 forecast 仍是 API-only：operator 在默认 `/token-router` admin console 里只能看到历史 `TrafficProfile`、下游 `SupplyDecision`、pricing recommendation 和 operating insight，看不到 forecast 的 source window、target window、confidence、predicted gap 或生成方法。这样 P9 的“流量即情报”在 dashboard 上仍断了一层，且 operator 很难区分历史画像、预测假设和人审决策。

## 决策

在默认 admin console `/token-router` 新增 `Forecasts` tab：

1. 调用 `GET /api/traffic_forecasts`，按全局 period 查询 forecast source window，并展示后端生成的 target window。
2. 提供 `Generate Traffic Forecasts` 操作，调用 `POST /api/traffic_forecasts/generate`，source window 使用全局 period；target window 由后端按 ADR 0052 默认推导。
3. 表格展示 slice、source period、target period、observed demand / peak、forecast demand / peak / headroom / gap、confidence、method、cache / SLA / gross profit / unit cost 和 reason。
4. 汇总展示 visible forecasts、forecast demand、open forecast gap、average confidence，方便 operator 快速判断下一周期是否有供给风险。
5. 前端类型/API/i18n 补齐 `TrafficForecast`。

该 tab 只展示和生成预测事实，不生成或审批 `SupplyDecision`，不自动创建 action plan，不激活 routing policy，不改价，不修改 supplier/channel/capacity，不触碰账单、结算或资金动作。

## 不做什么

1. 不在 dashboard 内引入 forecast 参数调优、LLM prompt、机器学习模型或后台定时任务。
2. 不把 forecast approve/reject 做成人审状态机；forecast 是可复算假设，不是决策记录。
3. 不改变 ADR 0052 的 moving-average 规则。
4. 不让 forecast 自动影响 `SupplyDecision` 权重；如果要接入决策生成，后续单独 ADR。

## 验收

1. `/token-router` tab strip 出现 `Forecasts`。
2. 前端类型/API 覆盖 `TrafficForecast` generate/query。
3. i18n 覆盖 en / zh / fr / ja / ru / vi。
4. TypeScript typecheck、i18n sync、targeted lint、production build 通过。
5. Playwright WebKit 以 API mocks 验证 tab、summary、table、generate POST 与 query params；布局在 desktop/mobile 不出现文字重叠。
