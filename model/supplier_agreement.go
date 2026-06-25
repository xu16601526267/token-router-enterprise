package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

type SupplierAgreement struct {
	Id                     int            `json:"id"`
	SupplierId             int            `json:"supplier_id" gorm:"not null;index:idx_supplier_agreement_lookup,priority:1"`
	ModelName              string         `json:"model_name" gorm:"size:128;default:'';index:idx_supplier_agreement_lookup,priority:2"`
	EffectiveFrom          int64          `json:"effective_from" gorm:"bigint;default:0;index:idx_supplier_agreement_lookup,priority:3"`
	EffectiveTo            int64          `json:"effective_to" gorm:"bigint;default:0;index"`
	UsePrice               bool           `json:"use_price" gorm:"default:false"`
	CostModelRatio         float64        `json:"cost_model_ratio" gorm:"default:1"`
	CostCompletionRatio    float64        `json:"cost_completion_ratio" gorm:"default:1"`
	CostCacheRatio         float64        `json:"cost_cache_ratio" gorm:"default:0.1"`
	CostCacheCreationRatio float64        `json:"cost_cache_creation_ratio" gorm:"default:1"`
	CostModelPrice         float64        `json:"cost_model_price" gorm:"default:0"`
	Priority               int            `json:"priority" gorm:"default:0;index"`
	Status                 int            `json:"status" gorm:"default:1;index"`
	Notes                  string         `json:"notes,omitempty" gorm:"type:text"`
	CreatedTime            int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime            int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt              gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *SupplierAgreement) normalize() {
	a.ModelName = strings.TrimSpace(a.ModelName)
	if a.Status == 0 {
		a.Status = 1
	}
	if a.CostModelRatio == 0 {
		a.CostModelRatio = 1
	}
	if a.CostCompletionRatio == 0 {
		a.CostCompletionRatio = 1
	}
	if a.CostCacheRatio == 0 {
		a.CostCacheRatio = 0.1
	}
	if a.CostCacheCreationRatio == 0 {
		a.CostCacheCreationRatio = 1
	}
}

func (a *SupplierAgreement) Insert() error {
	a.normalize()
	now := common.GetTimestamp()
	a.CreatedTime = now
	a.UpdatedTime = now
	return DB.Create(a).Error
}

func (a *SupplierAgreement) Update() error {
	a.normalize()
	a.UpdatedTime = common.GetTimestamp()
	return DB.Save(a).Error
}

func DeleteSupplierAgreementByID(id int) error {
	return DB.Delete(&SupplierAgreement{}, id).Error
}

func GetSupplierAgreementByID(id int) (*SupplierAgreement, error) {
	var agreement SupplierAgreement
	err := DB.First(&agreement, id).Error
	if err != nil {
		return nil, err
	}
	return &agreement, nil
}

func SearchSupplierAgreements(supplierId int, modelName string, status int, offset int, limit int) ([]*SupplierAgreement, int64, error) {
	db := DB.Model(&SupplierAgreement{})
	if supplierId > 0 {
		db = db.Where("supplier_id = ?", supplierId)
	}
	modelName = strings.TrimSpace(modelName)
	if modelName != "" {
		db = db.Where("model_name = ?", modelName)
	}
	if status != 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var agreements []*SupplierAgreement
	if err := db.Offset(offset).Limit(limit).Order("priority DESC, effective_from DESC, id DESC").Find(&agreements).Error; err != nil {
		return nil, 0, err
	}
	return agreements, total, nil
}

func FindActiveSupplierAgreement(supplierId int, modelName string, at int64) (*SupplierAgreement, error) {
	var agreement SupplierAgreement
	err := DB.Where("supplier_id = ? AND status = ? AND effective_from <= ? AND (effective_to = 0 OR effective_to >= ?) AND (model_name = ? OR model_name = '')",
		supplierId, 1, at, at, strings.TrimSpace(modelName)).
		Order("model_name DESC, priority DESC, effective_from DESC, id DESC").
		First(&agreement).Error
	if err != nil {
		return nil, err
	}
	return &agreement, nil
}
