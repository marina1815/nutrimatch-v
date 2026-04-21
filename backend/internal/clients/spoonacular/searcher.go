package spoonacular

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/marina1815/nutrimatch/internal/security"
)

var ErrCircuitOpen = errors.New("spoonacular circuit breaker open")

type Searcher interface {
	Search(ctx context.Context, opts SearchOptions) (*SearchResponse, error)
}

type PersistentCache interface {
	Get(ctx context.Context, cacheKey string) (*SearchResponse, bool, error)
	Set(ctx context.Context, cacheKey string, response *SearchResponse, ttl time.Duration) error
}

type ResilientSearcher struct {
	Base          Searcher
	Cache         *security.TTLCache[*SearchResponse]
	Persistent    PersistentCache
	PersistentTTL time.Duration
	MaxRetries    int
	RetryDelay    time.Duration

	CircuitBreakerThreshold int
	CircuitBreakerCooldown  time.Duration

	mu                  sync.Mutex
	consecutiveFailures int
	openedUntil         time.Time
}

func (s *ResilientSearcher) Search(ctx context.Context, opts SearchOptions) (*SearchResponse, error) {
	if s == nil || s.Base == nil {
		return nil, errors.New("spoonacular searcher unavailable")
	}

	cacheKey := searchCacheKey(opts)
	if s.Cache != nil {
		if cached, ok := s.Cache.Get(cacheKey); ok {
			copied := *cached
			copied.CacheHit = true
			return &copied, nil
		}
	}
	if s.Persistent != nil {
		if cached, ok, err := s.Persistent.Get(ctx, cacheKey); err == nil && ok && cached != nil {
			if s.Cache != nil {
				copied := *cached
				copied.CacheHit = false
				s.Cache.Set(cacheKey, &copied)
			}
			copied := *cached
			copied.CacheHit = true
			return &copied, nil
		}
	}
	if s.isCircuitOpen() {
		return nil, ErrCircuitOpen
	}

	attempts := s.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		response, err := s.Base.Search(ctx, opts)
		if err == nil {
			s.recordSuccess()
			if s.Cache != nil && response != nil {
				copied := *response
				copied.CacheHit = false
				s.Cache.Set(cacheKey, &copied)
			}
			if s.Persistent != nil && response != nil && s.PersistentTTL > 0 {
				_ = s.Persistent.Set(ctx, cacheKey, response, s.PersistentTTL)
			}
			return response, nil
		}

		lastErr = err
		s.recordFailure(err)
		if !shouldRetry(err) || attempt == attempts-1 {
			break
		}
		if s.RetryDelay > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(s.RetryDelay):
			}
		}
	}

	return nil, lastErr
}

func (s *ResilientSearcher) isCircuitOpen() bool {
	if s == nil || s.CircuitBreakerThreshold <= 0 || s.CircuitBreakerCooldown <= 0 {
		return false
	}

	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.openedUntil.IsZero() {
		return false
	}
	if now.Before(s.openedUntil) {
		return true
	}

	s.openedUntil = time.Time{}
	s.consecutiveFailures = 0
	return false
}

func (s *ResilientSearcher) recordSuccess() {
	if s == nil || s.CircuitBreakerThreshold <= 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.consecutiveFailures = 0
	s.openedUntil = time.Time{}
}

func (s *ResilientSearcher) recordFailure(err error) {
	if s == nil || s.CircuitBreakerThreshold <= 0 || s.CircuitBreakerCooldown <= 0 {
		return
	}
	if !shouldTripCircuit(err) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.consecutiveFailures++
	if s.consecutiveFailures >= s.CircuitBreakerThreshold {
		s.openedUntil = time.Now().Add(s.CircuitBreakerCooldown)
	}
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	var upstreamErr *UpstreamError
	if errors.As(err, &upstreamErr) {
		return upstreamErr.StatusCode == 429 || upstreamErr.StatusCode >= 500
	}

	return errors.Is(err, ErrUpstreamFailure)
}

func shouldTripCircuit(err error) bool {
	return shouldRetry(err)
}

func searchCacheKey(opts SearchOptions) string {
	payload, _ := json.Marshal(opts)
	return security.SecureCacheKey(string(payload))
}
