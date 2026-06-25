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
	SupplierRoutePreferenceStatusActive   = "active"
	SupplierRoutePreferenceStatusDisabled = "disabled"
)

const (
	SupplierRoutePreferenceDowngradeWeightPercent = 25
	SupplierRoutePreferenceBoostWeightPercent     = 150
	SupplierRoutePreferenceBaselineWeightPercent  = 100
	SupplierRoutePreferenceMaxWeightPercent       = 200
)

type SupplierRoutePreference struct {
	Id                            int    `json:"id"`
	SupplierId                    int    `json:"supplier_id" gorm:"not null;uniqueIndex:uk_supplier_route_preference_supplier;index"`
	SourcePostureRecommendationId int    `json:"source_posture_recommendation_id" gorm:"index;default:0"`
	Status                        string `json:"status" gorm:"size:32;not null;default:'active';index"`
	WeightPercent                 int    `json:"weight_percent" gorm:"not null;index"`
	Reason                        string `json:"reason" gorm:"type:text"`
	EffectiveFrom                 int64  `json:"effective_from" gorm:"bigint;default:0;index"`
	EffectiveTo                   int64  `json:"effective_to" gorm:"bigint;default:0;index"`
	ActivatedAt                   int64  `json:"activated_at" gorm:"bigint;default:0;index"`
	ActivatedBy                   int    `json:"activated_by" gorm:"default:0;index"`
	DisabledAt                    int64  `json:"disabled_at" gorm:"bigint;default:0;index"`
	DisabledBy                    int    `json:"disabled_by" gorm:"default:0;index"`
	OperatorNote                  string `json:"operator_note" gorm:"type:text"`
	CreatedAt                     int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt                     int64  `json:"updated_at" gorm:"bigint"`
}

type SupplierRoutePreferenceFilters struct {
	SupplierId                    int
	SourcePostureRecommendationId int
	Status                        string
	StartTime                     int64
	EndTime                       int64
}

type SupplierRoutePreferenceActivateInput struct {
	SupplierId    int    `json:"supplier_id"`
	WeightPercent int    `json:"weight_percent"`
	Reason        string `json:"reason"`
	EffectiveFrom int64  `json:"effective_from"`
	EffectiveTo   int64  `json:"effective_to"`
	OperatorNote  string `json:"operator_note"`
}

type SupplierRoutePreferenceDisableInput struct {
	OperatorNote string `json:"operator_note"`
}

func normalizeSupplierRoutePreferenceStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", SupplierRoutePreferenceStatusActive:
		return SupplierRoutePreferenceStatusActive
	case SupplierRoutePreferenceStatusDisabled:
		return SupplierRoutePreferenceStatusDisabled
	default:
		return ""
	}
}

func normalizeSupplierRoutePreferenceWeightPercent(percent int) int {
	if percent < 0 {
		return 0
	}
	if percent > SupplierRoutePreferenceMaxWeightPercent {
		return SupplierRoutePreferenceMaxWeightPercent
	}
	return percent
}

func validateSupplierRoutePreferenceActivateInput(input SupplierRoutePreferenceActivateInput) error {
	if input.SupplierId <= 0 {
		return errors.New("supplier_id is required")
	}
	if input.WeightPercent < 1 || input.WeightPercent > SupplierRoutePreferenceMaxWeightPercent {
		return fmt.Errorf("weight_percent must be between 1 and %d", SupplierRoutePreferenceMaxWeightPercent)
	}
	if strings.TrimSpace(input.Reason) == "" {
		return errors.New("reason is required")
	}
	if input.EffectiveFrom < 0 || input.EffectiveTo < 0 {
		return errors.New("effective window cannot be negative")
	}
	if input.EffectiveTo > 0 && input.EffectiveFrom > 0 && input.EffectiveTo <= input.EffectiveFrom {
		return errors.New("effective_to must be greater than effective_from")
	}
	return nil
}

