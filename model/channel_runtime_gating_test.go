package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

const runtimeGatingModel = "gpt-runtime-gating"
const runtimeGatingGroup = "default"

func resetRuntimeGatingTables(t *testing.T) {
	t.Helper()
	truncateTables(t)
	tables := []string{
		"sla_probe_runs",
		"sla_probe_plans",
		"sla_contracts",
		"operating_insights",
		"supply_routing_policies",
		"supply_action_executions",
		"supply_action_plans",
		"supplier_evaluations",
		"supplier_posture_recommendations",
		"supplier_route_preferences",
		"supplier_scorecards",
		"abilities",
		"channels",
		"suppliers",
	}
	for _, table := range tables {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
}

func withRuntimeMemoryCache(t *testing.T, enabled bool) {
	t.Helper()
	previous := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = enabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previous
		channelSyncLock.Lock()
		group2model2channels = nil
		channelsIDM = nil
		supplierStatusIDM = nil
		supplierRoutePreferencePercentIDM = nil
		channel2advancedCustomConfig = nil
		channelSyncLock.Unlock()
	})
}

func seedRuntimeGatingSupplierChannel(t *testing.T, channelID int, supplierType string, supplierStatus int, priority int64) (*Supplier, *Channel) {
	t.Helper()
	supplier := &Supplier{
		Name:   fmt.Sprintf("runtime-supplier-%d", channelID),
		Type:   supplierType,
		Status: supplierStatus,
	}
	require.NoError(t, supplier.Insert())

	weight := uint(100)
	channel := &Channel{
		Id:         channelID,
		Type:       constant.ChannelTypeOpenAI,
		Key:        fmt.Sprintf("sk-runtime-%d", channelID),
		Status:     common.ChannelStatusEnabled,
		Name:       fmt.Sprintf("runtime-channel-%d", channelID),
		SupplierId: supplier.Id,
		Models:     runtimeGatingModel,
		Group:      runtimeGatingGroup,
		Priority:   &priority,
		Weight:     &weight,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     runtimeGatingGroup,
		Model:     runtimeGatingModel,
		ChannelId: channel.Id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
	return supplier, channel
}

func seedRuntimeGatingSupplierRoutePreference(t *testing.T, supplierID int, weightPercent int) *SupplierRoutePreference {
	t.Helper()
	preference := &SupplierRoutePreference{
		SupplierId:    supplierID,
		Status:        SupplierRoutePreferenceStatusActive,
		WeightPercent: weightPercent,
		Reason:        "runtime gating test route preference",
		EffectiveFrom: 100,
		ActivatedAt:   100,
		ActivatedBy:   1,
		CreatedAt:     100,
		UpdatedAt:     100,
	}
	require.NoError(t, DB.Create(preference).Error)
	return preference
}

func TestGetRandomSatisfiedChannelSkipsDisabledSupplierMemoryCache(t *testing.T) {
	resetRuntimeGatingTables(t)
	_, disabledChannel := seedRuntimeGatingSupplierChannel(t, 101, SupplierTypeThirdParty, common.ChannelStatusManuallyDisabled, 100)
	_, enabledChannel := seedRuntimeGatingSupplierChannel(t, 102, SupplierTypeThirdParty, common.ChannelStatusEnabled, 10)
	withRuntimeMemoryCache(t, true)
	InitChannelCache()

	channel, err := GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, enabledChannel.Id, channel.Id)
	require.False(t, IsChannelEnabledForGroupModel(runtimeGatingGroup, runtimeGatingModel, disabledChannel.Id))
	require.True(t, IsChannelEnabledForGroupModel(runtimeGatingGroup, runtimeGatingModel, enabledChannel.Id))
}

