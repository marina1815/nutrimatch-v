package spoonacular

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marina1815/nutrimatch/internal/security"
)

type fakeBaseSearcher struct {
	responses []*SearchResponse
	errors    []error
	calls     int
}

type fakePersistentCache struct {
	response *SearchResponse
	ok       bool
	getCalls int
	setCalls int
}

func (f *fakeBaseSearcher) Search(_ context.Context, _ SearchOptions) (*SearchResponse, error) {
	index := f.calls
	f.calls++
	if index < len(f.responses) && f.responses[index] != nil {
		return f.responses[index], nil
	}
	if index < len(f.errors) {
		return nil, f.errors[index]
	}
	return nil, errors.New("unexpected call")
}

func (f *fakePersistentCache) Get(_ context.Context, _ string) (*SearchResponse, bool, error) {
	f.getCalls++
	if !f.ok || f.response == nil {
		return nil, false, nil
	}
	copied := *f.response
	return &copied, true, nil
}

func (f *fakePersistentCache) Set(_ context.Context, _ string, response *SearchResponse, _ time.Duration) error {
	f.setCalls++
	if response != nil {
		copied := *response
		f.response = &copied
		f.ok = true
	}
	return nil
}

func TestResilientSearcherUsesCache(t *testing.T) {
	base := &fakeBaseSearcher{
		responses: []*SearchResponse{{Results: []Recipe{{ID: 1, Title: "Meal"}}}},
	}

	searcher := &ResilientSearcher{
		Base:  base,
		Cache: security.NewTTLCache[*SearchResponse](time.Minute),
	}

	_, err := searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second, err := searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if base.calls != 1 {
		t.Fatalf("expected single upstream call, got %d", base.calls)
	}
	if !second.CacheHit {
		t.Fatalf("expected cached response on second call")
	}
}

func TestResilientSearcherRetriesRetryableError(t *testing.T) {
	base := &fakeBaseSearcher{
		errors: []error{
			&UpstreamError{StatusCode: 503},
		},
		responses: []*SearchResponse{
			nil,
			{Results: []Recipe{{ID: 2, Title: "Recovered"}}},
		},
	}

	searcher := &ResilientSearcher{
		Base:       base,
		MaxRetries: 1,
	}

	response, err := searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response == nil || len(response.Results) != 1 {
		t.Fatalf("expected successful retry result")
	}
	if base.calls != 2 {
		t.Fatalf("expected two calls, got %d", base.calls)
	}
}

func TestResilientSearcherUsesPersistentCacheBeforeUpstream(t *testing.T) {
	base := &fakeBaseSearcher{}
	store := &fakePersistentCache{
		response: &SearchResponse{Results: []Recipe{{ID: 3, Title: "Persisted"}}},
		ok:       true,
	}

	searcher := &ResilientSearcher{
		Base:       base,
		Persistent: store,
	}

	response, err := searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response == nil || len(response.Results) != 1 {
		t.Fatalf("expected cached persistent response")
	}
	if !response.CacheHit {
		t.Fatalf("expected persistent cache hit")
	}
	if base.calls != 0 {
		t.Fatalf("expected no upstream call when persistent cache hits")
	}
}

func TestResilientSearcherPersistsSuccessfulResponses(t *testing.T) {
	base := &fakeBaseSearcher{
		responses: []*SearchResponse{{Results: []Recipe{{ID: 4, Title: "Stored"}}}},
	}
	store := &fakePersistentCache{}

	searcher := &ResilientSearcher{
		Base:          base,
		Persistent:    store,
		PersistentTTL: 5 * time.Minute,
	}

	_, err := searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.setCalls != 1 {
		t.Fatalf("expected successful response to be persisted once, got %d", store.setCalls)
	}
}

func TestResilientSearcherOpensCircuitAfterRetryableFailures(t *testing.T) {
	base := &fakeBaseSearcher{
		errors: []error{
			&UpstreamError{StatusCode: 503},
			&UpstreamError{StatusCode: 503},
		},
	}

	searcher := &ResilientSearcher{
		Base:                    base,
		CircuitBreakerThreshold: 2,
		CircuitBreakerCooldown:  time.Minute,
	}

	_, err := searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if err == nil {
		t.Fatalf("expected first failure")
	}
	_, err = searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if err == nil {
		t.Fatalf("expected second failure")
	}

	_, err = searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit breaker to open, got %v", err)
	}
	if base.calls != 2 {
		t.Fatalf("expected circuit to stop upstream calls after opening, got %d", base.calls)
	}
}

func TestResilientSearcherClosesCircuitAfterCooldown(t *testing.T) {
	base := &fakeBaseSearcher{
		responses: []*SearchResponse{{Results: []Recipe{{ID: 5, Title: "Recovered"}}}},
	}

	searcher := &ResilientSearcher{
		Base:                    base,
		CircuitBreakerThreshold: 1,
		CircuitBreakerCooldown:  time.Minute,
		openedUntil:             time.Now().Add(-time.Second),
		consecutiveFailures:     1,
	}

	response, err := searcher.Search(context.Background(), SearchOptions{Query: "meal"})
	if err != nil {
		t.Fatalf("expected recovered search after cooldown, got %v", err)
	}
	if response == nil || len(response.Results) != 1 {
		t.Fatalf("expected upstream response after circuit cooldown")
	}
	if base.calls != 1 {
		t.Fatalf("expected upstream to be called after cooldown, got %d", base.calls)
	}
}
