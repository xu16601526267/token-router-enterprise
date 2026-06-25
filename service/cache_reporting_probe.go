package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

// CacheReportingVerdict is the advisory result of probing whether a supplier
// upstream honestly reports prefix-cache hits in its OpenAI usage payload.
//
// token-router cost/margin/settlement data is cache-aware: a cached prompt
// token is billed at the agreement CostCacheRatio instead of the full model
// ratio. If an upstream performs prefix caching but does not surface
// cached_tokens in usage.prompt_tokens_details, the platform records
// cached_tokens=0 for every request, silently bills the saved tokens at full
// price, and the cache-driven cost advantage becomes invisible and
// unauditable. This probe lets the supplier admission flow detect that
// condition before the upstream is trusted, keeping the metering data honest
// (product principle P2: no claim without data).
type CacheReportingVerdict struct {
	Probed             bool    `json:"probed"`
	Reported           bool    `json:"reported"`
	Status             string  `json:"status"`
	Model              string  `json:"model"`
	FirstCachedTokens  int     `json:"first_cached_tokens"`
	SecondCachedTokens int     `json:"second_cached_tokens"`
	SecondPromptTokens int     `json:"second_prompt_tokens"`
	SecondCacheRatio   float64 `json:"second_cache_ratio"`
	Reason             string  `json:"reason"`
}

// Verdict statuses for CacheReportingVerdict.Status. Reported is true only for
// CacheReportingHealthy.
const (
	CacheReportingHealthy        = "healthy"
	CacheReportingUnderReporting = "under_reporting"
	CacheReportingNotReported    = "not_reported"
	CacheReportingUnsupported    = "unsupported"
)

// cacheReportingMinHealthyRatio is the minimum share of the (fully repeated)
// second-probe prompt that must come back as cached for the upstream to count
// as honestly reporting. On a verbatim repeat almost the whole prompt is a
// cache hit, so honest upstreams report ~0.9+; a supplier that surfaces only a
// small fraction is systematically under-reporting cache (which understates
// cached tokens and overstates cost), even though cached_tokens > 0.
const cacheReportingMinHealthyRatio = 0.5

// cacheReportingProbePrefix is a deterministic prompt fragment. It is repeated
// to build a prefix long enough that a second identical request will hit an
// upstream prefix cache (KV-cache block sizes are small, but a few hundred
// shared tokens make a hit unambiguous).
const cacheReportingProbePrefix = "You are the cache-reporting admission probe for the token-router supplier selection process. Answer briefly. "

func cacheReportingProbeSupported(channelType int) bool {
	return channelType == constant.ChannelTypeOpenAI
}

// firstChannelKey returns the first key from a channel key field that may hold
// multiple keys separated by newlines or commas.
func firstChannelKey(rawKey string) string {
	key := rawKey
	for _, sep := range []string{"\n", ","} {
		if idx := strings.Index(key, sep); idx >= 0 {
			key = key[:idx]
		}
	}
	return strings.TrimSpace(key)
}

