# ADR 0008: Settlement statements and margin export

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P8 守住边界、P9 流量即情报
- 关联架构：M3 对账/毛利报表导出

## 背景

M1/M2 已经跑通：需求端请求进入 token-router，经 `gb10-4t` 供给侧 mock，成功落入 `UsageLedger`，并且同一 `SessionId` 会亲和到同一 channel。下一步要把事实台账变成财务和运营可以复核的对账数据。

架构文档要求：

1. 毛利报表基于 `UsageLedger` 聚合：收入、成本、毛利、cache 命中率。
2. 按周期×供应商、周期×客户生成 `SettlementStatement`。
3. 明细可 API/CSV 导出，交线下财务。

平台边界不变：不做支付、打款、发票或真实钱包状态机。

## 决策

新增 `SettlementStatement` 模型和 admin API：

1. `SettlementStatement` 只保存周期聚合结果，不代表付款执行。
2. subject 支持两类：
   - `supplier`：供应商维度，对应上游线下结算。
   - `user`：客户维度，对应下游线下对账。
3. 生成 statement 时按 `UsageLedger` 重新聚合，并以 `(subject_type, supplier_id, user_id, period_start, period_end)` 幂等 upsert。
4. 毛利 summary 直接从 `UsageLedger` group by 输出，不依赖预生成 statement。
5. CSV 明细从 statement 的 subject 和周期反查 `UsageLedger`，导出每笔 request 的卖价、成本、毛利、cache 信息。

## API

1. `GET /api/reports/margin_summary`
   - query：`group_by=supplier|channel|user|model|day`、`start_timestamp`、`end_timestamp`、`supplier_id`、`channel_id`、`user_id`、`model_name`
2. `POST /api/settlement_statements/generate`
   - body：`subject_type`、`supplier_id` 或 `user_id`、`period_start`、`period_end`
3. `GET /api/settlement_statements`
4. `GET /api/settlement_statements/:id`
5. `GET /api/settlement_statements/:id/items`
6. `GET /api/settlement_statements/:id/items.csv`

## 边界

本轮不做：

1. 不做 dashboard 页面。
2. 不做 statement 确认流、付款状态、发票状态。
3. 不做多租户组织维度；当前客户维度用 `UserId`。
4. 不做异步批处理；先用按需生成，方便 E2E 实测。

## 验证

扩展 E2E 和进程模拟器：

1. 跑真实请求生成两条 `UsageLedger`。
2. 调 `/api/reports/margin_summary?group_by=supplier`，断言请求数、收入、成本、毛利、cache 命中率正确。
3. 调 `/api/settlement_statements/generate` 生成 supplier 周期 statement。
4. 调 statement items API 和 CSV，断言两条明细可导出且金额与 ledger 一致。
