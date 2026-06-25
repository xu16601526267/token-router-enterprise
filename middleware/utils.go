package middleware

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func abortWithOpenAiMessage(c *gin.Context, statusCode int, message string, code ...types.ErrorCode) {
	codeStr := ""
	if len(code) > 0 {
		codeStr = string(code[0])
	}
	userId := c.GetInt("id")
	recordRelayAbortErrorLog(c, statusCode, message, codeStr)
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": common.MessageWithRequestId(message, c.GetString(common.RequestIdKey)),
			"type":    "new_api_error",
			"code":    codeStr,
		},
	})
	c.Abort()
	logger.LogError(c.Request.Context(), fmt.Sprintf("user %d | %s", userId, message))
}

func recordRelayAbortErrorLog(c *gin.Context, statusCode int, message string, code string) {
	if !constant.ErrorLogEnabled {
		return
	}
	if routeTag, ok := c.Get(RouteTagKey); !ok || routeTag != "relay" {
		return
	}
	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")
	if userId == 0 || tokenId == 0 {
		return
	}

	other := map[string]interface{}{
		"error_type":  "new_api_error",
		"error_code":  code,
		"status_code": statusCode,
		"source":      "middleware_abort",
	}
	if c.Request != nil && c.Request.URL != nil {
		other["request_path"] = c.Request.URL.Path
	}

	startTime := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
	if startTime.IsZero() {
		startTime = time.Now()
	}
	model.RecordErrorLog(c, userId, common.GetContextKeyInt(c, constant.ContextKeyChannelId),
		common.GetContextKeyString(c, constant.ContextKeyOriginalModel), c.GetString("token_name"),
		common.MaskSensitiveInfo(message), tokenId, int(time.Since(startTime).Seconds()),
		common.GetContextKeyBool(c, constant.ContextKeyIsStream),
		common.GetContextKeyString(c, constant.ContextKeyUsingGroup), other)
}

func abortWithMidjourneyMessage(c *gin.Context, statusCode int, code int, description string) {
	c.JSON(statusCode, gin.H{
		"description": description,
		"type":        "new_api_error",
		"code":        code,
	})
	c.Abort()
	logger.LogError(c.Request.Context(), description)
}
