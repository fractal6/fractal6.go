package cmd

import (
    "fmt"
    "log"
    "context"
    "encoding/json"
    "runtime/debug"
    "github.com/go-redis/redis/v8"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph"
    "fractale/fractal6.go/web/email"
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
        "api-notif-notification",
    )

    if _, err := subscriber.Receive(ctx); err != nil {
        log.Fatal("Failed to receive from suscriber: ", err)
        return
    }

    for msg := range subscriber.Channel() {
        switch msg.Channel {
        case "api-tension-notification":
            go processTensionNotification(msg)

        case "api-contract-notification":
            go processContractNotification(msg)

        case "api-notif-notification":
            go processNotifNotification(msg)

        }
    }
}

func handlePanic(info string) {
    if r := recover(); r != nil {
        fmt.Printf("error: Recovering from panic (%s): %v\n", info, r)
        email.SendMaintainerEmail(
            fmt.Sprintf("[%s/error] %v", info, r),
            string(debug.Stack()),
        )
    }
}

func processTensionNotification(msg *redis.Message) {
    defer handlePanic("tension event")
    // Extract message
    var notif model.EventNotif
    if err := json.Unmarshal([]byte(msg.Payload), &notif); err != nil {
        log.Printf("unmarshaling error for channel %s: %v", msg.Channel, err)
    }
    if len(notif.History) == 0 {
        log.Printf("No event in notif.")
        return
    }

    // Push notification
    if err := graph.PushHistory(&notif); err != nil {
        log.Printf("PushHistory error: %v", err)
    }
    if err := graph.PushEventNotifications(notif); err != nil {
        log.Printf("PushEventNotifications error: %v", err)
    }

    fmt.Printf(".")
}

func processContractNotification(msg *redis.Message) {
    defer handlePanic("contract event")
    // Extract message
    var notif model.ContractNotif
    if err := json.Unmarshal([]byte(msg.Payload), &notif); err != nil {
        log.Printf("unmarshaling error for channel %s: %v", msg.Channel, err)
    }
    if notif.Contract == nil {
        log.Printf("No contract in notif.")
        return
    }

    // Push notification
    if err := graph.PushContractNotifications(notif); err != nil {
        log.Printf("PushContractNotification error: %v: ", err)
    }

    fmt.Printf(".")
}

func processNotifNotification(msg *redis.Message) {
    defer handlePanic("notif event")
    // Extract message
    var notif model.NotifNotif
    if err := json.Unmarshal([]byte(msg.Payload), &notif); err != nil {
        log.Printf("unmarshaling error for channel %s: %v", msg.Channel, err)
    }
    if len(notif.Msg) == 0 {
        log.Printf("No message in notif.")
        return
    }

    // Push notification
    if err := graph.PushNotifNotifications(notif); err != nil {
        log.Printf("PushEventNotifications error: %v", err)
    }

    fmt.Printf(".")
}
