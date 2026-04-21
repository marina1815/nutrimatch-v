package security

import (
	"context"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestInMemoryTokenBucketStoreBlocksAfterBurst(t *testing.T) {
	store := NewInMemoryTokenBucketStore()
	now := time.Now()

	allowed, err := store.TakeToken(context.Background(), "bucket", "test", 1, 2, now)
	if err != nil || !allowed {
		t.Fatalf("expected first token to pass, got allowed=%v err=%v", allowed, err)
	}

	allowed, err = store.TakeToken(context.Background(), "bucket", "test", 1, 2, now)
	if err != nil || !allowed {
		t.Fatalf("expected second token to pass, got allowed=%v err=%v", allowed, err)
	}

	allowed, err = store.TakeToken(context.Background(), "bucket", "test", 1, 2, now)
	if err != nil {
		t.Fatalf("unexpected error on third token: %v", err)
	}
	if allowed {
		t.Fatalf("expected third token to be blocked")
	}
}

func TestInMemoryTokenBucketStoreRefillsOverTime(t *testing.T) {
	store := NewInMemoryTokenBucketStore()
	now := time.Now()

	_, _ = store.TakeToken(context.Background(), "bucket", "test", 1, 1, now)
	allowed, err := store.TakeToken(context.Background(), "bucket", "test", 1, 1, now.Add(1200*time.Millisecond))
	if err != nil {
		t.Fatalf("unexpected error after refill: %v", err)
	}
	if !allowed {
		t.Fatalf("expected bucket to refill after enough time")
	}
}

func TestQuotaManagerUsesConfiguredStore(t *testing.T) {
	quota := NewQuotaManager(rate.Every(time.Hour), 1)

	first, err := quota.Allow(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error on first quota check: %v", err)
	}
	second, err := quota.Allow(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error on second quota check: %v", err)
	}

	if !first || second {
		t.Fatalf("expected quota to allow once and then block, got first=%v second=%v", first, second)
	}
}
