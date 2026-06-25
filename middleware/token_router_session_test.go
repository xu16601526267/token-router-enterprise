package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTokenRouterSessionTestContext(method string, path string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	return ctx, recorder
}

func TestEnsureTokenRouterSessionID_RequestHeader(t *testing.T) {
	ctx, recorder := newTokenRouterSessionTestContext(http.MethodPost, "/v1/chat/completions", `{"model":"gpt-test"}`)
	ctx.Request.Header.Set("X-Session-Id", "session-from-header")

	sessionID := ensureTokenRouterSessionID(ctx)

	require.Equal(t, "session-from-header", sessionID)
	require.Equal(t, "session-from-header", ctx.Request.Header.Get("X-Session-Id"))
	require.Equal(t, "session-from-header", ctx.Request.Header.Get("session_id"))
	require.Equal(t, "session-from-header", recorder.Header().Get("X-Session-Id"))
}

func TestEnsureTokenRouterSessionID_JSONBodyUser(t *testing.T) {
	ctx, recorder := newTokenRouterSessionTestContext(http.MethodPost, "/v1/chat/completions", `{"model":"gpt-test","user":"openai-user-1"}`)
	ctx.Request.Header.Set("Content-Type", "application/json")

	sessionID := ensureTokenRouterSessionID(ctx)

	require.Equal(t, "openai-user-1", sessionID)
	require.Equal(t, "openai-user-1", ctx.Request.Header.Get("X-Session-Id"))
	require.Equal(t, "openai-user-1", ctx.Request.Header.Get("session_id"))
	require.Equal(t, "openai-user-1", recorder.Header().Get("X-Session-Id"))
}

func TestEnsureTokenRouterSessionID_Generated(t *testing.T) {
	ctx, recorder := newTokenRouterSessionTestContext(http.MethodPost, "/v1/chat/completions", `{"model":"gpt-test"}`)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(common.RequestIdKey, "router-req-1")

	sessionID := ensureTokenRouterSessionID(ctx)

	require.Equal(t, "trsess_router-req-1", sessionID)
	require.Equal(t, "trsess_router-req-1", ctx.Request.Header.Get("X-Session-Id"))
	require.Equal(t, "trsess_router-req-1", ctx.Request.Header.Get("session_id"))
	require.Equal(t, "trsess_router-req-1", recorder.Header().Get("X-Session-Id"))
}
