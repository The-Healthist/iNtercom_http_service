package handler

import (
	"fmt"
	"intercom_http_service/internal/service"
	"intercom_http_service/internal/errcode"
	"intercom_http_service/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// InterfaceRTCController 定义RTC控制器接口
type InterfaceRTCController interface {
	GetToken()
	StartCall()
}

// RTCController 处理RTC相关的请求
type RTCController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewRTCController 创建一个新的RTC控制器
func NewRTCController(ctx *gin.Context, container *service.ServiceContainer) *RTCController {
	return &RTCController{
		Ctx:       ctx,
		Container: container,
	}
}

// TokenRequest 表示RTC令牌请求
type TokenRequest struct {
	ChannelID string `json:"channel_id" binding:"required" example:"room123"`
	UserID    string `json:"user_id" binding:"required" example:"user456"`
}

// HandleRTCFunc 返回一个处理RTC请求的Gin处理函数
func HandleRTCFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewRTCController(ctx, container)

		switch method {
		case "getToken":
			controller.GetToken()
		case "startCall":
			controller.StartCall()
		default:
			errcode.FailWithMessage(ctx, errcode.ErrBind, "无效的方法", nil)
		}
	}
}

// 1. GetToken 处理获取RTC令牌的请求
// @Summary      Get RTC Token
// @Description  Get a RTC token for real-time communication
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Param        request body TokenRequest true "Token request parameters"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/rtc/token [post]
func (c *RTCController) GetToken() {
	var req TokenRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数", nil)
		return
	}

	// 获取服务
	rtcService := c.Container.GetService("rtc").(service.InterfaceRTCService)

	// 生成新令牌
	tokenInfo, err := rtcService.GetToken(req.ChannelID, req.UserID)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "生成令牌失败", nil)
		return
	}

	// 构建与示例格式完全匹配的响应
	errcode.Success(c.Ctx, gin.H{
		"token":       tokenInfo.Token,
		"channel_id":  tokenInfo.ChannelID,
		"user_id":     tokenInfo.UserID,
		"expire_time": tokenInfo.ExpireTime.Format(time.RFC3339),
		"rtc_app_id":  tokenInfo.AppID, // 添加RTC应用ID
	})
}

// CallRequest 表示发起通话的请求
type CallRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	ResidentID string `json:"resident_id" binding:"required"`
}

// 2. StartCall 处理发起通话的请求
// @Summary      Start Video Call
// @Description  Initiate a video call between a device and a resident
// @Tags         RTC
// @Accept       json
// @Produce      json
// @Param        request body CallRequest true "Call request parameters"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/rtc/call [post]
func (c *RTCController) StartCall() {
	var req CallRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数", nil)
		return
	}

	rtcService := c.Container.GetService("rtc").(service.InterfaceRTCService)

	// 创建通话通道
	channelID, err := rtcService.CreateVideoCall(req.DeviceID, req.ResidentID)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "发起通话失败", nil)
		return
	}

	// 为双方生成令牌
	deviceToken, err := rtcService.GetToken(channelID, req.DeviceID)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "生成设备令牌失败", nil)
		return
	}

	residentToken, err := rtcService.GetToken(channelID, req.ResidentID)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "生成居民令牌失败", nil)
		return
	}

	// 生成设备的用户名（游客+6位随机数）
	deviceUsername := fmt.Sprintf("游客%06d", utils.RandomInt32()%1000000)

	// 这里应该从数据库查询居民的用户名
	// 暂时使用residentID作为用户名
	residentUsername := req.ResidentID

	// TODO: 从数据库查询真实的居民用户名
	// db := c.Container.GetDB()
	// var resident model.Resident
	// if err := db.Where("id = ?", req.ResidentID).First(&resident).Error; err == nil {
	//     residentUsername = resident.Username
	// }

	// 构建新的标准响应格式
	errcode.Success(c.Ctx, gin.H{
		"channel_id": channelID,
		"rtc_app_id": deviceToken.AppID, // 添加RTC应用ID
		"device": gin.H{
			"id":          req.DeviceID,
			"token":       deviceToken.Token,
			"expire_time": deviceToken.ExpireTime.Format(time.RFC3339),
			"username":    deviceUsername,
		},
		"resident": gin.H{
			"id":          req.ResidentID,
			"token":       residentToken.Token,
			"expire_time": residentToken.ExpireTime.Format(time.RFC3339),
			"username":    residentUsername,
		},
	})
}
