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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Download, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react'
import {
  useMemo,
  useState,
  type Dispatch,
  type ReactNode,
  type SetStateAction,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  formatNumber,
  formatLogQuota,
  formatTimestampForInput,
  formatTimestampToDate,
  formatTokens,
  parseTimestampFromInput,
} from '@/lib/format'

import {
  acknowledgeOperatingInsight,
  activateSupplierRoutePreference,
  activateSupplyRoutingPolicy,
  approvePricingRecommendation,
  approveSupplierEvaluation,
  approveSupplierPostureRecommendation,
  approveSupplyDecision,
  applySupplierEvaluation,
  applySupplierPostureRecommendation,
  createSupplier,
  createSupplierAgreement,
  deleteSupplierAgreement,
  disableSupplierRoutePreference,
  disableSupplyRoutingPolicy,
  dismissOperatingInsight,
  generateOperatingInsights,
  generatePricingRecommendations,
  generateSlaProbePlan,
  generateSettlementStatement,
  generateSupplyExpansionOpportunities,
  generateSupplyActionPlans,
  generateSupplyDecisions,
  generateSupplierEvaluations,
  generateSupplierPostureRecommendations,
  generateSupplierScorecards,
  generateTrafficForecasts,
  generateTrafficProfiles,
  getMarginSummary,
  getOperatingInsights,
  getPricingRecommendations,
  getQualitySummary,
  getSettlementItemsCsvUrl,
  getSettlementStatements,
  getSlaContracts,
  getSlaProbePlans,
  getSlaProbeRuns,
  getSupplyCapacities,
  getSupplyCostProfiles,
  getSupplyPrepaidLots,
  getSupplyExpansionOpportunities,
  getSupplyActionExecutions,
  getSupplyActionPlans,
  getSupplyDecisions,
  getSupplyRoutingPolicies,
  getSupplierEvaluations,
  getSupplierPostureRecommendations,
  getSupplierRoutePreferences,
  getSupplierScorecards,
  getSupplierAgreements,
  getSuppliers,
  getTrafficForecasts,
  getTrafficProfiles,
  getUsageLedgers,
  importSlaContract,
  rejectPricingRecommendation,
  rejectSupplierEvaluation,
  rejectSupplierPostureRecommendation,
  rejectSupplyDecision,
  recordSlaProbeRun,
  recordSupplyCostProfile,
  recordSupplyPrepaidLot,
  recordSupplyActionExecution,
  refreshSupplyPrepaidLotUsage,
  refreshSupplyActionExecutionUsage,
  updateSupplyActionPlanStatus,
  updateSupplier,
  updateSupplierAgreement,
} from './api'
import { ControlTower } from './components/control-tower'
import type {
  MarginGroupBy,
  MarginSummaryRow,
  OperatingInsight,
  OperatingInsightCategory,
  OperatingInsightSeverity,
  OperatingInsightStatus,
  PricingRecommendation,
  PricingRecommendationAction,
  PricingRecommendationStatus,
  QualityGroupBy,
  QualitySummaryRow,
  SettlementStatement,
  SettlementSubjectType,
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
  SupplyCostProfile,
  SupplyCostProfileRecordInput,
  SupplyCostProfileSourceType,
  SupplyPrepaidLot,
  SupplyPrepaidLotRecordInput,
  SupplyPrepaidLotSourceType,
  SupplyActionExecution,
  SupplyActionExecutionRecordInput,
  SupplyActionExecutionStatus,
  SupplyPrepaidLotUsageRefreshInput,
  SupplyActionPlan,
  SupplyActionPlanStatus,
  SupplyDecision,
  SupplyDecisionStatus,
  SupplyDecisionTrack,
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
  SupplierScorecard,
  SupplierScorecardGrade,
  Supplier,
  SupplierAgreement,
  SupplierAgreementInput,
  SupplierInput,
  SupplierType,
  TrafficForecast,
  TrafficProfile,
  UsageLedger,
} from './types'

type TabValue =
  | 'control-tower'
  | 'overview'
  | 'suppliers'
  | 'quality'
  | 'capacity'
  | 'cost-profiles'
  | 'prepaid-lots'
  | 'scorecards'
  | 'evaluations'
  | 'posture'
  | 'profiles'
  | 'forecasts'
  | 'pricing'
  | 'insights'
  | 'sla-evidence'
  | 'decisions'
  | 'opportunities'
  | 'actions'
  | 'executions'
  | 'routing'
  | 'ledger'
  | 'settlements'
type Translator = (key: string) => string
type SupplierEvaluationSlaGateSummary = {
  run_key?: unknown
  hard_gate_passed?: unknown
  artifact_sha256?: unknown
  runtime_ref?: unknown
}
type SupplierFormState = {
  id?: number
  name: string
  type: SupplierType
  status: string
  notes: string
}
type SupplierAgreementFormState = {
  id?: number
  supplier_id: string
  model_name: string
  effective_from: string
  effective_to: string
  use_price: boolean
  cost_model_ratio: string
  cost_completion_ratio: string
  cost_cache_ratio: string
  cost_cache_creation_ratio: string
  cost_model_price: string
  priority: string
  status: string
  notes: string
}
type SupplierRoutePreferenceFormState = {
  supplier_id: string
  weight_percent: string
  reason: string
  effective_from: string
  effective_to: string
  operator_note: string
}
type SupplyRoutingPolicyActivateFormState = {
  execution: SupplyActionExecution | null
  traffic_percent: string
  operator_note: string
}
type SupplyCostProfileFormState = {
  supplier_id: string
  supply_node: string
  model_name: string
  period_start: string
  period_end: string
  capacity_tokens: string
  fixed_cost_quota: string
  variable_unit_cost_quota: string
  source_type: SupplyCostProfileSourceType
  source_ref: string
  observed_at: string
  notes: string
}
type SupplyPrepaidLotFormState = {
  supplier_id: string
  channel_id: string
  supply_node: string
  model_name: string
  period_start: string
  period_end: string
  purchased_tokens: string
  unit_cost_quota: string
  source_type: SupplyPrepaidLotSourceType
  source_ref: string
  observed_at: string
  external_ref: string
  notes: string
}
type ScorecardGradeFilter = 'all' | SupplierScorecardGrade
type EvaluationStatusFilter = 'all' | SupplierEvaluationStatus
type EvaluationRecommendationFilter = 'all' | SupplierEvaluationRecommendation
type PostureStatusFilter = 'all' | SupplierPostureRecommendationStatus
type PostureActionFilter = 'all' | SupplierPostureRecommendationAction
type PricingRecommendationStatusFilter = 'all' | PricingRecommendationStatus
type PricingRecommendationActionFilter = 'all' | PricingRecommendationAction
type OperatingInsightStatusFilter = 'all' | OperatingInsightStatus
type OperatingInsightSeverityFilter = 'all' | OperatingInsightSeverity
type OperatingInsightCategoryFilter = 'all' | OperatingInsightCategory
type TrackFilter = 'all' | SupplyDecisionTrack
type OpportunityTypeFilter = 'all' | SupplyExpansionOpportunityType
type OpportunityPriorityFilter = 'all' | SupplyExpansionOpportunityPriority
type ActionPlanStatusFilter = 'all' | SupplyActionPlanStatus
type ExecutionStatusFilter = 'all' | SupplyActionExecutionStatus
type RoutingPolicyStatusFilter = 'all' | SupplyRoutingPolicyStatus
type SlaContractStatusFilter = 'all' | SlaContractStatus
type SlaProbeTypeFilter = 'all' | SlaProbeType
type SlaProbeRouteModeFilter = 'all' | SlaProbeRouteMode
type SlaProbeRunStatusFilter = 'all' | SlaProbeRunStatus
type ActionPlanStatusFormState = {
  plan: SupplyActionPlan | null
  status: SupplyActionPlanStatus
  operator_note: string
}
type SupplyActionExecutionRecordFormState = {
  supply_action_plan_id: string
  supplier_id: string
  channel_id: string
  supply_capacity_id: string
  actual_capacity_tokens: string
  unit_cost_quota: string
  effective_from: string
  effective_to: string
  external_ref: string
  operator_note: string
}
type SlaContractImportFormState = {
  contract_key: string
  model_name: string
  model_aliases: string
  provider_family: string
  source_name: string
  source_ref: string
  source_sha256: string
  version: string
  status: SlaContractStatus
  effective_from: string
  effective_to: string
  measurement_profile_json: string
  hard_gate_json: string
  soft_gate_json: string
}
type SlaProbePlanGenerateFormState = {
  contract_id: string
  contract_key: string
  supplier_id: string
  channel_id: string
  model_name: string
  sla_tier: string
  probe_type: SlaProbeType
  route_mode: SlaProbeRouteMode
  prompt_suite_key: string
  tokenizer_ref: string
  sample_size: string
  repeat_count: string
  input_profile_json: string
  output_profile_json: string
  concurrency_profile_json: string
  rate_profile_json: string
  stream_profile_json: string
  error_profile_json: string
  availability_profile_json: string
  cache_profile: string
  schedule_interval_seconds: string
  jitter_seconds: string
  max_probe_quota: string
}
type SlaProbeRunRecordFormState = {
  run_key: string
  plan_id: string
  status: SlaProbeRunStatus
  started_at: string
  ended_at: string
  runner_version: string
  git_commit: string
  runtime_ref: string
  endpoint: string
  summary_json: string
  hard_gate_passed: boolean
  soft_gate_warnings: string
  failure_reasons: string
  artifact_uri: string
  artifact_sha256: string
}

const QUERY_PAGE_SIZE = 20
const EMPTY_SUPPLIERS: Supplier[] = []
const EMPTY_AGREEMENTS: SupplierAgreement[] = []
const EMPTY_LEDGERS: UsageLedger[] = []
const EMPTY_MARGIN_ROWS: MarginSummaryRow[] = []
const EMPTY_QUALITY_ROWS: QualitySummaryRow[] = []
const EMPTY_CAPACITIES: SupplyCapacity[] = []
const EMPTY_COST_PROFILES: SupplyCostProfile[] = []
const EMPTY_PREPAID_LOTS: SupplyPrepaidLot[] = []
const EMPTY_SCORECARDS: SupplierScorecard[] = []
const EMPTY_EVALUATIONS: SupplierEvaluation[] = []
const EMPTY_POSTURE_RECOMMENDATIONS: SupplierPostureRecommendation[] = []
const EMPTY_ROUTE_PREFERENCES: SupplierRoutePreference[] = []
const EMPTY_PROFILES: TrafficProfile[] = []
const EMPTY_FORECASTS: TrafficForecast[] = []
const EMPTY_PRICING_RECOMMENDATIONS: PricingRecommendation[] = []
const EMPTY_OPERATING_INSIGHTS: OperatingInsight[] = []
const EMPTY_DECISIONS: SupplyDecision[] = []
const EMPTY_OPPORTUNITIES: SupplyExpansionOpportunity[] = []
const EMPTY_ACTION_PLANS: SupplyActionPlan[] = []
const EMPTY_ACTION_EXECUTIONS: SupplyActionExecution[] = []
const EMPTY_ROUTING_POLICIES: SupplyRoutingPolicy[] = []
const EMPTY_SLA_CONTRACTS: SlaContract[] = []
const EMPTY_SLA_PROBE_PLANS: SlaProbePlan[] = []
const EMPTY_SLA_PROBE_RUNS: SlaProbeRun[] = []
const EMPTY_STATEMENTS: SettlementStatement[] = []
const SKELETON_ROW_KEYS = ['row-a', 'row-b', 'row-c', 'row-d', 'row-e', 'row-f']
const SKELETON_COLUMN_KEYS = [
  'col-a',
  'col-b',
  'col-c',
  'col-d',
  'col-e',
  'col-f',
  'col-g',
  'col-h',
  'col-i',
  'col-j',
]
const DEFAULT_SUPPLIER_FORM: SupplierFormState = {
  name: '',
  type: 'third_party',
  status: '1',
  notes: '',
}
const DEFAULT_AGREEMENT_FORM: SupplierAgreementFormState = {
  supplier_id: '',
  model_name: '',
  effective_from: '',
  effective_to: '',
  use_price: false,
  cost_model_ratio: '1',
  cost_completion_ratio: '1',
  cost_cache_ratio: '0.1',
  cost_cache_creation_ratio: '1',
  cost_model_price: '0',
  priority: '0',
  status: '1',
  notes: '',
}
const DEFAULT_ROUTE_PREFERENCE_FORM: SupplierRoutePreferenceFormState = {
  supplier_id: '',
  weight_percent: '25',
  reason: '',
  effective_from: '',
  effective_to: '',
  operator_note: '',
}
const DEFAULT_ROUTING_POLICY_ACTIVATE_FORM: SupplyRoutingPolicyActivateFormState =
  {
    execution: null,
    traffic_percent: '100',
    operator_note: 'activated from dashboard',
  }
const DEFAULT_COST_PROFILE_FORM: SupplyCostProfileFormState = {
  supplier_id: '',
  supply_node: '',
  model_name: '',
  period_start: '',
  period_end: '',
  capacity_tokens: '1000',
  fixed_cost_quota: '0',
  variable_unit_cost_quota: '0',
  source_type: 'accounting',
  source_ref: '',
  observed_at: '',
  notes: '',
}
const DEFAULT_PREPAID_LOT_FORM: SupplyPrepaidLotFormState = {
  supplier_id: '',
  channel_id: '',
  supply_node: '',
  model_name: '',
  period_start: '',
  period_end: '',
  purchased_tokens: '1000',
  unit_cost_quota: '0',
  source_type: 'accounting',
  source_ref: '',
  observed_at: '',
  external_ref: '',
  notes: '',
}
const DEFAULT_ACTION_PLAN_STATUS_FORM: ActionPlanStatusFormState = {
  plan: null,
  status: 'in_progress',
  operator_note: '',
}
const DEFAULT_ACTION_EXECUTION_RECORD_FORM: SupplyActionExecutionRecordFormState =
  {
    supply_action_plan_id: '',
    supplier_id: '',
    channel_id: '',
    supply_capacity_id: '',
    actual_capacity_tokens: '0',
    unit_cost_quota: '0',
    effective_from: '',
    effective_to: '',
    external_ref: '',
    operator_note: '',
  }
const DEFAULT_SLA_CONTRACT_IMPORT_FORM: SlaContractImportFormState = {
  contract_key: '',
  model_name: '',
  model_aliases: '',
  provider_family: '',
  source_name: '',
  source_ref: '',
  source_sha256: '',
  version: '',
  status: 'draft',
  effective_from: '',
  effective_to: '',
  measurement_profile_json:
    '{\n  "input_profile": {},\n  "output_profile": {},\n  "cache_profile": "cold_no_cache"\n}',
  hard_gate_json: '{}',
  soft_gate_json: '{}',
}
const DEFAULT_SLA_PROBE_PLAN_FORM: SlaProbePlanGenerateFormState = {
  contract_id: '',
  contract_key: '',
  supplier_id: '',
  channel_id: '',
  model_name: '',
  sla_tier: 'default',
  probe_type: 'admission',
  route_mode: 'through_token_router',
  prompt_suite_key: 'default',
  tokenizer_ref: 'contract',
  sample_size: '1',
  repeat_count: '1',
  input_profile_json: '',
  output_profile_json: '',
  concurrency_profile_json: '',
  rate_profile_json: '',
  stream_profile_json: '',
  error_profile_json: '',
  availability_profile_json: '',
  cache_profile: 'cold_no_cache',
  schedule_interval_seconds: '0',
  jitter_seconds: '0',
  max_probe_quota: '0',
}
const DEFAULT_SLA_PROBE_RUN_FORM: SlaProbeRunRecordFormState = {
  run_key: '',
  plan_id: '',
  status: 'passed',
  started_at: '',
  ended_at: '',
  runner_version: 'token-router-sla/manual',
  git_commit: '',
  runtime_ref: '',
  endpoint: '',
  summary_json: '{}',
  hard_gate_passed: true,
  soft_gate_warnings: '',
  failure_reasons: '',
  artifact_uri: '',
  artifact_sha256: '',
}

function formatRate(value: number | null | undefined) {
  if (value == null || Number.isNaN(value)) return '-'
  return Intl.NumberFormat(undefined, {
    style: 'percent',
    maximumFractionDigits: 1,
  }).format(value)
}

function formatSignedQuota(value: number) {
  if (value === 0) return formatLogQuota(0)
  return value > 0 ? `+${formatLogQuota(value)}` : formatLogQuota(value)
}

function groupLabel(row: MarginSummaryRow, groupBy: MarginGroupBy) {
  switch (groupBy) {
    case 'supplier':
      return row.supplier_id ? `#${row.supplier_id}` : row.group_key
    case 'channel':
      return row.channel_id ? `#${row.channel_id}` : row.group_key
    case 'user':
      return row.user_id ? `#${row.user_id}` : row.group_key
    case 'model':
      return row.model_name || row.group_key
    case 'day':
      return row.bucket_start ? formatTimestampToDate(row.bucket_start) : '-'
  }
}

function qualityGroupLabel(row: QualitySummaryRow, groupBy: QualityGroupBy) {
  switch (groupBy) {
    case 'supplier':
      return row.supplier_id ? `#${row.supplier_id}` : row.group_key
    case 'channel':
      return row.channel_id ? `#${row.channel_id}` : row.group_key
    case 'user':
      return row.user_id ? `#${row.user_id}` : row.group_key
    case 'model':
      return row.model_name || row.group_key || '-'
    case 'sla_tier':
      return row.sla_tier || tEmptyGroup()
    case 'supply_node':
      return row.supply_node || tEmptyGroup()
    case 'day':
      return row.bucket_start ? formatTimestampToDate(row.bucket_start) : '-'
  }
}

function tEmptyGroup() {
  return '-'
}

function formatLatency(value: number | null | undefined) {
  if (value == null || Number.isNaN(value)) return '-'
  return `${formatNumber(Math.round(value))} ms`
}

function formatUnitCost(value: number | null | undefined) {
  if (value == null || Number.isNaN(value)) return '-'
  return formatNumber(value)
}

function supplierTypeLabel(type: string, t: Translator) {
  switch (type) {
    case 'self_operated':
      return t('Self-operated supply')
    case 'self_hosted':
      return t('Self-hosted supply')
    default:
      return t('Third-party supply')
  }
}

function decisionTrackLabel(
  track: SupplyDecisionTrack | string,
  t: Translator
) {
  switch (track) {
    case 'self_operated':
      return t('Self-operated')
    case 'self_hosted':
      return t('Self-hosted')
    default:
      return t('Third-party')
  }
}

function decisionTypeLabel(type: string, t: Translator) {
  switch (type) {
    case 'third_party_recruit':
      return t('Recruit third-party supply')
    case 'self_operated_purchase':
      return t('Evaluate self-operated purchase')
    case 'self_hosted_evaluate':
      return t('Evaluate self-hosted capacity')
    case 'third_party_probe':
      return t('Keep third-party observation')
    default:
      return type || '-'
  }
}

function actionTypeLabel(type: string, t: Translator) {
  switch (type) {
    case 'recruit_third_party':
      return t('Recruit third-party supplier')
    case 'prepare_self_operated_purchase':
      return t('Prepare self-operated purchase')
    case 'evaluate_self_hosted_capacity':
      return t('Evaluate self-hosted capacity')
    case 'keep_third_party_observation':
      return t('Keep third-party observation')
    default:
      return type || '-'
  }
}

function actionPlanStatusLabel(status: string, t: Translator) {
  switch (status) {
    case 'in_progress':
      return t('In progress')
    case 'completed':
      return t('Completed')
    case 'cancelled':
      return t('Cancelled')
    default:
      return t('Planned')
  }
}

function actionPlanStatusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'completed') return 'secondary'
  if (status === 'cancelled') return 'destructive'
  return 'outline'
}

function nextActionPlanStatuses(status: string): SupplyActionPlanStatus[] {
  if (status === 'planned') {
    return ['in_progress', 'completed', 'cancelled']
  }
  if (status === 'in_progress') {
    return ['completed', 'cancelled']
  }
  return []
}

function executionStatusLabel(status: string, t: Translator) {
  switch (status) {
    case 'recorded':
      return t('Recorded')
    default:
      return status || '-'
  }
}

function executionStatusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'recorded') return 'secondary'
  return 'outline'
}

function routingPolicyStatusLabel(status: string, t: Translator) {
  switch (status) {
    case 'active':
      return t('Active')
    case 'disabled':
      return t('Disabled')
    default:
      return status || '-'
  }
}

function routingPolicyStatusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'active') return 'secondary'
  if (status === 'disabled') return 'outline'
  return 'outline'
}

function routingPolicyTrafficPercent(policy: SupplyRoutingPolicy) {
  const percent = Number(policy.traffic_percent)
  if (!Number.isFinite(percent) || percent <= 0) return 100
  return Math.min(percent, 100)
}

function slaContractStatusLabel(status: string, t: Translator) {
  switch (status) {
    case 'active':
      return t('Active')
    case 'retired':
      return t('Retired')
    default:
      return t('Draft')
  }
}

function slaContractStatusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'active') return 'secondary'
  if (status === 'retired') return 'outline'
  return 'outline'
}

function slaProbeTypeLabel(probeType: string, t: Translator) {
  switch (probeType) {
    case 'runtime_light':
      return t('Runtime Light')
    case 'runtime_deep':
      return t('Runtime Deep')
    case 'incident_recheck':
      return t('Incident Recheck')
    default:
      return t('Admission')
  }
}

function slaProbeRouteModeLabel(routeMode: string, t: Translator) {
  switch (routeMode) {
    case 'direct_upstream':
      return t('Direct Upstream')
    case 'through_token_router':
      return t('Through Token Router')
    default:
      return routeMode || '-'
  }
}

function slaProbeRunStatusLabel(status: string, t: Translator) {
  switch (status) {
    case 'passed':
      return t('Passed')
    case 'failed':
      return t('Failed')
    case 'invalid':
      return t('Invalid')
    case 'cancelled':
      return t('Cancelled')
    default:
      return t('Running')
  }
}

function slaProbeRunStatusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'passed') return 'secondary'
  if (status === 'failed' || status === 'invalid') return 'destructive'
  return 'outline'
}

function decisionStatusLabel(status: string, t: Translator) {
  switch (status) {
    case 'approved':
      return t('Approved')
    case 'rejected':
      return t('Rejected')
    default:
      return t('Draft')
  }
}

function decisionStatusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'approved') return 'secondary'
  if (status === 'rejected') return 'destructive'
  return 'outline'
}

function opportunityTypeLabel(opportunityType: string, t: Translator) {
  switch (opportunityType) {
    case 'third_party_gap':
      return t('Third-party Gap')
    case 'third_party_probe':
      return t('Third-party Probe')
    case 'self_operated_bulk':
      return t('Self-operated Bulk')
    case 'self_hosted_cache':
      return t('Self-hosted Cache')
    default:
      return opportunityType || '-'
  }
}

function opportunityPriorityLabel(priority: string, t: Translator) {
  switch (priority) {
    case 'action':
      return t('Action')
    case 'watch':
      return t('Watch')
    default:
      return t('Info')
  }
}

function opportunityPriorityVariant(
  priority: string
): 'secondary' | 'destructive' | 'outline' {
  if (priority === 'action') return 'destructive'
  if (priority === 'watch') return 'outline'
  return 'secondary'
}

function opportunityClusterLabel(clusterKey: string, t: Translator) {
  switch (clusterKey) {
    case 'capacity_gap':
      return t('Capacity Gap')
    case 'high_cache_stable':
      return t('High Cache Stable')
    case 'positive_margin':
      return t('Positive Margin')
    case 'observe':
      return t('Observe')
    default:
      return clusterKey || '-'
  }
}

function costProfileSourceLabel(sourceType: string, t: Translator) {
  switch (sourceType) {
    case 'accounting':
      return t('Accounting')
    case 'external':
      return t('External')
    default:
      return t('Manual')
  }
}

function pricingRecommendationActionLabel(action: string, t: Translator) {
  switch (action) {
    case 'raise_price':
      return t('Raise Price')
    case 'share_savings':
      return t('Share Savings')
    case 'keep_price':
      return t('Keep Price')
    default:
      return action || '-'
  }
}

function pricingRecommendationActionVariant(
  action: string
): 'secondary' | 'destructive' | 'outline' {
  if (action === 'share_savings') return 'secondary'
  if (action === 'raise_price') return 'destructive'
  return 'outline'
}

function operatingInsightStatusLabel(status: string, t: Translator) {
  switch (status) {
    case 'acknowledged':
      return t('Acknowledged')
    case 'dismissed':
      return t('Dismissed')
    default:
      return t('Draft')
  }
}

function operatingInsightStatusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'acknowledged') return 'secondary'
  if (status === 'dismissed') return 'destructive'
  return 'outline'
}

function operatingInsightSeverityLabel(severity: string, t: Translator) {
  switch (severity) {
    case 'action':
      return t('Action')
    case 'watch':
      return t('Watch')
    default:
      return t('Info')
  }
}

function operatingInsightSeverityVariant(
  severity: string
): 'secondary' | 'destructive' | 'outline' {
  if (severity === 'action') return 'destructive'
  if (severity === 'watch') return 'outline'
  return 'secondary'
}

function operatingInsightCategoryLabel(category: string, t: Translator) {
  switch (category) {
    case 'cache_efficiency':
      return t('Cache Efficiency')
    case 'capacity_risk':
      return t('Capacity Risk')
    case 'pricing_risk':
      return t('Pricing Risk')
    case 'quality_watch':
      return t('Quality Watch')
    case 'steady_state':
      return t('Steady State')
    default:
      return category || '-'
  }
}

function scorecardGradeVariant(
  grade: string
): 'secondary' | 'destructive' | 'outline' {
  if (grade === 'A') return 'secondary'
  if (grade === 'D') return 'destructive'
  return 'outline'
}

function supplierEvaluationRecommendationLabel(
  recommendation: string,
  t: Translator
) {
  switch (recommendation) {
    case 'admit':
      return t('Admit')
    case 'observe':
      return t('Observe')
    case 'reject':
      return t('Reject')
    default:
      return recommendation || '-'
  }
}

function supplierEvaluationRecommendationVariant(
  recommendation: string
): 'secondary' | 'destructive' | 'outline' {
  if (recommendation === 'admit') return 'secondary'
  if (recommendation === 'reject') return 'destructive'
  return 'outline'
}

function supplierPostureActionLabel(action: string, t: Translator) {
  switch (action) {
    case 'boost':
      return t('Boost')
    case 'disable':
      return t('Disable')
    case 'downgrade':
      return t('Downgrade')
    case 'observe':
      return t('Observe')
    default:
      return action || '-'
  }
}

function supplierPostureActionVariant(
  action: string
): 'secondary' | 'destructive' | 'outline' {
  if (action === 'disable') return 'destructive'
  if (action === 'boost' || action === 'observe') return 'secondary'
  return 'outline'
}

function supplierPostureStatusLabel(status: string, t: Translator) {
  switch (status) {
    case 'approved':
      return t('Approved')
    case 'rejected':
      return t('Rejected')
    case 'applied':
      return t('Applied')
    default:
      return t('Draft')
  }
}

function supplierPostureStatusVariant(
  status: string
): 'secondary' | 'destructive' | 'outline' {
  if (status === 'approved' || status === 'applied') return 'secondary'
  if (status === 'rejected') return 'destructive'
  return 'outline'
}

function parseSupplierEvaluationSlaGateSummary(
  value: string | null | undefined
): SupplierEvaluationSlaGateSummary | null {
  const trimmed = value?.trim() ?? ''
  if (!trimmed) return null
  try {
    const parsed = JSON.parse(trimmed)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as SupplierEvaluationSlaGateSummary
    }
  } catch {
    return null
  }
  return null
}

function summaryTextValue(value: unknown) {
  return typeof value === 'string' && value.trim() ? value.trim() : ''
}

function shortHash(value: string) {
  return value.length > 16 ? `${value.slice(0, 16)}...` : value
}

function renderSupplierEvaluationSlaEvidence(
  evaluation: SupplierEvaluation,
  t: Translator
): ReactNode {
  const summary = parseSupplierEvaluationSlaGateSummary(
    evaluation.sla_gate_summary_json
  )
  const runKey = summaryTextValue(summary?.run_key)
  const artifactSHA256 = summaryTextValue(summary?.artifact_sha256)
  const runtimeRef = summaryTextValue(summary?.runtime_ref)
  const hardGatePassed =
    typeof summary?.hard_gate_passed === 'boolean'
      ? summary.hard_gate_passed
      : null

  if (evaluation.sla_probe_run_id <= 0) {
    return (
      <span className='flex min-w-44 flex-col gap-1'>
        <Badge variant='outline'>{t('No SLA Evidence')}</Badge>
        <span className='text-muted-foreground'>{t('Scorecard only')}</span>
      </span>
    )
  }

  return (
    <span className='flex max-w-72 min-w-52 flex-col gap-0.5'>
      <Badge variant='secondary'>{t('SLA Evidence Linked')}</Badge>
      <span>
        {t('SLA Contract')}: #{evaluation.sla_contract_id || '-'}
      </span>
      <span className='text-muted-foreground'>
        {t('Run')}: #{evaluation.sla_probe_run_id}
      </span>
      {runKey ? (
        <span className='text-muted-foreground truncate'>
          {t('Run Key')}: {runKey}
        </span>
      ) : null}
      {hardGatePassed == null ? null : (
        <Badge variant={hardGatePassed ? 'secondary' : 'destructive'}>
          {hardGatePassed ? t('Hard Gate Passed') : t('Hard Gate Failed')}
        </Badge>
      )}
      {artifactSHA256 ? (
        <span className='text-muted-foreground truncate'>
          {t('Artifact')}: {shortHash(artifactSHA256)}
        </span>
      ) : null}
      {runtimeRef ? (
        <span className='text-muted-foreground truncate'>
          {t('Runtime')}: {runtimeRef}
        </span>
      ) : null}
    </span>
  )
}

function renderOperatingInsightSlaEvidence(
  insight: OperatingInsight,
  t: Translator
): ReactNode {
  if (insight.sla_probe_run_id <= 0) {
    return (
      <span className='flex min-w-44 flex-col gap-1'>
        <Badge variant='outline'>{t('No SLA Evidence')}</Badge>
      </span>
    )
  }

  return (
    <span className='flex max-w-72 min-w-52 flex-col gap-0.5'>
      <Badge variant='secondary'>{t('SLA Evidence Linked')}</Badge>
      <span>
        {t('SLA Contract')}: #{insight.sla_contract_id || '-'}
      </span>
      <span className='text-muted-foreground'>
        {t('Run')}: #{insight.sla_probe_run_id}
      </span>
      <span className='text-muted-foreground'>
        {t('Status')}:{' '}
        {insight.sla_probe_status
          ? slaProbeRunStatusLabel(insight.sla_probe_status, t)
          : '-'}
      </span>
      {insight.sla_probe_run_key ? (
        <span className='text-muted-foreground truncate'>
          {t('Run Key')}: {insight.sla_probe_run_key}
        </span>
      ) : null}
      <Badge
        variant={insight.sla_hard_gate_passed ? 'secondary' : 'destructive'}
      >
        {insight.sla_hard_gate_passed
          ? t('Hard Gate Passed')
          : t('Hard Gate Failed')}
      </Badge>
      {insight.sla_failure_reasons ? (
        <span className='text-muted-foreground truncate'>
          {t('Failure Reasons')}: {insight.sla_failure_reasons}
        </span>
      ) : null}
      {insight.sla_artifact_sha256 ? (
        <span className='text-muted-foreground truncate'>
          {t('Artifact')}: {shortHash(insight.sla_artifact_sha256)}
        </span>
      ) : null}
      {insight.sla_runtime_ref ? (
        <span className='text-muted-foreground truncate'>
          {t('Runtime')}: {insight.sla_runtime_ref}
        </span>
      ) : null}
    </span>
  )
}

function renderRoutingPolicySlaEvidence(
  policy: SupplyRoutingPolicy,
  t: Translator
): ReactNode {
  if (policy.sla_probe_run_id <= 0) {
    return (
      <span className='flex min-w-44 flex-col gap-1'>
        <Badge variant='outline'>{t('No SLA Evidence')}</Badge>
      </span>
    )
  }

  return (
    <span className='flex max-w-72 min-w-52 flex-col gap-0.5'>
      <Badge variant='secondary'>{t('SLA Evidence Linked')}</Badge>
      <span>
        {t('SLA Contract')}: #{policy.sla_contract_id || '-'}
      </span>
      <span className='text-muted-foreground'>
        {t('Run')}: #{policy.sla_probe_run_id}
      </span>
      {policy.sla_probe_run_key ? (
        <span className='text-muted-foreground truncate'>
          {t('Run Key')}: {policy.sla_probe_run_key}
        </span>
      ) : null}
      {policy.sla_artifact_sha256 ? (
        <span className='text-muted-foreground truncate'>
          {t('Artifact')}: {shortHash(policy.sla_artifact_sha256)}
        </span>
      ) : null}
      {policy.sla_runtime_ref ? (
        <span className='text-muted-foreground truncate'>
          {t('Runtime')}: {policy.sla_runtime_ref}
        </span>
      ) : null}
    </span>
  )
}

function supplierNameById(suppliers: Supplier[], id: number) {
  const supplier = suppliers.find((item) => item.id === id)
  return supplier ? `${supplier.name} (#${id})` : `#${id}`
}

function supplierStatusLabel(status: number, t: Translator) {
  if (status === 1) return t('Enabled')
  if (status === 2) return t('Disabled')
  return status > 0 ? `#${status}` : '-'
}

function supplierRoutePreferenceSourceLabel(
  preference: SupplierRoutePreference,
  t: Translator
) {
  if (preference.source_posture_recommendation_id > 0) {
    return `${t('Recommendation')} #${preference.source_posture_recommendation_id}`
  }
  return t('Manual')
}

function supplierToForm(supplier: Supplier): SupplierFormState {
  return {
    id: supplier.id,
    name: supplier.name,
    type:
      supplier.type === 'self_operated' || supplier.type === 'self_hosted'
        ? supplier.type
        : 'third_party',
    status: String(supplier.status || 1),
    notes: supplier.notes || '',
  }
}

function supplierRoutePreferenceToForm(
  preference: SupplierRoutePreference | null,
  supplierId: number | undefined
): SupplierRoutePreferenceFormState {
  return {
    supplier_id: String(preference?.supplier_id ?? supplierId ?? ''),
    weight_percent: String(preference?.weight_percent ?? 25),
    reason: preference?.reason ?? '',
    effective_from:
      preference && preference.effective_from > 0
        ? formatTimestampForInput(preference.effective_from)
        : '',
    effective_to:
      preference && preference.effective_to > 0
        ? formatTimestampForInput(preference.effective_to)
        : '',
    operator_note: preference?.operator_note ?? '',
  }
}

function agreementToForm(
  agreement: SupplierAgreement
): SupplierAgreementFormState {
  return {
    id: agreement.id,
    supplier_id: String(agreement.supplier_id),
    model_name: agreement.model_name || '',
    effective_from:
      agreement.effective_from > 0
        ? formatTimestampForInput(agreement.effective_from)
        : '',
    effective_to:
      agreement.effective_to > 0
        ? formatTimestampForInput(agreement.effective_to)
        : '',
    use_price: agreement.use_price,
    cost_model_ratio: String(agreement.cost_model_ratio),
    cost_completion_ratio: String(agreement.cost_completion_ratio),
    cost_cache_ratio: String(agreement.cost_cache_ratio),
    cost_cache_creation_ratio: String(agreement.cost_cache_creation_ratio),
    cost_model_price: String(agreement.cost_model_price),
    priority: String(agreement.priority),
    status: String(agreement.status || 1),
    notes: agreement.notes || '',
  }
}

function actionExecutionPlanLabel(plan: SupplyActionPlan, t: Translator) {
  return `#${plan.id} ${actionTypeLabel(plan.action_type, t)} / ${plan.model_name || '-'} / ${decisionTrackLabel(plan.track, t)} / ${t('Decision')} #${plan.supply_decision_id}`
}

