package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UsageAggregateDaily struct {
	Id                  int    `json:"id"`
	Day                 string `json:"day" gorm:"size:10;not null;uniqueIndex:uk_usage_aggregate_daily,priority:1;index"`
	TenantId            int    `json:"tenant_id" gorm:"not null;default:0;uniqueIndex:uk_usage_aggregate_daily,priority:2;index"`
	AppId               int    `json:"app_id" gorm:"not null;default:0;uniqueIndex:uk_usage_aggregate_daily,priority:3;index"`
	EndCustomerId       int    `json:"end_customer_id" gorm:"not null;default:0;uniqueIndex:uk_usage_aggregate_daily,priority:4;index"`
	UserId              int    `json:"user_id" gorm:"not null;default:0;uniqueIndex:uk_usage_aggregate_daily,priority:5;index"`
	TokenId             int    `json:"token_id" gorm:"not null;default:0;uniqueIndex:uk_usage_aggregate_daily,priority:6;index"`
	SupplierId          int    `json:"supplier_id" gorm:"not null;default:0;uniqueIndex:uk_usage_aggregate_daily,priority:7;index"`
	ChannelId           int    `json:"channel_id" gorm:"not null;default:0;uniqueIndex:uk_usage_aggregate_daily,priority:8;index"`
	ModelName           string `json:"model_name" gorm:"size:128;not null;default:'';uniqueIndex:uk_usage_aggregate_daily,priority:9;index"`
	BillingMode         string `json:"billing_mode" gorm:"size:32;not null;default:'';uniqueIndex:uk_usage_aggregate_daily,priority:10;index"`
	Status              string `json:"status" gorm:"size:32;not null;default:'success';uniqueIndex:uk_usage_aggregate_daily,priority:11;index"`
	RequestCount        int64  `json:"request_count" gorm:"default:0"`
	PromptTokens        int64  `json:"prompt_tokens" gorm:"default:0"`
	FreshPromptTokens   int64  `json:"fresh_prompt_tokens" gorm:"default:0"`
	CachedTokens        int64  `json:"cached_tokens" gorm:"default:0"`
	CacheCreationTokens int64  `json:"cache_creation_tokens" gorm:"default:0"`
	CompletionTokens    int64  `json:"completion_tokens" gorm:"default:0"`
	SellQuota           int64  `json:"sell_quota" gorm:"default:0"`
	CostQuota           int64  `json:"cost_quota" gorm:"default:0"`
	PostpaidQuota       int64  `json:"postpaid_quota" gorm:"default:0"`
	CacheHitCount       int64  `json:"cache_hit_count" gorm:"default:0"`
	LatencyMsTotal      int64  `json:"latency_ms_total" gorm:"default:0"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint;index"`
}

func UsageLedgerDay(createdAt int64) string {
	if createdAt <= 0 {
		createdAt = common.GetTimestamp()
	}
	return time.Unix(createdAt, 0).Format("2006-01-02")
}

func UpsertUsageAggregateDailyTx(tx *gorm.DB, ledger *UsageLedger) error {
	if tx == nil || ledger == nil {
		return nil
	}
	now := common.GetTimestamp()
	cacheHitCount := int64(0)
	if ledger.CacheHit {
		cacheHitCount = 1
	}
	item := &UsageAggregateDaily{
		Day:                 UsageLedgerDay(ledger.CreatedAt),
		TenantId:            ledger.TenantId,
		AppId:               ledger.AppId,
		EndCustomerId:       ledger.EndCustomerId,
		UserId:              ledger.UserId,
		TokenId:             ledger.TokenId,
		SupplierId:          ledger.SupplierId,
		ChannelId:           ledger.ChannelId,
		ModelName:           ledger.ModelName,
		BillingMode:         ledger.BillingMode,
		Status:              ledger.Status,
		RequestCount:        1,
		PromptTokens:        int64(ledger.PromptTokens),
		FreshPromptTokens:   int64(ledger.FreshPromptTokens),
		CachedTokens:        int64(ledger.CachedTokens),
		CacheCreationTokens: int64(ledger.CacheCreationTokens),
		CompletionTokens:    int64(ledger.CompletionTokens),
		SellQuota:           int64(ledger.SellQuota),
		CostQuota:           int64(ledger.CostQuota),
		PostpaidQuota:       int64(ledger.PostpaidQuota),
		CacheHitCount:       cacheHitCount,
		LatencyMsTotal:      int64(ledger.LatencyMs),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "day"},
			{Name: "tenant_id"},
			{Name: "app_id"},
			{Name: "end_customer_id"},
			{Name: "user_id"},
			{Name: "token_id"},
			{Name: "supplier_id"},
			{Name: "channel_id"},
			{Name: "model_name"},
			{Name: "billing_mode"},
			{Name: "status"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"request_count":         gorm.Expr("request_count + ?", item.RequestCount),
			"prompt_tokens":         gorm.Expr("prompt_tokens + ?", item.PromptTokens),
			"fresh_prompt_tokens":   gorm.Expr("fresh_prompt_tokens + ?", item.FreshPromptTokens),
			"cached_tokens":         gorm.Expr("cached_tokens + ?", item.CachedTokens),
			"cache_creation_tokens": gorm.Expr("cache_creation_tokens + ?", item.CacheCreationTokens),
			"completion_tokens":     gorm.Expr("completion_tokens + ?", item.CompletionTokens),
			"sell_quota":            gorm.Expr("sell_quota + ?", item.SellQuota),
			"cost_quota":            gorm.Expr("cost_quota + ?", item.CostQuota),
			"postpaid_quota":        gorm.Expr("postpaid_quota + ?", item.PostpaidQuota),
			"cache_hit_count":       gorm.Expr("cache_hit_count + ?", item.CacheHitCount),
			"latency_ms_total":      gorm.Expr("latency_ms_total + ?", item.LatencyMsTotal),
			"updated_at":            now,
		}),
	}).Create(item).Error
}
