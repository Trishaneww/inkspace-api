package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Lockout struct {
	rdb         redis.Cmdable
	maxFailures int
	window      time.Duration
}

func NewLockout(rdb redis.Cmdable, maxFailures int, window time.Duration) *Lockout {
	return &Lockout{rdb: rdb, maxFailures: maxFailures, window: window}
}

// Locked reports whether `key` has reached the failure threshold and is
// currently within its lock window.
func (l *Lockout) Locked(ctx context.Context, key string) (bool, error) {
	n, err := l.rdb.Get(ctx, key).Int()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return n >= l.maxFailures, nil
}

// RecordFailure increments the failure count (arming the window TTL on the
// first failure) and reports whether the account is now locked.
func (l *Lockout) RecordFailure(ctx context.Context, key string) (bool, error) {
	n, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		// First failure in this window — start the expiry clock.
		_ = l.rdb.Expire(ctx, key, l.window).Err()
	}
	return int(n) >= l.maxFailures, nil
}

// Reset clears the failure count after a successful attempt.
func (l *Lockout) Reset(ctx context.Context, key string) error {
	return l.rdb.Del(ctx, key).Err()
}
