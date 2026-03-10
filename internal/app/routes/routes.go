package routes

import (
	_ "intercom_http_service/docs"
	"intercom_http_service/internal/app/controllers"
	"intercom_http_service/internal/app/middleware"
	"intercom_http_service/internal/domain/services/container"
	"intercom_http_service/internal/infrastructure/config"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// SetupRouter 初始化并返回配置好的路由
func SetupRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	// 初始化 Gin
	r := gin.Default()

	// 添加 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:20033")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
	// 设置正确的Content-Type，确保UTF-8编码
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Next()
	})
	// 创建服务容器
	serviceContainer := container.NewServiceContainer(db, cfg, nil)
	// 初始化中间件
	middleware.InitAuthMiddleware(cfg, db)
	// 添加 Swagger 文档路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 注册路由
	registerRoutes(r, serviceContainer)
	return r
}

// registerRoutes 配置所有API路由
func registerRoutes(
	r *gin.Engine,
	container *container.ServiceContainer,
) {
	// API 路由根路径
	api := r.Group("/api")
	// 注册公共路由
	registerPublicRoutes(api, container)
	// 注册需要认证的路由
	registerAuthenticatedRoutes(api, container)
}

// registerPublicRoutes 注册公共路由
func registerPublicRoutes(
	api *gin.RouterGroup,
	container *container.ServiceContainer,
) {
	// 添加IP限流中间件 - 每秒允许10个请求，最多突发20个请求
	api.Use(middleware.IPRateLimiter(10, 20))

	// 健康检查路由
	api.GET("/ping", controllers.HandleHealthFunc(container, "ping"))
	api.GET("/health", controllers.HandleHealthFunc(container, "ping")) // 添加兼容Docker健康检查的路由

	// 健康状态路由组
	healthGroup := api.Group("/health")
	healthGroup.GET("/status", controllers.HandleHealthFunc(container, "status"))
	healthGroup.GET("/cache-stats", controllers.HandleHealthFunc(container, "cacheStats"))

	// 认证路由
	api.POST("/auth/login", controllers.HandleJWTFunc(container, "login"))
	// 阿里云RTC路由
	rtcGroup := api.Group("/rtc")
	rtcGroup.Use(middleware.PathRateLimiter(5, 10)) // 每秒5个请求，最多突发10个
	rtcGroup.POST("/token", controllers.HandleRTCFunc(container, "getToken"))
	rtcGroup.POST("/call", controllers.HandleRTCFunc(container, "startCall"))
	// 腾讯云RTC路由
	trtcGroup := api.Group("/trtc")
	trtcGroup.Use(middleware.PathRateLimiter(5, 10)) // 每秒5个请求，最多突发10个
	trtcGroup.POST("/usersig", controllers.HandleTencentRTCFunc(container, "getUserSig"))
	trtcGroup.POST("/call", controllers.HandleTencentRTCFunc(container, "startCall"))

	// MQTT通话和消息路由组 - 更新以匹配API文档
	mqttGroup := api.Group("/mqtt")
	mqttGroup.Use(middleware.PathRateLimiter(20, 40))                                                                                                             // 每秒20个请求，最多突发40个
	mqttGroup.POST("/call", controllers.HandleMQTTCallFunc(container, "initiateCall"))                                                                            // 发起通话，支持可选的户号参数或住户电话
	mqttGroup.POST("/controller/device", controllers.HandleMQTTCallFunc(container, "callerAction"))                                                               // 修改路径从caller-action到controller/device
	mqttGroup.POST("/controller/resident", controllers.HandleMQTTCallFunc(container, "calleeAction"))                                                             // 修改路径从callee-action到controller/resident
	mqttGroup.GET("/session", middleware.Cache(middleware.CacheConfig{Expiration: 5 * time.Second}), controllers.HandleMQTTCallFunc(container, "getCallSession")) // 修改为GET请求
	mqttGroup.POST("/end-session", controllers.HandleMQTTCallFunc(container, "endCallSession"))
	mqttGroup.POST("/device/status", controllers.HandleMQTTCallFunc(container, "publishDeviceStatus"))
	mqttGroup.POST("/system/message", controllers.HandleMQTTCallFunc(container, "publishSystemMessage"))

	// 设备健康检测路由
	api.POST("/device/status", controllers.HandleDeviceFunc(container, "checkDeviceHealth"))
}

