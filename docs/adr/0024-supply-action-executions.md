# ADR 0024: Supply action execution records

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P5 人在回路、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T2 到 T3 的供给决策闭环、供给三轨、completed `SupplyActionPlan`

## 背景

ADR 0020 到 0023 已经完成 `TrafficProfile` → `SupplyDecision` → `SupplyActionPlan` → dashboard lifecycle。approved decision 可以生成 action plan，operator 可以把 action plan 标记为 completed。

当前缺口是 completed action plan 只说明“线下工作被 operator 标记完成”，但没有结构化记录“完成后关联了哪个供应商 / channel / capacity 快照、实际承诺容量是多少、外部执行凭据是什么”。这会让 T2 到 T3 的接缝继续停留在备注文本里。

`traffic-and-supply.md` 的 T3 要求自持算力接入最终复用 Supplier / Channel / SupplyCapacity / 亲和路由 / cache-aware 成本模型。平台软件仍不能自动创建或修改这些供给对象，但应该允许 operator 把线下执行结果登记成可查询、可审计的数据事实。

## 决策

新增 `SupplyActionExecution` 事实表与 admin API：

1. `POST /api/supply_action_executions/record`：为 completed `SupplyActionPlan` 登记执行结果。
2. `GET /api/supply_action_executions`：按 action plan、decision、track、status、supplier、channel、capacity、周期查询执行结果。
3. 每个 action plan 最多一条 execution record，以 `supply_action_plan_id` 幂等 upsert。
4. execution record 复制 action plan 的核心事实：slice、model、SLA、user、period、decision type、track、action type、recommended capacity、gap、ROI、completed fields。
5. execution record 记录 operator 输入：`execution_status`、`supplier_id`、`channel_id`、`supply_capacity_id`、`actual_capacity_tokens`、`unit_cost_quota`、`effective_from`、`effective_to`、`external_ref`、`operator_note`、`recorded_by`、`recorded_at`。
6. `supplier_id` / `channel_id` / `supply_capacity_id` 只做可选引用校验；API 不创建、不修改这些对象。

## 边界

本轮不做：

1. 不自动创建 supplier / channel / supply capacity。
2. 不自动修改 channel weight、supplier status、capacity status 或 routing policy。
3. 不创建采购单、打款单、发票或真实资金状态。
4. 不把 execution record 当成真实支付、采购或部署证明；它只是 operator 登记的结构化结果。
5. 不实现自持流量定向；后续 T3 可以读取 execution record 和已有供给对象再接入路由策略。

## 影响

正向影响：

- completed action plan 有了可审计的执行结果事实，不再只靠备注文本。
- 后续自营采购、自持算力接入可以读取 execution records，而不用重新解释 completed action plans。
- 真实进程模拟器可以证明“建议 → 人审 → action plan → 完成 → 执行结果登记”的完整数据链路。

代价：

- 第一版 execution record 由 operator 手工登记，平台不证明线下事实真实性。
- 供给对象仍需 operator 或后续系统另行创建。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `SupplyActionExecution` 进入普通迁移和 fast migration。
2. 未 completed 的 action plan 不能登记 execution record。
3. completed action plan 可以登记 execution record，并校验 copied fields、operator fields、linked reference ids。
4. 重复 record 同一 action plan 幂等更新，不产生重复 execution record。
5. `token-router-sim run` 能在真实进程链路中核验 execution API。
6. `aima2` 上 focused Go 测试和 `go test ./...` 通过，README 记录证据。