func SearchSupplierRoutePreferences(filters SupplierRoutePreferenceFilters, offset int, limit int) ([]*SupplierRoutePreference, int64, error) {
	db := DB.Model(&SupplierRoutePreference{})
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if filters.SourcePostureRecommendationId > 0 {
		db = db.Where("source_posture_recommendation_id = ?", filters.SourcePostureRecommendationId)
	}
	if status := normalizeSupplierRoutePreferenceStatus(filters.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if filters.StartTime > 0 {
		db = db.Where("effective_to = 0 OR effective_to >= ?", filters.StartTime)
	}
	if filters.EndTime > 0 {
		db = db.Where("effective_from = 0 OR effective_from <= ?", filters.EndTime)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var preferences []*SupplierRoutePreference
	err := db.Offset(offset).Limit(limit).Order("status ASC, activated_at DESC, id DESC").Find(&preferences).Error
	return preferences, total, err
}

func GetActiveSupplierRoutePreferenceBySupplierID(supplierId int) (*SupplierRoutePreference, error) {
	if supplierId <= 0 {
		return nil, errors.New("supplier_id is required")
	}
	now := common.GetTimestamp()
	var preference SupplierRoutePreference
	err := DB.Where("supplier_id = ?", supplierId).
		Where("status = ?", SupplierRoutePreferenceStatusActive).
		Where("(effective_from = 0 OR effective_from <= ?)", now).
		Where("(effective_to = 0 OR effective_to >= ?)", now).
		First(&preference).Error
	if err != nil {
		return nil, err
	}
	return &preference, nil
}

func ActivateSupplierRoutePreference(input SupplierRoutePreferenceActivateInput, activatedBy int) (*SupplierRoutePreference, error) {
	if err := validateSupplierRoutePreferenceActivateInput(input); err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	if input.EffectiveFrom == 0 {
		input.EffectiveFrom = now
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var supplier Supplier
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&supplier, "id = ?", input.SupplierId).Error; err != nil {
			return err
		}
		if !isSupplierStatusEnabled(supplier.Id, map[int]int{supplier.Id: supplier.Status}) {
			return errors.New("supplier must be enabled before route preference can be activated")
		}
		preference := SupplierRoutePreference{
			SupplierId:                    input.SupplierId,
			SourcePostureRecommendationId: 0,
			Status:                        SupplierRoutePreferenceStatusActive,
			WeightPercent:                 input.WeightPercent,
			Reason:                        strings.TrimSpace(input.Reason),
			EffectiveFrom:                 input.EffectiveFrom,
			EffectiveTo:                   input.EffectiveTo,
			ActivatedAt:                   now,
			ActivatedBy:                   activatedBy,
			DisabledAt:                    0,
			DisabledBy:                    0,
			OperatorNote:                  strings.TrimSpace(input.OperatorNote),
			CreatedAt:                     now,
			UpdatedAt:                     now,
		}
		var existing SupplierRoutePreference
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("supplier_id = ?", input.SupplierId).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&preference).Error
		}
		if err != nil {
			return err
		}
		preference.Id = existing.Id
		if existing.CreatedAt > 0 {
			preference.CreatedAt = existing.CreatedAt
		}
		return tx.Save(&preference).Error
	})
	if err != nil {
		return nil, err
	}
	InitChannelCache()
	return GetActiveSupplierRoutePreferenceBySupplierID(input.SupplierId)
}

func DisableSupplierRoutePreference(supplierId int, disabledBy int, operatorNote string) (*SupplierRoutePreference, error) {
	if supplierId <= 0 {
		return nil, errors.New("supplier_id is required")
	}
	now := common.GetTimestamp()
	var preference SupplierRoutePreference
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("supplier_id = ?", supplierId).
			Where("status = ?", SupplierRoutePreferenceStatusActive).
			First(&preference).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"status":         SupplierRoutePreferenceStatusDisabled,
			"weight_percent": SupplierRoutePreferenceBaselineWeightPercent,
			"effective_to":   now,
			"disabled_at":    now,
			"disabled_by":    disabledBy,
			"operator_note":  strings.TrimSpace(operatorNote),
			"updated_at":     now,
		}
		return tx.Model(&SupplierRoutePreference{}).Where("id = ?", preference.Id).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	InitChannelCache()
	var disabled SupplierRoutePreference
	if err := DB.First(&disabled, "id = ?", preference.Id).Error; err != nil {
		return nil, err
	}
	return &disabled, nil
}

func loadActiveSupplierRoutePreferencePercents(supplierIDs []int, now int64) (map[int]int, error) {
	percents := make(map[int]int)
	if now <= 0 {
		now = common.GetTimestamp()
	}
	db := DB.Model(&SupplierRoutePreference{}).
		Select("supplier_id", "weight_percent").
		Where("status = ?", SupplierRoutePreferenceStatusActive).
		Where("(effective_from = 0 OR effective_from <= ?)", now).
		Where("(effective_to = 0 OR effective_to >= ?)", now)
	if len(supplierIDs) > 0 {
		db = db.Where("supplier_id IN ?", supplierIDs)
	}
	var preferences []SupplierRoutePreference
	if err := db.Find(&preferences).Error; err != nil {
		return percents, err
	}
	for _, preference := range preferences {
		percents[preference.SupplierId] = normalizeSupplierRoutePreferenceWeightPercent(preference.WeightPercent)
	}
	return percents, nil
}

