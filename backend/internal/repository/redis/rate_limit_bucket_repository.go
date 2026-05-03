package redisrepo

import (
	"context"
	_ "embed"
	"errors"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed token_bucket.lua
var tokenBucketScript string

type RateLimitBucketRepository struct {
	client *redis.Client
	script *redis.Script
}

func NewRateLimitBucketRepository(client *redis.Client) *RateLimitBucketRepository {
	return &RateLimitBucketRepository{
		client: client,
		script: redis.NewScript(tokenBucketScript),
	}
}

func NewClient(rawURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opts), nil
}

func (r *RateLimitBucketRepository) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return errors.New("redis client unavailable")
	}
	return r.client.Ping(ctx).Err()
}

func (r *RateLimitBucketRepository) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *RateLimitBucketRepository) TakeToken(ctx context.Context, key, bucketType string, refillPerSecond float64, burst int, now time.Time) (bool, error) {
	if r == nil || r.client == nil || r.script == nil {
		return false, errors.New("redis rate limiter unavailable")
	}
	if burst <= 0 || refillPerSecond <= 0 {
		return false, nil
	}

	ttlSeconds := math.Ceil(float64(burst)/refillPerSecond*2) + 60
	result, err := r.script.Run(ctx, r.client, []string{"nm:rate:" + bucketType + ":" + key},
		refillPerSecond,
		burst,
		now.UnixMilli(),
		int(ttlSeconds),
	).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
