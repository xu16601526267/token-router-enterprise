/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export type ApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export type PageData<T> = {
  page: number
  page_size: number
  total: number
  items: T[]
}

export type SupplierType = 'third_party' | 'self_operated' | 'self_hosted'

export type Supplier = {
  id: number
  name: string
  type: SupplierType | string
  status: number
  notes?: string
  created_time: number
  updated_time: number
}

export type SupplierInput = {
  id?: number
  name: string
  type: SupplierType
  status: number
  notes?: string
}

export type SupplierAgreement = {
  id: number
  supplier_id: number
  model_name: string
  effective_from: number
  effective_to: number
  use_price: boolean
  cost_model_ratio: number
  cost_completion_ratio: number
  cost_cache_ratio: number
  cost_cache_creation_ratio: number
  cost_model_price: number
  priority: number
  status: number
  notes?: string
  created_time: number
  updated_time: number
}

export type SupplierAgreementInput = {
  id?: number
  supplier_id: number
  model_name: string
  effective_from: number
  effective_to: number
  use_price: boolean
  cost_model_ratio: number
  cost_completion_ratio: number
  cost_cache_ratio: number
  cost_cache_creation_ratio: number
  cost_model_price: number
  priority: number
  status: number
  notes?: string
}

export type SupplyCapacityTelemetrySourceType =
  | 'node_report'
  | 'external'
  | 'manual'

export type SupplyCostProfileSourceType = 'manual' | 'accounting' | 'external'

export type SupplyPrepaidLotSourceType = 'manual' | 'accounting' | 'external'

export type SupplyCapacity = {
  id: number
  supplier_id: number
  supply_node: string
  model_name: string
  period_start: number
  period_end: number
  capacity_tokens: number
  used_tokens: number
  headroom_tokens: number
  utilization_rate: number
  gpu_utilization_rate: number
  quality_score: number
  unit_cost_quota: number
  telemetry_source_type: SupplyCapacityTelemetrySourceType | string
  telemetry_source_ref: string
  telemetry_observed_at: number
  last_telemetry_id: number
  status: number
  notes?: string
  created_time: number
  updated_time: number
}

export type SupplyCapacityTelemetry = {
  id: number
  telemetry_key: string
  supplier_id: number
  supply_node: string
  model_name: string
  period_start: number
  period_end: number
  capacity_tokens: number
  used_tokens: number
  headroom_tokens: number
  utilization_rate: number
  gpu_utilization_rate: number
  quality_score: number
  unit_cost_quota: number
  source_type: SupplyCapacityTelemetrySourceType | string
  source_ref: string
  observed_at: number
  applied_capacity_id: number
  recorded_by: number
  notes?: string
  created_at: number
  updated_at: number
}

export type SupplyCapacityTelemetryRecordInput = {
  telemetry_key?: string
  supplier_id: number
  supply_node: string
  model_name: string
  period_start: number
  period_end: number
  capacity_tokens: number
  used_tokens: number
  gpu_utilization_rate: number
  quality_score: number
  unit_cost_quota: number
  source_type: SupplyCapacityTelemetrySourceType | string
  source_ref: string
  observed_at: number
  notes?: string
}

export type SupplyCostProfile = {
  id: number
  cost_profile_key: string
  supplier_id: number
  supply_node: string
  model_name: string
  period_start: number
  period_end: number
  capacity_tokens: number
  fixed_cost_quota: number
  variable_unit_cost_quota: number
  amortized_unit_cost_quota: number
  source_type: SupplyCostProfileSourceType | string
  source_ref: string
  observed_at: number
  recorded_by: number
  notes?: string
  created_at: number
  updated_at: number
}

export type SupplyCostProfileRecordInput = {
  cost_profile_key?: string
  supplier_id: number
  supply_node: string
  model_name: string
  period_start: number
  period_end: number
  capacity_tokens: number
  fixed_cost_quota: number
  variable_unit_cost_quota: number
  source_type: SupplyCostProfileSourceType | string
  source_ref: string
  observed_at: number
  notes?: string
}

