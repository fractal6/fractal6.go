/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2024 Fractale Co
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

package sessions

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	"os"
	//"github.com/gomodule/redigo/redis"
)

type Session = redis.Client

var cache *Session

func init() {
	// Cache init
	initCache()
}

func GetCache() *Session {
	return cache
}

func GenerateToken() string {
	token, _ := uuid.NewV4()
	return token.String()
}

func initCache() {
	//con, err := redis.DialURL("redis://localhost")
	////defer con.Close()
	//if err != nil { panic("Redis connection error:" + err.Error()) }
	//cache = con
	cache = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		//Password: "", // no password set
		//DB:       0,  // use default DB
	})

	_, err := cache.Ping(context.Background()).Result()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Redis Error: %v\n", err)
		os.Exit(1)
	}
}
