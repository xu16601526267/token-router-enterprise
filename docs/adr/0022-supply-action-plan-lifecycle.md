# ADR 0022: Supply action plan lifecycle

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P5 人在回路、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T2 供给决策闭环、供给三轨、`SupplyActionPlan` 人审后工作项

## 背景

ADR 0020 / 0021 已经把 approved `SupplyDecision` 变成 planned `SupplyActionPlan`，并在 `/token-router` 上展示。这解决了“下一步该做什么”的结构化交接，但 planned 工作项缺少后续状态留痕。

`traffic-and-supply.md` 中的闭环要求 approved decision 后进入“加供应商 / 下采购单（线下） / 调度自有算力”的生效阶段。平台软件仍不能自动执行这些动作，但应该能记录 operator 在线下推进后的状态，否则 action plan 只是一张待办清单，无法证明工作是否已经开始、完成或取消。

## 决策

为 `SupplyActionPlan` 增加人工生命周期：

1. 状态扩展为 `planned` / `in_progress` / `completed` / `cancelled`。
2. 新增 `POST /api/supply_action_plans/:id/status`，由 admin 提交目标状态和 operator note。
3. 记录 `status_updated_at`、`status_updated_by`、`operator_note`。
4. 首次进入 `in_progress` 记录 `started_at`；进入 `completed` 记录 `completed_at`；进入 `cancelled` 记录 `cancelled_at`。
5. `completed` / `cancelled` 视作终态，不允许再切到其他状态，只允许重复提交同一状态刷新 note。
6. `generate` 幂等刷新 action plan 核心事实时，不回滚已有生命周期状态。
7. API-only 真实进程验证必须与主进程一致，在启用 memory cache 时初始化 channel cache，否则 seed 的 `gb10-4t` channel 不会进入分发器。

## 边界

本轮不做：

1. 不自动创建 supplier / channel。
2. 不自动修改 channel weight、supplier status 或 capacity。
3. 不创建采购单、打款单、发票或真实资金状态。
4. 不接真实 agent 或外部任务系统。
5. 不做自动执行器；状态只代表 operator 对线下工作的记录。

## 影响

正向影响：

- action plan 从“待办清单”变成可审计的人工工作流。
- 后续 dashboard 可以区分 planned / in progress / completed / cancelled 工作项。
- 真实进程 E2E 能证明 approved decision 后不仅生成计划，还能记录线下推进结果。

代价：

- 状态是人工声明，平台不证明线下采购或部署已经真实发生。
- 第一版不做复杂状态机、负责人分配、截止时间或审批附件。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `SupplyActionPlan` 迁移包含 lifecycle 字段。
2. `/api/supply_action_plans/:id/status` 能把 planned plan 更新为 `in_progress` 和 `completed`，并记录 operator fields。
3. 终态不能切换到其他状态。
4. 重新调用 `/api/supply_action_plans/generate` 不会把 completed/cancelled plan 回滚成 planned。
5. `token-router-sim run` 能在真实进程链路中核验 lifecycle API。
6. `aima2` 上 focused Go 测试和 `go test ./...` 通过，README 记录证据。
