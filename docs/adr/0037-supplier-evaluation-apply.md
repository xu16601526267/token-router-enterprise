# ADR 0037: Supplier evaluation apply workflow

- 状态：Accepted
- 日期：2026-06-23
- 关联原则：P1 严选不做集市；P4 供应商优胜劣汰；P5 agent 出建议、人审执行；P8 平台只做数据与信任中枢
- 关联 ADR：0030 Supplier admission evaluations；0031 Supplier evaluation dashboard

## 背景

ADR 0030/0031 已经建立了 supplier scorecard -> admission evaluation -> dashboard review 的链路，并且明确 approve/reject 只记录人审结果，不自动修改 supplier/channel/capacity/routing policy。

这条边界是正确的，但还缺少下一步人工控制面：当 operator 已经批准一条 `admit` / `observe` / `reject` evaluation 后，仍需要离开 `/token-router` 手工编辑 supplier 状态与备注。这样会让“严选入选/复评证据”和实际 supplier posture 脱节，也不利于审计“是谁基于哪条 evaluation 改了供应商状态”。

## 决策

新增一个显式 apply 步骤：

1. 新增 `POST /api/supplier_evaluations/:id/apply`。
2. apply 只允许作用于 `status=approved` 的 evaluation。
3. apply 是一次性动作，成功后写回 evaluation：
   - `applied_at`
   - `applied_by`
   - `applied_note`
   - `supplier_status_before`
   - `supplier_status_after`
4. apply 同步更新 `Supplier.status` 与 `Supplier.notes`：
   - `admit` -> supplier enabled。
   - `observe` -> supplier status 保持不变，仅追加审计备注。
   - `reject` -> supplier disabled。
5. apply 由 dashboard 显式按钮触发；approve 不自动 apply。
6. 已 apply 的 evaluation 不再允许重新 approve/reject，避免出现“已落 supplier posture 但 review 状态被改写”的审计矛盾。
7. 成功后刷新 supplier evaluations 与 suppliers。

## 不做什么

1. 不在 approve/reject 时自动 apply。
2. 不创建或修改 channel、capacity、routing policy。
3. 不修改 channel weight、模型映射或真实流量路由。
4. 不触碰支付、打款、发票、结算状态。
5. 不把 `observe` 表达成新的 supplier status；当前 supplier 只有 enabled / disabled，观察状态先记录在 evaluation apply audit 与 supplier notes 中。
6. 不做批量 apply；先保证单条证据到单个 supplier posture 的审计闭环。

## 验收

1. approved evaluation 可调用 apply，并回写 supplier 状态与 apply audit 字段。
2. draft/rejected evaluation 不能 apply。
3. 已 apply evaluation 不能重复 apply。
4. dashboard 只对 approved 且未 applied 的 evaluation 展示 apply 操作，并显示已 applied 证据。
5. i18n 覆盖 en / zh / fr / ja / ru / vi。
6. Go focused tests、TypeScript typecheck、i18n sync、targeted lint、production build 通过。
7. Playwright WebKit 以 API mocks 验证 apply 按钮、POST payload、applied row 与 supplier 状态刷新。
