package cmd

import (
    "fmt"
    "log"
    "context"
    "encoding/json"
    "github.com/go-redis/redis/v8"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/graph"
	//. "fractale/fractal6.go/tools"
)

var cache *redis.Client = redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    //Password: "", // no password set
    //DB:       0,  // use default DB
})

var ctx = context.Background()

func RunNotifier() {
    // Test connection
    if _, err := cache.Ping(ctx).Result(); err != nil {
        log.Fatal("redis error: ", err)
    }

    // Init Suscribe channel
    // Queuing limit: https://stackoverflow.com/questions/27745842/redis-pubsub-and-message-queueing
    subscriber := cache.Subscribe(ctx, "fractal6-tension-notification")

    for {
        var notif model.Notif

        // Get notification
        msg, err := subscriber.ReceiveMessage(ctx)
        if err != nil {
            log.Printf("redis suscribe error: %v", err)
        }

        // Unmarshal
        if err := json.Unmarshal([]byte(msg.Payload), &notif); err != nil {
            log.Printf("unmarshaling error for channel %s: %v", msg.Channel, err)
        }

        if len(notif.History) == 0 {
            log.Printf("No event in notif.")
            continue
        }

        // Push notification
        if err = graph.PushHistory(notif.Uctx, notif.Tid, notif.History); err != nil {
            log.Printf("PushHistory error: %v", err)
        }
        if err = graph.PushEventNotifications(notif.Tid, notif.History); err != nil {
            log.Printf("PushEventNotification error: %v", err)
        }

        fmt.Printf(".")
    }
}
