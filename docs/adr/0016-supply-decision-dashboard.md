# ADR 0016: Supply decision dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P5 人在 dashboard 决策、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T2 供给决策建议、`SupplyDecision` 人审记录、`TrafficProfile` L1 画像

## 背景

ADR 0015 已经新增 `SupplyDecision` 后端表与 admin API，并在 `gb10-4t` E2E 和真实进程模拟器里证明可以从 `TrafficProfile` 生成供给三轨建议，再把 draft 建议审批为 approved。

当前缺口是 operator 仍只能通过 API 或模拟器查看和审批建议。产品原则 P5 要求当前阶段是“agent 出建议，人在 dashboard 上批准后才生效”。因此需要把 `SupplyDecision` 接入 `/token-router` dashboard，让人能看见建议来源和风险边界，并完成 approve / reject 留痕。

## 决策

在默认前端 `/token-router` 新增 `Decisions` tab：

1. 新增 `SupplyDecision` 类型、查询 API、生成 API、approve API、reject API。
2. Decisions tab 按当前全局周期查询 `/api/supply_decisions`。
3. 支持按 `draft` / `approved` / `rejected` 过滤。
4. 提供 `Generate Decisions` 按钮，调用 `/api/supply_decisions/generate`，基于已物化 `TrafficProfile` 生成建议。
5. 表格展示：slice、track、status、decision type、demand、headroom、gap、recommended capacity、ROI、review 信息。
6. draft 行展示 `Approve` / `Reject` 操作；点击后只调用人审 API，并刷新 decisions 数据。

## 边界

本轮不做：

1. 不做自动调权、自动采购、自动注册自持算力。
2. 不在 dashboard 修改 `SupplyDecision` 的生成规则或 ROI 公式。
3. 不新增自定义审批流状态机；仅使用后端已有 draft / approved / rejected。
4. 不展示 TrafficProfile 的完整画像表；本轮只把 T2 建议接入 dashboard。

## 影响

正向影响：

- T2 从“API 可用”推进到“operator 可见、可审批”。
- 人审边界在 UI 上可见，避免把建议误读成自动执行策略。
- 后续自营采购、自持算力接入可以读取 approved decisions，而不用重新设计审批入口。

代价：

- 第一版审批按钮不采集复杂备注；只做最小留痕。
- Decisions 依赖已有 `TrafficProfile`，没有 profile 时 generate 不会产生建议。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `/token-router` 有 `Decisions` tab。
2. 前端能查询、生成、approve、reject `SupplyDecision`。
3. UI 文案完成 en / zh / fr / ja / ru / vi 翻译。
4. 前端 typecheck / build / targeted lint 通过。
5. README 记录 dashboard 边界与验证证据。
