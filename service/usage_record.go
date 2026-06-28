package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	usageRecordOutboxDefaultInterval = 5 * time.Second
	usageRecordOutboxStaleAfter      = 2 * time.Minute
	usageRecordOutboxBatchLimit      = 200
)

var usageRecordOutboxWorkerOnce sync.Once

type usageRecordOutboxPayload struct {
	RelayInfo usageRecordOutboxRelayInfo `json:"relay_info"`
	Usage     *dto.Usage                 `json:"usage"`
	SellQuota int                        `json:"sell_quota"`
	Meta      UsageRecordMeta            `json:"meta"`
}

type usageRecordOutboxRelayInfo struct {
	UserId                 int                    `json:"user_id"`
	TokenId                int                    `json:"token_id"`
	TokenKey               string                 `json:"token_key"`
	TenantId               int                    `json:"tenant_id"`
	AppId                  int                    `json:"app_id"`
	EndCustomerId          int                    `json:"end_customer_id"`
	ModelPolicyId          int                    `json:"model_policy_id"`
	TenantBillingMode      string                 `json:"tenant_billing_mode"`
	OriginModelName        string                 `json:"origin_model_name"`
	UsingGroup             string                 `json:"using_group"`
	FinalPreConsumedQuota  int                    `json:"final_pre_consumed_quota"`
	BillingSource          string                 `json:"billing_source"`
	PriceData              types.PriceData        `json:"price_data"`
	ChannelId              int                    `json:"channel_id"`
	ChannelBaseUrl         string                 `json:"channel_base_url"`
	StartUnixMilli         int64                  `json:"start_unix_milli"`
	RequestId              string                 `json:"request_id"`
	RequestHeaders         map[string]string      `json:"request_headers"`
	RuntimeHeadersOverride map[string]interface{} `json:"runtime_headers_override"`
}

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

func newUsageRecordOutboxPayload(relayInfo *relaycommon.RelayInfo, usage *dto.Usage, sellQuota int, meta UsageRecordMeta) usageRecordOutboxPayload {
	payload := usageRecordOutboxPayload{
		Usage:     usage,
		SellQuota: sellQuota,
		Meta:      meta,
	}
	if relayInfo == nil {
		return payload
	}
	payload.RelayInfo = usageRecordOutboxRelayInfo{
		UserId:                 relayInfo.UserId,
		TokenId:                relayInfo.TokenId,
		TokenKey:               relayInfo.TokenKey,
		TenantId:               relayInfo.TenantId,
		AppId:                  relayInfo.AppId,
		EndCustomerId:          relayInfo.EndCustomerId,
		ModelPolicyId:          relayInfo.ModelPolicyId,
		TenantBillingMode:      relayInfo.TenantBillingMode,
		OriginModelName:        relayInfo.OriginModelName,
		UsingGroup:             relayInfo.UsingGroup,
		FinalPreConsumedQuota:  relayInfo.FinalPreConsumedQuota,
		BillingSource:          relayInfo.BillingSource,
		PriceData:              relayInfo.PriceData,
		RequestId:              relayInfo.RequestId,
		RequestHeaders:         cloneStringMap(relayInfo.RequestHeaders),
		RuntimeHeadersOverride: cloneInterfaceMap(relayInfo.RuntimeHeadersOverride),
	}
	if !relayInfo.StartTime.IsZero() {
		payload.RelayInfo.StartUnixMilli = relayInfo.StartTime.UnixMilli()
	}
	if relayInfo.ChannelMeta != nil {
		payload.RelayInfo.ChannelId = relayInfo.ChannelMeta.ChannelId
		payload.RelayInfo.ChannelBaseUrl = relayInfo.ChannelMeta.ChannelBaseUrl
	}
	return payload
}

func (p usageRecordOutboxPayload) relayInfo() *relaycommon.RelayInfo {
	startTime := time.Now()
	if p.RelayInfo.StartUnixMilli > 0 {
		startTime = time.UnixMilli(p.RelayInfo.StartUnixMilli)
	}
	return &relaycommon.RelayInfo{
		UserId:                 p.RelayInfo.UserId,
		TokenId:                p.RelayInfo.TokenId,
		TokenKey:               p.RelayInfo.TokenKey,
		TenantId:               p.RelayInfo.TenantId,
		AppId:                  p.RelayInfo.AppId,
		EndCustomerId:          p.RelayInfo.EndCustomerId,
		ModelPolicyId:          p.RelayInfo.ModelPolicyId,
		TenantBillingMode:      p.RelayInfo.TenantBillingMode,
		OriginModelName:        p.RelayInfo.OriginModelName,
		UsingGroup:             p.RelayInfo.UsingGroup,
		FinalPreConsumedQuota:  p.RelayInfo.FinalPreConsumedQuota,
		BillingSource:          p.RelayInfo.BillingSource,
		PriceData:              p.RelayInfo.PriceData,
		RequestId:              p.RelayInfo.RequestId,
		RequestHeaders:         cloneStringMap(p.RelayInfo.RequestHeaders),
		RuntimeHeadersOverride: cloneInterfaceMap(p.RelayInfo.RuntimeHeadersOverride),
		StartTime:              startTime,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:      p.RelayInfo.ChannelId,
			ChannelBaseUrl: p.RelayInfo.ChannelBaseUrl,
		},
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneInterfaceMap(values map[string]interface{}) map[string]interface{} {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
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
			if err := applyTenantLedgerCreditTx(tx, ledger); err != nil {
				return err
			}
			return model.UpsertUsageAggregateDailyTx(tx, ledger)
		})
	})
	return rowsAffected, err
}

