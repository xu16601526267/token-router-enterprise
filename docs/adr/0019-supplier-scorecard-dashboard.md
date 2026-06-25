# ADR 0019: Supplier scorecard dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P1 严选不做集市、P3 先度量再承诺 SLA、P4 供应商优胜劣汰、P5 人在回路
- 关联架构：`SupplierScorecard` 事实层、dashboard 人审界面、后续 agent 建议输入

## 背景

ADR 0018 已经新增 `SupplierScorecard` 表和 admin API，并用 `gb10-4t` E2E 与真实进程模拟器证明可以从 `UsageLedger` 与 `SupplyCapacity` 生成供应商周期评分。

当前缺口是 operator 只能通过 API 或模拟器查看 scorecard。产品原则 P1 / P4 要求供应商持续评级，P5 要求当前阶段由人在 dashboard 上批准运营动作。因此需要把 scorecard 接入 `/token-router` dashboard，让人能直接看到评分构成，同时保留“不自动调权、不自动禁用、不承诺 SLA”的边界。

## 决策

在 `/token-router` 增加 `Scorecards` tab：

1. 新增 `SupplierScorecard` 类型、查询 API、生成 API。
2. 复用全局周期筛选，调用 `/api/supplier_scorecards` 查询当前周期 scorecards。
3. 提供 `Generate Scorecards` 按钮，调用 `/api/supplier_scorecards/generate` 物化当前周期评分。
4. 展示 visible scorecards、平均 score、A/B 供应商数量、供给余量汇总。
5. 表格展示 supplier、grade、score、requests、success/cache、latency、gross profit、capacity/headroom、quality/unit cost、generated time。

## 边界

本轮不做：

1. 不自动 approve / reject 供应商。
2. 不自动调 channel weight，不自动禁用 supplier / channel。
3. 不把 grade 暴露为对外 SLA 或赔付依据。
4. 不新增 scorecard 编辑表单；scorecard 由事实数据生成。
5. 不改变 ADR 0018 的评分公式。

## 影响

正向影响：

- operator 可以在 dashboard 上查看供应商周期评级和评分构成。
- 后续供给决策、供应商淘汰和 agent 建议有可解释 UI 入口。
- P1/P4 的“严选 + 持续评级”从 API 事实层推进到人工运营界面。

代价：

- 评分仍是第一版启发式公式，需要后续用真实数据校准。
- 没有 ledger 的供应商不会出现在 scorecard tab；这是事实层边界，不是供应商清单视图。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `/token-router` 能查询和生成 `SupplierScorecard`。
2. scorecard UI 文案覆盖 en / zh / fr / ja / ru / vi。
3. 前端 typecheck、i18n sync、targeted lint、build 通过。
4. README 记录 dashboard 能力与验证证据。
