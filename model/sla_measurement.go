package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm/clause"
)

const (
	SlaContractStatusDraft   = "draft"
	SlaContractStatusActive  = "active"
	SlaContractStatusRetired = "retired"

	SlaProbeTypeAdmission       = "admission"
	SlaProbeTypeRuntimeLight    = "runtime_light"
	SlaProbeTypeRuntimeDeep     = "runtime_deep"
	SlaProbeTypeIncidentRecheck = "incident_recheck"

	SlaProbeRouteModeDirectUpstream     = "direct_upstream"
	SlaProbeRouteModeThroughTokenRouter = "through_token_router"

	SlaProbeRunStatusRunning   = "running"
	SlaProbeRunStatusPassed    = "passed"
	SlaProbeRunStatusFailed    = "failed"
	SlaProbeRunStatusInvalid   = "invalid"
	SlaProbeRunStatusCancelled = "cancelled"
)

type SlaContract struct {
	Id                     int    `json:"id"`
	ContractKey            string `json:"contract_key" gorm:"size:256;not null;uniqueIndex:uk_sla_contract_key"`
	ModelName              string `json:"model_name" gorm:"size:128;not null;index"`
	ModelAliases           string `json:"model_aliases" gorm:"type:text"`
	ProviderFamily         string `json:"provider_family" gorm:"size:64;not null;index"`
	SourceName             string `json:"source_name" gorm:"size:256;not null"`
	SourceRef              string `json:"source_ref" gorm:"size:512;not null"`
	SourceSHA256           string `json:"source_sha256" gorm:"size:128;not null"`
	Version                string `json:"version" gorm:"size:128;not null;index"`
	Status                 string `json:"status" gorm:"size:32;not null;default:'draft';index"`
	EffectiveFrom          int64  `json:"effective_from" gorm:"bigint;default:0;index"`
	EffectiveTo            int64  `json:"effective_to" gorm:"bigint;default:0;index"`
	MeasurementProfileJSON string `json:"measurement_profile_json" gorm:"type:text"`
	HardGateJSON           string `json:"hard_gate_json" gorm:"type:text"`
	SoftGateJSON           string `json:"soft_gate_json" gorm:"type:text"`
	ImportedAt             int64  `json:"imported_at" gorm:"bigint;index"`
	ImportedBy             int    `json:"imported_by" gorm:"default:0;index"`
	CreatedAt              int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt              int64  `json:"updated_at" gorm:"bigint"`
}

type SlaContractImportInput struct {
	ContractKey            string `json:"contract_key"`
	ModelName              string `json:"model_name"`
	ModelAliases           string `json:"model_aliases"`
	ProviderFamily         string `json:"provider_family"`
	SourceName             string `json:"source_name"`
	SourceRef              string `json:"source_ref"`
	SourceSHA256           string `json:"source_sha256"`
	Version                string `json:"version"`
	Status                 string `json:"status"`
	EffectiveFrom          int64  `json:"effective_from"`
	EffectiveTo            int64  `json:"effective_to"`
	MeasurementProfileJSON string `json:"measurement_profile_json"`
	HardGateJSON           string `json:"hard_gate_json"`
	SoftGateJSON           string `json:"soft_gate_json"`
}

type SlaContractFilters struct {
	ModelName      string
	ProviderFamily string
	Status         string
	StartTime      int64
	EndTime        int64
}

