package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// dialTestRedis connects to a local Redis (db 15) or skips the test.
func dialTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	opt, err := redis.ParseURL("redis://localhost:6379/15")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	rdb.FlushDB(context.Background())
	t.Cleanup(func() {
		rdb.FlushDB(context.Background())
		_ = rdb.Close()
	})
	return rdb
}

func TestRedisLimiter_BurstThenBlockThenRefill(t *testing.T) {
	rdb := dialTestRedis(t)
	ctx := context.Background()
	l := NewRedisLimiter(rdb, Rule{RPS: 5, Burst: 3})
	key := "test:1.2.3.4"

	// The full burst is allowed.
	for i := 0; i < 3; i++ {
		res, err := l.CheckLimit(ctx, key)
		if err != nil {
			t.Fatalf("req %d: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("req %d should be allowed within burst", i)
		}
	}

	// The next one over budget is blocked, with a retry hint.
	res, err := l.CheckLimit(ctx, key)
	if err != nil {
		t.Fatalf("blocked req: %v", err)
	}
	if res.Allowed {
		t.Fatal("request beyond burst should be blocked")
	}
	if res.RetryAfter <= 0 {
		t.Fatal("blocked result should carry a Retry-After hint")
	}

	// After ~one refill interval (5 tokens/sec → 1 token per 200ms) it recovers.
	time.Sleep(250 * time.Millisecond)
	res, err = l.CheckLimit(ctx, key)
	if err != nil {
		t.Fatalf("post-refill req: %v", err)
	}
	if !res.Allowed {
		t.Fatal("request should be allowed after refill")
	}
}

func TestRedisLimiter_KeysAreIsolated(t *testing.T) {
	rdb := dialTestRedis(t)
	ctx := context.Background()
	l := NewRedisLimiter(rdb, Rule{RPS: 1, Burst: 1})

	first, err := l.CheckLimit(ctx, "ip:A")
	if err != nil || !first.Allowed {
		t.Fatalf("A should be allowed: allowed=%v err=%v", first.Allowed, err)
	}
	exhausted, err := l.CheckLimit(ctx, "ip:A")
	if err != nil {
		t.Fatal(err)
	}
	if exhausted.Allowed {
		t.Fatal("A should be blocked after consuming its single token")
	}
	other, err := l.CheckLimit(ctx, "ip:B")
	if err != nil || !other.Allowed {
		t.Fatalf("B should be allowed independently: allowed=%v err=%v", other.Allowed, err)
	}
}
