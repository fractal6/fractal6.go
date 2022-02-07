package sessions

import (
    "fmt"
    "os"
    "context"
    "github.com/gofrs/uuid"
    "github.com/go-redis/redis/v8"
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
        Addr:     "localhost:6379",
        //Password: "", // no password set
        //DB:       0,  // use default DB
    })

    _, err := cache.Ping(context.Background()).Result()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Redis Error: %v\n", err)
        os.Exit(1)
    }
}

