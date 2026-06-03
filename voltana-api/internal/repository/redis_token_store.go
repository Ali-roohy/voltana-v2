package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisTokenStore implements service.TokenStore using Redis.
type RedisTokenStore struct {
	rdb *redis.Client
}

func NewRedisTokenStore(rdb *redis.Client) *RedisTokenStore {
	return &RedisTokenStore{rdb: rdb}
}

func (s *RedisTokenStore) StoreRefreshToken(ctx context.Context, jti, userID string, ttl time.Duration) error {
	return s.rdb.Set(ctx, "refresh:"+jti, userID, ttl).Err()
}

// ConsumeRefreshToken atomically fetches and deletes the refresh token in a
// single round-trip (GETDEL). This guarantees single-use: under concurrent
// refresh attempts with the same token, exactly one caller receives the userID
// and every other caller gets a not-found error — preventing replay/forking.
func (s *RedisTokenStore) ConsumeRefreshToken(ctx context.Context, jti string) (string, error) {
	val, err := s.rdb.GetDel(ctx, "refresh:"+jti).Result()
	if errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("token not found")
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

func (s *RedisTokenStore) DeleteRefreshToken(ctx context.Context, jti string) error {
	return s.rdb.Del(ctx, "refresh:"+jti).Err()
}

// ── generic cache (analytics dashboard, TASK-0008) ──────────────────────────────

// CacheGet returns the cached value and ok=false on a miss (redis.Nil is not an error).
func (s *RedisTokenStore) CacheGet(ctx context.Context, key string) (string, bool, error) {
	val, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (s *RedisTokenStore) CacheSet(ctx context.Context, key, val string, ttl time.Duration) error {
	return s.rdb.Set(ctx, key, val, ttl).Err()
}

func (s *RedisTokenStore) CacheDel(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, key).Err()
}

// rateLimitScript performs INCR and the first-attempt EXPIRE as one atomic
// operation, so a crash/disconnect can never leave a counter without a TTL
// (which would otherwise lock an IP out permanently). Returns the new count.
var rateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`)

// CheckRateLimit increments the attempt counter for key and returns true if
// the caller is still within the allowed limit. The counter and its TTL are
// set atomically (fixed, not sliding, window).
func (s *RedisTokenStore) CheckRateLimit(ctx context.Context, key string, max int64, window time.Duration) (bool, error) {
	count, err := rateLimitScript.Run(ctx, s.rdb, []string{key}, int64(window.Seconds())).Int64()
	if err != nil {
		return false, fmt.Errorf("rate limit: %w", err)
	}
	return count <= max, nil
}
