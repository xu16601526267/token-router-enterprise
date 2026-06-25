package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm/clause"
)

const (
	SupplyCostProfileSourceManual     = "manual"
	SupplyCostProfileSourceAccounting = "accounting"
	SupplyCostProfileSourceExternal   = "external"
)

type SupplyCostProfile struct {
	Id                     int     `json:"id"`
	CostProfileKey         string  `json:"cost_profile_key" gorm:"size:768;not null;uniqueIndex:uk_supply_cost_profile_key"`
	SupplierId             int     `json:"supplier_id" gorm:"not null;index"`
	SupplyNode             string  `json:"supply_node" gorm:"size:128;default:'';index"`
	ModelName              string  `json:"model_name" gorm:"size:128;default:'';index"`
	PeriodStart            int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd              int64   `json:"period_end" gorm:"bigint;not null;index"`
	CapacityTokens         int64   `json:"capacity_tokens" gorm:"default:0"`
	FixedCostQuota         float64 `json:"fixed_cost_quota" gorm:"default:0"`
	VariableUnitCostQuota  float64 `json:"variable_unit_cost_quota" gorm:"default:0"`
	AmortizedUnitCostQuota float64 `json:"amortized_unit_cost_quota" gorm:"default:0"`
	SourceType             string  `json:"source_type" gorm:"size:64;not null;default:'manual';index"`
	SourceRef              string  `json:"source_ref" gorm:"size:256;not null;index"`
	ObservedAt             int64   `json:"observed_at" gorm:"bigint;not null;index"`
	RecordedBy             int     `json:"recorded_by" gorm:"default:0;index"`
	Notes                  string  `json:"notes,omitempty" gorm:"type:text"`
	CreatedAt              int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt              int64   `json:"updated_at" gorm:"bigint"`
}

type SupplyCostProfileRecordInput struct {
	CostProfileKey        string  `json:"cost_profile_key"`
	SupplierId            int     `json:"supplier_id"`
	SupplyNode            string  `json:"supply_node"`
	ModelName             string  `json:"model_name"`
	PeriodStart           int64   `json:"period_start"`
	PeriodEnd             int64   `json:"period_end"`
	CapacityTokens        int64   `json:"capacity_tokens"`
	FixedCostQuota        float64 `json:"fixed_cost_quota"`
	VariableUnitCostQuota float64 `json:"variable_unit_cost_quota"`
	SourceType            string  `json:"source_type"`
	SourceRef             string  `json:"source_ref"`
	ObservedAt            int64   `json:"observed_at"`
	Notes                 string  `json:"notes"`
}

type SupplyCostProfileFilters struct {
	SupplierId int
	SupplyNode string
	ModelName  string
	SourceType string
	StartTime  int64
	EndTime    int64
}

func normalizeSupplyCostProfileSource(value string) string {
	switch strings.TrimSpace(value) {
	case SupplyCostProfileSourceAccounting:
		return SupplyCostProfileSourceAccounting
	case SupplyCostProfileSourceExternal:
		return SupplyCostProfileSourceExternal
	default:
		return SupplyCostProfileSourceManual
	}
}

func (p *SupplyCostProfile) normalize() {
	p.CostProfileKey = strings.TrimSpace(p.CostProfileKey)
	p.SupplyNode = strings.TrimSpace(p.SupplyNode)
	p.ModelName = strings.TrimSpace(p.ModelName)
	p.SourceType = normalizeSupplyCostProfileSource(p.SourceType)
	p.SourceRef = strings.TrimSpace(p.SourceRef)
	p.Notes = strings.TrimSpace(p.Notes)
	if p.CapacityTokens > 0 {
		p.AmortizedUnitCostQuota = float64(p.FixedCostQuota)/float64(p.CapacityTokens) + p.VariableUnitCostQuota
	} else {
		p.AmortizedUnitCostQuota = p.VariableUnitCostQuota
	}
	if p.CostProfileKey == "" && p.SourceRef != "" {
		p.CostProfileKey = fmt.Sprintf("supply_cost_profile:%s:%s:%d:%s:%s:%d:%d",
			p.SourceType,
			p.SourceRef,
			p.SupplierId,
			p.SupplyNode,
			p.ModelName,
			p.PeriodStart,
			p.PeriodEnd,
		)
	}
}

