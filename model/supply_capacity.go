package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SupplyCapacityTelemetrySourceNodeReport = "node_report"
	SupplyCapacityTelemetrySourceExternal   = "external"
	SupplyCapacityTelemetrySourceManual     = "manual"

	SupplyCapacityTelemetryCollectPath = "/token-router/telemetry/capacity"
)

type SupplyCapacity struct {
	Id                  int            `json:"id"`
	SupplierId          int            `json:"supplier_id" gorm:"not null;index:idx_supply_capacity_lookup,priority:1"`
	SupplyNode          string         `json:"supply_node" gorm:"size:128;default:'';index:idx_supply_capacity_lookup,priority:2"`
	ModelName           string         `json:"model_name" gorm:"size:128;default:'';index:idx_supply_capacity_lookup,priority:3"`
	PeriodStart         int64          `json:"period_start" gorm:"bigint;not null;index:idx_supply_capacity_period,priority:1"`
	PeriodEnd           int64          `json:"period_end" gorm:"bigint;not null;index:idx_supply_capacity_period,priority:2"`
	CapacityTokens      int64          `json:"capacity_tokens" gorm:"default:0"`
	UsedTokens          int64          `json:"used_tokens" gorm:"default:0"`
	HeadroomTokens      int64          `json:"headroom_tokens" gorm:"default:0"`
	UtilizationRate     float64        `json:"utilization_rate" gorm:"default:0"`
	GpuUtilizationRate  float64        `json:"gpu_utilization_rate" gorm:"default:0"`
	QualityScore        float64        `json:"quality_score" gorm:"default:0"`
	UnitCostQuota       float64        `json:"unit_cost_quota" gorm:"default:0"`
	TelemetrySourceType string         `json:"telemetry_source_type" gorm:"size:64;default:'';index"`
	TelemetrySourceRef  string         `json:"telemetry_source_ref" gorm:"size:256;default:'';index"`
	TelemetryObservedAt int64          `json:"telemetry_observed_at" gorm:"bigint;default:0;index"`
	LastTelemetryId     int            `json:"last_telemetry_id" gorm:"default:0;index"`
	Status              int            `json:"status" gorm:"default:1;index"`
	Notes               string         `json:"notes,omitempty" gorm:"type:text"`
	CreatedTime         int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime         int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

type SupplyCapacityTelemetry struct {
	Id                 int     `json:"id"`
	TelemetryKey       string  `json:"telemetry_key" gorm:"size:768;not null;uniqueIndex:uk_supply_capacity_telemetry_key"`
	SupplierId         int     `json:"supplier_id" gorm:"not null;index"`
	SupplyNode         string  `json:"supply_node" gorm:"size:128;default:'';index"`
	ModelName          string  `json:"model_name" gorm:"size:128;default:'';index"`
	PeriodStart        int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd          int64   `json:"period_end" gorm:"bigint;not null;index"`
	CapacityTokens     int64   `json:"capacity_tokens" gorm:"default:0"`
	UsedTokens         int64   `json:"used_tokens" gorm:"default:0"`
	HeadroomTokens     int64   `json:"headroom_tokens" gorm:"default:0"`
	UtilizationRate    float64 `json:"utilization_rate" gorm:"default:0"`
	GpuUtilizationRate float64 `json:"gpu_utilization_rate" gorm:"default:0"`
	QualityScore       float64 `json:"quality_score" gorm:"default:0"`
	UnitCostQuota      float64 `json:"unit_cost_quota" gorm:"default:0"`
	SourceType         string  `json:"source_type" gorm:"size:64;not null;default:'node_report';index"`
	SourceRef          string  `json:"source_ref" gorm:"size:256;not null;index"`
	ObservedAt         int64   `json:"observed_at" gorm:"bigint;not null;index"`
	AppliedCapacityId  int     `json:"applied_capacity_id" gorm:"default:0;index"`
	RecordedBy         int     `json:"recorded_by" gorm:"default:0;index"`
	Notes              string  `json:"notes,omitempty" gorm:"type:text"`
	CreatedAt          int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt          int64   `json:"updated_at" gorm:"bigint"`
}

type SupplyCapacityFilters struct {
	SupplierId int
	SupplyNode string
	ModelName  string
	Status     int
	StartTime  int64
	EndTime    int64
}

type SupplyCapacityTelemetryRecordInput struct {
	TelemetryKey       string  `json:"telemetry_key"`
	SupplierId         int     `json:"supplier_id"`
	SupplyNode         string  `json:"supply_node"`
	ModelName          string  `json:"model_name"`
	PeriodStart        int64   `json:"period_start"`
	PeriodEnd          int64   `json:"period_end"`
	CapacityTokens     int64   `json:"capacity_tokens"`
	UsedTokens         int64   `json:"used_tokens"`
	GpuUtilizationRate float64 `json:"gpu_utilization_rate"`
	QualityScore       float64 `json:"quality_score"`
	UnitCostQuota      float64 `json:"unit_cost_quota"`
	SourceType         string  `json:"source_type"`
	SourceRef          string  `json:"source_ref"`
	ObservedAt         int64   `json:"observed_at"`
	Notes              string  `json:"notes"`
}

type SupplyCapacityTelemetryCollectInput struct {
	ChannelId   int    `json:"channel_id"`
	SupplierId  int    `json:"supplier_id"`
	SupplyNode  string `json:"supply_node"`
	ModelName   string `json:"model_name"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
}

type SupplyCapacityTelemetrySweepInput struct {
	ChannelId   int    `json:"channel_id"`
	SupplierId  int    `json:"supplier_id"`
	SupplyNode  string `json:"supply_node"`
	ModelName   string `json:"model_name"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
}

type SupplyCapacityTelemetrySweepSkip struct {
	CapacityId  int    `json:"capacity_id"`
	SupplierId  int    `json:"supplier_id"`
	SupplyNode  string `json:"supply_node"`
	ModelName   string `json:"model_name"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	Reason      string `json:"reason"`
}

type SupplyCapacityTelemetrySweepResult struct {
	AttemptedCount int                                `json:"attempted_count"`
	CollectedCount int                                `json:"collected_count"`
	SkippedCount   int                                `json:"skipped_count"`
	Collected      []*SupplyCapacityTelemetry         `json:"collected"`
	Skipped        []SupplyCapacityTelemetrySweepSkip `json:"skipped"`
}

type supplyCapacityTelemetryUpstreamResponse struct {
	SupplyNode         string  `json:"supply_node"`
	ModelName          string  `json:"model_name"`
	CapacityTokens     int64   `json:"capacity_tokens"`
	UsedTokens         int64   `json:"used_tokens"`
	GpuUtilizationRate float64 `json:"gpu_utilization_rate"`
	QualityScore       float64 `json:"quality_score"`
	UnitCostQuota      float64 `json:"unit_cost_quota"`
	ObservedAt         int64   `json:"observed_at"`
	SourceRef          string  `json:"source_ref"`
	Notes              string  `json:"notes"`
}

type SupplyCapacityTelemetryFilters struct {
	SupplierId        int
	SupplyNode        string
	ModelName         string
	SourceType        string
	AppliedCapacityId int
	StartTime         int64
	EndTime           int64
}

type SupplyCapacityUsageRefreshInput struct {
	CapacityId  int    `json:"capacity_id"`
	SupplierId  int    `json:"supplier_id"`
	SupplyNode  string `json:"supply_node"`
	ModelName   string `json:"model_name"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
}

func (c *SupplyCapacity) normalize() {
	c.SupplyNode = strings.TrimSpace(c.SupplyNode)
	c.ModelName = strings.TrimSpace(c.ModelName)
	c.TelemetrySourceType = strings.TrimSpace(c.TelemetrySourceType)
	c.TelemetrySourceRef = strings.TrimSpace(c.TelemetrySourceRef)
	if c.Status == 0 {
		c.Status = 1
	}
	c.HeadroomTokens = c.CapacityTokens - c.UsedTokens
	if c.CapacityTokens > 0 {
		c.UtilizationRate = float64(c.UsedTokens) / float64(c.CapacityTokens)
	} else {
		c.UtilizationRate = 0
	}
}

func (c *SupplyCapacity) validate() error {
	if c.SupplierId <= 0 {
		return errors.New("supplier_id is required")
	}
	if c.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if c.PeriodEnd <= c.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	if c.CapacityTokens < 0 {
		return errors.New("capacity_tokens cannot be negative")
	}
	if c.UsedTokens < 0 {
		return errors.New("used_tokens cannot be negative")
	}
	if c.QualityScore < 0 || c.QualityScore > 100 {
		return errors.New("quality_score must be between 0 and 100")
	}
	if c.GpuUtilizationRate < 0 || c.GpuUtilizationRate > 1 {
		return errors.New("gpu_utilization_rate must be between 0 and 1")
	}
	if c.UnitCostQuota < 0 {
		return errors.New("unit_cost_quota cannot be negative")
	}
	return nil
}

func (c *SupplyCapacity) Insert() error {
	c.normalize()
	if err := c.validate(); err != nil {
		return err
	}
	now := common.GetTimestamp()
	c.CreatedTime = now
	c.UpdatedTime = now
	return DB.Create(c).Error
}

func (c *SupplyCapacity) Update() error {
	c.normalize()
	if err := c.validate(); err != nil {
		return err
	}
	c.UpdatedTime = common.GetTimestamp()
	return DB.Save(c).Error
}

func DeleteSupplyCapacityByID(id int) error {
	return DB.Delete(&SupplyCapacity{}, id).Error
}

func GetSupplyCapacityByID(id int) (*SupplyCapacity, error) {
	var capacity SupplyCapacity
	err := DB.First(&capacity, id).Error
	if err != nil {
		return nil, err
	}
	return &capacity, nil
}

func SearchSupplyCapacities(filters SupplyCapacityFilters, offset int, limit int) ([]*SupplyCapacity, int64, error) {
	db := DB.Model(&SupplyCapacity{})
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if strings.TrimSpace(filters.SupplyNode) != "" {
		db = db.Where("supply_node = ?", strings.TrimSpace(filters.SupplyNode))
	}
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if filters.Status != 0 {
		db = db.Where("status = ?", filters.Status)
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
	var capacities []*SupplyCapacity
	if err := db.Offset(offset).Limit(limit).Order("period_start DESC, id DESC").Find(&capacities).Error; err != nil {
		return nil, 0, err
	}
	return capacities, total, nil
}

func RefreshSupplyCapacityUsage(input SupplyCapacityUsageRefreshInput) ([]*SupplyCapacity, error) {
	capacities, err := findSupplyCapacitiesForUsageRefresh(input)
	if err != nil {
		return nil, err
	}
	if len(capacities) == 0 {
		return []*SupplyCapacity{}, nil
	}

	now := common.GetTimestamp()
	updated := make([]*SupplyCapacity, 0, len(capacities))
	for i := range capacities {
		capacity := capacities[i]
		usedTokens, err := usageTokensForSupplyCapacity(capacity)
		if err != nil {
			return nil, err
		}
		capacity.UsedTokens = usedTokens
		capacity.normalize()
		capacity.UpdatedTime = now
		if err := DB.Model(&SupplyCapacity{}).
			Where("id = ?", capacity.Id).
			Updates(map[string]any{
				"used_tokens":      capacity.UsedTokens,
				"headroom_tokens":  capacity.HeadroomTokens,
				"utilization_rate": capacity.UtilizationRate,
				"updated_time":     capacity.UpdatedTime,
			}).Error; err != nil {
			return nil, err
		}
		saved, err := GetSupplyCapacityByID(capacity.Id)
		if err != nil {
			return nil, err
		}
		updated = append(updated, saved)
	}
	return updated, nil
}

func findSupplyCapacitiesForUsageRefresh(input SupplyCapacityUsageRefreshInput) ([]SupplyCapacity, error) {
	db := DB.Model(&SupplyCapacity{})
	if input.CapacityId > 0 {
		db = db.Where("id = ?", input.CapacityId)
	}
	if input.SupplierId > 0 {
		db = db.Where("supplier_id = ?", input.SupplierId)
	}
	if strings.TrimSpace(input.SupplyNode) != "" {
		db = db.Where("supply_node = ?", strings.TrimSpace(input.SupplyNode))
	}
	if strings.TrimSpace(input.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if input.PeriodStart > 0 {
		db = db.Where("period_end >= ?", input.PeriodStart)
	}
	if input.PeriodEnd > 0 {
		db = db.Where("period_start <= ?", input.PeriodEnd)
	}
	var capacities []SupplyCapacity
	err := db.Order("period_start DESC, id DESC").Find(&capacities).Error
	return capacities, err
}

func usageTokensForSupplyCapacity(capacity SupplyCapacity) (int64, error) {
	db := DB.Model(&UsageLedger{}).
		Where("supplier_id = ?", capacity.SupplierId).
		Where("status = ?", "success").
		Where("created_at >= ? AND created_at <= ?", capacity.PeriodStart, capacity.PeriodEnd)
	if strings.TrimSpace(capacity.SupplyNode) != "" {
		db = db.Where("supply_node = ?", strings.TrimSpace(capacity.SupplyNode))
	}
	if strings.TrimSpace(capacity.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(capacity.ModelName))
	}
	var row struct {
		UsedTokens int64
	}
	err := db.Select("COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS used_tokens").Scan(&row).Error
	return row.UsedTokens, err
}

func normalizeSupplyCapacityTelemetrySource(value string) string {
	switch strings.TrimSpace(value) {
	case SupplyCapacityTelemetrySourceExternal:
		return SupplyCapacityTelemetrySourceExternal
	case SupplyCapacityTelemetrySourceManual:
		return SupplyCapacityTelemetrySourceManual
	default:
		return SupplyCapacityTelemetrySourceNodeReport
	}
}

func (t *SupplyCapacityTelemetry) normalize() {
	t.TelemetryKey = strings.TrimSpace(t.TelemetryKey)
	t.SupplyNode = strings.TrimSpace(t.SupplyNode)
	t.ModelName = strings.TrimSpace(t.ModelName)
	t.SourceType = normalizeSupplyCapacityTelemetrySource(t.SourceType)
	t.SourceRef = strings.TrimSpace(t.SourceRef)
	t.Notes = strings.TrimSpace(t.Notes)
	t.HeadroomTokens = t.CapacityTokens - t.UsedTokens
	if t.CapacityTokens > 0 {
		t.UtilizationRate = float64(t.UsedTokens) / float64(t.CapacityTokens)
	} else {
		t.UtilizationRate = 0
	}
	if t.TelemetryKey == "" && t.SourceRef != "" {
		t.TelemetryKey = fmt.Sprintf("supply_capacity_telemetry:%s:%s:%d:%s:%s:%d:%d",
			t.SourceType,
			t.SourceRef,
			t.SupplierId,
			t.SupplyNode,
			t.ModelName,
			t.PeriodStart,
			t.PeriodEnd,
		)
	}
}

func (t *SupplyCapacityTelemetry) validate() error {
	if t.TelemetryKey == "" {
		return errors.New("telemetry_key is required")
	}
	if t.SupplierId <= 0 {
		return errors.New("supplier_id is required")
	}
	if t.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if t.PeriodEnd <= t.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	if t.CapacityTokens < 0 {
		return errors.New("capacity_tokens cannot be negative")
	}
	if t.UsedTokens < 0 {
		return errors.New("used_tokens cannot be negative")
	}
	if t.GpuUtilizationRate < 0 || t.GpuUtilizationRate > 1 {
		return errors.New("gpu_utilization_rate must be between 0 and 1")
	}
	if t.QualityScore < 0 || t.QualityScore > 100 {
		return errors.New("quality_score must be between 0 and 100")
	}
	if t.UnitCostQuota < 0 {
		return errors.New("unit_cost_quota cannot be negative")
	}
	if t.SourceRef == "" {
		return errors.New("source_ref is required")
	}
	if t.ObservedAt <= 0 {
		return errors.New("observed_at is required")
	}
	return nil
}

func RecordSupplyCapacityTelemetry(input SupplyCapacityTelemetryRecordInput, recordedBy int) (*SupplyCapacityTelemetry, error) {
	telemetry := SupplyCapacityTelemetry{
		TelemetryKey:       input.TelemetryKey,
		SupplierId:         input.SupplierId,
		SupplyNode:         input.SupplyNode,
		ModelName:          input.ModelName,
		PeriodStart:        input.PeriodStart,
		PeriodEnd:          input.PeriodEnd,
		CapacityTokens:     input.CapacityTokens,
		UsedTokens:         input.UsedTokens,
		GpuUtilizationRate: input.GpuUtilizationRate,
		QualityScore:       input.QualityScore,
		UnitCostQuota:      input.UnitCostQuota,
		SourceType:         input.SourceType,
		SourceRef:          input.SourceRef,
		ObservedAt:         input.ObservedAt,
		RecordedBy:         recordedBy,
		Notes:              input.Notes,
	}
	telemetry.normalize()
	if err := telemetry.validate(); err != nil {
		return nil, err
	}
	if _, err := GetSupplierByID(telemetry.SupplierId); err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	telemetry.CreatedAt = now
	telemetry.UpdatedAt = now

	var result SupplyCapacityTelemetry
	err := DB.Transaction(func(tx *gorm.DB) error {
		capacityID, err := upsertSupplyCapacityFromTelemetry(tx, telemetry, now)
		if err != nil {
			return err
		}
		telemetry.AppliedCapacityId = capacityID
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "telemetry_key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"supplier_id",
				"supply_node",
				"model_name",
				"period_start",
				"period_end",
				"capacity_tokens",
				"used_tokens",
				"headroom_tokens",
				"utilization_rate",
				"gpu_utilization_rate",
				"quality_score",
				"unit_cost_quota",
				"source_type",
				"source_ref",
				"observed_at",
				"applied_capacity_id",
				"recorded_by",
				"notes",
				"updated_at",
			}),
		}).Create(&telemetry).Error; err != nil {
			return err
		}
		if err := tx.Where("telemetry_key = ?", telemetry.TelemetryKey).First(&result).Error; err != nil {
			return err
		}
		return tx.Model(&SupplyCapacity{}).Where("id = ?", capacityID).Updates(map[string]any{
			"last_telemetry_id": result.Id,
			"updated_time":      now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func CollectSupplyCapacityTelemetry(input SupplyCapacityTelemetryCollectInput, recordedBy int) (*SupplyCapacityTelemetry, error) {
	if input.ChannelId <= 0 {
		return nil, errors.New("channel_id is required")
	}
	if input.PeriodStart <= 0 {
		return nil, errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return nil, errors.New("period_end must be greater than period_start")
	}
	supplyNode := strings.TrimSpace(input.SupplyNode)
	if supplyNode == "" {
		return nil, errors.New("supply_node is required")
	}
	modelName := strings.TrimSpace(input.ModelName)
	if modelName == "" {
		return nil, errors.New("model_name is required")
	}

	var channel Channel
	if err := DB.Where("id = ?", input.ChannelId).First(&channel).Error; err != nil {
		return nil, err
	}
	supplierID := channel.SupplierId
	if input.SupplierId > 0 {
		if supplierID > 0 && supplierID != input.SupplierId {
			return nil, errors.New("supplier_id does not match channel supplier_id")
		}
		supplierID = input.SupplierId
	}
	if supplierID <= 0 {
		return nil, errors.New("supplier_id is required")
	}
	if _, err := GetSupplierByID(supplierID); err != nil {
		return nil, err
	}

	upstream, err := collectSupplyCapacityTelemetryFromChannel(channel, supplyNode, modelName, input.PeriodStart, input.PeriodEnd)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(upstream.SupplyNode) != "" {
		supplyNode = strings.TrimSpace(upstream.SupplyNode)
	}
	if strings.TrimSpace(upstream.ModelName) != "" {
		modelName = strings.TrimSpace(upstream.ModelName)
	}
	observedAt := upstream.ObservedAt
	if observedAt <= 0 {
		observedAt = common.GetTimestamp()
	}
	sourceRef := strings.TrimSpace(upstream.SourceRef)
	if sourceRef == "" {
		sourceRef = fmt.Sprintf("channel:%d:%s:%d-%d", channel.Id, SupplyCapacityTelemetryCollectPath, input.PeriodStart, input.PeriodEnd)
	}

	return RecordSupplyCapacityTelemetry(SupplyCapacityTelemetryRecordInput{
		SupplierId:         supplierID,
		SupplyNode:         supplyNode,
		ModelName:          modelName,
		PeriodStart:        input.PeriodStart,
		PeriodEnd:          input.PeriodEnd,
		CapacityTokens:     upstream.CapacityTokens,
		UsedTokens:         upstream.UsedTokens,
		GpuUtilizationRate: upstream.GpuUtilizationRate,
		QualityScore:       upstream.QualityScore,
		UnitCostQuota:      upstream.UnitCostQuota,
		SourceType:         SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          sourceRef,
		ObservedAt:         observedAt,
		Notes:              upstream.Notes,
	}, recordedBy)
}

func SweepSupplyCapacityTelemetry(input SupplyCapacityTelemetrySweepInput, recordedBy int) (*SupplyCapacityTelemetrySweepResult, error) {
	if input.PeriodEnd > 0 && input.PeriodStart <= 0 {
		return nil, errors.New("period_start is required when period_end is set")
	}
	if input.PeriodStart > 0 && input.PeriodEnd <= input.PeriodStart {
		return nil, errors.New("period_end must be greater than period_start")
	}

	db := DB.Model(&SupplyCapacity{}).Where("status = ?", common.ChannelStatusEnabled)
	if input.SupplierId > 0 {
		db = db.Where("supplier_id = ?", input.SupplierId)
	}
	if strings.TrimSpace(input.SupplyNode) != "" {
		db = db.Where("supply_node = ?", strings.TrimSpace(input.SupplyNode))
	}
	if strings.TrimSpace(input.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if input.PeriodStart > 0 {
		db = db.Where("period_end >= ?", input.PeriodStart)
	}
	if input.PeriodEnd > 0 {
		db = db.Where("period_start <= ?", input.PeriodEnd)
	}

	var capacities []SupplyCapacity
	if err := db.Order("supplier_id ASC, supply_node ASC, model_name ASC, period_start ASC, id ASC").Find(&capacities).Error; err != nil {
		return nil, err
	}

	result := &SupplyCapacityTelemetrySweepResult{
		AttemptedCount: len(capacities),
		Collected:      []*SupplyCapacityTelemetry{},
		Skipped:        []SupplyCapacityTelemetrySweepSkip{},
	}
	for _, capacity := range capacities {
		channelID := input.ChannelId
		if channelID <= 0 {
			channel, err := findSupplyCapacityTelemetrySweepChannel(capacity)
			if err != nil {
				result.Skipped = append(result.Skipped, supplyCapacityTelemetrySweepSkipFromCapacity(capacity, err.Error()))
				continue
			}
			channelID = channel.Id
		}
		telemetry, err := CollectSupplyCapacityTelemetry(SupplyCapacityTelemetryCollectInput{
			ChannelId:   channelID,
			SupplierId:  capacity.SupplierId,
			SupplyNode:  capacity.SupplyNode,
			ModelName:   capacity.ModelName,
			PeriodStart: capacity.PeriodStart,
			PeriodEnd:   capacity.PeriodEnd,
		}, recordedBy)
		if err != nil {
			result.Skipped = append(result.Skipped, supplyCapacityTelemetrySweepSkipFromCapacity(capacity, err.Error()))
			continue
		}
		result.Collected = append(result.Collected, telemetry)
	}
	result.CollectedCount = len(result.Collected)
	result.SkippedCount = len(result.Skipped)
	return result, nil
}

func findSupplyCapacityTelemetrySweepChannel(capacity SupplyCapacity) (*Channel, error) {
	var channels []Channel
	err := DB.
		Where("supplier_id = ? AND status = ?", capacity.SupplierId, common.ChannelStatusEnabled).
		Where("base_url IS NOT NULL AND base_url <> ''").
		Order("id ASC").
		Find(&channels).Error
	if err != nil {
		return nil, err
	}
	for i := range channels {
		if channelSupportsTelemetrySweepModel(channels[i], capacity.ModelName) {
			return &channels[i], nil
		}
	}
	return nil, errors.New("no enabled channel with base_url supports capacity model")
}

func channelSupportsTelemetrySweepModel(channel Channel, modelName string) bool {
	modelName = strings.TrimSpace(modelName)
	for _, candidate := range strings.Split(channel.Models, ",") {
		if strings.TrimSpace(candidate) == modelName {
			return true
		}
	}
	return false
}

func supplyCapacityTelemetrySweepSkipFromCapacity(capacity SupplyCapacity, reason string) SupplyCapacityTelemetrySweepSkip {
	return SupplyCapacityTelemetrySweepSkip{
		CapacityId:  capacity.Id,
		SupplierId:  capacity.SupplierId,
		SupplyNode:  capacity.SupplyNode,
		ModelName:   capacity.ModelName,
		PeriodStart: capacity.PeriodStart,
		PeriodEnd:   capacity.PeriodEnd,
		Reason:      reason,
	}
}

func collectSupplyCapacityTelemetryFromChannel(channel Channel, supplyNode string, modelName string, periodStart int64, periodEnd int64) (supplyCapacityTelemetryUpstreamResponse, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(channel.GetBaseURL()), "/")
	if baseURL == "" {
		return supplyCapacityTelemetryUpstreamResponse{}, errors.New("channel base_url is required")
	}
	endpoint, err := url.Parse(baseURL + SupplyCapacityTelemetryCollectPath)
	if err != nil {
		return supplyCapacityTelemetryUpstreamResponse{}, err
	}
	values := endpoint.Query()
	values.Set("supply_node", supplyNode)
	values.Set("model", modelName)
	values.Set("period_start", strconv.FormatInt(periodStart, 10))
	values.Set("period_end", strconv.FormatInt(periodEnd, 10))
	endpoint.RawQuery = values.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return supplyCapacityTelemetryUpstreamResponse{}, err
	}
	if keys := channel.GetKeys(); len(keys) > 0 {
		key := strings.Trim(strings.TrimSpace(keys[0]), `"`)
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(key, "Bearer "))
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return supplyCapacityTelemetryUpstreamResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return supplyCapacityTelemetryUpstreamResponse{}, fmt.Errorf("upstream telemetry returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var upstream supplyCapacityTelemetryUpstreamResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&upstream); err != nil {
		return supplyCapacityTelemetryUpstreamResponse{}, err
	}
	return upstream, nil
}

func upsertSupplyCapacityFromTelemetry(tx *gorm.DB, telemetry SupplyCapacityTelemetry, now int64) (int, error) {
	var capacity SupplyCapacity
	err := tx.Where(
		"supplier_id = ? AND supply_node = ? AND model_name = ? AND period_start = ? AND period_end = ?",
		telemetry.SupplierId,
		telemetry.SupplyNode,
		telemetry.ModelName,
		telemetry.PeriodStart,
		telemetry.PeriodEnd,
	).First(&capacity).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		capacity = SupplyCapacity{
			SupplierId:  telemetry.SupplierId,
			SupplyNode:  telemetry.SupplyNode,
			ModelName:   telemetry.ModelName,
			PeriodStart: telemetry.PeriodStart,
			PeriodEnd:   telemetry.PeriodEnd,
			Status:      1,
			CreatedTime: now,
		}
	}
	capacity.CapacityTokens = telemetry.CapacityTokens
	capacity.UsedTokens = telemetry.UsedTokens
	capacity.GpuUtilizationRate = telemetry.GpuUtilizationRate
	capacity.QualityScore = telemetry.QualityScore
	capacity.UnitCostQuota = telemetry.UnitCostQuota
	capacity.TelemetrySourceType = telemetry.SourceType
	capacity.TelemetrySourceRef = telemetry.SourceRef
	capacity.TelemetryObservedAt = telemetry.ObservedAt
	if strings.TrimSpace(capacity.Notes) == "" && telemetry.Notes != "" {
		capacity.Notes = telemetry.Notes
	}
	capacity.normalize()
	capacity.UpdatedTime = now
	if capacity.Id == 0 {
		if err := tx.Create(&capacity).Error; err != nil {
			return 0, err
		}
		return capacity.Id, nil
	}
	if err := tx.Model(&SupplyCapacity{}).Where("id = ?", capacity.Id).Updates(map[string]any{
		"capacity_tokens":       capacity.CapacityTokens,
		"used_tokens":           capacity.UsedTokens,
		"headroom_tokens":       capacity.HeadroomTokens,
		"utilization_rate":      capacity.UtilizationRate,
		"gpu_utilization_rate":  capacity.GpuUtilizationRate,
		"quality_score":         capacity.QualityScore,
		"unit_cost_quota":       capacity.UnitCostQuota,
		"telemetry_source_type": capacity.TelemetrySourceType,
		"telemetry_source_ref":  capacity.TelemetrySourceRef,
		"telemetry_observed_at": capacity.TelemetryObservedAt,
		"notes":                 capacity.Notes,
		"updated_time":          capacity.UpdatedTime,
	}).Error; err != nil {
		return 0, err
	}
	return capacity.Id, nil
}

func SearchSupplyCapacityTelemetries(filters SupplyCapacityTelemetryFilters, offset int, limit int) ([]*SupplyCapacityTelemetry, int64, error) {
	db := DB.Model(&SupplyCapacityTelemetry{})
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if strings.TrimSpace(filters.SupplyNode) != "" {
		db = db.Where("supply_node = ?", strings.TrimSpace(filters.SupplyNode))
	}
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if sourceType := normalizeSupplyCapacityTelemetrySource(filters.SourceType); strings.TrimSpace(filters.SourceType) != "" {
		db = db.Where("source_type = ?", sourceType)
	}
	if filters.AppliedCapacityId > 0 {
		db = db.Where("applied_capacity_id = ?", filters.AppliedCapacityId)
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
	var rows []*SupplyCapacityTelemetry
	if err := db.Offset(offset).Limit(limit).Order("observed_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
