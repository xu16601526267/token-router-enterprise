package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	UsageRecordOutboxStatusPending    = "pending"
	UsageRecordOutboxStatusProcessing = "processing"
	UsageRecordOutboxStatusSucceeded  = "succeeded"
	UsageRecordOutboxStatusFailed     = "failed"
)

type UsageRecordOutbox struct {
	Id          int    `json:"id"`
	RequestId   string `json:"request_id" gorm:"size:128;not null;uniqueIndex"`
	Payload     string `json:"payload" gorm:"type:text;not null"`
	Status      string `json:"status" gorm:"size:32;default:'pending';index"`
	RetryCount  int    `json:"retry_count" gorm:"default:0"`
	LastError   string `json:"last_error" gorm:"type:text"`
	NextRetryAt int64  `json:"next_retry_at" gorm:"bigint;default:0;index"`
	LockedAt    int64  `json:"locked_at" gorm:"bigint;default:0;index"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;index"`
}

func (o *UsageRecordOutbox) normalize() {
	o.RequestId = strings.TrimSpace(o.RequestId)
	o.Status = strings.TrimSpace(o.Status)
	if o.Status == "" {
		o.Status = UsageRecordOutboxStatusPending
	}
	now := common.GetTimestamp()
	if o.CreatedAt == 0 {
		o.CreatedAt = now
	}
	o.UpdatedAt = now
}

func (o *UsageRecordOutbox) InsertIdempotent() error {
	if o == nil {
		return errors.New("usage record outbox is nil")
	}
	o.normalize()
	if o.RequestId == "" {
		return errors.New("usage record outbox request_id is required")
	}
	if strings.TrimSpace(o.Payload) == "" {
		return errors.New("usage record outbox payload is required")
	}
	return DB.Clauses(clause.OnConflict{DoNothing: true}).Create(o).Error
}

func ListDueUsageRecordOutbox(now int64, staleBefore int64, limit int) ([]UsageRecordOutbox, error) {
	if limit <= 0 {
		limit = 100
	}
	var items []UsageRecordOutbox
	err := DB.
		Where("(status IN ? AND next_retry_at <= ?) OR (status = ? AND locked_at > 0 AND locked_at < ?)",
			[]string{UsageRecordOutboxStatusPending, UsageRecordOutboxStatusFailed},
			now,
			UsageRecordOutboxStatusProcessing,
			staleBefore,
		).
		Order("id ASC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

func GetUsageRecordOutboxByRequestID(requestId string) (*UsageRecordOutbox, error) {
	var item UsageRecordOutbox
	err := DB.Where("request_id = ?", strings.TrimSpace(requestId)).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func MarkUsageRecordOutboxProcessing(id int, now int64, staleBefore int64) (bool, error) {
	result := DB.Model(&UsageRecordOutbox{}).
		Where(
			"id = ? AND (status IN ? OR (status = ? AND locked_at > 0 AND locked_at < ?))",
			id,
			[]string{UsageRecordOutboxStatusPending, UsageRecordOutboxStatusFailed},
			UsageRecordOutboxStatusProcessing,
			staleBefore,
		).
		Updates(map[string]interface{}{
			"status":     UsageRecordOutboxStatusProcessing,
			"locked_at":  now,
			"updated_at": now,
		})
	return result.RowsAffected > 0, result.Error
}

func MarkUsageRecordOutboxSucceeded(id int, now int64) error {
	return DB.Model(&UsageRecordOutbox{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        UsageRecordOutboxStatusSucceeded,
		"last_error":    "",
		"next_retry_at": 0,
		"locked_at":     0,
		"updated_at":    now,
	}).Error
}

func MarkUsageRecordOutboxFailed(id int, lastError string, nextRetryAt int64, now int64) error {
	return DB.Model(&UsageRecordOutbox{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        UsageRecordOutboxStatusFailed,
		"retry_count":   gorm.Expr("retry_count + ?", 1),
		"last_error":    lastError,
		"next_retry_at": nextRetryAt,
		"locked_at":     0,
		"updated_at":    now,
	}).Error
}
