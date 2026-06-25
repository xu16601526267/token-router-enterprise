package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx          *gin.Context
	TokenGroup   string
	ModelName    string
	RequestPath  string
	Retry        *int
	resetNextTry bool
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
// 尝试获取一个满足要求的随机渠道。
//
// For "auto" tokenGroup with cross-group Retry enabled:
// 对于启用了跨分组重试的 "auto" tokenGroup：
//
//   - Each group will exhaust all its priorities before moving to the next group.
//     每个分组会用完所有优先级后才会切换到下一个分组。
//
//   - Uses ContextKeyAutoGroupIndex to track current group index.
//     使用 ContextKeyAutoGroupIndex 跟踪当前分组索引。
//
//   - Uses ContextKeyAutoGroupRetryIndex to track the global Retry count when current group started.
//     使用 ContextKeyAutoGroupRetryIndex 跟踪当前分组开始时的全局重试次数。
//
//   - priorityRetry = Retry - startRetryIndex, represents the priority level within current group.
//     priorityRetry = Retry - startRetryIndex，表示当前分组内的优先级级别。
//
//   - When GetRandomSatisfiedChannel returns nil (priorities exhausted), moves to next group.
//     当 GetRandomSatisfiedChannel 返回 nil（优先级用完）时，切换到下一个分组。
//
// Example flow (2 groups, each with 2 priorities, RetryTimes=3):
// 示例流程（2个分组，每个有2个优先级，RetryTimes=3）：
//
//	Retry=0: GroupA, priority0 (startRetryIndex=0, priorityRetry=0)
//	         分组A, 优先级0
//
//	Retry=1: GroupA, priority1 (startRetryIndex=0, priorityRetry=1)
//	         分组A, 优先级1
//
//	Retry=2: GroupA exhausted → GroupB, priority0 (startRetryIndex=2, priorityRetry=0)
//	         分组A用完 → 分组B, 优先级0
//
//	Retry=3: GroupB, priority1 (startRetryIndex=2, priorityRetry=1)
//	         分组B, 优先级1
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := param.TokenGroup
	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)

	if param.TokenGroup == "auto" {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}
		autoGroups := GetUserAutoGroup(userGroup)

		// startGroupIndex: the group index to start searching from
		// startGroupIndex: 开始搜索的分组索引
		startGroupIndex := 0
		crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)

		if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				startGroupIndex = idx
			}
		}

		for i := startGroupIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			var policyMiss *model.SupplyRoutingPolicyMiss
			// Calculate priorityRetry for current group
			// 计算当前分组的 priorityRetry
			priorityRetry := param.GetRetry()
			// If moved to a new group, reset priorityRetry and update startRetryIndex
			// 如果切换到新分组，重置 priorityRetry 并更新 startRetryIndex
			if i > startGroupIndex {
				priorityRetry = 0
			}
			logger.LogDebug(param.Ctx, "Auto selecting group: %s, priorityRetry: %d", autoGroup, priorityRetry)

			if policyChannel, miss := getSupplyRoutingPolicyChannel(param, autoGroup); policyChannel != nil {
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, autoGroup)
				selectGroup = autoGroup
				return policyChannel, selectGroup, nil
			} else {
				policyMiss = miss
			}

			channel, _ = model.GetRandomSatisfiedChannel(autoGroup, param.ModelName, priorityRetry, param.RequestPath)
			if channel == nil {
				// Current group has no available channel for this model, try next group
				// 当前分组没有该模型的可用渠道，尝试下一个分组
				logger.LogDebug(param.Ctx, "No available channel in group %s for model %s at priorityRetry %d, trying next group", autoGroup, param.ModelName, priorityRetry)
				// 重置状态以尝试下一个分组
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupRetryIndex, 0)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				continue
			}
			RecordSupplyRoutingPolicyMissInsight(param.Ctx, policyMiss)
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, autoGroup)
			selectGroup = autoGroup
			logger.LogDebug(param.Ctx, "Auto selected group: %s", autoGroup)

			// Prepare state for next retry
			// 为下一次重试准备状态
			if crossGroupRetry && priorityRetry >= common.RetryTimes {
				// Current group has exhausted all retries, prepare to switch to next group
				// This request still uses current group, but next retry will use next group
				// 当前分组已用完所有重试次数，准备切换到下一个分组
				// 本次请求仍使用当前分组，但下次重试将使用下一个分组
				logger.LogDebug(param.Ctx, "Current group %s retries exhausted (priorityRetry=%d >= RetryTimes=%d), preparing switch to next group for next retry", autoGroup, priorityRetry, common.RetryTimes)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				param.ResetRetryNextTry()
			} else {
				// Stay in current group, save current state
				// 保持在当前分组，保存当前状态
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i)
			}
			break
		}
	} else {
		policyChannel, policyMiss := getSupplyRoutingPolicyChannel(param, param.TokenGroup)
		if policyChannel != nil {
			return policyChannel, param.TokenGroup, nil
		}
		channel, err = model.GetRandomSatisfiedChannel(param.TokenGroup, param.ModelName, param.GetRetry(), param.RequestPath)
		RecordSupplyRoutingPolicyMissInsight(param.Ctx, policyMiss)
		if err != nil {
			return nil, param.TokenGroup, err
		}
	}
	return channel, selectGroup, nil
}

