package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SupplyPrepaidLotSourceManual     = "manual"
	SupplyPrepaidLotSourceAccounting = "accounting"
	SupplyPrepaidLotSourceExternal   = "external"

	SupplyPrepaidLotDrawdownSourceUsageLedger = "usage_ledger"
)

type SupplyPrepaidLot struct {
	Id                   int     `json:"id"`
	PrepaidLotKey        string  `json:"prepaid_lot_key" gorm:"size:768;not null;uniqueIndex:uk_supply_prepaid_lot_key"`
	SupplierId           int     `json:"supplier_id" gorm:"not null;index"`
	ChannelId            int     `json:"channel_id" gorm:"index;default:0"`
	SupplyNode           string  `json:"supply_node" gorm:"size:128;default:'';index"`
	ModelName            string  `json:"model_name" gorm:"size:128;default:'';index"`
	PeriodStart          int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd            int64   `json:"period_end" gorm:"bigint;not null;index"`
	PurchasedTokens      int64   `json:"purchased_tokens" gorm:"default:0"`
	UnitCostQuota        float64 `json:"unit_cost_quota" gorm:"default:0"`
	TotalCostQuota       float64 `json:"total_cost_quota" gorm:"default:0"`
	DrawdownTokens       int64   `json:"drawdown_tokens" gorm:"default:0"`
	DrawdownRequestCount int64   `json:"drawdown_request_count" gorm:"default:0"`
	RemainingTokens      int64   `json:"remaining_tokens" gorm:"default:0"`
	DrawdownRate         float64 `json:"drawdown_rate" gorm:"default:0"`
	DrawdownSourceType   string  `json:"drawdown_source_type" gorm:"size:64;default:'';index"`
	DrawdownSourceRef    string  `json:"drawdown_source_ref" gorm:"size:256;default:'';index"`
	DrawdownRefreshedAt  int64   `json:"drawdown_refreshed_at" gorm:"bigint;default:0;index"`
	SourceType           string  `json:"source_type" gorm:"size:64;not null;default:'manual';index"`
	SourceRef            string  `json:"source_ref" gorm:"size:256;not null;index"`
	ObservedAt           int64   `json:"observed_at" gorm:"bigint;not null;index"`
	ExternalRef          string  `json:"external_ref" gorm:"size:256;default:'';index"`
	RecordedBy           int     `json:"recorded_by" gorm:"default:0;index"`
	Notes                string  `json:"notes,omitempty" gorm:"type:text"`
	CreatedAt            int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt            int64   `json:"updated_at" gorm:"bigint"`
}

type SupplyPrepaidLotRecordInput struct {
	PrepaidLotKey   string  `json:"prepaid_lot_key"`
	SupplierId      int     `json:"supplier_id"`
	ChannelId       int     `json:"channel_id"`
	SupplyNode      string  `json:"supply_node"`
	ModelName       string  `json:"model_name"`
	PeriodStart     int64   `json:"period_start"`
	PeriodEnd       int64   `json:"period_end"`
	PurchasedTokens int64   `json:"purchased_tokens"`
	UnitCostQuota   float64 `json:"unit_cost_quota"`
	SourceType      string  `json:"source_type"`
	SourceRef       string  `json:"source_ref"`
	ObservedAt      int64   `json:"observed_at"`
	ExternalRef     string  `json:"external_ref"`
	Notes           string  `json:"notes"`
}

type SupplyPrepaidLotFilters struct {
	PrepaidLotId int
	SupplierId   int
	ChannelId    int
	SupplyNode   string
	ModelName    string
	SourceType   string
	StartTime    int64
	EndTime      int64
}

type SupplyPrepaidLotUsageRefreshInput struct {
	PrepaidLotId int    `json:"prepaid_lot_id"`
	SupplierId   int    `json:"supplier_id"`
	ChannelId    int    `json:"channel_id"`
	SupplyNode   string `json:"supply_node"`
	ModelName    string `json:"model_name"`
	SourceType   string `json:"source_type"`
	StartTime    int64  `json:"start_timestamp"`
	EndTime      int64  `json:"end_timestamp"`
}

func normalizeSupplyPrepaidLotSource(value string) string {
	switch strings.TrimSpace(value) {
	case SupplyPrepaidLotSourceAccounting:
		return SupplyPrepaidLotSourceAccounting
	case SupplyPrepaidLotSourceExternal:
		return SupplyPrepaidLotSourceExternal
	default:
		return SupplyPrepaidLotSourceManual
	}
}

