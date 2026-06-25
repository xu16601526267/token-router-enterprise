# ADR 0025: Supply action execution dashboard

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P5 人在回路、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T2 到 T3 的供给闭环、`SupplyActionExecution`、admin console `/token-router`

## 背景

ADR 0024 已经新增 `SupplyActionExecution` 事实表和 admin API。completed `SupplyActionPlan` 可以被 operator 登记为 execution record，并关联 supplier / channel / supply capacity、实际容量、单位成本、外部凭据和记录人。

当前缺口是这些 execution records 只能通过 API 查询，dashboard 里的 T2 到 T3 链路仍停在 action plan lifecycle。按 P5，agent 和后台可以产出结构化事实，但 operator 需要在 dashboard 上看见事实链路后才能做下一步经营判断。按 P8，这个 dashboard 也不能把 execution record 展示成自动采购、支付或路由生效证明。

## 决策

在 `/token-router` admin console 增加 `Executions` tab：

1. 调用 `GET /api/supply_action_executions` 读取 execution records。
2. 复用全局周期过滤，并支持 status 与 track 过滤。
3. 汇总展示 visible executions、actual capacity、recommended capacity、unit cost。
4. 表格展示 action、track、slice、supplier/channel/capacity 引用、actual/recommended/gap、unit cost、effective period、external ref、recorded by/at、operator note、源 action plan completed by/at。
5. 表格只读，不在本轮提供创建或修改 execution record 的表单。

## 边界

本轮不做：

1. 不在前端创建 supplier / channel / supply capacity。
2. 不在前端自动登记 execution record；登记仍要求 operator 明确调用 record API 或后续专门表单填写完整线下凭据。
3. 不把 execution record 展示为付款、采购、部署或路由已生效证明。
4. 不修改亲和路由、channel weight、supplier status 或 capacity status。
5. 不进入 T3 自持切片定向路由；本轮只补齐 dashboard 可见性。

## 影响

正向影响：

- action plan 完成后的执行结果可以在 admin console 被直接审阅，不再只靠接口或数据库查询。
- T2 建议、人审、action plan、execution record 的事实链路在同一个运营界面闭合。
- 后续 T3 自持算力接入可以先从 dashboard 审计 execution records，再决定是否读取这些 facts 进入路由策略。

代价：

- 第一版仍是只读视图，operator 需要通过 API 或后续专门表单登记 execution。
- execution record 仍只是 operator 登记的结构化事实，不证明线下事实真实性。

## 验证

本 ADR 对应施工完成后，需要证明：

1. frontend type/API 覆盖 `SupplyActionExecution` 和 query filters。
2. `/token-router` 存在 `Executions` tab，并能按 period / status / track 查询 execution records。
3. 表格显示核心引用、容量、成本、外部凭据、recorded fields 和 completed action plan fields。
4. en / zh / fr / ja / ru / vi 翻译完整，i18n sync 无 missing / extras / untranslated。
5. `typecheck`、targeted `oxlint`、`build` 通过，README 记录实测证据。