export type SupplyPrepaidLot = {
  id: number
  prepaid_lot_key: string
  supplier_id: number
  channel_id: number
  supply_node: string
  model_name: string
  period_start: number
  period_end: number
  purchased_tokens: number
  unit_cost_quota: number
  total_cost_quota: number
  drawdown_tokens: number
  drawdown_request_count: number
  remaining_tokens: number
  drawdown_rate: number
  drawdown_source_type: string
  drawdown_source_ref: string
  drawdown_refreshed_at: number
  source_type: SupplyPrepaidLotSourceType | string
  source_ref: string
  observed_at: number
  external_ref: string
  recorded_by: number
  notes?: string
  created_at: number
  updated_at: number
}

export type SupplyPrepaidLotRecordInput = {
  prepaid_lot_key?: string
  supplier_id: number
  channel_id?: number
  supply_node?: string
  model_name?: string
  period_start: number
  period_end: number
  purchased_tokens: number
  unit_cost_quota: number
  source_type: SupplyPrepaidLotSourceType | string
  source_ref: string
  observed_at: number
  external_ref?: string
  notes?: string
}

export type SupplyPrepaidLotUsageRefreshInput = {
  prepaid_lot_id?: number
  supplier_id?: number
  channel_id?: number
  supply_node?: string
  model_name?: string
  source_type?: SupplyPrepaidLotSourceType | string
  start_timestamp?: number
  end_timestamp?: number
}

export type TrafficProfile = {
  id: number
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  period_start: number
  period_end: number
  request_count: number
  success_request_count: number
  demand_tokens: number
  peak_tokens: number
  peak_ratio: number
  unique_sessions: number
  cache_hit_count: number
  cache_hit_rate: number
  total_cached_tokens: number
  sla_met_rate: number
  avg_latency_ms: number
  max_latency_ms: number
  total_sell_quota: number
  total_cost_quota: number
  gross_profit_quota: number
  supply_capacity_tokens: number
  supply_used_tokens: number
  supply_headroom_tokens: number
  avg_supply_quality_score: number
  avg_unit_cost_quota: number
  generated_at: number
  created_at: number
  updated_at: number
}

export type GenerateTrafficProfileInput = {
  period_start: number
  period_end: number
  model_name?: string
  sla_tier?: string
  user_id?: number
}

export type TrafficForecast = {
  id: number
  forecast_key: string
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  source_period_start: number
  source_period_end: number
  target_period_start: number
  target_period_end: number
  source_profile_count: number
  observed_request_count: number
  observed_demand_tokens: number
  observed_peak_tokens: number
  baseline_demand_tokens: number
  forecast_demand_tokens: number
  forecast_peak_tokens: number
  forecast_headroom_tokens: number
  forecast_gap_tokens: number
  trend_demand_delta_tokens: number
  trend_demand_delta_rate: number
  seasonal_period_count: number
  seasonal_index: number
  seasonal_demand_tokens: number
  anomaly_status: string
  anomaly_profile_id: number
  anomaly_demand_ratio: number
  cache_hit_rate: number
  sla_met_rate: number
  gross_profit_quota: number
  avg_unit_cost_quota: number
  confidence: number
  method: string
  reason: string
  generated_at: number
  created_at: number
  updated_at: number
}

export type GenerateTrafficForecastInput = {
  period_start: number
  period_end: number
  target_period_start?: number
  target_period_end?: number
  model_name?: string
  sla_tier?: string
  user_id?: number
  seasonal_period_count?: number
  anomaly_guard?: boolean
  anomaly_threshold_rate?: number
}

export type SupplyDecisionStatus = 'draft' | 'approved' | 'rejected'
export type SupplyDecisionTrack =
  | 'third_party'
  | 'self_operated'
  | 'self_hosted'

export type SupplyDecision = {
  id: number
  decision_key: string
  traffic_profile_id: number
  traffic_forecast_id: number
  decision_source: 'profile' | 'forecast' | string
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  period_start: number
  period_end: number
  forecast_target_period_start: number
  forecast_target_period_end: number
  forecast_confidence: number
  forecast_method: string
  decision_type: string
  track: SupplyDecisionTrack | string
  status: SupplyDecisionStatus | string
  demand_tokens: number
  peak_tokens: number
  supply_headroom_tokens: number
  gap_tokens: number
  recommended_capacity: number
  cache_hit_rate: number
  sla_met_rate: number
  gross_profit_quota: number
  avg_supply_quality_score: number
  avg_unit_cost_quota: number
  roi_score: number
  reason: string
  generated_at: number
  reviewed_at: number
  reviewed_by: number
  review_note?: string
  created_at: number
  updated_at: number
}

