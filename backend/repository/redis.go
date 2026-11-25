package repository

import (
	"context"
	"fmt"
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

const UserStatusCachePrefix = "user:status:"

// CacheUserStatus 将用户状态写入多级缓存 (Redis 分布式 + MemoryCache 本地内存容灾)
func CacheUserStatus(userID uint64, status string, ttl time.Duration) {
	if userID == 0 || status == "" {
		return
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	key := fmt.Sprintf("%s%d", UserStatusCachePrefix, userID)

	// 1. 本地内存双写容灾
	DefaultCache.Set(key, status, ttl)

	// 2. 若 Redis 可用，写入 Redis
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = RDB.Set(ctx, key, status, ttl).Err()
	}
}

// GetUserStatus 从多级缓存获取用户账号状态 (优先本地内存，次查 Redis，大幅降低高频查库压力)
func GetUserStatus(userID uint64) (string, bool) {
	if userID == 0 {
		return "", false
	}
	key := fmt.Sprintf("%s%d", UserStatusCachePrefix, userID)

	// 1. 本地内存快速命中 (微秒级)
	if val, ok := DefaultCache.Get(key); ok {
		if s, ok := val.(string); ok {
			return s, true
		}
	}

	// 2. 查询 Redis 分布式缓存
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		val, err := RDB.Get(ctx, key).Result()
		if err == nil && val != "" {
			DefaultCache.Set(key, val, 60*time.Second)
			return val, true
		}
	}

	return "", false
}

// InvalidateUserStatus 主动使指定用户的状态缓存失效 (写后失效，保证状态秒级强一致性)
func InvalidateUserStatus(userID uint64) {
	if userID == 0 {
		return
	}
	key := fmt.Sprintf("%s%d", UserStatusCachePrefix, userID)

	// 1. 清理本地内存缓存
	DefaultCache.Delete(key)

	// 2. 清理 Redis 缓存
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = RDB.Del(ctx, key).Err()
	}
}

const SSETicketPrefix = "sse:ticket:"

// StoreSSETicket 将单次有效 SSE 接入票据存入缓存 (TTL 默认为 60s)
func StoreSSETicket(ticket, data string, ttl time.Duration) error {
	if ticket == "" || data == "" {
		return fmt.Errorf("ticket and data cannot be empty")
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	key := SSETicketPrefix + ticket

	// 1. 本地内存双写容灾
	DefaultCache.Set(key, data, ttl)

	// 2. Redis 写入
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if err := RDB.Set(ctx, key, data, ttl).Err(); err != nil {
			log.Printf("[Redis] 写入 SSE Ticket 失败: %v, 使用内存缓存兜底", err)
		}
	}
	return nil
}

// ConsumeSSETicket 原子获取并立即删除单次有效票据 (防止重放与重复使用)
//
// Redis 侧使用 GETDEL (Redis 6.2+) 保证 get+delete 的原子性：并发重放时只有一个请求能拿到票据。
// 内存侧先 Delete 再返回，避免 Get 与 Delete 之间的窗口期被其他 goroutine 命中。
func ConsumeSSETicket(ticket string) (string, bool) {
	if ticket == "" {
		return "", false
	}
	key := SSETicketPrefix + ticket

	// 1. 先走 Redis 原子 GETDEL (分布式场景权威来源)
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		val, err := RDB.GetDel(ctx, key).Result()
		if err == nil && val != "" {
			// Redis 已删除，同步清理本地内存副本避免残留
			DefaultCache.Delete(key)
			return val, true
		}
	}

	// 2. Redis 未命中或不可用：回落到本地内存 (先删除再判定，杜绝双花)
	if val, ok := DefaultCache.GetAndDelete(key); ok {
		if s, ok := val.(string); ok && s != "" {
			return s, true
		}
	}

	return "", false
}

const (
	UserPwdChangedPrefix  = "user:pwd_changed_ns:"
	KeyUnlockCounterPref  = "key_unlock:fail:"
	KeyUnlockLockPrefix   = "key_unlock:lock:"
)

// CachePasswordChangedAtNs 缓存用户最近一次改密/重置密码的纳秒时间戳（Redis + 本地内存双写）
func CachePasswordChangedAtNs(userID uint64, tsNs int64, ttl time.Duration) {
	if userID == 0 {
		return
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	key := fmt.Sprintf("%s%d", UserPwdChangedPrefix, userID)
	DefaultCache.Set(key, tsNs, ttl)
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = RDB.Set(ctx, key, fmt.Sprintf("%d", tsNs), ttl).Err()
	}
}

// GetPasswordChangedAtNs 读取用户最近改密时间戳（缓存命中返回 (tsNs, true)）
func GetPasswordChangedAtNs(userID uint64) (int64, bool) {
	if userID == 0 {
		return 0, false
	}
	key := fmt.Sprintf("%s%d", UserPwdChangedPrefix, userID)
	if val, ok := DefaultCache.Get(key); ok {
		if ts, ok := val.(int64); ok {
			return ts, true
		}
	}
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		val, err := RDB.Get(ctx, key).Int64()
		if err == nil {
			DefaultCache.Set(key, val, 5*time.Minute)
			return val, true
		}
	}
	return 0, false
}

// IncrementKeyUnlockFailure 递增密钥解锁失败计数（自动 TTL 过期，支持 Redis 多实例与本地内存容灾）
func IncrementKeyUnlockFailure(scopeKey string, ttl time.Duration) int {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	key := KeyUnlockCounterPref + scopeKey
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		cnt, err := RDB.Incr(ctx, key).Result()
		if err == nil {
			if cnt == 1 {
				_ = RDB.Expire(ctx, key, ttl).Err()
			}
			DefaultCache.Set(key, int(cnt), ttl)
			return int(cnt)
		}
	}
	cur := 0
	if val, ok := DefaultCache.Get(key); ok {
		if n, ok := val.(int); ok {
			cur = n
		}
	}
	cur++
	DefaultCache.Set(key, cur, ttl)
	return cur
}

// SetKeyUnlockLock 写入密钥解锁锁定窗口（带自动 TTL 淘汰）
func SetKeyUnlockLock(scopeKey string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	key := KeyUnlockLockPrefix + scopeKey
	until := time.Now().Add(ttl)
	DefaultCache.Set(key, until, ttl)
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = RDB.Set(ctx, key, until.Unix(), ttl).Err()
	}
}

// GetKeyUnlockLockRemaining 检查某维度是否处于密钥防爆破锁定窗口，返回 (locked, remainingMinutes)
func GetKeyUnlockLockRemaining(scopeKey string) (bool, int) {
	key := KeyUnlockLockPrefix + scopeKey
	now := time.Now()
	if val, ok := DefaultCache.Get(key); ok {
		if until, ok := val.(time.Time); ok && until.After(now) {
			return true, int(until.Sub(now).Minutes()) + 1
		}
	}
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		ts, err := RDB.Get(ctx, key).Int64()
		if err == nil {
			until := time.Unix(ts, 0)
			if until.After(now) {
				DefaultCache.Set(key, until, until.Sub(now))
				return true, int(until.Sub(now).Minutes()) + 1
			}
		}
	}
	return false, 0
}

// ClearKeyUnlockFailure 清除单维度密钥解锁失败计数与锁定状态
func ClearKeyUnlockFailure(scopeKey string) {
	cntKey := KeyUnlockCounterPref + scopeKey
	lockKey := KeyUnlockLockPrefix + scopeKey
	DefaultCache.Delete(cntKey)
	DefaultCache.Delete(lockKey)
	if redisAvailable && RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = RDB.Del(ctx, cntKey, lockKey).Err()
	}
}