func (lot *SupplyPrepaidLot) normalize() {
	lot.PrepaidLotKey = strings.TrimSpace(lot.PrepaidLotKey)
	lot.SupplyNode = strings.TrimSpace(lot.SupplyNode)
	lot.ModelName = strings.TrimSpace(lot.ModelName)
	lot.SourceType = normalizeSupplyPrepaidLotSource(lot.SourceType)
	lot.SourceRef = strings.TrimSpace(lot.SourceRef)
	lot.ExternalRef = strings.TrimSpace(lot.ExternalRef)
	lot.Notes = strings.TrimSpace(lot.Notes)
	lot.TotalCostQuota = float64(lot.PurchasedTokens) * lot.UnitCostQuota
	lot.RemainingTokens = lot.PurchasedTokens - lot.DrawdownTokens
	if lot.PurchasedTokens > 0 {
		lot.DrawdownRate = float64(lot.DrawdownTokens) / float64(lot.PurchasedTokens)
	} else {
		lot.DrawdownRate = 0
	}
	if lot.PrepaidLotKey == "" && lot.SourceRef != "" {
		lot.PrepaidLotKey = fmt.Sprintf("supply_prepaid_lot:%s:%s:%d:%d:%s:%s:%d:%d",
			lot.SourceType,
			lot.SourceRef,
			lot.SupplierId,
			lot.ChannelId,
			lot.SupplyNode,
			lot.ModelName,
			lot.PeriodStart,
			lot.PeriodEnd,
		)
	}
}

func (lot *SupplyPrepaidLot) validate() error {
	if lot.PrepaidLotKey == "" {
		return errors.New("prepaid_lot_key is required")
	}
	if lot.SupplierId <= 0 {
		return errors.New("supplier_id is required")
	}
	if lot.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if lot.PeriodEnd <= lot.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	if lot.PurchasedTokens <= 0 {
		return errors.New("purchased_tokens must be greater than 0")
	}
	if lot.UnitCostQuota < 0 {
		return errors.New("unit_cost_quota cannot be negative")
	}
	if lot.TotalCostQuota < 0 {
		return errors.New("total_cost_quota cannot be negative")
	}
	if lot.SourceRef == "" {
		return errors.New("source_ref is required")
	}
	if lot.ObservedAt <= 0 {
		return errors.New("observed_at is required")
	}
	return nil
}

func RecordSupplyPrepaidLot(input SupplyPrepaidLotRecordInput, recordedBy int) (*SupplyPrepaidLot, error) {
	lot := SupplyPrepaidLot{
		PrepaidLotKey:   input.PrepaidLotKey,
		SupplierId:      input.SupplierId,
		ChannelId:       input.ChannelId,
		SupplyNode:      input.SupplyNode,
		ModelName:       input.ModelName,
		PeriodStart:     input.PeriodStart,
		PeriodEnd:       input.PeriodEnd,
		PurchasedTokens: input.PurchasedTokens,
		UnitCostQuota:   input.UnitCostQuota,
		SourceType:      input.SourceType,
		SourceRef:       input.SourceRef,
		ObservedAt:      input.ObservedAt,
		ExternalRef:     input.ExternalRef,
		RecordedBy:      recordedBy,
		Notes:           input.Notes,
	}
	lot.normalize()
	if err := lot.validate(); err != nil {
		return nil, err
	}
	supplier, err := GetSupplierByID(lot.SupplierId)
	if err != nil {
		return nil, err
	}
	if supplier.Type != SupplierTypeSelfOperated {
		return nil, errors.New("supplier must be self_operated")
	}
	var existing SupplyPrepaidLot
	if err := DB.Where("prepaid_lot_key = ?", lot.PrepaidLotKey).First(&existing).Error; err == nil {
		lot.DrawdownTokens = existing.DrawdownTokens
		lot.DrawdownRequestCount = existing.DrawdownRequestCount
		lot.DrawdownSourceType = existing.DrawdownSourceType
		lot.DrawdownSourceRef = existing.DrawdownSourceRef
		lot.DrawdownRefreshedAt = existing.DrawdownRefreshedAt
		lot.CreatedAt = existing.CreatedAt
		lot.normalize()
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := common.GetTimestamp()
	if lot.CreatedAt <= 0 {
		lot.CreatedAt = now
	}
	lot.UpdatedAt = now

	if err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "prepaid_lot_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"supplier_id",
			"channel_id",
			"supply_node",
			"model_name",
			"period_start",
			"period_end",
			"purchased_tokens",
			"unit_cost_quota",
			"total_cost_quota",
			"drawdown_tokens",
			"drawdown_request_count",
			"remaining_tokens",
			"drawdown_rate",
			"drawdown_source_type",
			"drawdown_source_ref",
			"drawdown_refreshed_at",
			"source_type",
			"source_ref",
			"observed_at",
			"external_ref",
			"recorded_by",
			"notes",
			"updated_at",
		}),
	}).Create(&lot).Error; err != nil {
		return nil, err
	}

	var result SupplyPrepaidLot
	err = DB.Where("prepaid_lot_key = ?", lot.PrepaidLotKey).First(&result).Error
	return &result, err
}

