package middleware

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 缓存条目
type cacheEntry struct {
	Content    []byte
	Expiration time.Time
}

// 内存缓存
type memoryCache struct {
	sync.RWMutex
	items map[string]cacheEntry
}

// 全局缓存实例
var cache = &memoryCache{
	items: make(map[string]cacheEntry),
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Expiration time.Duration             // 缓存过期时间
	Methods    []string                  // 需要缓存的HTTP方法
	KeyFunc    func(*gin.Context) string // 自定义缓存键生成函数
}

// DefaultCacheConfig 默认缓存配置
var DefaultCacheConfig = CacheConfig{
	Expiration: 5 * time.Minute,
	Methods:    []string{http.MethodGet},
	KeyFunc:    defaultKeyFunc,
}

// 默认缓存键生成函数
func defaultKeyFunc(c *gin.Context) string {
	// 获取请求路径
	path := c.Request.URL.Path

	// 获取查询参数并排序
	queryParams := c.Request.URL.Query()
	var queryKeys []string
	for key := range queryParams {
		queryKeys = append(queryKeys, key)
	}
	sort.Strings(queryKeys)

	// 构建查询字符串
	var queryString string
	for _, key := range queryKeys {
		values := queryParams[key]
		sort.Strings(values)
		for _, value := range values {
			queryString += key + "=" + value + "&"
		}
	}

	// 生成缓存键
	key := path + "?" + queryString

	// 使用MD5哈希缓存键
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// Cache 创建缓存中间件
func Cache(config ...CacheConfig) gin.HandlerFunc {
	// 使用默认配置或自定义配置
	var cfg CacheConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultCacheConfig
	}

	// 确保配置有效
	if cfg.Expiration <= 0 {
		cfg.Expiration = DefaultCacheConfig.Expiration
	}
	if len(cfg.Methods) == 0 {
		cfg.Methods = DefaultCacheConfig.Methods
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = DefaultCacheConfig.KeyFunc
	}

	// 返回中间件函数
	return func(c *gin.Context) {
		// 检查请求方法是否需要缓存
		methodAllowed := false
		for _, method := range cfg.Methods {
			if c.Request.Method == method {
				methodAllowed = true
				break
			}
		}

		// 如果请求方法不需要缓存，直接处理请求
		if !methodAllowed {
			c.Next()
			return
		}

		// 生成缓存键
		key := cfg.KeyFunc(c)

		// 尝试从缓存获取响应
		cache.RLock()
		entry, found := cache.items[key]
		cache.RUnlock()

		if found && entry.Expiration.After(time.Now()) {
			// 缓存命中，直接返回缓存的响应
			c.Data(http.StatusOK, "application/json; charset=utf-8", entry.Content)
			c.Abort()
			return
		}

		// 缓存未命中，捕获响应
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// 处理请求
		c.Next()

		// 如果状态码为200，缓存响应
		if c.Writer.Status() == http.StatusOK {
			content := writer.body.Bytes()
			cache.Lock()
			cache.items[key] = cacheEntry{
				Content:    content,
				Expiration: time.Now().Add(cfg.Expiration),
			}
			cache.Unlock()
		}
	}
}

// CacheByParams 根据请求参数创建缓存中间件
func CacheByParams(expiration time.Duration, params ...string) gin.HandlerFunc {
	return Cache(CacheConfig{
		Expiration: expiration,
		Methods:    []string{http.MethodGet},
		KeyFunc: func(c *gin.Context) string {
			// 获取请求路径
			path := c.Request.URL.Path

			// 获取指定的查询参数
			var keyParts []string
			keyParts = append(keyParts, path)

			for _, param := range params {
				value := c.Query(param)
				if value != "" {
					keyParts = append(keyParts, param+"="+value)
				}
			}

			// 生成缓存键
			key := strings.Join(keyParts, "&")

			// 使用MD5哈希缓存键
			hasher := md5.New()
			hasher.Write([]byte(key))
			return hex.EncodeToString(hasher.Sum(nil))
		},
	})
}

// CacheByBody 根据请求体创建缓存中间件
func CacheByBody(expiration time.Duration, methods ...string) gin.HandlerFunc {
	if len(methods) == 0 {
		methods = []string{http.MethodPost, http.MethodPut}
	}

	return Cache(CacheConfig{
		Expiration: expiration,
		Methods:    methods,
		KeyFunc: func(c *gin.Context) string {
			// 获取请求路径
			path := c.Request.URL.Path

			// 读取并重置请求体
			var bodyBytes []byte
			if c.Request.Body != nil {
				bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
				c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// 生成缓存键
			key := path + string(bodyBytes)

			// 使用MD5哈希缓存键
			hasher := md5.New()
			hasher.Write([]byte(key))
			return hex.EncodeToString(hasher.Sum(nil))
		},
	})
}

// PurgeCache 清除所有缓存
func PurgeCache() {
	cache.Lock()
	cache.items = make(map[string]cacheEntry)
	cache.Unlock()
}

// PurgeCacheByPrefix 根据前缀清除缓存
func PurgeCacheByPrefix(prefix string) {
	cache.Lock()
	defer cache.Unlock()

	for key := range cache.items {
		if strings.HasPrefix(key, prefix) {
			delete(cache.items, key)
		}
	}
}

// 自定义响应写入器，用于捕获响应内容
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 重写Write方法，同时写入原始响应和缓冲区
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString 重写WriteString方法，同时写入原始响应和缓冲区
func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// CacheStats 获取缓存统计信息
func CacheStats() map[string]interface{} {
	cache.RLock()
	defer cache.RUnlock()

	stats := map[string]interface{}{
		"total_items": len(cache.items),
		"items":       make([]map[string]interface{}, 0),
	}

	for key, entry := range cache.items {
		item := map[string]interface{}{
			"key":        key,
			"size":       len(entry.Content),
			"expiration": entry.Expiration.Format(time.RFC3339),
			"expired":    entry.Expiration.Before(time.Now()),
		}
		stats["items"] = append(stats["items"].([]map[string]interface{}), item)
	}

	return stats
}

// 定期清理过期缓存
func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			cleanExpiredCache()
		}
	}()
}

// cleanExpiredCache 清理过期缓存
func cleanExpiredCache() {
	now := time.Now()

	cache.Lock()
	defer cache.Unlock()

	for key, entry := range cache.items {
		if entry.Expiration.Before(now) {
			delete(cache.items, key)
		}
	}
}
