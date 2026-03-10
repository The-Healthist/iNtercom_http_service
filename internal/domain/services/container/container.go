package container

import (
	"context"
	"log"
	"sync"
	"time"

	"intercom_http_service/internal/domain/services"
	"intercom_http_service/internal/infrastructure/config"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// ServiceContainer 管理所有服务的依赖注入
type ServiceContainer struct {
	db     *gorm.DB
	config *config.Config
	redis  *redis.Client

	// 基础服务
	jwtService services.InterfaceJWTService

	// RTC相关服务
	rtcService        services.InterfaceRTCService
	tencentRTCService services.InterfaceTencentRTCService

	// 数据存储服务
	redisService services.InterfaceRedisService

	// MQTT通话服务
	mqttCallService services.InterfaceMQTTCallService

	// 业务服务
	deviceService     services.InterfaceDeviceService
	adminService      services.InterfaceAdminService
	residentService   services.InterfaceResidentService
	staffService      services.InterfaceStaffService
	callRecordService services.InterfaceCallRecordService
	emergencyService  services.InterfaceEmergencyService
	buildingService   services.InterfaceBuildingService
	householdService  services.InterfaceHouseholdService

	mu sync.RWMutex
}

// NewServiceContainer 创建新的服务容器
func NewServiceContainer(db *gorm.DB, cfg *config.Config, redisClient *redis.Client) *ServiceContainer {
	if db == nil {
		panic("数据库连接为空")
	}

	if cfg == nil {
		panic("配置为空")
	}

	// 测试Redis连接
	if redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Printf("Redis连接测试失败: %v，将不使用Redis缓存", err)
		}
	}

	container := &ServiceContainer{
		db:     db,
		config: cfg,
		redis:  redisClient,
	}
	container.initializeServices()
	return container
}

// initializeServices 初始化所有服务
func (c *ServiceContainer) initializeServices() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 初始化基础服务
	c.jwtService = services.NewJWTService(c.config, c.db)

	// 初始化RTC服务
	c.rtcService = services.NewRTCService(c.config)
	c.tencentRTCService = services.NewTencentRTCService(c.config)

	// 初始化Redis服务
	c.redisService = services.NewRedisService(c.config)

	// 初始化MQTT通话服务 - 使用接口类型
	c.mqttCallService = services.NewMQTTCallService(c.db, c.config, c.tencentRTCService)

	// 连接MQTT服务器
	if err := c.mqttCallService.Connect(); err != nil {
		log.Printf("MQTT服务连接失败: %v", err)
	}

	// 初始化业务服务
	c.deviceService = services.NewDeviceService(c.db, c.config)
	c.adminService = services.NewAdminService(c.db, c.config)
	c.residentService = services.NewResidentService(c.db, c.config)
	c.staffService = services.NewStaffService(c.db, c.config)
	c.callRecordService = services.NewCallRecordService(c.db, c.config)
	c.emergencyService = services.NewEmergencyService(c.db, c.config)

	// 初始化楼号和户号服务
	c.buildingService = services.NewBuildingService(c.db, c.config)
	c.householdService = services.NewHouseholdService(c.db, c.config)
}

// GetService 获取指定名称的服务
func (c *ServiceContainer) GetService(name string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch name {
	case "config":
		return c.config
	case "db":
		return c.db
	case "jwt":
		return c.jwtService
	case "rtc":
		return c.rtcService
	case "tencent_rtc":
		return c.tencentRTCService
	case "mqtt_call":
		return c.mqttCallService
	case "redis":
		return c.redisService
	case "device":
		return c.deviceService
	case "admin":
		return c.adminService
	case "resident":
		return c.residentService
	case "staff":
		return c.staffService
	case "call_record":
		return c.callRecordService
	case "emergency":
		return c.emergencyService
	case "building":
		return c.buildingService
	case "household":
		return c.householdService
	default:
		return nil
	}
}

// GetDB 获取数据库连接
func (c *ServiceContainer) GetDB() *gorm.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}
