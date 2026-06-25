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
import { api, type ApiRequestConfig } from '@/lib/api'

import type {
  ActivateSupplyRoutingPolicyInput,
  ApplySupplierPostureRecommendationInput,
  ApplySupplierEvaluationInput,
  ApiResponse,
  DisableSupplyRoutingPolicyInput,
  GenerateOperatingInsightInput,
  GenerateSupplyExpansionOpportunityInput,
  GenerateSupplyActionPlanInput,
  GenerateTrafficForecastInput,
  GeneratePricingRecommendationInput,
  GenerateSupplierEvaluationInput,
  GenerateSupplierPostureRecommendationInput,
  GenerateSupplierScorecardInput,
  GenerateTrafficProfileInput,
  GenerateSupplyDecisionInput,
  GenerateSettlementInput,
  MarginGroupBy,
  MarginSummaryRow,
  OperatingInsight,
  OperatingInsightCategory,
  OperatingInsightSeverity,
  OperatingInsightStatus,
  PageData,
  PricingRecommendation,
  PricingRecommendationAction,
  PricingRecommendationStatus,
  QualityGroupBy,
  QualitySummaryRow,
  ReviewOperatingInsightInput,
  ReviewPricingRecommendationInput,
  ReviewSupplierEvaluationInput,
  ReviewSupplierPostureRecommendationInput,
  SettlementStatement,
  SlaContract,
  SlaContractImportInput,
  SlaContractStatus,
  SlaProbePlan,
  SlaProbePlanGenerateInput,
  SlaProbeRouteMode,
  SlaProbeRun,
  SlaProbeRunRecordInput,
  SlaProbeRunStatus,
  SlaProbeType,
  SupplyCapacity,
  SupplyCapacityTelemetry,
  SupplyCapacityTelemetryRecordInput,
  SupplyCapacityTelemetrySourceType,
  SupplyCostProfile,
  SupplyCostProfileRecordInput,
  SupplyCostProfileSourceType,
  SupplyPrepaidLot,
  SupplyPrepaidLotRecordInput,
  SupplyPrepaidLotSourceType,
  SupplyPrepaidLotUsageRefreshInput,
  SupplyActionExecution,
  SupplyActionExecutionRecordInput,
  SupplyActionExecutionStatus,
  SupplyActionExecutionUsageRefreshInput,
  SupplyActionPlan,
  SupplyActionPlanStatus,
  SupplyDecision,
  SupplyDecisionStatus,
  SupplyExpansionOpportunity,
  SupplyExpansionOpportunityPriority,
  SupplyExpansionOpportunityType,
  SupplyRoutingPolicy,
  SupplyRoutingPolicyStatus,
  SupplierEvaluation,
  SupplierEvaluationRecommendation,
  SupplierEvaluationStatus,
  SupplierPostureRecommendation,
  SupplierPostureRecommendationAction,
  SupplierPostureRecommendationStatus,
  SupplierRoutePreference,
  SupplierRoutePreferenceActivateInput,
  SupplierRoutePreferenceDisableInput,
  SupplierRoutePreferenceStatus,
  SupplierScorecard,
  SupplierScorecardGrade,
  TrafficProfile,
  Supplier,
  SupplierAgreement,
  SupplierAgreementInput,
  SupplierInput,
  UpdateSupplyActionPlanStatusInput,
  TrafficForecast,
  UsageLedger,
} from './types'

const actionConfig = (config: ApiRequestConfig = {}): ApiRequestConfig => ({
  ...config,
  skipBusinessError: true,
  skipErrorHandler: true,
})

export type PageParams = {
  p?: number
  page_size?: number
}

export type TimeRangeParams = {
  start_timestamp?: number
  end_timestamp?: number
}

export async function getSuppliers(
  params: PageParams = {}
): Promise<ApiResponse<PageData<Supplier>>> {
  const res = await api.get('/api/suppliers', { params })
  return res.data
}

export async function createSupplier(
  input: SupplierInput
): Promise<ApiResponse<Supplier>> {
  const res = await api.post('/api/suppliers', input, actionConfig())
  return res.data
}

