# ADR 0038: SLA measurement automation for supplier admission

- 日期：2026-06-23
- 状态：Proposed
- 关联原则：P1 严选不做集市、P2 无数据不承诺、P3 先度量再承诺 SLA、P4 供应商优胜劣汰、P5 人审、P8 守住边界
- 关联架构：`UsageLedger` 事实层、`SupplierScorecard` 运营评分、`SupplierEvaluation` 人审准入、`OperatingInsight` 运行期复盘

## 背景

当前 `SupplierScorecard` 和 `SupplierEvaluation` 已经能基于真实流量生成供应商评分和人审准入记录，但它们仍然是运营观测，不是官方 SLA 准入测量。

Kimi 和 GLM 的官方/合同口径都说明，供应商是否能准入不能只看平均成功率或均值延迟，必须按模型合同拆成可重复测量的维度：

1. 输入 token 分布或档位。
2. 输出 token 分布或固定输出约束。
3. TTFT 分位数。
4. TPOT / OTPS / inter-packet latency。
5. cache 口径，尤其 cold no-cache 与 prefix-cache 命中分开统计。
6. 错误率、429、超时和 10 分钟窗口可用性。
7. 正确性、模型版本、默认参数和 trace / request id。
8. 原始证据链，包括 prompt hash、usage、运行命令、endpoint、runtime 版本和采集工具版本。

因此需要在 token-router 里设计一套可导入模型 SLA 合同、生成探针计划、精确执行测量、写回证据、再由人审使用的自动化工具链。

## 参考合同维度

本 ADR 只借鉴维度并设计 token-router 的工具契约。具体阈值必须以导入到 token-router 的 `SlaContract` 为准，不能依赖外部 repo 路径在运行时存在。

### Kimi K2.5

参考来源：

- `a800-kimi/doc/reference/kimi-k26/Kimi-K2.5-serving-requirements-2026.txt`
- `a800-kimi/doc/reference/kimi-k26/kimi-k25-acceptance-spec.json`，作为官方文档的结构化导入镜像

关键维度：

| 维度 | 合同口径 |
|---|---|
| 输入 | TTFT 按增量 input tokens 分档统计，不含 cache；官方档位为 <4K、<8K、<32K、<64K、<128K、<256K |
| TTFT | 每档用通过比例判定，例如 <32K 档要求统计周期内 TTFT < 4s 的输入条数 >= 50%，TTFT < 8s 的输入条数 >= 90% |
| decode | OTPS 分 tier 报价和验收，tier1 > 40、tier2 > 10；统计周期内不满足 OTPS 的请求数占比超过 10% 则该周期不可用 |
| cache | cache hit settlement 以 LRU 模拟 prefix cache 的理论值为基准；默认 block size 16 tokens，按请求时间顺序 replay，命中率不到理论值 90% 时按理论值 90% 结算 |
| 429 / 负载阈值 | RPM 超过约定值或 7 天每分钟峰值平均值时应即时返回 429，不能排队等待；合同内请求率返回非 2xx 计失败 |
| 可用性 | 月可用性 >= 99.5%；任一连续 10 分钟窗口失败率 >= 1%，或 accuracy / TTFT / OTPS 任一合同项不满足，则该窗口不可用 |
| RTO | 北京时间 08:00-24:00 高峰期 RTO <= 10 分钟；00:00-08:00 低峰期 RTO <= 60 分钟 |
| 准确性 | 服务期须保持与上线前表现一致；未授权重大更新且任一 KVV 测试集结果差异 > 2% 时，对应时间段视为不可用 |
| 上线前协议 | thinking 开关采用 Anthropic style；默认参数必须与官方一致；默认不得添加 system prompt；tool call 前 interleaved thinking 必须回传，否则应返回 400 |

### GLM-5 / GLM-5.1

参考来源：

- `a800-kimi/doc/active/_cross/kimi-k26-glm51-a800-h200-capacity-estimate-2026-05-20.md`
- 该文档记录的官方输入为 `GLM-5&GLM-5.1 技术指标（20260409）.pdf`

关键维度：

| 维度 | 合同口径 |
|---|---|
| 输入分布 | input tokens: p50 50K、p90 70K、p99 120K |
| 输出分布 | output tokens: p50 0.2K、p90 0.8K、p99 2K |
| 高峰窗口 | 10:00-12:00、14:00-17:30 |
| TTFT | p50 2-4s、p75 5-10s、p90 10-15s、p99 20-30s |
| decode | TPOT/decode: p50 50 tok/s、p99 25 tok/s |
| streaming | inter-packet latency < 500ms；`stream_options.include_usage` 需要实时返回 usage |
| cache | 支持 cache；同一用户 cache hit rate 与 Zhipu MaaS 约 2% 内一致 |
| 可用性 | SLA 99.9%；请求超时 120 分钟；OpenAI chat-compatible API |
| 追踪 | request id 可记录为 traceId |

