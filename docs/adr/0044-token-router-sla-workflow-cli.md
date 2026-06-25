# ADR 0044: token-router-sla workflow CLI

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P1 严选不做集市；P2 无数据不承诺；P3 先度量再承诺 SLA；P5 人审；P8 守住边界
- 关联 ADR：0038 SLA measurement automation；0040 SLA measurement evidence APIs；0042 token-router-sla probe runner；0043 SLA evidence gated supplier evaluations

## 背景

ADR 0042 已实现 `token-router-sla probe run`，可以读取已存在的 plan/contract，执行 OpenAI-compatible probe，写 artifact，并可回写 `SlaProbeRun`。

但实际准入链路仍需要 operator 分别用 dashboard/API 导入 contract、生成 plan，再回到 CLI 执行 probe。这对 aima2 或后续专属服务器的自动化不够直接，也容易让"合同 -> 计划 -> 实测 -> 证据"链路变成散落的手工步骤。

## 决策

在 `cmd/token-router-sla` 中补齐两个无副作用的工作流命令：

```text
token-router-sla contract import \
  --api http://127.0.0.1:19090 \
  --admin-token <root access token> \
  --input contracts/kimi-k25.json

token-router-sla plan generate \
  --api http://127.0.0.1:19090 \
  --admin-token <root access token> \
  --contract-key kimi-k25 \
  --supplier-id 1 \
  --channel-id 2 \
  --type admission \
  --route-mode through_token_router
```

行为：

1. `contract import` 从 JSON 文件读取 API 已定义的 `SlaContractImportInput` payload，调用 `POST /api/sla_contracts/import`，并输出返回的 contract JSON。
2. `plan generate` 从 flags 组装 `SlaProbePlanGenerateInput`，调用 `POST /api/sla_probe_plans/generate`，并输出返回的 plan JSON。
3. 继续复用 `probe run --record` 完成执行和证据回写；本 ADR 不新增单独的 `probe record`。
4. CLI 只编排既有 admin API，不直接连 DB，不绕过 dashboard/API 权限模型。

## 边界

1. 不自动创建 supplier、channel、capacity、agreement 或 routing policy。
2. 不自动生成、approve、apply `SupplierEvaluation`。
3. 不在 CLI 内做真实资金、结算或账单动作。
4. 不把 contract import 或 plan generate 当成 SLA passed evidence；只有 passed `SlaProbeRun` 才是 admission gate 证据。
5. 不把当前 JSON 文件格式变成第二套 schema；文件 payload 与既有 API input 保持一致。

## 验收

1. `contract import` 能从文件读取 payload，调用 `/api/sla_contracts/import`，输出 contract。
2. `plan generate` 能通过 `contract_id` 或 `contract_key`、supplier/channel/model/SLA flags 调用 `/api/sla_probe_plans/generate`，输出 plan。
3. focused CLI tests 覆盖 admin headers、payload 字段和 API envelope 解析。
4. `go test ./cmd/token-router-sla` 通过。
