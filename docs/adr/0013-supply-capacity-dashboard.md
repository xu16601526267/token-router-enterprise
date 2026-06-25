# ADR 0013: Supply capacity dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P4 供应商优胜劣汰、P5 人在 dashboard 决策、P9 流量即情报
- 关联架构：T1 画像 + dashboard、`SupplyCapacity` 周期快照、供给三轨

## 背景

ADR 0012 已经新增 `SupplyCapacity` 周期快照和 `/api/supply_capacities` admin API，并在 `gb10-4t` E2E 与真实进程模拟器中证明可以记录容量、已用量、余量、利用率、质量分和单位成本。

当前缺口是这些供给侧快照只能通过 API 或模拟器核验，operator 在 `/token-router` 仍看不到供给余量。产品原则要求“先记录、后分析、最后才放权决策”；因此下一步应先把供给容量作为只读 dashboard 数据展示出来，让人能看懂供给侧状态，而不是直接进入自动调权或供给决策。

## 决策

在 `/token-router` 新增 Supply Capacity 只读 tab：

1. 前端新增 `SupplyCapacity` 类型和 `getSupplyCapacities()` API client。
2. 复用页面已有 period filter，调用 `/api/supply_capacities` 的周期重叠查询。
3. 顶部展示总容量、已用量、余量、整体利用率四个统计卡片。
4. 明细表按 supplier / supply node / model / period 展示容量、已用量、余量、利用率、质量分和单位成本。
5. 文案明确 capacity 是人工 / 周期快照，不代表自动路由策略已经生效。

## 边界

本轮不做：

1. 不新增 capacity 编辑表单；capacity CRUD 已存在，dashboard 先只读。
2. 不自动从 `UsageLedger` 回填 `used_tokens`。
3. 不生成 `SupplyDecision` 建议，不做 agent 自动调权。
4. 不承诺 SLA，不把质量分直接映射成商业 SLA。
5. 不新增支付、库存采购、付款、发票或资金状态。

## 影响

正向影响：

- T1 dashboard 开始覆盖供给余量，而不只是需求消耗、毛利和质量。
- `gb10-4t` capacity snapshot 能被 operator 从同一个 admin console 核验。
- 后续 `TrafficProfile`、缺口分析、自营采购和自持算力 ROI 都有明确的 dashboard 承接点。

代价：

- 第一版只读，运营仍需通过 API/seed/后续表单写入 capacity 数据。
- 利用率按 snapshot 自身字段展示，不在前端做跨节点自动校正。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `/token-router` 能编译并展示 Supply Capacity tab。
2. 新增 i18n key 在 en、zh、fr、ja、ru、vi 中完整。
3. 前端 i18n sync、typecheck、build 通过。
4. 改动文件 targeted lint 通过。
5. README 记录 supply capacity dashboard 的只读边界和验证证据。
