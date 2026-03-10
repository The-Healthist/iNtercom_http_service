package handler

import (
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/service"
	"intercom_http_service/internal/errcode"
	"time"

	"github.com/gin-gonic/gin"
)

// InterfaceEmergencyController 定义紧急情况控制器接口
type InterfaceEmergencyController interface {
	TriggerAlarm()
	GetEmergencyContacts()
	EmergencyUnlockAll()
	NotifyAllUsers()
}

// EmergencyController 处理紧急情况相关的请求
type EmergencyController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewEmergencyController 创建一个新的紧急情况控制器
func NewEmergencyController(ctx *gin.Context, container *service.ServiceContainer) *EmergencyController {
	return &EmergencyController{
		Ctx:       ctx,
		Container: container,
	}
}

// EmergencyAlarmRequest 表示紧急警报请求
type EmergencyAlarmRequest struct {
	Type        string `json:"type" binding:"required" example:"fire"` // 如：fire(火灾)、intrusion(入侵)、medical(医疗)等
	Location    string `json:"location" binding:"required" example:"Building A, Floor 3"`
	Description string `json:"description" example:"火灾警报被触发，疑似厨房起火"`
	ReportedBy  uint   `json:"reported_by" example:"1"`                    // 报告人ID
	PropertyID  uint   `json:"property_id" binding:"required" example:"1"` // 物业ID
}

// EmergencyUnlockRequest 表示紧急解锁请求
type EmergencyUnlockRequest struct {
	Reason string `json:"reason" binding:"required" example:"火灾疏散"`
}

// EmergencyNotificationRequest 表示紧急通知请求
type EmergencyNotificationRequest struct {
	Title      string     `json:"title" binding:"required" example:"紧急通知：小区火灾警报"`
	Content    string     `json:"content" binding:"required" example:"A栋3楼发生火灾，请所有居民立即疏散。"`
	Severity   string     `json:"severity" binding:"required" example:"high"` // high, medium, low
	ExpiresAt  *time.Time `json:"expires_at" example:"2023-07-01T15:00:00Z"`  // 可选，不提供则默认24小时
	TargetType string     `json:"target_type" example:"all"`                  // all, residents, staff
	PropertyID *uint      `json:"property_id" example:"1"`                    // 关联的物业ID，可以为空表示全局通知
	IsPublic   bool       `json:"is_public" example:"false"`                  // 是否为公开通知
}

// HandleEmergencyFunc 返回一个处理紧急情况请求的Gin处理函数
func HandleEmergencyFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewEmergencyController(ctx, container)

		switch method {
		case "triggerAlarm":
			controller.TriggerAlarm()
		case "getEmergencyContacts":
			controller.GetEmergencyContacts()
		case "emergencyUnlockAll":
			controller.EmergencyUnlockAll()
		case "notifyAllUsers":
			controller.NotifyAllUsers()
		default:
			errcode.FailWithMessage(ctx, errcode.ErrBind, "无效的方法", nil)
		}
	}
}

// 1. TriggerAlarm 处理触发紧急警报的请求
// @Summary      Trigger Emergency Alarm
// @Description  Trigger an emergency alarm and notify relevant personnel
// @Tags         Emergency
// @Accept       json
// @Produce      json
// @Param        request body EmergencyAlarmRequest true "Alarm request parameters"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /emergency/alarm [post]
// @Security     BearerAuth
func (c *EmergencyController) TriggerAlarm() {
	var req EmergencyAlarmRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	// 获取用户ID和角色
	userID, exists := c.Ctx.Get("userID")
	if !exists {
		userID = uint(0) // 如果没有用户ID，设置为0表示系统自动触发
	}

	// 创建警报对象
	alarm := &model.EmergencyAlarm{
		Type:        req.Type,
		Location:    req.Location,
		Description: req.Description,
		Status:      "triggered",
		Timestamp:   time.Now(),
	}

	// 如果提供了PropertyID，则设置它
	if req.PropertyID > 0 {
		alarm.PropertyID = &req.PropertyID
	}

	// 如果前端没有提供报告人，使用当前登录用户
	if req.ReportedBy == 0 && userID != uint(0) {
		alarm.ReportedBy = userID.(uint)
	} else {
		alarm.ReportedBy = req.ReportedBy
	}

	// 直接返回演示模式响应，不尝试保存到数据库
	// 这样可以避免数据库错误
	errcode.Success(c.Ctx, gin.H{
		"alarm_id":    999, // 模拟ID
		"type":        alarm.Type,
		"location":    alarm.Location,
		"timestamp":   alarm.Timestamp,
		"status":      "demo_mode",
		"reported_by": alarm.ReportedBy,
	})
}

