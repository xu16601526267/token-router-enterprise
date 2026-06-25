# ADR 0010: Supply configuration forms

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P5 经营即算法但人握方向盘、P8 守住边界
- 关联架构：M0 数据骨架、M0 前端表单、Channel.SupplierId 成本归集

## 背景

ADR 0009 已经把 token-router 的事实数据、毛利聚合和对账单暴露到 `/token-router`。但 M0 的架构要求不只是能看数据，还要能由管理员配置最小供给链路：

1. 建立上游 `Supplier`，标记第三方、自营或自持供给类型。
2. 为 supplier 配置 cache-aware 成本协议 `SupplierAgreement`。
3. 在 channel 上设置 `SupplierId`，让成功调用能归集到正确成本主体。

当前后端 CRUD 和 `Channel.SupplierId` 已经存在，缺的是 dashboard 上的人审配置入口。产品原则 P5 要求当前阶段仍是人在 dashboard 决策批准，agent 只能在后续给建议；P8 要求平台软件不碰银行、税号、打款或发票字段。

## 决策

本轮补齐最小人工配置闭环：

1. 在 `/token-router` 的 Suppliers tab 中新增 supplier 新建/编辑表单。
2. 在同一页面新增 supplier agreement 新建/编辑/删除入口，字段覆盖 supplier、model、effective period、ratio/price、priority、status 和备注。
3. 扩展 channel 新建/编辑表单，增加可选 `supplier_id` 数字字段，作为成本归集连接点。
4. 前端只调用已存在的 admin API，不新增资金相关后端模型。

## 边界

本轮不做：

1. 不采集银行账号、税号、付款方式、发票或真实收款信息。
2. 不做协议审批流或 agent 自动调价/自动调权。
3. 不做自营库存、预付采购、算力摊销核销 UI。
4. 不把 supplier 选择做成强约束；`supplier_id = 0` 仍表示该 channel 暂未纳入 token-router 成本归集。

## 影响

正向影响：

- M0 的“CRUD + 前端表单”闭环补齐，管理员不再需要直接调用 API 配置供给。
- `SupplierAgreement` 的 cache-aware 成本字段可在 dashboard 维护，后续 `UsageLedger` 成本计算可持续复核。
- channel 归集点进入现有 channel 编辑流，不需要新增单独的路由配置页面。

代价：

- supplier 选择先采用 ID 输入，避免把 token-router 查询耦合进通用 channel 表单；后续可以升级为复用 supplier 下拉。
- 协议表单仍是运营配置表单，不包含审批和历史版本比较。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `web/default` i18n 同步无 missing/extras/untranslated。
2. 前端 typecheck 和 build 通过。
3. 改动文件的 targeted lint 通过。
4. 后端 API 未改动时，至少保持既有 focused Go 测试不回退；若只改前端，记录未触碰后端逻辑。
