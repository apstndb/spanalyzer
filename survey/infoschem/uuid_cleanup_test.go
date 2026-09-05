package infoschem_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func uuidFixtureDropStatement(tableName string) string {
	return "DROP TABLE " + tableName
}

func runUUIDFixtureFallbackCleanup(created *bool, tableName string, timeout time.Duration, update func(ctx context.Context, statement string) error) error {
	if created == nil || !*created {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return update(ctx, uuidFixtureDropStatement(tableName))
}

func TestUUIDFixtureFallbackCleanupUsesFreshBoundedContext(t *testing.T) {
	primary, cancelPrimary := context.WithCancel(context.Background())
	cancelPrimary()
	if primary.Err() == nil {
		t.Fatal("primary context is still live")
	}

	const tableName = "UUIDDefaultProbe_cleanup"
	const timeout = 5 * time.Second
	created := true
	var got context.Context
	var statements []string
	if err := runUUIDFixtureFallbackCleanup(&created, tableName, timeout, func(ctx context.Context, statement string) error {
		got = ctx
		statements = append(statements, statement)
		if ctx.Err() != nil {
			t.Errorf("fallback context already done: %v", ctx.Err())
		}
		if ctx == primary {
			t.Error("fallback reused the canceled primary context")
		}
		if err := primary.Err(); err == nil {
			t.Error("primary context should remain canceled")
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("fallback context has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > timeout {
			t.Fatalf("fallback remaining deadline = %v, want (0, %v]", remaining, timeout)
		}
		return nil
	}); err != nil {
		t.Fatalf("runUUIDFixtureFallbackCleanup() error = %v", err)
	}
	if got == nil {
		t.Fatal("fallback update was not called")
	}
	if got, want := statements, []string{uuidFixtureDropStatement(tableName)}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("fallback statements = %#v, want %#v", statements, want)
	}
}

func TestUUIDFixtureFallbackCleanupSkipsWhenCreateDidNotComplete(t *testing.T) {
	created := false
	var calls int
	if err := runUUIDFixtureFallbackCleanup(&created, "UUIDDefaultProbe_skip", time.Second, func(context.Context, string) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("runUUIDFixtureFallbackCleanup() error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("fallback DROP calls = %d, want 0", calls)
	}
}

func TestUUIDFixtureFallbackCleanupSkipsAfterExplicitCleanup(t *testing.T) {
	created := true
	if !created {
		t.Fatal("created guard should be set after CREATE")
	}
	created = false
	var calls int
	if err := runUUIDFixtureFallbackCleanup(&created, "UUIDDefaultProbe_explicit", time.Second, func(context.Context, string) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("runUUIDFixtureFallbackCleanup() error = %v", err)
	}
	if calls != 0 {
		t.Fatalf("deferred DROP calls = %d, want 0 after explicit cleanup", calls)
	}
}

func TestUUIDFixtureFallbackCleanupErrorStaysDistinctFromPrimaryFailure(t *testing.T) {
	created := true
	primaryErr := errors.New("create managed-Spanner UUID fixture: primary failed")
	cleanupErr := runUUIDFixtureFallbackCleanup(&created, "UUIDDefaultProbe_err", time.Second, func(context.Context, string) error {
		return errors.New("ddl timeout")
	})
	if cleanupErr == nil {
		t.Fatal("runUUIDFixtureFallbackCleanup() error = nil, want cleanup failure")
	}
	if primaryErr == nil || !strings.Contains(primaryErr.Error(), "create managed-Spanner UUID fixture") {
		t.Fatalf("primary failure was lost: %v", primaryErr)
	}
	if strings.Contains(cleanupErr.Error(), primaryErr.Error()) {
		t.Fatalf("cleanup error aggregated the primary failure: %v", cleanupErr)
	}
	if !strings.Contains(cleanupErr.Error(), "ddl timeout") {
		t.Fatalf("cleanup error = %v, want ddl timeout", cleanupErr)
	}
}
