package controllers

import (
	"fmt"
	"intercom_http_service/internal/domain/services"
	"intercom_http_service/internal/domain/services/container"
	"intercom_http_service/internal/error/code"
	"intercom_http_service/internal/error/response"
	"intercom_http_service/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// InterfaceTencentRTCController 定义腾讯云RTC控制器接口
type InterfaceTencentRTCController interface {
	GetUserSig()
	StartTencentCall()
}

// TencentRTCController 处理腾讯云RTC相关的请求
type TencentRTCController struct {
	Ctx       *gin.Context
	Container *container.ServiceContainer
}

// NewTencentRTCController 创建一个新的腾讯云RTC控制器
func NewTencentRTCController(ctx *gin.Context, container *container.ServiceContainer) *TencentRTCController {
	return &TencentRTCController{
		Ctx:       ctx,
		Container: container,
	}
}

// GetUserSigRequest 表示获取UserSig的请求
type GetUserSigRequest struct {
	UserID string `json:"user_id" binding:"required" example:"user123"`
}

// GetUserSig 处理获取腾讯云RTC UserSig的请求
// @Summary      Get Tencent TRTC UserSig
// @Description  Get a UserSig credential for Tencent Cloud real-time communication
// @Tags         TencentRTC
// @Accept       json
// @Produce      json
// @Param        request body GetUserSigRequest true "UserSig request parameters"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /trtc/usersig [post]
func (c *TencentRTCController) GetUserSig() {
	var req GetUserSigRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrBind, "无效的请求参数", nil)
		return
	}

	// 获取腾讯云RTC服务
	tencentRTCService := c.Container.GetService("tencent_rtc").(services.InterfaceTencentRTCService)

	// 生成UserSig
	tokenInfo, err := tencentRTCService.GetUserSig(req.UserID)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "生成UserSig失败: "+err.Error(), nil)
		return
	}

	// 返回成功响应
	response.Success(c.Ctx, gin.H{
		"sdk_app_id":  tokenInfo.SDKAppID,
		"user_id":     tokenInfo.UserID,
		"user_sig":    tokenInfo.UserSig,
		"expire_time": tokenInfo.ExpireTime.Format(time.RFC3339),
	})
}

// TencentCallRequest 表示发起腾讯云通话的请求
type TencentCallRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	ResidentID string `json:"resident_id" binding:"required"`
}

// StartTencentCall 处理发起腾讯云视频通话的请求
// @Summary      Start Tencent Video Call
// @Description  Initiate a Tencent Cloud video call between a device and a resident
// @Tags         TencentRTC
// @Accept       json
// @Produce      json
// @Param        request body TencentCallRequest true "Call request parameters"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /trtc/call [post]
func (c *TencentRTCController) StartTencentCall() {
	var req TencentCallRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrBind, "无效的请求参数", nil)
		return
	}

	// 获取腾讯云RTC服务
	tencentRTCService := c.Container.GetService("tencent_rtc").(services.InterfaceTencentRTCService)

	// 创建通话通道
	roomID, err := tencentRTCService.CreateVideoCall(req.DeviceID, req.ResidentID)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "发起通话失败: "+err.Error(), nil)
		return
	}

	// 为设备生成UserSig
	deviceToken, err := tencentRTCService.GetUserSig(req.DeviceID)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "生成设备UserSig失败: "+err.Error(), nil)
		return
	}

	// 为居民生成UserSig
	residentToken, err := tencentRTCService.GetUserSig(req.ResidentID)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "生成居民UserSig失败: "+err.Error(), nil)
		return
	}

	// 生成设备的用户名（游客+6位随机数）
	deviceUsername := fmt.Sprintf("游客%06d", utils.RandomInt32()%1000000)

	// 这里可以从数据库查询居民的用户名，暂时使用residentID
	residentUsername := req.ResidentID

	// 返回成功响应
	response.Success(c.Ctx, gin.H{
		"room_id":    roomID,
		"sdk_app_id": deviceToken.SDKAppID,
		"device": gin.H{
			"id":          req.DeviceID,
			"user_sig":    deviceToken.UserSig,
			"expire_time": deviceToken.ExpireTime.Format(time.RFC3339),
			"username":    deviceUsername,
		},
		"resident": gin.H{
			"id":          req.ResidentID,
			"user_sig":    residentToken.UserSig,
			"expire_time": residentToken.ExpireTime.Format(time.RFC3339),
			"username":    residentUsername,
		},
	})
}

// HandleTencentRTCFunc 返回一个处理腾讯云RTC请求的Gin处理函数
func HandleTencentRTCFunc(container *container.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewTencentRTCController(ctx, container)

		switch method {
		case "getUserSig":
			controller.GetUserSig()
		case "startCall":
			controller.StartTencentCall()
		default:
			response.FailWithMessage(ctx, code.ErrBind, "无效的方法", nil)
		}
	}
}