func SearchSupplyPrepaidLots(filters SupplyPrepaidLotFilters, offset int, limit int) ([]*SupplyPrepaidLot, int64, error) {
	db := DB.Model(&SupplyPrepaidLot{})
	if filters.PrepaidLotId > 0 {
		db = db.Where("id = ?", filters.PrepaidLotId)
	}
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if filters.ChannelId > 0 {
		db = db.Where("channel_id = ?", filters.ChannelId)
	}
	if strings.TrimSpace(filters.SupplyNode) != "" {
		db = db.Where("supply_node = ?", strings.TrimSpace(filters.SupplyNode))
	}
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if sourceType := normalizeSupplyPrepaidLotSource(filters.SourceType); strings.TrimSpace(filters.SourceType) != "" {
		db = db.Where("source_type = ?", sourceType)
	}
	if filters.StartTime > 0 {
		db = db.Where("period_end >= ?", filters.StartTime)
	}
	if filters.EndTime > 0 {
		db = db.Where("period_start <= ?", filters.EndTime)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []*SupplyPrepaidLot
	err := db.Offset(offset).Limit(limit).Order("observed_at DESC, id DESC").Find(&rows).Error
	return rows, total, err
}

func RefreshSupplyPrepaidLotUsage(input SupplyPrepaidLotUsageRefreshInput) ([]*SupplyPrepaidLot, error) {
	lots, _, err := SearchSupplyPrepaidLots(SupplyPrepaidLotFilters{
		PrepaidLotId: input.PrepaidLotId,
		SupplierId:   input.SupplierId,
		ChannelId:    input.ChannelId,
		SupplyNode:   input.SupplyNode,
		ModelName:    input.ModelName,
		SourceType:   input.SourceType,
		StartTime:    input.StartTime,
		EndTime:      input.EndTime,
	}, 0, 100000)
	if err != nil {
		return nil, err
	}
	if len(lots) == 0 {
		return []*SupplyPrepaidLot{}, nil
	}

	now := common.GetTimestamp()
	updated := make([]*SupplyPrepaidLot, 0, len(lots))
	for _, lot := range lots {
		drawdown, err := usageDrawdownForSupplyPrepaidLot(*lot)
		if err != nil {
			return nil, err
		}
		lot.DrawdownTokens = drawdown.UsedTokens
		lot.DrawdownRequestCount = drawdown.RequestCount
		lot.RemainingTokens = lot.PurchasedTokens - drawdown.UsedTokens
		if lot.PurchasedTokens > 0 {
			lot.DrawdownRate = float64(drawdown.UsedTokens) / float64(lot.PurchasedTokens)
		} else {
			lot.DrawdownRate = 0
		}
		lot.DrawdownSourceType = SupplyPrepaidLotDrawdownSourceUsageLedger
		lot.DrawdownSourceRef = buildSupplyPrepaidLotDrawdownSourceRef(*lot)
		lot.DrawdownRefreshedAt = now
		lot.UpdatedAt = now
		if err := DB.Model(&SupplyPrepaidLot{}).
			Where("id = ?", lot.Id).
			Updates(map[string]any{
				"drawdown_tokens":        lot.DrawdownTokens,
				"drawdown_request_count": lot.DrawdownRequestCount,
				"remaining_tokens":       lot.RemainingTokens,
				"drawdown_rate":          lot.DrawdownRate,
				"drawdown_source_type":   lot.DrawdownSourceType,
				"drawdown_source_ref":    lot.DrawdownSourceRef,
				"drawdown_refreshed_at":  lot.DrawdownRefreshedAt,
				"updated_at":             lot.UpdatedAt,
			}).Error; err != nil {
			return nil, err
		}
		var saved SupplyPrepaidLot
		if err := DB.First(&saved, lot.Id).Error; err != nil {
			return nil, err
		}
		updated = append(updated, &saved)
	}
	return updated, nil
}

type supplyPrepaidLotUsageDrawdown struct {
	UsedTokens   int64
	RequestCount int64
}

func usageDrawdownForSupplyPrepaidLot(lot SupplyPrepaidLot) (supplyPrepaidLotUsageDrawdown, error) {
	if lot.SupplierId <= 0 || lot.PeriodStart <= 0 || lot.PeriodEnd < lot.PeriodStart {
		return supplyPrepaidLotUsageDrawdown{}, nil
	}
	db := DB.Model(&UsageLedger{}).
		Where("supplier_id = ?", lot.SupplierId).
		Where("status = ?", "success").
		Where("created_at >= ? AND created_at <= ?", lot.PeriodStart, lot.PeriodEnd)
	if lot.ChannelId > 0 {
		db = db.Where("channel_id = ?", lot.ChannelId)
	}
	if strings.TrimSpace(lot.SupplyNode) != "" {
		db = db.Where("supply_node = ?", strings.TrimSpace(lot.SupplyNode))
	}
	if strings.TrimSpace(lot.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(lot.ModelName))
	}

	var row supplyPrepaidLotUsageDrawdown
	err := db.Select("COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS used_tokens, COUNT(*) AS request_count").Scan(&row).Error
	return row, err
}

func buildSupplyPrepaidLotDrawdownSourceRef(lot SupplyPrepaidLot) string {
	return fmt.Sprintf("usage_ledger:prepaid_lot:%d:%d:%d", lot.Id, lot.PeriodStart, lot.PeriodEnd)
}
