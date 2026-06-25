package controller

import (
	"encoding/csv"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func GetMarginSummary(c *gin.Context) {
	rows, err := model.SearchMarginSummary(model.MarginSummaryFilters{
		GroupBy:    c.Query("group_by"),
		SupplierId: parseOptionalIntQuery(c, "supplier_id"),
		ChannelId:  parseOptionalIntQuery(c, "channel_id"),
		UserId:     parseOptionalIntQuery(c, "user_id"),
		TokenId:    parseOptionalIntQuery(c, "token_id"),
		ModelName:  c.Query("model_name"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}

func GetQualitySummary(c *gin.Context) {
	rows, err := model.SearchQualitySummary(model.QualitySummaryFilters{
		GroupBy:    c.Query("group_by"),
		SupplierId: parseOptionalIntQuery(c, "supplier_id"),
		ChannelId:  parseOptionalIntQuery(c, "channel_id"),
		UserId:     parseOptionalIntQuery(c, "user_id"),
		TokenId:    parseOptionalIntQuery(c, "token_id"),
		ModelName:  c.Query("model_name"),
		Status:     c.Query("status"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}

func GenerateSupplierScorecards(c *gin.Context) {
	var input model.SupplierScorecardGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	scorecards, err := model.GenerateSupplierScorecards(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, scorecards)
}

func GetSupplierScorecards(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	scorecards, total, err := model.SearchSupplierScorecards(model.SupplierScorecardFilters{
		SupplierId: parseOptionalIntQuery(c, "supplier_id"),
		Grade:      c.Query("grade"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(scorecards)
	common.ApiSuccess(c, pageInfo)
}

func GenerateSupplierEvaluations(c *gin.Context) {
	var input model.SupplierEvaluationGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	evaluations, err := model.GenerateSupplierEvaluations(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, evaluations)
}

func GetSupplierEvaluations(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	evaluations, total, err := model.SearchSupplierEvaluations(model.SupplierEvaluationFilters{
		SupplierId:     parseOptionalIntQuery(c, "supplier_id"),
		EvaluationType: c.Query("evaluation_type"),
		Status:         c.Query("status"),
		Recommendation: c.Query("recommendation"),
		Grade:          c.Query("grade"),
		StartTime:      parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:        parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(evaluations)
	common.ApiSuccess(c, pageInfo)
}

func ApproveSupplierEvaluation(c *gin.Context) {
	reviewSupplierEvaluation(c, model.SupplierEvaluationStatusApproved)
}

func RejectSupplierEvaluation(c *gin.Context) {
	reviewSupplierEvaluation(c, model.SupplierEvaluationStatusRejected)
}

func ApplySupplierEvaluation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.SupplierEvaluationApplyInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	evaluation, err := model.ApplySupplierEvaluation(id, c.GetInt("id"), input.OperatorNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, evaluation)
}

func GenerateSupplierPostureRecommendations(c *gin.Context) {
	var input model.SupplierPostureRecommendationGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	recommendations, err := model.GenerateSupplierPostureRecommendations(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, recommendations)
}

func GetSupplierPostureRecommendations(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	recommendations, total, err := model.SearchSupplierPostureRecommendations(model.SupplierPostureRecommendationFilters{
		SupplierId:        parseOptionalIntQuery(c, "supplier_id"),
		Status:            c.Query("status"),
		RecommendedAction: c.Query("recommended_action"),
		Grade:             c.Query("grade"),
		StartTime:         parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:           parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(recommendations)
	common.ApiSuccess(c, pageInfo)
}

func GetSupplierRoutePreferences(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	preferences, total, err := model.SearchSupplierRoutePreferences(model.SupplierRoutePreferenceFilters{
		SupplierId:                    parseOptionalIntQuery(c, "supplier_id"),
		SourcePostureRecommendationId: parseOptionalIntQuery(c, "source_posture_recommendation_id"),
		Status:                        c.Query("status"),
		StartTime:                     parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:                       parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(preferences)
	common.ApiSuccess(c, pageInfo)
}

func ActivateSupplierRoutePreference(c *gin.Context) {
	var input model.SupplierRoutePreferenceActivateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	preference, err := model.ActivateSupplierRoutePreference(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, preference)
}

func DisableSupplierRoutePreference(c *gin.Context) {
	supplierId, err := strconv.Atoi(c.Param("supplier_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.SupplierRoutePreferenceDisableInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	preference, err := model.DisableSupplierRoutePreference(supplierId, c.GetInt("id"), input.OperatorNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, preference)
}

func ApproveSupplierPostureRecommendation(c *gin.Context) {
	reviewSupplierPostureRecommendation(c, model.SupplierPostureRecommendationStatusApproved)
}

func RejectSupplierPostureRecommendation(c *gin.Context) {
	reviewSupplierPostureRecommendation(c, model.SupplierPostureRecommendationStatusRejected)
}

func ApplySupplierPostureRecommendation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.SupplierPostureRecommendationApplyInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	recommendation, err := model.ApplySupplierPostureRecommendation(id, c.GetInt("id"), input.OperatorNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, recommendation)
}

func ImportSlaContract(c *gin.Context) {
	var input model.SlaContractImportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	contract, err := model.ImportSlaContract(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, contract)
}

func GetSlaContracts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	contracts, total, err := model.SearchSlaContracts(model.SlaContractFilters{
		ModelName:      c.Query("model_name"),
		ProviderFamily: c.Query("provider_family"),
		Status:         c.Query("status"),
		StartTime:      parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:        parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(contracts)
	common.ApiSuccess(c, pageInfo)
}

func GetSlaContract(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	contract, err := model.GetSlaContractByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, contract)
}

func GenerateSlaProbePlan(c *gin.Context) {
	var input model.SlaProbePlanGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	plan, err := model.GenerateSlaProbePlan(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func GetSlaProbePlans(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	plans, total, err := model.SearchSlaProbePlans(model.SlaProbePlanFilters{
		ContractId: parseOptionalIntQuery(c, "contract_id"),
		SupplierId: parseOptionalIntQuery(c, "supplier_id"),
		ChannelId:  parseOptionalIntQuery(c, "channel_id"),
		ModelName:  c.Query("model_name"),
		SlaTier:    c.Query("sla_tier"),
		ProbeType:  c.Query("probe_type"),
		RouteMode:  c.Query("route_mode"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(plans)
	common.ApiSuccess(c, pageInfo)
}

func GetSlaProbePlan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	plan, err := model.GetSlaProbePlanByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func RecordSlaProbeRun(c *gin.Context) {
	var input model.SlaProbeRunRecordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	run, err := model.RecordSlaProbeRun(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, run)
}

func GetSlaProbeRuns(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	runs, total, err := model.SearchSlaProbeRuns(model.SlaProbeRunFilters{
		PlanId:     parseOptionalIntQuery(c, "plan_id"),
		ContractId: parseOptionalIntQuery(c, "contract_id"),
		SupplierId: parseOptionalIntQuery(c, "supplier_id"),
		ChannelId:  parseOptionalIntQuery(c, "channel_id"),
		ModelName:  c.Query("model_name"),
		SlaTier:    c.Query("sla_tier"),
		Status:     c.Query("status"),
		RouteMode:  c.Query("route_mode"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(runs)
	common.ApiSuccess(c, pageInfo)
}

func GetSlaProbeRun(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	run, err := model.GetSlaProbeRunByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, run)
}

func GenerateTrafficProfiles(c *gin.Context) {
	var input model.TrafficProfileGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	profiles, err := model.GenerateTrafficProfiles(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, profiles)
}

func GetTrafficProfiles(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	profiles, total, err := model.SearchTrafficProfiles(model.TrafficProfileFilters{
		ModelName: c.Query("model_name"),
		SlaTier:   c.Query("sla_tier"),
		UserId:    parseOptionalIntQuery(c, "user_id"),
		StartTime: parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:   parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(profiles)
	common.ApiSuccess(c, pageInfo)
}

func GenerateTrafficForecasts(c *gin.Context) {
	var input model.TrafficForecastGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	forecasts, err := model.GenerateTrafficForecasts(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, forecasts)
}

func GetTrafficForecasts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	forecasts, total, err := model.SearchTrafficForecasts(model.TrafficForecastFilters{
		ModelName:         c.Query("model_name"),
		SlaTier:           c.Query("sla_tier"),
		UserId:            parseOptionalIntQuery(c, "user_id"),
		Method:            c.Query("method"),
		SourcePeriodStart: parseOptionalInt64Query(c, "source_start_timestamp"),
		SourcePeriodEnd:   parseOptionalInt64Query(c, "source_end_timestamp"),
		TargetPeriodStart: parseOptionalInt64Query(c, "target_start_timestamp"),
		TargetPeriodEnd:   parseOptionalInt64Query(c, "target_end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(forecasts)
	common.ApiSuccess(c, pageInfo)
}

func GenerateSupplyDecisions(c *gin.Context) {
	var input model.SupplyDecisionGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	decisions, err := model.GenerateSupplyDecisions(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, decisions)
}

func GetSupplyDecisions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	decisions, total, err := model.SearchSupplyDecisions(model.SupplyDecisionFilters{
		ModelName:    c.Query("model_name"),
		SlaTier:      c.Query("sla_tier"),
		UserId:       parseOptionalIntQuery(c, "user_id"),
		Status:       c.Query("status"),
		Track:        c.Query("track"),
		DecisionType: c.Query("decision_type"),
		StartTime:    parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:      parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(decisions)
	common.ApiSuccess(c, pageInfo)
}

func ApproveSupplyDecision(c *gin.Context) {
	reviewSupplyDecision(c, model.SupplyDecisionStatusApproved)
}

func RejectSupplyDecision(c *gin.Context) {
	reviewSupplyDecision(c, model.SupplyDecisionStatusRejected)
}

func GenerateSupplyExpansionOpportunities(c *gin.Context) {
	var input model.SupplyExpansionOpportunityGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	opportunities, err := model.GenerateSupplyExpansionOpportunities(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, opportunities)
}

func GetSupplyExpansionOpportunities(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	opportunities, total, err := model.SearchSupplyExpansionOpportunities(model.SupplyExpansionOpportunityFilters{
		ModelName:       c.Query("model_name"),
		SlaTier:         c.Query("sla_tier"),
		UserId:          parseOptionalIntQuery(c, "user_id"),
		DecisionStatus:  c.Query("decision_status"),
		Track:           c.Query("track"),
		OpportunityType: c.Query("opportunity_type"),
		Priority:        c.Query("priority"),
		ClusterKey:      c.Query("cluster_key"),
		StartTime:       parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:         parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(opportunities)
	common.ApiSuccess(c, pageInfo)
}

func GeneratePricingRecommendations(c *gin.Context) {
	var input model.PricingRecommendationGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	recommendations, err := model.GeneratePricingRecommendations(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, recommendations)
}

func GetPricingRecommendations(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	recommendations, total, err := model.SearchPricingRecommendations(model.PricingRecommendationFilters{
		ModelName: c.Query("model_name"),
		SlaTier:   c.Query("sla_tier"),
		UserId:    parseOptionalIntQuery(c, "user_id"),
		Status:    c.Query("status"),
		Action:    c.Query("action"),
		StartTime: parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:   parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(recommendations)
	common.ApiSuccess(c, pageInfo)
}

func ApprovePricingRecommendation(c *gin.Context) {
	reviewPricingRecommendation(c, model.PricingRecommendationStatusApproved)
}

func RejectPricingRecommendation(c *gin.Context) {
	reviewPricingRecommendation(c, model.PricingRecommendationStatusRejected)
}

func GenerateOperatingInsights(c *gin.Context) {
	var input model.OperatingInsightGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	insights, err := model.GenerateOperatingInsights(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, insights)
}

func GetOperatingInsights(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	insights, total, err := model.SearchOperatingInsights(model.OperatingInsightFilters{
		ModelName: c.Query("model_name"),
		SlaTier:   c.Query("sla_tier"),
		UserId:    parseOptionalIntQuery(c, "user_id"),
		Status:    c.Query("status"),
		Severity:  c.Query("severity"),
		Category:  c.Query("category"),
		StartTime: parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:   parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(insights)
	common.ApiSuccess(c, pageInfo)
}

func AcknowledgeOperatingInsight(c *gin.Context) {
	reviewOperatingInsight(c, model.OperatingInsightStatusAcknowledged)
}

func DismissOperatingInsight(c *gin.Context) {
	reviewOperatingInsight(c, model.OperatingInsightStatusDismissed)
}

func GenerateSupplyActionPlans(c *gin.Context) {
	var input model.SupplyActionPlanGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	plans, err := model.GenerateSupplyActionPlans(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plans)
}

func GetSupplyActionPlans(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	plans, total, err := model.SearchSupplyActionPlans(model.SupplyActionPlanFilters{
		DecisionId: parseOptionalIntQuery(c, "decision_id"),
		Status:     c.Query("status"),
		Track:      c.Query("track"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(plans)
	common.ApiSuccess(c, pageInfo)
}

func UpdateSupplyActionPlanStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.SupplyActionPlanStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	plan, err := model.UpdateSupplyActionPlanStatus(id, input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func RecordSupplyActionExecution(c *gin.Context) {
	var input model.SupplyActionExecutionRecordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	execution, err := model.RecordSupplyActionExecution(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, execution)
}

func GetSupplyActionExecutions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	executions, total, err := model.SearchSupplyActionExecutions(model.SupplyActionExecutionFilters{
		ExecutionId:        parseOptionalIntQuery(c, "execution_id"),
		SupplyActionPlanId: parseOptionalIntQuery(c, "supply_action_plan_id"),
		SupplyDecisionId:   parseOptionalIntQuery(c, "supply_decision_id"),
		ExecutionStatus:    c.Query("execution_status"),
		Track:              c.Query("track"),
		SupplierId:         parseOptionalIntQuery(c, "supplier_id"),
		ChannelId:          parseOptionalIntQuery(c, "channel_id"),
		SupplyCapacityId:   parseOptionalIntQuery(c, "supply_capacity_id"),
		StartTime:          parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:            parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(executions)
	common.ApiSuccess(c, pageInfo)
}

func RefreshSupplyActionExecutionUsage(c *gin.Context) {
	var input model.SupplyActionExecutionUsageRefreshInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	executions, err := model.RefreshSupplyActionExecutionUsage(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, executions)
}

func ActivateSupplyRoutingPolicy(c *gin.Context) {
	var input model.SupplyRoutingPolicyActivateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	policy, err := model.ActivateSupplyRoutingPolicy(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, policy)
}

func DisableSupplyRoutingPolicy(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.SupplyRoutingPolicyDisableInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	policy, err := model.DisableSupplyRoutingPolicy(id, input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, policy)
}

func GetSupplyRoutingPolicies(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	policies, total, err := model.SearchSupplyRoutingPolicies(model.SupplyRoutingPolicyFilters{
		SupplyActionExecutionId: parseOptionalIntQuery(c, "supply_action_execution_id"),
		SupplyActionPlanId:      parseOptionalIntQuery(c, "supply_action_plan_id"),
		SupplyDecisionId:        parseOptionalIntQuery(c, "supply_decision_id"),
		Status:                  c.Query("status"),
		Track:                   c.Query("track"),
		SupplierId:              parseOptionalIntQuery(c, "supplier_id"),
		ChannelId:               parseOptionalIntQuery(c, "channel_id"),
		SupplyCapacityId:        parseOptionalIntQuery(c, "supply_capacity_id"),
		StartTime:               parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:                 parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(policies)
	common.ApiSuccess(c, pageInfo)
}

func reviewSupplyDecision(c *gin.Context, status string) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.SupplyDecisionReviewInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	decision, err := model.UpdateSupplyDecisionReview(id, status, c.GetInt("id"), input.ReviewNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, decision)
}

func reviewSupplierEvaluation(c *gin.Context, status string) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.SupplierEvaluationReviewInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	evaluation, err := model.UpdateSupplierEvaluationReview(id, status, c.GetInt("id"), input.ReviewNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, evaluation)
}

func reviewSupplierPostureRecommendation(c *gin.Context, status string) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.SupplierPostureRecommendationReviewInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	recommendation, err := model.UpdateSupplierPostureRecommendationReview(id, status, c.GetInt("id"), input.ReviewNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, recommendation)
}

func reviewPricingRecommendation(c *gin.Context, status string) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.PricingRecommendationReviewInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	recommendation, err := model.UpdatePricingRecommendationReview(id, status, c.GetInt("id"), input.ReviewNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, recommendation)
}

func reviewOperatingInsight(c *gin.Context, status string) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var input model.OperatingInsightReviewInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	insight, err := model.UpdateOperatingInsightReview(id, status, c.GetInt("id"), input.ReviewNote)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, insight)
}

func GenerateSettlementStatement(c *gin.Context) {
	var input model.SettlementStatementGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	statement, err := model.GenerateSettlementStatement(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, statement)
}

func GetSettlementStatements(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	statements, total, err := model.SearchSettlementStatements(model.SettlementStatementFilters{
		SubjectType: c.Query("subject_type"),
		SupplierId:  parseOptionalIntQuery(c, "supplier_id"),
		UserId:      parseOptionalIntQuery(c, "user_id"),
		Status:      c.Query("status"),
		StartTime:   parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:     parseOptionalInt64Query(c, "end_timestamp"),
	}, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(statements)
	common.ApiSuccess(c, pageInfo)
}

func GetSettlementStatement(c *gin.Context) {
	statement, ok := settlementStatementFromParam(c)
	if !ok {
		return
	}
	common.ApiSuccess(c, statement)
}

func GetSettlementStatementItems(c *gin.Context) {
	statement, ok := settlementStatementFromParam(c)
	if !ok {
		return
	}
	pageInfo := common.GetPageQuery(c)
	ledgers, total, err := model.SearchUsageLedgers(model.UsageLedgerFiltersForStatement(statement), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(ledgers)
	common.ApiSuccess(c, pageInfo)
}

func ExportSettlementStatementItemsCSV(c *gin.Context) {
	statement, ok := settlementStatementFromParam(c)
	if !ok {
		return
	}
	ledgers, _, err := model.SearchUsageLedgers(model.UsageLedgerFiltersForStatement(statement), 0, 100000)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filename := fmt.Sprintf("settlement-%d-items.csv", statement.Id)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()
	_ = writer.Write([]string{
		"request_id",
		"session_id",
		"supplier_id",
		"channel_id",
		"user_id",
		"token_id",
		"model_name",
		"prompt_tokens",
		"cached_tokens",
		"completion_tokens",
		"sell_quota",
		"cost_quota",
		"gross_profit_quota",
		"cache_hit",
		"created_at",
	})
	for _, ledger := range ledgers {
		grossProfit := ledger.SellQuota - ledger.CostQuota
		_ = writer.Write([]string{
			ledger.RequestId,
			ledger.SessionId,
			strconv.Itoa(ledger.SupplierId),
			strconv.Itoa(ledger.ChannelId),
			strconv.Itoa(ledger.UserId),
			strconv.Itoa(ledger.TokenId),
			ledger.ModelName,
			strconv.Itoa(ledger.PromptTokens),
			strconv.Itoa(ledger.CachedTokens),
			strconv.Itoa(ledger.CompletionTokens),
			strconv.Itoa(ledger.SellQuota),
			strconv.Itoa(ledger.CostQuota),
			strconv.Itoa(grossProfit),
			strconv.FormatBool(ledger.CacheHit),
			strconv.FormatInt(ledger.CreatedAt, 10),
		})
	}
}

func settlementStatementFromParam(c *gin.Context) (*model.SettlementStatement, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	statement, err := model.GetSettlementStatementByID(id)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	return statement, true
}
