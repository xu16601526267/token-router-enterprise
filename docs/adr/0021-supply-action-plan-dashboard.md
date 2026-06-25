# ADR 0021: Supply action plan dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P5 人在回路、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T2 供给决策闭环、`SupplyActionPlan` 人审后工作项、dashboard 运营界面

## 背景

ADR 0020 已经新增 `SupplyActionPlan` 后端事实层与 admin API，并用 `gb10-4t` E2E 和真实进程模拟器证明 approved `SupplyDecision` 可以生成 planned action plan。

当前缺口是 action plan 只能通过 API 或模拟器查看。`traffic-and-supply.md` 的闭环要求人批准后进入“加供应商 / 下采购单（线下） / 调度自有算力”的生效阶段；在平台软件仍不执行这些动作的前提下，operator 至少需要在 `/token-router` dashboard 中看见这些 planned 工作项。

## 决策

在 `/token-router` 增加 `Action Plans` tab：

1. 新增 `SupplyActionPlan` 类型、查询 API、生成 API。
2. 复用全局周期筛选，调用 `/api/supply_action_plans` 查询当前周期 planned action plans。
3. 支持按 All / third-party / self-operated / self-hosted track 筛选。
4. 提供 `Generate Action Plans` 按钮，调用 `/api/supply_action_plans/generate`，从当前周期 approved decisions 生成 planned plans。
5. 展示 visible action plans、recommended capacity、open gap、ROI score 汇总。
6. 表格展示 action type、track、slice、recommended capacity、gap、ROI、source review、generated time。

## 边界

本轮不做：

1. 不执行 action plan。
2. 不自动创建 supplier / channel。
3. 不自动修改 channel weight、supplier status 或 capacity。
4. 不创建采购单、打款单、发票或真实资金状态。
5. 不新增 action plan 状态流转；第一版只展示 `planned`。

## 影响

正向影响：

- T2 从 approved decision 推进到 operator 可见的 planned 工作队列。
- 后续自营采购、自持算力接入可以直接读取 action plans，而不是重新解释 approved decisions。
- dashboard 上能区分“建议审批”和“下一步工作项”，降低把建议误读成自动执行的风险。

代价：

- 第一版仍需要 operator 在线下执行工作项。
- 没有 approved decision 时 generate 不会产生 action plan。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `/token-router` 有 `Action Plans` tab。
2. 前端能查询和 generate `SupplyActionPlan`。
3. UI 文案覆盖 en / zh / fr / ja / ru / vi。
4. 前端 typecheck / build / targeted lint 通过。
5. README 记录 dashboard 能力与验证证据。
