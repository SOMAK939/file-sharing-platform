package config

import (
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/joho/godotenv"
    "github.com/redis/go-redis/v9"
)

var RDB *redis.Client
var Ctx = context.Background()

func InitRedis() {
    // Load env vars
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }

    host := strings.TrimSpace(os.Getenv("REDIS_HOST"))
    port := strings.TrimSpace(os.Getenv("REDIS_PORT"))
    password := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))

    addr := fmt.Sprintf("%s:%s", host, port)

    RDB = redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       0,
    })

    // Test the connection
    pong, err := RDB.Ping(Ctx).Result()
    if err != nil {
        log.Fatalf("Redis connection failed: %v", err)
    }

    fmt.Println("Redis connected:", pong)
}
