package middleware

import (
	"fmt"
	"metadatatool/internal/pkg/domain"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
	RedisClient       *redis.Client
}

// RateLimit creates a rate limiting middleware using Redis
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user identifier (API key or user ID)
		identifier := getUserIdentifier(c)
		if identifier == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authentication",
			})
			c.Abort()
			return
		}

		// Create Redis keys
		countKey := fmt.Sprintf("ratelimit:%s:count", identifier)
		timestampKey := fmt.Sprintf("ratelimit:%s:ts", identifier)

		// Get current count and timestamp
		pipe := cfg.RedisClient.Pipeline()
		countCmd := pipe.Get(c, countKey)
		timestampCmd := pipe.Get(c, timestampKey)
		_, err := pipe.Exec(c)

		var count int
		var lastTimestamp time.Time

		if err == redis.Nil {
			// First request
			count = 0
			lastTimestamp = time.Now()
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "rate limit check failed",
			})
			c.Abort()
			return
		} else {
			count, _ = strconv.Atoi(countCmd.Val())
			ts, _ := strconv.ParseInt(timestampCmd.Val(), 10, 64)
			lastTimestamp = time.Unix(ts, 0)
		}

		// Reset count if minute window has passed
		now := time.Now()
		if now.Sub(lastTimestamp) >= time.Minute {
			count = 0
			lastTimestamp = now
		}

		// Check rate limit
		if count >= cfg.RequestsPerMinute {
			c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(lastTimestamp.Add(time.Minute).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": lastTimestamp.Add(time.Minute).Unix(),
			})
			c.Abort()
			return
		}

		// Update rate limit in Redis
		pipe = cfg.RedisClient.Pipeline()
		pipe.Set(c, countKey, count+1, time.Minute)
		pipe.Set(c, timestampKey, lastTimestamp.Unix(), time.Minute)
		_, err = pipe.Exec(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "rate limit update failed",
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(cfg.RequestsPerMinute-count-1))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(lastTimestamp.Add(time.Minute).Unix(), 10))

		c.Next()
	}
}

func getUserIdentifier(c *gin.Context) string {
	// Try API key first
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Then try user claims from JWT
	if claims, exists := c.Get("user"); exists {
		if userClaims, ok := claims.(*domain.Claims); ok {
			return userClaims.UserID
		}
	}

	// Finally use IP address
	return c.ClientIP()
}
