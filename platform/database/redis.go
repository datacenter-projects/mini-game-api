package database

import (
	"context"
	"errors"
	"fmt"

	"app/pkg/configs"

	"github.com/redis/go-redis/v9"
)

// DBRedis เป็น global client ของ Redis — ใช้ผ่าน app/repository/redis เท่านั้น
var DBRedis *redis.Client

func RedisConnection(ctx context.Context, c configs.RedisConfig) error {
	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	DBRedis = client
	return nil
}

func PingRedis(ctx context.Context) error {
	if DBRedis == nil {
		return errors.New("redis not connected")
	}
	return DBRedis.Ping(ctx).Err()
}

func CloseRedis() error {
	if DBRedis == nil {
		return nil
	}
	return DBRedis.Close()
}