export type GenerateSupplyDecisionInput = {
  period_start: number
  period_end: number
  model_name?: string
  sla_tier?: string
  user_id?: number
}

export type SupplyExpansionOpportunityType =
  | 'third_party_gap'
  | 'third_party_probe'
  | 'self_operated_bulk'
  | 'self_hosted_cache'

export type SupplyExpansionOpportunityPriority = 'info' | 'watch' | 'action'

export type SupplyExpansionOpportunity = {
  id: number
  opportunity_key: string
  supply_decision_id: number
  traffic_profile_id: number
  traffic_forecast_id: number
  decision_source: 'profile' | 'forecast' | string
  decision_status: SupplyDecisionStatus | string
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  period_start: number
  period_end: number
  forecast_target_period_start: number
  forecast_target_period_end: number
  forecast_confidence: number
  forecast_method: string
  opportunity_type: SupplyExpansionOpportunityType | string
  track: SupplyDecisionTrack | string
  decision_type: string
  priority: SupplyExpansionOpportunityPriority | string
  cluster_key: string
  demand_tokens: number
  peak_tokens: number
  supply_headroom_tokens: number
  gap_tokens: number
  recommended_capacity: number
  cache_hit_rate: number
  sla_met_rate: number
  gross_profit_quota: number
  avg_supply_quality_score: number
  avg_unit_cost_quota: number
  roi_score: number
  self_hosted_cost_profile_id: number
  self_hosted_unit_cost_quota: number
  self_hosted_savings_unit_quota: number
  self_hosted_savings_quota: number
  peak_ratio: number
  unique_sessions: number
  locality_score: number
  stability_score: number
  headroom_risk_score: number
  rank_score: number
  reason: string
  generated_at: number
  created_at: number
  updated_at: number
}

export type GenerateSupplyExpansionOpportunityInput = {
  period_start: number
  period_end: number
  model_name?: string
  sla_tier?: string
  user_id?: number
  decision_status?: SupplyDecisionStatus
  track?: SupplyDecisionTrack
}

export type PricingRecommendationStatus = 'draft' | 'approved' | 'rejected'
export type PricingRecommendationAction =
  | 'raise_price'
  | 'keep_price'
  | 'share_savings'

export type PricingRecommendation = {
  id: number
  recommendation_key: string
  traffic_profile_id: number
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  period_start: number
  period_end: number
  status: PricingRecommendationStatus | string
  action: PricingRecommendationAction | string
  request_count: number
  demand_tokens: number
  peak_tokens: number
  supply_headroom_tokens: number
  cache_hit_rate: number
  sla_met_rate: number
  avg_latency_ms: number
  max_latency_ms: number
  total_sell_quota: number
  total_cost_quota: number
  gross_profit_quota: number
  current_unit_price_quota: number
  current_unit_cost_quota: number
  current_margin_rate: number
  recommended_unit_price_quota: number
  recommended_margin_rate: number
  avg_supply_quality_score: number
  avg_unit_cost_quota: number
  reason: string
  generated_at: number
  reviewed_at: number
  reviewed_by: number
  review_note?: string
  created_at: number
  updated_at: number
}

export type GeneratePricingRecommendationInput = {
  period_start: number
  period_end: number
  model_name?: string
  sla_tier?: string
  user_id?: number
}

export type ReviewPricingRecommendationInput = {
  review_note?: string
}

export type OperatingInsightStatus = 'draft' | 'acknowledged' | 'dismissed'
export type OperatingInsightSeverity = 'info' | 'watch' | 'action'
export type OperatingInsightCategory =
  | 'cache_efficiency'
  | 'capacity_risk'
  | 'pricing_risk'
  | 'quality_watch'
  | 'steady_state'