func supplierRoutePreferenceSelectionWeight(baseWeight int, supplierId int, allBaseWeightsZero bool, percents map[int]int) int {
	if allBaseWeightsZero {
		baseWeight = SupplierRoutePreferenceBaselineWeightPercent
	}
	if baseWeight <= 0 {
		return 0
	}
	percent := SupplierRoutePreferenceBaselineWeightPercent
	if supplierId > 0 && percents != nil {
		if value, ok := percents[supplierId]; ok {
			percent = normalizeSupplierRoutePreferenceWeightPercent(value)
		}
	}
	if percent <= 0 {
		return 0
	}
	weight := baseWeight * percent / SupplierRoutePreferenceBaselineWeightPercent
	if weight == 0 {
		return 1
	}
	return weight
}

func applySupplierRoutePreferenceForPostureTx(tx *gorm.DB, recommendation SupplierPostureRecommendation, appliedBy int, operatorNote string, now int64) error {
	switch recommendation.RecommendedAction {
	case SupplierPostureRecommendationActionDowngrade:
		return upsertSupplierRoutePreferenceTx(tx, recommendation, SupplierRoutePreferenceDowngradeWeightPercent, appliedBy, operatorNote, now)
	case SupplierPostureRecommendationActionBoost:
		return upsertSupplierRoutePreferenceTx(tx, recommendation, SupplierRoutePreferenceBoostWeightPercent, appliedBy, operatorNote, now)
	case SupplierPostureRecommendationActionObserve, SupplierPostureRecommendationActionDisable:
		return disableSupplierRoutePreferenceTx(tx, recommendation.SupplierId, recommendation.Id, appliedBy, operatorNote, now)
	default:
		return errors.New("invalid supplier posture recommendation action")
	}
}

func upsertSupplierRoutePreferenceTx(tx *gorm.DB, recommendation SupplierPostureRecommendation, weightPercent int, appliedBy int, operatorNote string, now int64) error {
	reason := fmt.Sprintf(
		"supplier_posture_recommendation #%d %s: grade=%s score=%.3f",
		recommendation.Id,
		recommendation.RecommendedAction,
		recommendation.Grade,
		recommendation.Score,
	)
	preference := SupplierRoutePreference{
		SupplierId:                    recommendation.SupplierId,
		SourcePostureRecommendationId: recommendation.Id,
		Status:                        SupplierRoutePreferenceStatusActive,
		WeightPercent:                 weightPercent,
		Reason:                        reason,
		EffectiveFrom:                 now,
		EffectiveTo:                   0,
		ActivatedAt:                   now,
		ActivatedBy:                   appliedBy,
		DisabledAt:                    0,
		DisabledBy:                    0,
		OperatorNote:                  strings.TrimSpace(operatorNote),
		CreatedAt:                     now,
		UpdatedAt:                     now,
	}
	var existing SupplierRoutePreference
	err := tx.Where("supplier_id = ?", recommendation.SupplierId).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&preference).Error
	}
	if err != nil {
		return err
	}
	preference.Id = existing.Id
	if existing.CreatedAt > 0 {
		preference.CreatedAt = existing.CreatedAt
	}
	return tx.Save(&preference).Error
}

func disableSupplierRoutePreferenceTx(tx *gorm.DB, supplierId int, sourcePostureRecommendationId int, disabledBy int, operatorNote string, now int64) error {
	if supplierId <= 0 {
		return nil
	}
	updates := map[string]any{
		"source_posture_recommendation_id": sourcePostureRecommendationId,
		"status":                           SupplierRoutePreferenceStatusDisabled,
		"weight_percent":                   SupplierRoutePreferenceBaselineWeightPercent,
		"effective_to":                     now,
		"disabled_at":                      now,
		"disabled_by":                      disabledBy,
		"operator_note":                    strings.TrimSpace(operatorNote),
		"updated_at":                       now,
	}
	return tx.Model(&SupplierRoutePreference{}).
		Where("supplier_id = ?", supplierId).
		Where("status = ?", SupplierRoutePreferenceStatusActive).
		Updates(updates).Error
}
