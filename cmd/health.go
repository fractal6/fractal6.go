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

package cmd

import (
	"context"
	"log"
	"time"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/storage"
	"fractale/fractal6.go/web/email"
)

// initStorage builds the S3 client from [storage] and registers it as the
// process-wide global (db cascade-GC and email attachments read storage.Global()).
// Returns nil when [storage] is unset; callers nil-check. Both the api server
// and the notifier daemon must call this.
func initStorage() *storage.Client {
	cli, err := storage.New(storage.LoadConfig())
	if err != nil {
		return nil
	}
	storage.SetGlobal(cli)
	return cli
}

// checkServices pings the external dependencies at startup so a broken
// deployment shows up in the first log lines instead of at the first failing
// request. Non-fatal: unconfigured or dead services only degrade their feature.
func checkServices(storageCli *storage.Client) {
	ping("dgraph", db.GetDB().Ping)
	if storageCli != nil {
		ping("storage", storageCli.Ping)
	} else {
		log.Print("[health] storage: not configured - /file/* returns 503")
	}
	if email.IsConfigured() {
		ping("email", email.Ping)
	} else {
		log.Print("[health] email: not configured - no emails sent")
	}
}

func ping(name string, f func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f(ctx); err != nil {
		log.Printf("[health] %s: WARN %v", name, err)
		return
	}
	log.Printf("[health] %s: ok", name)
}
