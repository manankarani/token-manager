package datasources

import (
	"context"
	"strconv"
	"time"

	"github.com/manankarani/token-manager/env"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient initializes and returns a Redis client.
func NewRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     env.Conf.Redis.Host + ":" + strconv.Itoa(env.Conf.Redis.Port),
		Username: "",
		Password: "",
		DB:       0,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		panic("Redis connection failed: " + err.Error())
	}

	return client
}
