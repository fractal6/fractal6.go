/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package notify

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

// TestNilGate_NoOp verifies the documented "no Redis client" path: every
// method must return cleanly so the email pipeline ships emails as-is when
// the gate is unconfigured.
func TestNilGate_NoOp(t *testing.T) {
	var g *Gate // nil receiver
	ctx := context.Background()
	if err := g.Register(ctx, "0xT", 3); err != nil {
		t.Fatalf("nil Gate Register: %v", err)
	}
	if err := g.Signal(ctx, "0xT"); err != nil {
		t.Fatalf("nil Gate Signal: %v", err)
	}
	// Wait should fall back to baseline-sleep and return true.
	ok, err := g.Wait(ctx, "0xT", 5*time.Millisecond, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("nil Gate Wait: %v", err)
	}
	if !ok {
		t.Fatal("nil Gate Wait: expected ok=true on baseline timeout")
	}
}

// TestGate_RegisterSignalWait exercises the happy path against a real Redis
// when REDIS_ADDR is set (the integration test harness sets this). Skipped
// otherwise so unit-test runs without a Redis don't fail.
func TestGate_RegisterSignalWait(t *testing.T) {
	rdb := redisOrSkip(t)
	defer rdb.Close()
	g := New(rdb)
	ctx := context.Background()
	tid := "test-tid-" + time.Now().Format("150405.000000")

	if err := g.Register(ctx, tid, 2); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := g.Signal(ctx, tid); err != nil {
		t.Fatalf("Signal #1: %v", err)
	}
	if err := g.Signal(ctx, tid); err != nil {
		t.Fatalf("Signal #2: %v", err)
	}
	// Counter should now read 0; Wait returns true immediately after baseline.
	start := time.Now()
	ok, err := g.Wait(ctx, tid, 50*time.Millisecond, 2*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !ok {
		t.Fatal("Wait: expected ok=true after counter drained")
	}
	if time.Since(start) > 1*time.Second {
		t.Errorf("Wait took too long after draining: %v", time.Since(start))
	}
}

// TestGate_Timeout_StuckCounter verifies Wait returns false when the
// counter stays >0 past the deadline. Critical: a stuck upload (browser
// crash before /file/upload fires) MUST NOT pin email delivery.
func TestGate_Timeout_StuckCounter(t *testing.T) {
	rdb := redisOrSkip(t)
	defer rdb.Close()
	g := New(rdb)
	ctx := context.Background()
	tid := "test-stuck-" + time.Now().Format("150405.000000")

	if err := g.Register(ctx, tid, 5); err != nil {
		t.Fatalf("Register: %v", err)
	}
	ok, err := g.Wait(ctx, tid, 10*time.Millisecond, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if ok {
		t.Fatal("Wait: expected timeout (ok=false), got ok=true")
	}
	// Cleanup so the next test run on the same redis isn't haunted.
	_ = rdb.Del(ctx, keyPrefix+tid).Err()
}

// TestGate_BaselineOnly verifies the baseline-sleep happens even when no
// count was registered — that's how plain (non-inline) attachments get a
// chance to land in Comment.files before the notifier reads it.
func TestGate_BaselineOnly(t *testing.T) {
	rdb := redisOrSkip(t)
	defer rdb.Close()
	g := New(rdb)
	ctx := context.Background()
	tid := "test-baseline-" + time.Now().Format("150405.000000")

	start := time.Now()
	ok, err := g.Wait(ctx, tid, 50*time.Millisecond, 1*time.Second)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if !ok {
		t.Fatal("Wait: expected ok=true on absent key")
	}
	if elapsed := time.Since(start); elapsed < 45*time.Millisecond {
		t.Errorf("Wait returned before baseline elapsed: %v", elapsed)
	}
}

// TestGate_SignalOnMissingKey verifies Signal is a no-op when the key has
// already expired (TTL drained, never registered, …). Late signals must not
// drive a fresh register on the same tid into negatives.
func TestGate_SignalOnMissingKey(t *testing.T) {
	rdb := redisOrSkip(t)
	defer rdb.Close()
	g := New(rdb)
	ctx := context.Background()
	tid := "test-missing-" + time.Now().Format("150405.000000")
	if err := g.Signal(ctx, tid); err != nil {
		t.Fatalf("Signal on missing key: %v", err)
	}
	// Now Register and verify the counter starts at +n (not n-1).
	if err := g.Register(ctx, tid, 2); err != nil {
		t.Fatalf("Register: %v", err)
	}
	v, err := rdb.Get(ctx, keyPrefix+tid).Int()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if v != 2 {
		t.Errorf("counter = %d, want 2 (Signal-on-missing must not have decremented)", v)
	}
	_ = rdb.Del(ctx, keyPrefix+tid).Err()
}

// redisOrSkip returns a connected redis client or skips the test when no
// REDIS_ADDR is configured. The integration test harness sets this; CI
// unit-only runs skip these tests cleanly.
func redisOrSkip(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR not set — skipping gate test")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		t.Skipf("redis not reachable at %s: %v", addr, err)
	}
	return rdb
}
