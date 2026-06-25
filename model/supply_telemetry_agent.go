package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

const (
	SupplyTelemetryAgentTypeTelemetry = "telemetry"

	SupplyTelemetryAgentStatusActive  = "active"
	SupplyTelemetryAgentStatusError   = "error"
	SupplyTelemetryAgentStatusStopped = "stopped"

	SupplyTelemetryAgentSweepStatusOK      = "ok"
	SupplyTelemetryAgentSweepStatusSkipped = "skipped"
	SupplyTelemetryAgentSweepStatusFailed  = "failed"
)

type SupplyTelemetryAgent struct {
	Id                      int    `json:"id"`
	AgentKey                string `json:"agent_key" gorm:"size:128;not null;uniqueIndex:uk_supply_telemetry_agent_key"`
	AgentType               string `json:"agent_type" gorm:"size:32;not null;default:'telemetry';index"`
	Hostname                string `json:"hostname" gorm:"size:128;default:'';index"`
	RuntimeRef              string `json:"runtime_ref" gorm:"size:256;default:'';index"`
	Version                 string `json:"version" gorm:"size:64;default:''"`
	Status                  string `json:"status" gorm:"size:32;not null;default:'active';index"`
	LastHeartbeatAt         int64  `json:"last_heartbeat_at" gorm:"bigint;default:0;index"`
	LastSweepStartedAt      int64  `json:"last_sweep_started_at" gorm:"bigint;default:0"`
	LastSweepFinishedAt     int64  `json:"last_sweep_finished_at" gorm:"bigint;default:0;index"`
	LastSweepStatus         string `json:"last_sweep_status" gorm:"size:32;default:'';index"`
	LastSweepError          string `json:"last_sweep_error,omitempty" gorm:"type:text"`
	LastSweepAttemptedCount int    `json:"last_sweep_attempted_count" gorm:"default:0"`
	LastSweepCollectedCount int    `json:"last_sweep_collected_count" gorm:"default:0"`
	LastSweepSkippedCount   int    `json:"last_sweep_skipped_count" gorm:"default:0"`
	LastSweepSupplierId     int    `json:"last_sweep_supplier_id" gorm:"default:0;index"`
	LastSweepSupplyNode     string `json:"last_sweep_supply_node" gorm:"size:128;default:'';index"`
	LastSweepModelName      string `json:"last_sweep_model_name" gorm:"size:128;default:'';index"`
	LastSweepPeriodStart    int64  `json:"last_sweep_period_start" gorm:"bigint;default:0"`
	LastSweepPeriodEnd      int64  `json:"last_sweep_period_end" gorm:"bigint;default:0"`
	RecordedBy              int    `json:"recorded_by" gorm:"default:0;index"`
	CreatedAt               int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt               int64  `json:"updated_at" gorm:"bigint"`
}