export type OperatingInsight = {
  id: number
  insight_key: string
  traffic_profile_id: number
  supply_decision_id: number
  pricing_recommendation_id: number
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  period_start: number
  period_end: number
  status: OperatingInsightStatus | string
  severity: OperatingInsightSeverity | string
  category: OperatingInsightCategory | string
  title: string
  summary: string
  recommended_action: string
  demand_tokens: number
  peak_tokens: number
  supply_headroom_tokens: number
  cache_hit_rate: number
  sla_met_rate: number
  gross_profit_quota: number
  avg_unit_cost_quota: number
  supply_decision_track: string
  supply_decision_type: string
  supply_decision_status: string
  supply_decision_roi_score: number
  pricing_recommendation_action: string
  pricing_recommendation_status: string
  recommended_unit_price_quota: number
  recommended_margin_rate: number
  sla_contract_id: number
  sla_probe_run_id: number
  sla_probe_run_key: string
  sla_probe_status: string
  sla_hard_gate_passed: boolean
  sla_failure_reasons: string
  sla_artifact_sha256: string
  sla_runtime_ref: string
  generated_at: number
  reviewed_at: number
  reviewed_by: number
  review_note?: string
  created_at: number
  updated_at: number
}

export type GenerateOperatingInsightInput = {
  period_start: number
  period_end: number
  model_name?: string
  sla_tier?: string
  user_id?: number
}

export type ReviewOperatingInsightInput = {
  review_note?: string
}

export type SupplyActionPlanStatus =
  | 'planned'
  | 'in_progress'
  | 'completed'
  | 'cancelled'

export type SupplyActionPlan = {
  id: number
  supply_decision_id: number
  decision_key: string
  traffic_profile_id: number
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  period_start: number
  period_end: number
  decision_type: string
  track: SupplyDecisionTrack | string
  action_type: string
  supply_expansion_opportunity_id: number
  opportunity_key: string
  opportunity_type: SupplyExpansionOpportunityType | string
  opportunity_priority: SupplyExpansionOpportunityPriority | string
  opportunity_cluster_key: string
  opportunity_rank_score: number
  status: SupplyActionPlanStatus | string
  recommended_capacity: number
  gap_tokens: number
  roi_score: number
  reason: string
  source_reviewed_at: number
  source_reviewed_by: number
  generated_at: number
  started_at: number
  completed_at: number
  cancelled_at: number
  status_updated_at: number
  status_updated_by: number
  operator_note: string
  created_at: number
  updated_at: number
}

export type GenerateSupplyActionPlanInput = {
  decision_id?: number
  period_start?: number
  period_end?: number
  track?: SupplyDecisionTrack
}

export type UpdateSupplyActionPlanStatusInput = {
  status: SupplyActionPlanStatus
  operator_note?: string
}

export type SupplyActionExecutionStatus = 'recorded'

export type SupplyActionExecution = {
  id: number
  supply_action_plan_id: number
  supply_decision_id: number
  decision_key: string
  traffic_profile_id: number
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  period_start: number
  period_end: number
  decision_type: string
  track: SupplyDecisionTrack | string
  action_type: string
  execution_status: SupplyActionExecutionStatus | string
  supplier_id: number
  channel_id: number
  supply_capacity_id: number
  recommended_capacity: number
  actual_capacity_tokens: number
  gap_tokens: number
  roi_score: number
  unit_cost_quota: number
  drawdown_tokens: number
  drawdown_request_count: number
  remaining_tokens: number
  drawdown_rate: number
  drawdown_source_type: string
  drawdown_source_ref: string
  drawdown_refreshed_at: number
  effective_from: number
  effective_to: number
  external_ref: string
  operator_note: string
  action_plan_completed_at: number
  action_plan_completed_by: number
  recorded_at: number
  recorded_by: number
  created_at: number
  updated_at: number
}

export type SupplyActionExecutionRecordInput = {
  supply_action_plan_id: number
  execution_status: SupplyActionExecutionStatus
  supplier_id?: number
  channel_id?: number
  supply_capacity_id?: number
  actual_capacity_tokens: number
  unit_cost_quota: number
  effective_from?: number
  effective_to?: number
  external_ref?: string
  operator_note?: string
}

