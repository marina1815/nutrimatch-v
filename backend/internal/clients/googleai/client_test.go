package googleai

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGenerateTextClassifiesConfigurationAndHTTPFailures(t *testing.T) {
	_, err := (&Client{BaseURL: "https://example.test", Model: "gemini-test"}).GenerateText(context.Background(), "prompt")
	assertAIErrorCode(t, err, CodeKeyMissing)

	cases := map[int]string{
		http.StatusBadRequest:          CodeBadRequest,
		http.StatusUnauthorized:        CodeAuthFailed,
		http.StatusForbidden:           CodeAuthFailed,
		http.StatusTooManyRequests:     CodeRateLimited,
		http.StatusInternalServerError: CodeUpstreamUnavailable,
	}
	for status, expectedCode := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))
		t.Cleanup(server.Close)

		_, err := (&Client{BaseURL: server.URL, APIKey: "secret-key", Model: "gemini-test"}).GenerateText(context.Background(), "prompt")
		assertAIErrorCode(t, err, expectedCode)
		if strings.Contains(err.Error(), "secret-key") {
			t.Fatalf("AI error leaked API key: %v", err)
		}
	}
}

func TestGenerateTextClassifiesTransportAndResponseFailures(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{name: "dns", err: &net.DNSError{Name: "generativelanguage.googleapis.com"}, code: CodeDNSError},
		{name: "timeout", err: context.DeadlineExceeded, code: CodeTimeout},
		{name: "network", err: errors.New("connection refused"), code: CodeNetworkUnreachable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, tc.err
			})}
			_, err := (&Client{BaseURL: "https://example.test", APIKey: "secret-key", Model: "gemini-test", HTTP: httpClient}).GenerateText(context.Background(), "prompt")
			assertAIErrorCode(t, err, tc.code)
		})
	}

	invalidJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{"))
	}))
	defer invalidJSON.Close()
	_, err := (&Client{BaseURL: invalidJSON.URL, APIKey: "secret-key", Model: "gemini-test"}).GenerateText(context.Background(), "prompt")
	assertAIErrorCode(t, err, CodeInvalidResponse)

	emptyResponse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[]}`))
	}))
	defer emptyResponse.Close()
	_, err = (&Client{BaseURL: emptyResponse.URL, APIKey: "secret-key", Model: "gemini-test"}).GenerateText(context.Background(), "prompt")
	assertAIErrorCode(t, err, CodeEmptyResponse)
}

func assertAIErrorCode(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected AI error code %q, got nil", expected)
	}
	var coded interface{ AIErrorCode() string }
	if !errors.As(err, &coded) {
		t.Fatalf("expected coded AI error, got %T: %v", err, err)
	}
	if coded.AIErrorCode() != expected {
		t.Fatalf("expected AI error code %q, got %q (%v)", expected, coded.AIErrorCode(), err)
	}
}