type SupplyTelemetryAgentHeartbeatInput struct {
	AgentKey    string `json:"agent_key"`
	AgentType   string `json:"agent_type"`
	Hostname    string `json:"hostname"`
	RuntimeRef  string `json:"runtime_ref"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	HeartbeatAt int64  `json:"heartbeat_at"`
}

type SupplyTelemetryAgentSweepResultInput struct {
	AgentKey       string `json:"agent_key"`
	AgentType      string `json:"agent_type"`
	Hostname       string `json:"hostname"`
	RuntimeRef     string `json:"runtime_ref"`
	Version        string `json:"version"`
	StartedAt      int64  `json:"started_at"`
	FinishedAt     int64  `json:"finished_at"`
	Status         string `json:"status"`
	Error          string `json:"error"`
	AttemptedCount int    `json:"attempted_count"`
	CollectedCount int    `json:"collected_count"`
	SkippedCount   int    `json:"skipped_count"`
	SupplierId     int    `json:"supplier_id"`
	SupplyNode     string `json:"supply_node"`
	ModelName      string `json:"model_name"`
	PeriodStart    int64  `json:"period_start"`
	PeriodEnd      int64  `json:"period_end"`
}

type SupplyTelemetryAgentFilters struct {
	AgentKey    string
	AgentType   string
	Status      string
	StaleBefore int64
}

func normalizeSupplyTelemetryAgentType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return SupplyTelemetryAgentTypeTelemetry
	}
	return value
}

func normalizeSupplyTelemetryAgentStatus(value string) string {
	switch strings.TrimSpace(value) {
	case SupplyTelemetryAgentStatusError:
		return SupplyTelemetryAgentStatusError
	case SupplyTelemetryAgentStatusStopped:
		return SupplyTelemetryAgentStatusStopped
	default:
		return SupplyTelemetryAgentStatusActive
	}
}

func normalizeSupplyTelemetryAgentSweepStatus(value string) string {
	switch strings.TrimSpace(value) {
	case SupplyTelemetryAgentSweepStatusSkipped:
		return SupplyTelemetryAgentSweepStatusSkipped
	case SupplyTelemetryAgentSweepStatusFailed:
		return SupplyTelemetryAgentSweepStatusFailed
	default:
		return SupplyTelemetryAgentSweepStatusOK
	}
}

func trimSupplyTelemetryAgentIdentity(agent *SupplyTelemetryAgent) {
	agent.AgentKey = strings.TrimSpace(agent.AgentKey)
	agent.AgentType = normalizeSupplyTelemetryAgentType(agent.AgentType)
	agent.Hostname = strings.TrimSpace(agent.Hostname)
	agent.RuntimeRef = strings.TrimSpace(agent.RuntimeRef)
	agent.Version = strings.TrimSpace(agent.Version)
	agent.Status = normalizeSupplyTelemetryAgentStatus(agent.Status)
	agent.LastSweepStatus = strings.TrimSpace(agent.LastSweepStatus)
	agent.LastSweepError = strings.TrimSpace(agent.LastSweepError)
	agent.LastSweepSupplyNode = strings.TrimSpace(agent.LastSweepSupplyNode)
	agent.LastSweepModelName = strings.TrimSpace(agent.LastSweepModelName)
}

func validateSupplyTelemetryAgent(agent SupplyTelemetryAgent) error {
	if strings.TrimSpace(agent.AgentKey) == "" {
		return errors.New("agent_key is required")
	}
	if len([]rune(agent.AgentKey)) > 128 {
		return errors.New("agent_key is too long")
	}
	return nil
}

func RecordSupplyTelemetryAgentHeartbeat(input SupplyTelemetryAgentHeartbeatInput, recordedBy int) (*SupplyTelemetryAgent, error) {
	now := time.Now().Unix()
	heartbeatAt := input.HeartbeatAt
	if heartbeatAt <= 0 {
		heartbeatAt = now
	}
	agent := SupplyTelemetryAgent{
		AgentKey:        input.AgentKey,
		AgentType:       input.AgentType,
		Hostname:        input.Hostname,
		RuntimeRef:      input.RuntimeRef,
		Version:         input.Version,
		Status:          input.Status,
		LastHeartbeatAt: heartbeatAt,
		RecordedBy:      recordedBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	trimSupplyTelemetryAgentIdentity(&agent)
	if err := validateSupplyTelemetryAgent(agent); err != nil {
		return nil, err
	}
	if err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "agent_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"agent_type",
			"hostname",
			"runtime_ref",
			"version",
			"status",
			"last_heartbeat_at",
			"recorded_by",
			"updated_at",
		}),
	}).Create(&agent).Error; err != nil {
		return nil, err
	}
	var result SupplyTelemetryAgent
	if err := DB.Where("agent_key = ?", agent.AgentKey).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func RecordSupplyTelemetryAgentSweepResult(input SupplyTelemetryAgentSweepResultInput, recordedBy int) (*SupplyTelemetryAgent, error) {
	now := time.Now().Unix()
	finishedAt := input.FinishedAt
	if finishedAt <= 0 {
		finishedAt = now
	}
	startedAt := input.StartedAt
	if startedAt <= 0 {
		startedAt = finishedAt
	}
	sweepStatus := normalizeSupplyTelemetryAgentSweepStatus(input.Status)
	agentStatus := SupplyTelemetryAgentStatusActive
	if sweepStatus == SupplyTelemetryAgentSweepStatusFailed {
		agentStatus = SupplyTelemetryAgentStatusError
	}
	agent := SupplyTelemetryAgent{
		AgentKey:                input.AgentKey,
		AgentType:               input.AgentType,
		Hostname:                input.Hostname,
		RuntimeRef:              input.RuntimeRef,
		Version:                 input.Version,
		Status:                  agentStatus,
		LastHeartbeatAt:         finishedAt,
		LastSweepStartedAt:      startedAt,
		LastSweepFinishedAt:     finishedAt,
		LastSweepStatus:         sweepStatus,
		LastSweepError:          input.Error,
		LastSweepAttemptedCount: input.AttemptedCount,
		LastSweepCollectedCount: input.CollectedCount,
		LastSweepSkippedCount:   input.SkippedCount,
		LastSweepSupplierId:     input.SupplierId,
		LastSweepSupplyNode:     input.SupplyNode,
		LastSweepModelName:      input.ModelName,
		LastSweepPeriodStart:    input.PeriodStart,
		LastSweepPeriodEnd:      input.PeriodEnd,
		RecordedBy:              recordedBy,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	trimSupplyTelemetryAgentIdentity(&agent)
	agent.LastSweepStatus = sweepStatus
	if err := validateSupplyTelemetryAgent(agent); err != nil {
		return nil, err
	}
	if err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "agent_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"agent_type",
			"hostname",
			"runtime_ref",
			"version",
			"status",
			"last_heartbeat_at",
			"last_sweep_started_at",
			"last_sweep_finished_at",
			"last_sweep_status",
			"last_sweep_error",
			"last_sweep_attempted_count",
			"last_sweep_collected_count",
			"last_sweep_skipped_count",
			"last_sweep_supplier_id",
			"last_sweep_supply_node",
			"last_sweep_model_name",
			"last_sweep_period_start",
			"last_sweep_period_end",
			"recorded_by",
			"updated_at",
		}),
	}).Create(&agent).Error; err != nil {
		return nil, err
	}
	var result SupplyTelemetryAgent
	if err := DB.Where("agent_key = ?", agent.AgentKey).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func SearchSupplyTelemetryAgents(filters SupplyTelemetryAgentFilters, offset int, limit int) ([]*SupplyTelemetryAgent, int64, error) {
	db := DB.Model(&SupplyTelemetryAgent{})
	if strings.TrimSpace(filters.AgentKey) != "" {
		db = db.Where("agent_key = ?", strings.TrimSpace(filters.AgentKey))
	}
	if strings.TrimSpace(filters.AgentType) != "" {
		db = db.Where("agent_type = ?", normalizeSupplyTelemetryAgentType(filters.AgentType))
	}
	if strings.TrimSpace(filters.Status) != "" {
		db = db.Where("status = ?", normalizeSupplyTelemetryAgentStatus(filters.Status))
	}
	if filters.StaleBefore > 0 {
		db = db.Where("last_heartbeat_at < ?", filters.StaleBefore)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var agents []*SupplyTelemetryAgent
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	if err := db.Order("last_heartbeat_at DESC, id DESC").Find(&agents).Error; err != nil {
		return nil, 0, err
	}
	return agents, total, nil
}
