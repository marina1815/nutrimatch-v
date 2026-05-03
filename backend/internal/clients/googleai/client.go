package googleai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20
const maxRateLimitRetries = 2
const rateLimitRetryDelay = 20 * time.Second

const (
	CodeKeyMissing          = "ai_key_missing"
	CodeNetworkUnreachable  = "ai_network_unreachable"
	CodeDNSError            = "ai_dns_error"
	CodeTimeout             = "ai_timeout"
	CodeAuthFailed          = "ai_auth_failed"
	CodeRateLimited         = "ai_rate_limited"
	CodeUpstreamUnavailable = "ai_upstream_unavailable"
	CodeBadRequest          = "ai_bad_request"
	CodeEmptyResponse       = "ai_empty_response"
	CodeInvalidResponse     = "ai_invalid_response"
	CodeGenerationFailed    = "ai_generation_failed"
)

type Error struct {
	Code       string
	StatusCode int
	Operation  string
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	op := strings.TrimSpace(e.Operation)
	if op == "" {
		op = "request"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("google ai %s failed: %s (status %d)", op, e.Code, e.StatusCode)
	}
	return fmt.Sprintf("google ai %s failed: %s", op, e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) AIErrorCode() string {
	if e == nil {
		return ""
	}
	return e.Code
}

type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

type GenerateRequest struct {
	Contents         []Content        `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

type GenerationConfig struct {
	ResponseMIMEType string         `json:"responseMimeType,omitempty"`
	MaxOutputTokens  int            `json:"maxOutputTokens,omitempty"`
	Temperature      float64        `json:"temperature,omitempty"`
	ThinkingConfig   ThinkingConfig `json:"thinkingConfig,omitempty"`
}

type ThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GenerateResponse struct {
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	Content Content `json:"content"`
}

func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return "", &Error{Code: CodeKeyMissing, Operation: "configuration"}
	}
	if strings.TrimSpace(c.Model) == "" || strings.TrimSpace(c.BaseURL) == "" {
		return "", &Error{Code: CodeBadRequest, Operation: "configuration"}
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 35 * time.Second}
	}
	base := strings.TrimRight(c.BaseURL, "/")
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", base, url.PathEscape(c.Model), url.QueryEscape(c.APIKey))

	payload := GenerateRequest{
		Contents: []Content{{Parts: []Part{{Text: prompt}}}},
		GenerationConfig: GenerationConfig{
			ResponseMIMEType: "application/json",
			MaxOutputTokens:  4096,
			Temperature:      0.2,
			ThinkingConfig:   ThinkingConfig{ThinkingBudget: 0},
		},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return "", &Error{Code: CodeBadRequest, Operation: "encode_request", Err: err}
	}

	var lastStatus int
	for attempt := 0; attempt <= maxRateLimitRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
		if err != nil {
			return "", &Error{Code: CodeBadRequest, Operation: "build_request", Err: err}
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return "", &Error{Code: classifyTransportError(err), Operation: "transport", Err: err}
		}

		if resp.StatusCode >= 400 {
			lastStatus = resp.StatusCode
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRateLimitRetries {
				if err := waitForRetry(ctx, rateLimitRetryDelay); err != nil {
					return "", &Error{Code: classifyTransportError(err), Operation: "retry_wait", Err: err}
				}
				continue
			}
			return "", &Error{Code: classifyStatus(resp.StatusCode), StatusCode: resp.StatusCode, Operation: "upstream_status"}
		}

		var out GenerateResponse
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&out); err != nil {
			_ = resp.Body.Close()
			return "", &Error{Code: CodeInvalidResponse, Operation: "decode_response", Err: err}
		}
		_ = resp.Body.Close()
		if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
			return "", &Error{Code: CodeEmptyResponse, Operation: "parse_response"}
		}
		text := strings.TrimSpace(out.Candidates[0].Content.Parts[0].Text)
		if text == "" {
			return "", &Error{Code: CodeEmptyResponse, Operation: "parse_response"}
		}
		return text, nil
	}
	return "", &Error{Code: classifyStatus(lastStatus), StatusCode: lastStatus, Operation: "upstream_status"}
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func classifyTransportError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return CodeTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return CodeTimeout
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return CodeDNSError
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return CodeNetworkUnreachable
	}
	return CodeNetworkUnreachable
}

func classifyStatus(status int) string {
	switch {
	case status == http.StatusBadRequest:
		return CodeBadRequest
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return CodeAuthFailed
	case status == http.StatusTooManyRequests:
		return CodeRateLimited
	case status >= 500:
		return CodeUpstreamUnavailable
	default:
		return CodeGenerationFailed
	}
}