func (p *SupplyCostProfile) validate() error {
	if p.CostProfileKey == "" {
		return errors.New("cost_profile_key is required")
	}
	if p.SupplierId <= 0 {
		return errors.New("supplier_id is required")
	}
	if p.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if p.PeriodEnd <= p.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	if p.CapacityTokens <= 0 {
		return errors.New("capacity_tokens must be greater than 0")
	}
	if p.FixedCostQuota < 0 {
		return errors.New("fixed_cost_quota cannot be negative")
	}
	if p.VariableUnitCostQuota < 0 {
		return errors.New("variable_unit_cost_quota cannot be negative")
	}
	if p.AmortizedUnitCostQuota < 0 {
		return errors.New("amortized_unit_cost_quota cannot be negative")
	}
	if p.SourceRef == "" {
		return errors.New("source_ref is required")
	}
	if p.ObservedAt <= 0 {
		return errors.New("observed_at is required")
	}
	return nil
}

func RecordSupplyCostProfile(input SupplyCostProfileRecordInput, recordedBy int) (*SupplyCostProfile, error) {
	profile := SupplyCostProfile{
		CostProfileKey:        input.CostProfileKey,
		SupplierId:            input.SupplierId,
		SupplyNode:            input.SupplyNode,
		ModelName:             input.ModelName,
		PeriodStart:           input.PeriodStart,
		PeriodEnd:             input.PeriodEnd,
		CapacityTokens:        input.CapacityTokens,
		FixedCostQuota:        input.FixedCostQuota,
		VariableUnitCostQuota: input.VariableUnitCostQuota,
		SourceType:            input.SourceType,
		SourceRef:             input.SourceRef,
		ObservedAt:            input.ObservedAt,
		RecordedBy:            recordedBy,
		Notes:                 input.Notes,
	}
	profile.normalize()
	if err := profile.validate(); err != nil {
		return nil, err
	}
	supplier, err := GetSupplierByID(profile.SupplierId)
	if err != nil {
		return nil, err
	}
	if supplier.Type != SupplierTypeSelfHosted {
		return nil, errors.New("supplier must be self_hosted")
	}

	now := common.GetTimestamp()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	if err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "cost_profile_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"supplier_id",
			"supply_node",
			"model_name",
			"period_start",
			"period_end",
			"capacity_tokens",
			"fixed_cost_quota",
			"variable_unit_cost_quota",
			"amortized_unit_cost_quota",
			"source_type",
			"source_ref",
			"observed_at",
			"recorded_by",
			"notes",
			"updated_at",
		}),
	}).Create(&profile).Error; err != nil {
		return nil, err
	}

	var result SupplyCostProfile
	err = DB.Where("cost_profile_key = ?", profile.CostProfileKey).First(&result).Error
	return &result, err
}

func SearchSupplyCostProfiles(filters SupplyCostProfileFilters, offset int, limit int) ([]*SupplyCostProfile, int64, error) {
	db := DB.Model(&SupplyCostProfile{})
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if strings.TrimSpace(filters.SupplyNode) != "" {
		db = db.Where("supply_node = ?", strings.TrimSpace(filters.SupplyNode))
	}
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if sourceType := normalizeSupplyCostProfileSource(filters.SourceType); strings.TrimSpace(filters.SourceType) != "" {
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
	var rows []*SupplyCostProfile
	err := db.Offset(offset).Limit(limit).Order("observed_at DESC, id DESC").Find(&rows).Error
	return rows, total, err
}