export type SupplyActionExecutionUsageRefreshInput = {
  execution_id?: number
  supply_action_plan_id?: number
  supply_decision_id?: number
  execution_status?: SupplyActionExecutionStatus
  track?: string
  supplier_id?: number
  channel_id?: number
  supply_capacity_id?: number
  start_timestamp?: number
  end_timestamp?: number
}

export type SupplyRoutingPolicyStatus = 'active' | 'disabled'

export type SupplyRoutingPolicy = {
  id: number
  supply_action_execution_id: number
  supply_action_plan_id: number
  supply_decision_id: number
  decision_key: string
  traffic_profile_id: number
  slice_key: string
  model_name: string
  sla_tier: string
  user_id: number
  period_start: number
  period_end: number
  track: SupplyDecisionTrack | string
  action_type: string
  status: SupplyRoutingPolicyStatus | string
  supplier_id: number
  channel_id: number
  supply_capacity_id: number
  sla_contract_id: number
  sla_probe_run_id: number
  sla_probe_run_key: string
  sla_artifact_sha256: string
  sla_runtime_ref: string
  effective_from: number
  effective_to: number
  priority: number
  traffic_percent: number
  reason: string
  activated_at: number
  activated_by: number
  disabled_at: number
  disabled_by: number
  operator_note: string
  created_at: number
  updated_at: number
}

export type ActivateSupplyRoutingPolicyInput = {
  supply_action_execution_id: number
  priority?: number
  traffic_percent?: number
  operator_note?: string
}

export type DisableSupplyRoutingPolicyInput = {
  operator_note?: string
}

export type SlaContractStatus = 'draft' | 'active' | 'retired'

export type SlaContract = {
  id: number
  contract_key: string
  model_name: string
  model_aliases: string
  provider_family: string
  source_name: string
  source_ref: string
  source_sha256: string
  version: string
  status: SlaContractStatus | string
  effective_from: number
  effective_to: number
  measurement_profile_json: string
  hard_gate_json: string
  soft_gate_json: string
  imported_at: number
  imported_by: number
  created_at: number
  updated_at: number
}

export type SlaContractImportInput = {
  contract_key: string
  model_name: string
  model_aliases?: string
  provider_family: string
  source_name: string
  source_ref: string
  source_sha256: string
  version: string
  status: SlaContractStatus
  effective_from?: number
  effective_to?: number
  measurement_profile_json: string
  hard_gate_json?: string
  soft_gate_json?: string
}

export type SlaProbeType =
  | 'admission'
  | 'runtime_light'
  | 'runtime_deep'
  | 'incident_recheck'

export type SlaProbeRouteMode = 'direct_upstream' | 'through_token_router'

export type SlaProbePlan = {
  id: number
  plan_key: string
  contract_id: number
  supplier_id: number
  channel_id: number
  model_name: string
  sla_tier: string
  probe_type: SlaProbeType | string
  route_mode: SlaProbeRouteMode | string
  prompt_suite_key: string
  tokenizer_ref: string
  sample_size: number
  repeat_count: number
  input_profile_json: string
  output_profile_json: string
  concurrency_profile_json: string
  rate_profile_json: string
  stream_profile_json: string
  error_profile_json: string
  availability_profile_json: string
  cache_profile: string
  schedule_interval_seconds: number
  jitter_seconds: number
  max_probe_quota: number
  measurement_profile_snapshot: string
  generated_at: number
  generated_by: number
  created_at: number
  updated_at: number
}

export type SlaProbePlanGenerateInput = {
  contract_id?: number
  contract_key?: string
  supplier_id: number
  channel_id?: number
  model_name?: string
  sla_tier?: string
  probe_type: SlaProbeType
  route_mode: SlaProbeRouteMode
  prompt_suite_key?: string
  tokenizer_ref?: string
  sample_size?: number
  repeat_count?: number
  input_profile_json?: string
  output_profile_json?: string
  concurrency_profile_json?: string
  rate_profile_json?: string
  stream_profile_json?: string
  error_profile_json?: string
  availability_profile_json?: string
  cache_profile?: string
  schedule_interval_seconds?: number
  jitter_seconds?: number
  max_probe_quota?: number
}

