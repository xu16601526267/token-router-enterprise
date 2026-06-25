package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProbeChannel(baseURL string) *model.Channel {
	url := baseURL
	return &model.Channel{
		Type:    constant.ChannelTypeOpenAI,
		Key:     "sk-probe",
		BaseURL: &url,
		Models:  "probe-model",
	}
}

// An upstream that surfaces a near-full prefix-cache hit on the second
// identical request must be reported as cache-measurable.
func TestProbeChannelCacheReporting_ReportsCacheHit(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer sk-probe", r.Header.Get("Authorization"))
		calls++
		cached := 0
		if calls > 1 {
			cached = 560
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"usage":{"prompt_tokens":600,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":%d}}}`, cached)
	}))
	defer server.Close()

	verdict, err := ProbeChannelCacheReporting(newProbeChannel(server.URL), "")
	require.NoError(t, err)
	assert.True(t, verdict.Probed)
	assert.Equal(t, CacheReportingHealthy, verdict.Status)
	assert.True(t, verdict.Reported)
	assert.Equal(t, 0, verdict.FirstCachedTokens)
	assert.Equal(t, 560, verdict.SecondCachedTokens)
	assert.Equal(t, "probe-model", verdict.Model)
	assert.Equal(t, 2, calls)
}

// vLLM default behaviour: prefix caching happens internally but cached_tokens
// is never surfaced. The probe must flag this so the operator does not admit a
// supplier whose cache savings would be metered as zero.
func TestProbeChannelCacheReporting_DoesNotReportCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"usage":{"prompt_tokens":600,"completion_tokens":4}}`)
	}))
	defer server.Close()

	verdict, err := ProbeChannelCacheReporting(newProbeChannel(server.URL), "probe-model")
	require.NoError(t, err)
	assert.True(t, verdict.Probed)
	assert.Equal(t, CacheReportingNotReported, verdict.Status)
	assert.False(t, verdict.Reported)
	assert.Equal(t, 0, verdict.SecondCachedTokens)
	assert.Contains(t, verdict.Reason, "did not report cached_tokens")
}

// A dishonest supplier that surfaces cached_tokens but systematically
// under-reports (here 10% of a verbatim-repeat prompt) would still pass a
// naive cached>0 check while inflating recorded cost. The ratio check must
// flag it as under-reporting.
func TestProbeChannelCacheReporting_UnderReporting(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		cached := 0
		if calls > 1 {
			cached = 60 // ~10% of 600 prompt tokens, far below a real verbatim repeat
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"usage":{"prompt_tokens":600,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":%d}}}`, cached)
	}))
	defer server.Close()

	verdict, err := ProbeChannelCacheReporting(newProbeChannel(server.URL), "")
	require.NoError(t, err)
	assert.True(t, verdict.Probed)
	assert.Equal(t, CacheReportingUnderReporting, verdict.Status)
	assert.False(t, verdict.Reported)
	assert.Equal(t, 60, verdict.SecondCachedTokens)
	assert.InDelta(t, 0.1, verdict.SecondCacheRatio, 0.001)
	assert.Contains(t, verdict.Reason, "under-report")
}

// Upstreams that report cache hits in prompt_cache_hit_tokens (for example
// DeepSeek) must also count as cache-measurable.
func TestProbeChannelCacheReporting_PromptCacheHitTokensFallback(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		hit := 0
		if calls > 1 {
			hit = 540
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"usage":{"prompt_tokens":600,"completion_tokens":4,"prompt_cache_hit_tokens":%d}}`, hit)
	}))
	defer server.Close()

	verdict, err := ProbeChannelCacheReporting(newProbeChannel(server.URL), "")
	require.NoError(t, err)
	assert.Equal(t, CacheReportingHealthy, verdict.Status)
	assert.True(t, verdict.Reported)
	assert.Equal(t, 540, verdict.SecondCachedTokens)
}

func TestProbeChannelCacheReporting_UnsupportedChannelType(t *testing.T) {
	ch := newProbeChannel("http://example.invalid")
	ch.Type = constant.ChannelTypeMidjourney
	verdict, err := ProbeChannelCacheReporting(ch, "")
	require.NoError(t, err)
	assert.False(t, verdict.Probed)
	assert.Equal(t, CacheReportingUnsupported, verdict.Status)
	assert.Contains(t, verdict.Reason, "OpenAI-compatible")
}

func TestProbeChannelCacheReporting_UpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error":"boom"}`)
	}))
	defer server.Close()

	_, err := ProbeChannelCacheReporting(newProbeChannel(server.URL), "")
	require.Error(t, err)
}

func TestFirstChannelKey(t *testing.T) {
	assert.Equal(t, "sk-one", firstChannelKey("sk-one"))
	assert.Equal(t, "sk-one", firstChannelKey("sk-one\nsk-two"))
	assert.Equal(t, "sk-one", firstChannelKey("sk-one,sk-two"))
	assert.Equal(t, "sk-one", firstChannelKey("  sk-one  \nsk-two"))
}
