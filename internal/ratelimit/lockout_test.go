package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestLockout_LocksAfterMaxFailures(t *testing.T) {
	rdb := dialTestRedis(t)
	ctx := context.Background()
	lock := NewLockout(rdb, 3, time.Minute)
	key := "login_fail:alice@example.com"

	// Not locked initially.
	if locked, err := lock.Locked(ctx, key); err != nil || locked {
		t.Fatalf("should start unlocked: locked=%v err=%v", locked, err)
	}

	// Three failures hits the threshold.
	for i := 1; i <= 3; i++ {
		nowLocked, err := lock.RecordFailure(ctx, key)
		if err != nil {
			t.Fatalf("failure %d: %v", i, err)
		}
		if i < 3 && nowLocked {
			t.Fatalf("should not be locked after %d/3 failures", i)
		}
		if i == 3 && !nowLocked {
			t.Fatal("should be locked at the 3rd failure")
		}
	}

	if locked, err := lock.Locked(ctx, key); err != nil || !locked {
		t.Fatalf("should report locked: locked=%v err=%v", locked, err)
	}
}

func TestLockout_ResetClearsFailures(t *testing.T) {
	rdb := dialTestRedis(t)
	ctx := context.Background()
	lock := NewLockout(rdb, 2, time.Minute)
	key := "login_fail:bob@example.com"

	if _, err := lock.RecordFailure(ctx, key); err != nil {
		t.Fatal(err)
	}
	if locked, _ := lock.RecordFailure(ctx, key); !locked {
		t.Fatal("should be locked after 2 failures")
	}

	// A successful login clears the slate.
	if err := lock.Reset(ctx, key); err != nil {
		t.Fatal(err)
	}
	if locked, err := lock.Locked(ctx, key); err != nil || locked {
		t.Fatalf("should be unlocked after reset: locked=%v err=%v", locked, err)
	}
}
