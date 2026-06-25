# ADR 0002: M0 供应商与双价台账数据骨架

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P3 先度量再承诺 SLA、P8 守住边界、P9 流量即情报
- 关联架构：M0 数据骨架、T0 遥测补齐

## 背景

上游 new-api 基线已经导入。下一步要让 token-router 拥有独立于 new-api `Vendor` 的结算事实层。

上游 `Vendor` 是模型展示、定价页和模型元数据的供应商概念；token-router 的 `Supplier` 是线下结算对手与成本归集对象。两者同名近义但边界不同，混用会让模型展示元数据、上游渠道和线下财务对账耦合在一起。

## 决策

新增独立后端数据骨架：

1. `Supplier`：上游结算对手，包含 `Type`（第三方 / 自营 / 自持）、状态、备注和时间戳，不包含银行、税号、收款账号等资金字段。
2. `Channel.SupplierId`：渠道归属的成本归集点。
3. `SupplierAgreement`：供应商协议成本，按 supplier + model + 生效时间 + priority 匹配，先支持 ratio/price 两种成本表达，为 M1 cache-aware 成本计算预留字段。
4. `UsageLedger`：每次成功调用一行事实台账，包含 request/session/channel/supplier/user/token/model、cache 拆分、双价 quota、T0 遥测字段和 request id 幂等唯一键。
5. Admin API：先提供供应商、协议、台账只读/管理接口，供后续模拟器和 E2E 验证直接调用。

本轮不做：

1. 不接支付、钱包、打款、发票或真实充值。
2. 不把 `Supplier` 复用到模型 `Vendor`。
3. 不做 dashboard 前端页面；前端会在后端链路可实测后补。
4. 不做 M1 计量插桩、M2 亲和路由或 M3 对账汇总；它们分别另写 ADR。

## 影响

正向影响：

- M1 可以直接根据 `Channel.SupplierId` 和 `SupplierAgreement` 写入 `UsageLedger`。
- 需求端模拟器可以在 E2E 中验证 request id 幂等、session 记录、cache 拆分和双价数据。
- `Supplier.Type` 先落库，后续 `gb10-4t`、自营采购、自持算力都能复用同一成本事实层。

代价：

- 后端会先有 API 而没有前端页面。
- 当前无法在本机做 Go 编译验证；需要等 Go/Docker 环境或 `aima2` SSH 恢复后补实测。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `Supplier`、`SupplierAgreement`、`UsageLedger` 和 `Channel.SupplierId` 已进入迁移。
2. Admin API 路由已注册。
3. 可用工具链下运行 `go test ./...`，或记录当前环境缺失导致无法运行的具体原因。
4. 后续 E2E ADR 必须使用这些 API 创建 `gb10-4t` 供给、需求模拟器请求和台账核验。