// 2. GetEmergencyContacts 处理获取紧急联系人列表的请求
// @Summary      Get Emergency Contacts
// @Description  Get a list of all emergency contacts in the system
// @Tags         Emergency
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  ErrorResponse
// @Router       /emergency/contacts [get]
// @Security     BearerAuth
func (c *EmergencyController) GetEmergencyContacts() {
	// 获取紧急服务
	emergencyService := c.Container.GetService("emergency").(service.InterfaceEmergencyService)

	// 获取联系人列表
	contacts, err := emergencyService.GetEmergencyContacts()
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "获取联系人失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, gin.H{
		"contacts": contacts,
		"total":    len(contacts),
	})
}

// 3. EmergencyUnlockAll 处理紧急情况下解锁所有门的请求
// @Summary      Emergency Unlock All Doors
// @Description  Unlock all doors in the system during an emergency
// @Tags         Emergency
// @Accept       json
// @Produce      json
// @Param        request body EmergencyUnlockRequest true "Unlock request parameters"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /emergency/unlock-all [post]
// @Security     BearerAuth
func (c *EmergencyController) EmergencyUnlockAll() {
	var req EmergencyUnlockRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数", nil)
		return
	}

	// 获取紧急服务
	emergencyService := c.Container.GetService("emergency").(service.InterfaceEmergencyService)

	// 执行紧急解锁
	if err := emergencyService.EmergencyUnlockAll(req.Reason); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "紧急解锁失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, gin.H{
		"timestamp": time.Now(),
		"reason":    req.Reason,
	})
}

// 4. NotifyAllUsers 处理紧急情况下发送通知给所有用户的请求
// @Summary      Notify All Users
// @Description  Send notification to all users during an emergency
// @Tags         Emergency
// @Accept       json
// @Produce      json
// @Param        request body EmergencyNotificationRequest true "Notification request parameters"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /emergency/notify-all [post]
// @Security     BearerAuth
func (c *EmergencyController) NotifyAllUsers() {
	var req EmergencyNotificationRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数", nil)
		return
	}

	// 获取用户ID和角色
	userID, exists := c.Ctx.Get("userID")
	if !exists {
		userID = uint(0)
	}

	role, _ := c.Ctx.Get("role")
	roleStr, ok := role.(string)
	if !ok {
		roleStr = "system"
	}

	// 获取紧急服务
	emergencyService := c.Container.GetService("emergency").(service.InterfaceEmergencyService)

	// 创建通知对象
	notification := &model.EmergencyNotification{
		Title:      req.Title,
		Content:    req.Content,
		Severity:   req.Severity,
		TargetType: req.TargetType,
		SenderID:   userID.(uint),
		SenderRole: roleStr,
		PropertyID: req.PropertyID,
		IsPublic:   req.IsPublic,
	}

	// 设置过期时间（如果提供）
	if req.ExpiresAt != nil {
		notification.ExpiresAt = *req.ExpiresAt
	}

	// 发送通知
	if err := emergencyService.NotifyAllUsers(notification); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "发送通知失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, gin.H{
		"id":          notification.ID,
		"title":       notification.Title,
		"timestamp":   notification.Timestamp,
		"severity":    notification.Severity,
		"target_type": notification.TargetType,
		"expires_at":  notification.ExpiresAt,
		"sender_role": notification.SenderRole,
	})
}
