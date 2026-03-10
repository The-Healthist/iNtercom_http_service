package handler

import (
	"intercom_http_service/internal/middleware"
	"intercom_http_service/internal/service"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// InterfaceHealthController 定义健康检查控制器接口
type InterfaceHealthController interface {
	Ping()
	Status()
	CacheStats()
}

// HealthController 处理健康检查相关的请求
type HealthController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewHealthController 创建一个新的健康检查控制器
func NewHealthController(ctx *gin.Context, container *service.ServiceContainer) *HealthController {
	return &HealthController{
		Ctx:       ctx,
		Container: container,
	}
}

// HandleHealthFunc 返回一个处理健康检查请求的Gin处理函数
func HandleHealthFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewHealthController(ctx, container)

		switch method {
		case "ping":
			controller.Ping()
		case "status":
			controller.Status()
		case "cacheStats":
			controller.CacheStats()
		default:
			ctx.JSON(400, gin.H{
				"code":    400,
				"message": "无效的方法",
				"data":    nil,
			})
		}
	}
}

// Ping 健康检查接口
// @Summary 健康检查
// @Description 简单的健康检查接口，返回pong表示服务正常
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /ping [get]
func (c *HealthController) Ping() {
	c.Ctx.JSON(200, gin.H{
		"message": "pong",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// Status 获取系统状态
// @Summary 获取系统状态
// @Description 获取系统详细状态，包括数据库连接、内存使用等
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health/status [get]
func (c *HealthController) Status() {
	// 获取数据库状态
	db := c.Container.GetDB()
	sqlDB, err := db.DB()

	dbStatus := map[string]interface{}{
		"connected": err == nil,
	}

	if err == nil {
		// 获取连接池统计信息
		stats := sqlDB.Stats()
		dbStatus["max_open_connections"] = stats.MaxOpenConnections
		dbStatus["open_connections"] = stats.OpenConnections
		dbStatus["in_use"] = stats.InUse
		dbStatus["idle"] = stats.Idle
		dbStatus["wait_count"] = stats.WaitCount
		dbStatus["wait_duration"] = stats.WaitDuration.String()
		dbStatus["max_idle_closed"] = stats.MaxIdleClosed
		dbStatus["max_lifetime_closed"] = stats.MaxLifetimeClosed
	}

	// 获取系统资源使用情况
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	systemStatus := map[string]interface{}{
		"cpu_cores":      runtime.NumCPU(),
		"goroutines":     runtime.NumGoroutine(),
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"gc_cycles":      m.NumGC,
	}

	// 返回完整状态
	c.Ctx.JSON(200, gin.H{
		"status":   "ok",
		"time":     time.Now().Format(time.RFC3339),
		"uptime":   time.Since(startTime).String(),
		"version":  "1.0.0",
		"database": dbStatus,
		"system":   systemStatus,
	})
}

// CacheStats 获取缓存统计信息
// @Summary 获取缓存统计信息
// @Description 获取系统缓存的详细统计信息
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health/cache-stats [get]
func (c *HealthController) CacheStats() {
	// 获取缓存统计信息
	cacheStats := middleware.CacheStats()

	c.Ctx.JSON(200, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
		"cache":  cacheStats,
	})
}

// 记录服务启动时间
var startTime = time.Now()