export async function updateSupplier(
  input: SupplierInput
): Promise<ApiResponse<Supplier>> {
  const res = await api.put('/api/suppliers', input, actionConfig())
  return res.data
}

export async function getSupplierAgreements(
  params: PageParams & { supplier_id?: number } = {}
): Promise<ApiResponse<PageData<SupplierAgreement>>> {
  const res = await api.get('/api/supplier_agreements', { params })
  return res.data
}

export async function getSupplyCapacities(
  params: PageParams & TimeRangeParams = {}
): Promise<ApiResponse<PageData<SupplyCapacity>>> {
  const res = await api.get('/api/supply_capacities', { params })
  return res.data
}

export async function getSupplyCapacityTelemetries(
  params: PageParams &
    TimeRangeParams & {
      supplier_id?: number
      supply_node?: string
      model_name?: string
      source_type?: SupplyCapacityTelemetrySourceType | string
      applied_capacity_id?: number
    } = {}
): Promise<ApiResponse<PageData<SupplyCapacityTelemetry>>> {
  const res = await api.get('/api/supply_capacity_telemetries', { params })
  return res.data
}

export async function recordSupplyCapacityTelemetry(
  input: SupplyCapacityTelemetryRecordInput
): Promise<ApiResponse<SupplyCapacityTelemetry>> {
  const res = await api.post(
    '/api/supply_capacity_telemetries/record',
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplyCostProfiles(
  params: PageParams &
    TimeRangeParams & {
      supplier_id?: number
      supply_node?: string
      model_name?: string
      source_type?: SupplyCostProfileSourceType | string
    } = {}
): Promise<ApiResponse<PageData<SupplyCostProfile>>> {
  const res = await api.get('/api/supply_cost_profiles', { params })
  return res.data
}

export async function recordSupplyCostProfile(
  input: SupplyCostProfileRecordInput
): Promise<ApiResponse<SupplyCostProfile>> {
  const res = await api.post(
    '/api/supply_cost_profiles/record',
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplyPrepaidLots(
  params: PageParams &
    TimeRangeParams & {
      prepaid_lot_id?: number
      supplier_id?: number
      channel_id?: number
      supply_node?: string
      model_name?: string
      source_type?: SupplyPrepaidLotSourceType | string
    } = {}
): Promise<ApiResponse<PageData<SupplyPrepaidLot>>> {
  const res = await api.get('/api/supply_prepaid_lots', { params })
  return res.data
}

export async function recordSupplyPrepaidLot(
  input: SupplyPrepaidLotRecordInput
): Promise<ApiResponse<SupplyPrepaidLot>> {
  const res = await api.post(
    '/api/supply_prepaid_lots/record',
    input,
    actionConfig()
  )
  return res.data
}

export async function refreshSupplyPrepaidLotUsage(
  input: SupplyPrepaidLotUsageRefreshInput = {}
): Promise<ApiResponse<SupplyPrepaidLot[]>> {
  const res = await api.post(
    '/api/supply_prepaid_lots/refresh_usage',
    input,
    actionConfig()
  )
  return res.data
}

export async function getTrafficProfiles(
  params: PageParams &
    TimeRangeParams & {
      model_name?: string
      sla_tier?: string
      user_id?: number
    } = {}
): Promise<ApiResponse<PageData<TrafficProfile>>> {
  const res = await api.get('/api/traffic_profiles', { params })
  return res.data
}

export async function generateTrafficProfiles(
  input: GenerateTrafficProfileInput
): Promise<ApiResponse<TrafficProfile[]>> {
  const res = await api.post(
    '/api/traffic_profiles/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function getTrafficForecasts(
  params: PageParams & {
    model_name?: string
    sla_tier?: string
    user_id?: number
    method?: string
    source_start_timestamp?: number
    source_end_timestamp?: number
    target_start_timestamp?: number
    target_end_timestamp?: number
  } = {}
): Promise<ApiResponse<PageData<TrafficForecast>>> {
  const res = await api.get('/api/traffic_forecasts', { params })
  return res.data
}

export async function generateTrafficForecasts(
  input: GenerateTrafficForecastInput
): Promise<ApiResponse<TrafficForecast[]>> {
  const res = await api.post(
    '/api/traffic_forecasts/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplyDecisions(
  params: PageParams &
    TimeRangeParams & {
      status?: SupplyDecisionStatus
      track?: string
      decision_type?: string
    } = {}
): Promise<ApiResponse<PageData<SupplyDecision>>> {
  const res = await api.get('/api/supply_decisions', { params })
  return res.data
}

export async function generateSupplyDecisions(
  input: GenerateSupplyDecisionInput
): Promise<ApiResponse<SupplyDecision[]>> {
  const res = await api.post(
    '/api/supply_decisions/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function approveSupplyDecision(
  id: number
): Promise<ApiResponse<SupplyDecision>> {
  const res = await api.post(
    `/api/supply_decisions/${id}/approve`,
    { review_note: 'approved from dashboard' },
    actionConfig()
  )
  return res.data
}

export async function rejectSupplyDecision(
  id: number
): Promise<ApiResponse<SupplyDecision>> {
  const res = await api.post(
    `/api/supply_decisions/${id}/reject`,
    { review_note: 'rejected from dashboard' },
    actionConfig()
  )
  return res.data
}

export async function getSupplyExpansionOpportunities(
  params: PageParams &
    TimeRangeParams & {
      decision_status?: SupplyDecisionStatus
      track?: string
      opportunity_type?: SupplyExpansionOpportunityType
      priority?: SupplyExpansionOpportunityPriority
      cluster_key?: string
    } = {}
): Promise<ApiResponse<PageData<SupplyExpansionOpportunity>>> {
  const res = await api.get('/api/supply_expansion_opportunities', { params })
  return res.data
}

export async function generateSupplyExpansionOpportunities(
  input: GenerateSupplyExpansionOpportunityInput
): Promise<ApiResponse<SupplyExpansionOpportunity[]>> {
  const res = await api.post(
    '/api/supply_expansion_opportunities/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function getPricingRecommendations(
  params: PageParams &
    TimeRangeParams & {
      status?: PricingRecommendationStatus
      action?: PricingRecommendationAction
      model_name?: string
      sla_tier?: string
      user_id?: number
    } = {}
): Promise<ApiResponse<PageData<PricingRecommendation>>> {
  const res = await api.get('/api/pricing_recommendations', { params })
  return res.data
}

export async function generatePricingRecommendations(
  input: GeneratePricingRecommendationInput
): Promise<ApiResponse<PricingRecommendation[]>> {
  const res = await api.post(
    '/api/pricing_recommendations/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function approvePricingRecommendation(
  id: number,
  input: ReviewPricingRecommendationInput = {
    review_note: 'approved from dashboard',
  }
): Promise<ApiResponse<PricingRecommendation>> {
  const res = await api.post(
    `/api/pricing_recommendations/${id}/approve`,
    input,
    actionConfig()
  )
  return res.data
}

export async function rejectPricingRecommendation(
  id: number,
  input: ReviewPricingRecommendationInput = {
    review_note: 'rejected from dashboard',
  }
): Promise<ApiResponse<PricingRecommendation>> {
  const res = await api.post(
    `/api/pricing_recommendations/${id}/reject`,
    input,
    actionConfig()
  )
  return res.data
}

export async function getOperatingInsights(
  params: PageParams &
    TimeRangeParams & {
      status?: OperatingInsightStatus
      severity?: OperatingInsightSeverity
      category?: OperatingInsightCategory
      model_name?: string
      sla_tier?: string
      user_id?: number
    } = {}
): Promise<ApiResponse<PageData<OperatingInsight>>> {
  const res = await api.get('/api/operating_insights', { params })
  return res.data
}

export async function generateOperatingInsights(
  input: GenerateOperatingInsightInput
): Promise<ApiResponse<OperatingInsight[]>> {
  const res = await api.post(
    '/api/operating_insights/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function acknowledgeOperatingInsight(
  id: number,
  input: ReviewOperatingInsightInput = {
    review_note: 'acknowledged from dashboard',
  }
): Promise<ApiResponse<OperatingInsight>> {
  const res = await api.post(
    `/api/operating_insights/${id}/acknowledge`,
    input,
    actionConfig()
  )
  return res.data
}

export async function dismissOperatingInsight(
  id: number,
  input: ReviewOperatingInsightInput = {
    review_note: 'dismissed from dashboard',
  }
): Promise<ApiResponse<OperatingInsight>> {
  const res = await api.post(
    `/api/operating_insights/${id}/dismiss`,
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplyActionPlans(
  params: PageParams &
    TimeRangeParams & {
      decision_id?: number
      status?: SupplyActionPlanStatus
      track?: string
    } = {}
): Promise<ApiResponse<PageData<SupplyActionPlan>>> {
  const res = await api.get('/api/supply_action_plans/', { params })
  return res.data
}

export async function generateSupplyActionPlans(
  input: GenerateSupplyActionPlanInput
): Promise<ApiResponse<SupplyActionPlan[]>> {
  const res = await api.post(
    '/api/supply_action_plans/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function updateSupplyActionPlanStatus(
  id: number,
  input: UpdateSupplyActionPlanStatusInput
): Promise<ApiResponse<SupplyActionPlan>> {
  const res = await api.post(
    `/api/supply_action_plans/${id}/status`,
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplyActionExecutions(
  params: PageParams &
    TimeRangeParams & {
      supply_action_plan_id?: number
      supply_decision_id?: number
      execution_status?: SupplyActionExecutionStatus
      track?: string
      supplier_id?: number
      channel_id?: number
      supply_capacity_id?: number
    } = {}
): Promise<ApiResponse<PageData<SupplyActionExecution>>> {
  const res = await api.get('/api/supply_action_executions/', { params })
  return res.data
}

export async function recordSupplyActionExecution(
  input: SupplyActionExecutionRecordInput
): Promise<ApiResponse<SupplyActionExecution>> {
  const res = await api.post(
    '/api/supply_action_executions/record',
    input,
    actionConfig()
  )
  return res.data
}

export async function refreshSupplyActionExecutionUsage(
  input: SupplyActionExecutionUsageRefreshInput = {}
): Promise<ApiResponse<SupplyActionExecution[]>> {
  const res = await api.post(
    '/api/supply_action_executions/refresh_usage',
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplyRoutingPolicies(
  params: PageParams &
    TimeRangeParams & {
      supply_action_execution_id?: number
      supply_action_plan_id?: number
      supply_decision_id?: number
      status?: SupplyRoutingPolicyStatus | 'all'
      track?: string
      supplier_id?: number
      channel_id?: number
      supply_capacity_id?: number
    } = {}
): Promise<ApiResponse<PageData<SupplyRoutingPolicy>>> {
  const res = await api.get('/api/supply_routing_policies/', { params })
  return res.data
}

export async function activateSupplyRoutingPolicy(
  input: ActivateSupplyRoutingPolicyInput
): Promise<ApiResponse<SupplyRoutingPolicy>> {
  const res = await api.post(
    '/api/supply_routing_policies/activate',
    input,
    actionConfig()
  )
  return res.data
}

export async function disableSupplyRoutingPolicy(
  id: number,
  input: DisableSupplyRoutingPolicyInput = {}
): Promise<ApiResponse<SupplyRoutingPolicy>> {
  const res = await api.post(
    `/api/supply_routing_policies/${id}/disable`,
    input,
    actionConfig()
  )
  return res.data
}

export async function getSlaContracts(
  params: PageParams &
    TimeRangeParams & {
      model_name?: string
      provider_family?: string
      status?: SlaContractStatus
    } = {}
): Promise<ApiResponse<PageData<SlaContract>>> {
  const res = await api.get('/api/sla_contracts', { params })
  return res.data
}

export async function importSlaContract(
  input: SlaContractImportInput
): Promise<ApiResponse<SlaContract>> {
  const res = await api.post('/api/sla_contracts/import', input, actionConfig())
  return res.data
}

export async function getSlaProbePlans(
  params: PageParams &
    TimeRangeParams & {
      contract_id?: number
      supplier_id?: number
      channel_id?: number
      model_name?: string
      sla_tier?: string
      probe_type?: SlaProbeType
      route_mode?: SlaProbeRouteMode
    } = {}
): Promise<ApiResponse<PageData<SlaProbePlan>>> {
  const res = await api.get('/api/sla_probe_plans', { params })
  return res.data
}

export async function generateSlaProbePlan(
  input: SlaProbePlanGenerateInput
): Promise<ApiResponse<SlaProbePlan>> {
  const res = await api.post(
    '/api/sla_probe_plans/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function getSlaProbeRuns(
  params: PageParams &
    TimeRangeParams & {
      plan_id?: number
      contract_id?: number
      supplier_id?: number
      channel_id?: number
      model_name?: string
      sla_tier?: string
      status?: SlaProbeRunStatus
      route_mode?: SlaProbeRouteMode
    } = {}
): Promise<ApiResponse<PageData<SlaProbeRun>>> {
  const res = await api.get('/api/sla_probe_runs', { params })
  return res.data
}

export async function recordSlaProbeRun(
  input: SlaProbeRunRecordInput
): Promise<ApiResponse<SlaProbeRun>> {
  const res = await api.post(
    '/api/sla_probe_runs/record',
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplierScorecards(
  params: PageParams &
    TimeRangeParams & {
      supplier_id?: number
      grade?: SupplierScorecardGrade
    } = {}
): Promise<ApiResponse<PageData<SupplierScorecard>>> {
  const res = await api.get('/api/supplier_scorecards/', { params })
  return res.data
}

export async function generateSupplierScorecards(
  input: GenerateSupplierScorecardInput
): Promise<ApiResponse<SupplierScorecard[]>> {
  const res = await api.post(
    '/api/supplier_scorecards/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplierEvaluations(
  params: PageParams &
    TimeRangeParams & {
      supplier_id?: number
      evaluation_type?: string
      status?: SupplierEvaluationStatus
      recommendation?: SupplierEvaluationRecommendation
      grade?: SupplierScorecardGrade
    } = {}
): Promise<ApiResponse<PageData<SupplierEvaluation>>> {
  const res = await api.get('/api/supplier_evaluations/', { params })
  return res.data
}

export async function generateSupplierEvaluations(
  input: GenerateSupplierEvaluationInput
): Promise<ApiResponse<SupplierEvaluation[]>> {
  const res = await api.post(
    '/api/supplier_evaluations/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function approveSupplierEvaluation(
  id: number,
  input: ReviewSupplierEvaluationInput = {
    review_note: 'approved from dashboard',
  }
): Promise<ApiResponse<SupplierEvaluation>> {
  const res = await api.post(
    `/api/supplier_evaluations/${id}/approve`,
    input,
    actionConfig()
  )
  return res.data
}

export async function rejectSupplierEvaluation(
  id: number,
  input: ReviewSupplierEvaluationInput = {
    review_note: 'rejected from dashboard',
  }
): Promise<ApiResponse<SupplierEvaluation>> {
  const res = await api.post(
    `/api/supplier_evaluations/${id}/reject`,
    input,
    actionConfig()
  )
  return res.data
}

export async function applySupplierEvaluation(
  id: number,
  input: ApplySupplierEvaluationInput = {
    operator_note: 'applied approved supplier evaluation from dashboard',
  }
): Promise<ApiResponse<SupplierEvaluation>> {
  const res = await api.post(
    `/api/supplier_evaluations/${id}/apply`,
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplierPostureRecommendations(
  params: PageParams &
    TimeRangeParams & {
      supplier_id?: number
      status?: SupplierPostureRecommendationStatus
      recommended_action?: SupplierPostureRecommendationAction
      grade?: SupplierScorecardGrade
    } = {}
): Promise<ApiResponse<PageData<SupplierPostureRecommendation>>> {
  const res = await api.get('/api/supplier_posture_recommendations/', {
    params,
  })
  return res.data
}

export async function generateSupplierPostureRecommendations(
  input: GenerateSupplierPostureRecommendationInput
): Promise<ApiResponse<SupplierPostureRecommendation[]>> {
  const res = await api.post(
    '/api/supplier_posture_recommendations/generate',
    input,
    actionConfig()
  )
  return res.data
}

export async function approveSupplierPostureRecommendation(
  id: number,
  input: ReviewSupplierPostureRecommendationInput = {
    review_note: 'approved supplier posture recommendation from dashboard',
  }
): Promise<ApiResponse<SupplierPostureRecommendation>> {
  const res = await api.post(
    `/api/supplier_posture_recommendations/${id}/approve`,
    input,
    actionConfig()
  )
  return res.data
}

export async function rejectSupplierPostureRecommendation(
  id: number,
  input: ReviewSupplierPostureRecommendationInput = {
    review_note: 'rejected supplier posture recommendation from dashboard',
  }
): Promise<ApiResponse<SupplierPostureRecommendation>> {
  const res = await api.post(
    `/api/supplier_posture_recommendations/${id}/reject`,
    input,
    actionConfig()
  )
  return res.data
}

export async function applySupplierPostureRecommendation(
  id: number,
  input: ApplySupplierPostureRecommendationInput = {
    operator_note: 'applied supplier posture recommendation from dashboard',
  }
): Promise<ApiResponse<SupplierPostureRecommendation>> {
  const res = await api.post(
    `/api/supplier_posture_recommendations/${id}/apply`,
    input,
    actionConfig()
  )
  return res.data
}

export async function getSupplierRoutePreferences(
  params: PageParams &
    TimeRangeParams & {
      supplier_id?: number
      source_posture_recommendation_id?: number
      status?: SupplierRoutePreferenceStatus
    } = {}
): Promise<ApiResponse<PageData<SupplierRoutePreference>>> {
  const res = await api.get('/api/supplier_route_preferences/', { params })
  return res.data
}

export async function activateSupplierRoutePreference(
  input: SupplierRoutePreferenceActivateInput
): Promise<ApiResponse<SupplierRoutePreference>> {
  const res = await api.post(
    '/api/supplier_route_preferences/activate',
    input,
    actionConfig()
  )
  return res.data
}

export async function disableSupplierRoutePreference(
  supplierId: number,
  input: SupplierRoutePreferenceDisableInput = {
    operator_note: 'disabled supplier route preference from dashboard',
  }
): Promise<ApiResponse<SupplierRoutePreference>> {
  const res = await api.post(
    `/api/supplier_route_preferences/${supplierId}/disable`,
    input,
    actionConfig()
  )
  return res.data
}

export async function createSupplierAgreement(
  input: SupplierAgreementInput
): Promise<ApiResponse<SupplierAgreement>> {
  const res = await api.post('/api/supplier_agreements', input, actionConfig())
  return res.data
}

export async function updateSupplierAgreement(
  input: SupplierAgreementInput
): Promise<ApiResponse<SupplierAgreement>> {
  const res = await api.put('/api/supplier_agreements', input, actionConfig())
  return res.data
}

export async function deleteSupplierAgreement(
  id: number
): Promise<ApiResponse<null>> {
  const res = await api.delete(`/api/supplier_agreements/${id}`, actionConfig())
  return res.data
}

export async function getUsageLedgers(
  params: PageParams & TimeRangeParams = {}
): Promise<ApiResponse<PageData<UsageLedger>>> {
  const res = await api.get('/api/usage_ledgers', { params })
  return res.data
}

export async function getMarginSummary(
  params: TimeRangeParams & { group_by: MarginGroupBy }
): Promise<ApiResponse<MarginSummaryRow[]>> {
  const res = await api.get('/api/reports/margin_summary', { params })
  return res.data
}

export async function getQualitySummary(
  params: TimeRangeParams & { group_by: QualityGroupBy }
): Promise<ApiResponse<QualitySummaryRow[]>> {
  const res = await api.get('/api/reports/quality_summary', { params })
  return res.data
}

export async function getSettlementStatements(
  params: PageParams & TimeRangeParams = {}
): Promise<ApiResponse<PageData<SettlementStatement>>> {
  const res = await api.get('/api/settlement_statements', { params })
  return res.data
}

export async function generateSettlementStatement(
  input: GenerateSettlementInput
): Promise<ApiResponse<SettlementStatement>> {
  const res = await api.post(
    '/api/settlement_statements/generate',
    input,
    actionConfig()
  )
  return res.data
}

export function getSettlementItemsCsvUrl(statementId: number): string {
  return `/api/settlement_statements/${statementId}/items.csv`
}
