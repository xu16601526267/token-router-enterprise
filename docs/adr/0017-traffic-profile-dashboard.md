# ADR 0017: Traffic profile dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P5 人在 dashboard 决策、P9 流量即情报
- 关联架构：T1 流量画像、`TrafficProfile` L1 画像事实层、`SupplyDecision` T2 建议输入

## 背景

ADR 0014 已经新增 `TrafficProfile` 物化表与 admin API，并在 `gb10-4t` E2E 和真实进程模拟器里证明可以从 `UsageLedger` 与 `SupplyCapacity` 生成 model / SLA / user / period 切片画像。

ADR 0016 把 `SupplyDecision` 接入了 `/token-router` dashboard，但刻意没有展示完整 `TrafficProfile`。这留下一个可解释性缺口：operator 能审批建议，却不能在同一个 dashboard 里直接查看建议所依赖的需求、cache、SLA、毛利和供给余量事实。

## 决策

在默认前端 `/token-router` 新增 `Traffic Profiles` tab：

1. 新增 `TrafficProfile` 类型、查询 API、生成 API。
2. Traffic Profiles tab 按当前全局周期查询 `/api/traffic_profiles`。
3. 提供 `Generate Profiles` 按钮，调用 `/api/traffic_profiles/generate`，基于已有 `UsageLedger` 与 `SupplyCapacity` 物化当前周期画像。
4. 汇总展示可见 profiles、需求 token、cached token、供给余量、毛利。
5. 表格展示：slice、requests / sessions、demand / peak / peak ratio、cache、SLA / latency、gross profit、supply headroom、generated time。

## 边界

本轮不做：

1. 不修改 `TrafficProfile` 的聚合规则或数据库结构。
2. 不新增预测模型、定时任务或数据仓库。
3. 不从 Traffic Profiles tab 自动生成或自动审批 `SupplyDecision`。
4. 不把 profile 当成 SLA 承诺；它只是已物化的事实层输入。

## 影响

正向影响：

- T1 从“API 可用”推进到“operator 可见”。
- T2 建议的上游事实来源可在 dashboard 直接核验。
- 后续自营采购、自持算力 ROI 分析可以复用同一入口查看切片画像。

代价：

- 第一版只支持当前全局周期和列表视图；不做 profile drill-down。
- 没有 `UsageLedger` 或 `SupplyCapacity` 时，generate 可能返回空或供给余量为 0，需要 operator 结合上游数据理解。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `/token-router` 有 `Traffic Profiles` tab。
2. 前端能查询、生成 `TrafficProfile`。
3. UI 文案完成 en / zh / fr / ja / ru / vi 翻译。
4. 前端 typecheck / build / targeted lint 通过。
5. README 记录 dashboard 边界与验证证据。
