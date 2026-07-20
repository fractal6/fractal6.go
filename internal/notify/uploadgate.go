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

// Package notify holds cross-process coordination primitives shared between
// the api server and the notifier daemon (cobra subcommands at cmd/root.go).
//
// uploadgate.go is a Redis-backed gate keyed by tension uid. The api server
// counts the inline-paste references in a freshly added/updated comment and
// pre-Registers that many "pending uploads" on the gate before publishing the
// notification. Each subsequent POST /file/upload whose rewrite matched the
// inline filename Signal()s the gate. The notifier Wait()s until all expected
// uploads have arrived (or a hard timeout elapses) before fetching the
// comment's file list and shipping the email — so pasted images make it into
// the outgoing CID attachments.
//
// The gate is best-effort: when the api and notifier processes don't share a
// Redis (see cmd/notifier.go hardcoded localhost:6379 caveat in
// docs/file-storage.md), Wait simply times out and emails ship as they do
// today (broken inline <img>). No-op-on-failure is the design.
package notify

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// keyPrefix is the Redis key namespace. We use a single counter per tension;
// concurrent comments on the same tension share the gate, which only ever
// makes Wait return slightly later — never sooner — so the trade-off is safe.
const keyPrefix = "upload-gate:"

// gateTTL bounds the lifetime of a registered gate: if uploads never arrive
// (browser crash, network drop), the key expires and Wait stops blocking on
// the next tick. 5 minutes is well beyond the worst-case upload latency for
// a 10 MiB asset over a poor connection.
const gateTTL = 300 * time.Second

// pollInterval is how often Wait re-reads the counter. Net QPS is bounded
// by upload_gate_timeout_sec / pollInterval — for a 30s timeout that's 60
// reads, negligible against the noise floor of the api/notifier processes.
const pollInterval = 500 * time.Millisecond

// Gate is a Redis-backed pending-uploads counter shared between the api and
// notifier processes.
type Gate struct {
	rdb *redis.Client
}

// New wraps the given redis client. The client is expected to be shared with
// the rest of the codebase (graph/dgraph_resolver.go::cache for the api,
// cmd/notifier.go::cache for the daemon).
func New(rdb *redis.Client) *Gate {
	return &Gate{rdb: rdb}
}

// Register adds n pending uploads to the gate for tid, refreshing the TTL.
// No-op when n <= 0 or when the underlying redis client is unset (tests).
//
// Always paired with EXPIRE so a crashed registration can't pin the gate
// forever — the TTL guarantees liveness even when nobody signals.
func (g *Gate) Register(ctx context.Context, tid string, n int) error {
	if g == nil || g.rdb == nil || n <= 0 || tid == "" {
		return nil
	}
	key := keyPrefix + tid
	pipe := g.rdb.TxPipeline()
	pipe.IncrBy(ctx, key, int64(n))
	pipe.Expire(ctx, key, gateTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// signalScript decrements the counter only when the key exists, atomically.
// A plain Exists+Decr pair races with itself: two concurrent signals both
// pass the Exists check and drive the counter negative (masking a fresh
// Register on the same tid), and a Decr landing just after TTL expiry
// recreates the key with no expiry.
var signalScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 1 then
	return redis.call("DECR", KEYS[1])
end
return nil
`)

// Signal decrements the pending counter for tid. No-op when the key is
// already gone (TTL expired or never registered) so Signals from late
// uploads can't drive the counter negative in a way that would mask a fresh
// Register on the same tid.
func (g *Gate) Signal(ctx context.Context, tid string) error {
	if g == nil || g.rdb == nil || tid == "" {
		return nil
	}
	return signalScript.Run(ctx, g.rdb, []string{keyPrefix + tid}).Err()
}

// Wait blocks until the gate for tid reaches zero (or is absent), the
// timeout elapses, or ctx is cancelled. Always sleeps `baseline` first to
// give regular (non-inline) attachments a chance to land in Comment.files
// even when no inline pastes seeded a count — without this, plain-only
// comments would race the notifier and lose attachments.
//
// Returns (true, nil) when the gate drained (or never existed), (false, nil)
// on timeout. ctx errors are surfaced as (false, err).
func (g *Gate) Wait(ctx context.Context, tid string, baseline, timeout time.Duration) (bool, error) {
	if g == nil || g.rdb == nil || tid == "" {
		// No gate available — apply the baseline and report success so the
		// caller proceeds; the email will use whatever is in Comment.files
		// at that moment.
		return baselineSleep(ctx, baseline), nil
	}
	if !baselineSleep(ctx, baseline) {
		return false, ctx.Err()
	}

	key := keyPrefix + tid
	deadline := time.Now().Add(timeout)
	for {
		drained, err := readGate(ctx, g.rdb, key)
		if err != nil {
			return false, err
		}
		if drained {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// baselineSleep waits for d (or returns early on ctx cancel). Returns true
// when the full duration elapsed, false when ctx was cancelled.
func baselineSleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// readGate returns true when the gate has drained (key absent or counter <= 0).
func readGate(ctx context.Context, rdb *redis.Client, key string) (bool, error) {
	v, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		// Treat non-numeric value as drained: a foreign writer is using the
		// key, which we'd rather not block on indefinitely.
		return true, nil
	}
	return n <= 0, nil
}
