# ADR 0011: Quality summary dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P3 先度量再承诺 SLA、P4 供应商优胜劣汰、P9 流量即情报
- 关联架构：T0 遥测补齐、T1 画像 + dashboard、`UsageLedger` 事实层

## 背景

M0-M3 已经覆盖供应商配置、cache-aware 双价台账、会话亲和、毛利汇总与对账导出。`UsageLedger` 也已经记录了 `LatencyMs`、`Status`、`SlaTier`、`SupplyNode` 等质量遥测字段，但当前 dashboard 只展示毛利、台账和对账，不方便按 supplier/channel/model 观察质量。

产品原则要求：没有可验证的质量度量之前，不对外承诺 SLA；供应商评级和分流优化也必须先基于事实数据。因此下一步先补只读质量画像，让运营能看见延迟、成功请求、cache 命中、SLA 档和供给节点，不直接做自动调权或 SLA 承诺。

## 决策

新增只读质量汇总：

1. 后端新增 `/api/reports/quality_summary`，基于 `UsageLedger` 聚合。
2. 支持 `group_by=supplier|channel|model|sla_tier|supply_node|day` 与现有时间、supplier/channel/user/model/status 过滤。
3. 返回每组 `total_requests`、`success_requests`、`error_requests`、`success_rate`、`avg_latency_ms`、`max_latency_ms`、`cache_hit_rate`、token 与 quota 汇总。
4. 默认不创建新表；先用 GORM 聚合，后续量大再物化 `TrafficProfile`。
5. `/token-router` 新增 Quality tab，展示质量画像表，并沿用页面已有周期过滤。

## 边界

本轮不做：

1. 不对外承诺或售卖 SLA 等级。
2. 不自动调 channel 权重、优先级或 supplier 状态。
3. 不实现 error ledger 插桩；本轮只聚合现有 `UsageLedger.Status`，为未来错误记录兼容预留字段。
4. 不新建 `SupplyCapacity`、`TrafficProfile`、`SupplyDecision` 表；这些放到后续 T1/T2。

## 影响

正向影响：

- 质量统计进入 dashboard，P3 的“先度量”有可见入口。
- `Status`、`SlaTier`、`SupplyNode` 等已记录字段开始产生运营价值。
- 后续 supplier 评级、分流建议和供给缺口分析可以复用同一聚合接口。

代价：

- 当前成功率只反映已进入 `UsageLedger` 的记录；由于主链路目前只在成功 post-consume 后落账，错误率在补 error ledger 前不能代表完整请求失败率。
- p95 延迟暂不做跨数据库聚合，先提供平均和最大延迟，避免引入方言特化 SQL。

## 验证

本 ADR 对应施工完成后，需要证明：

1. 后端 E2E 能通过 `/api/reports/quality_summary` 读到 `gb10-4t` 两次请求的质量汇总。
2. 前端 i18n sync、typecheck、build 通过。
3. 改动文件 targeted lint 通过。
4. README 明确 quality summary 是只读观测，不等同于已承诺 SLA。