export type SlaProbeRunStatus =
  | 'running'
  | 'passed'
  | 'failed'
  | 'invalid'
  | 'cancelled'

export type SlaProbeRun = {
  id: number
  run_key: string
  plan_id: number
  contract_id: number
  supplier_id: number
  channel_id: number
  status: SlaProbeRunStatus | string
  started_at: number
  ended_at: number
  runner_version: string
  git_commit: string
  runtime_ref: string
  endpoint: string
  route_mode: SlaProbeRouteMode | string
  model_name: string
  sla_tier: string
  summary_json: string
  hard_gate_passed: boolean
  soft_gate_warnings: string
  failure_reasons: string
  artifact_uri: string
  artifact_sha256: string
  recorded_at: number
  recorded_by: number
  created_at: number
  updated_at: number
}

export type SlaProbeRunRecordInput = {
  run_key?: string
  plan_id: number
  status: SlaProbeRunStatus
  started_at?: number
  ended_at?: number
  runner_version?: string
  git_commit?: string
  runtime_ref?: string
  endpoint?: string
  summary_json?: string
  hard_gate_passed?: boolean
  soft_gate_warnings?: string
  failure_reasons?: string
  artifact_uri?: string
  artifact_sha256?: string
}

export type SupplierScorecardGrade = 'A' | 'B' | 'C' | 'D'

export type SupplierScorecard = {
  id: number
  supplier_id: number
  period_start: number
  period_end: number
  total_requests: number
  success_requests: number
  error_requests: number
  success_rate: number
  avg_latency_ms: number
  max_latency_ms: number
  cache_hit_count: number
  cache_hit_rate: number
  total_sell_quota: number
  total_cost_quota: number
  gross_profit_quota: number
  supply_capacity_tokens: number
  supply_used_tokens: number
  supply_headroom_tokens: number
  avg_supply_quality_score: number
  avg_unit_cost_quota: number
  score: number
  grade: SupplierScorecardGrade | string
  generated_at: number
  created_at: number
  updated_at: number
}

export type GenerateSupplierScorecardInput = {
  period_start: number
  period_end: number
  supplier_id?: number
}

export type SupplierEvaluationStatus = 'draft' | 'approved' | 'rejected'
export type SupplierEvaluationRecommendation = 'admit' | 'observe' | 'reject'

export type SupplierEvaluation = {
  id: number
  evaluation_key: string
  evaluation_type: string
  supplier_id: number
  supplier_scorecard_id: number
  sla_contract_id: number
  sla_probe_run_id: number
  sla_gate_summary_json: string
  period_start: number
  period_end: number
  status: SupplierEvaluationStatus | string
  recommendation: SupplierEvaluationRecommendation | string
  score: number
  grade: SupplierScorecardGrade | string
  total_requests: number
  success_rate: number
  avg_latency_ms: number
  cache_hit_rate: number
  gross_profit_quota: number
  supply_headroom_tokens: number
  avg_supply_quality_score: number
  avg_unit_cost_quota: number
  reason: string
  generated_at: number
  reviewed_at: number
  reviewed_by: number
  review_note?: string
  applied_at: number
  applied_by: number
  applied_note?: string
  supplier_status_before: number
  supplier_status_after: number
  created_at: number
  updated_at: number
}

export type GenerateSupplierEvaluationInput = {
  period_start: number
  period_end: number
  supplier_id?: number
}

export type ReviewSupplierEvaluationInput = {
  review_note?: string
}

export type ApplySupplierEvaluationInput = {
  operator_note?: string
}

export type SupplierPostureRecommendationStatus =
  | 'draft'
  | 'approved'
  | 'rejected'
  | 'applied'
export type SupplierPostureRecommendationAction =
  | 'boost'
  | 'observe'
  | 'downgrade'
  | 'disable'

