package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UsageLedger struct {
	Id                  int    `json:"id"`
	RequestId           string `json:"request_id" gorm:"size:128;not null;uniqueIndex:uk_usage_ledger_request_id"`
	SessionId           string `json:"session_id" gorm:"size:128;default:'';index"`
	SupplierId          int    `json:"supplier_id" gorm:"index"`
	ChannelId           int    `json:"channel_id" gorm:"index"`
	UserId              int    `json:"user_id" gorm:"index"`
	TokenId             int    `json:"token_id" gorm:"index"`
	TenantId            int    `json:"tenant_id" gorm:"index;default:0"`
	EndCustomerId       int    `json:"end_customer_id" gorm:"index;default:0"`
	AppId               int    `json:"app_id" gorm:"index;default:0"`
	ModelName           string `json:"model_name" gorm:"size:128;default:'';index"`
	PromptTokens        int    `json:"prompt_tokens" gorm:"default:0"`
	FreshPromptTokens   int    `json:"fresh_prompt_tokens" gorm:"default:0"`
	CachedTokens        int    `json:"cached_tokens" gorm:"default:0"`
	CacheCreationTokens int    `json:"cache_creation_tokens" gorm:"default:0"`
	CompletionTokens    int    `json:"completion_tokens" gorm:"default:0"`
	SellQuota           int    `json:"sell_quota" gorm:"default:0"`
	CostQuota           int    `json:"cost_quota" gorm:"default:0"`
	BillingMode         string `json:"billing_mode" gorm:"size:32;default:'';index"`
	BillingPeriod       string `json:"billing_period" gorm:"size:32;default:'';index"`
	PriceSnapshot       string `json:"price_snapshot" gorm:"type:text"`
	PostpaidQuota       int    `json:"postpaid_quota" gorm:"default:0"`
	CacheHit            bool   `json:"cache_hit" gorm:"default:false;index"`
	LatencyMs           int    `json:"latency_ms" gorm:"default:0"`
	Status              string `json:"status" gorm:"size:32;default:'success';index"`
	SlaTier             string `json:"sla_tier" gorm:"size:64;default:'';index"`
	SupplyNode          string `json:"supply_node" gorm:"size:128;default:'';index"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint;index"`
}

func (l *UsageLedger) normalize() {
	l.RequestId = strings.TrimSpace(l.RequestId)
	l.SessionId = strings.TrimSpace(l.SessionId)
	l.ModelName = strings.TrimSpace(l.ModelName)
	l.Status = strings.TrimSpace(l.Status)
	l.SlaTier = strings.TrimSpace(l.SlaTier)
	l.SupplyNode = strings.TrimSpace(l.SupplyNode)
	l.BillingMode = strings.TrimSpace(l.BillingMode)
	l.BillingPeriod = strings.TrimSpace(l.BillingPeriod)
	if l.Status == "" {
		l.Status = "success"
	}
	if l.CreatedAt == 0 {
		l.CreatedAt = common.GetTimestamp()
	}
	if l.CachedTokens > 0 {
		l.CacheHit = true
	}
	if l.FreshPromptTokens == 0 && l.PromptTokens > 0 {
		l.FreshPromptTokens = l.PromptTokens - l.CachedTokens - l.CacheCreationTokens
		if l.FreshPromptTokens < 0 {
			l.FreshPromptTokens = 0
		}
	}
}

func (l *UsageLedger) InsertIdempotent() error {
	_, err := l.InsertIdempotentRowsAffected()
	return err
}

func (l *UsageLedger) InsertIdempotentRowsAffected() (int64, error) {
	return l.InsertIdempotentRowsAffectedTx(DB)
}

func (l *UsageLedger) InsertIdempotentRowsAffectedTx(tx *gorm.DB) (int64, error) {
	if tx == nil {
		return 0, errors.New("usage ledger db is required")
	}
	l.normalize()
	if l.RequestId == "" {
		return 0, errors.New("usage ledger request_id is required")
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(l)
	return result.RowsAffected, result.Error
}

func GetUsageLedgerByRequestID(requestId string) (*UsageLedger, error) {
	var ledger UsageLedger
	err := DB.Where("request_id = ?", strings.TrimSpace(requestId)).First(&ledger).Error
	if err != nil {
		return nil, err
	}
	return &ledger, nil
}

type UsageLedgerFilters struct {
	RequestId     string
	SessionId     string
	SupplierId    int
	ChannelId     int
	UserId        int
	TokenId       int
	TenantId      int
	AppId         int
	EndCustomerId int
	BillingMode   string
	BillingPeriod string
	ModelName     string
	Status        string
	StartTime     int64
	EndTime       int64
}

func SearchUsageLedgers(filters UsageLedgerFilters, offset int, limit int) ([]*UsageLedger, int64, error) {
	db := DB.Model(&UsageLedger{})
	if filters.RequestId != "" {
		db = db.Where("request_id = ?", strings.TrimSpace(filters.RequestId))
	}
	if filters.SessionId != "" {
		db = db.Where("session_id = ?", strings.TrimSpace(filters.SessionId))
	}
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if filters.ChannelId > 0 {
		db = db.Where("channel_id = ?", filters.ChannelId)
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
	}
	if filters.TokenId > 0 {
		db = db.Where("token_id = ?", filters.TokenId)
	}
	if filters.TenantId > 0 {
		db = db.Where("tenant_id = ?", filters.TenantId)
	}
	if filters.AppId > 0 {
		db = db.Where("app_id = ?", filters.AppId)
	}
	if filters.EndCustomerId > 0 {
		db = db.Where("end_customer_id = ?", filters.EndCustomerId)
	}
	if filters.BillingMode != "" {
		db = db.Where("billing_mode = ?", strings.TrimSpace(filters.BillingMode))
	}
	if filters.BillingPeriod != "" {
		db = db.Where("billing_period = ?", strings.TrimSpace(filters.BillingPeriod))
	}
	if filters.ModelName != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if filters.Status != "" {
		db = db.Where("status = ?", strings.TrimSpace(filters.Status))
	}
	if filters.StartTime > 0 {
		db = db.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime > 0 {
		db = db.Where("created_at <= ?", filters.EndTime)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var ledgers []*UsageLedger
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&ledgers).Error; err != nil {
		return nil, 0, err
	}
	return ledgers, total, nil
}
