// Package ratelimit provides a distributed, per-key token-bucket rate limiter
// backed by Redis. All API instances share the same buckets, so a limit of
// "N requests" is enforced across the whole fleet, not per process.
package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter decides whether a request identified by `key` may proceed.
type Limiter interface {
	CheckLimit(ctx context.Context, key string) (Result, error)
}

type Result struct {
	Allowed    bool
	RetryAfter time.Duration // hint for the Retry-After header when blocked
}

// Rule is a token bucket: it refills at RPS tokens/second up to Burst capacity,
// so it permits short bursts up to Burst then settles to RPS sustained.
type Rule struct {
	RPS   float64
	Burst int
}

// tokenBucket is an atomic refill-and-consume executed entirely inside Redis,
// so concurrent requests across instances can't race the read/modify/write.
// KEYS[1]=bucket  ARGV: rate, burst, now(ms), cost → returns {allowed, retryMs}.
const tokenBucket = `
local rate  = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now   = tonumber(ARGV[3])
local cost  = tonumber(ARGV[4])

local bucket = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(bucket[1])
local ts     = tonumber(bucket[2])
if tokens == nil then
  tokens = burst
  ts = now
end

local elapsed = now - ts
if elapsed < 0 then elapsed = 0 end
tokens = math.min(burst, tokens + (elapsed / 1000.0) * rate)

local allowed = 0
local retry_after = 0
if tokens >= cost then
  allowed = 1
  tokens = tokens - cost
else
  retry_after = math.ceil(((cost - tokens) / rate) * 1000.0)
end

redis.call('HMSET', KEYS[1], 'tokens', tokens, 'ts', now)
-- Expire idle buckets once they've had time to fully refill.
local ttl = math.ceil(burst / rate) + 1
redis.call('PEXPIRE', KEYS[1], ttl * 1000)

return {allowed, retry_after}
`

type RedisLimiter struct {
	rdb    redis.Scripter
	script *redis.Script
	rule   Rule
}

func NewRedisLimiter(rdb redis.Scripter, rule Rule) *RedisLimiter {
	return &RedisLimiter{
		rdb:    rdb,
		script: redis.NewScript(tokenBucket),
		rule:   rule,
	}
}

func (l *RedisLimiter) CheckLimit(ctx context.Context, key string) (Result, error) {
	now := time.Now().UnixMilli()
	out, err := l.script.Run(
		ctx, l.rdb, []string{key},
		l.rule.RPS, l.rule.Burst, now, 1,
	).Int64Slice()
	if err != nil {
		return Result{}, err
	}

	res := Result{Allowed: len(out) > 0 && out[0] == 1}
	if len(out) > 1 && out[1] > 0 {
		res.RetryAfter = time.Duration(out[1]) * time.Millisecond
	}
	return res, nil
}