// registerAuthenticatedRoutes 注册需要认证的路由
func registerAuthenticatedRoutes(
	api *gin.RouterGroup,
	container *container.ServiceContainer,
) {
	// 添加认证中间件
	auth := api.Group("/")
	auth.Use(middleware.AuthenticateSystemAdmin())

	// 添加通用限流中间件 - 每秒30个请求，最多突发50个请求
	auth.Use(middleware.IPRateLimiter(30, 50))

	// 管理员路由
	adminGroup := auth.Group("/admin")
	adminGroup.GET("", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleAdminFunc(container, "getAdmins"))
	adminGroup.GET("/:id", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleAdminFunc(container, "getAdmin"))
	adminGroup.POST("", controllers.HandleAdminFunc(container, "createAdmin"))
	adminGroup.PUT("/:id", controllers.HandleAdminFunc(container, "updateAdmin"))
	adminGroup.DELETE("/:id", controllers.HandleAdminFunc(container, "deleteAdmin"))

	// 设备路由
	devicesGroup := auth.Group("/devices")
	{
		devicesGroup.GET("", middleware.Cache(middleware.CacheConfig{Expiration: 30 * time.Second}), controllers.HandleDeviceFunc(container, "getDevices"))
		devicesGroup.GET("/:id", middleware.Cache(middleware.CacheConfig{Expiration: 30 * time.Second}), controllers.HandleDeviceFunc(container, "getDevice"))
		devicesGroup.POST("", controllers.HandleDeviceFunc(container, "createDevice"))
		devicesGroup.PUT("/:id", controllers.HandleDeviceFunc(container, "updateDevice"))
		devicesGroup.DELETE("/:id", controllers.HandleDeviceFunc(container, "deleteDevice"))
		devicesGroup.GET("/:id/status", controllers.HandleDeviceFunc(container, "getDeviceStatus"))
		devicesGroup.POST("/:id/building", controllers.HandleDeviceFunc(container, "associateDeviceWithBuilding"))
		devicesGroup.GET("/:id/households", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleDeviceFunc(container, "getDeviceHouseholds"))
		devicesGroup.POST("/:id/households", controllers.HandleDeviceFunc(container, "associateDeviceWithHousehold"))
		devicesGroup.DELETE("/:id/households", controllers.HandleDeviceFunc(container, "removeDeviceHouseholdAssociation"))
	}

	// 居民路由
	residentGroup := auth.Group("/residents")
	residentGroup.GET("", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleResidentFunc(container, "getResidents"))
	residentGroup.GET("/:id", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleResidentFunc(container, "getResident"))
	residentGroup.POST("", controllers.HandleResidentFunc(container, "createResident"))
	residentGroup.PUT("/:id", controllers.HandleResidentFunc(container, "updateResident"))
	residentGroup.DELETE("/:id", controllers.HandleResidentFunc(container, "deleteResident"))

	// 物业员工路由
	staffGroup := auth.Group("/staffs")
	staffGroup.GET("", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleStaffFunc(container, "getStaff"))
	staffGroup.GET("/with-devices", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleStaffFunc(container, "getStaffWithDevices"))
	staffGroup.GET("/:id", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleStaffFunc(container, "getStaffByID"))
	staffGroup.POST("", controllers.HandleStaffFunc(container, "createStaff"))
	staffGroup.PUT("/:id", controllers.HandleStaffFunc(container, "updateStaff"))
	staffGroup.DELETE("/:id", controllers.HandleStaffFunc(container, "deleteStaff"))

	// 通话记录路由
	callRecordGroup := auth.Group("/call-records")
	callRecordGroup.GET("", middleware.Cache(middleware.CacheConfig{Expiration: 30 * time.Second}), controllers.HandleCallRecordFunc(container, "getCallRecords"))
	callRecordGroup.GET("/statistics", middleware.Cache(middleware.CacheConfig{Expiration: 5 * time.Minute}), controllers.HandleCallRecordFunc(container, "getCallStatistics"))
	callRecordGroup.GET("/device/:deviceId", middleware.Cache(middleware.CacheConfig{Expiration: 30 * time.Second}), controllers.HandleCallRecordFunc(container, "getDeviceCallRecords"))
	callRecordGroup.GET("/resident/:residentId", middleware.Cache(middleware.CacheConfig{Expiration: 30 * time.Second}), controllers.HandleCallRecordFunc(container, "getResidentCallRecords"))
	callRecordGroup.GET("/session", middleware.Cache(middleware.CacheConfig{Expiration: 5 * time.Second}), controllers.HandleCallRecordFunc(container, "getCallSession"))
	callRecordGroup.GET("/:id", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleCallRecordFunc(container, "getCallRecordByID"))
	callRecordGroup.POST("/:id/feedback", controllers.HandleCallRecordFunc(container, "submitCallFeedback"))

	// 紧急情况路由
	emergencyGroup := auth.Group("/emergency")
	emergencyGroup.GET("", middleware.Cache(middleware.CacheConfig{Expiration: 10 * time.Second}), controllers.HandleEmergencyFunc(container, "getEmergencyLogs"))
	emergencyGroup.GET("/:id", middleware.Cache(middleware.CacheConfig{Expiration: 30 * time.Second}), controllers.HandleEmergencyFunc(container, "getEmergencyLogByID"))
	emergencyGroup.PUT("/:id", controllers.HandleEmergencyFunc(container, "updateEmergencyLog"))
	emergencyGroup.POST("/trigger", controllers.HandleEmergencyFunc(container, "triggerEmergency"))
	emergencyGroup.POST("/alarm", controllers.HandleEmergencyFunc(container, "triggerAlarm"))
	emergencyGroup.GET("/contacts", middleware.Cache(middleware.CacheConfig{Expiration: 5 * time.Minute}), controllers.HandleEmergencyFunc(container, "getEmergencyContacts"))
	emergencyGroup.POST("/notify-all", controllers.HandleEmergencyFunc(container, "notifyAllUsers"))
	emergencyGroup.POST("/unlock-all", controllers.HandleEmergencyFunc(container, "emergencyUnlockAll"))

	// 楼号路由
	buildingGroup := auth.Group("/buildings")
	buildingGroup.GET("", middleware.Cache(middleware.CacheConfig{Expiration: 5 * time.Minute}), controllers.HandleBuildingFunc(container, "getBuildings"))
	buildingGroup.GET("/:id", middleware.Cache(middleware.CacheConfig{Expiration: 5 * time.Minute}), controllers.HandleBuildingFunc(container, "getBuilding"))
	buildingGroup.POST("", controllers.HandleBuildingFunc(container, "createBuilding"))
	buildingGroup.PUT("/:id", controllers.HandleBuildingFunc(container, "updateBuilding"))
	buildingGroup.DELETE("/:id", controllers.HandleBuildingFunc(container, "deleteBuilding"))
	buildingGroup.GET("/:id/devices", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleBuildingFunc(container, "getBuildingDevices"))
	buildingGroup.GET("/:id/households", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleBuildingFunc(container, "getBuildingHouseholds"))

	// 户号路由
	householdGroup := auth.Group("/households")
	householdGroup.GET("", middleware.Cache(middleware.CacheConfig{Expiration: 5 * time.Minute}), controllers.HandleHouseholdFunc(container, "getHouseholds"))
	householdGroup.GET("/:id", middleware.Cache(middleware.CacheConfig{Expiration: 5 * time.Minute}), controllers.HandleHouseholdFunc(container, "getHousehold"))
	householdGroup.POST("", controllers.HandleHouseholdFunc(container, "createHousehold"))
	householdGroup.PUT("/:id", controllers.HandleHouseholdFunc(container, "updateHousehold"))
	householdGroup.DELETE("/:id", controllers.HandleHouseholdFunc(container, "deleteHousehold"))
	householdGroup.GET("/:id/devices", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleHouseholdFunc(container, "getHouseholdDevices"))
	householdGroup.GET("/:id/residents", middleware.Cache(middleware.CacheConfig{Expiration: 1 * time.Minute}), controllers.HandleHouseholdFunc(container, "getHouseholdResidents"))
	householdGroup.POST("/:id/devices", controllers.HandleHouseholdFunc(container, "associateHouseholdWithDevice"))
	householdGroup.DELETE("/:id/devices/:device_id", controllers.HandleHouseholdFunc(container, "removeHouseholdDeviceAssociation"))
}
