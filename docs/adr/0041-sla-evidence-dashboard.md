# ADR 0041: SLA evidence dashboard

- 状态：Accepted
- 日期：2026-06-23
- 关联原则：P1 严选不做集市；P2 无数据不承诺；P3 先度量再承诺 SLA；P5 人审；P8 平台只做数据与信任中枢
- 关联 ADR：0040 SLA measurement evidence APIs；0038 SLA measurement automation for supplier admission；0031 Supplier evaluation dashboard

## 背景

ADR 0040 已经补齐 SLA contract / probe plan / probe run 的数据表与 admin API。现在 operator 可以通过 API 导入合同、生成计划、写回 runner 证据，但默认 `/token-router` admin console 仍看不到这些证据。

SLA 测量证据是 supplier admission 的上游事实，不等于运行期 scorecard，也不等于 supplier posture 变更。dashboard 需要让人能录入和复核证据，同时清楚保持“证据入口”边界。

## 决策

在默认 admin console `/token-router` 新增 `SLA Evidence` tab：

1. 调用 `GET /api/sla_contracts`，按全局 period 查询 imported contract，并支持 All / draft / active / retired filter。
2. 提供 `Import SLA Contract` 表单，调用 `POST /api/sla_contracts/import`，录入 contract key、model、provider、source、version、effective period、measurement profile JSON、hard gate JSON、soft gate JSON。
3. 调用 `GET /api/sla_probe_plans`，按全局 period 查询 generated plan，并支持 probe type 和 route mode filter。
4. 提供 `Generate Probe Plan` 表单，调用 `POST /api/sla_probe_plans/generate`，从 contract / supplier / channel / model / SLA tier / probe type / route mode 生成探针计划。
5. 调用 `GET /api/sla_probe_runs`，按全局 period 查询 runner 写回记录，并支持 run status filter。
6. 提供 `Record Probe Run` 表单，调用 `POST /api/sla_probe_runs/record`，写回 run key、plan、status、start/end time、runner/runtime refs、summary JSON、hard gate、soft warnings、failure reasons、artifact URI/hash。
7. 表格展示 contract、plan、run 三层证据链，包含 profile snapshot、gate summary、artifact hash、record/import/generate 人和时间。

该 tab 只做 evidence import / plan generation / run recording / query，不把测量结果自动推进 supplier admission、routing、billing、settlement 或 payment。

## 不做什么

1. 不实现 `token-router-sla` runner，也不从浏览器发起长时间 benchmark。
2. 不把 passed/failed run 自动映射到 `SupplierEvaluation` admit/reject。
3. 不自动创建 supplier、channel、capacity snapshot 或 routing policy。
4. 不自动 apply supplier evaluation，不修改 `Supplier.status` 或 `Channel.status`。
5. 不在 UI 中承诺某模型或供应商已经通过官方 SLA，只展示已记录证据。
6. 不触碰账单、结算、付款、发票状态。

## 验收

1. `/token-router` tab strip 出现 `SLA Evidence`。
2. 前端类型/API 覆盖 contract import/query、plan generate/query、run record/query。
3. i18n 覆盖 en / zh / fr / ja / ru / vi。
4. TypeScript typecheck、i18n sync、targeted lint、production build 通过。
5. Playwright WebKit 以 API mocks 验证 tab、三张表、filters、import/generate/record POST payload。