## 决策

新增一套 SLA 自动化测量设计，分为四层。

### 1. 合同层：`SlaContract`

`SlaContract` 是模型 SLA 的版本化数据源，而不是写死在评分公式里的常量。

建议字段：

- `contract_key`：例如 `kimi-k25-official-v2026`、`glm51-official-v20260409`。
- `model_name`、`model_aliases`、`provider_family`。
- `source_name`、`source_ref`、`source_sha256`、`version`。
- `status`：`draft` / `active` / `retired`。
- `effective_from` / `effective_to`。
- `measurement_profile_json`：输入/输出分布、bucket、percentile、cache、streaming、error、availability、default params。
- `hard_gate_json`：必须通过的硬门槛。
- `soft_gate_json`：进入 observe 或 warning 的软门槛。

合同 JSON 必须表达两种阈值语义：

1. `pass_fraction`：例如“至少 90% 请求 TTFT < 8s”。
2. `quantile_value`：例如“插值 p90 <= 8s”。

工具默认同时输出两种结果，但准入判定以合同声明的 `percentile_mode` 为准，避免 p50/p90 的计算方式不一致。

### 2. 计划层：`SlaProbePlan`

`SlaProbePlan` 从 `SlaContract` 和 supplier/channel/model 生成。

建议字段：

- `plan_key`、`contract_id`、`supplier_id`、`channel_id`、`model_name`、`sla_tier`。
- `probe_type`：`admission` / `runtime_light` / `runtime_deep` / `incident_recheck`。
- `route_mode`：`direct_upstream` / `through_token_router`。
- `prompt_suite_key`、`tokenizer_ref`、`sample_size`、`repeat_count`。
- `input_profile_json`、`output_profile_json`、`concurrency_profile_json`、`rate_profile_json`。
- `cache_profile`：`cold_no_cache` / `warm_same_session` / `mixed_trace_replay`。
- `schedule_interval_seconds`、`jitter_seconds`、`max_probe_quota`。

`direct_upstream` 用于测供应商裸能力，`through_token_router` 用于测实际路由、session 透传、cache-aware 计量和 usage ledger 侧证据。两者不能互相替代。

### 3. 执行层：`token-router-sla` runner

建议新增独立 CLI / worker，而不是把长时间 benchmark 放进 API handler：

```text
token-router-sla contract validate --contract contracts/kimi-k25-official.json
token-router-sla plan generate --contract kimi-k25-official --supplier 12 --channel 34 --type admission
token-router-sla probe run --plan <plan_id> --out output/sla/<run_id>
token-router-sla probe summarize --run-dir output/sla/<run_id>
token-router-sla probe record --run-dir output/sla/<run_id> --api http://127.0.0.1:3000
```

runner 必须做精确采集：

1. 使用固定 prompt suite 或 trace replay，记录 prompt hash 和目标 token 数。
2. 发送前用合同指定 tokenizer 或已验收的 token counter 预估 token 数。
3. 响应后读取 provider usage，记录 `prompt_tokens`、`cached_tokens`、`completion_tokens`；实际 token 数与目标不符时标记为 `invalid_sample`。
4. streaming 请求记录 request start、first byte、first SSE event、first token、last token、final usage、response close。
5. 非 streaming 请求不能声称真实 first-token TTFT，只能记录 full response latency 或标记 `ttft_observed=false`。
6. cold no-cache 行必须使用唯一 session/cache key，并要求 `cached_tokens=0`。
7. warm cache 行必须使用同一 user/session/cache key，并把 cache hit 与 TTFT 改善独立上报。
8. warmup、JIT、diagnostic/profiler 行不得混入 claim-grade 样本。
9. 429 在超过合同 RPM 的主动限流场景可记为 `admission_limited`；在合同内请求率出现则记为 SLA failure。
10. 所有失败必须分类：connect、timeout、http_5xx、http_4xx、upstream_error、invalid_json、missing_usage、wrong_model、wrong_output_shape、content_check_fail。

### 4. 证据层：`SlaProbeRun` / `SlaProbeSample`

`SlaProbeRun` 是一次测量的汇总证据。

建议字段：