function slaProbePlanLabel(plan: SlaProbePlan, t: Translator) {
  return `#${plan.id} ${plan.model_name || '-'} / ${slaProbeTypeLabel(plan.probe_type, t)} / ${slaProbeRouteModeLabel(plan.route_mode, t)}`
}

function actionExecutionRecordFormFromPlan(
  plan?: SupplyActionPlan
): SupplyActionExecutionRecordFormState {
  if (!plan) {
    return DEFAULT_ACTION_EXECUTION_RECORD_FORM
  }

  return {
    ...DEFAULT_ACTION_EXECUTION_RECORD_FORM,
    supply_action_plan_id: String(plan.id),
    actual_capacity_tokens: String(Math.max(plan.recommended_capacity, 0)),
    effective_from:
      plan.period_start > 0 ? formatTimestampForInput(plan.period_start) : '',
    effective_to:
      plan.period_end > 0 ? formatTimestampForInput(plan.period_end) : '',
    operator_note: plan.operator_note || '',
  }
}

function parseInteger(value: string, fallback = 0) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  return Math.trunc(parsed)
}

function parseNumber(value: string, fallback = 0) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function parseOptionalTimestamp(value: string) {
  if (!value) return 0
  return parseTimestampFromInput(value)
}

function parseOptionalId(value: string, errorMessage: string) {
  const trimmed = value.trim()
  if (!trimmed) return 0
  const parsed = Number(trimmed)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    throw new Error(errorMessage)
  }
  return parsed
}

function requiredText(value: string, errorMessage: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    throw new Error(errorMessage)
  }
  return trimmed
}

function parseOptionalNonNegativeInteger(
  value: string,
  errorMessage: string,
  fallback = 0
) {
  const trimmed = value.trim()
  if (!trimmed) return fallback
  const parsed = Number(trimmed)
  if (!Number.isInteger(parsed) || parsed < 0) {
    throw new Error(errorMessage)
  }
  return parsed
}

function parsePositiveInteger(value: string, errorMessage: string) {
  const parsed = parseOptionalNonNegativeInteger(value, errorMessage)
  if (parsed <= 0) {
    throw new Error(errorMessage)
  }
  return parsed
}

function parseRoutingPolicyTrafficPercent(value: string, t: Translator) {
  const percent = parsePositiveInteger(
    value,
    t('Enter a valid traffic percent')
  )
  if (percent > 100) {
    throw new Error(t('Enter a valid traffic percent'))
  }
  return percent
}

function normalizeJsonText(
  value: string,
  label: string,
  t: Translator,
  required = false
) {
  const trimmed = value.trim()
  if (!trimmed) {
    if (required) {
      throw new Error(`${label}: ${t('JSON is required')}`)
    }
    return ''
  }
  try {
    JSON.parse(trimmed)
  } catch {
    throw new Error(`${label}: ${t('Invalid JSON')}`)
  }
  return trimmed
}

function buildSlaContractImportInput(
  form: SlaContractImportFormState,
  t: Translator
): SlaContractImportInput {
  const effectiveFrom = parseOptionalTimestamp(form.effective_from)
  const effectiveTo = parseOptionalTimestamp(form.effective_to)
  if (
    (form.effective_from && effectiveFrom <= 0) ||
    (form.effective_to && effectiveTo <= 0) ||
    (effectiveFrom > 0 && effectiveTo > 0 && effectiveTo <= effectiveFrom)
  ) {
    throw new Error(t('Enter a valid contract effective period'))
  }

  return {
    contract_key: requiredText(
      form.contract_key,
      t('Contract key is required')
    ),
    model_name: requiredText(form.model_name, t('Model is required')),
    model_aliases: form.model_aliases.trim(),
    provider_family: requiredText(
      form.provider_family,
      t('Provider family is required')
    ),
    source_name: requiredText(form.source_name, t('Source name is required')),
    source_ref: requiredText(
      form.source_ref,
      t('Source reference is required')
    ),
    source_sha256: requiredText(
      form.source_sha256,
      t('Source SHA256 is required')
    ),
    version: requiredText(form.version, t('Version is required')),
    status: form.status,
    effective_from: effectiveFrom,
    effective_to: effectiveTo,
    measurement_profile_json: normalizeJsonText(
      form.measurement_profile_json,
      t('Measurement Profile JSON'),
      t,
      true
    ),
    hard_gate_json:
      normalizeJsonText(form.hard_gate_json, t('Hard Gate JSON'), t) || '{}',
    soft_gate_json:
      normalizeJsonText(form.soft_gate_json, t('Soft Gate JSON'), t) || '{}',
  }
}

function buildSlaProbePlanGenerateInput(
  form: SlaProbePlanGenerateFormState,
  t: Translator
): SlaProbePlanGenerateInput {
  const contractId = parseOptionalId(
    form.contract_id,
    t('Enter a valid contract ID')
  )
  const contractKey = form.contract_key.trim()
  if (contractId <= 0 && !contractKey) {
    throw new Error(t('Select a contract or enter a contract key'))
  }
  const supplierId = parseOptionalId(
    form.supplier_id,
    t('Enter a valid supplier ID')
  )
  if (supplierId <= 0) {
    throw new Error(t('Enter a valid supplier ID'))
  }
  const channelId = parseOptionalId(
    form.channel_id,
    t('Enter a valid channel ID')
  )

  const input: SlaProbePlanGenerateInput = {
    supplier_id: supplierId,
    probe_type: form.probe_type,
    route_mode: form.route_mode,
    sample_size: parsePositiveInteger(
      form.sample_size,
      t('Enter a valid sample size')
    ),
    repeat_count: parsePositiveInteger(
      form.repeat_count,
      t('Enter a valid repeat count')
    ),
    schedule_interval_seconds: parseOptionalNonNegativeInteger(
      form.schedule_interval_seconds,
      t('Enter a valid schedule interval')
    ),
    jitter_seconds: parseOptionalNonNegativeInteger(
      form.jitter_seconds,
      t('Enter a valid jitter seconds')
    ),
    max_probe_quota: parseOptionalNonNegativeInteger(
      form.max_probe_quota,
      t('Enter a valid max probe quota')
    ),
  }
  if (contractId > 0) input.contract_id = contractId
  else input.contract_key = contractKey
  if (channelId > 0) input.channel_id = channelId
  if (form.model_name.trim()) input.model_name = form.model_name.trim()
  if (form.sla_tier.trim()) input.sla_tier = form.sla_tier.trim()
  if (form.prompt_suite_key.trim()) {
    input.prompt_suite_key = form.prompt_suite_key.trim()
  }
  if (form.tokenizer_ref.trim()) input.tokenizer_ref = form.tokenizer_ref.trim()
  if (form.cache_profile.trim()) input.cache_profile = form.cache_profile.trim()

  const jsonFields: Array<
    [
      keyof Pick<
        SlaProbePlanGenerateInput,
        | 'input_profile_json'
        | 'output_profile_json'
        | 'concurrency_profile_json'
        | 'rate_profile_json'
        | 'stream_profile_json'
        | 'error_profile_json'
        | 'availability_profile_json'
      >,
      string,
      string,
    ]
  > = [
    ['input_profile_json', form.input_profile_json, t('Input Profile JSON')],
    ['output_profile_json', form.output_profile_json, t('Output Profile JSON')],
    [
      'concurrency_profile_json',
      form.concurrency_profile_json,
      t('Concurrency Profile JSON'),
    ],
    ['rate_profile_json', form.rate_profile_json, t('Rate Profile JSON')],
    ['stream_profile_json', form.stream_profile_json, t('Stream Profile JSON')],
    ['error_profile_json', form.error_profile_json, t('Error Profile JSON')],
    [
      'availability_profile_json',
      form.availability_profile_json,
      t('Availability Profile JSON'),
    ],
  ]
  for (const [key, value, label] of jsonFields) {
    const normalized = normalizeJsonText(value, label, t)
    if (normalized) input[key] = normalized
  }
  return input
}

function buildSlaProbeRunRecordInput(
  form: SlaProbeRunRecordFormState,
  t: Translator
): SlaProbeRunRecordInput {
  const planId = parseOptionalId(form.plan_id, t('Enter a valid probe plan ID'))
  if (planId <= 0) {
    throw new Error(t('Enter a valid probe plan ID'))
  }
  const startedAt = parseOptionalTimestamp(form.started_at)
  const endedAt = parseOptionalTimestamp(form.ended_at)
  if (
    (form.started_at && startedAt <= 0) ||
    (form.ended_at && endedAt <= 0) ||
    (startedAt > 0 && endedAt > 0 && endedAt < startedAt)
  ) {
    throw new Error(t('Enter a valid probe run period'))
  }

  const input: SlaProbeRunRecordInput = {
    plan_id: planId,
    status: form.status,
    hard_gate_passed: form.hard_gate_passed,
    summary_json:
      normalizeJsonText(form.summary_json, t('Summary JSON'), t) || '{}',
    run_key: form.run_key.trim(),
    runner_version: form.runner_version.trim(),
    git_commit: form.git_commit.trim(),
    runtime_ref: form.runtime_ref.trim(),
    endpoint: form.endpoint.trim(),
    soft_gate_warnings: form.soft_gate_warnings.trim(),
    failure_reasons: form.failure_reasons.trim(),
    artifact_uri: form.artifact_uri.trim(),
    artifact_sha256: form.artifact_sha256.trim(),
  }
  if (startedAt > 0) input.started_at = startedAt
  if (endedAt > 0) input.ended_at = endedAt
  return input
}

function buildSupplierInput(
  form: SupplierFormState,
  t: Translator
): SupplierInput {
  const name = form.name.trim()
  if (!name) {
    throw new Error(t('Supplier name is required'))
  }
  return {
    id: form.id,
    name,
    type: form.type,
    status: parseInteger(form.status, 1),
    notes: form.notes.trim(),
  }
}

function buildAgreementInput(
  form: SupplierAgreementFormState,
  t: Translator
): SupplierAgreementInput {
  const supplierId = parseInteger(form.supplier_id)
  if (supplierId <= 0) {
    throw new Error(t('Enter a valid supplier ID'))
  }

  const effectiveFrom = parseOptionalTimestamp(form.effective_from)
  const effectiveTo = parseOptionalTimestamp(form.effective_to)
  if (
    (form.effective_from && effectiveFrom <= 0) ||
    (form.effective_to && effectiveTo <= 0) ||
    (effectiveFrom > 0 && effectiveTo > 0 && effectiveTo < effectiveFrom)
  ) {
    throw new Error(t('Enter a valid effective period'))
  }

  const input = {
    id: form.id,
    supplier_id: supplierId,
    model_name: form.model_name.trim(),
    effective_from: effectiveFrom,
    effective_to: effectiveTo,
    use_price: form.use_price,
    cost_model_ratio: parseNumber(form.cost_model_ratio, 1),
    cost_completion_ratio: parseNumber(form.cost_completion_ratio, 1),
    cost_cache_ratio: parseNumber(form.cost_cache_ratio, 0.1),
    cost_cache_creation_ratio: parseNumber(form.cost_cache_creation_ratio, 1),
    cost_model_price: parseNumber(form.cost_model_price, 0),
    priority: parseInteger(form.priority),
    status: parseInteger(form.status, 1),
    notes: form.notes.trim(),
  }

  const hasInvalidCost = [
    input.cost_model_ratio,
    input.cost_completion_ratio,
    input.cost_cache_ratio,
    input.cost_cache_creation_ratio,
    input.cost_model_price,
  ].some((value) => value < 0)
  if (hasInvalidCost) {
    throw new Error(t('Enter valid cost values'))
  }
  if (input.use_price && input.cost_model_price <= 0) {
    throw new Error(t('Enter a valid model price'))
  }
  return input
}

function buildSupplyCostProfileInput(
  form: SupplyCostProfileFormState,
  t: Translator
): SupplyCostProfileRecordInput {
  const supplierId = parseInteger(form.supplier_id)
  if (supplierId <= 0) {
    throw new Error(t('Enter a valid supplier ID'))
  }
  const supplyNode = form.supply_node.trim()
  if (!supplyNode) {
    throw new Error(t('Supply node is required'))
  }
  const modelName = form.model_name.trim()
  if (!modelName) {
    throw new Error(t('Model name is required'))
  }

  const periodStart = parseOptionalTimestamp(form.period_start)
  const periodEnd = parseOptionalTimestamp(form.period_end)
  if (periodStart <= 0 || periodEnd <= periodStart) {
    throw new Error(t('Enter a valid cost profile period'))
  }

  const capacityTokens = parseInteger(form.capacity_tokens)
  if (capacityTokens <= 0) {
    throw new Error(t('Enter a valid capacity token amount'))
  }
  const fixedCostQuota = parseNumber(form.fixed_cost_quota)
  const variableUnitCostQuota = parseNumber(form.variable_unit_cost_quota)
  if (fixedCostQuota < 0 || variableUnitCostQuota < 0) {
    throw new Error(t('Enter valid cost values'))
  }

  const sourceRef = form.source_ref.trim()
  if (!sourceRef) {
    throw new Error(t('Source reference is required'))
  }
  const observedAt = parseOptionalTimestamp(form.observed_at)
  if (observedAt <= 0) {
    throw new Error(t('Enter a valid observed time'))
  }

  return {
    supplier_id: supplierId,
    supply_node: supplyNode,
    model_name: modelName,
    period_start: periodStart,
    period_end: periodEnd,
    capacity_tokens: capacityTokens,
    fixed_cost_quota: fixedCostQuota,
    variable_unit_cost_quota: variableUnitCostQuota,
    source_type: form.source_type,
    source_ref: sourceRef,
    observed_at: observedAt,
    notes: form.notes.trim(),
  }
}

function buildSupplyPrepaidLotInput(
  form: SupplyPrepaidLotFormState,
  t: Translator
): SupplyPrepaidLotRecordInput {
  const supplierId = parseInteger(form.supplier_id)
  if (supplierId <= 0) {
    throw new Error(t('Enter a valid supplier ID'))
  }

  const periodStart = parseOptionalTimestamp(form.period_start)
  const periodEnd = parseOptionalTimestamp(form.period_end)
  if (periodStart <= 0 || periodEnd <= periodStart) {
    throw new Error(t('Enter a valid prepaid lot period'))
  }

  const purchasedTokens = parseInteger(form.purchased_tokens)
  if (purchasedTokens <= 0) {
    throw new Error(t('Enter a valid purchased token amount'))
  }

  const unitCost = parseNumber(form.unit_cost_quota)
  if (unitCost < 0) {
    throw new Error(t('Enter a valid unit cost'))
  }

  const sourceRef = form.source_ref.trim()
  if (!sourceRef) {
    throw new Error(t('Source reference is required'))
  }
  const observedAt = parseOptionalTimestamp(form.observed_at)
  if (observedAt <= 0) {
    throw new Error(t('Enter a valid observed time'))
  }

  const channelId = parseOptionalId(
    form.channel_id,
    t('Enter a valid channel ID')
  )
  const input: SupplyPrepaidLotRecordInput = {
    supplier_id: supplierId,
    period_start: periodStart,
    period_end: periodEnd,
    purchased_tokens: purchasedTokens,
    unit_cost_quota: unitCost,
    source_type: form.source_type,
    source_ref: sourceRef,
    observed_at: observedAt,
    supply_node: form.supply_node.trim(),
    model_name: form.model_name.trim(),
    external_ref: form.external_ref.trim(),
    notes: form.notes.trim(),
  }
  if (channelId > 0) input.channel_id = channelId
  return input
}

function buildSupplyActionExecutionRecordInput(
  form: SupplyActionExecutionRecordFormState,
  t: Translator
): SupplyActionExecutionRecordInput {
  const planId = parseOptionalId(
    form.supply_action_plan_id,
    t('Enter a valid action plan ID')
  )
  if (planId <= 0) {
    throw new Error(t('Enter a valid action plan ID'))
  }
  const supplierId = parseOptionalId(
    form.supplier_id,
    t('Enter a valid supplier ID')
  )
  const channelId = parseOptionalId(
    form.channel_id,
    t('Enter a valid channel ID')
  )
  const supplyCapacityId = parseOptionalId(
    form.supply_capacity_id,
    t('Enter a valid capacity snapshot ID')
  )
  const actualCapacity = Number(form.actual_capacity_tokens)
  if (!Number.isFinite(actualCapacity) || actualCapacity < 0) {
    throw new Error(t('Enter a valid actual capacity'))
  }

  const unitCost = Number(form.unit_cost_quota)
  if (!Number.isFinite(unitCost) || unitCost < 0) {
    throw new Error(t('Enter a valid unit cost'))
  }

  const effectiveFrom = parseOptionalTimestamp(form.effective_from)
  const effectiveTo = parseOptionalTimestamp(form.effective_to)
  if (
    (form.effective_from && effectiveFrom <= 0) ||
    (form.effective_to && effectiveTo <= 0) ||
    (effectiveFrom > 0 && effectiveTo > 0 && effectiveTo < effectiveFrom)
  ) {
    throw new Error(t('Enter a valid execution effective period'))
  }

  const input: SupplyActionExecutionRecordInput = {
    supply_action_plan_id: planId,
    execution_status: 'recorded',
    actual_capacity_tokens: actualCapacity,
    unit_cost_quota: unitCost,
    external_ref: form.external_ref.trim(),
    operator_note: form.operator_note.trim(),
  }
  if (supplierId > 0) input.supplier_id = supplierId
  if (channelId > 0) input.channel_id = channelId
  if (supplyCapacityId > 0) input.supply_capacity_id = supplyCapacityId
  if (effectiveFrom > 0) input.effective_from = effectiveFrom
  if (effectiveTo > 0) input.effective_to = effectiveTo
  return input
}