func getSupplyRoutingPolicyChannel(param *RetryParam, group string) (*model.Channel, *model.SupplyRoutingPolicyMiss) {
	if param == nil || param.Ctx == nil || group == "" {
		return nil, nil
	}
	policy, miss, err := model.ResolveActiveSupplyRoutingPolicyForRequest(model.SupplyRoutingPolicyMatchInput{
		Group:     group,
		ModelName: param.ModelName,
		SlaTier:   param.Ctx.GetString("sla_tier"),
		UserId:    param.Ctx.GetInt("id"),
		RouteKey:  supplyRoutingPolicyRouteKey(param.Ctx),
		Now:       common.GetTimestamp(),
	})
	if err != nil {
		logger.LogError(param.Ctx, fmt.Sprintf("supply routing policy lookup failed: %v", err))
		return nil, nil
	}
	if policy == nil || policy.ChannelId <= 0 {
		return nil, miss
	}
	channel, err := model.CacheGetChannel(policy.ChannelId)
	if err != nil || channel == nil {
		logger.LogError(param.Ctx, fmt.Sprintf("supply routing policy channel unavailable: policy_id=%d channel_id=%d err=%v", policy.Id, policy.ChannelId, err))
		return nil, &model.SupplyRoutingPolicyMiss{
			Policy: *policy,
			Group:  group,
			Reason: model.SupplyRoutingPolicyMissReasonChannelMissing,
		}
	}
	if channel.Status != common.ChannelStatusEnabled {
		logger.LogError(param.Ctx, fmt.Sprintf("supply routing policy channel disabled: policy_id=%d channel_id=%d status=%d", policy.Id, policy.ChannelId, channel.Status))
		return nil, &model.SupplyRoutingPolicyMiss{
			Policy: *policy,
			Group:  group,
			Reason: model.SupplyRoutingPolicyMissReasonChannelDisabled,
		}
	}
	if !model.IsChannelSupplierEnabled(channel) {
		logger.LogError(param.Ctx, fmt.Sprintf("supply routing policy supplier unavailable: policy_id=%d channel_id=%d supplier_id=%d", policy.Id, policy.ChannelId, channel.SupplierId))
		return nil, &model.SupplyRoutingPolicyMiss{
			Policy: *policy,
			Group:  group,
			Reason: model.SupplyRoutingPolicyMissReasonSupplierDisabled,
		}
	}
	param.Ctx.Set("supply_routing_policy_id", policy.Id)
	param.Ctx.Set("supply_routing_policy_channel_id", policy.ChannelId)
	param.Ctx.Set("supply_node", channel.Name)
	logger.LogDebug(param.Ctx, "supply routing policy selected channel: policy_id=%d channel_id=%d group=%s model=%s", policy.Id, policy.ChannelId, group, param.ModelName)
	return channel, nil
}

func supplyRoutingPolicyRouteKey(c *gin.Context) string {
	if c == nil {
		return ""
	}
	for _, key := range []string{"X-Session-Id", "X-Session-ID", "session_id", "prompt_cache_key", "X-Prompt-Cache-Key"} {
		if value := strings.TrimSpace(c.GetHeader(key)); value != "" {
			return "session:" + value
		}
	}
	if value := strings.TrimSpace(c.GetString(common.RequestIdKey)); value != "" {
		return "request:" + value
	}
	if value := strings.TrimSpace(c.GetHeader(common.UsageLedgerRequestIdHeader)); value != "" {
		return "request:" + value
	}
	return ""
}

func GetSupplyRoutingPolicyChannelForRequest(c *gin.Context, modelName string, usingGroup string) (*model.Channel, string, *model.SupplyRoutingPolicyMiss) {
	if c == nil || modelName == "" || usingGroup == "" {
		return nil, "", nil
	}
	param := &RetryParam{
		Ctx:       c,
		ModelName: modelName,
	}
	if usingGroup == "auto" {
		userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
		var firstMiss *model.SupplyRoutingPolicyMiss
		for _, group := range GetUserAutoGroup(userGroup) {
			if channel, miss := getSupplyRoutingPolicyChannel(param, group); channel != nil {
				return channel, group, nil
			} else if firstMiss == nil {
				firstMiss = miss
			}
		}
		return nil, "", firstMiss
	}
	if channel, miss := getSupplyRoutingPolicyChannel(param, usingGroup); channel != nil {
		return channel, usingGroup, nil
	} else {
		return nil, "", miss
	}
}

func RecordSupplyRoutingPolicyMissInsight(c *gin.Context, miss *model.SupplyRoutingPolicyMiss) {
	if c == nil || miss == nil {
		return
	}
	insight, err := model.RecordSupplyRoutingPolicyMissInsight(miss, common.GetTimestamp())
	if err != nil {
		logger.LogError(c, fmt.Sprintf("supply routing policy miss insight write failed: policy_id=%d reason=%s err=%v", miss.Policy.Id, miss.Reason, err))
		return
	}
	if insight != nil {
		logger.LogDebug(c, "supply routing policy miss insight recorded: policy_id=%d insight_id=%d reason=%s", miss.Policy.Id, insight.Id, miss.Reason)
	}
}