type SlaProbePlan struct {
	Id                         int    `json:"id"`
	PlanKey                    string `json:"plan_key" gorm:"size:512;not null;uniqueIndex:uk_sla_probe_plan_key"`
	ContractId                 int    `json:"contract_id" gorm:"not null;index"`
	SupplierId                 int    `json:"supplier_id" gorm:"not null;index"`
	ChannelId                  int    `json:"channel_id" gorm:"default:0;index"`
	ModelName                  string `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier                    string `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	ProbeType                  string `json:"probe_type" gorm:"size:32;not null;index"`
	RouteMode                  string `json:"route_mode" gorm:"size:32;not null;index"`
	PromptSuiteKey             string `json:"prompt_suite_key" gorm:"size:256;not null"`
	TokenizerRef               string `json:"tokenizer_ref" gorm:"size:256;not null"`
	SampleSize                 int    `json:"sample_size" gorm:"default:0"`
	RepeatCount                int    `json:"repeat_count" gorm:"default:1"`
	InputProfileJSON           string `json:"input_profile_json" gorm:"type:text"`
	OutputProfileJSON          string `json:"output_profile_json" gorm:"type:text"`
	ConcurrencyProfileJSON     string `json:"concurrency_profile_json" gorm:"type:text"`
	RateProfileJSON            string `json:"rate_profile_json" gorm:"type:text"`
	StreamProfileJSON          string `json:"stream_profile_json" gorm:"type:text"`
	ErrorProfileJSON           string `json:"error_profile_json" gorm:"type:text"`
	AvailabilityProfileJSON    string `json:"availability_profile_json" gorm:"type:text"`
	CacheProfile               string `json:"cache_profile" gorm:"size:64;not null;default:'cold_no_cache';index"`
	ScheduleIntervalSeconds    int    `json:"schedule_interval_seconds" gorm:"default:0"`
	JitterSeconds              int    `json:"jitter_seconds" gorm:"default:0"`
	MaxProbeQuota              int64  `json:"max_probe_quota" gorm:"default:0"`
	MeasurementProfileSnapshot string `json:"measurement_profile_snapshot" gorm:"type:text"`
	GeneratedAt                int64  `json:"generated_at" gorm:"bigint;index"`
	GeneratedBy                int    `json:"generated_by" gorm:"default:0;index"`
	CreatedAt                  int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt                  int64  `json:"updated_at" gorm:"bigint"`
}

type SlaProbePlanGenerateInput struct {
	ContractId              int    `json:"contract_id"`
	ContractKey             string `json:"contract_key"`
	SupplierId              int    `json:"supplier_id"`
	ChannelId               int    `json:"channel_id"`
	ModelName               string `json:"model_name"`
	SlaTier                 string `json:"sla_tier"`
	ProbeType               string `json:"probe_type"`
	RouteMode               string `json:"route_mode"`
	PromptSuiteKey          string `json:"prompt_suite_key"`
	TokenizerRef            string `json:"tokenizer_ref"`
	SampleSize              int    `json:"sample_size"`
	RepeatCount             int    `json:"repeat_count"`
	InputProfileJSON        string `json:"input_profile_json"`
	OutputProfileJSON       string `json:"output_profile_json"`
	ConcurrencyProfileJSON  string `json:"concurrency_profile_json"`
	RateProfileJSON         string `json:"rate_profile_json"`
	StreamProfileJSON       string `json:"stream_profile_json"`
	ErrorProfileJSON        string `json:"error_profile_json"`
	AvailabilityProfileJSON string `json:"availability_profile_json"`
	CacheProfile            string `json:"cache_profile"`
	ScheduleIntervalSeconds int    `json:"schedule_interval_seconds"`
	JitterSeconds           int    `json:"jitter_seconds"`
	MaxProbeQuota           int64  `json:"max_probe_quota"`
}

type SlaProbePlanFilters struct {
	ContractId int
	SupplierId int
	ChannelId  int
	ModelName  string
	SlaTier    string
	ProbeType  string
	RouteMode  string
	StartTime  int64
	EndTime    int64
}