// ProbeChannelCacheReporting sends two identical cache-friendly chat
// completions to the channel upstream and reports whether the second one
// surfaces a prefix-cache hit in usage.prompt_tokens_details.cached_tokens.
// The verdict is advisory: the operator decides whether to admit the supplier.
func ProbeChannelCacheReporting(channel *model.Channel, requestedModel string) (CacheReportingVerdict, error) {
	if channel == nil {
		return CacheReportingVerdict{}, fmt.Errorf("channel is nil")
	}
	if !cacheReportingProbeSupported(channel.Type) {
		return CacheReportingVerdict{
			Probed: false,
			Status: CacheReportingUnsupported,
			Reason: fmt.Sprintf("cache-reporting probe only supports OpenAI-compatible channels, got %s", constant.GetChannelTypeName(channel.Type)),
		}, nil
	}

	probeModel := strings.TrimSpace(requestedModel)
	if probeModel == "" {
		if models := channel.GetModels(); len(models) > 0 {
			probeModel = strings.TrimSpace(models[0])
		}
	}
	if probeModel == "" {
		return CacheReportingVerdict{}, fmt.Errorf("channel has no model to probe")
	}

	baseURL := strings.TrimRight(channel.GetBaseURL(), "/")
	if baseURL == "" {
		return CacheReportingVerdict{}, fmt.Errorf("channel has no base url")
	}
	endpoint := baseURL + "/v1/chat/completions"
	key := firstChannelKey(channel.Key)

	body, err := common.Marshal(map[string]any{
		"model": probeModel,
		"messages": []map[string]string{
			{"role": "system", "content": strings.Repeat(cacheReportingProbePrefix, 16)},
			{"role": "user", "content": "Reply with the single word ok."},
		},
		"max_tokens":  4,
		"temperature": 0,
		"stream":      false,
	})
	if err != nil {
		return CacheReportingVerdict{}, err
	}

	client := &http.Client{Timeout: 60 * time.Second}

	firstCached, _, err := postCacheProbe(client, endpoint, key, body)
	if err != nil {
		return CacheReportingVerdict{}, fmt.Errorf("first probe request failed: %w", err)
	}
	secondCached, secondPrompt, err := postCacheProbe(client, endpoint, key, body)
	if err != nil {
		return CacheReportingVerdict{}, fmt.Errorf("second probe request failed: %w", err)
	}

	verdict := CacheReportingVerdict{
		Probed:             true,
		Model:              probeModel,
		FirstCachedTokens:  firstCached,
		SecondCachedTokens: secondCached,
		SecondPromptTokens: secondPrompt,
	}
	if secondPrompt > 0 {
		verdict.SecondCacheRatio = float64(secondCached) / float64(secondPrompt)
	}
	switch {
	case secondCached <= 0:
		verdict.Status = CacheReportingNotReported
		verdict.Reported = false
		verdict.Reason = "upstream did not report cached_tokens on a repeated-prefix probe; either it does not do prefix caching, or cache reporting is disabled (for example vLLM needs --enable-prompt-tokens-details). cache-aware cost would be recorded as 0 for this supplier"
	case verdict.SecondCacheRatio < cacheReportingMinHealthyRatio:
		verdict.Status = CacheReportingUnderReporting
		verdict.Reported = false
		verdict.Reason = fmt.Sprintf("upstream surfaced cached_tokens but only %d of ~%d repeated prompt tokens (ratio %.2f) on a verbatim-repeat probe; it systematically under-reports cache, which understates cached tokens and overstates cost for this supplier — audit before admission", secondCached, secondPrompt, verdict.SecondCacheRatio)
	default:
		verdict.Status = CacheReportingHealthy
		verdict.Reported = true
		verdict.Reason = "upstream reports prefix-cache hits in usage.prompt_tokens_details.cached_tokens at a plausible ratio; cache-aware cost is measurable"
	}
	return verdict, nil
}

// postCacheProbe sends one probe request and returns the upstream-reported
// cached and prompt token counts.
func postCacheProbe(client *http.Client, endpoint, key string, body []byte) (cachedTokens int, promptTokens int, err error) {
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	if key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, 0, err
	}
	defer response.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return 0, 0, err
	}
	if response.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("upstream returned status %d: %s", response.StatusCode, truncateProbeError(string(respBody)))
	}
	var parsed struct {
		Usage dto.Usage `json:"usage"`
	}
	if err := common.Unmarshal(respBody, &parsed); err != nil {
		return 0, 0, fmt.Errorf("failed to parse upstream usage: %w", err)
	}
	cached := parsed.Usage.PromptTokensDetails.CachedTokens
	if cached == 0 && parsed.Usage.PromptCacheHitTokens > 0 {
		// some OpenAI-compatible upstreams (for example DeepSeek) report cache
		// hits in prompt_cache_hit_tokens instead of prompt_tokens_details.
		cached = parsed.Usage.PromptCacheHitTokens
	}
	return cached, parsed.Usage.PromptTokens, nil
}

func truncateProbeError(s string) string {
	const limit = 256
	s = strings.TrimSpace(s)
	if len(s) > limit {
		return s[:limit] + "..."
	}
	return s
}
