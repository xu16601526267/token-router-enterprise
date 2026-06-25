# ADR 0009: Token Router admin console

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P5 经营即算法但人握方向盘、P8 守住边界
- 关联架构：M0 前端表单、M3 对账/毛利报表导出

## 背景

M0 到 M3 的后端闭环已经具备供应商、协议价、双价台账、会话亲和、毛利聚合、对账单生成和 CSV 明细导出。当前缺口是 operator 只能通过 API 或模拟器核验数据，无法在 dashboard 上查看这一条链路。

产品原则要求：

1. 数据可信先于上层经营动作。
2. AI/agent 可以给建议，但当前必须人在 dashboard 上决策批准。
3. 平台软件只做数据与信任中枢，不做支付、钱包、打款或发票状态机。

因此本轮先补一个管理台入口，把已实测的事实数据和对账数据暴露给管理员。

## 决策

在默认前端新增 authenticated admin route：`/token-router`。

页面采用现有 `SectionPageLayout`、shadcn/base-nova 组件和 React Query API 封装，包含四块：

1. Overview：展示毛利、收入、成本、请求数、cache 命中率等汇总卡片。
2. Suppliers：展示供应商和协议价，用于确认 channel 的成本归集对象和 cache-aware 协议价。
3. Usage Ledger：展示每笔请求的 request/session/channel/supplier、token 拆分、卖价、成本、毛利和 cache 命中。
4. Settlements：生成 supplier/user 周期对账单，展示历史 statement，并提供明细 CSV 下载。

导航把入口放在 Admin 分组下，命名为 `Token Router`，与渠道、模型、用户等运营入口同级。

## 边界

本轮不做：

1. 不做支付、打款、发票、真实钱包或确认后付款状态机。
2. 不做 agent 自动调价、自动调权或自动批准。
3. 不做自营采购/自持算力库存核销 UI。
4. 不做复杂审批流；statement 仍是 draft 数据供线下财务复核。

Supplier 和 agreement 先以列表核验为主，后续可在同一路由补齐编辑抽屉和审批建议。

## 验证

1. 前端类型检查和构建需通过。
2. i18n 同步需确认新增 `t()` key 在全部 locale 中存在。
3. 保持后端 E2E 不回退：`UsageLedger`、margin summary、statement items、CSV 导出仍通过现有模拟器和测试核验。
