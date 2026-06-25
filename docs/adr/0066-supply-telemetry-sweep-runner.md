# ADR 0066: supply telemetry sweep runner

## Status

Accepted

## Context

ADR 0064 / 0065 已经把供给容量遥测从人工 record 推进到 upstream collect 和按已有 capacity snapshot sweep。现在系统有了可调用的 admin API，但部署侧仍缺一个 repo-native runner：operator 需要用 `curl` 或临时脚本拼 admin header、period filter 和退出码，不利于迁移到专属服务器后的 cron / systemd timer。

这一步要把“可调度入口”补成“可部署的一次性命令”，但仍不把 API server 变成后台 worker，也不让系统自动调权、禁用 channel、激活 policy 或触碰资金动作。

## Decision

新增 `cmd/token-router-supply` CLI，第一版只实现：

```text
token-router-supply telemetry sweep [options]
```

能力边界：

1. CLI 通过 admin API 调用 `POST /api/supply_capacity_telemetries/sweep`。
2. 支持与 API 一致的过滤参数：`--supplier-id`、`--supply-node`、`--model`、`--period-start`、`--period-end`、`--channel-id`。
3. 输出 API 返回的 JSON result，保留 `attempted_count`、`collected_count`、`skipped_count`、`collected` 和 `skipped`，方便 cron 日志和后续审计。
4. `--fail-on-skip` 让任何 skipped capacity 返回非零错误；`--min-collected` 让“没有足够采集成功”返回非零错误。默认不因 skipped 失败，保持和 sweep API 一致。
5. 复用现有 admin token header 约定：`Authorization: Bearer <admin-token>` 与 `New-Api-User: 1`。

## Non-goals

1. 不新增常驻后台 goroutine、API server 内部 scheduler、队列、分布式锁或重试退避。
2. 不实现真实 fleet agent 注册、agent 心跳、远程命令下发或节点资产发现。
3. 不新增 supplier/channel/capacity，不扫描任意 URL，只调用已有 admin API。
4. 不自动生成 operating insight、不调权、不禁用 channel、不激活 routing policy、不改价、不触碰账单、结算或资金动作。

## Consequences

- aima2 和后续专属服务器可以直接用同一个 binary 做 one-shot telemetry sweep，并用系统级 cron / timer 管理周期。
- 运行证据由 API response、DB telemetry/capacity row 和 CLI 退出码共同构成，避免把 transport-only evidence 当成真实采集成功。
- 后续如果要实现真正 fleet agent，应另起 ADR，定义注册、身份、心跳、任务租约和审计模型。

## Validation

1. `cmd/token-router-supply` 单测覆盖 flag payload、admin headers、API envelope 解析、`--fail-on-skip` 和 `--min-collected` 退出语义。
2. 在 `aima2` 构建 `token-router-api`、`token-router-sim`、`token-router-supply`。
3. 真实进程启动 gb10-4t mock supply、seed SQLite、启动 API-only server 后，先运行需求 simulator 生成 capacity snapshot，再用 `token-router-supply telemetry sweep` 调用 admin API。
4. DB 回读证明 `SupplyCapacityTelemetry` 与 `SupplyCapacity.last_telemetry_id` 使用 `source_ref=gb10-4t-mock-capacity` 的真实 upstream telemetry evidence。
