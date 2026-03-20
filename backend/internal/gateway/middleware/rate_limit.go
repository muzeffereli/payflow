package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"payment-platform/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit applies a sliding-window rate limit of limit requests per window,
// keyed by client IP. Intended for broad traffic shaping on all routes.
func RateLimit(cfg config.RedisConfig, limit int, window time.Duration) gin.HandlerFunc {
	return slidingWindowLimit(newRedisClient(cfg), "ratelimit", limit, window)
}

// AuthRateLimit applies a tighter sliding-window rate limit, keyed by client IP
// under a separate namespace. Use this on /auth/login and /auth/register to
// mitigate brute-force attacks independently of the global rate limit.
func AuthRateLimit(cfg config.RedisConfig, limit int, window time.Duration) gin.HandlerFunc {
	return slidingWindowLimit(newRedisClient(cfg), "authlimit", limit, window)
}

func newRedisClient(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
	})
}

func slidingWindowLimit(rdb *redis.Client, keyPrefix string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyPrefix + ":" + c.ClientIP()
		now := time.Now()
		windowStart := now.Add(-window).UnixNano()

		pipe := rdb.Pipeline()
		pipe.ZRemRangeByScore(context.Background(), key, "0", strconv.FormatInt(windowStart, 10))
		countCmd := pipe.ZCard(context.Background(), key)
		nowNano := strconv.FormatInt(now.UnixNano(), 10)
		pipe.ZAdd(context.Background(), key, redis.Z{
			Score:  float64(now.UnixNano()),
			Member: nowNano,
		})
		pipe.Expire(context.Background(), key, window)

		if _, err := pipe.Exec(context.Background()); err != nil {
			c.Next()
			return
		}

		count := countCmd.Val()

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(0, limit-int(count))))

		if int(count) > limit {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": window.Seconds(),
			})
			return
		}

		c.Next()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
