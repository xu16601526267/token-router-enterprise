package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCalculateUsageCostQuotaCacheAware(t *testing.T) {
	input := UsageCostInput{
		PromptTokens:        1000,
		CachedTokens:        300,
		CacheCreationTokens: 100,
		CompletionTokens:    200,
	}
	agreement := &model.SupplierAgreement{
		CostModelRatio:         0.5,
		CostCompletionRatio:    2,
		CostCacheRatio:         0.1,
		CostCacheCreationRatio: 0.25,
	}

	// (fresh 600 + cached 300*0.1 + cache-write 100*0.25 + completion 200*2) * 0.5 = 527.5 => 528
	require.Equal(t, 528, CalculateUsageCostQuota(input, agreement))
}

func TestRecordUsageWritesLedgerAndIsIdempotent(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	supplier := &model.Supplier{Name: "gb10-4t", Type: model.SupplierTypeThirdParty}
	require.NoError(t, supplier.Insert())
	channel := &model.Channel{
		Id:         101,
		Name:       "gb10-4t-channel",
		Key:        "sk-test",
		Status:     common.ChannelStatusEnabled,
		SupplierId: supplier.Id,
	}
	require.NoError(t, model.DB.Create(channel).Error)
	agreement := &model.SupplierAgreement{
		SupplierId:             supplier.Id,
		ModelName:              "gpt-test",
		CostModelRatio:         1,
		CostCompletionRatio:    2,
		CostCacheRatio:         0.1,
		CostCacheCreationRatio: 0.5,
		Status:                 1,
	}
	require.NoError(t, agreement.Insert())

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("X-Session-Id", "session-a")
	req.Header.Set(common.UsageLedgerRequestIdHeader, "demand-req-1")
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Set(common.RequestIdKey, "relay-req-1")
	ctx.Set("sla_tier", "standard")

	relayInfo := &relaycommon.RelayInfo{
		UserId:          7,
		TokenId:         8,
		RequestId:       "relay-req-1",
		OriginModelName: "gpt-test",
		StartTime:       time.Now().Add(-2 * time.Second),
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:      channel.Id,
			ChannelBaseUrl: "gb10-4t",
		},
	}
	usage := &dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 20,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         40,
			CachedCreationTokens: 10,
		},
	}

	ledger, err := RecordUsage(ctx, relayInfo, usage, 200)
	require.NoError(t, err)
	require.Equal(t, "demand-req-1", ledger.RequestId)
	require.Equal(t, "session-a", ledger.SessionId)
	require.Equal(t, supplier.Id, ledger.SupplierId)
	require.Equal(t, channel.Id, ledger.ChannelId)
	require.Equal(t, 50, ledger.FreshPromptTokens)
	require.Equal(t, 99, ledger.CostQuota)
	require.Equal(t, 200, ledger.SellQuota)
	require.True(t, ledger.CacheHit)
	require.Equal(t, "standard", ledger.SlaTier)
	require.Equal(t, "gb10-4t-channel", ledger.SupplyNode)

	_, err = RecordUsage(ctx, relayInfo, usage, 999)
	require.NoError(t, err)
	var count int64
	require.NoError(t, model.DB.Model(&model.UsageLedger{}).Where("request_id = ?", "demand-req-1").Count(&count).Error)
	require.Equal(t, int64(1), count)

	var saved model.UsageLedger
	require.NoError(t, model.DB.Where("request_id = ?", "demand-req-1").First(&saved).Error)
	require.Equal(t, 200, saved.SellQuota)
}

func TestRecordUsageAsyncOutboxWritesLedgerAggregateAndIsIdempotent(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	supplier := &model.Supplier{Name: "async-supplier", Type: model.SupplierTypeThirdParty}
	require.NoError(t, supplier.Insert())
	channel := &model.Channel{
		Id:         202,
		Name:       "async-channel",
		Key:        "sk-test",
		Status:     common.ChannelStatusEnabled,
		SupplierId: supplier.Id,
	}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, (&model.SupplierAgreement{
		SupplierId:             supplier.Id,
		ModelName:              "gpt-async",
		CostModelRatio:         1,
		CostCompletionRatio:    2,
		CostCacheRatio:         0.1,
		CostCacheCreationRatio: 0.5,
		Status:                 1,
	}).Insert())

	ctx := tenantTestContext("async-usage-req-1")
	ctx.Set("sla_tier", "standard")
	relayInfo := &relaycommon.RelayInfo{
		UserId:            77,
		TokenId:           88,
		RequestId:         "async-usage-req-1",
		OriginModelName:   "gpt-async",
		TenantBillingMode: model.BillingModePrepaid,
		StartTime:         time.Now().Add(-time.Second),
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:      channel.Id,
			ChannelBaseUrl: "async-upstream",
		},
	}
	usage := &dto.Usage{PromptTokens: 100, CompletionTokens: 20}

	RecordUsageAsync(ctx, relayInfo, usage, 200)
	require.Eventually(t, func() bool {
		_ = ProcessUsageRecordOutboxByRequestID(context.Background(), "async-usage-req-1")
		var count int64
		if err := model.DB.Model(&model.UsageLedger{}).Where("request_id = ?", "async-usage-req-1").Count(&count).Error; err != nil {
			return false
		}
		return count == 1
	}, time.Second, 10*time.Millisecond)

	item, err := model.GetUsageRecordOutboxByRequestID("async-usage-req-1")
	require.NoError(t, err)
	require.Equal(t, model.UsageRecordOutboxStatusSucceeded, item.Status)
	var aggregate model.UsageAggregateDaily
	require.NoError(t, model.DB.Where("day = ? AND user_id = ? AND token_id = ? AND model_name = ?", model.UsageLedgerDay(common.GetTimestamp()), relayInfo.UserId, relayInfo.TokenId, "gpt-async").First(&aggregate).Error)
	require.Equal(t, int64(1), aggregate.RequestCount)
	require.Equal(t, int64(200), aggregate.SellQuota)

	RecordUsageAsync(ctx, relayInfo, usage, 999)
	require.Eventually(t, func() bool {
		var count int64
		if err := model.DB.Model(&model.UsageLedger{}).Where("request_id = ?", "async-usage-req-1").Count(&count).Error; err != nil {
			return false
		}
		return count == 1
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, model.DB.Where("day = ? AND user_id = ? AND token_id = ? AND model_name = ?", aggregate.Day, relayInfo.UserId, relayInfo.TokenId, "gpt-async").First(&aggregate).Error)
	require.Equal(t, int64(1), aggregate.RequestCount)
	require.Equal(t, int64(200), aggregate.SellQuota)
}
