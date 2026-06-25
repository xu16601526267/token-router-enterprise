package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func withTightGlobalAPIRateLimit(t *testing.T) {
	t.Helper()

	previousRedisEnabled := common.RedisEnabled
	previousEnabled := common.GlobalApiRateLimitEnable
	previousNum := common.GlobalApiRateLimitNum
	previousDuration := common.GlobalApiRateLimitDuration

	common.RedisEnabled = false
	common.GlobalApiRateLimitEnable = true
	common.GlobalApiRateLimitNum = 1
	common.GlobalApiRateLimitDuration = 3600

	t.Cleanup(func() {
		common.RedisEnabled = previousRedisEnabled
		common.GlobalApiRateLimitEnable = previousEnabled
		common.GlobalApiRateLimitNum = previousNum
		common.GlobalApiRateLimitDuration = previousDuration
	})
}

func newGlobalAPIRateLimitTestRouter(routePath string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GlobalAPIRateLimit())
	router.GET(routePath, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

func performRateLimitRequest(router *gin.Engine, path string, remoteAddr string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = remoteAddr
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestGlobalAPIRateLimitExemptsDownstreamTokenAccountingEndpoints(t *testing.T) {
	withTightGlobalAPIRateLimit(t)

	testCases := []struct {
		name string
		path string
	}{
		{name: "token logs", path: "/api/log/token"},
		{name: "token usage", path: "/api/usage/token/"},
	}

	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := newGlobalAPIRateLimitTestRouter(testCase.path)
			remoteAddr := "203.0.113." + string(rune('1'+index)) + ":12345"

			first := performRateLimitRequest(router, testCase.path, remoteAddr)
			second := performRateLimitRequest(router, testCase.path, remoteAddr)

			require.Equal(t, http.StatusOK, first.Code)
			require.Equal(t, http.StatusOK, second.Code)
		})
	}
}

func TestGlobalAPIRateLimitStillLimitsRegularAPIRoutes(t *testing.T) {
	withTightGlobalAPIRateLimit(t)

	router := newGlobalAPIRateLimitTestRouter("/api/status")
	remoteAddr := "203.0.113.50:12345"

	first := performRateLimitRequest(router, "/api/status", remoteAddr)
	second := performRateLimitRequest(router, "/api/status", remoteAddr)

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusTooManyRequests, second.Code)
}
