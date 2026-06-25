package model

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestRefreshSupplyCapacityUsageFromLedger(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	supplier := &Supplier{Name: "gb10-4t-capacity-refresh", Type: SupplierTypeThirdParty, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())
	capacity := &SupplyCapacity{
		SupplierId:     supplier.Id,
		SupplyNode:     "gb10-4t-refresh",
		ModelName:      "gpt-capacity-refresh",
		PeriodStart:    now - 3600,
		PeriodEnd:      now + 3600,
		CapacityTokens: 1000,
		UsedTokens:     999,
		QualityScore:   98.5,
		UnitCostQuota:  0.5,
		Status:         1,
	}
	require.NoError(t, capacity.Insert())
	require.Equal(t, int64(1), capacity.HeadroomTokens)

	require.NoError(t, (&UsageLedger{
		RequestId:        "capacity-refresh-1",
		SessionId:        "session-capacity-refresh",
		SupplierId:       supplier.Id,
		ChannelId:        101,
		UserId:           2,
		TokenId:          1,
		ModelName:        "gpt-capacity-refresh",
		PromptTokens:     100,
		CompletionTokens: 40,
		Status:           "success",
		SupplyNode:       "gb10-4t-refresh",
		CreatedAt:        now,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "capacity-refresh-2",
		SessionId:        "session-capacity-refresh",
		SupplierId:       supplier.Id,
		ChannelId:        101,
		UserId:           2,
		TokenId:          1,
		ModelName:        "gpt-capacity-refresh",
		PromptTokens:     120,
		CompletionTokens: 60,
		Status:           "success",
		SupplyNode:       "gb10-4t-refresh",
		CreatedAt:        now + 1,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "capacity-refresh-failed",
		SessionId:        "session-capacity-refresh",
		SupplierId:       supplier.Id,
		ChannelId:        101,
		UserId:           2,
		TokenId:          1,
		ModelName:        "gpt-capacity-refresh",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "failed",
		SupplyNode:       "gb10-4t-refresh",
		CreatedAt:        now + 2,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "capacity-refresh-other-node",
		SessionId:        "session-capacity-refresh",
		SupplierId:       supplier.Id,
		ChannelId:        102,
		UserId:           2,
		TokenId:          1,
		ModelName:        "gpt-capacity-refresh",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SupplyNode:       "gb10-4t-other",
		CreatedAt:        now + 3,
	}).InsertIdempotent())

	refreshed, err := RefreshSupplyCapacityUsage(SupplyCapacityUsageRefreshInput{
		CapacityId: capacity.Id,
	})
	require.NoError(t, err)
	require.Len(t, refreshed, 1)
	require.Equal(t, int64(320), refreshed[0].UsedTokens)
	require.Equal(t, int64(680), refreshed[0].HeadroomTokens)
	require.InDelta(t, 0.32, refreshed[0].UtilizationRate, 0.000001)
	require.InDelta(t, 98.5, refreshed[0].QualityScore, 0.000001)
	require.InDelta(t, 0.5, refreshed[0].UnitCostQuota, 0.000001)

	saved, err := GetSupplyCapacityByID(capacity.Id)
	require.NoError(t, err)
	require.Equal(t, int64(320), saved.UsedTokens)
	require.Equal(t, int64(680), saved.HeadroomTokens)
}

