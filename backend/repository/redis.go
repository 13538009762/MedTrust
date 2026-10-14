package repository

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"medtrust-backend/config"
)

var (
	RDB            *redis.Client
	redisOnce      sync.Once
	redisAvailable bool
)

// InitRedis 初始化 Redis 客户端连接并校验连通性，若未配置或连接失败则自动平滑降级至 MemoryCache 内存缓存
func InitRedis() {
	redisOnce.Do(func() {
		addr := config.AppConfig.Redis.Addr
		if addr == "" {
			addr = "127.0.0.1:6379"
		}

		RDB = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: config.AppConfig.Redis.Password,
			DB:       config.AppConfig.Redis.DB,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		if err := RDB.Ping(ctx).Err(); err != nil {
			redisAvailable = false
			log.Printf("[Redis] ⚠️ Redis 服务未就绪 (%s: %v)，系统平滑启用 MemoryCache 内存缓存与 Token 黑名单容灾机制", addr, err)
		} else {
			redisAvailable = true
			log.Printf("[Redis] ✅ Redis 服务连接成功 (%s)，已启用分布式 Token 黑名单与高性能状态缓存", addr)
		}
	})
}

// IsRedisAvailable 返回当前 Redis 是否连通可用
func IsRedisAvailable() bool {
	return redisAvailable
}

// BlacklistToken 将注销的 JWT Token 加入黑名单 (支持 Redis 分布式存储 + MemoryCache 本地内存容灾双写)
func BlacklistToken(tokenStr string, ttl time.Duration) error {
	key := "blacklist:" + tokenStr
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	// 1. 本地内存双写容灾
	DefaultCache.Set(key, true, ttl)

	// 2. 若 Redis 可用，写入 Redis 分布式黑名单
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := RDB.Set(ctx, key, "revoked", ttl).Err(); err != nil {
			log.Printf("[Redis] 写入 Token 黑名单失败: %v, 使用内存缓存兜底", err)
			return err
		}
	}
	return nil
}

// IsTokenBlacklisted 检查 Token 是否在黑名单中 (优先查本地内存缓存，未命中再查 Redis)
func IsTokenBlacklisted(tokenStr string) bool {
	key := "blacklist:" + tokenStr

	// 1. 本地内存快速命中
	if _, ok := DefaultCache.Get(key); ok {
		return true
	}

	// 2. 查询 Redis
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		val, err := RDB.Get(ctx, key).Result()
		if err == nil && val != "" {
			DefaultCache.Set(key, true, 10*time.Minute)
			return true
		}
	}

	return false
}

// CloseRedis 优雅关闭 Redis 连接
func CloseRedis() {
	if RDB != nil {
		_ = RDB.Close()
	}
}