func TestGetRandomSatisfiedChannelSkipsDisabledSupplierDBFallback(t *testing.T) {
	resetRuntimeGatingTables(t)
	_, disabledChannel := seedRuntimeGatingSupplierChannel(t, 111, SupplierTypeThirdParty, common.ChannelStatusManuallyDisabled, 100)
	_, enabledChannel := seedRuntimeGatingSupplierChannel(t, 112, SupplierTypeThirdParty, common.ChannelStatusEnabled, 10)
	withRuntimeMemoryCache(t, false)

	channel, err := GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, enabledChannel.Id, channel.Id)
	require.False(t, IsChannelEnabledForGroupModel(runtimeGatingGroup, runtimeGatingModel, disabledChannel.Id))
	require.True(t, IsChannelEnabledForGroupModel(runtimeGatingGroup, runtimeGatingModel, enabledChannel.Id))
}

func TestGetRandomSatisfiedChannelAppliesSupplierRoutePreferenceMemoryCache(t *testing.T) {
	resetRuntimeGatingTables(t)
	downgradedSupplier, downgradedChannel := seedRuntimeGatingSupplierChannel(t, 113, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	_, normalChannel := seedRuntimeGatingSupplierChannel(t, 114, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	seedRuntimeGatingSupplierRoutePreference(t, downgradedSupplier.Id, 0)
	loadedPreferences, err := loadActiveSupplierRoutePreferencePercents([]int{downgradedSupplier.Id}, common.GetTimestamp())
	require.NoError(t, err)
	require.Contains(t, loadedPreferences, downgradedSupplier.Id)
	require.Equal(t, 0, loadedPreferences[downgradedSupplier.Id])
	withRuntimeMemoryCache(t, true)
	InitChannelCache()
	require.Contains(t, supplierRoutePreferencePercentIDM, downgradedSupplier.Id)
	require.Equal(t, 0, supplierRoutePreferencePercentIDM[downgradedSupplier.Id])

	channel, err := GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, normalChannel.Id, channel.Id)
	require.True(t, IsChannelEnabledForGroupModel(runtimeGatingGroup, runtimeGatingModel, downgradedChannel.Id))
}

func TestGetRandomSatisfiedChannelAppliesSupplierRoutePreferenceBoostMemoryCache(t *testing.T) {
	resetRuntimeGatingTables(t)
	boostedSupplier, boostedChannel := seedRuntimeGatingSupplierChannel(t, 117, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	_, normalChannel := seedRuntimeGatingSupplierChannel(t, 118, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	seedRuntimeGatingSupplierRoutePreference(t, boostedSupplier.Id, SupplierRoutePreferenceMaxWeightPercent)
	withRuntimeMemoryCache(t, true)
	InitChannelCache()
	require.Contains(t, supplierRoutePreferencePercentIDM, boostedSupplier.Id)
	require.Equal(t, SupplierRoutePreferenceMaxWeightPercent, supplierRoutePreferencePercentIDM[boostedSupplier.Id])

	const attempts = 600
	boostedSelections := 0
	normalSelections := 0
	for i := 0; i < attempts; i++ {
		channel, err := GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
		require.NoError(t, err)
		require.NotNil(t, channel)
		switch channel.Id {
		case boostedChannel.Id:
			boostedSelections++
		case normalChannel.Id:
			normalSelections++
		default:
			t.Fatalf("unexpected channel selected: %d", channel.Id)
		}
	}
	require.Greater(t, boostedSelections, normalSelections)
}

func TestGetRandomSatisfiedChannelAppliesSupplierRoutePreferenceDBFallback(t *testing.T) {
	resetRuntimeGatingTables(t)
	downgradedSupplier, downgradedChannel := seedRuntimeGatingSupplierChannel(t, 115, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	_, normalChannel := seedRuntimeGatingSupplierChannel(t, 116, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	seedRuntimeGatingSupplierRoutePreference(t, downgradedSupplier.Id, 0)
	loadedPreferences, err := loadActiveSupplierRoutePreferencePercents([]int{downgradedSupplier.Id}, common.GetTimestamp())
	require.NoError(t, err)
	require.Contains(t, loadedPreferences, downgradedSupplier.Id)
	require.Equal(t, 0, loadedPreferences[downgradedSupplier.Id])
	withRuntimeMemoryCache(t, false)

	channel, err := GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, normalChannel.Id, channel.Id)
	require.True(t, IsChannelEnabledForGroupModel(runtimeGatingGroup, runtimeGatingModel, downgradedChannel.Id))
}

func TestGetRandomSatisfiedChannelAppliesSupplierRoutePreferenceBoostDBFallback(t *testing.T) {
	resetRuntimeGatingTables(t)
	boostedSupplier, boostedChannel := seedRuntimeGatingSupplierChannel(t, 119, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	_, normalChannel := seedRuntimeGatingSupplierChannel(t, 120, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	seedRuntimeGatingSupplierRoutePreference(t, boostedSupplier.Id, SupplierRoutePreferenceMaxWeightPercent)
	withRuntimeMemoryCache(t, false)

	const attempts = 600
	boostedSelections := 0
	normalSelections := 0
	for i := 0; i < attempts; i++ {
		channel, err := GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
		require.NoError(t, err)
		require.NotNil(t, channel)
		switch channel.Id {
		case boostedChannel.Id:
			boostedSelections++
		case normalChannel.Id:
			normalSelections++
		default:
			t.Fatalf("unexpected channel selected: %d", channel.Id)
		}
	}
	require.Greater(t, boostedSelections, normalSelections)
}

func TestApplySupplierEvaluationRefreshesRuntimeChannelCache(t *testing.T) {
	resetRuntimeGatingTables(t)
	supplier, channel := seedRuntimeGatingSupplierChannel(t, 121, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	withRuntimeMemoryCache(t, true)
	InitChannelCache()

	selected, err := GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, channel.Id, selected.Id)

	evaluation := &SupplierEvaluation{
		EvaluationKey:       "runtime-gating-reject",
		EvaluationType:      SupplierEvaluationTypeAdmission,
		SupplierId:          supplier.Id,
		SupplierScorecardId: 1,
		PeriodStart:         100,
		PeriodEnd:           200,
		Status:              SupplierEvaluationStatusApproved,
		Recommendation:      SupplierEvaluationRecommendationReject,
		Score:               50,
		Grade:               SupplierScorecardGradeD,
	}
	require.NoError(t, DB.Create(evaluation).Error)

	applied, err := ApplySupplierEvaluation(evaluation.Id, 7, "disable supplier after review")
	require.NoError(t, err)
	require.Equal(t, common.ChannelStatusEnabled, applied.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusManuallyDisabled, applied.SupplierStatusAfter)

	selected, err = GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
	require.NoError(t, err)
	require.Nil(t, selected)
	require.False(t, IsChannelEnabledForGroupModel(runtimeGatingGroup, runtimeGatingModel, channel.Id))
}

func TestApplySupplierPostureRecommendationRefreshesRuntimeChannelCache(t *testing.T) {
	resetRuntimeGatingTables(t)
	supplier, channel := seedRuntimeGatingSupplierChannel(t, 122, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	withRuntimeMemoryCache(t, true)
	InitChannelCache()

	selected, err := GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, channel.Id, selected.Id)

	recommendation := &SupplierPostureRecommendation{
		RecommendationKey:   "runtime-gating-posture-disable",
		SupplierId:          supplier.Id,
		SupplierScorecardId: 1,
		PeriodStart:         100,
		PeriodEnd:           200,
		Status:              SupplierPostureRecommendationStatusApproved,
		RecommendedAction:   SupplierPostureRecommendationActionDisable,
		Score:               35,
		Grade:               SupplierScorecardGradeD,
	}
	require.NoError(t, DB.Create(recommendation).Error)

	applied, err := ApplySupplierPostureRecommendation(recommendation.Id, 7, "disable supplier after posture review")
	require.NoError(t, err)
	require.Equal(t, common.ChannelStatusEnabled, applied.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusManuallyDisabled, applied.SupplierStatusAfter)

	selected, err = GetRandomSatisfiedChannel(runtimeGatingGroup, runtimeGatingModel, 0, "")
	require.NoError(t, err)
	require.Nil(t, selected)
	require.False(t, IsChannelEnabledForGroupModel(runtimeGatingGroup, runtimeGatingModel, channel.Id))
}

func TestManualSupplierRoutePreferenceRefreshesRuntimeChannelCache(t *testing.T) {
	resetRuntimeGatingTables(t)
	supplier, _ := seedRuntimeGatingSupplierChannel(t, 123, SupplierTypeThirdParty, common.ChannelStatusEnabled, 100)
	withRuntimeMemoryCache(t, true)
	InitChannelCache()
	require.NotContains(t, supplierRoutePreferencePercentIDM, supplier.Id)

	active, err := ActivateSupplierRoutePreference(SupplierRoutePreferenceActivateInput{
		SupplierId:    supplier.Id,
		WeightPercent: 150,
		Reason:        "manual route preference cache refresh",
		EffectiveFrom: common.GetTimestamp() - 1,
		OperatorNote:  "operator cache check",
	}, 7)
	require.NoError(t, err)
	require.Equal(t, SupplierRoutePreferenceStatusActive, active.Status)
	require.Equal(t, 150, active.WeightPercent)
	require.Contains(t, supplierRoutePreferencePercentIDM, supplier.Id)
	require.Equal(t, 150, supplierRoutePreferencePercentIDM[supplier.Id])

	disabled, err := DisableSupplierRoutePreference(supplier.Id, 8, "restore baseline")
	require.NoError(t, err)
	require.Equal(t, SupplierRoutePreferenceStatusDisabled, disabled.Status)
	require.NotContains(t, supplierRoutePreferencePercentIDM, supplier.Id)
}

func TestSupplyRoutingPolicyRequiresEnabledSupplier(t *testing.T) {
	resetRuntimeGatingTables(t)
	supplier, channel := seedRuntimeGatingSupplierChannel(t, 131, SupplierTypeSelfHosted, common.ChannelStatusManuallyDisabled, 100)
	withRuntimeMemoryCache(t, false)

	policy := &SupplyRoutingPolicy{
		SupplyActionExecutionId: 1001,
		SupplyActionPlanId:      1002,
		SupplyDecisionId:        1003,
		DecisionKey:             "runtime-gating-policy",
		SliceKey:                "runtime-gating-slice",
		ModelName:               runtimeGatingModel,
		SlaTier:                 "default",
		PeriodStart:             100,
		PeriodEnd:               200,
		Track:                   SupplierTypeSelfHosted,
		ActionType:              SupplyActionTypeEvaluateSelfHostedCapacity,
		Status:                  SupplyRoutingPolicyStatusActive,
		SupplierId:              supplier.Id,
		ChannelId:               channel.Id,
		Priority:                100,
		ActivatedAt:             100,
		CreatedAt:               100,
		UpdatedAt:               100,
	}
	require.NoError(t, DB.Create(policy).Error)

	matched, err := FindActiveSupplyRoutingPolicyForRequest(SupplyRoutingPolicyMatchInput{
		Group:     runtimeGatingGroup,
		ModelName: runtimeGatingModel,
		SlaTier:   "default",
		Now:       150,
	})
	require.NoError(t, err)
	require.Nil(t, matched)

	execution := &SupplyActionExecution{
		SupplyActionPlanId: 1004,
		SupplyDecisionId:   1005,
		DecisionKey:        "runtime-gating-execution",
		SliceKey:           "runtime-gating-slice",
		ModelName:          runtimeGatingModel,
		SlaTier:            "default",
		PeriodStart:        100,
		PeriodEnd:          200,
		DecisionType:       SupplyDecisionTypeSelfHostedEvaluate,
		Track:              SupplierTypeSelfHosted,
		ActionType:         SupplyActionTypeEvaluateSelfHostedCapacity,
		ExecutionStatus:    SupplyActionExecutionStatusRecorded,
		SupplierId:         supplier.Id,
		ChannelId:          channel.Id,
		RecordedAt:         150,
		CreatedAt:          150,
		UpdatedAt:          150,
	}
	require.NoError(t, DB.Create(execution).Error)

	_, err = ActivateSupplyRoutingPolicy(SupplyRoutingPolicyActivateInput{
		SupplyActionExecutionId: execution.Id,
		Priority:                100,
	}, 1)
	require.ErrorContains(t, err, "is not enabled")
}

func TestSupplyRoutingPolicyMissRecordsOperatingInsight(t *testing.T) {
	resetRuntimeGatingTables(t)
	supplier, channel := seedRuntimeGatingSupplierChannel(t, 136, SupplierTypeSelfHosted, common.ChannelStatusEnabled, 100)
	require.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusManuallyDisabled, "policy miss test"))
	withRuntimeMemoryCache(t, false)

	policy := &SupplyRoutingPolicy{
		SupplyActionExecutionId: 1011,
		SupplyActionPlanId:      1012,
		SupplyDecisionId:        1013,
		TrafficProfileId:        1014,
		DecisionKey:             "runtime-gating-policy-miss",
		SliceKey:                "runtime-gating-slice",
		ModelName:               runtimeGatingModel,
		SlaTier:                 "default",
		UserId:                  7,
		PeriodStart:             3600,
		PeriodEnd:               7200,
		Track:                   SupplierTypeSelfHosted,
		ActionType:              SupplyActionTypeEvaluateSelfHostedCapacity,
		Status:                  SupplyRoutingPolicyStatusActive,
		SupplierId:              supplier.Id,
		ChannelId:               channel.Id,
		SupplyCapacityId:        1015,
		SlaContractId:           1016,
		SlaProbeRunId:           1017,
		SlaProbeRunKey:          "runtime-policy-miss-run",
		SlaArtifactSHA256:       "runtime-policy-miss-sha",
		SlaRuntimeRef:           "runtime/policy-miss",
		Priority:                100,
		ActivatedAt:             3600,
		CreatedAt:               3600,
		UpdatedAt:               3600,
	}
	require.NoError(t, DB.Create(policy).Error)

	matched, miss, err := ResolveActiveSupplyRoutingPolicyForRequest(SupplyRoutingPolicyMatchInput{
		Group:     runtimeGatingGroup,
		ModelName: runtimeGatingModel,
		SlaTier:   "default",
		UserId:    7,
		Now:       3700,
	})
	require.NoError(t, err)
	require.Nil(t, matched)
	require.NotNil(t, miss)
	require.Equal(t, policy.Id, miss.Policy.Id)
	require.Equal(t, runtimeGatingGroup, miss.Group)
	require.Equal(t, SupplyRoutingPolicyMissReasonChannelDisabled, miss.Reason)

	insight, err := RecordSupplyRoutingPolicyMissInsight(miss, 3700)
	require.NoError(t, err)
	require.NotNil(t, insight)
	require.Equal(t, OperatingInsightCategoryQualityWatch, insight.Category)
	require.Equal(t, OperatingInsightSeverityWatch, insight.Severity)
	require.Equal(t, OperatingInsightStatusDraft, insight.Status)
	require.Equal(t, policy.TrafficProfileId, insight.TrafficProfileId)
	require.Equal(t, policy.SupplyDecisionId, insight.SupplyDecisionId)
	require.Equal(t, policy.ModelName, insight.ModelName)
	require.Equal(t, policy.SlaProbeRunId, insight.SlaProbeRunId)
	require.Equal(t, SlaProbeRunStatusPassed, insight.SlaProbeStatus)
	require.True(t, insight.SlaHardGatePassed)
	require.Contains(t, insight.InsightKey, "supply_routing_policy_miss")
	require.Contains(t, insight.Summary, "policy channel is disabled")

	regenerated, err := RecordSupplyRoutingPolicyMissInsight(miss, 3710)
	require.NoError(t, err)
	require.Equal(t, insight.Id, regenerated.Id)
	var count int64
	require.NoError(t, DB.Model(&OperatingInsight{}).Where("insight_key = ?", insight.InsightKey).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestSupplyRoutingPolicyTrafficPercentCanary(t *testing.T) {
	resetRuntimeGatingTables(t)
	supplier, channel := seedRuntimeGatingSupplierChannel(t, 137, SupplierTypeSelfHosted, common.ChannelStatusEnabled, 100)
	withRuntimeMemoryCache(t, false)

	policy := &SupplyRoutingPolicy{
		SupplyActionExecutionId: 1021,
		SupplyActionPlanId:      1022,
		SupplyDecisionId:        1023,
		DecisionKey:             "runtime-gating-policy-canary",
		SliceKey:                "runtime-gating-slice",
		ModelName:               runtimeGatingModel,
		SlaTier:                 "default",
		UserId:                  7,
		PeriodStart:             3600,
		PeriodEnd:               7200,
		Track:                   SupplierTypeSelfHosted,
		ActionType:              SupplyActionTypeEvaluateSelfHostedCapacity,
		Status:                  SupplyRoutingPolicyStatusActive,
		SupplierId:              supplier.Id,
		ChannelId:               channel.Id,
		Priority:                100,
		TrafficPercent:          50,
		ActivatedAt:             3600,
		CreatedAt:               3600,
		UpdatedAt:               3600,
	}
	require.NoError(t, DB.Create(policy).Error)

	var includedRouteKey string
	var excludedRouteKey string
	for i := 0; i < 1000 && (includedRouteKey == "" || excludedRouteKey == ""); i++ {
		routeKey := fmt.Sprintf("session:canary-%d", i)
		if supplyRoutingPolicyIncludesRouteKey(policy, routeKey) {
			includedRouteKey = routeKey
		} else {
			excludedRouteKey = routeKey
		}
	}
	require.NotEmpty(t, includedRouteKey)
	require.NotEmpty(t, excludedRouteKey)

	matched, miss, err := ResolveActiveSupplyRoutingPolicyForRequest(SupplyRoutingPolicyMatchInput{
		Group:     runtimeGatingGroup,
		ModelName: runtimeGatingModel,
		SlaTier:   "default",
		UserId:    7,
		RouteKey:  includedRouteKey,
		Now:       3700,
	})
	require.NoError(t, err)
	require.Nil(t, miss)
	require.NotNil(t, matched)
	require.Equal(t, policy.Id, matched.Id)

	matched, miss, err = ResolveActiveSupplyRoutingPolicyForRequest(SupplyRoutingPolicyMatchInput{
		Group:     runtimeGatingGroup,
		ModelName: runtimeGatingModel,
		SlaTier:   "default",
		UserId:    7,
		RouteKey:  excludedRouteKey,
		Now:       3700,
	})
	require.NoError(t, err)
	require.Nil(t, matched)
	require.Nil(t, miss)

	require.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusManuallyDisabled, "canary miss test"))
	matched, miss, err = ResolveActiveSupplyRoutingPolicyForRequest(SupplyRoutingPolicyMatchInput{
		Group:     runtimeGatingGroup,
		ModelName: runtimeGatingModel,
		SlaTier:   "default",
		UserId:    7,
		RouteKey:  excludedRouteKey,
		Now:       3700,
	})
	require.NoError(t, err)
	require.Nil(t, matched)
	require.Nil(t, miss)
	matched, miss, err = ResolveActiveSupplyRoutingPolicyForRequest(SupplyRoutingPolicyMatchInput{
		Group:     runtimeGatingGroup,
		ModelName: runtimeGatingModel,
		SlaTier:   "default",
		UserId:    7,
		RouteKey:  includedRouteKey,
		Now:       3700,
	})
	require.NoError(t, err)
	require.Nil(t, matched)
	require.NotNil(t, miss)
	require.Equal(t, SupplyRoutingPolicyMissReasonChannelDisabled, miss.Reason)
	require.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, "restore canary test"))

	require.NoError(t, DB.Model(policy).Update("traffic_percent", 0).Error)
	policy.TrafficPercent = 0
	matched, miss, err = ResolveActiveSupplyRoutingPolicyForRequest(SupplyRoutingPolicyMatchInput{
		Group:     runtimeGatingGroup,
		ModelName: runtimeGatingModel,
		SlaTier:   "default",
		UserId:    7,
		RouteKey:  excludedRouteKey,
		Now:       3700,
	})
	require.NoError(t, err)
	require.Nil(t, miss)
	require.NotNil(t, matched)
	require.Equal(t, policy.Id, matched.Id)
}

func TestSupplyRoutingPolicyRequiresPassedRuntimeSlaEvidence(t *testing.T) {
	resetRuntimeGatingTables(t)
	supplier, channel := seedRuntimeGatingSupplierChannel(t, 141, SupplierTypeSelfHosted, common.ChannelStatusEnabled, 100)
	withRuntimeMemoryCache(t, false)

	execution := &SupplyActionExecution{
		SupplyActionPlanId: 1104,
		SupplyDecisionId:   1105,
		DecisionKey:        "runtime-sla-gated-execution",
		SliceKey:           "runtime-sla-gated-slice",
		ModelName:          runtimeGatingModel,
		SlaTier:            "default",
		PeriodStart:        100,
		PeriodEnd:          200,
		DecisionType:       SupplyDecisionTypeSelfHostedEvaluate,
		Track:              SupplierTypeSelfHosted,
		ActionType:         SupplyActionTypeEvaluateSelfHostedCapacity,
		ExecutionStatus:    SupplyActionExecutionStatusRecorded,
		SupplierId:         supplier.Id,
		ChannelId:          channel.Id,
		RecordedAt:         150,
		CreatedAt:          150,
		UpdatedAt:          150,
	}
	require.NoError(t, DB.Create(execution).Error)

	_, err := ActivateSupplyRoutingPolicy(SupplyRoutingPolicyActivateInput{
		SupplyActionExecutionId: execution.Id,
		Priority:                100,
		TrafficPercent:          101,
	}, 1)
	require.ErrorContains(t, err, "traffic_percent must be between 1 and 100")

	_, err = ActivateSupplyRoutingPolicy(SupplyRoutingPolicyActivateInput{
		SupplyActionExecutionId: execution.Id,
		Priority:                100,
	}, 1)
	require.ErrorContains(t, err, "passed runtime SLA probe run is required")

	contract, err := ImportSlaContract(SlaContractImportInput{
		ContractKey:            "runtime-routing-sla-test",
		ModelName:              runtimeGatingModel,
		ProviderFamily:         "kimi",
		SourceName:             "runtime routing SLA test",
		SourceRef:              "test://runtime-routing-sla",
		SourceSHA256:           "runtime-routing-sla-sha",
		Version:                "2026-06-23",
		Status:                 SlaContractStatusActive,
		MeasurementProfileJSON: `{"input_profile":{"tokens":128},"output_profile":{"target_tokens":16},"cache_profile":"cold_no_cache"}`,
		HardGateJSON:           `{"ttft_ms":{"p90_lte":1000}}`,
		SoftGateJSON:           `{}`,
	}, 1)
	require.NoError(t, err)
	admissionPlan, err := GenerateSlaProbePlan(SlaProbePlanGenerateInput{
		ContractId:     contract.Id,
		SupplierId:     supplier.Id,
		ChannelId:      channel.Id,
		SlaTier:        "default",
		ProbeType:      SlaProbeTypeAdmission,
		RouteMode:      SlaProbeRouteModeDirectUpstream,
		PromptSuiteKey: "runtime-routing-admission",
		SampleSize:     1,
		RepeatCount:    1,
		MaxProbeQuota:  100,
	}, 1)
	require.NoError(t, err)
	_, err = RecordSlaProbeRun(SlaProbeRunRecordInput{
		RunKey:         "runtime-routing-admission-run",
		PlanId:         admissionPlan.Id,
		Status:         SlaProbeRunStatusPassed,
		StartedAt:      160,
		EndedAt:        170,
		RunnerVersion:  "token-router-sla-test",
		RuntimeRef:     "aima2/admission",
		SummaryJSON:    `{"ttft_ms":{"p90":500}}`,
		HardGatePassed: true,
		ArtifactSHA256: "admission-sha",
	}, 1)
	require.NoError(t, err)

	_, err = ActivateSupplyRoutingPolicy(SupplyRoutingPolicyActivateInput{
		SupplyActionExecutionId: execution.Id,
		Priority:                100,
	}, 1)
	require.ErrorContains(t, err, "passed runtime SLA probe run is required")

	runtimePlan, err := GenerateSlaProbePlan(SlaProbePlanGenerateInput{
		ContractId:     contract.Id,
		SupplierId:     supplier.Id,
		ChannelId:      channel.Id,
		SlaTier:        "default",
		ProbeType:      SlaProbeTypeRuntimeLight,
		RouteMode:      SlaProbeRouteModeDirectUpstream,
		PromptSuiteKey: "runtime-routing-light",
		SampleSize:     1,
		RepeatCount:    1,
		MaxProbeQuota:  100,
	}, 1)
	require.NoError(t, err)
	_, err = RecordSlaProbeRun(SlaProbeRunRecordInput{
		RunKey:         "runtime-routing-failed-run",
		PlanId:         runtimePlan.Id,
		Status:         SlaProbeRunStatusFailed,
		StartedAt:      180,
		EndedAt:        190,
		RunnerVersion:  "token-router-sla-test",
		RuntimeRef:     "aima2/runtime-failed",
		SummaryJSON:    `{"ttft_ms":{"p90":9000}}`,
		HardGatePassed: false,
		FailureReasons: "ttft exceeded hard gate",
		ArtifactSHA256: "failed-sha",
	}, 1)
	require.NoError(t, err)

	_, err = ActivateSupplyRoutingPolicy(SupplyRoutingPolicyActivateInput{
		SupplyActionExecutionId: execution.Id,
		Priority:                100,
	}, 1)
	require.ErrorContains(t, err, "passed runtime SLA probe run is required")

	run, err := RecordSlaProbeRun(SlaProbeRunRecordInput{
		RunKey:         "runtime-routing-passed-run",
		PlanId:         runtimePlan.Id,
		Status:         SlaProbeRunStatusPassed,
		StartedAt:      200,
		EndedAt:        210,
		RunnerVersion:  "token-router-sla-test",
		RuntimeRef:     "aima2/runtime-passed",
		SummaryJSON:    `{"ttft_ms":{"p90":500}}`,
		HardGatePassed: true,
		ArtifactSHA256: "passed-sha",
	}, 1)
	require.NoError(t, err)

	policy, err := ActivateSupplyRoutingPolicy(SupplyRoutingPolicyActivateInput{
		SupplyActionExecutionId: execution.Id,
		Priority:                100,
		TrafficPercent:          25,
	}, 1)
	require.NoError(t, err)
	require.Equal(t, SupplyRoutingPolicyStatusActive, policy.Status)
	require.Equal(t, 25, policy.TrafficPercent)
	require.Equal(t, contract.Id, policy.SlaContractId)
	require.Equal(t, run.Id, policy.SlaProbeRunId)
	require.Equal(t, run.RunKey, policy.SlaProbeRunKey)
	require.Equal(t, "passed-sha", policy.SlaArtifactSHA256)
	require.Equal(t, "aima2/runtime-passed", policy.SlaRuntimeRef)
}
