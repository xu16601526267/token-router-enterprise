package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func parseOptionalIntQuery(c *gin.Context, key string) int {
	value := c.Query(key)
	if value == "" {
		return 0
	}
	parsed, _ := strconv.Atoi(value)
	return parsed
}

func parseOptionalInt64Query(c *gin.Context, key string) int64 {
	value := c.Query(key)
	if value == "" {
		return 0
	}
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}

func GetSuppliers(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := parseOptionalIntQuery(c, "status")
	suppliers, total, err := model.SearchSuppliers(c.Query("keyword"), c.Query("type"), status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(suppliers)
	common.ApiSuccess(c, pageInfo)
}

func GetSupplier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	supplier, err := model.GetSupplierByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, supplier)
}

func CreateSupplier(c *gin.Context) {
	var supplier model.Supplier
	if err := c.ShouldBindJSON(&supplier); err != nil {
		common.ApiError(c, err)
		return
	}
	if supplier.Name == "" {
		common.ApiErrorMsg(c, "供应商名称不能为空")
		return
	}
	if duplicated, err := model.IsSupplierNameDuplicated(0, supplier.Name); err != nil {
		common.ApiError(c, err)
		return
	} else if duplicated {
		common.ApiErrorMsg(c, "供应商名称已存在")
		return
	}
	if err := supplier.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &supplier)
}

func UpdateSupplier(c *gin.Context) {
	var supplier model.Supplier
	if err := c.ShouldBindJSON(&supplier); err != nil {
		common.ApiError(c, err)
		return
	}
	if supplier.Id == 0 {
		common.ApiErrorMsg(c, "缺少供应商 ID")
		return
	}
	if supplier.Name == "" {
		common.ApiErrorMsg(c, "供应商名称不能为空")
		return
	}
	if duplicated, err := model.IsSupplierNameDuplicated(supplier.Id, supplier.Name); err != nil {
		common.ApiError(c, err)
		return
	} else if duplicated {
		common.ApiErrorMsg(c, "供应商名称已存在")
		return
	}
	if err := supplier.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &supplier)
}

func DeleteSupplier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	supplier, err := model.GetSupplierByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := supplier.Delete(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetSupplierAgreements(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	supplierId := parseOptionalIntQuery(c, "supplier_id")
	status := parseOptionalIntQuery(c, "status")
	agreements, total, err := model.SearchSupplierAgreements(supplierId, c.Query("model_name"), status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(agreements)
	common.ApiSuccess(c, pageInfo)
}

func GetSupplierAgreement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	agreement, err := model.GetSupplierAgreementByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, agreement)
}

func CreateSupplierAgreement(c *gin.Context) {
	var agreement model.SupplierAgreement
	if err := c.ShouldBindJSON(&agreement); err != nil {
		common.ApiError(c, err)
		return
	}
	if agreement.SupplierId <= 0 {
		common.ApiErrorMsg(c, "缺少供应商 ID")
		return
	}
	if _, err := model.GetSupplierByID(agreement.SupplierId); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := agreement.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &agreement)
}

func UpdateSupplierAgreement(c *gin.Context) {
	var agreement model.SupplierAgreement
	if err := c.ShouldBindJSON(&agreement); err != nil {
		common.ApiError(c, err)
		return
	}
	if agreement.Id == 0 {
		common.ApiErrorMsg(c, "缺少协议 ID")
		return
	}
	if agreement.SupplierId <= 0 {
		common.ApiErrorMsg(c, "缺少供应商 ID")
		return
	}
	if _, err := model.GetSupplierByID(agreement.SupplierId); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := agreement.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &agreement)
}

