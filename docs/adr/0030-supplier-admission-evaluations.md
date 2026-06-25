# ADR 0030: Supplier admission evaluations

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P1 严选不做集市、P2 无数据不承诺、P3 先度量再承诺 SLA、P4 供应商优胜劣汰、P5 经营即算法但人握方向盘、P8 守住边界
- 关联架构：L0 `UsageLedger` 事实层、L1 `SupplierScorecard` 质量评分、L2 人审决策层

## 背景

当前链路已经有：

1. `UsageLedger` 记录每次成功调用的 session、cache、成本、售价、延迟和供给节点。
2. `SupplierScorecard` 基于运行期 ledger/capacity 生成供应商评分与 grade。
3. `SupplyDecision` / action plan / execution / routing policy 形成供给扩容与自持路由的人审链路。

但产品原则 P1/P4 明确要求“严入选、有门槛、持续评级、优胜劣汰”。当前 scorecard 是运行期观察结果，还缺一个可查询、可审计的“是否达到入选/继续合作门槛”的结构化记录。

## 决策

新增 `SupplierEvaluation` 作为供应商入选/复评的被动事实层：

1. `SupplierEvaluation` 从已生成的 `SupplierScorecard` 生成，不绕过 scorecard 另算质量。
2. 每条 evaluation 绑定一个 scorecard 和周期，复制关键证据：
   - supplier
   - period
   - score / grade
   - success rate / latency / cache hit
   - gross profit
   - supply headroom / unit cost
3. 生成时给出 `recommendation`：
   - `admit`：score >= 85
   - `observe`：70 <= score < 85
   - `reject`：score < 70
4. evaluation 初始为 `draft`，只能由 admin review 为：
   - `approved`
   - `rejected`
5. approve/reject 只记录 `reviewed_at`、`reviewed_by`、`review_note`，不自动修改 supplier/channel/capacity/routing policy。
6. API：
   - `GET /api/supplier_evaluations`
   - `POST /api/supplier_evaluations/generate`
   - `POST /api/supplier_evaluations/:id/approve`
   - `POST /api/supplier_evaluations/:id/reject`

## 边界

本轮不做：

1. 不自动禁用低分 supplier。
2. 不自动创建 supplier/channel。
3. 不自动改变 channel weight 或 routing policy。
4. 不做真实采购、付款、打款、发票状态。
5. 不承诺外部 SLA，只记录是否具备入选/复评证据。
6. 不新增 agent runtime；本轮只落数据结构与 API，让未来 agent 能写入或触发生成。

## 影响

正向影响：

- P1/P4 的“严入选/优胜劣汰”从口号变成可审计事实。
- 供应商是否被接受进入高质量供给池，有 scorecard 证据和人审记录。
- 保持 P5：系统给建议，人做决定。
- 保持 P8：不触碰资金动作。

代价：

- 需要先有 `SupplierScorecard`，没有运行期数据的供应商暂时无法自动生成 evaluation。
- 阈值先固定在代码中，后续若要按模型/SLA/业务线细分，需要再增加策略表或配置。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `SupplierEvaluation` 迁移和基本生成逻辑可用。
2. 从 `SupplierScorecard` 能生成 `admit` draft evaluation，且重复 generate 幂等更新证据但不清空 review。
3. approve/reject 只改变 evaluation review 字段，不改变 supplier/channel/capacity/routing policy。
4. E2E 能通过 API 生成、查询、approve evaluation。
5. `token-router-sim run` 在真实进程链路中核验 supplier evaluation API。
6. `aima2` focused Go 测试、`go test ./...` 和真实进程 simulator 通过，README 记录证据。
