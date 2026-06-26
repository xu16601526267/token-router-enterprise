package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type UsageCostInput struct {
	PromptTokens        int
	CachedTokens        int
	CacheCreationTokens int
	CompletionTokens    int
}

type UsageRecordMeta struct {
	RequestID  string
	SessionID  string
	SlaTier    string
	SupplyNode string
	LatencyMs  int
	CreatedAt  int64
}

func freshPromptTokens(input UsageCostInput) int {
	fresh := input.PromptTokens - input.CachedTokens - input.CacheCreationTokens
	if fresh < 0 {
		return 0
	}
	return fresh
}

func CalculateUsageCostQuota(input UsageCostInput, agreement *model.SupplierAgreement) int {
	if agreement == nil {
		return 0
	}
	if agreement.UsePrice {
		if agreement.CostModelPrice <= 0 {
			return 0
		}
		quota := decimal.NewFromFloat(agreement.CostModelPrice).
			Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
			Round(0).
			IntPart()
		return int(quota)
	}
	if agreement.CostModelRatio <= 0 {
		return 0
	}

	cost := decimal.NewFromInt(int64(freshPromptTokens(input))).
		Add(decimal.NewFromInt(int64(input.CachedTokens)).Mul(decimal.NewFromFloat(agreement.CostCacheRatio))).
		Add(decimal.NewFromInt(int64(input.CacheCreationTokens)).Mul(decimal.NewFromFloat(agreement.CostCacheCreationRatio))).
		Add(decimal.NewFromInt(int64(input.CompletionTokens)).Mul(decimal.NewFromFloat(agreement.CostCompletionRatio))).
		Mul(decimal.NewFromFloat(agreement.CostModelRatio))
	if !cost.IsPositive() {
		return 0
	}
	quota := int(cost.Round(0).IntPart())
	if quota == 0 {
		return 1
	}
	return quota
}

func usageCostInputFromUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) UsageCostInput {
	if usage == nil {
		promptTokens := 0
		if relayInfo != nil {
			promptTokens = relayInfo.GetEstimatePromptTokens()
		}
		return UsageCostInput{PromptTokens: promptTokens}
	}
	promptTokens := usage.PromptTokens
	if promptTokens == 0 && usage.InputTokens > 0 {
		promptTokens = usage.InputTokens
	}
	completionTokens := usage.CompletionTokens
	if completionTokens == 0 && usage.OutputTokens > 0 {
		completionTokens = usage.OutputTokens
	}
	cacheCreationTokens := usage.PromptTokensDetails.CachedCreationTokens
	if cacheCreationTokens == 0 {
		cacheCreationTokens = usage.ClaudeCacheCreation5mTokens + usage.ClaudeCacheCreation1hTokens
	}
	return UsageCostInput{
		PromptTokens:        promptTokens,
		CachedTokens:        usage.PromptTokensDetails.CachedTokens,
		CacheCreationTokens: cacheCreationTokens,
		CompletionTokens:    completionTokens,
	}
}

func usageFromRealtimeUsage(usage *dto.RealtimeUsage) *dto.Usage {
	if usage == nil {
		return nil
	}
	return &dto.Usage{
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: usage.InputTokenDetails.CachedTokens,
			TextTokens:   usage.InputTokenDetails.TextTokens,
			AudioTokens:  usage.InputTokenDetails.AudioTokens,
			ImageTokens:  usage.InputTokenDetails.ImageTokens,
		},
		CompletionTokenDetails: usage.OutputTokenDetails,
		InputTokens:            usage.InputTokens,
		OutputTokens:           usage.OutputTokens,
	}
}

func usageRecordRequestID(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) string {
	if ctx != nil {
		if requestID := strings.TrimSpace(ctx.GetHeader(common.UsageLedgerRequestIdHeader)); requestID != "" {
			return requestID
		}
	}
	if relayInfo != nil && relayInfo.RequestHeaders != nil {
		if requestID := strings.TrimSpace(relayInfo.RequestHeaders[common.UsageLedgerRequestIdHeader]); requestID != "" {
			return requestID
		}
	}
	if relayInfo != nil && strings.TrimSpace(relayInfo.RequestId) != "" {
		return strings.TrimSpace(relayInfo.RequestId)
	}
	if ctx == nil {
		return ""
	}
	return strings.TrimSpace(ctx.GetString(common.RequestIdKey))
}

func usageRecordSessionID(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) string {
	keys := []string{"X-Session-Id", "X-Session-ID", "session_id", "prompt_cache_key", "X-Prompt-Cache-Key"}
	if relayInfo != nil {
		for _, key := range keys {
			if relayInfo.RuntimeHeadersOverride != nil {
				if value, ok := relayInfo.RuntimeHeadersOverride[key]; ok {
					if sessionID := strings.TrimSpace(fmt.Sprint(value)); sessionID != "" {
						return sessionID
					}
				}
			}
			if relayInfo.RequestHeaders != nil {
				if value := strings.TrimSpace(relayInfo.RequestHeaders[key]); value != "" {
					return value
				}
			}
		}
	}
	if ctx == nil {
		return ""
	}
	for _, key := range keys {
		if value := strings.TrimSpace(ctx.GetHeader(key)); value != "" {
			return value
		}
		if value := strings.TrimSpace(ctx.GetString(key)); value != "" {
			return value
		}
	}
	return ""
}

func usageRecordChannelID(relayInfo *relaycommon.RelayInfo) int {
	if relayInfo == nil || relayInfo.ChannelMeta == nil {
		return 0
	}
	return relayInfo.ChannelMeta.ChannelId
}