func DeleteSupplierAgreement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteSupplierAgreementByID(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetSupplyCapacities(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	filters := model.SupplyCapacityFilters{
		SupplierId: parseOptionalIntQuery(c, "supplier_id"),
		SupplyNode: c.Query("supply_node"),
		ModelName:  c.Query("model_name"),
		Status:     parseOptionalIntQuery(c, "status"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	}
	capacities, total, err := model.SearchSupplyCapacities(filters, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(capacities)
	common.ApiSuccess(c, pageInfo)
}

func GetSupplyCapacity(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	capacity, err := model.GetSupplyCapacityByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, capacity)
}

func CreateSupplyCapacity(c *gin.Context) {
	var capacity model.SupplyCapacity
	if err := c.ShouldBindJSON(&capacity); err != nil {
		common.ApiError(c, err)
		return
	}
	if capacity.SupplierId <= 0 {
		common.ApiErrorMsg(c, "缺少供应商 ID")
		return
	}
	if _, err := model.GetSupplierByID(capacity.SupplierId); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := capacity.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &capacity)
}

func RefreshSupplyCapacityUsage(c *gin.Context) {
	var input model.SupplyCapacityUsageRefreshInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	capacities, err := model.RefreshSupplyCapacityUsage(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, capacities)
}

func RecordSupplyCapacityTelemetry(c *gin.Context) {
	var input model.SupplyCapacityTelemetryRecordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	telemetry, err := model.RecordSupplyCapacityTelemetry(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, telemetry)
}

func CollectSupplyCapacityTelemetry(c *gin.Context) {
	var input model.SupplyCapacityTelemetryCollectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	telemetry, err := model.CollectSupplyCapacityTelemetry(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, telemetry)
}

func SweepSupplyCapacityTelemetry(c *gin.Context) {
	var input model.SupplyCapacityTelemetrySweepInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	result, err := model.SweepSupplyCapacityTelemetry(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetSupplyCapacityTelemetries(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	filters := model.SupplyCapacityTelemetryFilters{
		SupplierId:        parseOptionalIntQuery(c, "supplier_id"),
		SupplyNode:        c.Query("supply_node"),
		ModelName:         c.Query("model_name"),
		SourceType:        c.Query("source_type"),
		AppliedCapacityId: parseOptionalIntQuery(c, "applied_capacity_id"),
		StartTime:         parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:           parseOptionalInt64Query(c, "end_timestamp"),
	}
	telemetries, total, err := model.SearchSupplyCapacityTelemetries(filters, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(telemetries)
	common.ApiSuccess(c, pageInfo)
}

func RecordSupplyTelemetryAgentHeartbeat(c *gin.Context) {
	var input model.SupplyTelemetryAgentHeartbeatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	agent, err := model.RecordSupplyTelemetryAgentHeartbeat(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, agent)
}

func RecordSupplyTelemetryAgentSweepResult(c *gin.Context) {
	var input model.SupplyTelemetryAgentSweepResultInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	agent, err := model.RecordSupplyTelemetryAgentSweepResult(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, agent)
}

func GetSupplyTelemetryAgents(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	filters := model.SupplyTelemetryAgentFilters{
		AgentKey:    c.Query("agent_key"),
		AgentType:   c.Query("agent_type"),
		Status:      c.Query("status"),
		StaleBefore: parseOptionalInt64Query(c, "stale_before"),
	}
	agents, total, err := model.SearchSupplyTelemetryAgents(filters, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(agents)
	common.ApiSuccess(c, pageInfo)
}

func RecordSupplyCostProfile(c *gin.Context) {
	var input model.SupplyCostProfileRecordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	profile, err := model.RecordSupplyCostProfile(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, profile)
}

func GetSupplyCostProfiles(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	filters := model.SupplyCostProfileFilters{
		SupplierId: parseOptionalIntQuery(c, "supplier_id"),
		SupplyNode: c.Query("supply_node"),
		ModelName:  c.Query("model_name"),
		SourceType: c.Query("source_type"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	}
	profiles, total, err := model.SearchSupplyCostProfiles(filters, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(profiles)
	common.ApiSuccess(c, pageInfo)
}

func RecordSupplyPrepaidLot(c *gin.Context) {
	var input model.SupplyPrepaidLotRecordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	lot, err := model.RecordSupplyPrepaidLot(input, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, lot)
}

func GetSupplyPrepaidLots(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	filters := model.SupplyPrepaidLotFilters{
		PrepaidLotId: parseOptionalIntQuery(c, "prepaid_lot_id"),
		SupplierId:   parseOptionalIntQuery(c, "supplier_id"),
		ChannelId:    parseOptionalIntQuery(c, "channel_id"),
		SupplyNode:   c.Query("supply_node"),
		ModelName:    c.Query("model_name"),
		SourceType:   c.Query("source_type"),
		StartTime:    parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:      parseOptionalInt64Query(c, "end_timestamp"),
	}
	lots, total, err := model.SearchSupplyPrepaidLots(filters, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(lots)
	common.ApiSuccess(c, pageInfo)
}

func RefreshSupplyPrepaidLotUsage(c *gin.Context) {
	var input model.SupplyPrepaidLotUsageRefreshInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	lots, err := model.RefreshSupplyPrepaidLotUsage(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, lots)
}

func UpdateSupplyCapacity(c *gin.Context) {
	var capacity model.SupplyCapacity
	if err := c.ShouldBindJSON(&capacity); err != nil {
		common.ApiError(c, err)
		return
	}
	if capacity.Id == 0 {
		common.ApiErrorMsg(c, "缺少供给容量快照 ID")
		return
	}
	if capacity.SupplierId <= 0 {
		common.ApiErrorMsg(c, "缺少供应商 ID")
		return
	}
	if _, err := model.GetSupplierByID(capacity.SupplierId); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := capacity.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &capacity)
}

func DeleteSupplyCapacity(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteSupplyCapacityByID(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetUsageLedgers(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	filters := model.UsageLedgerFilters{
		RequestId:  c.Query("request_id"),
		SessionId:  c.Query("session_id"),
		SupplierId: parseOptionalIntQuery(c, "supplier_id"),
		ChannelId:  parseOptionalIntQuery(c, "channel_id"),
		UserId:     parseOptionalIntQuery(c, "user_id"),
		TokenId:    parseOptionalIntQuery(c, "token_id"),
		ModelName:  c.Query("model_name"),
		Status:     c.Query("status"),
		StartTime:  parseOptionalInt64Query(c, "start_timestamp"),
		EndTime:    parseOptionalInt64Query(c, "end_timestamp"),
	}
	ledgers, total, err := model.SearchUsageLedgers(filters, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(ledgers)
	common.ApiSuccess(c, pageInfo)
}

func GetUsageLedgerByRequestID(c *gin.Context) {
	ledger, err := model.GetUsageLedgerByRequestID(c.Param("request_id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ledger)
}
