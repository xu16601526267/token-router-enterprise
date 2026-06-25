# ADR 0023: Supply action plan lifecycle dashboard controls

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P5 人在回路、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T2 供给决策闭环、`SupplyActionPlan` 人工生命周期、dashboard 运营界面

## 背景

ADR 0022 已经为 `SupplyActionPlan` 增加 `planned` / `in_progress` / `completed` / `cancelled` 生命周期，以及 `POST /api/supply_action_plans/:id/status` 人工状态更新 API，并通过 `gb10-4t` 真实进程模拟器验证。

当前 `/token-router` 的 `Action Plans` tab 仍只查询 planned 工作项和执行 generate。operator 如果要推进状态，只能直接调用 API，dashboard 还不能承担 P5 中“人在 dashboard 上决策批准后才生效”的运营入口职责。

## 决策

在 `/token-router` 的 `Action Plans` tab 增加 lifecycle 控制：

1. 查询状态筛选从固定 `planned` 改为 All / Planned / In progress / Completed / Cancelled。
2. 表格展示本地化 status label，并显示 operator note、status updater、started / completed / cancelled timestamps。
3. 对非终态 action plan 提供人工状态更新入口。
4. 状态更新使用 Dialog + Textarea，由 operator 选择 `in_progress` / `completed` / `cancelled` 并填写 note。
5. 成功后刷新 action plan query；失败只在本 tab 的 toast 中反馈。

## 边界

本轮不做：

1. 不自动创建 supplier / channel。
2. 不自动修改 channel weight、supplier status 或 capacity。
3. 不创建采购单、打款单、发票或真实资金状态。
4. 不把 completed 解释为线下采购或自持算力已经真实发生；completed 只是 operator 的状态记录。
5. 不做负责人、截止时间、附件或外部任务系统集成。

## 影响

正向影响：

- operator 可以在 dashboard 内完成 action plan 的人工推进留痕。
- dashboard 能看到 planned 之外的历史完成项，减少运营工作项丢失。
- 后续自营采购 / 自持算力接入可以读取 completed action plan 作为人工确认记录。

代价：

- UI 增加一个小型状态更新弹窗和多语言文案。
- 状态机合法性仍以后端为准；前端只隐藏明显不可用操作。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `/token-router` 的 `Action Plans` tab 支持 action plan status filter。
2. 表格能展示 lifecycle fields。
3. UI 能调用 `/api/supply_action_plans/:id/status` 更新状态，并刷新 query。
4. UI 文案覆盖 en / zh / fr / ja / ru / vi。
5. 前端 typecheck / build / targeted lint 通过。
6. README 记录 dashboard lifecycle 能力与验证证据。