function StatCard(props: {
  title: string
  value: string
  description?: string
  isLoading?: boolean
}) {
  return (
    <Card size='sm'>
      <CardHeader>
        <CardTitle>{props.title}</CardTitle>
        {props.description && (
          <CardDescription>{props.description}</CardDescription>
        )}
      </CardHeader>
      <CardContent>
        {props.isLoading ? (
          <Skeleton className='h-8 w-28' />
        ) : (
          <div className='text-2xl font-semibold tabular-nums'>
            {props.value}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function DataPanel(props: {
  title: string
  description?: string
  action?: ReactNode
  children: ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{props.title}</CardTitle>
        {props.description && (
          <CardDescription>{props.description}</CardDescription>
        )}
        {props.action && <CardAction>{props.action}</CardAction>}
      </CardHeader>
      <CardContent>{props.children}</CardContent>
    </Card>
  )
}

function LoadingRows({
  columns,
  rows = 4,
}: {
  columns: number
  rows?: number
}) {
  const rowKeys = SKELETON_ROW_KEYS.slice(0, rows)
  const columnKeys = SKELETON_COLUMN_KEYS.slice(0, columns)

  return (
    <>
      {rowKeys.map((rowKey) => (
        <TableRow key={rowKey}>
          {columnKeys.map((columnKey) => (
            <TableCell key={`${rowKey}-${columnKey}`}>
              <Skeleton className='h-5 w-full' />
            </TableCell>
          ))}
        </TableRow>
      ))}
    </>
  )
}

function EmptyRow({ columns, message }: { columns: number; message: string }) {
  return (
    <TableRow>
      <TableCell
        colSpan={columns}
        className='text-muted-foreground py-8 text-center'
      >
        {message}
      </TableCell>
    </TableRow>
  )
}

function TableRowsState(props: {
  isLoading: boolean
  isEmpty: boolean
  columns: number
  emptyMessage: string
  children: ReactNode
}) {
  if (props.isLoading) {
    return <LoadingRows columns={props.columns} />
  }
  if (props.isEmpty) {
    return <EmptyRow columns={props.columns} message={props.emptyMessage} />
  }
  return props.children
}

function SupplierFormDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  form: SupplierFormState
  setForm: Dispatch<SetStateAction<SupplierFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  const isEditing = Boolean(props.form.id)

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>
            {isEditing ? props.t('Edit Supplier') : props.t('Add Supplier')}
          </DialogTitle>
          <DialogDescription>
            {props.t(
              'Suppliers are upstream settlement parties; no payment fields are stored here.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='token-router-supplier-name'>
              {props.t('Name')}
            </FieldLabel>
            <Input
              id='token-router-supplier-name'
              value={props.form.name}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
            />
          </Field>

          <Field>
            <FieldTitle>{props.t('Type')}</FieldTitle>
            <ToggleGroup
              value={[props.form.type]}
              onValueChange={(value) => {
                const next = value.find((item) => item !== props.form.type)
                if (next) {
                  props.setForm((current) => ({
                    ...current,
                    type: next as SupplierType,
                  }))
                }
              }}
              aria-label={props.t('Type')}
              variant='outline'
              size='sm'
              spacing={2}
              className='flex-wrap justify-start'
            >
              <ToggleGroupItem value='third_party'>
                {props.t('Third-party supply')}
              </ToggleGroupItem>
              <ToggleGroupItem value='self_operated'>
                {props.t('Self-operated supply')}
              </ToggleGroupItem>
              <ToggleGroupItem value='self_hosted'>
                {props.t('Self-hosted supply')}
              </ToggleGroupItem>
            </ToggleGroup>
          </Field>

          <Field>
            <FieldTitle>{props.t('Status')}</FieldTitle>
            <ToggleGroup
              value={[props.form.status]}
              onValueChange={(value) => {
                const next = value.find((item) => item !== props.form.status)
                if (next) {
                  props.setForm((current) => ({ ...current, status: next }))
                }
              }}
              aria-label={props.t('Status')}
              variant='outline'
              size='sm'
              spacing={2}
            >
              <ToggleGroupItem value='1'>{props.t('Enabled')}</ToggleGroupItem>
              <ToggleGroupItem value='2'>{props.t('Disabled')}</ToggleGroupItem>
            </ToggleGroup>
          </Field>

          <Field>
            <FieldLabel htmlFor='token-router-supplier-notes'>
              {props.t('Notes')}
            </FieldLabel>
            <Textarea
              id='token-router-supplier-notes'
              rows={3}
              value={props.form.notes}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  notes: event.target.value,
                }))
              }
            />
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving}
          >
            {props.isSaving && <Spinner data-icon='inline-start' />}
            {isEditing ? props.t('Save') : props.t('Create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SupplierRoutePreferenceDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  enabledSuppliers: Supplier[]
  form: SupplierRoutePreferenceFormState
  setForm: Dispatch<SetStateAction<SupplierRoutePreferenceFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100svh-1rem)] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{props.t('Set Supplier Route Preference')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Apply a bounded supplier route preference without changing channel weights, ability weights, pricing, billing, or settlement.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='token-router-route-preference-supplier'>
              {props.t('Supplier')}
            </FieldLabel>
            <NativeSelect
              id='token-router-route-preference-supplier'
              className='w-full'
              value={props.form.supplier_id}
              disabled={props.enabledSuppliers.length === 0}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  supplier_id: event.target.value,
                }))
              }
            >
              <NativeSelectOption value=''>
                {props.t('Select supplier')}
              </NativeSelectOption>
              {props.enabledSuppliers.map((supplier) => (
                <NativeSelectOption
                  key={supplier.id}
                  value={String(supplier.id)}
                >
                  {supplier.name} (#{supplier.id})
                </NativeSelectOption>
              ))}
            </NativeSelect>
            <FieldDescription>
              {props.t('Only enabled suppliers can receive route preferences.')}
            </FieldDescription>
          </Field>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-route-preference-weight'>
                {props.t('Route Weight Percent')}
              </FieldLabel>
              <Input
                id='token-router-route-preference-weight'
                type='number'
                min='1'
                max='200'
                step='1'
                value={props.form.weight_percent}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    weight_percent: event.target.value,
                  }))
                }
              />
              <FieldDescription>
                {props.t(
                  'Use 1 to 200; 100 is baseline, lower values reduce weight, higher values boost weight.'
                )}
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-route-preference-note'>
                {props.t('Operator Note')}
              </FieldLabel>
              <Input
                id='token-router-route-preference-note'
                value={props.form.operator_note}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    operator_note: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-route-preference-from'>
                {props.t('Effective From')}
              </FieldLabel>
              <Input
                id='token-router-route-preference-from'
                type='datetime-local'
                value={props.form.effective_from}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    effective_from: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-route-preference-to'>
                {props.t('Effective To')}
              </FieldLabel>
              <Input
                id='token-router-route-preference-to'
                type='datetime-local'
                value={props.form.effective_to}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    effective_to: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <Field>
            <FieldLabel htmlFor='token-router-route-preference-reason'>
              {props.t('Reason')}
            </FieldLabel>
            <Textarea
              id='token-router-route-preference-reason'
              rows={3}
              value={props.form.reason}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  reason: event.target.value,
                }))
              }
            />
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving || props.enabledSuppliers.length === 0}
          >
            {props.isSaving && <Spinner data-icon='inline-start' />}
            {props.t('Set Route Preference')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SupplyRoutingPolicyActivateDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  form: SupplyRoutingPolicyActivateFormState
  setForm: Dispatch<SetStateAction<SupplyRoutingPolicyActivateFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  const execution = props.form.execution

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100svh-1rem)] overflow-y-auto sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{props.t('Activate Routing Policy')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Choose the traffic share for this self-hosted route. The backend still requires passed runtime SLA evidence.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldTitle>{props.t('Execution')}</FieldTitle>
            <FieldDescription>
              {execution
                ? `#${execution.id} / ${execution.model_name || '-'} / ${execution.sla_tier || '-'} / ${props.t('User')} #${execution.user_id}`
                : '-'}
            </FieldDescription>
          </Field>

          <Field>
            <FieldLabel htmlFor='token-router-routing-policy-traffic-percent'>
              {props.t('Traffic Percent')}
            </FieldLabel>
            <Input
              id='token-router-routing-policy-traffic-percent'
              type='number'
              min='1'
              max='100'
              step='1'
              value={props.form.traffic_percent}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  traffic_percent: event.target.value,
                }))
              }
            />
            <FieldDescription>
              {props.t(
                'Use 1 to 100; 100 routes every matching session, lower values create a deterministic session canary.'
              )}
            </FieldDescription>
          </Field>

          <Field>
            <FieldLabel htmlFor='token-router-routing-policy-note'>
              {props.t('Operator Note')}
            </FieldLabel>
            <Textarea
              id='token-router-routing-policy-note'
              rows={3}
              value={props.form.operator_note}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  operator_note: event.target.value,
                }))
              }
            />
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSaving}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving || !execution}
          >
            {props.isSaving && <Spinner data-icon='inline-start' />}
            {props.t('Activate Policy')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SupplierAgreementFormDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  form: SupplierAgreementFormState
  setForm: Dispatch<SetStateAction<SupplierAgreementFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  const isEditing = Boolean(props.form.id)
  const pricingMode = props.form.use_price ? 'price' : 'ratio'

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[min(760px,calc(100vh-2rem))] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {isEditing
              ? props.t('Edit Supplier Agreement')
              : props.t('Add Supplier Agreement')}
          </DialogTitle>
          <DialogDescription>
            {props.t(
              'Cost agreements are cache-aware ledger inputs, not payment instructions.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <div className='grid gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-agreement-supplier'>
                {props.t('Supplier ID')}
              </FieldLabel>
              <Input
                id='token-router-agreement-supplier'
                type='number'
                min={1}
                value={props.form.supplier_id}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    supplier_id: event.target.value,
                  }))
                }
              />
              <FieldDescription>
                {props.t('Use the numeric ID from the Suppliers table.')}
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-agreement-model'>
                {props.t('Model')}
              </FieldLabel>
              <Input
                id='token-router-agreement-model'
                value={props.form.model_name}
                placeholder={props.t('All Models')}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    model_name: event.target.value,
                  }))
                }
              />
            </Field>
          </div>

          <div className='grid gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-agreement-from'>
                {props.t('Effective From')}
              </FieldLabel>
              <Input
                id='token-router-agreement-from'
                type='datetime-local'
                value={props.form.effective_from}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    effective_from: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-agreement-to'>
                {props.t('Effective To')}
              </FieldLabel>
              <Input
                id='token-router-agreement-to'
                type='datetime-local'
                value={props.form.effective_to}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    effective_to: event.target.value,
                  }))
                }
              />
              <FieldDescription>
                {props.t('Leave empty for no end time.')}
              </FieldDescription>
            </Field>
          </div>

          <Field>
            <FieldTitle>{props.t('Cost Mode')}</FieldTitle>
            <ToggleGroup
              value={[pricingMode]}
              onValueChange={(value) => {
                const next = value.find((item) => item !== pricingMode)
                if (next) {
                  props.setForm((current) => ({
                    ...current,
                    use_price: next === 'price',
                  }))
                }
              }}
              aria-label={props.t('Cost Mode')}
              variant='outline'
              size='sm'
              spacing={2}
            >
              <ToggleGroupItem value='ratio'>
                {props.t('Ratio')}
              </ToggleGroupItem>
              <ToggleGroupItem value='price'>
                {props.t('Fixed Price')}
              </ToggleGroupItem>
            </ToggleGroup>
          </Field>

          {props.form.use_price ? (
            <Field>
              <FieldLabel htmlFor='token-router-agreement-price'>
                {props.t('Model Price')}
              </FieldLabel>
              <Input
                id='token-router-agreement-price'
                type='number'
                min={0}
                step='0.000001'
                value={props.form.cost_model_price}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    cost_model_price: event.target.value,
                  }))
                }
              />
            </Field>
          ) : (
            <div className='grid gap-4 sm:grid-cols-2'>
              <Field>
                <FieldLabel htmlFor='token-router-agreement-prompt-ratio'>
                  {props.t('Prompt Ratio')}
                </FieldLabel>
                <Input
                  id='token-router-agreement-prompt-ratio'
                  type='number'
                  min={0}
                  step='0.000001'
                  value={props.form.cost_model_ratio}
                  onChange={(event) =>
                    props.setForm((current) => ({
                      ...current,
                      cost_model_ratio: event.target.value,
                    }))
                  }
                />
              </Field>

              <Field>
                <FieldLabel htmlFor='token-router-agreement-completion-ratio'>
                  {props.t('Completion Ratio')}
                </FieldLabel>
                <Input
                  id='token-router-agreement-completion-ratio'
                  type='number'
                  min={0}
                  step='0.000001'
                  value={props.form.cost_completion_ratio}
                  onChange={(event) =>
                    props.setForm((current) => ({
                      ...current,
                      cost_completion_ratio: event.target.value,
                    }))
                  }
                />
              </Field>

              <Field>
                <FieldLabel htmlFor='token-router-agreement-cache-ratio'>
                  {props.t('Cache Ratio')}
                </FieldLabel>
                <Input
                  id='token-router-agreement-cache-ratio'
                  type='number'
                  min={0}
                  step='0.000001'
                  value={props.form.cost_cache_ratio}
                  onChange={(event) =>
                    props.setForm((current) => ({
                      ...current,
                      cost_cache_ratio: event.target.value,
                    }))
                  }
                />
              </Field>

              <Field>
                <FieldLabel htmlFor='token-router-agreement-cache-create-ratio'>
                  {props.t('Cache Creation Ratio')}
                </FieldLabel>
                <Input
                  id='token-router-agreement-cache-create-ratio'
                  type='number'
                  min={0}
                  step='0.000001'
                  value={props.form.cost_cache_creation_ratio}
                  onChange={(event) =>
                    props.setForm((current) => ({
                      ...current,
                      cost_cache_creation_ratio: event.target.value,
                    }))
                  }
                />
              </Field>
            </div>
          )}

          <div className='grid gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-agreement-priority'>
                {props.t('Priority')}
              </FieldLabel>
              <Input
                id='token-router-agreement-priority'
                type='number'
                value={props.form.priority}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    priority: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldTitle>{props.t('Status')}</FieldTitle>
              <ToggleGroup
                value={[props.form.status]}
                onValueChange={(value) => {
                  const next = value.find((item) => item !== props.form.status)
                  if (next) {
                    props.setForm((current) => ({ ...current, status: next }))
                  }
                }}
                aria-label={props.t('Status')}
                variant='outline'
                size='sm'
                spacing={2}
              >
                <ToggleGroupItem value='1'>
                  {props.t('Enabled')}
                </ToggleGroupItem>
                <ToggleGroupItem value='2'>
                  {props.t('Disabled')}
                </ToggleGroupItem>
              </ToggleGroup>
            </Field>
          </div>

          <Field>
            <FieldLabel htmlFor='token-router-agreement-notes'>
              {props.t('Notes')}
            </FieldLabel>
            <Textarea
              id='token-router-agreement-notes'
              rows={3}
              value={props.form.notes}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  notes: event.target.value,
                }))
              }
            />
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving}
          >
            {props.isSaving && <Spinner data-icon='inline-start' />}
            {isEditing ? props.t('Save') : props.t('Create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SupplyCostProfileRecordDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  selfHostedSuppliers: Supplier[]
  form: SupplyCostProfileFormState
  setForm: Dispatch<SetStateAction<SupplyCostProfileFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100svh-1rem)] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{props.t('Record Cost Profile')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Record a self-hosted amortized cost basis for opportunity ranking; this does not purchase capacity, change pricing, activate routing, or touch settlement.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-cost-supplier'>
                {props.t('Self-hosted Supplier')}
              </FieldLabel>
              <NativeSelect
                id='token-router-cost-supplier'
                className='w-full'
                value={props.form.supplier_id}
                disabled={props.selfHostedSuppliers.length === 0}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    supplier_id: event.target.value,
                  }))
                }
              >
                <NativeSelectOption value=''>
                  {props.selfHostedSuppliers.length > 0
                    ? props.t('Select a self-hosted supplier')
                    : props.t('No self-hosted suppliers available')}
                </NativeSelectOption>
                {props.selfHostedSuppliers.map((supplier) => (
                  <NativeSelectOption
                    key={supplier.id}
                    value={String(supplier.id)}
                  >
                    #{supplier.id} {supplier.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
              <FieldDescription>
                {props.t(
                  'Cost profiles are accepted only for suppliers whose type is self_hosted.'
                )}
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-cost-node'>
                {props.t('Supply Node')}
              </FieldLabel>
              <Input
                id='token-router-cost-node'
                value={props.form.supply_node}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    supply_node: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-cost-model'>
                {props.t('Model')}
              </FieldLabel>
              <Input
                id='token-router-cost-model'
                value={props.form.model_name}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    model_name: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-cost-source-type'>
                {props.t('Source Type')}
              </FieldLabel>
              <NativeSelect
                id='token-router-cost-source-type'
                className='w-full'
                value={props.form.source_type}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    source_type: event.target
                      .value as SupplyCostProfileSourceType,
                  }))
                }
              >
                <NativeSelectOption value='accounting'>
                  {props.t('Accounting')}
                </NativeSelectOption>
                <NativeSelectOption value='manual'>
                  {props.t('Manual')}
                </NativeSelectOption>
                <NativeSelectOption value='external'>
                  {props.t('External')}
                </NativeSelectOption>
              </NativeSelect>
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-cost-period-start'>
                {props.t('Period Start')}
              </FieldLabel>
              <Input
                id='token-router-cost-period-start'
                type='datetime-local'
                value={props.form.period_start}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    period_start: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-cost-period-end'>
                {props.t('Period End')}
              </FieldLabel>
              <Input
                id='token-router-cost-period-end'
                type='datetime-local'
                value={props.form.period_end}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    period_end: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-cost-observed-at'>
                {props.t('Observed At')}
              </FieldLabel>
              <Input
                id='token-router-cost-observed-at'
                type='datetime-local'
                value={props.form.observed_at}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    observed_at: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-cost-source-ref'>
                {props.t('Source Reference')}
              </FieldLabel>
              <Input
                id='token-router-cost-source-ref'
                value={props.form.source_ref}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    source_ref: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-3'>
            <Field>
              <FieldLabel htmlFor='token-router-cost-capacity'>
                {props.t('Capacity Tokens')}
              </FieldLabel>
              <Input
                id='token-router-cost-capacity'
                type='number'
                inputMode='numeric'
                min='1'
                value={props.form.capacity_tokens}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    capacity_tokens: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-cost-fixed'>
                {props.t('Fixed Cost Quota')}
              </FieldLabel>
              <Input
                id='token-router-cost-fixed'
                type='number'
                inputMode='decimal'
                min='0'
                step='0.000001'
                value={props.form.fixed_cost_quota}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    fixed_cost_quota: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-cost-variable'>
                {props.t('Variable Unit Cost')}
              </FieldLabel>
              <Input
                id='token-router-cost-variable'
                type='number'
                inputMode='decimal'
                min='0'
                step='0.000001'
                value={props.form.variable_unit_cost_quota}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    variable_unit_cost_quota: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <Field>
            <FieldLabel htmlFor='token-router-cost-notes'>
              {props.t('Notes')}
            </FieldLabel>
            <Textarea
              id='token-router-cost-notes'
              rows={4}
              value={props.form.notes}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  notes: event.target.value,
                }))
              }
            />
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSaving}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving || props.selfHostedSuppliers.length === 0}
          >
            {props.isSaving ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <Plus data-icon='inline-start' />
            )}
            {props.t('Record Cost Profile')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SupplyPrepaidLotRecordDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  selfOperatedSuppliers: Supplier[]
  form: SupplyPrepaidLotFormState
  setForm: Dispatch<SetStateAction<SupplyPrepaidLotFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100svh-1rem)] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{props.t('Record Prepaid Lot')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Record offline self-operated prepaid token evidence; this does not create payments, purchase approvals, capacity, routing, billing, or settlement.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-prepaid-supplier'>
                {props.t('Self-operated Supplier')}
              </FieldLabel>
              <NativeSelect
                id='token-router-prepaid-supplier'
                className='w-full'
                value={props.form.supplier_id}
                disabled={props.selfOperatedSuppliers.length === 0}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    supplier_id: event.target.value,
                  }))
                }
              >
                <NativeSelectOption value=''>
                  {props.selfOperatedSuppliers.length > 0
                    ? props.t('Select a self-operated supplier')
                    : props.t('No self-operated suppliers available')}
                </NativeSelectOption>
                {props.selfOperatedSuppliers.map((supplier) => (
                  <NativeSelectOption
                    key={supplier.id}
                    value={String(supplier.id)}
                  >
                    #{supplier.id} {supplier.name}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
              <FieldDescription>
                {props.t(
                  'Prepaid lots are accepted only for suppliers whose type is self_operated.'
                )}
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-channel'>
                {props.t('Channel ID')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-channel'
                type='number'
                inputMode='numeric'
                min='1'
                value={props.form.channel_id}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    channel_id: event.target.value,
                  }))
                }
              />
              <FieldDescription>
                {props.t('Optional; leave empty to match all channels.')}
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-node'>
                {props.t('Supply Node')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-node'
                value={props.form.supply_node}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    supply_node: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-model'>
                {props.t('Model')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-model'
                value={props.form.model_name}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    model_name: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-source-type'>
                {props.t('Source Type')}
              </FieldLabel>
              <NativeSelect
                id='token-router-prepaid-source-type'
                className='w-full'
                value={props.form.source_type}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    source_type: event.target
                      .value as SupplyPrepaidLotSourceType,
                  }))
                }
              >
                <NativeSelectOption value='accounting'>
                  {props.t('Accounting')}
                </NativeSelectOption>
                <NativeSelectOption value='manual'>
                  {props.t('Manual')}
                </NativeSelectOption>
                <NativeSelectOption value='external'>
                  {props.t('External')}
                </NativeSelectOption>
              </NativeSelect>
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-source-ref'>
                {props.t('Source Reference')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-source-ref'
                value={props.form.source_ref}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    source_ref: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-prepaid-period-start'>
                {props.t('Period Start')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-period-start'
                type='datetime-local'
                value={props.form.period_start}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    period_start: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-period-end'>
                {props.t('Period End')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-period-end'
                type='datetime-local'
                value={props.form.period_end}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    period_end: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-observed-at'>
                {props.t('Observed At')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-observed-at'
                type='datetime-local'
                value={props.form.observed_at}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    observed_at: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-external-ref'>
                {props.t('External Reference')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-external-ref'
                value={props.form.external_ref}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    external_ref: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-prepaid-purchased'>
                {props.t('Purchased Tokens')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-purchased'
                type='number'
                inputMode='numeric'
                min='1'
                value={props.form.purchased_tokens}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    purchased_tokens: event.target.value,
                  }))
                }
              />
            </Field>

            <Field>
              <FieldLabel htmlFor='token-router-prepaid-unit-cost'>
                {props.t('Unit Cost Quota')}
              </FieldLabel>
              <Input
                id='token-router-prepaid-unit-cost'
                type='number'
                inputMode='decimal'
                min='0'
                step='0.000001'
                value={props.form.unit_cost_quota}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    unit_cost_quota: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <Field>
            <FieldLabel htmlFor='token-router-prepaid-notes'>
              {props.t('Notes')}
            </FieldLabel>
            <Textarea
              id='token-router-prepaid-notes'
              rows={4}
              value={props.form.notes}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  notes: event.target.value,
                }))
              }
            />
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSaving}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={
              props.isSaving || props.selfOperatedSuppliers.length === 0
            }
          >
            {props.isSaving ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <Plus data-icon='inline-start' />
            )}
            {props.t('Record Prepaid Lot')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ActionPlanStatusDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  form: ActionPlanStatusFormState
  setForm: Dispatch<SetStateAction<ActionPlanStatusFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  const plan = props.form.plan
  const statusOptions = plan ? nextActionPlanStatuses(plan.status) : []

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{props.t('Update Action Plan Status')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Record the operator-managed offline progress; this does not mutate suppliers, channels, routing, purchasing, or payments.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldTitle>{props.t('Action Plan')}</FieldTitle>
            <FieldDescription>
              {plan
                ? `${actionTypeLabel(plan.action_type, props.t)} / ${plan.model_name || '-'} / ${props.t('Decision')} #${plan.supply_decision_id}`
                : '-'}
            </FieldDescription>
          </Field>

          <Field>
            <FieldTitle>{props.t('Status')}</FieldTitle>
            <ToggleGroup
              value={[props.form.status]}
              onValueChange={(value) => {
                const next = value.find((item) => item !== props.form.status)
                if (next) {
                  props.setForm((current) => ({
                    ...current,
                    status: next as SupplyActionPlanStatus,
                  }))
                }
              }}
              aria-label={props.t('Status')}
              variant='outline'
              size='sm'
              spacing={2}
              className='flex-wrap justify-start'
            >
              {statusOptions.map((status) => (
                <ToggleGroupItem key={status} value={status}>
                  {actionPlanStatusLabel(status, props.t)}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
          </Field>

          <Field>
            <FieldLabel htmlFor='token-router-action-plan-note'>
              {props.t('Operator Note')}
            </FieldLabel>
            <Textarea
              id='token-router-action-plan-note'
              rows={4}
              value={props.form.operator_note}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  operator_note: event.target.value,
                }))
              }
            />
            <FieldDescription>
              {props.t('Record how the offline work moved forward.')}
            </FieldDescription>
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSaving}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving || !plan || statusOptions.length === 0}
          >
            {props.isSaving ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <Pencil data-icon='inline-start' />
            )}
            {props.t('Save Status')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SupplyActionExecutionRecordDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  completedPlans: SupplyActionPlan[]
  form: SupplyActionExecutionRecordFormState
  setForm: Dispatch<SetStateAction<SupplyActionExecutionRecordFormState>>
  isSaving: boolean
  isLoadingPlans: boolean
  onSubmit: () => void
  t: Translator
}) {
  const selectedPlan = props.completedPlans.find(
    (plan) => String(plan.id) === props.form.supply_action_plan_id
  )

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='z-[100] max-h-[calc(100svh-1rem)] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{props.t('Record Supply Execution')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Record offline execution facts for a completed action plan; this does not create suppliers, mutate capacity, activate routing, or touch payments.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='token-router-execution-plan'>
              {props.t('Completed Action Plan')}
            </FieldLabel>
            <NativeSelect
              id='token-router-execution-plan'
              className='w-full'
              value={props.form.supply_action_plan_id}
              disabled={
                props.isLoadingPlans || props.completedPlans.length === 0
              }
              onChange={(event) => {
                const nextPlan = props.completedPlans.find(
                  (plan) => String(plan.id) === event.target.value
                )
                props.setForm((current) => ({
                  ...actionExecutionRecordFormFromPlan(nextPlan),
                  supplier_id: current.supplier_id,
                  channel_id: current.channel_id,
                  supply_capacity_id: current.supply_capacity_id,
                  unit_cost_quota: current.unit_cost_quota,
                  external_ref: current.external_ref,
                  operator_note:
                    current.operator_note || nextPlan?.operator_note || '',
                }))
              }}
            >
              <NativeSelectOption value=''>
                {props.completedPlans.length > 0
                  ? props.t('Select a completed action plan')
                  : props.t('No completed action plans available')}
              </NativeSelectOption>
              {props.completedPlans.map((plan) => (
                <NativeSelectOption key={plan.id} value={String(plan.id)}>
                  {actionExecutionPlanLabel(plan, props.t)}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            {selectedPlan && (
              <FieldDescription>
                {selectedPlan.sla_tier || '-'} / {props.t('User')} #
                {selectedPlan.user_id} / {props.t('Recommended')}:{' '}
                {formatTokens(selectedPlan.recommended_capacity)}
              </FieldDescription>
            )}
          </Field>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-3'>
            <Field>
              <FieldLabel htmlFor='token-router-execution-supplier'>
                {props.t('Supplier ID')}
              </FieldLabel>
              <Input
                id='token-router-execution-supplier'
                type='number'
                inputMode='numeric'
                min='1'
                value={props.form.supplier_id}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    supplier_id: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-execution-channel'>
                {props.t('Channel ID')}
              </FieldLabel>
              <Input
                id='token-router-execution-channel'
                type='number'
                inputMode='numeric'
                min='1'
                value={props.form.channel_id}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    channel_id: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-execution-capacity'>
                {props.t('Capacity Snapshot ID')}
              </FieldLabel>
              <Input
                id='token-router-execution-capacity'
                type='number'
                inputMode='numeric'
                min='1'
                value={props.form.supply_capacity_id}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    supply_capacity_id: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-execution-actual-capacity'>
                {props.t('Actual Capacity Tokens')}
              </FieldLabel>
              <Input
                id='token-router-execution-actual-capacity'
                type='number'
                inputMode='numeric'
                min='0'
                value={props.form.actual_capacity_tokens}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    actual_capacity_tokens: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-execution-unit-cost'>
                {props.t('Unit Cost Quota')}
              </FieldLabel>
              <Input
                id='token-router-execution-unit-cost'
                type='number'
                inputMode='decimal'
                min='0'
                step='0.000001'
                value={props.form.unit_cost_quota}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    unit_cost_quota: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-execution-effective-from'>
                {props.t('Effective From')}
              </FieldLabel>
              <Input
                id='token-router-execution-effective-from'
                type='datetime-local'
                value={props.form.effective_from}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    effective_from: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-execution-effective-to'>
                {props.t('Effective To')}
              </FieldLabel>
              <Input
                id='token-router-execution-effective-to'
                type='datetime-local'
                value={props.form.effective_to}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    effective_to: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <Field>
            <FieldLabel htmlFor='token-router-execution-external-ref'>
              {props.t('External Reference')}
            </FieldLabel>
            <Input
              id='token-router-execution-external-ref'
              value={props.form.external_ref}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  external_ref: event.target.value,
                }))
              }
            />
          </Field>

          <Field>
            <FieldLabel htmlFor='token-router-execution-note'>
              {props.t('Operator Note')}
            </FieldLabel>
            <Textarea
              id='token-router-execution-note'
              rows={4}
              value={props.form.operator_note}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  operator_note: event.target.value,
                }))
              }
            />
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSaving}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving || !selectedPlan}
          >
            {props.isSaving ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <Pencil data-icon='inline-start' />
            )}
            {props.t('Record Execution')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SlaContractImportDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  form: SlaContractImportFormState
  setForm: Dispatch<SetStateAction<SlaContractImportFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100svh-1rem)] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>{props.t('Import SLA Contract')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Import versioned SLA evidence only; this does not admit suppliers, alter routing, or touch settlement.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-sla-contract-key'>
                {props.t('Contract Key')}
              </FieldLabel>
              <Input
                id='token-router-sla-contract-key'
                value={props.form.contract_key}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    contract_key: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-model'>
                {props.t('Model')}
              </FieldLabel>
              <Input
                id='token-router-sla-model'
                value={props.form.model_name}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    model_name: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-aliases'>
                {props.t('Model Aliases')}
              </FieldLabel>
              <Input
                id='token-router-sla-aliases'
                value={props.form.model_aliases}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    model_aliases: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-provider-family'>
                {props.t('Provider Family')}
              </FieldLabel>
              <Input
                id='token-router-sla-provider-family'
                value={props.form.provider_family}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    provider_family: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-source-name'>
                {props.t('Source Name')}
              </FieldLabel>
              <Input
                id='token-router-sla-source-name'
                value={props.form.source_name}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    source_name: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-source-ref'>
                {props.t('Source Reference')}
              </FieldLabel>
              <Input
                id='token-router-sla-source-ref'
                value={props.form.source_ref}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    source_ref: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-source-sha'>
                {props.t('Source SHA256')}
              </FieldLabel>
              <Input
                id='token-router-sla-source-sha'
                value={props.form.source_sha256}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    source_sha256: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-version'>
                {props.t('Version')}
              </FieldLabel>
              <Input
                id='token-router-sla-version'
                value={props.form.version}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    version: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-effective-from'>
                {props.t('Effective From')}
              </FieldLabel>
              <Input
                id='token-router-sla-effective-from'
                type='datetime-local'
                value={props.form.effective_from}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    effective_from: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-effective-to'>
                {props.t('Effective To')}
              </FieldLabel>
              <Input
                id='token-router-sla-effective-to'
                type='datetime-local'
                value={props.form.effective_to}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    effective_to: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <Field>
            <FieldTitle>{props.t('Contract Status')}</FieldTitle>
            <ToggleGroup
              value={[props.form.status]}
              onValueChange={(value) => {
                const next = value.find((item) => item !== props.form.status)
                if (next) {
                  props.setForm((current) => ({
                    ...current,
                    status: next as SlaContractStatus,
                  }))
                }
              }}
              aria-label={props.t('Contract Status')}
              variant='outline'
              size='sm'
              spacing={2}
              className='flex-wrap justify-start'
            >
              <ToggleGroupItem value='draft'>
                {props.t('Draft')}
              </ToggleGroupItem>
              <ToggleGroupItem value='active'>
                {props.t('Active')}
              </ToggleGroupItem>
              <ToggleGroupItem value='retired'>
                {props.t('Retired')}
              </ToggleGroupItem>
            </ToggleGroup>
          </Field>

          <Field>
            <FieldLabel htmlFor='token-router-sla-measurement-json'>
              {props.t('Measurement Profile JSON')}
            </FieldLabel>
            <Textarea
              id='token-router-sla-measurement-json'
              rows={8}
              value={props.form.measurement_profile_json}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  measurement_profile_json: event.target.value,
                }))
              }
            />
          </Field>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-sla-hard-gate-json'>
                {props.t('Hard Gate JSON')}
              </FieldLabel>
              <Textarea
                id='token-router-sla-hard-gate-json'
                rows={5}
                value={props.form.hard_gate_json}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    hard_gate_json: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-soft-gate-json'>
                {props.t('Soft Gate JSON')}
              </FieldLabel>
              <Textarea
                id='token-router-sla-soft-gate-json'
                rows={5}
                value={props.form.soft_gate_json}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    soft_gate_json: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSaving}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving}
          >
            {props.isSaving ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <Plus data-icon='inline-start' />
            )}
            {props.t('Import')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SlaProbePlanGenerateDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  contracts: SlaContract[]
  form: SlaProbePlanGenerateFormState
  setForm: Dispatch<SetStateAction<SlaProbePlanGenerateFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  const selectedContract = props.contracts.find(
    (contract) => String(contract.id) === props.form.contract_id
  )

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100svh-1rem)] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>{props.t('Generate Probe Plan')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Generate an SLA probe plan from a contract; execution remains outside the browser.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='token-router-sla-plan-contract'>
              {props.t('SLA Contract')}
            </FieldLabel>
            <NativeSelect
              id='token-router-sla-plan-contract'
              className='w-full'
              value={props.form.contract_id}
              onChange={(event) => {
                const contract = props.contracts.find(
                  (item) => String(item.id) === event.target.value
                )
                props.setForm((current) => ({
                  ...current,
                  contract_id: event.target.value,
                  contract_key: contract?.contract_key || current.contract_key,
                  model_name: contract?.model_name || current.model_name,
                }))
              }}
            >
              <NativeSelectOption value=''>
                {props.t('Use contract key')}
              </NativeSelectOption>
              {props.contracts.map((contract) => (
                <NativeSelectOption
                  key={contract.id}
                  value={String(contract.id)}
                >
                  #{contract.id} {contract.contract_key} / {contract.model_name}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            {selectedContract && (
              <FieldDescription>
                {selectedContract.provider_family} / {selectedContract.version}
              </FieldDescription>
            )}
          </Field>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-3'>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-contract-key'>
                {props.t('Contract Key')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-contract-key'
                value={props.form.contract_key}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    contract_key: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-supplier'>
                {props.t('Supplier ID')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-supplier'
                type='number'
                min='1'
                value={props.form.supplier_id}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    supplier_id: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-channel'>
                {props.t('Channel ID')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-channel'
                type='number'
                min='1'
                value={props.form.channel_id}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    channel_id: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-model'>
                {props.t('Model')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-model'
                value={props.form.model_name}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    model_name: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-tier'>
                {props.t('SLA Tier')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-tier'
                value={props.form.sla_tier}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    sla_tier: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-cache-profile'>
                {props.t('Cache Profile')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-cache-profile'
                value={props.form.cache_profile}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    cache_profile: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldTitle>{props.t('Probe Type')}</FieldTitle>
              <ToggleGroup
                value={[props.form.probe_type]}
                onValueChange={(value) => {
                  const next = value.find(
                    (item) => item !== props.form.probe_type
                  )
                  if (next) {
                    props.setForm((current) => ({
                      ...current,
                      probe_type: next as SlaProbeType,
                    }))
                  }
                }}
                aria-label={props.t('Probe Type')}
                variant='outline'
                size='sm'
                spacing={2}
                className='flex-wrap justify-start'
              >
                <ToggleGroupItem value='admission'>
                  {props.t('Admission')}
                </ToggleGroupItem>
                <ToggleGroupItem value='runtime_light'>
                  {props.t('Runtime Light')}
                </ToggleGroupItem>
                <ToggleGroupItem value='runtime_deep'>
                  {props.t('Runtime Deep')}
                </ToggleGroupItem>
                <ToggleGroupItem value='incident_recheck'>
                  {props.t('Incident Recheck')}
                </ToggleGroupItem>
              </ToggleGroup>
            </Field>
            <Field>
              <FieldTitle>{props.t('Route Mode')}</FieldTitle>
              <ToggleGroup
                value={[props.form.route_mode]}
                onValueChange={(value) => {
                  const next = value.find(
                    (item) => item !== props.form.route_mode
                  )
                  if (next) {
                    props.setForm((current) => ({
                      ...current,
                      route_mode: next as SlaProbeRouteMode,
                    }))
                  }
                }}
                aria-label={props.t('Route Mode')}
                variant='outline'
                size='sm'
                spacing={2}
                className='flex-wrap justify-start'
              >
                <ToggleGroupItem value='direct_upstream'>
                  {props.t('Direct Upstream')}
                </ToggleGroupItem>
                <ToggleGroupItem value='through_token_router'>
                  {props.t('Through Token Router')}
                </ToggleGroupItem>
              </ToggleGroup>
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-3'>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-suite'>
                {props.t('Prompt Suite')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-suite'
                value={props.form.prompt_suite_key}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    prompt_suite_key: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-tokenizer'>
                {props.t('Tokenizer Ref')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-tokenizer'
                value={props.form.tokenizer_ref}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    tokenizer_ref: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-sample'>
                {props.t('Sample Size')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-sample'
                type='number'
                min='1'
                value={props.form.sample_size}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    sample_size: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-repeat'>
                {props.t('Repeat Count')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-repeat'
                type='number'
                min='1'
                value={props.form.repeat_count}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    repeat_count: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-interval'>
                {props.t('Schedule Interval Seconds')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-interval'
                type='number'
                min='0'
                value={props.form.schedule_interval_seconds}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    schedule_interval_seconds: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-jitter'>
                {props.t('Jitter Seconds')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-jitter'
                type='number'
                min='0'
                value={props.form.jitter_seconds}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    jitter_seconds: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-plan-quota'>
                {props.t('Max Probe Quota')}
              </FieldLabel>
              <Input
                id='token-router-sla-plan-quota'
                type='number'
                min='0'
                value={props.form.max_probe_quota}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    max_probe_quota: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            {[
              ['input_profile_json', 'Input Profile JSON'],
              ['output_profile_json', 'Output Profile JSON'],
              ['concurrency_profile_json', 'Concurrency Profile JSON'],
              ['rate_profile_json', 'Rate Profile JSON'],
              ['stream_profile_json', 'Stream Profile JSON'],
              ['error_profile_json', 'Error Profile JSON'],
              ['availability_profile_json', 'Availability Profile JSON'],
            ].map(([key, label]) => (
              <Field key={key}>
                <FieldLabel htmlFor={`token-router-sla-plan-${key}`}>
                  {props.t(label)}
                </FieldLabel>
                <Textarea
                  id={`token-router-sla-plan-${key}`}
                  rows={3}
                  value={
                    props.form[
                      key as keyof SlaProbePlanGenerateFormState
                    ] as string
                  }
                  onChange={(event) =>
                    props.setForm((current) => ({
                      ...current,
                      [key]: event.target.value,
                    }))
                  }
                />
              </Field>
            ))}
          </FieldGroup>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSaving}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving}
          >
            {props.isSaving ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <Plus data-icon='inline-start' />
            )}
            {props.t('Generate')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SlaProbeRunRecordDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  plans: SlaProbePlan[]
  form: SlaProbeRunRecordFormState
  setForm: Dispatch<SetStateAction<SlaProbeRunRecordFormState>>
  isSaving: boolean
  onSubmit: () => void
  t: Translator
}) {
  const selectedPlan = props.plans.find(
    (plan) => String(plan.id) === props.form.plan_id
  )

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[calc(100svh-1rem)] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>{props.t('Record Probe Run')}</DialogTitle>
          <DialogDescription>
            {props.t(
              'Record runner evidence for an existing plan; this does not change supplier status or routing.'
            )}
          </DialogDescription>
        </DialogHeader>

        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='token-router-sla-run-plan'>
              {props.t('Probe Plan')}
            </FieldLabel>
            <NativeSelect
              id='token-router-sla-run-plan'
              className='w-full'
              value={props.form.plan_id}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  plan_id: event.target.value,
                }))
              }
            >
              <NativeSelectOption value=''>
                {props.plans.length > 0
                  ? props.t('Select a probe plan')
                  : props.t('No probe plans available')}
              </NativeSelectOption>
              {props.plans.map((plan) => (
                <NativeSelectOption key={plan.id} value={String(plan.id)}>
                  {slaProbePlanLabel(plan, props.t)}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            {selectedPlan && (
              <FieldDescription>
                {props.t('Contract')} #{selectedPlan.contract_id} /{' '}
                {props.t('Supplier')} #{selectedPlan.supplier_id} /{' '}
                {props.t('Channel')} #{selectedPlan.channel_id || '-'}
              </FieldDescription>
            )}
          </Field>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-sla-run-key'>
                {props.t('Run Key')}
              </FieldLabel>
              <Input
                id='token-router-sla-run-key'
                value={props.form.run_key}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    run_key: event.target.value,
                  }))
                }
              />
              <FieldDescription>
                {props.t('Leave empty to derive from plan and start time.')}
              </FieldDescription>
            </Field>
            <Field>
              <FieldTitle>{props.t('Run Status')}</FieldTitle>
              <ToggleGroup
                value={[props.form.status]}
                onValueChange={(value) => {
                  const next = value.find((item) => item !== props.form.status)
                  if (next) {
                    props.setForm((current) => ({
                      ...current,
                      status: next as SlaProbeRunStatus,
                    }))
                  }
                }}
                aria-label={props.t('Run Status')}
                variant='outline'
                size='sm'
                spacing={2}
                className='flex-wrap justify-start'
              >
                <ToggleGroupItem value='running'>
                  {props.t('Running')}
                </ToggleGroupItem>
                <ToggleGroupItem value='passed'>
                  {props.t('Passed')}
                </ToggleGroupItem>
                <ToggleGroupItem value='failed'>
                  {props.t('Failed')}
                </ToggleGroupItem>
                <ToggleGroupItem value='invalid'>
                  {props.t('Invalid')}
                </ToggleGroupItem>
                <ToggleGroupItem value='cancelled'>
                  {props.t('Cancelled')}
                </ToggleGroupItem>
              </ToggleGroup>
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-run-started'>
                {props.t('Started At')}
              </FieldLabel>
              <Input
                id='token-router-sla-run-started'
                type='datetime-local'
                value={props.form.started_at}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    started_at: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-run-ended'>
                {props.t('Ended At')}
              </FieldLabel>
              <Input
                id='token-router-sla-run-ended'
                type='datetime-local'
                value={props.form.ended_at}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    ended_at: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-runner'>
                {props.t('Runner Version')}
              </FieldLabel>
              <Input
                id='token-router-sla-runner'
                value={props.form.runner_version}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    runner_version: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-git-commit'>
                {props.t('Git Commit')}
              </FieldLabel>
              <Input
                id='token-router-sla-git-commit'
                value={props.form.git_commit}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    git_commit: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-runtime-ref'>
                {props.t('Runtime Ref')}
              </FieldLabel>
              <Input
                id='token-router-sla-runtime-ref'
                value={props.form.runtime_ref}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    runtime_ref: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-endpoint'>
                {props.t('Endpoint')}
              </FieldLabel>
              <Input
                id='token-router-sla-endpoint'
                value={props.form.endpoint}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    endpoint: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>

          <Field>
            <FieldTitle>{props.t('Hard Gate Passed')}</FieldTitle>
            <ToggleGroup
              value={[props.form.hard_gate_passed ? 'true' : 'false']}
              onValueChange={(value) => {
                const current = props.form.hard_gate_passed ? 'true' : 'false'
                const next = value.find((item) => item !== current)
                if (next) {
                  props.setForm((state) => ({
                    ...state,
                    hard_gate_passed: next === 'true',
                  }))
                }
              }}
              aria-label={props.t('Hard Gate Passed')}
              variant='outline'
              size='sm'
              spacing={2}
              className='flex-wrap justify-start'
            >
              <ToggleGroupItem value='true'>
                {props.t('Passed')}
              </ToggleGroupItem>
              <ToggleGroupItem value='false'>
                {props.t('Not Passed')}
              </ToggleGroupItem>
            </ToggleGroup>
          </Field>

          <Field>
            <FieldLabel htmlFor='token-router-sla-summary-json'>
              {props.t('Summary JSON')}
            </FieldLabel>
            <Textarea
              id='token-router-sla-summary-json'
              rows={6}
              value={props.form.summary_json}
              onChange={(event) =>
                props.setForm((current) => ({
                  ...current,
                  summary_json: event.target.value,
                }))
              }
            />
          </Field>

          <FieldGroup className='grid grid-cols-1 gap-4 sm:grid-cols-2'>
            <Field>
              <FieldLabel htmlFor='token-router-sla-soft-warnings'>
                {props.t('Soft Gate Warnings')}
              </FieldLabel>
              <Textarea
                id='token-router-sla-soft-warnings'
                rows={3}
                value={props.form.soft_gate_warnings}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    soft_gate_warnings: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-failure-reasons'>
                {props.t('Failure Reasons')}
              </FieldLabel>
              <Textarea
                id='token-router-sla-failure-reasons'
                rows={3}
                value={props.form.failure_reasons}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    failure_reasons: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-artifact-uri'>
                {props.t('Artifact URI')}
              </FieldLabel>
              <Input
                id='token-router-sla-artifact-uri'
                value={props.form.artifact_uri}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    artifact_uri: event.target.value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='token-router-sla-artifact-sha'>
                {props.t('Artifact SHA256')}
              </FieldLabel>
              <Input
                id='token-router-sla-artifact-sha'
                value={props.form.artifact_sha256}
                onChange={(event) =>
                  props.setForm((current) => ({
                    ...current,
                    artifact_sha256: event.target.value,
                  }))
                }
              />
            </Field>
          </FieldGroup>
        </FieldGroup>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            disabled={props.isSaving}
          >
            {props.t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={props.onSubmit}
            disabled={props.isSaving || !selectedPlan}
          >
            {props.isSaving ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <Pencil data-icon='inline-start' />
            )}
            {props.t('Record')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function useTokenRouterPeriod() {
  const now = Math.floor(Date.now() / 1000)
  const [startInput, setStartInput] = useState(
    formatTimestampForInput(now - 7 * 24 * 3600)
  )
  const [endInput, setEndInput] = useState(formatTimestampForInput(now + 3600))
  const period = useMemo(
    () => ({
      start_timestamp: parseTimestampFromInput(startInput),
      end_timestamp: parseTimestampFromInput(endInput),
    }),
    [startInput, endInput]
  )

  return {
    startInput,
    setStartInput,
    endInput,
    setEndInput,
    period,
  }
}

export function TokenRouter() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<TabValue>('control-tower')
  const [groupBy, setGroupBy] = useState<MarginGroupBy>('supplier')
  const [qualityGroupBy, setQualityGroupBy] =
    useState<QualityGroupBy>('supplier')
  const [decisionStatus, setDecisionStatus] =
    useState<SupplyDecisionStatus>('draft')
  const [opportunityType, setOpportunityType] =
    useState<OpportunityTypeFilter>('all')
  const [opportunityPriority, setOpportunityPriority] =
    useState<OpportunityPriorityFilter>('all')
  const [pricingRecommendationStatus, setPricingRecommendationStatus] =
    useState<PricingRecommendationStatusFilter>('draft')
  const [pricingRecommendationAction, setPricingRecommendationAction] =
    useState<PricingRecommendationActionFilter>('all')
  const [operatingInsightStatus, setOperatingInsightStatus] =
    useState<OperatingInsightStatusFilter>('draft')
  const [operatingInsightSeverity, setOperatingInsightSeverity] =
    useState<OperatingInsightSeverityFilter>('all')
  const [operatingInsightCategory, setOperatingInsightCategory] =
    useState<OperatingInsightCategoryFilter>('all')
  const [actionTrack, setActionTrack] = useState<TrackFilter>('all')
  const [actionStatus, setActionStatus] =
    useState<ActionPlanStatusFilter>('planned')
  const [executionTrack, setExecutionTrack] = useState<TrackFilter>('all')
  const [executionStatus, setExecutionStatus] =
    useState<ExecutionStatusFilter>('recorded')
  const [routingPolicyStatus, setRoutingPolicyStatus] =
    useState<RoutingPolicyStatusFilter>('active')
  const [slaContractStatus, setSlaContractStatus] =
    useState<SlaContractStatusFilter>('all')
  const [slaProbeType, setSlaProbeType] = useState<SlaProbeTypeFilter>('all')
  const [slaProbeRouteMode, setSlaProbeRouteMode] =
    useState<SlaProbeRouteModeFilter>('all')
  const [slaProbeRunStatus, setSlaProbeRunStatus] =
    useState<SlaProbeRunStatusFilter>('all')
  const [scorecardGrade, setScorecardGrade] =
    useState<ScorecardGradeFilter>('all')
  const [evaluationStatus, setEvaluationStatus] =
    useState<EvaluationStatusFilter>('draft')
  const [evaluationRecommendation, setEvaluationRecommendation] =
    useState<EvaluationRecommendationFilter>('all')
  const [evaluationGrade, setEvaluationGrade] =
    useState<ScorecardGradeFilter>('all')
  const [postureStatus, setPostureStatus] =
    useState<PostureStatusFilter>('draft')
  const [postureAction, setPostureAction] = useState<PostureActionFilter>('all')
  const [postureGrade, setPostureGrade] = useState<ScorecardGradeFilter>('all')
  const [subjectType, setSubjectType] =
    useState<SettlementSubjectType>('supplier')
  const [subjectId, setSubjectId] = useState('1')
  const [supplierDialogOpen, setSupplierDialogOpen] = useState(false)
  const [supplierForm, setSupplierForm] = useState<SupplierFormState>(
    DEFAULT_SUPPLIER_FORM
  )
  const [agreementDialogOpen, setAgreementDialogOpen] = useState(false)
  const [agreementForm, setAgreementForm] =
    useState<SupplierAgreementFormState>(DEFAULT_AGREEMENT_FORM)
  const [agreementToDelete, setAgreementToDelete] =
    useState<SupplierAgreement | null>(null)
  const [routePreferenceDialogOpen, setRoutePreferenceDialogOpen] =
    useState(false)
  const [routePreferenceForm, setRoutePreferenceForm] =
    useState<SupplierRoutePreferenceFormState>(DEFAULT_ROUTE_PREFERENCE_FORM)
  const [routePreferenceToDisable, setRoutePreferenceToDisable] =
    useState<SupplierRoutePreference | null>(null)
  const [routingPolicyActivateForm, setRoutingPolicyActivateForm] =
    useState<SupplyRoutingPolicyActivateFormState>(
      DEFAULT_ROUTING_POLICY_ACTIVATE_FORM
    )
  const [costProfileDialogOpen, setCostProfileDialogOpen] = useState(false)
  const [costProfileForm, setCostProfileForm] =
    useState<SupplyCostProfileFormState>(DEFAULT_COST_PROFILE_FORM)
  const [prepaidLotDialogOpen, setPrepaidLotDialogOpen] = useState(false)
  const [prepaidLotForm, setPrepaidLotForm] =
    useState<SupplyPrepaidLotFormState>(DEFAULT_PREPAID_LOT_FORM)
  const [actionPlanStatusDialogOpen, setActionPlanStatusDialogOpen] =
    useState(false)
  const [actionPlanStatusForm, setActionPlanStatusForm] =
    useState<ActionPlanStatusFormState>(DEFAULT_ACTION_PLAN_STATUS_FORM)
  const [actionExecutionRecordDialogOpen, setActionExecutionRecordDialogOpen] =
    useState(false)
  const [actionExecutionRecordForm, setActionExecutionRecordForm] =
    useState<SupplyActionExecutionRecordFormState>(
      DEFAULT_ACTION_EXECUTION_RECORD_FORM
    )
  const [slaContractDialogOpen, setSlaContractDialogOpen] = useState(false)
  const [slaContractForm, setSlaContractForm] =
    useState<SlaContractImportFormState>(DEFAULT_SLA_CONTRACT_IMPORT_FORM)
  const [slaProbePlanDialogOpen, setSlaProbePlanDialogOpen] = useState(false)
  const [slaProbePlanForm, setSlaProbePlanForm] =
    useState<SlaProbePlanGenerateFormState>(DEFAULT_SLA_PROBE_PLAN_FORM)
  const [slaProbeRunDialogOpen, setSlaProbeRunDialogOpen] = useState(false)
  const [slaProbeRunForm, setSlaProbeRunForm] =
    useState<SlaProbeRunRecordFormState>(DEFAULT_SLA_PROBE_RUN_FORM)
  const [policyToDisable, setPolicyToDisable] =
    useState<SupplyRoutingPolicy | null>(null)
  const { startInput, setStartInput, endInput, setEndInput, period } =
    useTokenRouterPeriod()

  const suppliersQuery = useQuery({
    queryKey: ['token-router', 'suppliers'],
    queryFn: () => getSuppliers({ p: 1, page_size: 100 }),
  })
  const agreementsQuery = useQuery({
    queryKey: ['token-router', 'supplier-agreements'],
    queryFn: () => getSupplierAgreements({ p: 1, page_size: 100 }),
  })
  const ledgersQuery = useQuery({
    queryKey: ['token-router', 'usage-ledgers', period],
    queryFn: () =>
      getUsageLedgers({ p: 1, page_size: QUERY_PAGE_SIZE, ...period }),
  })
  const marginQuery = useQuery({
    queryKey: ['token-router', 'margin-summary', groupBy, period],
    queryFn: () => getMarginSummary({ group_by: groupBy, ...period }),
  })
  const qualityQuery = useQuery({
    queryKey: ['token-router', 'quality-summary', qualityGroupBy, period],
    queryFn: () => getQualitySummary({ group_by: qualityGroupBy, ...period }),
  })
  const capacitiesQuery = useQuery({
    queryKey: ['token-router', 'supply-capacities', period],
    queryFn: () => getSupplyCapacities({ p: 1, page_size: 100, ...period }),
  })
  const costProfilesQuery = useQuery({
    queryKey: ['token-router', 'supply-cost-profiles', period],
    queryFn: () => getSupplyCostProfiles({ p: 1, page_size: 100, ...period }),
  })
  const prepaidLotsQuery = useQuery({
    queryKey: ['token-router', 'supply-prepaid-lots', period],
    queryFn: () => getSupplyPrepaidLots({ p: 1, page_size: 100, ...period }),
  })
  const profilesQuery = useQuery({
    queryKey: ['token-router', 'traffic-profiles', period],
    queryFn: () =>
      getTrafficProfiles({ p: 1, page_size: QUERY_PAGE_SIZE, ...period }),
  })
  const forecastsQuery = useQuery({
    queryKey: ['token-router', 'traffic-forecasts', period],
    queryFn: () =>
      getTrafficForecasts({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        source_start_timestamp: period.start_timestamp,
        source_end_timestamp: period.end_timestamp,
      }),
  })
  const decisionsQuery = useQuery({
    queryKey: ['token-router', 'supply-decisions', decisionStatus, period],
    queryFn: () =>
      getSupplyDecisions({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status: decisionStatus,
        ...period,
      }),
  })
  const opportunitiesQuery = useQuery({
    queryKey: [
      'token-router',
      'supply-expansion-opportunities',
      opportunityType,
      opportunityPriority,
      period,
    ],
    queryFn: () =>
      getSupplyExpansionOpportunities({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        opportunity_type:
          opportunityType === 'all' ? undefined : opportunityType,
        priority:
          opportunityPriority === 'all' ? undefined : opportunityPriority,
        ...period,
      }),
  })
  const pricingRecommendationsQuery = useQuery({
    queryKey: [
      'token-router',
      'pricing-recommendations',
      pricingRecommendationStatus,
      pricingRecommendationAction,
      period,
    ],
    queryFn: () =>
      getPricingRecommendations({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status:
          pricingRecommendationStatus === 'all'
            ? undefined
            : pricingRecommendationStatus,
        action:
          pricingRecommendationAction === 'all'
            ? undefined
            : pricingRecommendationAction,
        ...period,
      }),
  })
  const operatingInsightsQuery = useQuery({
    queryKey: [
      'token-router',
      'operating-insights',
      operatingInsightStatus,
      operatingInsightSeverity,
      operatingInsightCategory,
      period,
    ],
    queryFn: () =>
      getOperatingInsights({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status:
          operatingInsightStatus === 'all' ? undefined : operatingInsightStatus,
        severity:
          operatingInsightSeverity === 'all'
            ? undefined
            : operatingInsightSeverity,
        category:
          operatingInsightCategory === 'all'
            ? undefined
            : operatingInsightCategory,
        ...period,
      }),
  })
  const actionPlansQuery = useQuery({
    queryKey: [
      'token-router',
      'supply-action-plans',
      actionTrack,
      actionStatus,
      period,
    ],
    queryFn: () =>
      getSupplyActionPlans({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status: actionStatus === 'all' ? undefined : actionStatus,
        track: actionTrack === 'all' ? undefined : actionTrack,
        ...period,
      }),
  })
  const executionSourcePlansQuery = useQuery({
    queryKey: [
      'token-router',
      'supply-action-plans',
      'execution-source',
      period,
    ],
    queryFn: () =>
      getSupplyActionPlans({
        p: 1,
        page_size: 100,
        status: 'completed',
        ...period,
      }),
  })
  const actionExecutionsQuery = useQuery({
    queryKey: [
      'token-router',
      'supply-action-executions',
      executionTrack,
      executionStatus,
      period,
    ],
    queryFn: () =>
      getSupplyActionExecutions({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        execution_status:
          executionStatus === 'all' ? undefined : executionStatus,
        track: executionTrack === 'all' ? undefined : executionTrack,
        ...period,
      }),
  })
  const routingPoliciesQuery = useQuery({
    queryKey: [
      'token-router',
      'supply-routing-policies',
      routingPolicyStatus,
      period,
    ],
    queryFn: () =>
      getSupplyRoutingPolicies({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status: routingPolicyStatus,
        ...period,
      }),
  })
  const routingSourceExecutionsQuery = useQuery({
    queryKey: ['token-router', 'routing-source-executions', period],
    queryFn: () =>
      getSupplyActionExecutions({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        execution_status: 'recorded',
        track: 'self_hosted',
        ...period,
      }),
  })
  const slaContractsQuery = useQuery({
    queryKey: ['token-router', 'sla-contracts', slaContractStatus, period],
    queryFn: () =>
      getSlaContracts({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status: slaContractStatus === 'all' ? undefined : slaContractStatus,
        ...period,
      }),
  })
  const slaProbePlansQuery = useQuery({
    queryKey: [
      'token-router',
      'sla-probe-plans',
      slaProbeType,
      slaProbeRouteMode,
      period,
    ],
    queryFn: () =>
      getSlaProbePlans({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        probe_type: slaProbeType === 'all' ? undefined : slaProbeType,
        route_mode: slaProbeRouteMode === 'all' ? undefined : slaProbeRouteMode,
        ...period,
      }),
  })
  const slaProbeRunsQuery = useQuery({
    queryKey: [
      'token-router',
      'sla-probe-runs',
      slaProbeRunStatus,
      slaProbeRouteMode,
      period,
    ],
    queryFn: () =>
      getSlaProbeRuns({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status: slaProbeRunStatus === 'all' ? undefined : slaProbeRunStatus,
        route_mode: slaProbeRouteMode === 'all' ? undefined : slaProbeRouteMode,
        ...period,
      }),
  })
  const scorecardsQuery = useQuery({
    queryKey: ['token-router', 'supplier-scorecards', scorecardGrade, period],
    queryFn: () =>
      getSupplierScorecards({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        grade: scorecardGrade === 'all' ? undefined : scorecardGrade,
        ...period,
      }),
  })
  const evaluationsQuery = useQuery({
    queryKey: [
      'token-router',
      'supplier-evaluations',
      evaluationStatus,
      evaluationRecommendation,
      evaluationGrade,
      period,
    ],
    queryFn: () =>
      getSupplierEvaluations({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status: evaluationStatus === 'all' ? undefined : evaluationStatus,
        recommendation:
          evaluationRecommendation === 'all'
            ? undefined
            : evaluationRecommendation,
        grade: evaluationGrade === 'all' ? undefined : evaluationGrade,
        ...period,
      }),
  })
  const postureRecommendationsQuery = useQuery({
    queryKey: [
      'token-router',
      'supplier-posture-recommendations',
      postureStatus,
      postureAction,
      postureGrade,
      period,
    ],
    queryFn: () =>
      getSupplierPostureRecommendations({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status: postureStatus === 'all' ? undefined : postureStatus,
        recommended_action: postureAction === 'all' ? undefined : postureAction,
        grade: postureGrade === 'all' ? undefined : postureGrade,
        ...period,
      }),
  })
  const routePreferencesQuery = useQuery({
    queryKey: ['token-router', 'supplier-route-preferences', 'active'],
    queryFn: () =>
      getSupplierRoutePreferences({
        p: 1,
        page_size: QUERY_PAGE_SIZE,
        status: 'active',
      }),
  })
  const statementsQuery = useQuery({
    queryKey: ['token-router', 'settlement-statements', period],
    queryFn: () =>
      getSettlementStatements({ p: 1, page_size: QUERY_PAGE_SIZE, ...period }),
  })

  const suppliers = suppliersQuery.data?.data?.items ?? EMPTY_SUPPLIERS
  const agreements = agreementsQuery.data?.data?.items ?? EMPTY_AGREEMENTS
  const ledgers = ledgersQuery.data?.data?.items ?? EMPTY_LEDGERS
  const marginRows = marginQuery.data?.data ?? EMPTY_MARGIN_ROWS
  const qualityRows = qualityQuery.data?.data ?? EMPTY_QUALITY_ROWS
  const capacities = capacitiesQuery.data?.data?.items ?? EMPTY_CAPACITIES
  const costProfiles =
    costProfilesQuery.data?.data?.items ?? EMPTY_COST_PROFILES
  const prepaidLots = prepaidLotsQuery.data?.data?.items ?? EMPTY_PREPAID_LOTS
  const profiles = profilesQuery.data?.data?.items ?? EMPTY_PROFILES
  const forecasts = forecastsQuery.data?.data?.items ?? EMPTY_FORECASTS
  const decisions = decisionsQuery.data?.data?.items ?? EMPTY_DECISIONS
  const opportunities =
    opportunitiesQuery.data?.data?.items ?? EMPTY_OPPORTUNITIES
  const pricingRecommendations =
    pricingRecommendationsQuery.data?.data?.items ??
    EMPTY_PRICING_RECOMMENDATIONS
  const operatingInsights =
    operatingInsightsQuery.data?.data?.items ?? EMPTY_OPERATING_INSIGHTS
  const actionPlans = actionPlansQuery.data?.data?.items ?? EMPTY_ACTION_PLANS
  const completedActionPlans =
    executionSourcePlansQuery.data?.data?.items ?? EMPTY_ACTION_PLANS
  const actionExecutions =
    actionExecutionsQuery.data?.data?.items ?? EMPTY_ACTION_EXECUTIONS
  const routingPolicies =
    routingPoliciesQuery.data?.data?.items ?? EMPTY_ROUTING_POLICIES
  const routingSourceExecutions =
    routingSourceExecutionsQuery.data?.data?.items ?? EMPTY_ACTION_EXECUTIONS
  const slaContracts =
    slaContractsQuery.data?.data?.items ?? EMPTY_SLA_CONTRACTS
  const slaProbePlans =
    slaProbePlansQuery.data?.data?.items ?? EMPTY_SLA_PROBE_PLANS
  const slaProbeRuns =
    slaProbeRunsQuery.data?.data?.items ?? EMPTY_SLA_PROBE_RUNS
  const scorecards = scorecardsQuery.data?.data?.items ?? EMPTY_SCORECARDS
  const evaluations = evaluationsQuery.data?.data?.items ?? EMPTY_EVALUATIONS
  const postureRecommendations =
    postureRecommendationsQuery.data?.data?.items ??
    EMPTY_POSTURE_RECOMMENDATIONS
  const routePreferences =
    routePreferencesQuery.data?.data?.items ?? EMPTY_ROUTE_PREFERENCES
  const statements = statementsQuery.data?.data?.items ?? EMPTY_STATEMENTS

  const totals = useMemo(
    () =>
      marginRows.reduce(
        (acc, row) => {
          acc.requests += row.total_requests
          acc.sell += row.total_sell_quota
          acc.cost += row.total_cost_quota
          acc.profit += row.gross_profit_quota
          acc.prompt += row.total_prompt_tokens
          acc.cached += row.total_cached_tokens
          acc.completion += row.total_completion_tokens
          acc.cacheHits += row.cache_hit_count
          return acc
        },
        {
          requests: 0,
          sell: 0,
          cost: 0,
          profit: 0,
          prompt: 0,
          cached: 0,
          completion: 0,
          cacheHits: 0,
        }
      ),
    [marginRows]
  )
  const cacheHitRate =
    totals.requests > 0 ? totals.cacheHits / totals.requests : 0
  const capacityTotals = useMemo(
    () =>
      capacities.reduce(
        (acc, row) => {
          acc.capacity += row.capacity_tokens
          acc.used += row.used_tokens
          acc.headroom += row.headroom_tokens
          return acc
        },
        { capacity: 0, used: 0, headroom: 0 }
      ),
    [capacities]
  )
  const capacityUtilization =
    capacityTotals.capacity > 0
      ? capacityTotals.used / capacityTotals.capacity
      : 0
  const costProfileTotals = useMemo(
    () =>
      costProfiles.reduce(
        (acc, profile) => {
          acc.capacity += profile.capacity_tokens
          acc.fixedCost += profile.fixed_cost_quota
          acc.unitCost += profile.amortized_unit_cost_quota
          return acc
        },
        { capacity: 0, fixedCost: 0, unitCost: 0 }
      ),
    [costProfiles]
  )
  const averageCostProfileUnitCost =
    costProfiles.length > 0
      ? costProfileTotals.unitCost / costProfiles.length
      : 0
  const prepaidLotTotals = useMemo(
    () =>
      prepaidLots.reduce(
        (acc, lot) => {
          acc.purchased += lot.purchased_tokens
          acc.drawdown += lot.drawdown_tokens
          acc.remaining += lot.remaining_tokens
          acc.totalCost += lot.total_cost_quota
          acc.drawdownRate += lot.drawdown_rate
          return acc
        },
        {
          purchased: 0,
          drawdown: 0,
          remaining: 0,
          totalCost: 0,
          drawdownRate: 0,
        }
      ),
    [prepaidLots]
  )
  const averagePrepaidLotDrawdownRate =
    prepaidLots.length > 0
      ? prepaidLotTotals.drawdownRate / prepaidLots.length
      : 0
  const profileTotals = useMemo(
    () =>
      profiles.reduce(
        (acc, profile) => {
          acc.demand += profile.demand_tokens
          acc.cached += profile.total_cached_tokens
          acc.headroom += profile.supply_headroom_tokens
          acc.profit += profile.gross_profit_quota
          return acc
        },
        { demand: 0, cached: 0, headroom: 0, profit: 0 }
      ),
    [profiles]
  )
  const forecastTotals = useMemo(
    () =>
      forecasts.reduce(
        (acc, forecast) => {
          acc.demand += forecast.forecast_demand_tokens
          acc.gap += forecast.forecast_gap_tokens
          acc.confidence += forecast.confidence
          return acc
        },
        { demand: 0, gap: 0, confidence: 0 }
      ),
    [forecasts]
  )
  const averageForecastConfidence =
    forecasts.length > 0 ? forecastTotals.confidence / forecasts.length : 0
  const decisionTotals = useMemo(
    () =>
      decisions.reduce(
        (acc, decision) => {
          acc.recommended += decision.recommended_capacity
          acc.gap += decision.gap_tokens
          acc.roi += decision.roi_score
          return acc
        },
        { recommended: 0, gap: 0, roi: 0 }
      ),
    [decisions]
  )
  const opportunityTotals = useMemo(
    () =>
      opportunities.reduce(
        (acc, opportunity) => {
          acc.recommended += opportunity.recommended_capacity
          acc.rank += opportunity.rank_score
          acc.savings += opportunity.self_hosted_savings_quota || 0
          if (opportunity.priority === 'action') {
            acc.action += 1
          }
          return acc
        },
        { recommended: 0, rank: 0, action: 0, savings: 0 }
      ),
    [opportunities]
  )
  const pricingRecommendationTotals = useMemo(
    () =>
      pricingRecommendations.reduce(
        (acc, recommendation) => {
          acc.recommendedUnitPrice +=
            recommendation.recommended_unit_price_quota
          if (recommendation.status === 'draft') {
            acc.draft += 1
          }
          if (recommendation.action === 'raise_price') {
            acc.raisePrice += 1
          }
          if (recommendation.action === 'share_savings') {
            acc.shareSavings += 1
          }
          return acc
        },
        {
          draft: 0,
          raisePrice: 0,
          shareSavings: 0,
          recommendedUnitPrice: 0,
        }
      ),
    [pricingRecommendations]
  )
  const averageRecommendedUnitPrice =
    pricingRecommendations.length > 0
      ? pricingRecommendationTotals.recommendedUnitPrice /
        pricingRecommendations.length
      : 0
  const operatingInsightTotals = useMemo(
    () =>
      operatingInsights.reduce(
        (acc, insight) => {
          if (insight.severity === 'action') {
            acc.action += 1
          }
          if (insight.severity === 'watch') {
            acc.watch += 1
          }
          if (insight.category === 'cache_efficiency') {
            acc.cacheEfficiency += 1
          }
          if (insight.status === 'acknowledged') {
            acc.acknowledged += 1
          }
          return acc
        },
        { action: 0, watch: 0, cacheEfficiency: 0, acknowledged: 0 }
      ),
    [operatingInsights]
  )
  const actionPlanTotals = useMemo(
    () =>
      actionPlans.reduce(
        (acc, plan) => {
          acc.recommended += plan.recommended_capacity
          acc.gap += plan.gap_tokens
          acc.roi += plan.roi_score
          return acc
        },
        { recommended: 0, gap: 0, roi: 0 }
      ),
    [actionPlans]
  )
  const actionExecutionTotals = useMemo(
    () =>
      actionExecutions.reduce(
        (acc, execution) => {
          acc.actual += execution.actual_capacity_tokens
          acc.recommended += execution.recommended_capacity
          acc.drawdown += execution.drawdown_tokens
          acc.remaining += execution.remaining_tokens
          acc.unitCost += execution.unit_cost_quota
          return acc
        },
        { actual: 0, recommended: 0, drawdown: 0, remaining: 0, unitCost: 0 }
      ),
    [actionExecutions]
  )
  const averageExecutionUnitCost =
    actionExecutions.length > 0
      ? actionExecutionTotals.unitCost / actionExecutions.length
      : 0
  const routingPolicyTotals = useMemo(
    () =>
      routingPolicies.reduce(
        (acc, policy) => {
          if (policy.status === 'active') {
            acc.active += 1
          }
          if (policy.status === 'disabled') {
            acc.disabled += 1
          }
          if (policy.track === 'self_hosted') {
            acc.selfHosted += 1
          }
          return acc
        },
        { active: 0, disabled: 0, selfHosted: 0 }
      ),
    [routingPolicies]
  )
  const policyByExecutionId = useMemo(() => {
    const map = new Map<number, SupplyRoutingPolicy>()
    for (const policy of routingPolicies) {
      map.set(policy.supply_action_execution_id, policy)
    }
    return map
  }, [routingPolicies])
  const slaEvidenceTotals = useMemo(
    () => ({
      activeContracts: slaContracts.filter(
        (contract) => contract.status === 'active'
      ).length,
      admissionPlans: slaProbePlans.filter(
        (plan) => plan.probe_type === 'admission'
      ).length,
      passedRuns: slaProbeRuns.filter((run) => run.status === 'passed').length,
      failedRuns: slaProbeRuns.filter((run) => run.status === 'failed').length,
    }),
    [slaContracts, slaProbePlans, slaProbeRuns]
  )
  const scorecardTotals = useMemo(
    () =>
      scorecards.reduce(
        (acc, scorecard) => {
          acc.score += scorecard.score
          acc.headroom += scorecard.supply_headroom_tokens
          if (scorecard.grade === 'A' || scorecard.grade === 'B') {
            acc.strong += 1
          }
          return acc
        },
        { score: 0, headroom: 0, strong: 0 }
      ),
    [scorecards]
  )
  const averageScore =
    scorecards.length > 0 ? scorecardTotals.score / scorecards.length : 0
  const evaluationTotals = useMemo(
    () =>
      evaluations.reduce(
        (acc, evaluation) => {
          acc.score += evaluation.score
          if (evaluation.status === 'draft') {
            acc.draft += 1
          }
          if (evaluation.status === 'approved') {
            acc.approved += 1
          }
          if (evaluation.recommendation === 'admit') {
            acc.admit += 1
          }
          return acc
        },
        { score: 0, draft: 0, approved: 0, admit: 0 }
      ),
    [evaluations]
  )
  const averageEvaluationScore =
    evaluations.length > 0 ? evaluationTotals.score / evaluations.length : 0
  const postureTotals = useMemo(
    () =>
      postureRecommendations.reduce(
        (acc, recommendation) => {
          acc.score += recommendation.score
          if (recommendation.status === 'draft') {
            acc.draft += 1
          }
          if (recommendation.status === 'applied') {
            acc.applied += 1
          }
          if (recommendation.recommended_action === 'disable') {
            acc.disable += 1
          }
          return acc
        },
        { score: 0, draft: 0, applied: 0, disable: 0 }
      ),
    [postureRecommendations]
  )
  const averagePostureScore =
    postureRecommendations.length > 0
      ? postureTotals.score / postureRecommendations.length
      : 0
  const routePreferenceByRecommendationId = useMemo(() => {
    const preferences = new Map<number, SupplierRoutePreference>()
    for (const preference of routePreferences) {
      if (preference.source_posture_recommendation_id > 0) {
        preferences.set(preference.source_posture_recommendation_id, preference)
      }
    }
    return preferences
  }, [routePreferences])
  const enabledSuppliers = useMemo(
    () => suppliers.filter((supplier) => supplier.status === 1),
    [suppliers]
  )
  const selfHostedSuppliers = useMemo(
    () => suppliers.filter((supplier) => supplier.type === 'self_hosted'),
    [suppliers]
  )
  const selfOperatedSuppliers = useMemo(
    () => suppliers.filter((supplier) => supplier.type === 'self_operated'),
    [suppliers]
  )

  const openSupplierDialog = (supplier?: Supplier) => {
    setSupplierForm(supplier ? supplierToForm(supplier) : DEFAULT_SUPPLIER_FORM)
    setSupplierDialogOpen(true)
  }

  const openAgreementDialog = (agreement?: SupplierAgreement) => {
    setAgreementForm(
      agreement
        ? agreementToForm(agreement)
        : {
            ...DEFAULT_AGREEMENT_FORM,
            supplier_id: suppliers[0]?.id ? String(suppliers[0].id) : '',
          }
    )
    setAgreementDialogOpen(true)
  }

  const openRoutePreferenceDialog = (preference?: SupplierRoutePreference) => {
    setRoutePreferenceForm(
      supplierRoutePreferenceToForm(preference ?? null, enabledSuppliers[0]?.id)
    )
    setRoutePreferenceDialogOpen(true)
  }

  const closeRoutePreferenceDialog = (open: boolean) => {
    setRoutePreferenceDialogOpen(open)
    if (!open) {
      setRoutePreferenceForm(DEFAULT_ROUTE_PREFERENCE_FORM)
    }
  }

  const openRoutingPolicyActivateDialog = (
    execution: SupplyActionExecution,
    existingPolicy?: SupplyRoutingPolicy
  ) => {
    setRoutingPolicyActivateForm({
      execution,
      traffic_percent: String(
        existingPolicy ? routingPolicyTrafficPercent(existingPolicy) : 100
      ),
      operator_note: existingPolicy
        ? 'reactivated from dashboard'
        : 'activated from dashboard',
    })
  }

  const closeRoutingPolicyActivateDialog = (open: boolean) => {
    if (!open) {
      setRoutingPolicyActivateForm(DEFAULT_ROUTING_POLICY_ACTIVATE_FORM)
    }
  }

  const openCostProfileDialog = () => {
    const supplier = selfHostedSuppliers[0]
    const referenceOpportunity =
      opportunities.find(
        (opportunity) => opportunity.opportunity_type === 'self_hosted_cache'
      ) ?? opportunities[0]
    const referenceCapacity = capacities.find(
      (capacity) => capacity.supplier_id === supplier?.id
    )

    setCostProfileForm({
      ...DEFAULT_COST_PROFILE_FORM,
      supplier_id: supplier?.id ? String(supplier.id) : '',
      supply_node:
        referenceCapacity?.supply_node || supplier?.name || 'gb10-4t',
      model_name:
        referenceOpportunity?.model_name || referenceCapacity?.model_name || '',
      period_start:
        period.start_timestamp > 0
          ? formatTimestampForInput(period.start_timestamp)
          : '',
      period_end:
        period.end_timestamp > 0
          ? formatTimestampForInput(period.end_timestamp)
          : '',
      observed_at: formatTimestampForInput(Math.floor(Date.now() / 1000)),
      source_ref: '',
    })
    setCostProfileDialogOpen(true)
  }

  const closeCostProfileDialog = (open: boolean) => {
    setCostProfileDialogOpen(open)
    if (!open) {
      setCostProfileForm(DEFAULT_COST_PROFILE_FORM)
    }
  }

  const openPrepaidLotDialog = () => {
    const supplier = selfOperatedSuppliers[0]
    const referenceCapacity =
      capacities.find((capacity) => capacity.supplier_id === supplier?.id) ??
      capacities[0]
    const referenceProfile = profiles[0]

    setPrepaidLotForm({
      ...DEFAULT_PREPAID_LOT_FORM,
      supplier_id: supplier?.id ? String(supplier.id) : '',
      supply_node:
        referenceCapacity?.supply_node || supplier?.name || 'gb10-4t',
      model_name:
        referenceCapacity?.model_name || referenceProfile?.model_name || '',
      period_start:
        period.start_timestamp > 0
          ? formatTimestampForInput(period.start_timestamp)
          : '',
      period_end:
        period.end_timestamp > 0
          ? formatTimestampForInput(period.end_timestamp)
          : '',
      observed_at: formatTimestampForInput(Math.floor(Date.now() / 1000)),
      source_ref: '',
    })
    setPrepaidLotDialogOpen(true)
  }

  const closePrepaidLotDialog = (open: boolean) => {
    setPrepaidLotDialogOpen(open)
    if (!open) {
      setPrepaidLotForm(DEFAULT_PREPAID_LOT_FORM)
    }
  }

  const openActionPlanStatusDialog = (plan: SupplyActionPlan) => {
    const nextStatus = nextActionPlanStatuses(plan.status)[0]
    if (!nextStatus) return
    setActionPlanStatusForm({
      plan,
      status: nextStatus,
      operator_note: plan.operator_note || '',
    })
    setActionPlanStatusDialogOpen(true)
  }

  const closeActionPlanStatusDialog = (open: boolean) => {
    setActionPlanStatusDialogOpen(open)
    if (!open) {
      setActionPlanStatusForm(DEFAULT_ACTION_PLAN_STATUS_FORM)
    }
  }

  const openActionExecutionRecordDialog = (plan?: SupplyActionPlan) => {
    const sourcePlan = plan ?? completedActionPlans[0]
    setActionExecutionRecordForm(actionExecutionRecordFormFromPlan(sourcePlan))
    setActionExecutionRecordDialogOpen(true)
  }

  const closeActionExecutionRecordDialog = (open: boolean) => {
    setActionExecutionRecordDialogOpen(open)
    if (!open) {
      setActionExecutionRecordForm(DEFAULT_ACTION_EXECUTION_RECORD_FORM)
    }
  }

  const openSlaContractImportDialog = () => {
    setSlaContractForm(DEFAULT_SLA_CONTRACT_IMPORT_FORM)
    setSlaContractDialogOpen(true)
  }

  const openSlaProbePlanDialog = (contract?: SlaContract) => {
    const sourceContract = contract ?? slaContracts[0]
    setSlaProbePlanForm({
      ...DEFAULT_SLA_PROBE_PLAN_FORM,
      contract_id: sourceContract?.id ? String(sourceContract.id) : '',
      contract_key: sourceContract?.contract_key || '',
      model_name: sourceContract?.model_name || '',
      supplier_id: suppliers[0]?.id ? String(suppliers[0].id) : '',
    })
    setSlaProbePlanDialogOpen(true)
  }

  const closeSlaProbePlanDialog = (open: boolean) => {
    setSlaProbePlanDialogOpen(open)
    if (!open) {
      setSlaProbePlanForm(DEFAULT_SLA_PROBE_PLAN_FORM)
    }
  }

  const openSlaProbeRunDialog = (plan?: SlaProbePlan) => {
    const sourcePlan = plan ?? slaProbePlans[0]
    setSlaProbeRunForm({
      ...DEFAULT_SLA_PROBE_RUN_FORM,
      plan_id: sourcePlan?.id ? String(sourcePlan.id) : '',
    })
    setSlaProbeRunDialogOpen(true)
  }

  const closeSlaProbeRunDialog = (open: boolean) => {
    setSlaProbeRunDialogOpen(open)
    if (!open) {
      setSlaProbeRunForm(DEFAULT_SLA_PROBE_RUN_FORM)
    }
  }

  const saveSupplier = useMutation({
    mutationFn: async () => {
      const input = buildSupplierInput(supplierForm, t)
      const result = input.id
        ? await updateSupplier(input)
        : await createSupplier(input)
      if (!result.success) {
        throw new Error(result.message || t('Failed to save supplier'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Supplier saved'))
      setSupplierDialogOpen(false)
      void queryClient.invalidateQueries({ queryKey: ['token-router'] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const saveAgreement = useMutation({
    mutationFn: async () => {
      const input = buildAgreementInput(agreementForm, t)
      const result = input.id
        ? await updateSupplierAgreement(input)
        : await createSupplierAgreement(input)
      if (!result.success) {
        throw new Error(result.message || t('Failed to save agreement'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Supplier agreement saved'))
      setAgreementDialogOpen(false)
      void queryClient.invalidateQueries({ queryKey: ['token-router'] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const removeAgreement = useMutation({
    mutationFn: async (id: number) => {
      const result = await deleteSupplierAgreement(id)
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to delete supplier agreement')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Supplier agreement deleted'))
      setAgreementToDelete(null)
      void queryClient.invalidateQueries({ queryKey: ['token-router'] })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const recordCostProfile = useMutation({
    mutationFn: async () => {
      const input = buildSupplyCostProfileInput(costProfileForm, t)
      const result = await recordSupplyCostProfile(input)
      if (!result.success) {
        throw new Error(result.message || t('Failed to record cost profile'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Cost profile recorded'))
      closeCostProfileDialog(false)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-cost-profiles'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-expansion-opportunities'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const recordPrepaidLot = useMutation({
    mutationFn: async () => {
      const input = buildSupplyPrepaidLotInput(prepaidLotForm, t)
      const result = await recordSupplyPrepaidLot(input)
      if (!result.success) {
        throw new Error(result.message || t('Failed to record prepaid lot'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Prepaid lot recorded'))
      closePrepaidLotDialog(false)
      setActiveTab('prepaid-lots')
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-prepaid-lots'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const refreshPrepaidLotUsage = useMutation({
    mutationFn: async () => {
      const input: SupplyPrepaidLotUsageRefreshInput = {
        ...period,
      }
      const result = await refreshSupplyPrepaidLotUsage(input)
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to refresh prepaid lot drawdown')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Prepaid lot drawdown refreshed'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-prepaid-lots'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateStatement = useMutation({
    mutationFn: async () => {
      const id = Number(subjectId)
      if (!Number.isFinite(id) || id <= 0) {
        throw new Error(t('Enter a valid subject ID'))
      }
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid settlement period'))
      }
      return generateSettlementStatement({
        subject_type: subjectType,
        supplier_id: subjectType === 'supplier' ? id : 0,
        user_id: subjectType === 'user' ? id : 0,
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to generate statement'))
        return
      }
      toast.success(t('Settlement statement generated'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'settlement-statements'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'margin-summary'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateProfiles = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid profile period'))
      }
      return generateTrafficProfiles({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to generate profiles'))
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0 ? t('Traffic profiles generated') : t('No profiles generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'traffic-profiles'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateForecasts = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid forecast source period'))
      }
      return generateTrafficForecasts({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to generate forecasts'))
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0
          ? t('Traffic forecasts generated')
          : t('No forecasts generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'traffic-forecasts'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateDecisions = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid decision period'))
      }
      return generateSupplyDecisions({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to generate decisions'))
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0
          ? t('Supply decisions generated')
          : t('No decisions generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-decisions'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateOpportunities = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid opportunity period'))
      }
      return generateSupplyExpansionOpportunities({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to generate opportunities'))
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0
          ? t('Supply opportunities generated')
          : t('No opportunities generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-expansion-opportunities'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generatePricingRecommendationsMutation = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid pricing period'))
      }
      return generatePricingRecommendations({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(
          result.message || t('Failed to generate pricing recommendations')
        )
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0
          ? t('Pricing recommendations generated')
          : t('No pricing recommendations generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'pricing-recommendations'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateOperatingInsightsMutation = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid operating insight period'))
      }
      return generateOperatingInsights({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(
          result.message || t('Failed to generate operating insights')
        )
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0
          ? t('Operating insights generated')
          : t('No operating insights generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'operating-insights'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateActionPlans = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid action plan period'))
      }
      return generateSupplyActionPlans({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
        track: actionTrack === 'all' ? undefined : actionTrack,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to generate action plans'))
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0 ? t('Action plans generated') : t('No action plans generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-action-plans'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const updateActionPlanStatus = useMutation({
    mutationFn: async () => {
      const plan = actionPlanStatusForm.plan
      if (!plan) {
        throw new Error(t('Select an action plan first'))
      }
      const result = await updateSupplyActionPlanStatus(plan.id, {
        status: actionPlanStatusForm.status,
        operator_note: actionPlanStatusForm.operator_note,
      })
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to update action plan status')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Action plan status updated'))
      closeActionPlanStatusDialog(false)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-action-plans'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const recordActionExecution = useMutation({
    mutationFn: async () => {
      const input = buildSupplyActionExecutionRecordInput(
        actionExecutionRecordForm,
        t
      )
      const result = await recordSupplyActionExecution(input)
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to record supply action execution')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Supply action execution recorded'))
      closeActionExecutionRecordDialog(false)
      setActiveTab('executions')
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-action-executions'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'routing-source-executions'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-routing-policies'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const refreshActionExecutionUsage = useMutation({
    mutationFn: async () => {
      const result = await refreshSupplyActionExecutionUsage({
        execution_status:
          executionStatus === 'all' ? undefined : executionStatus,
        track: executionTrack === 'all' ? undefined : executionTrack,
        ...period,
      })
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to refresh execution drawdown')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Execution drawdown refreshed'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-action-executions'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'routing-source-executions'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const importSlaContractMutation = useMutation({
    mutationFn: async () => {
      const input = buildSlaContractImportInput(slaContractForm, t)
      const result = await importSlaContract(input)
      if (!result.success) {
        throw new Error(result.message || t('Failed to import SLA contract'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('SLA contract imported'))
      setSlaContractDialogOpen(false)
      setSlaContractForm(DEFAULT_SLA_CONTRACT_IMPORT_FORM)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'sla-contracts'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateSlaProbePlanMutation = useMutation({
    mutationFn: async () => {
      const input = buildSlaProbePlanGenerateInput(slaProbePlanForm, t)
      const result = await generateSlaProbePlan(input)
      if (!result.success) {
        throw new Error(result.message || t('Failed to generate probe plan'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('SLA probe plan generated'))
      closeSlaProbePlanDialog(false)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'sla-probe-plans'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const recordSlaProbeRunMutation = useMutation({
    mutationFn: async () => {
      const input = buildSlaProbeRunRecordInput(slaProbeRunForm, t)
      const result = await recordSlaProbeRun(input)
      if (!result.success) {
        throw new Error(result.message || t('Failed to record probe run'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('SLA probe run recorded'))
      closeSlaProbeRunDialog(false)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'sla-probe-runs'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const activateRoutingPolicy = useMutation({
    mutationFn: async () => {
      const execution = routingPolicyActivateForm.execution
      if (!execution) {
        throw new Error(t('Select a recorded execution first'))
      }
      if (execution.supplier_id <= 0 || execution.channel_id <= 0) {
        throw new Error(
          t('Execution needs supplier and channel before routing')
        )
      }
      const trafficPercent = parseRoutingPolicyTrafficPercent(
        routingPolicyActivateForm.traffic_percent,
        t
      )
      const result = await activateSupplyRoutingPolicy({
        supply_action_execution_id: execution.id,
        traffic_percent: trafficPercent,
        operator_note: routingPolicyActivateForm.operator_note.trim(),
      })
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to activate routing policy')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Routing policy activated'))
      closeRoutingPolicyActivateDialog(false)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-routing-policies'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'routing-source-executions'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const disableRoutingPolicy = useMutation({
    mutationFn: async (policy: SupplyRoutingPolicy) => {
      const result = await disableSupplyRoutingPolicy(policy.id, {
        operator_note: 'disabled from dashboard',
      })
      if (!result.success) {
        throw new Error(result.message || t('Failed to disable routing policy'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Routing policy disabled'))
      setPolicyToDisable(null)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-routing-policies'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'routing-source-executions'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'usage-ledgers'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateScorecards = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid scorecard period'))
      }
      return generateSupplierScorecards({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to generate scorecards'))
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0
          ? t('Supplier scorecards generated')
          : t('No scorecards generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-scorecards'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generateEvaluations = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid evaluation period'))
      }
      return generateSupplierEvaluations({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to generate evaluations'))
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0
          ? t('Supplier evaluations generated')
          : t('No evaluations generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-evaluations'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const approveEvaluation = useMutation({
    mutationFn: async (id: number) => {
      const result = await approveSupplierEvaluation(id)
      if (!result.success) {
        throw new Error(result.message || t('Failed to approve evaluation'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Evaluation approved'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-evaluations'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const rejectEvaluation = useMutation({
    mutationFn: async (id: number) => {
      const result = await rejectSupplierEvaluation(id)
      if (!result.success) {
        throw new Error(result.message || t('Failed to reject evaluation'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Evaluation rejected'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-evaluations'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const applyEvaluation = useMutation({
    mutationFn: async (id: number) => {
      const result = await applySupplierEvaluation(id, {
        operator_note: t('applied approved supplier evaluation from dashboard'),
      })
      if (!result.success) {
        throw new Error(result.message || t('Failed to apply evaluation'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Evaluation applied'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-evaluations'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'suppliers'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const generatePostureRecommendations = useMutation({
    mutationFn: async () => {
      if (
        period.start_timestamp <= 0 ||
        period.end_timestamp <= 0 ||
        period.end_timestamp < period.start_timestamp
      ) {
        throw new Error(t('Enter a valid posture recommendation period'))
      }
      return generateSupplierPostureRecommendations({
        period_start: period.start_timestamp,
        period_end: period.end_timestamp,
      })
    },
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(
          result.message || t('Failed to generate posture recommendations')
        )
        return
      }
      const count = result.data?.length ?? 0
      toast.success(
        count > 0
          ? t('Supplier posture recommendations generated')
          : t('No posture recommendations generated')
      )
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-posture-recommendations'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-route-preferences'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const approvePostureRecommendation = useMutation({
    mutationFn: async (id: number) => {
      const result = await approveSupplierPostureRecommendation(id, {
        review_note: t(
          'approved supplier posture recommendation from dashboard'
        ),
      })
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to approve posture recommendation')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Posture recommendation approved'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-posture-recommendations'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-route-preferences'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const rejectPostureRecommendation = useMutation({
    mutationFn: async (id: number) => {
      const result = await rejectSupplierPostureRecommendation(id, {
        review_note: t(
          'rejected supplier posture recommendation from dashboard'
        ),
      })
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to reject posture recommendation')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Posture recommendation rejected'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-posture-recommendations'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-route-preferences'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const applyPostureRecommendation = useMutation({
    mutationFn: async (id: number) => {
      const result = await applySupplierPostureRecommendation(id, {
        operator_note: t(
          'applied supplier posture recommendation from dashboard'
        ),
      })
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to apply posture recommendation')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Posture recommendation applied'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-posture-recommendations'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-route-preferences'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'suppliers'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const activateRoutePreference = useMutation({
    mutationFn: async () => {
      const supplierId = Number(routePreferenceForm.supplier_id)
      const weightPercent = Number(routePreferenceForm.weight_percent)
      const reason = routePreferenceForm.reason.trim()
      const effectiveFrom = routePreferenceForm.effective_from.trim()
        ? parseTimestampFromInput(routePreferenceForm.effective_from)
        : 0
      const effectiveTo = routePreferenceForm.effective_to.trim()
        ? parseTimestampFromInput(routePreferenceForm.effective_to)
        : 0

      if (!Number.isInteger(supplierId) || supplierId <= 0) {
        throw new Error(t('Select an enabled supplier first'))
      }
      if (
        !Number.isInteger(weightPercent) ||
        weightPercent < 1 ||
        weightPercent > 200
      ) {
        throw new Error(t('Enter a valid route weight percent'))
      }
      if (!reason) {
        throw new Error(t('Route preference reason is required'))
      }
      if (
        !Number.isFinite(effectiveFrom) ||
        !Number.isFinite(effectiveTo) ||
        effectiveFrom < 0 ||
        effectiveTo < 0 ||
        (effectiveFrom > 0 && effectiveTo > 0 && effectiveTo <= effectiveFrom)
      ) {
        throw new Error(t('Enter a valid route preference effective window'))
      }

      const input: SupplierRoutePreferenceActivateInput = {
        supplier_id: supplierId,
        weight_percent: weightPercent,
        reason,
        effective_from: effectiveFrom,
        effective_to: effectiveTo,
        operator_note: routePreferenceForm.operator_note.trim(),
      }
      const result = await activateSupplierRoutePreference(input)
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to activate route preference')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Route preference activated'))
      setRoutePreferenceDialogOpen(false)
      setRoutePreferenceForm(DEFAULT_ROUTE_PREFERENCE_FORM)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-route-preferences'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-posture-recommendations'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const disableRoutePreference = useMutation({
    mutationFn: async (preference: SupplierRoutePreference) => {
      const result = await disableSupplierRoutePreference(
        preference.supplier_id,
        {
          operator_note: t('disabled supplier route preference from dashboard'),
        }
      )
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to disable route preference')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Route preference disabled'))
      setRoutePreferenceToDisable(null)
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-route-preferences'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supplier-posture-recommendations'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const approveDecision = useMutation({
    mutationFn: async (id: number) => {
      const result = await approveSupplyDecision(id)
      if (!result.success) {
        throw new Error(result.message || t('Failed to approve decision'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Decision approved'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-decisions'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const rejectDecision = useMutation({
    mutationFn: async (id: number) => {
      const result = await rejectSupplyDecision(id)
      if (!result.success) {
        throw new Error(result.message || t('Failed to reject decision'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Decision rejected'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-decisions'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const approvePricing = useMutation({
    mutationFn: async (id: number) => {
      const result = await approvePricingRecommendation(id)
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to approve pricing recommendation')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Pricing recommendation approved'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'pricing-recommendations'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const rejectPricing = useMutation({
    mutationFn: async (id: number) => {
      const result = await rejectPricingRecommendation(id)
      if (!result.success) {
        throw new Error(
          result.message || t('Failed to reject pricing recommendation')
        )
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Pricing recommendation rejected'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'pricing-recommendations'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const acknowledgeInsight = useMutation({
    mutationFn: async (id: number) => {
      const result = await acknowledgeOperatingInsight(id)
      if (!result.success) {
        throw new Error(result.message || t('Failed to acknowledge insight'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Operating insight acknowledged'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'operating-insights'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const dismissInsight = useMutation({
    mutationFn: async (id: number) => {
      const result = await dismissOperatingInsight(id)
      if (!result.success) {
        throw new Error(result.message || t('Failed to dismiss insight'))
      }
      return result
    },
    onSuccess: () => {
      toast.success(t('Operating insight dismissed'))
      void queryClient.invalidateQueries({
        queryKey: ['token-router', 'operating-insights'],
      })
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : t('Request failed'))
    },
  })

  const renderEvaluationActions = (
    evaluation: SupplierEvaluation
  ): ReactNode => {
    if (evaluation.status === 'draft') {
      return (
        <div className='flex flex-wrap gap-2'>
          <Button
            variant='outline'
            size='sm'
            disabled={approveEvaluation.isPending || rejectEvaluation.isPending}
            onClick={() => approveEvaluation.mutate(evaluation.id)}
          >
            {t('Approve')}
          </Button>
          <Button
            variant='outline'
            size='sm'
            disabled={approveEvaluation.isPending || rejectEvaluation.isPending}
            onClick={() => rejectEvaluation.mutate(evaluation.id)}
          >
            {t('Reject')}
          </Button>
        </div>
      )
    }

    if (evaluation.status === 'approved' && evaluation.applied_at <= 0) {
      return (
        <Button
          variant='outline'
          size='sm'
          disabled={applyEvaluation.isPending}
          onClick={() => applyEvaluation.mutate(evaluation.id)}
        >
          {applyEvaluation.isPending ? (
            <Spinner data-icon='inline-start' />
          ) : (
            <Check data-icon='inline-start' />
          )}
          {t('Apply')}
        </Button>
      )
    }

    if (evaluation.applied_at > 0) {
      return <Badge variant='outline'>{t('Applied')}</Badge>
    }

    return '-'
  }

  const renderPostureActions = (
    recommendation: SupplierPostureRecommendation
  ): ReactNode => {
    if (recommendation.status === 'draft') {
      return (
        <div className='flex flex-wrap gap-2'>
          <Button
            variant='outline'
            size='sm'
            disabled={
              approvePostureRecommendation.isPending ||
              rejectPostureRecommendation.isPending
            }
            onClick={() =>
              approvePostureRecommendation.mutate(recommendation.id)
            }
          >
            {t('Approve')}
          </Button>
          <Button
            variant='outline'
            size='sm'
            disabled={
              approvePostureRecommendation.isPending ||
              rejectPostureRecommendation.isPending
            }
            onClick={() =>
              rejectPostureRecommendation.mutate(recommendation.id)
            }
          >
            {t('Reject')}
          </Button>
        </div>
      )
    }

    if (
      recommendation.status === 'approved' &&
      recommendation.applied_at <= 0
    ) {
      return (
        <Button
          variant='outline'
          size='sm'
          disabled={applyPostureRecommendation.isPending}
          onClick={() => applyPostureRecommendation.mutate(recommendation.id)}
        >
          {applyPostureRecommendation.isPending ? (
            <Spinner data-icon='inline-start' />
          ) : (
            <Check data-icon='inline-start' />
          )}
          {t('Apply')}
        </Button>
      )
    }

    if (recommendation.applied_at > 0 || recommendation.status === 'applied') {
      return <Badge variant='outline'>{t('Applied')}</Badge>
    }

    return '-'
  }

  const isOverviewLoading = marginQuery.isLoading || marginQuery.isFetching

  return (
    <>
      <SupplierFormDialog
        open={supplierDialogOpen}
        onOpenChange={setSupplierDialogOpen}
        form={supplierForm}
        setForm={setSupplierForm}
        isSaving={saveSupplier.isPending}
        onSubmit={() => saveSupplier.mutate()}
        t={t}
      />
      <SupplierAgreementFormDialog
        open={agreementDialogOpen}
        onOpenChange={setAgreementDialogOpen}
        form={agreementForm}
        setForm={setAgreementForm}
        isSaving={saveAgreement.isPending}
        onSubmit={() => saveAgreement.mutate()}
        t={t}
      />
      <SupplierRoutePreferenceDialog
        open={routePreferenceDialogOpen}
        onOpenChange={closeRoutePreferenceDialog}
        enabledSuppliers={enabledSuppliers}
        form={routePreferenceForm}
        setForm={setRoutePreferenceForm}
        isSaving={activateRoutePreference.isPending}
        onSubmit={() => activateRoutePreference.mutate()}
        t={t}
      />
      <SupplyRoutingPolicyActivateDialog
        open={Boolean(routingPolicyActivateForm.execution)}
        onOpenChange={closeRoutingPolicyActivateDialog}
        form={routingPolicyActivateForm}
        setForm={setRoutingPolicyActivateForm}
        isSaving={activateRoutingPolicy.isPending}
        onSubmit={() => activateRoutingPolicy.mutate()}
        t={t}
      />
      <SupplyCostProfileRecordDialog
        open={costProfileDialogOpen}
        onOpenChange={closeCostProfileDialog}
        selfHostedSuppliers={selfHostedSuppliers}
        form={costProfileForm}
        setForm={setCostProfileForm}
        isSaving={recordCostProfile.isPending}
        onSubmit={() => recordCostProfile.mutate()}
        t={t}
      />
      <SupplyPrepaidLotRecordDialog
        open={prepaidLotDialogOpen}
        onOpenChange={closePrepaidLotDialog}
        selfOperatedSuppliers={selfOperatedSuppliers}
        form={prepaidLotForm}
        setForm={setPrepaidLotForm}
        isSaving={recordPrepaidLot.isPending}
        onSubmit={() => recordPrepaidLot.mutate()}
        t={t}
      />
      <ActionPlanStatusDialog
        open={actionPlanStatusDialogOpen}
        onOpenChange={closeActionPlanStatusDialog}
        form={actionPlanStatusForm}
        setForm={setActionPlanStatusForm}
        isSaving={updateActionPlanStatus.isPending}
        onSubmit={() => updateActionPlanStatus.mutate()}
        t={t}
      />
      <SupplyActionExecutionRecordDialog
        open={actionExecutionRecordDialogOpen}
        onOpenChange={closeActionExecutionRecordDialog}
        completedPlans={completedActionPlans}
        form={actionExecutionRecordForm}
        setForm={setActionExecutionRecordForm}
        isSaving={recordActionExecution.isPending}
        isLoadingPlans={executionSourcePlansQuery.isLoading}
        onSubmit={() => recordActionExecution.mutate()}
        t={t}
      />
      <SlaContractImportDialog
        open={slaContractDialogOpen}
        onOpenChange={setSlaContractDialogOpen}
        form={slaContractForm}
        setForm={setSlaContractForm}
        isSaving={importSlaContractMutation.isPending}
        onSubmit={() => importSlaContractMutation.mutate()}
        t={t}
      />
      <SlaProbePlanGenerateDialog
        open={slaProbePlanDialogOpen}
        onOpenChange={closeSlaProbePlanDialog}
        contracts={slaContracts}
        form={slaProbePlanForm}
        setForm={setSlaProbePlanForm}
        isSaving={generateSlaProbePlanMutation.isPending}
        onSubmit={() => generateSlaProbePlanMutation.mutate()}
        t={t}
      />
      <SlaProbeRunRecordDialog
        open={slaProbeRunDialogOpen}
        onOpenChange={closeSlaProbeRunDialog}
        plans={slaProbePlans}
        form={slaProbeRunForm}
        setForm={setSlaProbeRunForm}
        isSaving={recordSlaProbeRunMutation.isPending}
        onSubmit={() => recordSlaProbeRunMutation.mutate()}
        t={t}
      />
      <AlertDialog
        open={Boolean(agreementToDelete)}
        onOpenChange={(open) => {
          if (!open) {
            setAgreementToDelete(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t('Delete Supplier Agreement')}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'This removes the cost agreement from future matching. Existing UsageLedger rows are unchanged.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={removeAgreement.isPending}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={removeAgreement.isPending}
              onClick={() => {
                if (agreementToDelete) {
                  removeAgreement.mutate(agreementToDelete.id)
                }
              }}
            >
              {removeAgreement.isPending && (
                <Spinner data-icon='inline-start' />
              )}
              {t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog
        open={Boolean(routePreferenceToDisable)}
        onOpenChange={(open) => {
          if (!open) {
            setRoutePreferenceToDisable(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Disable Route Preference')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'This clears the active supplier route preference. Existing UsageLedger rows are unchanged.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={disableRoutePreference.isPending}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={disableRoutePreference.isPending}
              onClick={() => {
                if (routePreferenceToDisable) {
                  disableRoutePreference.mutate(routePreferenceToDisable)
                }
              }}
            >
              {disableRoutePreference.isPending && (
                <Spinner data-icon='inline-start' />
              )}
              {t('Disable')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <AlertDialog
        open={Boolean(policyToDisable)}
        onOpenChange={(open) => {
          if (!open) {
            setPolicyToDisable(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Disable Routing Policy')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'This stops the active self-hosted routing preference for matching future requests. Existing UsageLedger rows are unchanged.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={disableRoutingPolicy.isPending}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={disableRoutingPolicy.isPending}
              onClick={() => {
                if (policyToDisable) {
                  disableRoutingPolicy.mutate(policyToDisable)
                }
              }}
            >
              {disableRoutingPolicy.isPending && (
                <Spinner data-icon='inline-start' />
              )}
              {t('Disable')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <SectionPageLayout fixedContent>
        {activeTab !== 'control-tower' ? (
          <SectionPageLayout.Title>{t('Token Router')}</SectionPageLayout.Title>
        ) : null}
        {activeTab !== 'control-tower' ? (
          <SectionPageLayout.Actions>
            <Button
              variant='outline'
              onClick={() => {
                void queryClient.invalidateQueries({
                  queryKey: ['token-router'],
                })
              }}
            >
              <RefreshCw data-icon='inline-start' />
              {t('Refresh')}
            </Button>
          </SectionPageLayout.Actions>
        ) : null}
        <SectionPageLayout.Content>
          <div className='flex h-full min-h-0 flex-col gap-4'>
            {activeTab !== 'control-tower' && (
              <div className='flex flex-wrap items-end gap-3'>
                <Field className='w-full sm:w-auto'>
                  <FieldLabel htmlFor='token-router-start'>
                    {t('Period Start')}
                  </FieldLabel>
                  <Input
                    id='token-router-start'
                    type='datetime-local'
                    value={startInput}
                    onChange={(event) => setStartInput(event.target.value)}
                  />
                </Field>
                <Field className='w-full sm:w-auto'>
                  <FieldLabel htmlFor='token-router-end'>
                    {t('Period End')}
                  </FieldLabel>
                  <Input
                    id='token-router-end'
                    type='datetime-local'
                    value={endInput}
                    onChange={(event) => setEndInput(event.target.value)}
                  />
                </Field>
              </div>
            )}

            <Tabs
              value={activeTab}
              onValueChange={(value) => setActiveTab(value as TabValue)}
              className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'
            >
              {activeTab !== 'control-tower' && (
                <div className='min-w-0 overflow-x-auto pb-1'>
                  <TabsList className='inline-flex w-max max-w-none flex-nowrap justify-start group-data-horizontal/tabs:h-auto [&_[data-slot=tabs-trigger]]:flex-none'>
                    <TabsTrigger value='control-tower'>控制塔</TabsTrigger>
                    <TabsTrigger value='overview'>总览</TabsTrigger>
                    <TabsTrigger value='suppliers'>供应商</TabsTrigger>
                    <TabsTrigger value='quality'>质量</TabsTrigger>
                    <TabsTrigger value='capacity'>供给容量</TabsTrigger>
                    <TabsTrigger value='cost-profiles'>成本档案</TabsTrigger>
                    <TabsTrigger value='prepaid-lots'>预付批次</TabsTrigger>
                    <TabsTrigger value='profiles'>流量画像</TabsTrigger>
                    <TabsTrigger value='forecasts'>预测</TabsTrigger>
                    <TabsTrigger value='pricing'>定价</TabsTrigger>
                    <TabsTrigger value='insights'>运营洞察</TabsTrigger>
                    <TabsTrigger value='sla-evidence'>SLA 证据</TabsTrigger>
                    <TabsTrigger value='scorecards'>评分卡</TabsTrigger>
                    <TabsTrigger value='evaluations'>评估</TabsTrigger>
                    <TabsTrigger value='posture'>供应状态</TabsTrigger>
                    <TabsTrigger value='decisions'>决策</TabsTrigger>
                    <TabsTrigger value='opportunities'>机会</TabsTrigger>
                    <TabsTrigger value='actions'>行动计划</TabsTrigger>
                    <TabsTrigger value='executions'>执行记录</TabsTrigger>
                    <TabsTrigger value='routing'>路由策略</TabsTrigger>
                    <TabsTrigger value='ledger'>用量账本</TabsTrigger>
                    <TabsTrigger value='settlements'>结算</TabsTrigger>
                  </TabsList>
                </div>
              )}

              <TabsContent
                value='control-tower'
                className='min-h-0 overflow-auto'
              >
                <ControlTower
                  nav={
                    <div className='min-w-0 overflow-x-auto border-b border-slate-200'>
                      <TabsList
                        className='inline-flex h-10 w-max max-w-none flex-nowrap justify-start gap-1 rounded-none bg-transparent p-0 [&_[data-slot=tabs-trigger]]:h-8 [&_[data-slot=tabs-trigger]]:flex-none [&_[data-slot=tabs-trigger]]:rounded-md [&_[data-slot=tabs-trigger]]:px-3.5 [&_[data-slot=tabs-trigger]]:text-[13px] [&_[data-slot=tabs-trigger]]:font-medium [&_[data-slot=tabs-trigger][data-active]]:bg-blue-50 [&_[data-slot=tabs-trigger][data-active]]:text-blue-600 [&_[data-slot=tabs-trigger][data-active]]:shadow-none'
                      >
                        <TabsTrigger value='control-tower'>
                          流量画像
                        </TabsTrigger>
                        <TabsTrigger value='forecasts'>预测</TabsTrigger>
                        <TabsTrigger value='pricing'>定价</TabsTrigger>
                        <TabsTrigger value='insights'>运营洞察</TabsTrigger>
                        <TabsTrigger value='sla-evidence'>SLA</TabsTrigger>
                        <TabsTrigger value='decisions'>决策</TabsTrigger>
                        <TabsTrigger value='actions'>行动计划</TabsTrigger>
                        <TabsTrigger value='executions'>执行记录</TabsTrigger>
                      </TabsList>
                    </div>
                  }
                  onOpenRoutingPolicies={() => {
                    setActiveTab('routing')
                    toast.info(
                      t(
                        'Select a recorded self-hosted execution source to activate a routing policy.'
                      )
                    )
                  }}
                />
              </TabsContent>

              <TabsContent value='overview' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6'>
                    <StatCard
                      title={t('Requests')}
                      value={formatNumber(totals.requests)}
                      isLoading={isOverviewLoading}
                    />
                    <StatCard
                      title={t('Sell Quota')}
                      value={formatLogQuota(totals.sell)}
                      isLoading={isOverviewLoading}
                    />
                    <StatCard
                      title={t('Cost Quota')}
                      value={formatLogQuota(totals.cost)}
                      isLoading={isOverviewLoading}
                    />
                    <StatCard
                      title={t('Gross Profit')}
                      value={formatSignedQuota(totals.profit)}
                      isLoading={isOverviewLoading}
                    />
                    <StatCard
                      title={t('Cache Hit Rate')}
                      value={formatRate(cacheHitRate)}
                      isLoading={isOverviewLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Margin Summary')}
                    description={t(
                      'Aggregated from UsageLedger; no payment state is implied.'
                    )}
                    action={
                      <ToggleGroup
                        value={[groupBy]}
                        onValueChange={(value) => {
                          const next = value.find((item) => item !== groupBy)
                          if (next) {
                            setGroupBy(next as MarginGroupBy)
                          }
                        }}
                        aria-label={t('Group By')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='supplier'>
                          {t('Supplier')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='channel'>
                          {t('Channel')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='user'>
                          {t('User')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='model'>
                          {t('Model')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='day'>
                          {t('Day')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Group')}</TableHead>
                          <TableHead>{t('Requests')}</TableHead>
                          <TableHead>{t('Sell Quota')}</TableHead>
                          <TableHead>{t('Cost Quota')}</TableHead>
                          <TableHead>{t('Gross Profit')}</TableHead>
                          <TableHead>{t('Cache Hit Rate')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={marginQuery.isLoading}
                          isEmpty={marginRows.length === 0}
                          columns={6}
                          emptyMessage={t('No margin data in this period.')}
                        >
                          {marginRows.map((row) => (
                            <TableRow key={`${groupBy}-${row.group_key}`}>
                              <TableCell>{groupLabel(row, groupBy)}</TableCell>
                              <TableCell>
                                {formatNumber(row.total_requests)}
                              </TableCell>
                              <TableCell>
                                {formatLogQuota(row.total_sell_quota)}
                              </TableCell>
                              <TableCell>
                                {formatLogQuota(row.total_cost_quota)}
                              </TableCell>
                              <TableCell>
                                {formatSignedQuota(row.gross_profit_quota)}
                              </TableCell>
                              <TableCell>
                                {formatRate(row.cache_hit_rate)}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='quality' className='min-h-0 overflow-auto'>
                <DataPanel
                  title={t('Quality Summary')}
                  description={t(
                    'Read-only telemetry from UsageLedger; not an SLA commitment.'
                  )}
                  action={
                    <ToggleGroup
                      value={[qualityGroupBy]}
                      onValueChange={(value) => {
                        const next = value.find(
                          (item) => item !== qualityGroupBy
                        )
                        if (next) {
                          setQualityGroupBy(next as QualityGroupBy)
                        }
                      }}
                      aria-label={t('Group By')}
                      variant='outline'
                      size='sm'
                      spacing={2}
                      className='flex-wrap justify-end'
                    >
                      <ToggleGroupItem value='supplier'>
                        {t('Supplier')}
                      </ToggleGroupItem>
                      <ToggleGroupItem value='channel'>
                        {t('Channel')}
                      </ToggleGroupItem>
                      <ToggleGroupItem value='model'>
                        {t('Model')}
                      </ToggleGroupItem>
                      <ToggleGroupItem value='sla_tier'>
                        {t('SLA Tier')}
                      </ToggleGroupItem>
                      <ToggleGroupItem value='supply_node'>
                        {t('Supply Node')}
                      </ToggleGroupItem>
                      <ToggleGroupItem value='day'>{t('Day')}</ToggleGroupItem>
                    </ToggleGroup>
                  }
                >
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Group')}</TableHead>
                        <TableHead>{t('Requests')}</TableHead>
                        <TableHead>{t('Success Rate')}</TableHead>
                        <TableHead>{t('Average Latency')}</TableHead>
                        <TableHead>{t('Max Latency')}</TableHead>
                        <TableHead>{t('Cache Hit Rate')}</TableHead>
                        <TableHead>{t('Tokens')}</TableHead>
                        <TableHead>{t('Gross Profit')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      <TableRowsState
                        isLoading={qualityQuery.isLoading}
                        isEmpty={qualityRows.length === 0}
                        columns={8}
                        emptyMessage={t('No quality data in this period.')}
                      >
                        {qualityRows.map((row) => (
                          <TableRow
                            key={`${qualityGroupBy}-${row.group_key || '-'}`}
                          >
                            <TableCell>
                              {qualityGroupLabel(row, qualityGroupBy)}
                            </TableCell>
                            <TableCell>
                              {formatNumber(row.total_requests)}
                            </TableCell>
                            <TableCell>
                              {formatRate(row.success_rate)}
                            </TableCell>
                            <TableCell>
                              {formatLatency(row.avg_latency_ms)}
                            </TableCell>
                            <TableCell>
                              {formatLatency(row.max_latency_ms)}
                            </TableCell>
                            <TableCell>
                              {formatRate(row.cache_hit_rate)}
                            </TableCell>
                            <TableCell>
                              <span className='flex flex-col gap-0.5'>
                                <span>
                                  {t('Prompt')}:{' '}
                                  {formatTokens(row.total_prompt_tokens)}
                                </span>
                                <span className='text-muted-foreground'>
                                  {t('Cached')}:{' '}
                                  {formatTokens(row.total_cached_tokens)}
                                </span>
                              </span>
                            </TableCell>
                            <TableCell>
                              {formatSignedQuota(row.gross_profit_quota)}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableRowsState>
                    </TableBody>
                  </Table>
                </DataPanel>
              </TabsContent>

              <TabsContent value='capacity' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
                    <StatCard
                      title={t('Capacity Tokens')}
                      value={formatNumber(capacityTotals.capacity)}
                      isLoading={capacitiesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Used Tokens')}
                      value={formatNumber(capacityTotals.used)}
                      isLoading={capacitiesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Headroom Tokens')}
                      value={formatNumber(capacityTotals.headroom)}
                      isLoading={capacitiesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Utilization')}
                      value={formatRate(capacityUtilization)}
                      isLoading={capacitiesQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supply Capacity Snapshots')}
                    description={t(
                      'Period snapshots of supply headroom with optional telemetry evidence; not an automatic routing policy.'
                    )}
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Supply Node')}</TableHead>
                          <TableHead>{t('Model')}</TableHead>
                          <TableHead>{t('Period')}</TableHead>
                          <TableHead>{t('Tokens')}</TableHead>
                          <TableHead>{t('Utilization')}</TableHead>
                          <TableHead>{t('Quality / Cost')}</TableHead>
                          <TableHead>{t('Telemetry')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={capacitiesQuery.isLoading}
                          isEmpty={capacities.length === 0}
                          columns={8}
                          emptyMessage={t(
                            'No supply capacity snapshots in this period.'
                          )}
                        >
                          {capacities.map((capacity) => (
                            <TableRow key={capacity.id}>
                              <TableCell>
                                {supplierNameById(
                                  suppliers,
                                  capacity.supplier_id
                                )}
                              </TableCell>
                              <TableCell>
                                {capacity.supply_node || '-'}
                              </TableCell>
                              <TableCell>
                                {capacity.model_name || '-'}
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatTimestampToDate(
                                      capacity.period_start
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(capacity.period_end)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatNumber(capacity.capacity_tokens)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Used')}:{' '}
                                    {formatNumber(capacity.used_tokens)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Headroom')}:{' '}
                                    {formatNumber(capacity.headroom_tokens)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatRate(capacity.utilization_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('GPU Utilization')}:{' '}
                                    {formatRate(capacity.gpu_utilization_rate)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatNumber(capacity.quality_score)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Unit Cost')}:{' '}
                                    {formatUnitCost(capacity.unit_cost_quota)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {capacity.last_telemetry_id > 0 ? (
                                  <span className='flex max-w-56 flex-col gap-0.5'>
                                    <span>
                                      {capacity.telemetry_source_type || '-'}
                                    </span>
                                    <span className='text-muted-foreground truncate'>
                                      {capacity.telemetry_source_ref || '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      #{capacity.last_telemetry_id} /{' '}
                                      {formatTimestampToDate(
                                        capacity.telemetry_observed_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent
                value='cost-profiles'
                className='min-h-0 overflow-auto'
              >
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
                    <StatCard
                      title={t('Visible Cost Profiles')}
                      value={formatNumber(costProfiles.length)}
                      isLoading={costProfilesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Cost Basis Capacity')}
                      value={formatTokens(costProfileTotals.capacity)}
                      isLoading={costProfilesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Fixed Cost')}
                      value={formatLogQuota(costProfileTotals.fixedCost)}
                      isLoading={costProfilesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Avg Amortized Unit Cost')}
                      value={formatUnitCost(averageCostProfileUnitCost)}
                      isLoading={costProfilesQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Self-hosted Cost Profiles')}
                    description={t(
                      'Self-hosted cost profiles are accounting evidence for opportunity ranking only; they do not buy capacity, change prices, route traffic, bill users, settle suppliers, or mutate suppliers.'
                    )}
                    action={
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={openCostProfileDialog}
                      >
                        <Plus data-icon='inline-start' />
                        {t('Record Cost Profile')}
                      </Button>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Supply Node')}</TableHead>
                          <TableHead>{t('Model')}</TableHead>
                          <TableHead>{t('Period')}</TableHead>
                          <TableHead>{t('Capacity')}</TableHead>
                          <TableHead>{t('Cost Basis')}</TableHead>
                          <TableHead>{t('Unit Cost')}</TableHead>
                          <TableHead>{t('Source')}</TableHead>
                          <TableHead>{t('Notes')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={costProfilesQuery.isLoading}
                          isEmpty={costProfiles.length === 0}
                          columns={9}
                          emptyMessage={t('No cost profiles in this period.')}
                        >
                          {costProfiles.map((profile) => (
                            <TableRow key={profile.id}>
                              <TableCell>
                                {supplierNameById(
                                  suppliers,
                                  profile.supplier_id
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-52 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {profile.supply_node || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    #{profile.id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>{profile.model_name || '-'}</TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>
                                    {formatTimestampToDate(
                                      profile.period_start
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(profile.period_end)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatTokens(profile.capacity_tokens)}
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5 tabular-nums'>
                                  <span>
                                    {t('Fixed')}:{' '}
                                    {formatLogQuota(profile.fixed_cost_quota)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Variable')}:{' '}
                                    {formatUnitCost(
                                      profile.variable_unit_cost_quota
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatUnitCost(
                                  profile.amortized_unit_cost_quota
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-56 flex-col gap-0.5'>
                                  <Badge variant='secondary'>
                                    {costProfileSourceLabel(
                                      profile.source_type,
                                      t
                                    )}
                                  </Badge>
                                  <span className='text-muted-foreground truncate'>
                                    {profile.source_ref || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(profile.observed_at)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('User')} #{profile.recorded_by || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell className='max-w-64 truncate'>
                                {profile.notes || profile.cost_profile_key}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent
                value='prepaid-lots'
                className='min-h-0 overflow-auto'
              >
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6'>
                    <StatCard
                      title={t('Visible Prepaid Lots')}
                      value={formatNumber(prepaidLots.length)}
                      isLoading={prepaidLotsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Purchased Tokens')}
                      value={formatTokens(prepaidLotTotals.purchased)}
                      isLoading={prepaidLotsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Drawdown Tokens')}
                      value={formatTokens(prepaidLotTotals.drawdown)}
                      isLoading={prepaidLotsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Remaining Tokens')}
                      value={formatTokens(prepaidLotTotals.remaining)}
                      isLoading={prepaidLotsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Avg Drawdown Rate')}
                      value={formatRate(averagePrepaidLotDrawdownRate)}
                      isLoading={prepaidLotsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Self-operated Prepaid Lots')}
                    description={t(
                      'Self-operated prepaid lots are offline procurement evidence with ledger-backed token drawdown; they do not create payments, approvals, capacity, routing, billing, or settlement.'
                    )}
                  >
                    <div className='flex flex-col gap-3'>
                      <div className='flex flex-wrap justify-end gap-2'>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => refreshPrepaidLotUsage.mutate()}
                          disabled={
                            refreshPrepaidLotUsage.isPending ||
                            prepaidLotsQuery.isLoading
                          }
                        >
                          {refreshPrepaidLotUsage.isPending ? (
                            <Spinner data-icon='inline-start' />
                          ) : (
                            <RefreshCw data-icon='inline-start' />
                          )}
                          {t('Refresh Drawdown')}
                        </Button>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={openPrepaidLotDialog}
                          disabled={
                            recordPrepaidLot.isPending ||
                            selfOperatedSuppliers.length === 0
                          }
                        >
                          {recordPrepaidLot.isPending ? (
                            <Spinner data-icon='inline-start' />
                          ) : (
                            <Plus data-icon='inline-start' />
                          )}
                          {t('Record Prepaid Lot')}
                        </Button>
                      </div>
                      <div className='max-w-full overflow-x-auto'>
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>{t('Supplier')}</TableHead>
                              <TableHead>{t('Channel')}</TableHead>
                              <TableHead>{t('Supply Node')}</TableHead>
                              <TableHead>{t('Model')}</TableHead>
                              <TableHead>{t('Period')}</TableHead>
                              <TableHead>{t('Purchase')}</TableHead>
                              <TableHead>{t('Drawdown')}</TableHead>
                              <TableHead>{t('Source')}</TableHead>
                              <TableHead>{t('Recorded')}</TableHead>
                              <TableHead>{t('Notes')}</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            <TableRowsState
                              isLoading={prepaidLotsQuery.isLoading}
                              isEmpty={prepaidLots.length === 0}
                              columns={10}
                              emptyMessage={t(
                                'No prepaid lots in this period.'
                              )}
                            >
                              {prepaidLots.map((lot) => (
                                <TableRow key={lot.id}>
                                  <TableCell>
                                    {supplierNameById(
                                      suppliers,
                                      lot.supplier_id
                                    )}
                                  </TableCell>
                                  <TableCell>
                                    {lot.channel_id > 0
                                      ? `#${lot.channel_id}`
                                      : t('All channels')}
                                  </TableCell>
                                  <TableCell>
                                    <span className='flex max-w-52 flex-col gap-0.5'>
                                      <span className='truncate'>
                                        {lot.supply_node || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        #{lot.id}
                                      </span>
                                    </span>
                                  </TableCell>
                                  <TableCell>{lot.model_name || '-'}</TableCell>
                                  <TableCell>
                                    <span className='flex min-w-40 flex-col gap-0.5'>
                                      <span>
                                        {formatTimestampToDate(
                                          lot.period_start
                                        )}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(lot.period_end)}
                                      </span>
                                    </span>
                                  </TableCell>
                                  <TableCell>
                                    <span className='flex min-w-48 flex-col gap-0.5 tabular-nums'>
                                      <span>
                                        {t('Purchased')}:{' '}
                                        {formatTokens(lot.purchased_tokens)}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {t('Total Cost')}:{' '}
                                        {formatLogQuota(lot.total_cost_quota)}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {t('Unit Cost')}:{' '}
                                        {formatUnitCost(lot.unit_cost_quota)}
                                      </span>
                                    </span>
                                  </TableCell>
                                  <TableCell>
                                    {lot.drawdown_refreshed_at > 0 ? (
                                      <span className='flex min-w-52 flex-col gap-0.5'>
                                        <span>
                                          {t('Used')}:{' '}
                                          {formatTokens(lot.drawdown_tokens)}
                                        </span>
                                        <span className='text-muted-foreground'>
                                          {t('Remaining')}:{' '}
                                          {formatTokens(lot.remaining_tokens)}
                                        </span>
                                        <span className='text-muted-foreground'>
                                          {t('Requests')}:{' '}
                                          {formatNumber(
                                            lot.drawdown_request_count
                                          )}{' '}
                                          / {formatRate(lot.drawdown_rate)}
                                        </span>
                                        <span className='text-muted-foreground truncate'>
                                          {lot.drawdown_source_type || '-'} /{' '}
                                          {formatTimestampToDate(
                                            lot.drawdown_refreshed_at
                                          )}
                                        </span>
                                      </span>
                                    ) : (
                                      <span className='flex min-w-44 flex-col gap-0.5'>
                                        <Badge variant='outline'>
                                          {t('Unrefreshed')}
                                        </Badge>
                                        <span className='text-muted-foreground'>
                                          {t('Remaining')}:{' '}
                                          {formatTokens(lot.remaining_tokens)}
                                        </span>
                                      </span>
                                    )}
                                  </TableCell>
                                  <TableCell>
                                    <span className='flex max-w-64 flex-col gap-0.5'>
                                      <Badge variant='secondary'>
                                        {costProfileSourceLabel(
                                          lot.source_type,
                                          t
                                        )}
                                      </Badge>
                                      <span className='text-muted-foreground truncate'>
                                        {lot.source_ref || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(lot.observed_at)}
                                      </span>
                                      {lot.external_ref ? (
                                        <span className='text-muted-foreground truncate'>
                                          {lot.external_ref}
                                        </span>
                                      ) : null}
                                    </span>
                                  </TableCell>
                                  <TableCell>
                                    <span className='flex min-w-40 flex-col gap-0.5'>
                                      <span>
                                        {t('User')} #{lot.recorded_by || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(lot.created_at)}
                                      </span>
                                    </span>
                                  </TableCell>
                                  <TableCell className='max-w-64 truncate'>
                                    {lot.notes || lot.prepaid_lot_key}
                                  </TableCell>
                                </TableRow>
                              ))}
                            </TableRowsState>
                          </TableBody>
                        </Table>
                      </div>
                    </div>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='profiles' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6'>
                    <StatCard
                      title={t('Visible Profiles')}
                      value={formatNumber(profiles.length)}
                      isLoading={profilesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Demand Tokens')}
                      value={formatTokens(profileTotals.demand)}
                      isLoading={profilesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Cached Tokens')}
                      value={formatTokens(profileTotals.cached)}
                      isLoading={profilesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Supply Headroom')}
                      value={formatTokens(profileTotals.headroom)}
                      isLoading={profilesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Gross Profit')}
                      value={formatSignedQuota(profileTotals.profit)}
                      isLoading={profilesQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Slice Profiles')}
                    description={t(
                      'Materialized demand and supply headroom slices from UsageLedger and SupplyCapacity; this is the factual input for supply decisions.'
                    )}
                    action={
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => generateProfiles.mutate()}
                        disabled={generateProfiles.isPending}
                      >
                        {generateProfiles.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <RefreshCw data-icon='inline-start' />
                        )}
                        {t('Generate Profiles')}
                      </Button>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Requests')}</TableHead>
                          <TableHead>{t('Sessions')}</TableHead>
                          <TableHead>{t('Demand Tokens')}</TableHead>
                          <TableHead>{t('Peak Tokens')}</TableHead>
                          <TableHead>{t('Cache')}</TableHead>
                          <TableHead>{t('SLA Met')}</TableHead>
                          <TableHead>{t('Latency')}</TableHead>
                          <TableHead>{t('Gross Profit')}</TableHead>
                          <TableHead>{t('Supply')}</TableHead>
                          <TableHead>{t('Generated')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={profilesQuery.isLoading}
                          isEmpty={profiles.length === 0}
                          columns={11}
                          emptyMessage={t(
                            'No traffic profiles in this period.'
                          )}
                        >
                          {profiles.map((profile) => (
                            <TableRow key={profile.id}>
                              <TableCell>
                                <span className='flex max-w-72 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {profile.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {profile.sla_tier || '-'} / {t('User')} #
                                    {profile.user_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatNumber(profile.request_count)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Success')}:{' '}
                                    {formatNumber(
                                      profile.success_request_count
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatNumber(profile.unique_sessions)}
                              </TableCell>
                              <TableCell>
                                {formatTokens(profile.demand_tokens)}
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatTokens(profile.peak_tokens)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Peak Ratio')}:{' '}
                                    {formatNumber(profile.peak_ratio)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatRate(profile.cache_hit_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTokens(profile.total_cached_tokens)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatRate(profile.sla_met_rate)}
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {t('Average')}:{' '}
                                    {formatLatency(profile.avg_latency_ms)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Max')}:{' '}
                                    {formatLatency(profile.max_latency_ms)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatSignedQuota(profile.gross_profit_quota)}
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {t('Headroom')}:{' '}
                                    {formatTokens(
                                      profile.supply_headroom_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Quality Score')}:{' '}
                                    {formatNumber(
                                      profile.avg_supply_quality_score
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {profile.generated_at > 0
                                  ? formatTimestampToDate(profile.generated_at)
                                  : '-'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='forecasts' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5'>
                    <StatCard
                      title={t('Visible Forecasts')}
                      value={formatNumber(forecasts.length)}
                      isLoading={forecastsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Forecast Demand')}
                      value={formatTokens(forecastTotals.demand)}
                      isLoading={forecastsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Open Forecast Gap')}
                      value={formatTokens(forecastTotals.gap)}
                      isLoading={forecastsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Average Confidence')}
                      value={formatRate(averageForecastConfidence)}
                      isLoading={forecastsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Traffic Forecasts')}
                    description={t(
                      'Next-period demand hypotheses materialized from TrafficProfile; forecasts are planning evidence and do not mutate decisions, pricing, routing, suppliers, capacity, billing, settlement, or payments.'
                    )}
                    action={
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => generateForecasts.mutate()}
                        disabled={generateForecasts.isPending}
                      >
                        {generateForecasts.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <RefreshCw data-icon='inline-start' />
                        )}
                        {t('Generate Traffic Forecasts')}
                      </Button>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Source Period')}</TableHead>
                          <TableHead>{t('Target Period')}</TableHead>
                          <TableHead>{t('Observed Evidence')}</TableHead>
                          <TableHead>{t('Forecast')}</TableHead>
                          <TableHead>{t('Supply Gap')}</TableHead>
                          <TableHead>{t('Confidence')}</TableHead>
                          <TableHead>{t('Economics')}</TableHead>
                          <TableHead>{t('Reason')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={forecastsQuery.isLoading}
                          isEmpty={forecasts.length === 0}
                          columns={9}
                          emptyMessage={t(
                            'No traffic forecasts for this source period.'
                          )}
                        >
                          {forecasts.map((forecast) => (
                            <TableRow key={forecast.id}>
                              <TableCell>
                                <span className='flex max-w-72 min-w-48 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {forecast.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {forecast.sla_tier || '-'} / {t('User')} #
                                    {forecast.user_id}
                                  </span>
                                  <span className='text-muted-foreground max-w-64 truncate'>
                                    {forecast.slice_key || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>
                                    {formatTimestampToDate(
                                      forecast.source_period_start
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(
                                      forecast.source_period_end
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>
                                    {formatTimestampToDate(
                                      forecast.target_period_start
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(
                                      forecast.target_period_end
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Profiles')}:{' '}
                                    {formatNumber(
                                      forecast.source_profile_count
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Requests')}:{' '}
                                    {formatNumber(
                                      forecast.observed_request_count
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Demand')}:{' '}
                                    {formatTokens(
                                      forecast.observed_demand_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Peak')}:{' '}
                                    {formatTokens(
                                      forecast.observed_peak_tokens
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Demand')}:{' '}
                                    {formatTokens(
                                      forecast.forecast_demand_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Peak')}:{' '}
                                    {formatTokens(
                                      forecast.forecast_peak_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Headroom')}:{' '}
                                    {formatTokens(
                                      forecast.forecast_headroom_tokens
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={
                                    forecast.forecast_gap_tokens > 0
                                      ? 'destructive'
                                      : 'secondary'
                                  }
                                >
                                  {formatTokens(forecast.forecast_gap_tokens)}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-36 flex-col gap-1'>
                                  <span className='tabular-nums'>
                                    {formatRate(forecast.confidence)}
                                  </span>
                                  <Badge variant='outline'>
                                    {forecast.method || '-'}
                                  </Badge>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Cache Hit Rate')}:{' '}
                                    {formatRate(forecast.cache_hit_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('SLA Met')}:{' '}
                                    {formatRate(forecast.sla_met_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Gross Profit')}:{' '}
                                    {formatSignedQuota(
                                      forecast.gross_profit_quota
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Unit Cost')}:{' '}
                                    {formatUnitCost(
                                      forecast.avg_unit_cost_quota
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell className='max-w-72 truncate'>
                                {forecast.reason || '-'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='pricing' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5'>
                    <StatCard
                      title={t('Visible Pricing Recommendations')}
                      value={formatNumber(pricingRecommendations.length)}
                      isLoading={pricingRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Draft Pricing Reviews')}
                      value={formatNumber(pricingRecommendationTotals.draft)}
                      isLoading={pricingRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Raise Price')}
                      value={formatNumber(
                        pricingRecommendationTotals.raisePrice
                      )}
                      isLoading={pricingRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Share Savings')}
                      value={formatNumber(
                        pricingRecommendationTotals.shareSavings
                      )}
                      isLoading={pricingRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Average Recommended Unit Price')}
                      value={formatUnitCost(averageRecommendedUnitPrice)}
                      isLoading={pricingRecommendationsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Pricing Recommendations')}
                    description={t(
                      'Human-reviewed SLA and price recommendations from TrafficProfile; approval records evidence only and does not mutate pricing, billing, settlement, or routing.'
                    )}
                  >
                    <div className='mb-4 flex flex-wrap justify-end gap-2'>
                      <ToggleGroup
                        value={[pricingRecommendationStatus]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== pricingRecommendationStatus
                          )
                          if (next) {
                            setPricingRecommendationStatus(
                              next as PricingRecommendationStatusFilter
                            )
                          }
                        }}
                        aria-label={t('Pricing Status')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='draft'>
                          {t('Draft')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='approved'>
                          {t('Approved')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='rejected'>
                          {t('Rejected')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <ToggleGroup
                        value={[pricingRecommendationAction]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== pricingRecommendationAction
                          )
                          if (next) {
                            setPricingRecommendationAction(
                              next as PricingRecommendationActionFilter
                            )
                          }
                        }}
                        aria-label={t('Pricing Action')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='raise_price'>
                          {t('Raise Price')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='keep_price'>
                          {t('Keep Price')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='share_savings'>
                          {t('Share Savings')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          generatePricingRecommendationsMutation.mutate()
                        }
                        disabled={
                          generatePricingRecommendationsMutation.isPending
                        }
                      >
                        {generatePricingRecommendationsMutation.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <RefreshCw data-icon='inline-start' />
                        )}
                        {t('Generate Pricing Recommendations')}
                      </Button>
                    </div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Action')}</TableHead>
                          <TableHead>{t('Review')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                          <TableHead>{t('Unit Economics')}</TableHead>
                          <TableHead>{t('Recommended Price')}</TableHead>
                          <TableHead>{t('Demand Evidence')}</TableHead>
                          <TableHead>{t('Supply Evidence')}</TableHead>
                          <TableHead>{t('Reason')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={pricingRecommendationsQuery.isLoading}
                          isEmpty={pricingRecommendations.length === 0}
                          columns={9}
                          emptyMessage={t(
                            'No pricing recommendations in this period.'
                          )}
                        >
                          {pricingRecommendations.map((recommendation) => (
                            <TableRow key={recommendation.id}>
                              <TableCell>
                                <span className='flex max-w-72 min-w-48 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {recommendation.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {recommendation.sla_tier || '-'} /{' '}
                                    {t('User')} #{recommendation.user_id}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Profile')} #
                                    {recommendation.traffic_profile_id}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(
                                      recommendation.period_start
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(
                                      recommendation.period_end
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={pricingRecommendationActionVariant(
                                    recommendation.action
                                  )}
                                >
                                  {pricingRecommendationActionLabel(
                                    recommendation.action,
                                    t
                                  )}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-32 flex-col gap-1'>
                                  <Badge
                                    variant={decisionStatusVariant(
                                      recommendation.status
                                    )}
                                  >
                                    {decisionStatusLabel(
                                      recommendation.status,
                                      t
                                    )}
                                  </Badge>
                                  {recommendation.reviewed_at > 0 ? (
                                    <>
                                      <span className='text-muted-foreground'>
                                        {t('User')} #
                                        {recommendation.reviewed_by || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          recommendation.reviewed_at
                                        )}
                                      </span>
                                      <span className='text-muted-foreground max-w-40 truncate'>
                                        {recommendation.review_note || '-'}
                                      </span>
                                    </>
                                  ) : (
                                    <span className='text-muted-foreground'>
                                      -
                                    </span>
                                  )}
                                </span>
                              </TableCell>
                              <TableCell>
                                {recommendation.status === 'draft' ? (
                                  <div className='flex flex-wrap gap-2'>
                                    <Button
                                      variant='outline'
                                      size='sm'
                                      disabled={
                                        approvePricing.isPending ||
                                        rejectPricing.isPending
                                      }
                                      onClick={() =>
                                        approvePricing.mutate(recommendation.id)
                                      }
                                    >
                                      {t('Approve')}
                                    </Button>
                                    <Button
                                      variant='outline'
                                      size='sm'
                                      disabled={
                                        approvePricing.isPending ||
                                        rejectPricing.isPending
                                      }
                                      onClick={() =>
                                        rejectPricing.mutate(recommendation.id)
                                      }
                                    >
                                      {t('Reject')}
                                    </Button>
                                  </div>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Current Unit Price')}:{' '}
                                    {formatUnitCost(
                                      recommendation.current_unit_price_quota
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Current Unit Cost')}:{' '}
                                    {formatUnitCost(
                                      recommendation.current_unit_cost_quota
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Current Margin')}:{' '}
                                    {formatRate(
                                      recommendation.current_margin_rate
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Gross Profit')}:{' '}
                                    {formatSignedQuota(
                                      recommendation.gross_profit_quota
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>
                                    {t('Unit Price')}:{' '}
                                    {formatUnitCost(
                                      recommendation.recommended_unit_price_quota
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Margin')}:{' '}
                                    {formatRate(
                                      recommendation.recommended_margin_rate
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Requests')}:{' '}
                                    {formatNumber(recommendation.request_count)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Demand')}:{' '}
                                    {formatTokens(recommendation.demand_tokens)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Cache Hit Rate')}:{' '}
                                    {formatRate(recommendation.cache_hit_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('SLA Met')}:{' '}
                                    {formatRate(recommendation.sla_met_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Latency')}:{' '}
                                    {formatLatency(
                                      recommendation.avg_latency_ms
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>
                                    {t('Headroom')}:{' '}
                                    {formatTokens(
                                      recommendation.supply_headroom_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Quality Score')}:{' '}
                                    {formatNumber(
                                      recommendation.avg_supply_quality_score
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Unit Cost')}:{' '}
                                    {formatUnitCost(
                                      recommendation.avg_unit_cost_quota
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell className='max-w-72 truncate'>
                                {recommendation.reason || '-'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='insights' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5'>
                    <StatCard
                      title={t('Visible Operating Insights')}
                      value={formatNumber(operatingInsights.length)}
                      isLoading={operatingInsightsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Action Insights')}
                      value={formatNumber(operatingInsightTotals.action)}
                      isLoading={operatingInsightsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Watch Insights')}
                      value={formatNumber(operatingInsightTotals.watch)}
                      isLoading={operatingInsightsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Cache Efficiency Insights')}
                      value={formatNumber(
                        operatingInsightTotals.cacheEfficiency
                      )}
                      isLoading={operatingInsightsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Acknowledged Insights')}
                      value={formatNumber(operatingInsightTotals.acknowledged)}
                      isLoading={operatingInsightsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Operating Insights')}
                    description={t(
                      'Agent-readable operating hypotheses synthesized from TrafficProfile, SupplyDecision, and PricingRecommendation; review records evidence only and does not mutate pricing, suppliers, capacity, routing, billing, settlement, or payments.'
                    )}
                  >
                    <div className='mb-4 flex flex-wrap justify-end gap-2'>
                      <ToggleGroup
                        value={[operatingInsightStatus]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== operatingInsightStatus
                          )
                          if (next) {
                            setOperatingInsightStatus(
                              next as OperatingInsightStatusFilter
                            )
                          }
                        }}
                        aria-label={t('Insight Status')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='draft'>
                          {t('Draft')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='acknowledged'>
                          {t('Acknowledged')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='dismissed'>
                          {t('Dismissed')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <ToggleGroup
                        value={[operatingInsightSeverity]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== operatingInsightSeverity
                          )
                          if (next) {
                            setOperatingInsightSeverity(
                              next as OperatingInsightSeverityFilter
                            )
                          }
                        }}
                        aria-label={t('Insight Severity')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='action'>
                          {t('Action')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='watch'>
                          {t('Watch')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='info'>
                          {t('Info')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <ToggleGroup
                        value={[operatingInsightCategory]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== operatingInsightCategory
                          )
                          if (next) {
                            setOperatingInsightCategory(
                              next as OperatingInsightCategoryFilter
                            )
                          }
                        }}
                        aria-label={t('Insight Category')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='cache_efficiency'>
                          {t('Cache Efficiency')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='capacity_risk'>
                          {t('Capacity Risk')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='pricing_risk'>
                          {t('Pricing Risk')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='quality_watch'>
                          {t('Quality Watch')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='steady_state'>
                          {t('Steady State')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          generateOperatingInsightsMutation.mutate()
                        }
                        disabled={generateOperatingInsightsMutation.isPending}
                      >
                        {generateOperatingInsightsMutation.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <RefreshCw data-icon='inline-start' />
                        )}
                        {t('Generate Operating Insights')}
                      </Button>
                    </div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Insight')}</TableHead>
                          <TableHead>{t('Review')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                          <TableHead>{t('Linked Decisions')}</TableHead>
                          <TableHead>{t('Linked Pricing')}</TableHead>
                          <TableHead>{t('SLA Evidence')}</TableHead>
                          <TableHead>{t('Traffic Evidence')}</TableHead>
                          <TableHead>{t('Economics')}</TableHead>
                          <TableHead>{t('Recommended Action')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={operatingInsightsQuery.isLoading}
                          isEmpty={operatingInsights.length === 0}
                          columns={10}
                          emptyMessage={t(
                            'No operating insights in this period.'
                          )}
                        >
                          {operatingInsights.map((insight) => (
                            <TableRow key={insight.id}>
                              <TableCell>
                                <span className='flex max-w-72 min-w-48 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {insight.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {insight.sla_tier || '-'} / {t('User')} #
                                    {insight.user_id}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Profile')} #{insight.traffic_profile_id}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(
                                      insight.period_start
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(insight.period_end)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-80 min-w-64 flex-col gap-1'>
                                  <span className='flex flex-wrap gap-1'>
                                    <Badge
                                      variant={operatingInsightSeverityVariant(
                                        insight.severity
                                      )}
                                    >
                                      {operatingInsightSeverityLabel(
                                        insight.severity,
                                        t
                                      )}
                                    </Badge>
                                    <Badge variant='outline'>
                                      {operatingInsightCategoryLabel(
                                        insight.category,
                                        t
                                      )}
                                    </Badge>
                                  </span>
                                  <span className='font-medium'>
                                    {insight.title || '-'}
                                  </span>
                                  <span className='text-muted-foreground line-clamp-2'>
                                    {insight.summary || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-36 flex-col gap-1'>
                                  <Badge
                                    variant={operatingInsightStatusVariant(
                                      insight.status
                                    )}
                                  >
                                    {operatingInsightStatusLabel(
                                      insight.status,
                                      t
                                    )}
                                  </Badge>
                                  {insight.reviewed_at > 0 ? (
                                    <>
                                      <span className='text-muted-foreground'>
                                        {t('User')} #
                                        {insight.reviewed_by || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          insight.reviewed_at
                                        )}
                                      </span>
                                      <span className='text-muted-foreground max-w-40 truncate'>
                                        {insight.review_note || '-'}
                                      </span>
                                    </>
                                  ) : (
                                    <span className='text-muted-foreground'>
                                      -
                                    </span>
                                  )}
                                </span>
                              </TableCell>
                              <TableCell>
                                {insight.status === 'draft' ? (
                                  <div className='flex flex-wrap gap-2'>
                                    <Button
                                      variant='outline'
                                      size='sm'
                                      disabled={
                                        acknowledgeInsight.isPending ||
                                        dismissInsight.isPending
                                      }
                                      onClick={() =>
                                        acknowledgeInsight.mutate(insight.id)
                                      }
                                    >
                                      {t('Acknowledge')}
                                    </Button>
                                    <Button
                                      variant='outline'
                                      size='sm'
                                      disabled={
                                        acknowledgeInsight.isPending ||
                                        dismissInsight.isPending
                                      }
                                      onClick={() =>
                                        dismissInsight.mutate(insight.id)
                                      }
                                    >
                                      {t('Dismiss')}
                                    </Button>
                                  </div>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Decision')} #
                                    {insight.supply_decision_id || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Track')}:{' '}
                                    {insight.supply_decision_track
                                      ? decisionTrackLabel(
                                          insight.supply_decision_track,
                                          t
                                        )
                                      : '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Type')}:{' '}
                                    {decisionTypeLabel(
                                      insight.supply_decision_type,
                                      t
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Status')}:{' '}
                                    {insight.supply_decision_status
                                      ? decisionStatusLabel(
                                          insight.supply_decision_status,
                                          t
                                        )
                                      : '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('ROI Score')}:{' '}
                                    {formatNumber(
                                      insight.supply_decision_roi_score
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {renderOperatingInsightSlaEvidence(insight, t)}
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Pricing')} #
                                    {insight.pricing_recommendation_id || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Action')}:{' '}
                                    {insight.pricing_recommendation_action
                                      ? pricingRecommendationActionLabel(
                                          insight.pricing_recommendation_action,
                                          t
                                        )
                                      : '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Status')}:{' '}
                                    {insight.pricing_recommendation_status
                                      ? decisionStatusLabel(
                                          insight.pricing_recommendation_status,
                                          t
                                        )
                                      : '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Unit Price')}:{' '}
                                    {formatUnitCost(
                                      insight.recommended_unit_price_quota
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Margin')}:{' '}
                                    {formatRate(
                                      insight.recommended_margin_rate
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Demand')}:{' '}
                                    {formatTokens(insight.demand_tokens)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Peak')}:{' '}
                                    {formatTokens(insight.peak_tokens)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Headroom')}:{' '}
                                    {formatTokens(
                                      insight.supply_headroom_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Cache Hit Rate')}:{' '}
                                    {formatRate(insight.cache_hit_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('SLA Met')}:{' '}
                                    {formatRate(insight.sla_met_rate)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>
                                    {t('Gross Profit')}:{' '}
                                    {formatSignedQuota(
                                      insight.gross_profit_quota
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Unit Cost')}:{' '}
                                    {formatUnitCost(
                                      insight.avg_unit_cost_quota
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell className='max-w-80'>
                                <span className='line-clamp-3'>
                                  {insight.recommended_action || '-'}
                                </span>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent
                value='sla-evidence'
                className='min-h-0 overflow-auto'
              >
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5'>
                    <StatCard
                      title={t('SLA Contracts')}
                      value={formatNumber(slaContracts.length)}
                      isLoading={slaContractsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Active SLA Contracts')}
                      value={formatNumber(slaEvidenceTotals.activeContracts)}
                      isLoading={slaContractsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Admission Probe Plans')}
                      value={formatNumber(slaEvidenceTotals.admissionPlans)}
                      isLoading={slaProbePlansQuery.isLoading}
                    />
                    <StatCard
                      title={t('Passed Probe Runs')}
                      value={formatNumber(slaEvidenceTotals.passedRuns)}
                      isLoading={slaProbeRunsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Failed Probe Runs')}
                      value={formatNumber(slaEvidenceTotals.failedRuns)}
                      isLoading={slaProbeRunsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('SLA Contracts')}
                    description={t(
                      'Versioned SLA source evidence for measurement profiles and gates; importing a contract does not admit suppliers or alter routing.'
                    )}
                  >
                    <div className='mb-4 flex flex-wrap justify-end gap-2'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => openSlaContractImportDialog()}
                        disabled={importSlaContractMutation.isPending}
                      >
                        {importSlaContractMutation.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <Plus data-icon='inline-start' />
                        )}
                        {t('Import SLA Contract')}
                      </Button>
                      <ToggleGroup
                        value={[slaContractStatus]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== slaContractStatus
                          )
                          if (next) {
                            setSlaContractStatus(
                              next as SlaContractStatusFilter
                            )
                          }
                        }}
                        aria-label={t('Contract Status')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='draft'>
                          {t('Draft')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='active'>
                          {t('Active')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='retired'>
                          {t('Retired')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                    </div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Contract')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Source')}</TableHead>
                          <TableHead>{t('Effective Period')}</TableHead>
                          <TableHead>{t('Gate Profiles')}</TableHead>
                          <TableHead>{t('Imported')}</TableHead>
                          <TableHead>{t('Action')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={slaContractsQuery.isLoading}
                          isEmpty={slaContracts.length === 0}
                          columns={7}
                          emptyMessage={t('No SLA contracts in this period.')}
                        >
                          {slaContracts.map((contract) => (
                            <TableRow key={contract.id}>
                              <TableCell>
                                <span className='flex max-w-80 min-w-56 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {contract.contract_key}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {contract.model_name || '-'} /{' '}
                                    {contract.provider_family || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {contract.version || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={slaContractStatusVariant(
                                    contract.status
                                  )}
                                >
                                  {slaContractStatusLabel(contract.status, t)}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-80 min-w-56 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {contract.source_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {contract.source_ref || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {contract.source_sha256 || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {contract.effective_from > 0 ||
                                contract.effective_to > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {contract.effective_from > 0
                                        ? formatTimestampToDate(
                                            contract.effective_from
                                          )
                                        : '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {contract.effective_to > 0
                                        ? formatTimestampToDate(
                                            contract.effective_to
                                          )
                                        : '-'}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-72 min-w-48 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {t('Measurement')}:{' '}
                                    {contract.measurement_profile_json || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {t('Hard Gate')}:{' '}
                                    {contract.hard_gate_json || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {t('Soft Gate')}:{' '}
                                    {contract.soft_gate_json || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {contract.imported_at > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #{contract.imported_by || '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        contract.imported_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                <Button
                                  variant='outline'
                                  size='sm'
                                  onClick={() =>
                                    openSlaProbePlanDialog(contract)
                                  }
                                  disabled={
                                    generateSlaProbePlanMutation.isPending
                                  }
                                >
                                  <Plus data-icon='inline-start' />
                                  {t('Generate Plan')}
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>

                  <DataPanel
                    title={t('SLA Probe Plans')}
                    description={t(
                      'Generated measurement plans for suppliers and channels; plans are evidence inputs, not browser-executed benchmarks.'
                    )}
                  >
                    <div className='mb-4 flex flex-wrap justify-end gap-2'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => openSlaProbePlanDialog()}
                        disabled={generateSlaProbePlanMutation.isPending}
                      >
                        {generateSlaProbePlanMutation.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <Plus data-icon='inline-start' />
                        )}
                        {t('Generate Probe Plan')}
                      </Button>
                      <ToggleGroup
                        value={[slaProbeType]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== slaProbeType
                          )
                          if (next) {
                            setSlaProbeType(next as SlaProbeTypeFilter)
                          }
                        }}
                        aria-label={t('Probe Type')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='admission'>
                          {t('Admission')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='runtime_light'>
                          {t('Runtime Light')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='runtime_deep'>
                          {t('Runtime Deep')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='incident_recheck'>
                          {t('Incident Recheck')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <ToggleGroup
                        value={[slaProbeRouteMode]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== slaProbeRouteMode
                          )
                          if (next) {
                            setSlaProbeRouteMode(
                              next as SlaProbeRouteModeFilter
                            )
                          }
                        }}
                        aria-label={t('Route Mode')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='direct_upstream'>
                          {t('Direct Upstream')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='through_token_router'>
                          {t('Through Token Router')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                    </div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Plan')}</TableHead>
                          <TableHead>{t('Target')}</TableHead>
                          <TableHead>{t('Probe')}</TableHead>
                          <TableHead>{t('Profiles')}</TableHead>
                          <TableHead>{t('Schedule')}</TableHead>
                          <TableHead>{t('Generated')}</TableHead>
                          <TableHead>{t('Action')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={slaProbePlansQuery.isLoading}
                          isEmpty={slaProbePlans.length === 0}
                          columns={7}
                          emptyMessage={t('No SLA probe plans in this period.')}
                        >
                          {slaProbePlans.map((plan) => (
                            <TableRow key={plan.id}>
                              <TableCell>
                                <span className='flex max-w-80 min-w-56 flex-col gap-0.5'>
                                  <span>#{plan.id}</span>
                                  <span className='text-muted-foreground truncate'>
                                    {plan.plan_key || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Contract')} #{plan.contract_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>{plan.model_name || '-'}</span>
                                  <span className='text-muted-foreground'>
                                    {plan.sla_tier || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Supplier')} #{plan.supplier_id} /{' '}
                                    {t('Channel')} #{plan.channel_id || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-1'>
                                  <Badge variant='outline'>
                                    {slaProbeTypeLabel(plan.probe_type, t)}
                                  </Badge>
                                  <Badge variant='outline'>
                                    {slaProbeRouteModeLabel(plan.route_mode, t)}
                                  </Badge>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-72 min-w-52 flex-col gap-0.5'>
                                  <span>
                                    {t('Prompt Suite')}:{' '}
                                    {plan.prompt_suite_key || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Tokenizer Ref')}:{' '}
                                    {plan.tokenizer_ref || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Sample Size')}:{' '}
                                    {formatNumber(plan.sample_size)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Repeat Count')}:{' '}
                                    {formatNumber(plan.repeat_count)}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {t('Cache Profile')}:{' '}
                                    {plan.cache_profile || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Interval')}:{' '}
                                    {formatNumber(
                                      plan.schedule_interval_seconds
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Jitter')}:{' '}
                                    {formatNumber(plan.jitter_seconds)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Max Probe Quota')}:{' '}
                                    {formatLogQuota(plan.max_probe_quota)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {plan.generated_at > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #{plan.generated_by || '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(plan.generated_at)}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                <Button
                                  variant='outline'
                                  size='sm'
                                  onClick={() => openSlaProbeRunDialog(plan)}
                                  disabled={recordSlaProbeRunMutation.isPending}
                                >
                                  <Pencil data-icon='inline-start' />
                                  {t('Record Run')}
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>

                  <DataPanel
                    title={t('SLA Probe Runs')}
                    description={t(
                      'Runner-recorded SLA evidence and artifacts; pass/fail status is not automatically applied to supplier admission.'
                    )}
                  >
                    <div className='mb-4 flex flex-wrap justify-end gap-2'>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => openSlaProbeRunDialog()}
                        disabled={
                          recordSlaProbeRunMutation.isPending ||
                          slaProbePlans.length === 0
                        }
                      >
                        {recordSlaProbeRunMutation.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <Pencil data-icon='inline-start' />
                        )}
                        {t('Record Probe Run')}
                      </Button>
                      <ToggleGroup
                        value={[slaProbeRunStatus]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== slaProbeRunStatus
                          )
                          if (next) {
                            setSlaProbeRunStatus(
                              next as SlaProbeRunStatusFilter
                            )
                          }
                        }}
                        aria-label={t('Run Status')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='running'>
                          {t('Running')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='passed'>
                          {t('Passed')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='failed'>
                          {t('Failed')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='invalid'>
                          {t('Invalid')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='cancelled'>
                          {t('Cancelled')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                    </div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Run')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Plan')}</TableHead>
                          <TableHead>{t('Gate Result')}</TableHead>
                          <TableHead>{t('Runtime')}</TableHead>
                          <TableHead>{t('Artifact')}</TableHead>
                          <TableHead>{t('Recorded')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={slaProbeRunsQuery.isLoading}
                          isEmpty={slaProbeRuns.length === 0}
                          columns={7}
                          emptyMessage={t('No SLA probe runs in this period.')}
                        >
                          {slaProbeRuns.map((run) => (
                            <TableRow key={run.id}>
                              <TableCell>
                                <span className='flex max-w-80 min-w-56 flex-col gap-0.5'>
                                  <span>#{run.id}</span>
                                  <span className='text-muted-foreground truncate'>
                                    {run.run_key || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {run.model_name || '-'} /{' '}
                                    {run.sla_tier || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={slaProbeRunStatusVariant(run.status)}
                                >
                                  {slaProbeRunStatusLabel(run.status, t)}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Plan')} #{run.plan_id} / {t('Contract')}{' '}
                                    #{run.contract_id}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Supplier')} #{run.supplier_id} /{' '}
                                    {t('Channel')} #{run.channel_id || '-'}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {slaProbeRouteModeLabel(run.route_mode, t)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-72 min-w-48 flex-col gap-1'>
                                  <Badge
                                    variant={
                                      run.hard_gate_passed
                                        ? 'secondary'
                                        : 'destructive'
                                    }
                                  >
                                    {run.hard_gate_passed
                                      ? t('Hard Gate Passed')
                                      : t('Hard Gate Failed')}
                                  </Badge>
                                  <span className='text-muted-foreground truncate'>
                                    {t('Soft Gate Warnings')}:{' '}
                                    {run.soft_gate_warnings || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {t('Failure Reasons')}:{' '}
                                    {run.failure_reasons || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {t('Summary')}: {run.summary_json || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-72 min-w-48 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {run.runner_version || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {run.git_commit || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {run.runtime_ref || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {run.endpoint || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-72 min-w-48 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {run.artifact_uri || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {run.artifact_sha256 || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  {run.started_at > 0 && (
                                    <span>
                                      {t('Started')}:{' '}
                                      {formatTimestampToDate(run.started_at)}
                                    </span>
                                  )}
                                  {run.ended_at > 0 && (
                                    <span className='text-muted-foreground'>
                                      {t('Ended')}:{' '}
                                      {formatTimestampToDate(run.ended_at)}
                                    </span>
                                  )}
                                  {run.recorded_at > 0 && (
                                    <>
                                      <span className='text-muted-foreground'>
                                        {t('User')} #{run.recorded_by || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(run.recorded_at)}
                                      </span>
                                    </>
                                  )}
                                </span>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='scorecards' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5'>
                    <StatCard
                      title={t('Visible Scorecards')}
                      value={formatNumber(scorecards.length)}
                      isLoading={scorecardsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Average Score')}
                      value={formatNumber(averageScore)}
                      isLoading={scorecardsQuery.isLoading}
                    />
                    <StatCard
                      title={t('A/B Suppliers')}
                      value={formatNumber(scorecardTotals.strong)}
                      isLoading={scorecardsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Supply Headroom')}
                      value={formatTokens(scorecardTotals.headroom)}
                      isLoading={scorecardsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supplier Scorecards')}
                    description={t(
                      'Supplier period ratings from UsageLedger and SupplyCapacity; no automatic routing or disabling is executed.'
                    )}
                    action={
                      <div className='flex flex-wrap justify-end gap-2'>
                        <ToggleGroup
                          value={[scorecardGrade]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== scorecardGrade
                            )
                            if (next) {
                              setScorecardGrade(next as ScorecardGradeFilter)
                            }
                          }}
                          aria-label={t('Grade')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                          className='flex-wrap justify-end'
                        >
                          <ToggleGroupItem value='all'>
                            {t('All')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='A'>A</ToggleGroupItem>
                          <ToggleGroupItem value='B'>B</ToggleGroupItem>
                          <ToggleGroupItem value='C'>C</ToggleGroupItem>
                          <ToggleGroupItem value='D'>D</ToggleGroupItem>
                        </ToggleGroup>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => generateScorecards.mutate()}
                          disabled={generateScorecards.isPending}
                        >
                          {generateScorecards.isPending ? (
                            <Spinner data-icon='inline-start' />
                          ) : (
                            <RefreshCw data-icon='inline-start' />
                          )}
                          {t('Generate Scorecards')}
                        </Button>
                      </div>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Grade')}</TableHead>
                          <TableHead>{t('Score')}</TableHead>
                          <TableHead>{t('Requests')}</TableHead>
                          <TableHead>{t('Success Rate')}</TableHead>
                          <TableHead>{t('Cache')}</TableHead>
                          <TableHead>{t('Latency')}</TableHead>
                          <TableHead>{t('Gross Profit')}</TableHead>
                          <TableHead>{t('Supply')}</TableHead>
                          <TableHead>{t('Quality')}</TableHead>
                          <TableHead>{t('Generated')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={scorecardsQuery.isLoading}
                          isEmpty={scorecards.length === 0}
                          columns={11}
                          emptyMessage={t(
                            'No supplier scorecards in this period.'
                          )}
                        >
                          {scorecards.map((scorecard) => (
                            <TableRow key={scorecard.id}>
                              <TableCell>
                                {supplierNameById(
                                  suppliers,
                                  scorecard.supplier_id
                                )}
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={scorecardGradeVariant(
                                    scorecard.grade
                                  )}
                                >
                                  {scorecard.grade || '-'}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {formatNumber(scorecard.score)}
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatNumber(scorecard.total_requests)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Success')}:{' '}
                                    {formatNumber(scorecard.success_requests)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Errors')}:{' '}
                                    {formatNumber(scorecard.error_requests)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatRate(scorecard.success_rate)}
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {formatRate(scorecard.cache_hit_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatNumber(scorecard.cache_hit_count)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {t('Average')}:{' '}
                                    {formatLatency(scorecard.avg_latency_ms)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Max')}:{' '}
                                    {formatLatency(scorecard.max_latency_ms)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatSignedQuota(
                                  scorecard.gross_profit_quota
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {t('Capacity')}:{' '}
                                    {formatTokens(
                                      scorecard.supply_capacity_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Headroom')}:{' '}
                                    {formatTokens(
                                      scorecard.supply_headroom_tokens
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-0.5'>
                                  <span>
                                    {t('Quality Score')}:{' '}
                                    {formatNumber(
                                      scorecard.avg_supply_quality_score
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Unit Cost')}:{' '}
                                    {formatUnitCost(
                                      scorecard.avg_unit_cost_quota
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {scorecard.generated_at > 0
                                  ? formatTimestampToDate(
                                      scorecard.generated_at
                                    )
                                  : '-'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent
                value='evaluations'
                className='min-h-0 overflow-auto'
              >
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5'>
                    <StatCard
                      title={t('Visible Evaluations')}
                      value={formatNumber(evaluations.length)}
                      isLoading={evaluationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Average Evaluation Score')}
                      value={formatNumber(averageEvaluationScore)}
                      isLoading={evaluationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Draft Evaluations')}
                      value={formatNumber(evaluationTotals.draft)}
                      isLoading={evaluationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Admit Recommendations')}
                      value={formatNumber(evaluationTotals.admit)}
                      isLoading={evaluationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Approved Evaluations')}
                      value={formatNumber(evaluationTotals.approved)}
                      isLoading={evaluationsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supplier Evaluations')}
                    description={t(
                      'Supplier admission evaluations generated from scorecards; review records stay inert, and Apply only updates supplier status and notes.'
                    )}
                  >
                    <div className='mb-4 flex flex-wrap justify-end gap-2'>
                      <ToggleGroup
                        value={[evaluationStatus]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== evaluationStatus
                          )
                          if (next) {
                            setEvaluationStatus(next as EvaluationStatusFilter)
                          }
                        }}
                        aria-label={t('Evaluation Status')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='draft'>
                          {t('Draft')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='approved'>
                          {t('Approved')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='rejected'>
                          {t('Rejected')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <ToggleGroup
                        value={[evaluationRecommendation]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== evaluationRecommendation
                          )
                          if (next) {
                            setEvaluationRecommendation(
                              next as EvaluationRecommendationFilter
                            )
                          }
                        }}
                        aria-label={t('Recommendation')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='admit'>
                          {t('Admit')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='observe'>
                          {t('Observe')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='reject'>
                          {t('Reject')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <ToggleGroup
                        value={[evaluationGrade]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== evaluationGrade
                          )
                          if (next) {
                            setEvaluationGrade(next as ScorecardGradeFilter)
                          }
                        }}
                        aria-label={t('Grade')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='A'>A</ToggleGroupItem>
                        <ToggleGroupItem value='B'>B</ToggleGroupItem>
                        <ToggleGroupItem value='C'>C</ToggleGroupItem>
                        <ToggleGroupItem value='D'>D</ToggleGroupItem>
                      </ToggleGroup>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => generateEvaluations.mutate()}
                        disabled={generateEvaluations.isPending}
                      >
                        {generateEvaluations.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <RefreshCw data-icon='inline-start' />
                        )}
                        {t('Generate Evaluations')}
                      </Button>
                    </div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Evaluation')}</TableHead>
                          <TableHead>{t('Recommendation')}</TableHead>
                          <TableHead>{t('Score')}</TableHead>
                          <TableHead>{t('Review')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                          <TableHead>{t('SLA Evidence')}</TableHead>
                          <TableHead>{t('Runtime Evidence')}</TableHead>
                          <TableHead>{t('Economics')}</TableHead>
                          <TableHead>{t('Supply')}</TableHead>
                          <TableHead>{t('Reason')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={evaluationsQuery.isLoading}
                          isEmpty={evaluations.length === 0}
                          columns={11}
                          emptyMessage={t(
                            'No supplier evaluations in this period.'
                          )}
                        >
                          {evaluations.map((evaluation) => (
                            <TableRow key={evaluation.id}>
                              <TableCell>
                                {supplierNameById(
                                  suppliers,
                                  evaluation.supplier_id
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>#{evaluation.id}</span>
                                  <span className='text-muted-foreground'>
                                    {t('Scorecard')} #
                                    {evaluation.supplier_scorecard_id}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(
                                      evaluation.period_start
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {formatTimestampToDate(
                                      evaluation.period_end
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={supplierEvaluationRecommendationVariant(
                                    evaluation.recommendation
                                  )}
                                >
                                  {supplierEvaluationRecommendationLabel(
                                    evaluation.recommendation,
                                    t
                                  )}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <span className='flex flex-col gap-1'>
                                  <span>{formatNumber(evaluation.score)}</span>
                                  <Badge
                                    variant={scorecardGradeVariant(
                                      evaluation.grade
                                    )}
                                  >
                                    {evaluation.grade || '-'}
                                  </Badge>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-32 flex-col gap-1'>
                                  <Badge
                                    variant={decisionStatusVariant(
                                      evaluation.status
                                    )}
                                  >
                                    {decisionStatusLabel(evaluation.status, t)}
                                  </Badge>
                                  {evaluation.reviewed_at > 0 ? (
                                    <>
                                      <span className='text-muted-foreground'>
                                        {t('User')} #
                                        {evaluation.reviewed_by || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          evaluation.reviewed_at
                                        )}
                                      </span>
                                      <span className='text-muted-foreground max-w-40 truncate'>
                                        {evaluation.review_note || '-'}
                                      </span>
                                    </>
                                  ) : (
                                    <span className='text-muted-foreground'>
                                      -
                                    </span>
                                  )}
                                  {evaluation.applied_at > 0 ? (
                                    <>
                                      <Badge variant='outline'>
                                        {t('Applied')}
                                      </Badge>
                                      <span className='text-muted-foreground'>
                                        {t('User')} #
                                        {evaluation.applied_by || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          evaluation.applied_at
                                        )}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {t('Status')}: {t('Before')}{' '}
                                        {supplierStatusLabel(
                                          evaluation.supplier_status_before,
                                          t
                                        )}{' '}
                                        / {t('After')}{' '}
                                        {supplierStatusLabel(
                                          evaluation.supplier_status_after,
                                          t
                                        )}
                                      </span>
                                      <span className='text-muted-foreground max-w-40 truncate'>
                                        {evaluation.applied_note || '-'}
                                      </span>
                                    </>
                                  ) : null}
                                </span>
                              </TableCell>
                              <TableCell>
                                {renderEvaluationActions(evaluation)}
                              </TableCell>
                              <TableCell>
                                {renderSupplierEvaluationSlaEvidence(
                                  evaluation,
                                  t
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Requests')}:{' '}
                                    {formatNumber(evaluation.total_requests)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Success Rate')}:{' '}
                                    {formatRate(evaluation.success_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Cache Hit Rate')}:{' '}
                                    {formatRate(evaluation.cache_hit_rate)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Latency')}:{' '}
                                    {formatLatency(evaluation.avg_latency_ms)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>
                                    {t('Gross Profit')}:{' '}
                                    {formatSignedQuota(
                                      evaluation.gross_profit_quota
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Unit Cost')}:{' '}
                                    {formatUnitCost(
                                      evaluation.avg_unit_cost_quota
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <span>
                                    {t('Headroom')}:{' '}
                                    {formatTokens(
                                      evaluation.supply_headroom_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Quality Score')}:{' '}
                                    {formatNumber(
                                      evaluation.avg_supply_quality_score
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell className='max-w-72 truncate'>
                                {evaluation.reason || '-'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='posture' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5'>
                    <StatCard
                      title={t('Visible Posture Recommendations')}
                      value={formatNumber(postureRecommendations.length)}
                      isLoading={postureRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Average Posture Score')}
                      value={formatNumber(averagePostureScore)}
                      isLoading={postureRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Draft Posture Reviews')}
                      value={formatNumber(postureTotals.draft)}
                      isLoading={postureRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Disable Recommendations')}
                      value={formatNumber(postureTotals.disable)}
                      isLoading={postureRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Applied Posture Changes')}
                      value={formatNumber(postureTotals.applied)}
                      isLoading={postureRecommendationsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Active Route Preferences')}
                      value={formatNumber(routePreferences.length)}
                      isLoading={routePreferencesQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supplier Posture Recommendations')}
                    description={t(
                      'Runtime supplier posture recommendations generated from scorecards and open quality or capacity insights; approved downgrade or boost applies a route preference overlay, and manual preferences remain bounded operator controls.'
                    )}
                  >
                    <div className='mb-4 flex flex-wrap justify-end gap-2'>
                      <ToggleGroup
                        value={[postureStatus]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== postureStatus
                          )
                          if (next) {
                            setPostureStatus(next as PostureStatusFilter)
                          }
                        }}
                        aria-label={t('Posture Status')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='draft'>
                          {t('Draft')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='approved'>
                          {t('Approved')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='rejected'>
                          {t('Rejected')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='applied'>
                          {t('Applied')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <ToggleGroup
                        value={[postureAction]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== postureAction
                          )
                          if (next) {
                            setPostureAction(next as PostureActionFilter)
                          }
                        }}
                        aria-label={t('Posture Action')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='boost'>
                          {t('Boost')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='observe'>
                          {t('Observe')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='downgrade'>
                          {t('Downgrade')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='disable'>
                          {t('Disable')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                      <ToggleGroup
                        value={[postureGrade]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== postureGrade
                          )
                          if (next) {
                            setPostureGrade(next as ScorecardGradeFilter)
                          }
                        }}
                        aria-label={t('Grade')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='A'>A</ToggleGroupItem>
                        <ToggleGroupItem value='B'>B</ToggleGroupItem>
                        <ToggleGroupItem value='C'>C</ToggleGroupItem>
                        <ToggleGroupItem value='D'>D</ToggleGroupItem>
                      </ToggleGroup>
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => generatePostureRecommendations.mutate()}
                        disabled={generatePostureRecommendations.isPending}
                      >
                        {generatePostureRecommendations.isPending ? (
                          <Spinner data-icon='inline-start' />
                        ) : (
                          <RefreshCw data-icon='inline-start' />
                        )}
                        {t('Generate Posture')}
                      </Button>
                    </div>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Recommendation')}</TableHead>
                          <TableHead>{t('Action')}</TableHead>
                          <TableHead>{t('Score')}</TableHead>
                          <TableHead>{t('Review')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                          <TableHead>{t('Insight Evidence')}</TableHead>
                          <TableHead>{t('Runtime Evidence')}</TableHead>
                          <TableHead>{t('Supply')}</TableHead>
                          <TableHead>{t('Status Change')}</TableHead>
                          <TableHead>{t('Reason')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={postureRecommendationsQuery.isLoading}
                          isEmpty={postureRecommendations.length === 0}
                          columns={11}
                          emptyMessage={t(
                            'No supplier posture recommendations in this period.'
                          )}
                        >
                          {postureRecommendations.map((recommendation) => {
                            const routePreference =
                              routePreferenceByRecommendationId.get(
                                recommendation.id
                              )
                            return (
                              <TableRow key={recommendation.id}>
                                <TableCell>
                                  {supplierNameById(
                                    suppliers,
                                    recommendation.supplier_id
                                  )}
                                </TableCell>
                                <TableCell>
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>#{recommendation.id}</span>
                                    <span className='text-muted-foreground'>
                                      {t('Scorecard')} #
                                      {recommendation.supplier_scorecard_id}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        recommendation.period_start
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        recommendation.period_end
                                      )}
                                    </span>
                                    {routePreference && (
                                      <Badge
                                        variant='secondary'
                                        className='h-auto max-w-36 justify-start overflow-visible py-1 text-left leading-tight whitespace-normal sm:max-w-none'
                                      >
                                        {t('Route Preference Active')} ·{' '}
                                        {formatNumber(
                                          routePreference.weight_percent
                                        )}
                                        %
                                      </Badge>
                                    )}
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <Badge
                                    variant={supplierPostureActionVariant(
                                      recommendation.recommended_action
                                    )}
                                  >
                                    {supplierPostureActionLabel(
                                      recommendation.recommended_action,
                                      t
                                    )}
                                  </Badge>
                                </TableCell>
                                <TableCell>
                                  <span className='flex flex-col gap-1'>
                                    <span>
                                      {formatNumber(recommendation.score)}
                                    </span>
                                    <Badge
                                      variant={scorecardGradeVariant(
                                        recommendation.grade
                                      )}
                                    >
                                      {recommendation.grade || '-'}
                                    </Badge>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <span className='flex min-w-36 flex-col gap-1'>
                                    <Badge
                                      variant={supplierPostureStatusVariant(
                                        recommendation.status
                                      )}
                                    >
                                      {supplierPostureStatusLabel(
                                        recommendation.status,
                                        t
                                      )}
                                    </Badge>
                                    {recommendation.reviewed_at > 0 ? (
                                      <>
                                        <span className='text-muted-foreground'>
                                          {t('User')} #
                                          {recommendation.reviewed_by || '-'}
                                        </span>
                                        <span className='text-muted-foreground'>
                                          {formatTimestampToDate(
                                            recommendation.reviewed_at
                                          )}
                                        </span>
                                        <span className='text-muted-foreground max-w-40 truncate'>
                                          {recommendation.review_note || '-'}
                                        </span>
                                      </>
                                    ) : (
                                      <span className='text-muted-foreground'>
                                        -
                                      </span>
                                    )}
                                  </span>
                                </TableCell>
                                <TableCell>
                                  {renderPostureActions(recommendation)}
                                </TableCell>
                                <TableCell>
                                  <span className='flex min-w-44 flex-col gap-0.5'>
                                    <span>
                                      {t('Quality Insights')}:{' '}
                                      {formatNumber(
                                        recommendation.quality_insight_count
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Capacity Insights')}:{' '}
                                      {formatNumber(
                                        recommendation.capacity_insight_count
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Action Severity Insights')}:{' '}
                                      {formatNumber(
                                        recommendation.action_insight_count
                                      )}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <span className='flex min-w-44 flex-col gap-0.5'>
                                    <span>
                                      {t('Requests')}:{' '}
                                      {formatNumber(
                                        recommendation.total_requests
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Success Rate')}:{' '}
                                      {formatRate(recommendation.success_rate)}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Latency')}:{' '}
                                      {formatLatency(
                                        recommendation.avg_latency_ms
                                      )}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {t('Headroom')}:{' '}
                                      {formatTokens(
                                        recommendation.supply_headroom_tokens
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Quality Score')}:{' '}
                                      {formatNumber(
                                        recommendation.avg_supply_quality_score
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Current Status')}:{' '}
                                      {supplierStatusLabel(
                                        recommendation.supplier_status_current,
                                        t
                                      )}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  {recommendation.applied_at > 0 ? (
                                    <span className='flex min-w-44 flex-col gap-0.5'>
                                      <Badge variant='outline'>
                                        {t('Applied')}
                                      </Badge>
                                      <span className='text-muted-foreground'>
                                        {t('User')} #
                                        {recommendation.applied_by || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          recommendation.applied_at
                                        )}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {t('Before')}{' '}
                                        {supplierStatusLabel(
                                          recommendation.supplier_status_before,
                                          t
                                        )}{' '}
                                        / {t('After')}{' '}
                                        {supplierStatusLabel(
                                          recommendation.supplier_status_after,
                                          t
                                        )}
                                      </span>
                                      <span className='text-muted-foreground max-w-40 truncate'>
                                        {recommendation.applied_note || '-'}
                                      </span>
                                    </span>
                                  ) : (
                                    <span className='text-muted-foreground'>
                                      -
                                    </span>
                                  )}
                                </TableCell>
                                <TableCell className='max-w-72 truncate'>
                                  {recommendation.reason || '-'}
                                </TableCell>
                              </TableRow>
                            )
                          })}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>

                  <DataPanel
                    title={t('Active Route Preferences')}
                    description={t(
                      'Current supplier-level routing overlays; manual preferences are bounded and never change channel weights, ability weights, pricing, billing, or settlement.'
                    )}
                    action={
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => openRoutePreferenceDialog()}
                        disabled={
                          routePreferencesQuery.isLoading ||
                          enabledSuppliers.length === 0
                        }
                      >
                        <Plus data-icon='inline-start' />
                        {t('Set Route Preference')}
                      </Button>
                    }
                  >
                    <div className='hidden md:block'>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>{t('Supplier')}</TableHead>
                            <TableHead>{t('Preference')}</TableHead>
                            <TableHead>{t('Details')}</TableHead>
                            <TableHead>{t('Actions')}</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          <TableRowsState
                            isLoading={routePreferencesQuery.isLoading}
                            isEmpty={routePreferences.length === 0}
                            columns={4}
                            emptyMessage={t(
                              'No active supplier route preferences.'
                            )}
                          >
                            {routePreferences.map((preference) => (
                              <TableRow key={preference.id}>
                                <TableCell>
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {supplierNameById(
                                        suppliers,
                                        preference.supplier_id
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      #{preference.supplier_id}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <span className='flex min-w-36 flex-col gap-1'>
                                    <span className='flex flex-wrap items-center gap-2'>
                                      <Badge variant='secondary'>
                                        {t('Active')}
                                      </Badge>
                                      <span className='font-medium tabular-nums'>
                                        {formatNumber(
                                          preference.weight_percent
                                        )}
                                        %
                                      </span>
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {supplierRoutePreferenceSourceLabel(
                                        preference,
                                        t
                                      )}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <span className='flex min-w-72 flex-col gap-0.5'>
                                    <span>
                                      {t('From')}{' '}
                                      {preference.effective_from > 0
                                        ? formatTimestampToDate(
                                            preference.effective_from
                                          )
                                        : '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('To')}{' '}
                                      {preference.effective_to > 0
                                        ? formatTimestampToDate(
                                            preference.effective_to
                                          )
                                        : t('Open-ended')}
                                      {' · '}
                                      {t('Activated by')} #
                                      {preference.activated_by || '-'}
                                    </span>
                                    <span className='text-muted-foreground max-w-72 truncate'>
                                      {preference.operator_note || '-'}
                                    </span>
                                    <span className='max-w-72 truncate'>
                                      {preference.reason || '-'}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <Button
                                    variant='outline'
                                    size='sm'
                                    disabled={disableRoutePreference.isPending}
                                    onClick={() =>
                                      setRoutePreferenceToDisable(preference)
                                    }
                                  >
                                    <Trash2 data-icon='inline-start' />
                                    {t('Disable')}
                                  </Button>
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableRowsState>
                        </TableBody>
                      </Table>
                    </div>
                    <div className='flex flex-col gap-3 md:hidden'>
                      {routePreferencesQuery.isLoading && (
                        <Skeleton className='h-36 w-full' />
                      )}
                      {!routePreferencesQuery.isLoading &&
                        routePreferences.length === 0 && (
                          <div className='text-muted-foreground rounded-md border p-3 text-sm'>
                            {t('No active supplier route preferences.')}
                          </div>
                        )}
                      {!routePreferencesQuery.isLoading &&
                        routePreferences.length > 0 &&
                        routePreferences.map((preference) => (
                          <div
                            key={preference.id}
                            className='flex flex-col gap-3 rounded-md border p-3'
                          >
                            <div className='flex items-start justify-between gap-3'>
                              <span className='flex min-w-0 flex-col gap-0.5'>
                                <span className='truncate'>
                                  {supplierNameById(
                                    suppliers,
                                    preference.supplier_id
                                  )}
                                </span>
                                <span className='text-muted-foreground'>
                                  #{preference.supplier_id}
                                </span>
                              </span>
                              <Button
                                variant='outline'
                                size='sm'
                                disabled={disableRoutePreference.isPending}
                                onClick={() =>
                                  setRoutePreferenceToDisable(preference)
                                }
                              >
                                <Trash2 data-icon='inline-start' />
                                {t('Disable')}
                              </Button>
                            </div>
                            <div className='flex flex-wrap items-center gap-2'>
                              <Badge variant='secondary'>{t('Active')}</Badge>
                              <span className='font-medium tabular-nums'>
                                {formatNumber(preference.weight_percent)}%
                              </span>
                              <span className='text-muted-foreground'>
                                {supplierRoutePreferenceSourceLabel(
                                  preference,
                                  t
                                )}
                              </span>
                            </div>
                            <span className='flex flex-col gap-0.5'>
                              <span>
                                {t('From')}{' '}
                                {preference.effective_from > 0
                                  ? formatTimestampToDate(
                                      preference.effective_from
                                    )
                                  : '-'}
                              </span>
                              <span className='text-muted-foreground'>
                                {t('To')}{' '}
                                {preference.effective_to > 0
                                  ? formatTimestampToDate(
                                      preference.effective_to
                                    )
                                  : t('Open-ended')}
                                {' · '}
                                {t('Activated by')} #
                                {preference.activated_by || '-'}
                              </span>
                              <span className='text-muted-foreground break-words'>
                                {preference.operator_note || '-'}
                              </span>
                              <span className='break-words'>
                                {preference.reason || '-'}
                              </span>
                            </span>
                          </div>
                        ))}
                    </div>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='decisions' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
                    <StatCard
                      title={t('Visible Decisions')}
                      value={formatNumber(decisions.length)}
                      isLoading={decisionsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Recommended Capacity')}
                      value={formatTokens(decisionTotals.recommended)}
                      isLoading={decisionsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Open Gap')}
                      value={formatTokens(decisionTotals.gap)}
                      isLoading={decisionsQuery.isLoading}
                    />
                    <StatCard
                      title={t('ROI Score')}
                      value={formatNumber(decisionTotals.roi)}
                      isLoading={decisionsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supply Decisions')}
                    description={t(
                      'Human-reviewed supply recommendations from TrafficProfile; no automatic routing or purchasing is executed.'
                    )}
                    action={
                      <div className='flex flex-wrap justify-end gap-2'>
                        <ToggleGroup
                          value={[decisionStatus]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== decisionStatus
                            )
                            if (next) {
                              setDecisionStatus(next as SupplyDecisionStatus)
                            }
                          }}
                          aria-label={t('Decision Status')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                        >
                          <ToggleGroupItem value='draft'>
                            {t('Draft')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='approved'>
                            {t('Approved')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='rejected'>
                            {t('Rejected')}
                          </ToggleGroupItem>
                        </ToggleGroup>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => generateDecisions.mutate()}
                          disabled={generateDecisions.isPending}
                        >
                          {generateDecisions.isPending ? (
                            <Spinner data-icon='inline-start' />
                          ) : (
                            <RefreshCw data-icon='inline-start' />
                          )}
                          {t('Generate Decisions')}
                        </Button>
                      </div>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Track')}</TableHead>
                          <TableHead>{t('Decision')}</TableHead>
                          <TableHead>{t('Source')}</TableHead>
                          <TableHead>{t('Demand')}</TableHead>
                          <TableHead>{t('Headroom')}</TableHead>
                          <TableHead>{t('Gap')}</TableHead>
                          <TableHead>{t('Recommended Capacity')}</TableHead>
                          <TableHead>{t('ROI Score')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Review')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={decisionsQuery.isLoading}
                          isEmpty={decisions.length === 0}
                          columns={12}
                          emptyMessage={t(
                            'No supply decisions in this period.'
                          )}
                        >
                          {decisions.map((decision) => (
                            <TableRow key={decision.id}>
                              <TableCell>
                                <span className='flex max-w-64 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {decision.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {decision.sla_tier || '-'} / {t('User')} #
                                    {decision.user_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {decisionTrackLabel(decision.track, t)}
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-64 flex-col gap-0.5'>
                                  <span>
                                    {decisionTypeLabel(
                                      decision.decision_type,
                                      t
                                    )}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {decision.reason || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <Badge
                                    variant={
                                      decision.decision_source === 'forecast'
                                        ? 'default'
                                        : 'secondary'
                                    }
                                  >
                                    {decision.decision_source === 'forecast'
                                      ? t('Forecast')
                                      : t('Profile')}
                                  </Badge>
                                  {decision.decision_source === 'forecast' ? (
                                    <>
                                      <span className='text-muted-foreground'>
                                        {t('Target Period')}
                                      </span>
                                      <span>
                                        {formatTimestampToDate(
                                          decision.forecast_target_period_start
                                        )}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          decision.forecast_target_period_end
                                        )}
                                      </span>
                                      <span className='text-muted-foreground tabular-nums'>
                                        {t('Confidence')}:{' '}
                                        {formatRate(
                                          decision.forecast_confidence
                                        )}
                                      </span>
                                      <span className='text-muted-foreground max-w-40 truncate'>
                                        {decision.forecast_method || '-'}
                                      </span>
                                    </>
                                  ) : (
                                    <span className='text-muted-foreground'>
                                      #{decision.traffic_profile_id || '-'}
                                    </span>
                                  )}
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatTokens(decision.demand_tokens)}
                              </TableCell>
                              <TableCell>
                                {formatTokens(decision.supply_headroom_tokens)}
                              </TableCell>
                              <TableCell>
                                {formatTokens(decision.gap_tokens)}
                              </TableCell>
                              <TableCell>
                                {formatTokens(decision.recommended_capacity)}
                              </TableCell>
                              <TableCell>
                                {formatNumber(decision.roi_score)}
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={decisionStatusVariant(
                                    decision.status
                                  )}
                                >
                                  {decisionStatusLabel(decision.status, t)}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {decision.reviewed_at > 0 ? (
                                  <span className='flex flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #{decision.reviewed_by || '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        decision.reviewed_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {decision.status === 'draft' ? (
                                  <div className='flex flex-wrap gap-2'>
                                    <Button
                                      variant='outline'
                                      size='sm'
                                      disabled={
                                        approveDecision.isPending ||
                                        rejectDecision.isPending
                                      }
                                      onClick={() =>
                                        approveDecision.mutate(decision.id)
                                      }
                                    >
                                      {t('Approve')}
                                    </Button>
                                    <Button
                                      variant='outline'
                                      size='sm'
                                      disabled={
                                        approveDecision.isPending ||
                                        rejectDecision.isPending
                                      }
                                      onClick={() =>
                                        rejectDecision.mutate(decision.id)
                                      }
                                    >
                                      {t('Reject')}
                                    </Button>
                                  </div>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent
                value='opportunities'
                className='min-h-0 overflow-auto'
              >
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
                    <StatCard
                      title={t('Visible Opportunities')}
                      value={formatNumber(opportunities.length)}
                      isLoading={opportunitiesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Action Opportunities')}
                      value={formatNumber(opportunityTotals.action)}
                      isLoading={opportunitiesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Recommended Capacity')}
                      value={formatTokens(opportunityTotals.recommended)}
                      isLoading={opportunitiesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Rank Score')}
                      value={formatNumber(opportunityTotals.rank)}
                      isLoading={opportunitiesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Self-hosted Savings')}
                      value={formatSignedQuota(opportunityTotals.savings)}
                      isLoading={opportunitiesQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supply Opportunities')}
                    description={t(
                      'Ranked supply expansion read model from SupplyDecision and optional self-hosted cost-profile evidence; generate/query records analysis only and does not create action plans, routing policies, suppliers, capacity, billing, settlement, or payments.'
                    )}
                    action={
                      <div className='flex flex-wrap justify-end gap-2'>
                        <ToggleGroup
                          value={[opportunityType]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== opportunityType
                            )
                            if (next) {
                              setOpportunityType(next as OpportunityTypeFilter)
                            }
                          }}
                          aria-label={t('Opportunity Type')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                          className='flex-wrap justify-end'
                        >
                          <ToggleGroupItem value='all'>
                            {t('All')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='self_hosted_cache'>
                            {t('Self-hosted Cache')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='third_party_gap'>
                            {t('Third-party Gap')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='self_operated_bulk'>
                            {t('Self-operated Bulk')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='third_party_probe'>
                            {t('Third-party Probe')}
                          </ToggleGroupItem>
                        </ToggleGroup>
                        <ToggleGroup
                          value={[opportunityPriority]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== opportunityPriority
                            )
                            if (next) {
                              setOpportunityPriority(
                                next as OpportunityPriorityFilter
                              )
                            }
                          }}
                          aria-label={t('Opportunity Priority')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                          className='flex-wrap justify-end'
                        >
                          <ToggleGroupItem value='all'>
                            {t('All')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='action'>
                            {t('Action')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='watch'>
                            {t('Watch')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='info'>
                            {t('Info')}
                          </ToggleGroupItem>
                        </ToggleGroup>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => generateOpportunities.mutate()}
                          disabled={generateOpportunities.isPending}
                        >
                          {generateOpportunities.isPending ? (
                            <Spinner data-icon='inline-start' />
                          ) : (
                            <RefreshCw data-icon='inline-start' />
                          )}
                          {t('Generate Opportunities')}
                        </Button>
                      </div>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Opportunity')}</TableHead>
                          <TableHead>{t('Priority')}</TableHead>
                          <TableHead>{t('Decision')}</TableHead>
                          <TableHead>{t('Source')}</TableHead>
                          <TableHead>{t('Demand')}</TableHead>
                          <TableHead>{t('Headroom')}</TableHead>
                          <TableHead>{t('Gap')}</TableHead>
                          <TableHead>{t('Recommended Capacity')}</TableHead>
                          <TableHead>{t('Cost Evidence')}</TableHead>
                          <TableHead>{t('Scores')}</TableHead>
                          <TableHead>{t('Reason')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={opportunitiesQuery.isLoading}
                          isEmpty={opportunities.length === 0}
                          columns={12}
                          emptyMessage={t(
                            'No supply opportunities in this period.'
                          )}
                        >
                          {opportunities.map((opportunity) => (
                            <TableRow key={opportunity.id}>
                              <TableCell>
                                <span className='flex max-w-64 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {opportunity.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {opportunity.sla_tier || '-'} / {t('User')}{' '}
                                    #{opportunity.user_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-48 flex-col gap-1'>
                                  <Badge variant='secondary'>
                                    {opportunityTypeLabel(
                                      opportunity.opportunity_type,
                                      t
                                    )}
                                  </Badge>
                                  <span className='text-muted-foreground'>
                                    {opportunityClusterLabel(
                                      opportunity.cluster_key,
                                      t
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-32 flex-col gap-1'>
                                  <Badge
                                    variant={opportunityPriorityVariant(
                                      opportunity.priority
                                    )}
                                  >
                                    {opportunityPriorityLabel(
                                      opportunity.priority,
                                      t
                                    )}
                                  </Badge>
                                  <span className='text-muted-foreground'>
                                    {decisionStatusLabel(
                                      opportunity.decision_status,
                                      t
                                    )}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-52 flex-col gap-0.5'>
                                  <span>
                                    {decisionTrackLabel(opportunity.track, t)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {decisionTypeLabel(
                                      opportunity.decision_type,
                                      t
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    #{opportunity.supply_decision_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-40 flex-col gap-0.5'>
                                  <Badge
                                    variant={
                                      opportunity.decision_source === 'forecast'
                                        ? 'default'
                                        : 'secondary'
                                    }
                                  >
                                    {opportunity.decision_source === 'forecast'
                                      ? t('Forecast')
                                      : t('Profile')}
                                  </Badge>
                                  {opportunity.decision_source ===
                                  'forecast' ? (
                                    <>
                                      <span className='text-muted-foreground'>
                                        {t('Target Period')}
                                      </span>
                                      <span>
                                        {formatTimestampToDate(
                                          opportunity.forecast_target_period_start
                                        )}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          opportunity.forecast_target_period_end
                                        )}
                                      </span>
                                      <span className='text-muted-foreground tabular-nums'>
                                        {t('Confidence')}:{' '}
                                        {formatRate(
                                          opportunity.forecast_confidence
                                        )}
                                      </span>
                                      <span className='text-muted-foreground max-w-40 truncate'>
                                        {opportunity.forecast_method || '-'}
                                      </span>
                                    </>
                                  ) : (
                                    <span className='text-muted-foreground'>
                                      #{opportunity.traffic_profile_id || '-'}
                                    </span>
                                  )}
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatTokens(opportunity.demand_tokens)}
                              </TableCell>
                              <TableCell>
                                {formatTokens(
                                  opportunity.supply_headroom_tokens
                                )}
                              </TableCell>
                              <TableCell>
                                {formatTokens(opportunity.gap_tokens)}
                              </TableCell>
                              <TableCell>
                                {formatTokens(opportunity.recommended_capacity)}
                              </TableCell>
                              <TableCell>
                                {opportunity.self_hosted_cost_profile_id > 0 ? (
                                  <span className='flex min-w-48 flex-col gap-0.5 tabular-nums'>
                                    <span>
                                      {t('Cost Profile')} #
                                      {opportunity.self_hosted_cost_profile_id}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Self-hosted Unit')}:{' '}
                                      {formatUnitCost(
                                        opportunity.self_hosted_unit_cost_quota
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Unit Savings')}:{' '}
                                      {formatSignedQuota(
                                        opportunity.self_hosted_savings_unit_quota
                                      )}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Total Savings')}:{' '}
                                      {formatSignedQuota(
                                        opportunity.self_hosted_savings_quota
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  <span className='text-muted-foreground'>
                                    {t('No cost profile')}
                                  </span>
                                )}
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5 tabular-nums'>
                                  <span>
                                    {t('Locality')}:{' '}
                                    {formatRate(opportunity.locality_score)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Stability')}:{' '}
                                    {formatRate(opportunity.stability_score)}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Headroom Risk')}:{' '}
                                    {formatRate(
                                      opportunity.headroom_risk_score
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Rank Score')}:{' '}
                                    {formatNumber(opportunity.rank_score)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell className='max-w-72 truncate'>
                                {opportunity.reason || '-'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='actions' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
                    <StatCard
                      title={t('Visible Action Plans')}
                      value={formatNumber(actionPlans.length)}
                      isLoading={actionPlansQuery.isLoading}
                    />
                    <StatCard
                      title={t('Recommended Capacity')}
                      value={formatTokens(actionPlanTotals.recommended)}
                      isLoading={actionPlansQuery.isLoading}
                    />
                    <StatCard
                      title={t('Open Gap')}
                      value={formatTokens(actionPlanTotals.gap)}
                      isLoading={actionPlansQuery.isLoading}
                    />
                    <StatCard
                      title={t('ROI Score')}
                      value={formatNumber(actionPlanTotals.roi)}
                      isLoading={actionPlansQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supply Action Plans')}
                    description={t(
                      'Operator-managed work items from approved supply decisions; execution stays outside automatic routing, purchasing, and supplier mutation.'
                    )}
                    action={
                      <div className='flex flex-wrap justify-end gap-2'>
                        <ToggleGroup
                          value={[actionStatus]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== actionStatus
                            )
                            if (next) {
                              setActionStatus(next as ActionPlanStatusFilter)
                            }
                          }}
                          aria-label={t('Status')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                          className='flex-wrap justify-end'
                        >
                          <ToggleGroupItem value='all'>
                            {t('All')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='planned'>
                            {t('Planned')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='in_progress'>
                            {t('In progress')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='completed'>
                            {t('Completed')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='cancelled'>
                            {t('Cancelled')}
                          </ToggleGroupItem>
                        </ToggleGroup>
                        <ToggleGroup
                          value={[actionTrack]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== actionTrack
                            )
                            if (next) {
                              setActionTrack(next as TrackFilter)
                            }
                          }}
                          aria-label={t('Track')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                          className='flex-wrap justify-end'
                        >
                          <ToggleGroupItem value='all'>
                            {t('All')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='third_party'>
                            {t('Third-party')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='self_operated'>
                            {t('Self-operated')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='self_hosted'>
                            {t('Self-hosted')}
                          </ToggleGroupItem>
                        </ToggleGroup>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => generateActionPlans.mutate()}
                          disabled={generateActionPlans.isPending}
                        >
                          {generateActionPlans.isPending ? (
                            <Spinner data-icon='inline-start' />
                          ) : (
                            <RefreshCw data-icon='inline-start' />
                          )}
                          {t('Generate Action Plans')}
                        </Button>
                      </div>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Action')}</TableHead>
                          <TableHead>{t('Opportunity')}</TableHead>
                          <TableHead>{t('Track')}</TableHead>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Recommended Capacity')}</TableHead>
                          <TableHead>{t('Gap')}</TableHead>
                          <TableHead>{t('ROI Score')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Operator')}</TableHead>
                          <TableHead>{t('Lifecycle')}</TableHead>
                          <TableHead>{t('Source Review')}</TableHead>
                          <TableHead>{t('Generated')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={actionPlansQuery.isLoading}
                          isEmpty={actionPlans.length === 0}
                          columns={13}
                          emptyMessage={t(
                            'No supply action plans in this period.'
                          )}
                        >
                          {actionPlans.map((plan) => (
                            <TableRow key={plan.id}>
                              <TableCell>
                                <span className='flex max-w-72 flex-col gap-0.5'>
                                  <span>
                                    {actionTypeLabel(plan.action_type, t)}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {plan.reason || '-'}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {plan.supply_expansion_opportunity_id > 0 ? (
                                  <span className='flex max-w-64 flex-col gap-1'>
                                    <span className='flex flex-wrap gap-1'>
                                      <Badge variant='outline'>
                                        {opportunityTypeLabel(
                                          plan.opportunity_type,
                                          t
                                        )}
                                      </Badge>
                                      <Badge
                                        variant={opportunityPriorityVariant(
                                          plan.opportunity_priority
                                        )}
                                      >
                                        {opportunityPriorityLabel(
                                          plan.opportunity_priority,
                                          t
                                        )}
                                      </Badge>
                                    </span>
                                    <span className='text-muted-foreground truncate'>
                                      {opportunityClusterLabel(
                                        plan.opportunity_cluster_key,
                                        t
                                      )}
                                      {' / '}
                                      {t('Rank Score')}:{' '}
                                      {formatNumber(
                                        plan.opportunity_rank_score
                                      )}
                                    </span>
                                    <span className='text-muted-foreground truncate'>
                                      #{plan.supply_expansion_opportunity_id}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {decisionTrackLabel(plan.track, t)}
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-64 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {plan.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {plan.sla_tier || '-'} / {t('User')} #
                                    {plan.user_id}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {t('Decision')} #{plan.supply_decision_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {formatTokens(plan.recommended_capacity)}
                              </TableCell>
                              <TableCell>
                                {formatTokens(plan.gap_tokens)}
                              </TableCell>
                              <TableCell>
                                {formatNumber(plan.roi_score)}
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={actionPlanStatusVariant(plan.status)}
                                >
                                  {actionPlanStatusLabel(plan.status, t)}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {plan.status_updated_at > 0 ||
                                plan.operator_note ? (
                                  <span className='flex max-w-56 flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #
                                      {plan.status_updated_by || '-'}
                                    </span>
                                    {plan.status_updated_at > 0 && (
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          plan.status_updated_at
                                        )}
                                      </span>
                                    )}
                                    <span className='text-muted-foreground truncate'>
                                      {plan.operator_note || '-'}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {plan.started_at > 0 ||
                                plan.completed_at > 0 ||
                                plan.cancelled_at > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    {plan.started_at > 0 && (
                                      <span>
                                        {t('Started')}:{' '}
                                        {formatTimestampToDate(plan.started_at)}
                                      </span>
                                    )}
                                    {plan.completed_at > 0 && (
                                      <span>
                                        {t('Completed')}:{' '}
                                        {formatTimestampToDate(
                                          plan.completed_at
                                        )}
                                      </span>
                                    )}
                                    {plan.cancelled_at > 0 && (
                                      <span>
                                        {t('Cancelled')}:{' '}
                                        {formatTimestampToDate(
                                          plan.cancelled_at
                                        )}
                                      </span>
                                    )}
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {plan.source_reviewed_at > 0 ? (
                                  <span className='flex flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #
                                      {plan.source_reviewed_by || '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        plan.source_reviewed_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {plan.generated_at > 0
                                  ? formatTimestampToDate(plan.generated_at)
                                  : '-'}
                              </TableCell>
                              <TableCell>
                                {(() => {
                                  if (
                                    nextActionPlanStatuses(plan.status).length >
                                    0
                                  ) {
                                    return (
                                      <Button
                                        variant='outline'
                                        size='sm'
                                        onClick={() =>
                                          openActionPlanStatusDialog(plan)
                                        }
                                        disabled={
                                          updateActionPlanStatus.isPending
                                        }
                                      >
                                        <Pencil data-icon='inline-start' />
                                        {t('Update Status')}
                                      </Button>
                                    )
                                  }
                                  if (plan.status === 'completed') {
                                    return (
                                      <Button
                                        variant='outline'
                                        size='sm'
                                        onClick={() =>
                                          openActionExecutionRecordDialog(plan)
                                        }
                                        disabled={
                                          recordActionExecution.isPending
                                        }
                                      >
                                        <Pencil data-icon='inline-start' />
                                        {t('Record Execution')}
                                      </Button>
                                    )
                                  }
                                  return '-'
                                })()}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='executions' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5'>
                    <StatCard
                      title={t('Visible Executions')}
                      value={formatNumber(actionExecutions.length)}
                      isLoading={actionExecutionsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Actual Capacity')}
                      value={formatTokens(actionExecutionTotals.actual)}
                      isLoading={actionExecutionsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Recommended Capacity')}
                      value={formatTokens(actionExecutionTotals.recommended)}
                      isLoading={actionExecutionsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Drawdown Tokens')}
                      value={formatTokens(actionExecutionTotals.drawdown)}
                      isLoading={actionExecutionsQuery.isLoading}
                    />
                    <StatCard
                      title={t('Average Unit Cost')}
                      value={formatUnitCost(averageExecutionUnitCost)}
                      isLoading={actionExecutionsQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supply Action Executions')}
                    description={t(
                      'Operator-recorded execution facts for completed action plans; this view does not create suppliers, mutate capacity, route traffic, or touch payments.'
                    )}
                    action={
                      <div className='flex flex-wrap justify-end gap-2'>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => refreshActionExecutionUsage.mutate()}
                          disabled={
                            refreshActionExecutionUsage.isPending ||
                            actionExecutionsQuery.isLoading
                          }
                        >
                          {refreshActionExecutionUsage.isPending ? (
                            <Spinner data-icon='inline-start' />
                          ) : (
                            <RefreshCw data-icon='inline-start' />
                          )}
                          {t('Refresh Drawdown')}
                        </Button>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => openActionExecutionRecordDialog()}
                          disabled={
                            recordActionExecution.isPending ||
                            executionSourcePlansQuery.isLoading ||
                            completedActionPlans.length === 0
                          }
                        >
                          {recordActionExecution.isPending ? (
                            <Spinner data-icon='inline-start' />
                          ) : (
                            <Pencil data-icon='inline-start' />
                          )}
                          {t('Record Execution')}
                        </Button>
                        <ToggleGroup
                          value={[executionStatus]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== executionStatus
                            )
                            if (next) {
                              setExecutionStatus(next as ExecutionStatusFilter)
                            }
                          }}
                          aria-label={t('Execution Status')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                          className='flex-wrap justify-end'
                        >
                          <ToggleGroupItem value='all'>
                            {t('All')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='recorded'>
                            {t('Recorded')}
                          </ToggleGroupItem>
                        </ToggleGroup>
                        <ToggleGroup
                          value={[executionTrack]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== executionTrack
                            )
                            if (next) {
                              setExecutionTrack(next as TrackFilter)
                            }
                          }}
                          aria-label={t('Track')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                          className='flex-wrap justify-end'
                        >
                          <ToggleGroupItem value='all'>
                            {t('All')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='third_party'>
                            {t('Third-party')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='self_operated'>
                            {t('Self-operated')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='self_hosted'>
                            {t('Self-hosted')}
                          </ToggleGroupItem>
                        </ToggleGroup>
                      </div>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Action')}</TableHead>
                          <TableHead>{t('Execution Status')}</TableHead>
                          <TableHead>{t('Track')}</TableHead>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Channel')}</TableHead>
                          <TableHead>{t('Capacity Snapshot')}</TableHead>
                          <TableHead>{t('Capacity')}</TableHead>
                          <TableHead>{t('Drawdown')}</TableHead>
                          <TableHead>{t('Unit Cost')}</TableHead>
                          <TableHead>{t('Effective Period')}</TableHead>
                          <TableHead>{t('External Ref')}</TableHead>
                          <TableHead>{t('Recorded')}</TableHead>
                          <TableHead>{t('Source Completion')}</TableHead>
                          <TableHead>{t('Operator Note')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={actionExecutionsQuery.isLoading}
                          isEmpty={actionExecutions.length === 0}
                          columns={15}
                          emptyMessage={t(
                            'No supply action executions in this period.'
                          )}
                        >
                          {actionExecutions.map((execution) => (
                            <TableRow key={execution.id}>
                              <TableCell>
                                <span className='flex max-w-72 flex-col gap-0.5'>
                                  <span>
                                    {actionTypeLabel(execution.action_type, t)}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {t('Action Plan')} #
                                    {execution.supply_action_plan_id} /{' '}
                                    {t('Decision')} #
                                    {execution.supply_decision_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={executionStatusVariant(
                                    execution.execution_status
                                  )}
                                >
                                  {executionStatusLabel(
                                    execution.execution_status,
                                    t
                                  )}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {decisionTrackLabel(execution.track, t)}
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-64 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {execution.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {execution.sla_tier || '-'} / {t('User')} #
                                    {execution.user_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {execution.supplier_id > 0
                                  ? supplierNameById(
                                      suppliers,
                                      execution.supplier_id
                                    )
                                  : '-'}
                              </TableCell>
                              <TableCell>
                                {execution.channel_id > 0
                                  ? `#${execution.channel_id}`
                                  : '-'}
                              </TableCell>
                              <TableCell>
                                {execution.supply_capacity_id > 0
                                  ? `#${execution.supply_capacity_id}`
                                  : '-'}
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Actual')}:{' '}
                                    {formatTokens(
                                      execution.actual_capacity_tokens
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Recommended')}:{' '}
                                    {formatTokens(
                                      execution.recommended_capacity
                                    )}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Gap')}:{' '}
                                    {formatTokens(execution.gap_tokens)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {execution.drawdown_refreshed_at > 0 ? (
                                  <span className='flex min-w-48 flex-col gap-0.5'>
                                    <span>
                                      {t('Used')}:{' '}
                                      {formatTokens(execution.drawdown_tokens)}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Remaining')}:{' '}
                                      {formatTokens(execution.remaining_tokens)}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {t('Requests')}:{' '}
                                      {formatNumber(
                                        execution.drawdown_request_count
                                      )}{' '}
                                      / {formatRate(execution.drawdown_rate)}
                                    </span>
                                    <span className='text-muted-foreground truncate'>
                                      {execution.drawdown_source_type || '-'} /{' '}
                                      {formatTimestampToDate(
                                        execution.drawdown_refreshed_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {formatUnitCost(execution.unit_cost_quota)}
                              </TableCell>
                              <TableCell>
                                {execution.effective_from > 0 ||
                                execution.effective_to > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {execution.effective_from > 0
                                        ? formatTimestampToDate(
                                            execution.effective_from
                                          )
                                        : '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {execution.effective_to > 0
                                        ? formatTimestampToDate(
                                            execution.effective_to
                                          )
                                        : '-'}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell className='max-w-56 truncate'>
                                {execution.external_ref || '-'}
                              </TableCell>
                              <TableCell>
                                {execution.recorded_at > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #
                                      {execution.recorded_by || '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        execution.recorded_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {execution.action_plan_completed_at > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #
                                      {execution.action_plan_completed_by ||
                                        '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        execution.action_plan_completed_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell className='max-w-56 truncate'>
                                {execution.operator_note || '-'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='routing' className='min-h-0 overflow-auto'>
                <div className='flex flex-col gap-4'>
                  <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4'>
                    <StatCard
                      title={t('Visible Policies')}
                      value={formatNumber(routingPolicies.length)}
                      isLoading={routingPoliciesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Active Policies')}
                      value={formatNumber(routingPolicyTotals.active)}
                      isLoading={routingPoliciesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Disabled Policies')}
                      value={formatNumber(routingPolicyTotals.disabled)}
                      isLoading={routingPoliciesQuery.isLoading}
                    />
                    <StatCard
                      title={t('Self-hosted Routes')}
                      value={formatNumber(routingPolicyTotals.selfHosted)}
                      isLoading={routingPoliciesQuery.isLoading}
                    />
                  </div>

                  <DataPanel
                    title={t('Supply Routing Policies')}
                    description={t(
                      'Human-activated routing preferences from recorded self-hosted executions; they do not create suppliers, mutate channels, or touch payments.'
                    )}
                    action={
                      <ToggleGroup
                        value={[routingPolicyStatus]}
                        onValueChange={(value) => {
                          const next = value.find(
                            (item) => item !== routingPolicyStatus
                          )
                          if (next) {
                            setRoutingPolicyStatus(
                              next as RoutingPolicyStatusFilter
                            )
                          }
                        }}
                        aria-label={t('Routing Policy Status')}
                        variant='outline'
                        size='sm'
                        spacing={2}
                        className='flex-wrap justify-end'
                      >
                        <ToggleGroupItem value='all'>
                          {t('All')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='active'>
                          {t('Active')}
                        </ToggleGroupItem>
                        <ToggleGroupItem value='disabled'>
                          {t('Disabled')}
                        </ToggleGroupItem>
                      </ToggleGroup>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Policy')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Source')}</TableHead>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Channel')}</TableHead>
                          <TableHead>{t('Capacity Snapshot')}</TableHead>
                          <TableHead>{t('SLA Evidence')}</TableHead>
                          <TableHead>{t('Priority')}</TableHead>
                          <TableHead>{t('Traffic')}</TableHead>
                          <TableHead>{t('Effective Period')}</TableHead>
                          <TableHead>{t('Activated')}</TableHead>
                          <TableHead>{t('Disabled')}</TableHead>
                          <TableHead>{t('Operator Note')}</TableHead>
                          <TableHead>{t('Action')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={routingPoliciesQuery.isLoading}
                          isEmpty={routingPolicies.length === 0}
                          columns={15}
                          emptyMessage={t(
                            'No supply routing policies in this period.'
                          )}
                        >
                          {routingPolicies.map((policy) => (
                            <TableRow key={policy.id}>
                              <TableCell>
                                <span className='flex max-w-56 flex-col gap-0.5'>
                                  <span>#{policy.id}</span>
                                  <span className='text-muted-foreground truncate'>
                                    {decisionTrackLabel(policy.track, t)}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={routingPolicyStatusVariant(
                                    policy.status
                                  )}
                                >
                                  {routingPolicyStatusLabel(policy.status, t)}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <span className='flex min-w-44 flex-col gap-0.5'>
                                  <span>
                                    {t('Execution')} #
                                    {policy.supply_action_execution_id}
                                  </span>
                                  <span className='text-muted-foreground'>
                                    {t('Action Plan')} #
                                    {policy.supply_action_plan_id} /{' '}
                                    {t('Decision')} #{policy.supply_decision_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                <span className='flex max-w-64 flex-col gap-0.5'>
                                  <span className='truncate'>
                                    {policy.model_name || '-'}
                                  </span>
                                  <span className='text-muted-foreground truncate'>
                                    {policy.sla_tier || '-'} / {t('User')} #
                                    {policy.user_id}
                                  </span>
                                </span>
                              </TableCell>
                              <TableCell>
                                {policy.supplier_id > 0
                                  ? supplierNameById(
                                      suppliers,
                                      policy.supplier_id
                                    )
                                  : '-'}
                              </TableCell>
                              <TableCell>
                                {policy.channel_id > 0
                                  ? `#${policy.channel_id}`
                                  : '-'}
                              </TableCell>
                              <TableCell>
                                {policy.supply_capacity_id > 0
                                  ? `#${policy.supply_capacity_id}`
                                  : '-'}
                              </TableCell>
                              <TableCell>
                                {renderRoutingPolicySlaEvidence(policy, t)}
                              </TableCell>
                              <TableCell>
                                {formatNumber(policy.priority)}
                              </TableCell>
                              <TableCell>
                                <Badge variant='outline'>
                                  {formatNumber(
                                    routingPolicyTrafficPercent(policy)
                                  )}
                                  %
                                </Badge>
                              </TableCell>
                              <TableCell>
                                {policy.effective_from > 0 ||
                                policy.effective_to > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {policy.effective_from > 0
                                        ? formatTimestampToDate(
                                            policy.effective_from
                                          )
                                        : '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {policy.effective_to > 0
                                        ? formatTimestampToDate(
                                            policy.effective_to
                                          )
                                        : '-'}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {policy.activated_at > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #{policy.activated_by || '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        policy.activated_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {policy.disabled_at > 0 ? (
                                  <span className='flex min-w-40 flex-col gap-0.5'>
                                    <span>
                                      {t('User')} #{policy.disabled_by || '-'}
                                    </span>
                                    <span className='text-muted-foreground'>
                                      {formatTimestampToDate(
                                        policy.disabled_at
                                      )}
                                    </span>
                                  </span>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell className='max-w-56 truncate'>
                                {policy.operator_note || '-'}
                              </TableCell>
                              <TableCell>
                                {policy.status === 'active' ? (
                                  <Button
                                    variant='outline'
                                    size='sm'
                                    onClick={() => setPolicyToDisable(policy)}
                                    disabled={disableRoutingPolicy.isPending}
                                  >
                                    <Trash2 data-icon='inline-start' />
                                    {t('Disable')}
                                  </Button>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>

                  <DataPanel
                    title={t('Self-hosted Execution Sources')}
                    description={t(
                      'Recorded self-hosted executions that can be explicitly promoted into routing policy; invalid channel or supplier references remain backend-rejected.'
                    )}
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Execution')}</TableHead>
                          <TableHead>{t('Slice')}</TableHead>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Channel')}</TableHead>
                          <TableHead>{t('Capacity Snapshot')}</TableHead>
                          <TableHead>{t('Effective Period')}</TableHead>
                          <TableHead>{t('Policy')}</TableHead>
                          <TableHead>{t('Recorded')}</TableHead>
                          <TableHead>{t('Action')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={routingSourceExecutionsQuery.isLoading}
                          isEmpty={routingSourceExecutions.length === 0}
                          columns={9}
                          emptyMessage={t(
                            'No recorded self-hosted executions in this period.'
                          )}
                        >
                          {routingSourceExecutions.map((execution) => {
                            const existingPolicy = policyByExecutionId.get(
                              execution.id
                            )
                            const canActivate =
                              execution.supplier_id > 0 &&
                              execution.channel_id > 0
                            const isActivePolicy =
                              existingPolicy?.status === 'active'
                            return (
                              <TableRow key={execution.id}>
                                <TableCell>
                                  <span className='flex min-w-44 flex-col gap-0.5'>
                                    <span>#{execution.id}</span>
                                    <span className='text-muted-foreground truncate'>
                                      {actionTypeLabel(
                                        execution.action_type,
                                        t
                                      )}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <span className='flex max-w-64 flex-col gap-0.5'>
                                    <span className='truncate'>
                                      {execution.model_name || '-'}
                                    </span>
                                    <span className='text-muted-foreground truncate'>
                                      {execution.sla_tier || '-'} / {t('User')}{' '}
                                      #{execution.user_id}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  {execution.supplier_id > 0
                                    ? supplierNameById(
                                        suppliers,
                                        execution.supplier_id
                                      )
                                    : '-'}
                                </TableCell>
                                <TableCell>
                                  {execution.channel_id > 0
                                    ? `#${execution.channel_id}`
                                    : '-'}
                                </TableCell>
                                <TableCell>
                                  {execution.supply_capacity_id > 0
                                    ? `#${execution.supply_capacity_id}`
                                    : '-'}
                                </TableCell>
                                <TableCell>
                                  {execution.effective_from > 0 ||
                                  execution.effective_to > 0 ? (
                                    <span className='flex min-w-40 flex-col gap-0.5'>
                                      <span>
                                        {execution.effective_from > 0
                                          ? formatTimestampToDate(
                                              execution.effective_from
                                            )
                                          : '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {execution.effective_to > 0
                                          ? formatTimestampToDate(
                                              execution.effective_to
                                            )
                                          : '-'}
                                      </span>
                                    </span>
                                  ) : (
                                    '-'
                                  )}
                                </TableCell>
                                <TableCell>
                                  {existingPolicy ? (
                                    <span className='flex min-w-28 flex-col gap-1'>
                                      <Badge
                                        variant={routingPolicyStatusVariant(
                                          existingPolicy.status
                                        )}
                                      >
                                        {routingPolicyStatusLabel(
                                          existingPolicy.status,
                                          t
                                        )}
                                      </Badge>
                                      <span className='text-muted-foreground'>
                                        {formatNumber(
                                          routingPolicyTrafficPercent(
                                            existingPolicy
                                          )
                                        )}
                                        %
                                      </span>
                                    </span>
                                  ) : (
                                    '-'
                                  )}
                                </TableCell>
                                <TableCell>
                                  {execution.recorded_at > 0 ? (
                                    <span className='flex min-w-40 flex-col gap-0.5'>
                                      <span>
                                        {t('User')} #
                                        {execution.recorded_by || '-'}
                                      </span>
                                      <span className='text-muted-foreground'>
                                        {formatTimestampToDate(
                                          execution.recorded_at
                                        )}
                                      </span>
                                    </span>
                                  ) : (
                                    '-'
                                  )}
                                </TableCell>
                                <TableCell>
                                  {isActivePolicy ? (
                                    '-'
                                  ) : (
                                    <Button
                                      variant='outline'
                                      size='sm'
                                      onClick={() =>
                                        openRoutingPolicyActivateDialog(
                                          execution,
                                          existingPolicy
                                        )
                                      }
                                      disabled={
                                        activateRoutingPolicy.isPending ||
                                        !canActivate
                                      }
                                    >
                                      {activateRoutingPolicy.isPending ? (
                                        <Spinner data-icon='inline-start' />
                                      ) : (
                                        <Plus data-icon='inline-start' />
                                      )}
                                      {existingPolicy
                                        ? t('Reactivate Policy')
                                        : t('Activate Policy')}
                                    </Button>
                                  )}
                                </TableCell>
                              </TableRow>
                            )
                          })}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='suppliers' className='min-h-0 overflow-auto'>
                <div className='grid grid-cols-1 gap-4 xl:grid-cols-2'>
                  <DataPanel
                    title={t('Suppliers')}
                    description={t(
                      'Business-side upstream settlement parties only.'
                    )}
                    action={
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => openSupplierDialog()}
                      >
                        <Plus data-icon='inline-start' />
                        {t('Add Supplier')}
                      </Button>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('ID')}</TableHead>
                          <TableHead>{t('Name')}</TableHead>
                          <TableHead>{t('Type')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={suppliersQuery.isLoading}
                          isEmpty={suppliers.length === 0}
                          columns={5}
                          emptyMessage={t('No suppliers configured.')}
                        >
                          {suppliers.map((supplier) => (
                            <TableRow key={supplier.id}>
                              <TableCell>#{supplier.id}</TableCell>
                              <TableCell>{supplier.name}</TableCell>
                              <TableCell>
                                {supplierTypeLabel(supplier.type, t)}
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant={
                                    supplier.status === 1
                                      ? 'secondary'
                                      : 'outline'
                                  }
                                >
                                  {supplier.status === 1
                                    ? t('Enabled')
                                    : t('Disabled')}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <Button
                                  variant='outline'
                                  size='sm'
                                  onClick={() => openSupplierDialog(supplier)}
                                >
                                  <Pencil data-icon='inline-start' />
                                  {t('Edit')}
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>

                  <DataPanel
                    title={t('Supplier Agreements')}
                    description={t(
                      'Cache-aware cost ratios used by the ledger writer.'
                    )}
                    action={
                      <Button
                        variant='outline'
                        size='sm'
                        onClick={() => openAgreementDialog()}
                      >
                        <Plus data-icon='inline-start' />
                        {t('Add Agreement')}
                      </Button>
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('Supplier')}</TableHead>
                          <TableHead>{t('Model')}</TableHead>
                          <TableHead>{t('Prompt Ratio')}</TableHead>
                          <TableHead>{t('Cache Ratio')}</TableHead>
                          <TableHead>{t('Priority')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Actions')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={agreementsQuery.isLoading}
                          isEmpty={agreements.length === 0}
                          columns={7}
                          emptyMessage={t('No supplier agreements configured.')}
                        >
                          {agreements.map((agreement) => (
                            <TableRow key={agreement.id}>
                              <TableCell>
                                {supplierNameById(
                                  suppliers,
                                  agreement.supplier_id
                                )}
                              </TableCell>
                              <TableCell>
                                {agreement.model_name || t('All Models')}
                              </TableCell>
                              <TableCell>
                                {formatNumber(agreement.cost_model_ratio)}
                              </TableCell>
                              <TableCell>
                                {formatNumber(agreement.cost_cache_ratio)}
                              </TableCell>
                              <TableCell>{agreement.priority}</TableCell>
                              <TableCell>
                                <Badge
                                  variant={
                                    agreement.status === 1
                                      ? 'secondary'
                                      : 'outline'
                                  }
                                >
                                  {agreement.status === 1
                                    ? t('Enabled')
                                    : t('Disabled')}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <div className='flex flex-wrap gap-2'>
                                  <Button
                                    variant='outline'
                                    size='sm'
                                    onClick={() =>
                                      openAgreementDialog(agreement)
                                    }
                                  >
                                    <Pencil data-icon='inline-start' />
                                    {t('Edit')}
                                  </Button>
                                  <Button
                                    variant='outline'
                                    size='sm'
                                    disabled={removeAgreement.isPending}
                                    onClick={() =>
                                      setAgreementToDelete(agreement)
                                    }
                                  >
                                    <Trash2 data-icon='inline-start' />
                                    {t('Delete')}
                                  </Button>
                                </div>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>

              <TabsContent value='ledger' className='min-h-0 overflow-auto'>
                <DataPanel
                  title={t('Usage Ledger')}
                  description={t(
                    'One idempotent row per accepted API request, including session and cache split.'
                  )}
                >
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Request')}</TableHead>
                        <TableHead>{t('Session')}</TableHead>
                        <TableHead>{t('Supplier')}</TableHead>
                        <TableHead>{t('Channel')}</TableHead>
                        <TableHead>{t('Model')}</TableHead>
                        <TableHead>{t('Tokens')}</TableHead>
                        <TableHead>{t('Sell Quota')}</TableHead>
                        <TableHead>{t('Cost Quota')}</TableHead>
                        <TableHead>{t('Gross Profit')}</TableHead>
                        <TableHead>{t('Created At')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      <TableRowsState
                        isLoading={ledgersQuery.isLoading}
                        isEmpty={ledgers.length === 0}
                        columns={10}
                        emptyMessage={t('No usage ledger rows in this period.')}
                      >
                        {ledgers.map((ledger) => (
                          <TableRow key={ledger.id}>
                            <TableCell className='max-w-48 truncate'>
                              {ledger.request_id}
                            </TableCell>
                            <TableCell className='max-w-40 truncate'>
                              {ledger.session_id || '-'}
                            </TableCell>
                            <TableCell>
                              {supplierNameById(suppliers, ledger.supplier_id)}
                            </TableCell>
                            <TableCell>#{ledger.channel_id}</TableCell>
                            <TableCell>{ledger.model_name || '-'}</TableCell>
                            <TableCell>
                              <span className='flex flex-col gap-0.5'>
                                <span>
                                  {t('Prompt')}:{' '}
                                  {formatTokens(ledger.prompt_tokens)}
                                </span>
                                <span className='text-muted-foreground'>
                                  {t('Cached')}:{' '}
                                  {formatTokens(ledger.cached_tokens)}
                                </span>
                              </span>
                            </TableCell>
                            <TableCell>
                              {formatLogQuota(ledger.sell_quota)}
                            </TableCell>
                            <TableCell>
                              {formatLogQuota(ledger.cost_quota)}
                            </TableCell>
                            <TableCell>
                              {formatSignedQuota(
                                ledger.sell_quota - ledger.cost_quota
                              )}
                            </TableCell>
                            <TableCell>
                              {formatTimestampToDate(ledger.created_at)}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableRowsState>
                    </TableBody>
                  </Table>
                </DataPanel>
              </TabsContent>

              <TabsContent
                value='settlements'
                className='min-h-0 overflow-auto'
              >
                <div className='grid grid-cols-1 gap-4 xl:grid-cols-[360px_1fr]'>
                  <DataPanel
                    title={t('Generate Statement')}
                    description={t(
                      'Creates a draft data statement for offline finance review.'
                    )}
                  >
                    <FieldGroup>
                      <Field>
                        <FieldTitle>{t('Subject Type')}</FieldTitle>
                        <ToggleGroup
                          value={[subjectType]}
                          onValueChange={(value) => {
                            const next = value.find(
                              (item) => item !== subjectType
                            )
                            if (next) {
                              setSubjectType(next as SettlementSubjectType)
                            }
                          }}
                          aria-label={t('Subject Type')}
                          variant='outline'
                          size='sm'
                          spacing={2}
                        >
                          <ToggleGroupItem value='supplier'>
                            {t('Supplier')}
                          </ToggleGroupItem>
                          <ToggleGroupItem value='user'>
                            {t('User')}
                          </ToggleGroupItem>
                        </ToggleGroup>
                      </Field>
                      <Field>
                        <FieldLabel htmlFor='token-router-subject-id'>
                          {subjectType === 'supplier'
                            ? t('Supplier ID')
                            : t('User ID')}
                        </FieldLabel>
                        <Input
                          id='token-router-subject-id'
                          type='number'
                          min={1}
                          value={subjectId}
                          onChange={(event) => setSubjectId(event.target.value)}
                        />
                        <FieldDescription>
                          {t(
                            'Statements are data exports only; payments stay offline.'
                          )}
                        </FieldDescription>
                      </Field>
                      <Button
                        onClick={() => generateStatement.mutate()}
                        disabled={generateStatement.isPending}
                      >
                        {generateStatement.isPending
                          ? t('Generating...')
                          : t('Generate Statement')}
                      </Button>
                    </FieldGroup>
                  </DataPanel>

                  <DataPanel
                    title={t('Settlement Statements')}
                    description={t(
                      'Draft statements can be exported as item-level CSV.'
                    )}
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>{t('ID')}</TableHead>
                          <TableHead>{t('Subject')}</TableHead>
                          <TableHead>{t('Period')}</TableHead>
                          <TableHead>{t('Requests')}</TableHead>
                          <TableHead>{t('Gross Profit')}</TableHead>
                          <TableHead>{t('Cache Hit Rate')}</TableHead>
                          <TableHead>{t('Status')}</TableHead>
                          <TableHead>{t('Export')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        <TableRowsState
                          isLoading={statementsQuery.isLoading}
                          isEmpty={statements.length === 0}
                          columns={8}
                          emptyMessage={t(
                            'No settlement statements in this period.'
                          )}
                        >
                          {statements.map((statement) => (
                            <TableRow key={statement.id}>
                              <TableCell>#{statement.id}</TableCell>
                              <TableCell>
                                {statement.subject_type === 'supplier'
                                  ? supplierNameById(
                                      suppliers,
                                      statement.supplier_id
                                    )
                                  : `${t('User')} #${statement.user_id}`}
                              </TableCell>
                              <TableCell>
                                {formatTimestampToDate(statement.period_start)}{' '}
                                - {formatTimestampToDate(statement.period_end)}
                              </TableCell>
                              <TableCell>
                                {formatNumber(statement.total_requests)}
                              </TableCell>
                              <TableCell>
                                {formatSignedQuota(
                                  statement.gross_profit_quota
                                )}
                              </TableCell>
                              <TableCell>
                                {formatRate(statement.cache_hit_rate)}
                              </TableCell>
                              <TableCell>
                                <Badge variant='outline'>
                                  {statement.status === 'draft'
                                    ? t('Draft')
                                    : statement.status}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <Button
                                  variant='outline'
                                  size='sm'
                                  onClick={() => {
                                    window.open(
                                      getSettlementItemsCsvUrl(statement.id),
                                      '_blank',
                                      'noopener,noreferrer'
                                    )
                                  }}
                                >
                                  <Download data-icon='inline-start' />
                                  CSV
                                </Button>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableRowsState>
                      </TableBody>
                    </Table>
                  </DataPanel>
                </div>
              </TabsContent>
            </Tabs>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </>
  )
}
