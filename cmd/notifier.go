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
    // Queuing limit, and concurency see:
    // https://stackoverflow.com/questions/27745842/redis-pubsub-and-message-queueing
    // https://github.com/go-redis/redis/issues/653
    subscriber := cache.Subscribe(
        ctx,
        "api-tension-notification",
        "api-contract-notification",
    )

    if _, err := subscriber.Receive(ctx); err != nil {
        log.Fatal("Failed to receive from suscriber: ", err)
        return
    }

    ch := subscriber.Channel()

    for msg := range ch {
        switch msg.Channel {
        case "api-tension-notification":
            // go func() { }

            // Extract message
            var notif model.EventNotif
            if err := json.Unmarshal([]byte(msg.Payload), &notif); err != nil {
                log.Printf("unmarshaling error for channel %s: %v", msg.Channel, err)
            }
            if len(notif.History) == 0 {
                log.Printf("No event in notif.")
                continue
            }

            // Push notification
            if err := graph.PushHistory(&notif); err != nil {
                log.Printf("PushHistory error: %v", err)
            }
            if err := graph.PushEventNotifications(notif); err != nil {
                log.Printf("PushEventNotifications error: %v", err)
            }

            fmt.Printf(".")
        case "api-contract-notification":
            // go func() { }

            // Extract message
            var notif model.ContractNotif
            if err := json.Unmarshal([]byte(msg.Payload), &notif); err != nil {
                log.Printf("unmarshaling error for channel %s: %v", msg.Channel, err)
            }
            if notif.Contract == nil {
                log.Printf("No contract in notif.")
                continue
            }

            // Push notification
            if err := graph.PushContractNotifications(notif); err != nil {
                log.Printf("PushContractNotification error: %v: ", err)
            }

            fmt.Printf(".")
        }
    }
}
