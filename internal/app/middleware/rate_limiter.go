package middleware

import (
	"intercom_http_service/internal/error/code"
	"intercom_http_service/internal/error/response"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 简单的令牌桶限流器
type TokenBucket struct {
	rate       float64    // 每秒填充的令牌数
	capacity   int        // 桶的容量
	tokens     float64    // 当前令牌数
	lastRefill time.Time  // 上次填充时间
	mu         sync.Mutex // 互斥锁
}

// 创建新的令牌桶限流器
func NewTokenBucket(rate float64, capacity int) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     float64(capacity),
		lastRefill: time.Now(),
	}
}

// 尝试获取令牌
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	// 填充令牌
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}

	// 尝试获取令牌
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

// 限流器映射
var (
	ipLimiters     = make(map[string]*TokenBucket)
	ipLimitersMu   sync.RWMutex
	pathLimiters   = make(map[string]*TokenBucket)
	pathLimitersMu sync.RWMutex
)

// RateLimiterConfig 限流器配置
type RateLimiterConfig struct {
	Rate       float64                   // 每秒允许的请求数
	Burst      int                       // 允许的突发请求数
	ExpiryTime time.Duration             // 限流器过期时间
	LimitType  string                    // 限流类型: "ip", "path", "combined"
	KeyFunc    func(*gin.Context) string // 自定义键生成函数
}

// DefaultRateLimiterConfig 默认限流器配置
var DefaultRateLimiterConfig = RateLimiterConfig{
	Rate:       1,             // 每秒1个请求
	Burst:      5,             // 允许5个突发请求
	ExpiryTime: 1 * time.Hour, // 1小时后过期
	LimitType:  "ip",          // 默认按IP限流
	KeyFunc:    nil,           // 默认为nil，根据LimitType自动选择
}

// 获取IP限流器
func getIPLimiter(ip string, cfg RateLimiterConfig) *TokenBucket {
	ipLimitersMu.RLock()
	limiter, exists := ipLimiters[ip]
	ipLimitersMu.RUnlock()

	if !exists {
		limiter = NewTokenBucket(cfg.Rate, cfg.Burst)
		ipLimitersMu.Lock()
		ipLimiters[ip] = limiter
		ipLimitersMu.Unlock()

		// 设置过期时间
		if cfg.ExpiryTime > 0 {
			go func() {
				time.Sleep(cfg.ExpiryTime)
				ipLimitersMu.Lock()
				delete(ipLimiters, ip)
				ipLimitersMu.Unlock()
			}()
		}
	}

	return limiter
}

// 获取路径限流器
func getPathLimiter(path string, cfg RateLimiterConfig) *TokenBucket {
	pathLimitersMu.RLock()
	limiter, exists := pathLimiters[path]
	pathLimitersMu.RUnlock()

	if !exists {
		limiter = NewTokenBucket(cfg.Rate, cfg.Burst)
		pathLimitersMu.Lock()
		pathLimiters[path] = limiter
		pathLimitersMu.Unlock()
	}

	return limiter
}

// RateLimiter 创建限流中间件
func RateLimiter(config ...RateLimiterConfig) gin.HandlerFunc {
	// 使用默认配置或自定义配置
	var cfg RateLimiterConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultRateLimiterConfig
	}

	// 确保配置有效
	if cfg.Rate <= 0 {
		cfg.Rate = DefaultRateLimiterConfig.Rate
	}
	if cfg.Burst <= 0 {
		cfg.Burst = DefaultRateLimiterConfig.Burst
	}
	if cfg.LimitType == "" {
		cfg.LimitType = DefaultRateLimiterConfig.LimitType
	}

	// 返回中间件函数
	return func(c *gin.Context) {
		var limiter *TokenBucket

		// 根据限流类型选择限流器
		switch cfg.LimitType {
		case "ip":
			// 按IP限流
			ip := c.ClientIP()
			limiter = getIPLimiter(ip, cfg)
		case "path":
			// 按路径限流
			path := c.Request.URL.Path
			limiter = getPathLimiter(path, cfg)
		case "combined":
			// 按IP和路径组合限流
			ip := c.ClientIP()
			path := c.Request.URL.Path
			key := ip + ":" + path
			limiter = getIPLimiter(key, cfg)
		default:
			// 自定义键限流
			if cfg.KeyFunc != nil {
				key := cfg.KeyFunc(c)
				limiter = getIPLimiter(key, cfg)
			} else {
				// 默认按IP限流
				ip := c.ClientIP()
				limiter = getIPLimiter(ip, cfg)
			}
		}

		// 检查是否允许请求
		if !limiter.Allow() {
			response.FailWithMessage(c, code.ErrTooManyRequests, "请求频率过高，请稍后再试", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimiter 按IP限流
func IPRateLimiter(rate float64, burst int) gin.HandlerFunc {
	return RateLimiter(RateLimiterConfig{
		Rate:      rate,
		Burst:     burst,
		LimitType: "ip",
	})
}

// PathRateLimiter 按路径限流
func PathRateLimiter(rate float64, burst int) gin.HandlerFunc {
	return RateLimiter(RateLimiterConfig{
		Rate:      rate,
		Burst:     burst,
		LimitType: "path",
	})
}

// CombinedRateLimiter 按IP和路径组合限流
func CombinedRateLimiter(rate float64, burst int) gin.HandlerFunc {
	return RateLimiter(RateLimiterConfig{
		Rate:      rate,
		Burst:     burst,
		LimitType: "combined",
	})
}

// CustomRateLimiter 自定义键限流
func CustomRateLimiter(rate float64, burst int, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return RateLimiter(RateLimiterConfig{
		Rate:      rate,
		Burst:     burst,
		LimitType: "custom",
		KeyFunc:   keyFunc,
	})
}

// 定期清理过期的限流器
func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			cleanExpiredLimiters()
		}
	}()
}

// cleanExpiredLimiters 清理过期的限流器
func cleanExpiredLimiters() {
	now := time.Now()

	// 清理IP限流器 - 这里我们可以添加一些额外的清理逻辑
	ipLimitersMu.Lock()
	for ip := range ipLimiters {
		// 简单示例：随机清理一些限流器
		if now.Nanosecond()%2 == 0 {
			delete(ipLimiters, ip)
		}
	}
	ipLimitersMu.Unlock()

	// 清理路径限流器
	pathLimitersMu.Lock()
	for path := range pathLimiters {
		// 简单示例：随机清理一些限流器
		if now.Nanosecond()%3 == 0 {
			delete(pathLimiters, path)
		}
	}
	pathLimitersMu.Unlock()
}