- `run_key`、`plan_id`、`contract_id`、`supplier_id`、`channel_id`。
- `status`：`running` / `passed` / `failed` / `invalid` / `cancelled`。
- `started_at`、`ended_at`、`runner_version`、`git_commit`、`runtime_ref`。
- `endpoint`、`route_mode`、`model_name`、`sla_tier`。
- `summary_json`：各 bucket / profile 的 p50/p75/p90/p99、success rate、failure taxonomy、cache hit、TPOT/OTPS、inter-packet。
- `hard_gate_passed`、`soft_gate_warnings`、`failure_reasons`。
- `artifact_uri`、`artifact_sha256`。

`SlaProbeSample` 是可选明细表；量大时可以只存 artifact，表里保留索引：

- `run_id`、`sample_id`、`prompt_hash`、`session_id`、`cache_key`。
- `input_tokens`、`cached_tokens`、`output_tokens`。
- `ttft_ms`、`first_byte_ms`、`total_latency_ms`、`tpot_ms`、`inter_packet_max_ms`。
- `http_status`、`ok`、`failure_class`、`trace_id`、`request_id`。

## 准入方法

供应商准入不再只由 `SupplierScorecard.score` 决定，而是：

1. 没有 active `SlaContract`：只能生成 `observe`，不能 `admit`。
2. 没有最新 passed `SlaProbeRun`：只能生成 `observe`，不能 `admit`。
3. hard gate fail：生成 `reject` 或 `observe`，取决于失败类别是否可重测。
4. hard gate pass 后，再叠加 `SupplierScorecard` 的成功率、cache、毛利、供给余量和单位成本。
5. `SupplierEvaluation` 记录 `sla_contract_id`、`sla_probe_run_id` 和 gate summary；approve/apply 仍然只由 admin 操作。

这保持现有边界：自动化工具产出证据，不自动创建 supplier/channel、不自动调权、不自动禁用、不触碰资金动作。

## 运行时定期抽查

运行时探针用于发现供应商退化，不用于替代正式准入。

建议分三类：

1. `runtime_light`：每 5-15 分钟对 active supplier/model/sla 做低成本探针，覆盖健康、usage、streaming、session/cache 透传和一小段 TTFT。
2. `runtime_deep`：每 1-24 小时按合同抽样，覆盖关键 input/output bucket 和 cache replay。
3. `incident_recheck`：当 `UsageLedger` / quality summary 出现错误率、TTFT、cache hit、streaming usage 异常时触发。

运行时结果写入 `SlaProbeRun`，并可生成 `OperatingInsight`：

- `quality_watch`：轻量探针失败但未达到强制复测阈值。
- `capacity_risk`：合同内请求率开始排队或 429 异常。
- `action`：连续窗口 hard gate fail，建议人工复评 supplier 或下调承诺。

第一版不自动 disable channel 或改 routing policy；后续如要自动化，也必须经过新的 ADR。

## API 规划

建议新增 admin API：

- `GET /api/sla_contracts`
- `POST /api/sla_contracts/import`
- `POST /api/sla_probe_plans/generate`
- `GET /api/sla_probe_plans`
- `POST /api/sla_probe_runs/record`
- `GET /api/sla_probe_runs`
- `GET /api/sla_probe_runs/:id`

`/api/supplier_evaluations/generate` 后续接受可选 `sla_contract_id` / `sla_probe_run_id`，用于生成模型 SLA 维度的 admission evaluation。

## 实施顺序

1. 先落 `SlaContract` JSON schema 和只读导入，不做 runner。
2. 增加 `SlaProbePlan` / `SlaProbeRun` record API，允许外部 runner 写回证据。
3. 实现 `token-router-sla probe run`，先支持 OpenAI-compatible chat streaming。
4. 把 Kimi K2.5 官方 SLA 与 GLM-5/5.1 合同 profile 导入为示例 contract。
5. `SupplierEvaluation` 引用最近一次 passed admission run，作为 `admit` 的硬前置。
6. 增加 runtime_light 定时抽查和 `OperatingInsight` 生成。

## 验收

第一版施工完成后，需要证明：

1. 可以导入 Kimi/GLM contract，并回读合同维度。
2. 可以生成 admission plan，且计划里包含 input/output/cache/stream/error/availability 维度。
3. runner 能对 mock OpenAI-compatible endpoint 采集 streaming TTFT、TPOT、usage 和失败分类。
4. cold no-cache 与 warm cache 样本分开统计，cache 命中不会混入 cold SLA 证明。
5. `SlaProbeRun` record 后可查询 summary 和 artifact hash。
6. `SupplierEvaluation` 在无 passed SLA run 时不会给出 `admit`。
7. 运行时 light probe 失败只生成 insight / review 证据，不自动改 supplier/channel/capacity/routing policy。