func RecordUsage(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, sellQuota int) (*model.UsageLedger, error) {
	return recordUsageWithMeta(relayInfo, usage, sellQuota, newUsageRecordMeta(ctx, relayInfo))
}

func RecordUsageAsync(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, sellQuota int) {
	meta := newUsageRecordMeta(ctx, relayInfo)
	payload := newUsageRecordOutboxPayload(relayInfo, usage, sellQuota, meta)
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		logger.LogError(context.Background(), "failed to marshal usage record outbox payload: "+err.Error())
		gopool.Go(func() {
			if _, err := recordUsageWithMeta(relayInfo, usage, sellQuota, meta); err != nil {
				logger.LogError(context.Background(), "failed to record usage ledger: "+err.Error())
			}
		})
		return
	}
	outbox := &model.UsageRecordOutbox{
		RequestId:   meta.RequestID,
		Payload:     string(rawPayload),
		Status:      model.UsageRecordOutboxStatusPending,
		NextRetryAt: common.GetTimestamp(),
	}
	if err := outbox.InsertIdempotent(); err != nil {
		logger.LogError(context.Background(), "failed to enqueue usage record outbox: "+err.Error())
		gopool.Go(func() {
			if _, err := recordUsageWithMeta(relayInfo, usage, sellQuota, meta); err != nil {
				logger.LogError(context.Background(), "failed to record usage ledger: "+err.Error())
			}
		})
		return
	}

	gopool.Go(func() {
		if err := ProcessUsageRecordOutboxByRequestID(context.Background(), meta.RequestID); err != nil {
			logger.LogError(context.Background(), "failed to process usage record outbox: "+err.Error())
		}
	})
}

func StartUsageRecordOutboxWorker() {
	if parseTruthyEnv(os.Getenv("USAGE_RECORD_OUTBOX_WORKER_DISABLED")) || !common.IsMasterNode {
		return
	}
	usageRecordOutboxWorkerOnce.Do(func() {
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("usage record outbox worker started: interval=%s", usageRecordOutboxDefaultInterval))
			ticker := time.NewTicker(usageRecordOutboxDefaultInterval)
			defer ticker.Stop()
			runUsageRecordOutboxWorkerOnce(context.Background())
			for range ticker.C {
				runUsageRecordOutboxWorkerOnce(context.Background())
			}
		})
	})
}

func runUsageRecordOutboxWorkerOnce(ctx context.Context) {
	if _, err := ProcessUsageRecordOutboxBatch(ctx, usageRecordOutboxBatchLimit); err != nil {
		logger.LogError(ctx, "usage record outbox worker failed: "+err.Error())
	}
}

func ProcessUsageRecordOutboxBatch(ctx context.Context, limit int) (int, error) {
	now := common.GetTimestamp()
	staleBefore := now - int64(usageRecordOutboxStaleAfter/time.Second)
	items, err := model.ListDueUsageRecordOutbox(now, staleBefore, limit)
	if err != nil {
		return 0, err
	}
	var lastErr error
	processed := 0
	for _, item := range items {
		if err := processUsageRecordOutboxItem(ctx, item); err != nil {
			lastErr = err
			logger.LogError(ctx, fmt.Sprintf("usage record outbox item %d failed: %v", item.Id, err))
		}
		processed++
	}
	return processed, lastErr
}

func ProcessUsageRecordOutboxByRequestID(ctx context.Context, requestID string) error {
	if strings.TrimSpace(requestID) == "" {
		return errors.New("usage record outbox request id is required")
	}
	item, err := model.GetUsageRecordOutboxByRequestID(requestID)
	if err != nil {
		return err
	}
	if item.Status == model.UsageRecordOutboxStatusSucceeded {
		return nil
	}
	return processUsageRecordOutboxItem(ctx, *item)
}

func processUsageRecordOutboxItem(ctx context.Context, item model.UsageRecordOutbox) error {
	now := common.GetTimestamp()
	staleBefore := now - int64(usageRecordOutboxStaleAfter/time.Second)
	claimed, err := model.MarkUsageRecordOutboxProcessing(item.Id, now, staleBefore)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	var payload usageRecordOutboxPayload
	if err := json.Unmarshal([]byte(item.Payload), &payload); err != nil {
		_ = markUsageRecordOutboxFailed(item, err)
		return err
	}
	if _, err := recordUsageWithMeta(payload.relayInfo(), payload.Usage, payload.SellQuota, payload.Meta); err != nil {
		_ = markUsageRecordOutboxFailed(item, err)
		return err
	}
	if err := model.MarkUsageRecordOutboxSucceeded(item.Id, common.GetTimestamp()); err != nil {
		logger.LogError(ctx, fmt.Sprintf("failed to mark usage record outbox %d succeeded: %v", item.Id, err))
		return err
	}
	return nil
}

func markUsageRecordOutboxFailed(item model.UsageRecordOutbox, err error) error {
	return model.MarkUsageRecordOutboxFailed(item.Id, usageRecordOutboxErrorString(err), nextUsageRecordOutboxRetryAt(item.RetryCount+1), common.GetTimestamp())
}

func nextUsageRecordOutboxRetryAt(retryCount int) int64 {
	if retryCount <= 0 {
		retryCount = 1
	}
	delay := 5 * time.Second
	for i := 1; i < retryCount && delay < 5*time.Minute; i++ {
		delay *= 2
	}
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	return common.GetTimestamp() + int64(delay/time.Second)
}

func usageRecordOutboxErrorString(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 2000 {
		return message[:2000]
	}
	return message
}
