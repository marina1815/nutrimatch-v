package security

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type TokenBucketStore interface {
	TakeToken(ctx context.Context, key, bucketType string, refillPerSecond float64, burst int, now time.Time) (bool, error)
}

type InMemoryTokenBucketStore struct {
	mu      sync.Mutex
	buckets map[string]bucketState
}

type bucketState struct {
	tokens    float64
	updatedAt time.Time
}

type QuotaManager struct {
	store      TokenBucketStore
	limit      rate.Limit
	burst      int
	bucketType string
	keyPrefix  string
}

func NewQuotaManager(limit rate.Limit, burst int) *QuotaManager {
	return &QuotaManager{
		store:      NewInMemoryTokenBucketStore(),
		limit:      limit,
		burst:      burst,
		bucketType: "recommendation_quota",
		keyPrefix:  "recommendation_quota",
	}
}

func NewPersistentQuotaManager(store TokenBucketStore, limit rate.Limit, burst int) *QuotaManager {
	if store == nil {
		store = NewInMemoryTokenBucketStore()
	}
	return &QuotaManager{
		store:      store,
		limit:      limit,
		burst:      burst,
		bucketType: "recommendation_quota",
		keyPrefix:  "recommendation_quota",
	}
}

func NewInMemoryTokenBucketStore() *InMemoryTokenBucketStore {
	return &InMemoryTokenBucketStore{
		buckets: make(map[string]bucketState),
	}
}

func (m *QuotaManager) Allow(ctx context.Context, subject string) (bool, error) {
	if m == nil || m.store == nil {
		return true, nil
	}
	key := SecureCacheKey(m.keyPrefix, subject)
	return m.store.TakeToken(ctx, key, m.bucketType, float64(m.limit), m.burst, time.Now())
}

func (s *InMemoryTokenBucketStore) TakeToken(_ context.Context, key, _ string, refillPerSecond float64, burst int, now time.Time) (bool, error) {
	if s == nil {
		return false, nil
	}
	if burst <= 0 || refillPerSecond <= 0 {
		return false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.buckets[key]
	if !ok {
		s.buckets[key] = bucketState{
			tokens:    float64(burst - 1),
			updatedAt: now,
		}
		return true, nil
	}

	tokens := refillTokens(state.tokens, state.updatedAt, now, refillPerSecond, burst)
	if tokens < 1 {
		s.buckets[key] = bucketState{
			tokens:    tokens,
			updatedAt: now,
		}
		return false, nil
	}

	s.buckets[key] = bucketState{
		tokens:    tokens - 1,
		updatedAt: now,
	}
	return true, nil
}

func (s *InMemoryTokenBucketStore) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets = make(map[string]bucketState)
}

func refillTokens(current float64, updatedAt, now time.Time, refillPerSecond float64, burst int) float64 {
	if burst <= 0 {
		return 0
	}
	if current < 0 {
		current = 0
	}
	maxTokens := float64(burst)
	if now.After(updatedAt) {
		current += now.Sub(updatedAt).Seconds() * refillPerSecond
	}
	if current > maxTokens {
		current = maxTokens
	}
	return current
}
