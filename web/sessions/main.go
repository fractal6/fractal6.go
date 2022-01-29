package sessions

import (
    "github.com/gomodule/redigo/redis"
    "github.com/gofrs/uuid"
)

type Session = redis.Conn

var cache Session

func init() {
    // Cache init
    initCache()
}

func GetCache() Session {
    return cache
}

func GenerateToken() string {
    token, _ := uuid.NewV4()
    return token.String()
}

func initCache() {
    con, err := redis.DialURL("redis://localhost")
    if err != nil { panic("Redis connection error:" + err.Error()) }
    cache = con
}