func TestRecordSupplyCapacityTelemetryUpsertsSnapshot(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	supplier := &Supplier{Name: "gb10-4t-telemetry", Type: SupplierTypeSelfHosted, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())

	first, err := RecordSupplyCapacityTelemetry(SupplyCapacityTelemetryRecordInput{
		SupplierId:         supplier.Id,
		SupplyNode:         "gb10-4t-telemetry",
		ModelName:          "gpt-telemetry",
		PeriodStart:        now - 3600,
		PeriodEnd:          now + 3600,
		CapacityTokens:     2000,
		UsedTokens:         350,
		GpuUtilizationRate: 0.72,
		QualityScore:       99,
		UnitCostQuota:      0.34,
		SourceType:         SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          "gb10-4t-telemetry/run-1",
		ObservedAt:         now,
		Notes:              "node reported capacity telemetry",
	}, 7)
	require.NoError(t, err)
	require.Positive(t, first.Id)
	require.Positive(t, first.AppliedCapacityId)
	require.Equal(t, 7, first.RecordedBy)
	require.Equal(t, int64(1650), first.HeadroomTokens)
	require.InDelta(t, 0.175, first.UtilizationRate, 0.000001)
	require.InDelta(t, 0.72, first.GpuUtilizationRate, 0.000001)

	capacity, err := GetSupplyCapacityByID(first.AppliedCapacityId)
	require.NoError(t, err)
	require.Equal(t, supplier.Id, capacity.SupplierId)
	require.Equal(t, "gb10-4t-telemetry", capacity.SupplyNode)
	require.Equal(t, int64(2000), capacity.CapacityTokens)
	require.Equal(t, int64(350), capacity.UsedTokens)
	require.Equal(t, int64(1650), capacity.HeadroomTokens)
	require.InDelta(t, 0.175, capacity.UtilizationRate, 0.000001)
	require.InDelta(t, 0.72, capacity.GpuUtilizationRate, 0.000001)
	require.InDelta(t, 99, capacity.QualityScore, 0.000001)
	require.InDelta(t, 0.34, capacity.UnitCostQuota, 0.000001)
	require.Equal(t, SupplyCapacityTelemetrySourceNodeReport, capacity.TelemetrySourceType)
	require.Equal(t, "gb10-4t-telemetry/run-1", capacity.TelemetrySourceRef)
	require.Equal(t, now, capacity.TelemetryObservedAt)
	require.Equal(t, first.Id, capacity.LastTelemetryId)

	second, err := RecordSupplyCapacityTelemetry(SupplyCapacityTelemetryRecordInput{
		SupplierId:         supplier.Id,
		SupplyNode:         "gb10-4t-telemetry",
		ModelName:          "gpt-telemetry",
		PeriodStart:        now - 3600,
		PeriodEnd:          now + 3600,
		CapacityTokens:     2400,
		UsedTokens:         400,
		GpuUtilizationRate: 0.66,
		QualityScore:       98.5,
		UnitCostQuota:      0.31,
		SourceType:         SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          "gb10-4t-telemetry/run-1",
		ObservedAt:         now + 30,
		Notes:              "updated node telemetry",
	}, 9)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.Equal(t, first.AppliedCapacityId, second.AppliedCapacityId)
	require.Equal(t, 9, second.RecordedBy)
	require.Equal(t, int64(2000), second.HeadroomTokens)
	require.InDelta(t, 0.166666, second.UtilizationRate, 0.000001)

	capacity, err = GetSupplyCapacityByID(first.AppliedCapacityId)
	require.NoError(t, err)
	require.Equal(t, int64(2400), capacity.CapacityTokens)
	require.Equal(t, int64(400), capacity.UsedTokens)
	require.Equal(t, int64(2000), capacity.HeadroomTokens)
	require.InDelta(t, 0.66, capacity.GpuUtilizationRate, 0.000001)
	require.InDelta(t, 98.5, capacity.QualityScore, 0.000001)
	require.InDelta(t, 0.31, capacity.UnitCostQuota, 0.000001)
	require.Equal(t, second.Id, capacity.LastTelemetryId)

	telemetries, total, err := SearchSupplyCapacityTelemetries(SupplyCapacityTelemetryFilters{
		SupplierId: supplier.Id,
		ModelName:  "gpt-telemetry",
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, telemetries, 1)
	require.Equal(t, second.Id, telemetries[0].Id)
}

func TestCollectSupplyCapacityTelemetryFromChannel(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	var requested bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != SupplyCapacityTelemetryCollectPath {
			t.Errorf("unexpected telemetry path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-gb10-collector" {
			t.Errorf("unexpected telemetry authorization %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("supply_node") != "gb10-4t-collector" {
			t.Errorf("unexpected supply_node query %q", r.URL.Query().Get("supply_node"))
		}
		if r.URL.Query().Get("model") != "gpt-collector" {
			t.Errorf("unexpected model query %q", r.URL.Query().Get("model"))
		}
		if r.URL.Query().Get("period_start") != "1000" {
			t.Errorf("unexpected period_start query %q", r.URL.Query().Get("period_start"))
		}
		if r.URL.Query().Get("period_end") != "2000" {
			t.Errorf("unexpected period_end query %q", r.URL.Query().Get("period_end"))
		}
		requested = true
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"supply_node":          "gb10-4t-collector",
			"model_name":           "gpt-collector",
			"capacity_tokens":      1800,
			"used_tokens":          450,
			"gpu_utilization_rate": 0.61,
			"quality_score":        98.25,
			"unit_cost_quota":      0.42,
			"observed_at":          now,
			"source_ref":           "gb10-4t-collector/node-report",
			"notes":                "collector test telemetry",
		}); err != nil {
			t.Errorf("encode telemetry response: %v", err)
		}
	}))
	defer server.Close()

	supplier := &Supplier{Name: "gb10-4t-collector", Type: SupplierTypeSelfHosted, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())
	baseURL := server.URL
	channel := &Channel{
		Type:       1,
		Key:        "sk-gb10-collector",
		Status:     common.ChannelStatusEnabled,
		Name:       "gb10-4t-collector",
		SupplierId: supplier.Id,
		BaseURL:    &baseURL,
		Models:     "gpt-collector",
		Group:      "default",
	}
	require.NoError(t, DB.Create(channel).Error)

	telemetry, err := CollectSupplyCapacityTelemetry(SupplyCapacityTelemetryCollectInput{
		ChannelId:   channel.Id,
		SupplyNode:  "gb10-4t-collector",
		ModelName:   "gpt-collector",
		PeriodStart: 1000,
		PeriodEnd:   2000,
	}, 11)
	require.NoError(t, err)
	require.True(t, requested)
	require.Positive(t, telemetry.Id)
	require.Equal(t, supplier.Id, telemetry.SupplierId)
	require.Equal(t, "gb10-4t-collector/node-report", telemetry.SourceRef)
	require.Equal(t, SupplyCapacityTelemetrySourceNodeReport, telemetry.SourceType)
	require.Equal(t, int64(1800), telemetry.CapacityTokens)
	require.Equal(t, int64(450), telemetry.UsedTokens)
	require.Equal(t, int64(1350), telemetry.HeadroomTokens)
	require.InDelta(t, 0.25, telemetry.UtilizationRate, 0.000001)
	require.InDelta(t, 0.61, telemetry.GpuUtilizationRate, 0.000001)
	require.InDelta(t, 98.25, telemetry.QualityScore, 0.000001)
	require.InDelta(t, 0.42, telemetry.UnitCostQuota, 0.000001)
	require.Equal(t, now, telemetry.ObservedAt)
	require.Equal(t, 11, telemetry.RecordedBy)

	capacity, err := GetSupplyCapacityByID(telemetry.AppliedCapacityId)
	require.NoError(t, err)
	require.Equal(t, supplier.Id, capacity.SupplierId)
	require.Equal(t, "gb10-4t-collector", capacity.SupplyNode)
	require.Equal(t, "gpt-collector", capacity.ModelName)
	require.Equal(t, int64(1800), capacity.CapacityTokens)
	require.Equal(t, int64(450), capacity.UsedTokens)
	require.Equal(t, int64(1350), capacity.HeadroomTokens)
	require.Equal(t, telemetry.Id, capacity.LastTelemetryId)
	require.Equal(t, "gb10-4t-collector/node-report", capacity.TelemetrySourceRef)
	require.Equal(t, now, capacity.TelemetryObservedAt)
}

