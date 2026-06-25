package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	SupplierTypeThirdParty   = "third_party"
	SupplierTypeSelfOperated = "self_operated"
	SupplierTypeSelfHosted   = "self_hosted"
)

type Supplier struct {
	Id          int            `json:"id"`
	Name        string         `json:"name" gorm:"size:128;not null;uniqueIndex:uk_supplier_name_deleted_at,priority:1"`
	Type        string         `json:"type" gorm:"size:32;not null;default:'third_party';index"`
	Status      int            `json:"status" gorm:"default:1;index"`
	Notes       string         `json:"notes,omitempty" gorm:"type:text"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;uniqueIndex:uk_supplier_name_deleted_at,priority:2"`
}

func normalizeSupplierType(value string) string {
	switch strings.TrimSpace(value) {
	case SupplierTypeSelfOperated:
		return SupplierTypeSelfOperated
	case SupplierTypeSelfHosted:
		return SupplierTypeSelfHosted
	default:
		return SupplierTypeThirdParty
	}
}

func (s *Supplier) normalize() {
	s.Name = strings.TrimSpace(s.Name)
	s.Type = normalizeSupplierType(s.Type)
	if s.Status == 0 {
		s.Status = 1
	}
}

func (s *Supplier) Insert() error {
	s.normalize()
	if s.Name == "" {
		return errors.New("supplier name is required")
	}
	now := common.GetTimestamp()
	s.CreatedTime = now
	s.UpdatedTime = now
	err := DB.Create(s).Error
	if err == nil {
		InitChannelCache()
	}
	return err
}

func (s *Supplier) Update() error {
	s.normalize()
	if s.Name == "" {
		return errors.New("supplier name is required")
	}
	s.UpdatedTime = common.GetTimestamp()
	err := DB.Save(s).Error
	if err == nil {
		InitChannelCache()
	}
	return err
}

func (s *Supplier) Delete() error {
	err := DB.Delete(s).Error
	if err == nil {
		InitChannelCache()
	}
	return err
}

func GetSupplierByID(id int) (*Supplier, error) {
	var supplier Supplier
	err := DB.First(&supplier, id).Error
	if err != nil {
		return nil, err
	}
	return &supplier, nil
}

func IsSupplierEnabled(id int) (bool, error) {
	if id <= 0 {
		return true, nil
	}
	var supplier Supplier
	if err := DB.Select("id", "status").First(&supplier, id).Error; err != nil {
		return false, err
	}
	return supplier.Status == common.ChannelStatusEnabled, nil
}

func IsChannelSupplierEnabled(channel *Channel) bool {
	if channel == nil {
		return false
	}
	if channel.SupplierId <= 0 {
		return true
	}
	if common.MemoryCacheEnabled {
		channelSyncLock.RLock()
		statuses := supplierStatusIDM
		if statuses != nil {
			enabled := isSupplierStatusEnabled(channel.SupplierId, statuses)
			channelSyncLock.RUnlock()
			return enabled
		}
		channelSyncLock.RUnlock()
	}
	enabled, err := IsSupplierEnabled(channel.SupplierId)
	return err == nil && enabled
}

func isSupplierStatusEnabled(supplierId int, statuses map[int]int) bool {
	if supplierId <= 0 {
		return true
	}
	status, ok := statuses[supplierId]
	return ok && status == common.ChannelStatusEnabled
}

func IsSupplierNameDuplicated(id int, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	var count int64
	err := DB.Model(&Supplier{}).Where("name = ? AND id <> ?", name, id).Count(&count).Error
	return count > 0, err
}

func SearchSuppliers(keyword string, supplierType string, status int, offset int, limit int) ([]*Supplier, int64, error) {
	db := DB.Model(&Supplier{})
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR notes LIKE ?", like, like)
	}
	if supplierType != "" {
		db = db.Where("type = ?", normalizeSupplierType(supplierType))
	}
	if status != 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var suppliers []*Supplier
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&suppliers).Error; err != nil {
		return nil, 0, err
	}
	return suppliers, total, nil
}