type SlaProbeRun struct {
	Id               int    `json:"id"`
	RunKey           string `json:"run_key" gorm:"size:512;not null;uniqueIndex:uk_sla_probe_run_key"`
	PlanId           int    `json:"plan_id" gorm:"not null;index"`
	ContractId       int    `json:"contract_id" gorm:"not null;index"`
	SupplierId       int    `json:"supplier_id" gorm:"not null;index"`
	ChannelId        int    `json:"channel_id" gorm:"default:0;index"`
	Status           string `json:"status" gorm:"size:32;not null;default:'running';index"`
	StartedAt        int64  `json:"started_at" gorm:"bigint;default:0;index"`
	EndedAt          int64  `json:"ended_at" gorm:"bigint;default:0;index"`
	RunnerVersion    string `json:"runner_version" gorm:"size:128"`
	GitCommit        string `json:"git_commit" gorm:"size:128"`
	RuntimeRef       string `json:"runtime_ref" gorm:"size:256"`
	Endpoint         string `json:"endpoint" gorm:"size:512"`
	RouteMode        string `json:"route_mode" gorm:"size:32;not null;index"`
	ModelName        string `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier          string `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	SummaryJSON      string `json:"summary_json" gorm:"type:text"`
	HardGatePassed   bool   `json:"hard_gate_passed" gorm:"default:false;index"`
	SoftGateWarnings string `json:"soft_gate_warnings" gorm:"type:text"`
	FailureReasons   string `json:"failure_reasons" gorm:"type:text"`
	ArtifactURI      string `json:"artifact_uri" gorm:"size:1024"`
	ArtifactSHA256   string `json:"artifact_sha256" gorm:"size:128"`
	RecordedAt       int64  `json:"recorded_at" gorm:"bigint;index"`
	RecordedBy       int    `json:"recorded_by" gorm:"default:0;index"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt        int64  `json:"updated_at" gorm:"bigint"`
}

type SlaProbeRunRecordInput struct {
	RunKey           string `json:"run_key"`
	PlanId           int    `json:"plan_id"`
	Status           string `json:"status"`
	StartedAt        int64  `json:"started_at"`
	EndedAt          int64  `json:"ended_at"`
	RunnerVersion    string `json:"runner_version"`
	GitCommit        string `json:"git_commit"`
	RuntimeRef       string `json:"runtime_ref"`
	Endpoint         string `json:"endpoint"`
	SummaryJSON      string `json:"summary_json"`
	HardGatePassed   bool   `json:"hard_gate_passed"`
	SoftGateWarnings string `json:"soft_gate_warnings"`
	FailureReasons   string `json:"failure_reasons"`
	ArtifactURI      string `json:"artifact_uri"`
	ArtifactSHA256   string `json:"artifact_sha256"`
}

type SlaProbeRunFilters struct {
	PlanId     int
	ContractId int
	SupplierId int
	ChannelId  int
	ModelName  string
	SlaTier    string
	Status     string
	RouteMode  string
	StartTime  int64
	EndTime    int64
}

func normalizeSlaContractStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", SlaContractStatusDraft:
		return SlaContractStatusDraft
	case SlaContractStatusActive:
		return SlaContractStatusActive
	case SlaContractStatusRetired:
		return SlaContractStatusRetired
	default:
		return ""
	}
}

func normalizeSlaProbeType(probeType string) string {
	switch strings.TrimSpace(probeType) {
	case "", SlaProbeTypeAdmission:
		return SlaProbeTypeAdmission
	case SlaProbeTypeRuntimeLight:
		return SlaProbeTypeRuntimeLight
	case SlaProbeTypeRuntimeDeep:
		return SlaProbeTypeRuntimeDeep
	case SlaProbeTypeIncidentRecheck:
		return SlaProbeTypeIncidentRecheck
	default:
		return ""
	}
}

func normalizeSlaProbeRouteMode(routeMode string) string {
	switch strings.TrimSpace(routeMode) {
	case "", SlaProbeRouteModeDirectUpstream:
		return SlaProbeRouteModeDirectUpstream
	case SlaProbeRouteModeThroughTokenRouter:
		return SlaProbeRouteModeThroughTokenRouter
	default:
		return ""
	}
}

func normalizeSlaProbeRunStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", SlaProbeRunStatusRunning:
		return SlaProbeRunStatusRunning
	case SlaProbeRunStatusPassed:
		return SlaProbeRunStatusPassed
	case SlaProbeRunStatusFailed:
		return SlaProbeRunStatusFailed
	case SlaProbeRunStatusInvalid:
		return SlaProbeRunStatusInvalid
	case SlaProbeRunStatusCancelled:
		return SlaProbeRunStatusCancelled
	default:
		return ""
	}
}

func normalizeJSONText(value string, required bool, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		if required {
			return "", fmt.Errorf("%s is required", field)
		}
		return "{}", nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", fmt.Errorf("%s must be valid JSON: %w", field, err)
	}
	return value, nil
}

func ImportSlaContract(input SlaContractImportInput, importedBy int) (*SlaContract, error) {
	status := normalizeSlaContractStatus(input.Status)
	if status == "" {
		return nil, errors.New("invalid SLA contract status")
	}
	measurementProfile, err := normalizeJSONText(input.MeasurementProfileJSON, true, "measurement_profile_json")
	if err != nil {
		return nil, err
	}
	hardGate, err := normalizeJSONText(input.HardGateJSON, false, "hard_gate_json")
	if err != nil {
		return nil, err
	}
	softGate, err := normalizeJSONText(input.SoftGateJSON, false, "soft_gate_json")
	if err != nil {
		return nil, err
	}

	contract := SlaContract{
		ContractKey:            strings.TrimSpace(input.ContractKey),
		ModelName:              strings.TrimSpace(input.ModelName),
		ModelAliases:           strings.TrimSpace(input.ModelAliases),
		ProviderFamily:         strings.TrimSpace(input.ProviderFamily),
		SourceName:             strings.TrimSpace(input.SourceName),
		SourceRef:              strings.TrimSpace(input.SourceRef),
		SourceSHA256:           strings.TrimSpace(input.SourceSHA256),
		Version:                strings.TrimSpace(input.Version),
		Status:                 status,
		EffectiveFrom:          input.EffectiveFrom,
		EffectiveTo:            input.EffectiveTo,
		MeasurementProfileJSON: measurementProfile,
		HardGateJSON:           hardGate,
		SoftGateJSON:           softGate,
	}
	if contract.ContractKey == "" {
		return nil, errors.New("contract_key is required")
	}
	if contract.ModelName == "" {
		return nil, errors.New("model_name is required")
	}
	if contract.ProviderFamily == "" {
		return nil, errors.New("provider_family is required")
	}
	if contract.SourceName == "" {
		return nil, errors.New("source_name is required")
	}
	if contract.SourceRef == "" {
		return nil, errors.New("source_ref is required")
	}
	if contract.SourceSHA256 == "" {
		return nil, errors.New("source_sha256 is required")
	}
	if contract.Version == "" {
		return nil, errors.New("version is required")
	}
	if contract.EffectiveTo > 0 && contract.EffectiveFrom > 0 && contract.EffectiveTo <= contract.EffectiveFrom {
		return nil, errors.New("effective_to must be greater than effective_from")
	}

	now := common.GetTimestamp()
	contract.ImportedAt = now
	contract.ImportedBy = importedBy
	contract.CreatedAt = now
	contract.UpdatedAt = now
	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "contract_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"model_name",
			"model_aliases",
			"provider_family",
			"source_name",
			"source_ref",
			"source_sha256",
			"version",
			"status",
			"effective_from",
			"effective_to",
			"measurement_profile_json",
			"hard_gate_json",
			"soft_gate_json",
			"imported_at",
			"imported_by",
			"updated_at",
		}),
	}).Create(&contract).Error
	if err != nil {
		return nil, err
	}
	var saved SlaContract
	err = DB.Where("contract_key = ?", contract.ContractKey).First(&saved).Error
	return &saved, err
}

func SearchSlaContracts(filters SlaContractFilters, offset int, limit int) ([]*SlaContract, int64, error) {
	db := DB.Model(&SlaContract{})
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.ProviderFamily) != "" {
		db = db.Where("provider_family = ?", strings.TrimSpace(filters.ProviderFamily))
	}
	if strings.TrimSpace(filters.Status) != "" {
		if status := normalizeSlaContractStatus(filters.Status); status != "" {
			db = db.Where("status = ?", status)
		}
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
	var contracts []*SlaContract
	err := db.Offset(offset).Limit(limit).Order("imported_at DESC, id DESC").Find(&contracts).Error
	return contracts, total, err
}

func GetSlaContractByID(id int) (*SlaContract, error) {
	var contract SlaContract
	err := DB.First(&contract, id).Error
	return &contract, err
}

func getSlaContractForPlan(input SlaProbePlanGenerateInput) (*SlaContract, error) {
	var contract SlaContract
	if input.ContractId > 0 {
		if err := DB.First(&contract, input.ContractId).Error; err != nil {
			return nil, err
		}
		return &contract, nil
	}
	contractKey := strings.TrimSpace(input.ContractKey)
	if contractKey == "" {
		return nil, errors.New("contract_id or contract_key is required")
	}
	if err := DB.Where("contract_key = ?", contractKey).First(&contract).Error; err != nil {
		return nil, err
	}
	return &contract, nil
}

func extractProfileJSON(measurementProfile string, keys ...string) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(measurementProfile), &raw); err != nil {
		return "{}"
	}
	for _, key := range keys {
		if value, ok := raw[key]; ok && len(value) > 0 {
			return strings.TrimSpace(string(value))
		}
	}
	return "{}"
}

func extractProfileString(measurementProfile string, keys ...string) string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(measurementProfile), &raw); err != nil {
		return ""
	}
	for _, key := range keys {
		value, ok := raw[key]
		if !ok || len(value) == 0 {
			continue
		}
		var asString string
		if err := json.Unmarshal(value, &asString); err == nil {
			return strings.TrimSpace(asString)
		}
		return strings.TrimSpace(string(value))
	}
	return ""
}

func GenerateSlaProbePlan(input SlaProbePlanGenerateInput, generatedBy int) (*SlaProbePlan, error) {
	contract, err := getSlaContractForPlan(input)
	if err != nil {
		return nil, err
	}
	if _, err := GetSupplierByID(input.SupplierId); err != nil {
		return nil, err
	}
	if input.ChannelId > 0 {
		channel, err := GetChannelById(input.ChannelId, true)
		if err != nil {
			return nil, err
		}
		if channel.SupplierId != input.SupplierId {
			return nil, fmt.Errorf("channel_id=%d does not belong to supplier_id=%d", input.ChannelId, input.SupplierId)
		}
	}

	probeType := normalizeSlaProbeType(input.ProbeType)
	if probeType == "" {
		return nil, errors.New("invalid SLA probe type")
	}
	routeMode := normalizeSlaProbeRouteMode(input.RouteMode)
	if routeMode == "" {
		return nil, errors.New("invalid SLA probe route mode")
	}
	modelName := strings.TrimSpace(input.ModelName)
	if modelName == "" {
		modelName = contract.ModelName
	}
	slaTier := normalizeRoutingSlaTier(input.SlaTier)
	promptSuiteKey := strings.TrimSpace(input.PromptSuiteKey)
	if promptSuiteKey == "" {
		promptSuiteKey = "default"
	}
	tokenizerRef := strings.TrimSpace(input.TokenizerRef)
	if tokenizerRef == "" {
		tokenizerRef = "contract"
	}
	sampleSize := input.SampleSize
	if sampleSize <= 0 {
		sampleSize = 1
	}
	repeatCount := input.RepeatCount
	if repeatCount <= 0 {
		repeatCount = 1
	}
	inputProfile := strings.TrimSpace(input.InputProfileJSON)
	if inputProfile == "" {
		inputProfile = extractProfileJSON(contract.MeasurementProfileJSON, "input_profile", "input")
	}
	outputProfile := strings.TrimSpace(input.OutputProfileJSON)
	if outputProfile == "" {
		outputProfile = extractProfileJSON(contract.MeasurementProfileJSON, "output_profile", "output")
	}
	concurrencyProfile := strings.TrimSpace(input.ConcurrencyProfileJSON)
	if concurrencyProfile == "" {
		concurrencyProfile = extractProfileJSON(contract.MeasurementProfileJSON, "concurrency_profile", "concurrency")
	}
	rateProfile := strings.TrimSpace(input.RateProfileJSON)
	if rateProfile == "" {
		rateProfile = extractProfileJSON(contract.MeasurementProfileJSON, "rate_profile", "rate")
	}
	streamProfile := strings.TrimSpace(input.StreamProfileJSON)
	if streamProfile == "" {
		streamProfile = extractProfileJSON(contract.MeasurementProfileJSON, "stream_profile", "stream")
	}
	errorProfile := strings.TrimSpace(input.ErrorProfileJSON)
	if errorProfile == "" {
		errorProfile = extractProfileJSON(contract.MeasurementProfileJSON, "error_profile", "error")
	}
	availabilityProfile := strings.TrimSpace(input.AvailabilityProfileJSON)
	if availabilityProfile == "" {
		availabilityProfile = extractProfileJSON(contract.MeasurementProfileJSON, "availability_profile", "availability")
	}
	cacheProfile := strings.TrimSpace(input.CacheProfile)
	if cacheProfile == "" {
		cacheProfile = extractProfileString(contract.MeasurementProfileJSON, "cache_profile", "cache")
	}
	if cacheProfile == "" {
		cacheProfile = "cold_no_cache"
	}
	if inputProfile, err = normalizeJSONText(inputProfile, false, "input_profile_json"); err != nil {
		return nil, err
	}
	if outputProfile, err = normalizeJSONText(outputProfile, false, "output_profile_json"); err != nil {
		return nil, err
	}
	if concurrencyProfile, err = normalizeJSONText(concurrencyProfile, false, "concurrency_profile_json"); err != nil {
		return nil, err
	}
	if rateProfile, err = normalizeJSONText(rateProfile, false, "rate_profile_json"); err != nil {
		return nil, err
	}
	if streamProfile, err = normalizeJSONText(streamProfile, false, "stream_profile_json"); err != nil {
		return nil, err
	}
	if errorProfile, err = normalizeJSONText(errorProfile, false, "error_profile_json"); err != nil {
		return nil, err
	}
	if availabilityProfile, err = normalizeJSONText(availabilityProfile, false, "availability_profile_json"); err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	plan := SlaProbePlan{
		PlanKey:                    slaProbePlanKey(contract.Id, input.SupplierId, input.ChannelId, modelName, slaTier, probeType, routeMode),
		ContractId:                 contract.Id,
		SupplierId:                 input.SupplierId,
		ChannelId:                  input.ChannelId,
		ModelName:                  modelName,
		SlaTier:                    slaTier,
		ProbeType:                  probeType,
		RouteMode:                  routeMode,
		PromptSuiteKey:             promptSuiteKey,
		TokenizerRef:               tokenizerRef,
		SampleSize:                 sampleSize,
		RepeatCount:                repeatCount,
		InputProfileJSON:           inputProfile,
		OutputProfileJSON:          outputProfile,
		ConcurrencyProfileJSON:     concurrencyProfile,
		RateProfileJSON:            rateProfile,
		StreamProfileJSON:          streamProfile,
		ErrorProfileJSON:           errorProfile,
		AvailabilityProfileJSON:    availabilityProfile,
		CacheProfile:               cacheProfile,
		ScheduleIntervalSeconds:    input.ScheduleIntervalSeconds,
		JitterSeconds:              input.JitterSeconds,
		MaxProbeQuota:              input.MaxProbeQuota,
		MeasurementProfileSnapshot: contract.MeasurementProfileJSON,
		GeneratedAt:                now,
		GeneratedBy:                generatedBy,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "plan_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"contract_id",
			"supplier_id",
			"channel_id",
			"model_name",
			"sla_tier",
			"probe_type",
			"route_mode",
			"prompt_suite_key",
			"tokenizer_ref",
			"sample_size",
			"repeat_count",
			"input_profile_json",
			"output_profile_json",
			"concurrency_profile_json",
			"rate_profile_json",
			"stream_profile_json",
			"error_profile_json",
			"availability_profile_json",
			"cache_profile",
			"schedule_interval_seconds",
			"jitter_seconds",
			"max_probe_quota",
			"measurement_profile_snapshot",
			"generated_at",
			"generated_by",
			"updated_at",
		}),
	}).Create(&plan).Error
	if err != nil {
		return nil, err
	}
	var saved SlaProbePlan
	err = DB.Where("plan_key = ?", plan.PlanKey).First(&saved).Error
	return &saved, err
}

func slaProbePlanKey(contractId int, supplierId int, channelId int, modelName string, slaTier string, probeType string, routeMode string) string {
	return fmt.Sprintf("contract:%d|supplier:%d|channel:%d|model:%s|sla:%s|type:%s|route:%s", contractId, supplierId, channelId, modelName, slaTier, probeType, routeMode)
}

func SearchSlaProbePlans(filters SlaProbePlanFilters, offset int, limit int) ([]*SlaProbePlan, int64, error) {
	db := DB.Model(&SlaProbePlan{})
	if filters.ContractId > 0 {
		db = db.Where("contract_id = ?", filters.ContractId)
	}
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if filters.ChannelId > 0 {
		db = db.Where("channel_id = ?", filters.ChannelId)
	}
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeRoutingSlaTier(filters.SlaTier))
	}
	if strings.TrimSpace(filters.ProbeType) != "" {
		if probeType := normalizeSlaProbeType(filters.ProbeType); probeType != "" {
			db = db.Where("probe_type = ?", probeType)
		}
	}
	if strings.TrimSpace(filters.RouteMode) != "" {
		if routeMode := normalizeSlaProbeRouteMode(filters.RouteMode); routeMode != "" {
			db = db.Where("route_mode = ?", routeMode)
		}
	}
	if filters.StartTime > 0 {
		db = db.Where("generated_at >= ?", filters.StartTime)
	}
	if filters.EndTime > 0 {
		db = db.Where("generated_at <= ?", filters.EndTime)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var plans []*SlaProbePlan
	err := db.Offset(offset).Limit(limit).Order("generated_at DESC, id DESC").Find(&plans).Error
	return plans, total, err
}

func GetSlaProbePlanByID(id int) (*SlaProbePlan, error) {
	var plan SlaProbePlan
	err := DB.First(&plan, id).Error
	return &plan, err
}

func RecordSlaProbeRun(input SlaProbeRunRecordInput, recordedBy int) (*SlaProbeRun, error) {
	if input.PlanId <= 0 {
		return nil, errors.New("plan_id is required")
	}
	var plan SlaProbePlan
	if err := DB.First(&plan, input.PlanId).Error; err != nil {
		return nil, err
	}
	status := normalizeSlaProbeRunStatus(input.Status)
	if status == "" {
		return nil, errors.New("invalid SLA probe run status")
	}
	summaryJSON, err := normalizeJSONText(input.SummaryJSON, false, "summary_json")
	if err != nil {
		return nil, err
	}
	startedAt := input.StartedAt
	if startedAt <= 0 {
		startedAt = common.GetTimestamp()
	}
	endedAt := input.EndedAt
	if endedAt > 0 && endedAt < startedAt {
		return nil, errors.New("ended_at must be greater than or equal to started_at")
	}
	runKey := strings.TrimSpace(input.RunKey)
	if runKey == "" {
		runKey = fmt.Sprintf("%s|started:%d", plan.PlanKey, startedAt)
	}
	now := common.GetTimestamp()
	run := SlaProbeRun{
		RunKey:           runKey,
		PlanId:           plan.Id,
		ContractId:       plan.ContractId,
		SupplierId:       plan.SupplierId,
		ChannelId:        plan.ChannelId,
		Status:           status,
		StartedAt:        startedAt,
		EndedAt:          endedAt,
		RunnerVersion:    strings.TrimSpace(input.RunnerVersion),
		GitCommit:        strings.TrimSpace(input.GitCommit),
		RuntimeRef:       strings.TrimSpace(input.RuntimeRef),
		Endpoint:         strings.TrimSpace(input.Endpoint),
		RouteMode:        plan.RouteMode,
		ModelName:        plan.ModelName,
		SlaTier:          plan.SlaTier,
		SummaryJSON:      summaryJSON,
		HardGatePassed:   input.HardGatePassed,
		SoftGateWarnings: strings.TrimSpace(input.SoftGateWarnings),
		FailureReasons:   strings.TrimSpace(input.FailureReasons),
		ArtifactURI:      strings.TrimSpace(input.ArtifactURI),
		ArtifactSHA256:   strings.TrimSpace(input.ArtifactSHA256),
		RecordedAt:       now,
		RecordedBy:       recordedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "run_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"plan_id",
			"contract_id",
			"supplier_id",
			"channel_id",
			"status",
			"started_at",
			"ended_at",
			"runner_version",
			"git_commit",
			"runtime_ref",
			"endpoint",
			"route_mode",
			"model_name",
			"sla_tier",
			"summary_json",
			"hard_gate_passed",
			"soft_gate_warnings",
			"failure_reasons",
			"artifact_uri",
			"artifact_sha256",
			"recorded_at",
			"recorded_by",
			"updated_at",
		}),
	}).Create(&run).Error
	if err != nil {
		return nil, err
	}
	var saved SlaProbeRun
	err = DB.Where("run_key = ?", run.RunKey).First(&saved).Error
	return &saved, err
}

func SearchSlaProbeRuns(filters SlaProbeRunFilters, offset int, limit int) ([]*SlaProbeRun, int64, error) {
	db := DB.Model(&SlaProbeRun{})
	if filters.PlanId > 0 {
		db = db.Where("plan_id = ?", filters.PlanId)
	}
	if filters.ContractId > 0 {
		db = db.Where("contract_id = ?", filters.ContractId)
	}
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if filters.ChannelId > 0 {
		db = db.Where("channel_id = ?", filters.ChannelId)
	}
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeRoutingSlaTier(filters.SlaTier))
	}
	if strings.TrimSpace(filters.Status) != "" {
		if status := normalizeSlaProbeRunStatus(filters.Status); status != "" {
			db = db.Where("status = ?", status)
		}
	}
	if strings.TrimSpace(filters.RouteMode) != "" {
		if routeMode := normalizeSlaProbeRouteMode(filters.RouteMode); routeMode != "" {
			db = db.Where("route_mode = ?", routeMode)
		}
	}
	if filters.StartTime > 0 {
		db = db.Where("started_at >= ?", filters.StartTime)
	}
	if filters.EndTime > 0 {
		db = db.Where("started_at <= ?", filters.EndTime)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var runs []*SlaProbeRun
	err := db.Offset(offset).Limit(limit).Order("started_at DESC, id DESC").Find(&runs).Error
	return runs, total, err
}

func GetSlaProbeRunByID(id int) (*SlaProbeRun, error) {
	var run SlaProbeRun
	err := DB.First(&run, id).Error
	return &run, err
}
