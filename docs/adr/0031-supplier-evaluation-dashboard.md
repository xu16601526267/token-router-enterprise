# ADR 0031: Supplier evaluation dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P1 严选不做集市、P2 无数据不承诺、P4 供应商优胜劣汰、P5 经营即算法但人握方向盘、P8 守住边界
- 关联架构：L1 `SupplierScorecard`、L2 `SupplierEvaluation`、Admin Console

## 背景

ADR 0030 已经把供应商入选/复评评估落成 `SupplierEvaluation` 和 admin API：

1. evaluation 从已生成的 `SupplierScorecard` 派生。
2. 系统给出 `admit` / `observe` / `reject` recommendation。
3. admin 可以 approve/reject，但 review 只更新 evaluation，不改变 supplier、channel、capacity、routing policy 或结算。

当前缺口是 Admin Console 仍只展示 scorecard 和供给决策，运营人员无法在仪表盘里看到入选评估、筛选 draft/recommendation、执行人审动作。这不满足 P5 “系统给洞察，人握方向盘”的交互闭环。

## 决策

在 `/token-router` Admin Console 中新增 `Evaluations` 标签页：

1. 周期选择沿用现有页面的 `Period Start` / `Period End`。
2. 列表读取 `GET /api/supplier_evaluations`，支持：
   - `status`
   - `recommendation`
   - `grade`
   - `start_timestamp` / `end_timestamp`
3. 提供 `Generate Evaluations` 按钮，调用 `POST /api/supplier_evaluations/generate`，从同周期 scorecard 生成/刷新 evaluation。
4. 表格展示供应商、recommendation、grade/score、运行证据、供给证据、review 状态与 reason。
5. 对 `draft` evaluation 展示 `Approve` / `Reject` 操作，调用对应 review API。
6. review note 使用 dashboard 固定短语，复杂备注暂不在本轮加入弹窗。

## 边界

本轮不做：

1. 不自动创建、启用、禁用或修改 supplier。
2. 不自动创建 channel。
3. 不自动修改 routing policy、channel weight 或执行供给切换。
4. 不做采购、付款、发票、结算状态变更。
5. 不新增 agent runtime；dashboard 只是人审入口。
6. 不新增图表或多维钻取，先保证可查询、可生成、可 review。

## 影响

正向影响：

- P1/P4 的入选评估可以被运营人员直接查看和审批。
- scorecard -> evaluation -> human review 的链路在 UI 上闭环。
- 保持 P8 边界，review 不触碰资金和自动路由动作。

代价：

- 页面多一个标签页，需要维护 i18n 和前端类型。
- review note 先固定，后续若需要审批说明，需要增加轻量弹窗。

## 验证

本 ADR 对应施工完成后，需要证明：

1. TypeScript 能识别 `SupplierEvaluation` 类型和 API。
2. `/token-router` 页面能通过构建和静态检查。
3. i18n 同步后各 locale 无缺失 key。
4. 页面能访问并渲染，不因新 tab 或 query 报错。
5. README 记录本轮前端验证证据。