func usageRecordLatencyMs(relayInfo *relaycommon.RelayInfo) int {
	if relayInfo == nil || relayInfo.StartTime.IsZero() {
		return 0
	}
	return int(time.Since(relayInfo.StartTime).Milliseconds())
}

func usageRecordSupplyNode(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) string {
	if ctx != nil {
		if value := strings.TrimSpace(ctx.GetString("supply_node")); value != "" {
			return value
		}
	}
	if relayInfo != nil && relayInfo.ChannelMeta != nil {
		if relayInfo.ChannelMeta.ChannelId > 0 {
			if channel, err := model.CacheGetChannel(relayInfo.ChannelMeta.ChannelId); err == nil && channel != nil {
				if name := strings.TrimSpace(channel.Name); name != "" {
					return name
				}
			}
		}
		return strings.TrimSpace(relayInfo.ChannelMeta.ChannelBaseUrl)
	}
	return ""
}

func usageRecordSlaTier(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	return strings.TrimSpace(ctx.GetString("sla_tier"))
}

func resolveUsageRecordSupplierID(channelID int) (int, error) {
	if channelID <= 0 {
		return 0, nil
	}
	channel, err := model.CacheGetChannel(channelID)
	if err != nil {
		return 0, err
	}
	return channel.SupplierId, nil
}

func resolveUsageRecordAgreement(supplierID int, modelName string, at int64) (*model.SupplierAgreement, error) {
	if supplierID <= 0 {
		return nil, nil
	}
	agreement, err := model.FindActiveSupplierAgreement(supplierID, modelName, at)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("no active supplier agreement for supplier_id=%d model=%s", supplierID, modelName)
	}
	return agreement, err
}

func newUsageRecordMeta(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) UsageRecordMeta {
	return UsageRecordMeta{
		RequestID:  usageRecordRequestID(ctx, relayInfo),
		SessionID:  usageRecordSessionID(ctx, relayInfo),
		SlaTier:    usageRecordSlaTier(ctx),
		SupplyNode: usageRecordSupplyNode(ctx, relayInfo),
		LatencyMs:  usageRecordLatencyMs(relayInfo),
		CreatedAt:  common.GetTimestamp(),
	}
}

func recordUsageWithMeta(relayInfo *relaycommon.RelayInfo, usage *dto.Usage, sellQuota int, meta UsageRecordMeta) (*model.UsageLedger, error) {
	if relayInfo == nil {
		return nil, errors.New("relayInfo is required")
	}
	if meta.RequestID == "" {
		return nil, errors.New("request id is required")
	}

	channelID := usageRecordChannelID(relayInfo)
	supplierID, err := resolveUsageRecordSupplierID(channelID)
	if err != nil {
		return nil, err
	}
	if meta.CreatedAt == 0 {
		meta.CreatedAt = common.GetTimestamp()
	}
	agreement, err := resolveUsageRecordAgreement(supplierID, relayInfo.OriginModelName, meta.CreatedAt)
	if err != nil {
		return nil, err
	}
	costInput := usageCostInputFromUsage(relayInfo, usage)
	costQuota := CalculateUsageCostQuota(costInput, agreement)

	ledger := &model.UsageLedger{
		RequestId:           meta.RequestID,
		SessionId:           meta.SessionID,
		SupplierId:          supplierID,
		ChannelId:           channelID,
		UserId:              relayInfo.UserId,
		TokenId:             relayInfo.TokenId,
		ModelName:           relayInfo.OriginModelName,
		PromptTokens:        costInput.PromptTokens,
		FreshPromptTokens:   freshPromptTokens(costInput),
		CachedTokens:        costInput.CachedTokens,
		CacheCreationTokens: costInput.CacheCreationTokens,
		CompletionTokens:    costInput.CompletionTokens,
		SellQuota:           sellQuota,
		CostQuota:           costQuota,
		CacheHit:            costInput.CachedTokens > 0,
		LatencyMs:           meta.LatencyMs,
		Status:              "success",
		SlaTier:             meta.SlaTier,
		SupplyNode:          meta.SupplyNode,
		CreatedAt:           meta.CreatedAt,
	}
	DecorateTenantLedger(ledger, relayInfo)
	_, err = insertUsageLedgerWithTenantCredit(ledger)
	if err != nil {
		return nil, err
	}
	saved, err := model.GetUsageLedgerByRequestID(meta.RequestID)
	if err != nil {
		return ledger, nil
	}
	return saved, nil
}

func insertUsageLedgerWithTenantCredit(ledger *model.UsageLedger) (int64, error) {
	var rowsAffected int64
	err := retryTenantDBWrite(func() error {
		return model.DB.Transaction(func(tx *gorm.DB) error {
			affected, err := ledger.InsertIdempotentRowsAffectedTx(tx)
			if err != nil {
				return err
			}
			rowsAffected = affected
			if affected == 0 {
				return nil
			}
			return applyTenantLedgerCreditTx(tx, ledger)
		})
	})
	return rowsAffected, err
}

func RecordUsage(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, sellQuota int) (*model.UsageLedger, error) {
	return recordUsageWithMeta(relayInfo, usage, sellQuota, newUsageRecordMeta(ctx, relayInfo))
}

func RecordUsageAsync(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, sellQuota int) {
	meta := newUsageRecordMeta(ctx, relayInfo)
	gopool.Go(func() {
		if _, err := recordUsageWithMeta(relayInfo, usage, sellQuota, meta); err != nil {
			logger.LogError(context.Background(), "failed to record usage ledger: "+err.Error())
		}
	})
}
