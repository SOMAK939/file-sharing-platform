package config

import (
	"context"
	"fmt"
	
	"os"

	
	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

// Initialize Redis connection
func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"), // Leave empty if no password
		DB:       0,  // Default DB
	})

	// Check connection
	ctx := context.Background()
	_, err := RDB.Ping(ctx).Result()
	if err != nil {
		fmt.Println(" Redis connection failed:", err)
		return
	}
	fmt.Println("Connected to Redis Cloud!")
}
// Set key-value pair in Redis with optional expiration
func SetCache(key, value string, expiration int) error {
	ctx := context.Background()
	return RDB.Set(ctx, key, value, 0).Err()
}

// Get value from Redis cache
func GetCache(key string) (string, error) {
	ctx := context.Background()
	return RDB.Get(ctx, key).Result()
}

// Delete a key from Redis cache
func DeleteCache(key string) error {
	ctx := context.Background()
	return RDB.Del(ctx, key).Err()
}
