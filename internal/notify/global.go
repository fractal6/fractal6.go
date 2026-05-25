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
	"sync/atomic"

	"github.com/go-redis/redis/v8"
)

// global holds the process-wide Gate registered by SetGlobal. May be nil
// when no Redis client has been configured (e.g. unit tests); callers MUST
// nil-check via Global() and treat absence as best-effort no-op.
var global atomic.Pointer[Gate]

// SetGlobal registers the process-wide Gate. Called once during process
// startup from the api server and the notifier daemon — they each pass in
// the same shared go-redis client they already use for pubsub / sessions.
func SetGlobal(g *Gate) { global.Store(g) }

// SetGlobalClient is a convenience that wraps `rdb` in a Gate and registers
// it. Equivalent to SetGlobal(New(rdb)).
func SetGlobalClient(rdb *redis.Client) { global.Store(New(rdb)) }

// Global returns the registered Gate, or nil. Methods on *Gate are safe to
// call on nil (best-effort no-op semantics) so callers can write:
//
//	notify.Global().Signal(ctx, tid)
//
// without an extra nil-check.
func Global() *Gate { return global.Load() }