func TestSweepSupplyCapacityTelemetryCollectsAndSkips(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != SupplyCapacityTelemetryCollectPath {
			t.Errorf("unexpected telemetry path %q", r.URL.Path)
		}
		if r.URL.Query().Get("model") != "gpt-sweep" {
			t.Errorf("unexpected model query %q", r.URL.Query().Get("model"))
		}
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"supply_node":          r.URL.Query().Get("supply_node"),
			"model_name":           "gpt-sweep",
			"capacity_tokens":      2200,
			"used_tokens":          550,
			"gpu_utilization_rate": 0.58,
			"quality_score":        98.75,
			"unit_cost_quota":      0.39,
			"observed_at":          now,
			"source_ref":           "gb10-4t-sweep/node-report",
			"notes":                "sweep test telemetry",
		}); err != nil {
			t.Errorf("encode telemetry response: %v", err)
		}
	}))
	defer server.Close()

	supplier := &Supplier{Name: "gb10-4t-sweep", Type: SupplierTypeThirdParty, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())
	skippedSupplier := &Supplier{Name: "gb10-4t-sweep-no-channel", Type: SupplierTypeThirdParty, Status: common.ChannelStatusEnabled}
	require.NoError(t, skippedSupplier.Insert())
	baseURL := server.URL
	channel := &Channel{
		Type:       1,
		Key:        "sk-gb10-sweep",
		Status:     common.ChannelStatusEnabled,
		Name:       "gb10-4t-sweep",
		SupplierId: supplier.Id,
		BaseURL:    &baseURL,
		Models:     "other-model,gpt-sweep",
		Group:      "default",
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, (&SupplyCapacity{
		SupplierId:     supplier.Id,
		SupplyNode:     "gb10-4t-sweep",
		ModelName:      "gpt-sweep",
		PeriodStart:    1000,
		PeriodEnd:      2000,
		CapacityTokens: 1000,
		Status:         1,
	}).Insert())
	require.NoError(t, (&SupplyCapacity{
		SupplierId:     skippedSupplier.Id,
		SupplyNode:     "gb10-4t-sweep-skipped",
		ModelName:      "gpt-sweep",
		PeriodStart:    1000,
		PeriodEnd:      2000,
		CapacityTokens: 1000,
		Status:         1,
	}).Insert())

	result, err := SweepSupplyCapacityTelemetry(SupplyCapacityTelemetrySweepInput{
		ModelName:   "gpt-sweep",
		PeriodStart: 1000,
		PeriodEnd:   2000,
	}, 13)
	require.NoError(t, err)
	require.Equal(t, 2, result.AttemptedCount)
	require.Equal(t, 1, result.CollectedCount)
	require.Equal(t, 1, result.SkippedCount)
	require.Len(t, result.Collected, 1)
	require.Len(t, result.Skipped, 1)
	require.Equal(t, 1, requestCount)
	require.Equal(t, "gb10-4t-sweep/node-report", result.Collected[0].SourceRef)
	require.Equal(t, int64(550), result.Collected[0].UsedTokens)
	require.Contains(t, result.Skipped[0].Reason, "no enabled channel")

	capacity, err := GetSupplyCapacityByID(result.Collected[0].AppliedCapacityId)
	require.NoError(t, err)
	require.Equal(t, result.Collected[0].Id, capacity.LastTelemetryId)
	require.Equal(t, "gb10-4t-sweep/node-report", capacity.TelemetrySourceRef)
	require.Equal(t, int64(550), capacity.UsedTokens)
	require.Equal(t, int64(1650), capacity.HeadroomTokens)
}