export type SupplierPostureRecommendation = {
  id: number
  recommendation_key: string
  supplier_id: number
  supplier_scorecard_id: number
  period_start: number
  period_end: number
  status: SupplierPostureRecommendationStatus | string
  recommended_action: SupplierPostureRecommendationAction | string
  score: number
  grade: SupplierScorecardGrade | string
  total_requests: number
  success_rate: number
  avg_latency_ms: number
  supply_headroom_tokens: number
  avg_supply_quality_score: number
  quality_insight_count: number
  capacity_insight_count: number
  action_insight_count: number
  supplier_status_current: number
  reason: string
  evidence_json: string
  generated_at: number
  reviewed_at: number
  reviewed_by: number
  review_note?: string
  applied_at: number
  applied_by: number
  applied_note?: string
  supplier_status_before: number
  supplier_status_after: number
  created_at: number
  updated_at: number
}

export type GenerateSupplierPostureRecommendationInput = {
  period_start: number
  period_end: number
  supplier_id?: number
}

export type ReviewSupplierPostureRecommendationInput = {
  review_note?: string
}

export type ApplySupplierPostureRecommendationInput = {
  operator_note?: string
}

export type SupplierRoutePreferenceActivateInput = {
  supplier_id: number
  weight_percent: number
  reason: string
  effective_from?: number
  effective_to?: number
  operator_note?: string
}

export type SupplierRoutePreferenceDisableInput = {
  operator_note?: string
}

export type SupplierRoutePreferenceStatus = 'active' | 'disabled'

export type SupplierRoutePreference = {
  id: number
  supplier_id: number
  source_posture_recommendation_id: number
  status: SupplierRoutePreferenceStatus | string
  weight_percent: number
  reason: string
  effective_from: number
  effective_to: number
  activated_at: number
  activated_by: number
  disabled_at: number
  disabled_by: number
  operator_note?: string
  created_at: number
  updated_at: number
}

export type UsageLedger = {
  id: number
  request_id: string
  session_id: string
  supplier_id: number
  channel_id: number
  user_id: number
  token_id: number
  model_name: string
  prompt_tokens: number
  fresh_prompt_tokens: number
  cached_tokens: number
  cache_creation_tokens: number
  completion_tokens: number
  sell_quota: number
  cost_quota: number
  cache_hit: boolean
  latency_ms: number
  status: string
  sla_tier: string
  supply_node: string
  created_at: number
}

export type MarginGroupBy = 'supplier' | 'channel' | 'user' | 'model' | 'day'

export type MarginSummaryRow = {
  group_key: string
  bucket_start?: number
  supplier_id?: number
  channel_id?: number
  user_id?: number
  model_name?: string
  total_requests: number
  total_sell_quota: number
  total_cost_quota: number
  gross_profit_quota: number
  total_prompt_tokens: number
  total_cached_tokens: number
  total_completion_tokens: number
  cache_hit_count: number
  cache_hit_rate: number
}

export type QualityGroupBy =
  | 'supplier'
  | 'channel'
  | 'user'
  | 'model'
  | 'sla_tier'
  | 'supply_node'
  | 'day'

export type QualitySummaryRow = {
  group_key: string
  bucket_start?: number
  supplier_id?: number
  channel_id?: number
  user_id?: number
  model_name?: string
  sla_tier?: string
  supply_node?: string
  total_requests: number
  success_requests: number
  error_requests: number
  success_rate: number
  avg_latency_ms: number
  max_latency_ms: number
  total_prompt_tokens: number
  total_cached_tokens: number
  total_completion_tokens: number
  total_sell_quota: number
  total_cost_quota: number
  gross_profit_quota: number
  cache_hit_count: number
  cache_hit_rate: number
}

export type SettlementSubjectType = 'supplier' | 'user'

export type SettlementStatement = {
  id: number
  subject_type: SettlementSubjectType
  supplier_id: number
  user_id: number
  period_start: number
  period_end: number
  total_sell_quota: number
  total_cost_quota: number
  gross_profit_quota: number
  total_requests: number
  total_prompt_tokens: number
  total_cached_tokens: number
  total_completion_tokens: number
  cache_hit_rate: number
  status: string
  generated_at: number
  created_at: number
  updated_at: number
}

export type GenerateSettlementInput = {
  subject_type: SettlementSubjectType
  supplier_id?: number
  user_id?: number
  period_start: number
  period_end: number
}
