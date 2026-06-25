package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// ProbeChannelCache reports whether a channel upstream honestly surfaces
// prefix-cache hits in its OpenAI usage payload. The operator runs it during
// supplier admission so cache-aware cost/settlement data stays trustworthy: an
// upstream that performs prefix caching but never reports cached_tokens would
// otherwise be metered as if nothing was cached, silently inflating recorded
// cost and corrupting margin and reconciliation data.
func ProbeChannelCache(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	verdict, err := service.ProbeChannelCacheReporting(channel, c.Query("model"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    verdict,
	})
}
