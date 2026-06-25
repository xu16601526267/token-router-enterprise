# ADR 0026: Self-hosted routing policies from execution records

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P5 人在回路、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T3 自持算力接入、会话亲和路由、`SupplyActionExecution`

## 背景

T0 到 T2 已经完成从 `UsageLedger` 到 `TrafficProfile`、`SupplyDecision`、`SupplyActionPlan` 的数据链路，并通过 ADR 0024/0025 让 completed action plan 的线下执行结果成为 `SupplyActionExecution` 事实和 dashboard 可见数据。

`traffic-and-supply.md` 的 T3 要求把自持算力注册成一个自持 supplier，并让亲和路由把目标切片的 session 固定到自有节点。这里必须守住两个边界：

1. `SupplyActionExecution` 只是 operator 登记的执行事实，不能被隐式当成路由策略。
2. 路由策略可以让平台软件影响流量，但它仍不代表采购、付款、部署证明或资金状态。

因此需要一个显式的、可审计的 policy 层，把 recorded execution 转成 active routing intent，再由现有 channel selection 读取。

## 决策

新增 `SupplyRoutingPolicy`：

1. `POST /api/supply_routing_policies/activate`：从 recorded self-hosted `SupplyActionExecution` 激活一条 routing policy。
2. `GET /api/supply_routing_policies`：按 execution、plan、decision、status、track、supplier、channel、capacity、周期查询 policy。
3. `POST /api/supply_routing_policies/:id/disable`：人工禁用 policy。
4. 每个 execution 最多一条 policy，以 `supply_action_execution_id` 幂等 upsert。
5. policy 复制 execution 的切片字段：model、SLA、user、period、track、action type、supplier/channel/capacity、effective period。
6. policy 只允许来自 `track=self_hosted`、`execution_status=recorded`、有效 `channel_id` 的 execution。
7. 激活时验证 supplier 是 `self_hosted`，channel 存在、启用、属于该 supplier，并且能服务目标 group/model。
8. channel selection 在正常随机/权重选择前查找 active policy；命中时选择 policy 指向的 channel，未命中或 policy channel 当前不可用时回到现有选择逻辑。

## 边界

本轮不做：

1. 不自动从 execution record 创建 policy；必须显式 activate。
2. 不创建或修改 supplier / channel / supply capacity。
3. 不修改 channel weight、priority、status 或 supplier status。
4. 不跳过既有 token group、model ability、channel enabled 校验。
5. 不做支付、采购、发票、库存或真实部署证明。
6. 不改变 cache-aware 计量公式；命中自持 channel 后仍走同一套 `UsageLedger` 和 supplier agreement 成本模型。

## 影响

正向影响：

- T3 有了最小可验证闭环：recorded self-hosted execution -> active routing policy -> selected self-hosted channel -> `UsageLedger` 归集到 self-hosted supplier。
- routing policy 是显式人审后的运营配置，不会让 execution fact 隐式改变生产流量。
- 继续复用现有 `Channel`、`Ability`、`SupplierAgreement`、`SupplyCapacity`、`UsageLedger` 和 session affinity。

代价：

- 第一版只支持按 model / SLA / user 切片匹配，不做复杂 DSL。
- policy channel 不可用时会 fallback 到现有渠道选择；这保证可用性，但需要后续 dashboard/告警暴露 policy miss。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `SupplyRoutingPolicy` 进入普通迁移和 fast migration。
2. 非 recorded / 非 self-hosted / 无 channel 的 execution 不能 activate。
3. recorded self-hosted execution 可以 activate policy，并且重复 activate 幂等更新。
4. channel selection 在匹配 model / SLA / user / period 时优先选择 policy channel。
5. demand simulator 真实进程链路能证明 policy 激活后新请求进入 self-hosted supplier/channel，`UsageLedger` 记录对应 supplier/channel/supply node，cache-aware 成本和亲和链路仍成立。
6. `aima2` 上 focused Go 测试和 `go test ./...` 通过，README 记录证据。
