package handler

import (
	"intercom_http_service/internal/config"
	"intercom_http_service/internal/errcode"
	"intercom_http_service/internal/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// InterfaceMQTTCallController 定义MQTT通话控制器接口
type InterfaceMQTTCallController interface {
	InitiateCall()
	CallerAction()
	CalleeAction()
	GetCallSession()
	EndCallSession()
	PublishDeviceStatus()
	PublishSystemMessage()
}

// MQTTCallController MQTT通话控制器实现
type MQTTCallController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewMQTTCallController 创建一个新的MQTT通话控制器
func NewMQTTCallController(ctx *gin.Context, container *service.ServiceContainer) InterfaceMQTTCallController {
	return &MQTTCallController{
		Ctx:       ctx,
		Container: container,
	}
}

// 请求结构体定义
type (
	// InitiateCallRequest 发起通话请求
	InitiateCallRequest struct {
		DeviceID        string `json:"device_id" binding:"required" example:"5"`      // 设备ID
		HouseholdNumber string `json:"household_number,omitempty" example:"MQTT-101"` // 可选，指定户号
		Timestamp       int64  `json:"timestamp,omitempty" example:"1651234567890"`   // 可选时间戳
	}

	// CallActionRequest 通话控制请求
	CallActionRequest struct {
		Action     string `json:"action" binding:"required" example:"answered"`
		CallID     string `json:"call_id" binding:"required" example:"mqtt-call-20250510-abcdef123456"`
		Timestamp  int64  `json:"timestamp,omitempty" example:"1651234567890"`
		Reason     string `json:"reason,omitempty" example:"user_busy"`
		ResidentID string `json:"resident_id,omitempty" example:"6"`
	}

	// GetCallSessionRequest 获取通话会话请求
	GetCallSessionRequest struct {
		CallID string `json:"call_id" binding:"required"`
	}

	// EndCallSessionRequest 结束通话会话请求
	EndCallSessionRequest struct {
		CallID string `json:"call_id" binding:"required" example:"mqtt-call-20250510-abcdef123456"`
		Reason string `json:"reason,omitempty" example:"call_completed"`
	}

	// PublishDeviceStatusRequest 发布设备状态请求
	PublishDeviceStatusRequest struct {
		DeviceID   string                 `json:"device_id" binding:"required" example:"5"`
		Online     bool                   `json:"online" example:"true"`
		Battery    int                    `json:"battery" example:"85"`
		Properties map[string]interface{} `json:"properties,omitempty"`
	}

	// PublishSystemMessageRequest 发布系统消息请求
	PublishSystemMessageRequest struct {
		Type      string                 `json:"type" binding:"required" example:"notification"`
		Level     string                 `json:"level" binding:"required" example:"info"`
		Message   string                 `json:"message" binding:"required" example:"系统将于今晚22:00进行升级维护"`
		Data      map[string]interface{} `json:"data,omitempty"`
		Timestamp int64                  `json:"timestamp,omitempty" example:"1651234567890"`
	}

	// CallSessionResponse 通话会话响应
	CallSessionResponse struct {
		CallID       string    `json:"call_id" example:"mqtt-call-20250510-abcdef123456"`
		DeviceID     string    `json:"device_id" example:"5"`
		ResidentID   string    `json:"resident_id" example:"6"`
		StartTime    time.Time `json:"start_time" example:"2025-05-10T15:04:05Z"`
		Status       string    `json:"status" example:"connected"`
		LastActivity time.Time `json:"last_activity" example:"2025-05-10T15:09:10Z"`
		TencentRTC   *TRTCInfo `json:"tencen_rtc,omitempty"`
	}

	// TRTCInfo 腾讯云RTC信息
	TRTCInfo struct {
		SDKAppID   int    `json:"sdk_app_id" example:"1400000001"`
		UserID     string `json:"user_id" example:"5"`
		UserSig    string `json:"user_sig" example:"eJwtzM1Og0AUhmG..."`
		RoomID     string `json:"room_id" example:"room_5_6_1746870072"`
		RoomIDType string `json:"room_id_type" example:"string"`
	}

	// InitiateCallResponse 发起通话响应
	InitiateCallResponse struct {
		CallID            string    `json:"call_id" example:"mqtt-call-20250510-abcdef123456"`
		DeviceDeviceID    string    `json:"device_device_id" example:"5"`
		TargetResidentIDs []string  `json:"target_resident_ids" example:"[\"6\",\"7\"]"`
		Timestamp         int64     `json:"timestamp" example:"1651234567890"`
		TencentRTC        *TRTCInfo `json:"tencen_rtc,omitempty"`
		CallInfo          *CallInfo `json:"call_info,omitempty"`
	}

	// CallInfo 通话信息
	CallInfo struct {
		CallID    string `json:"call_id" example:"mqtt-call-20250510-abcdef123456"`
		Action    string `json:"action" example:"answered"`
		Reason    string `json:"reason,omitempty" example:"user_busy"`
		Timestamp int64  `json:"timestamp,omitempty" example:"1651234567890"`
	}
)

// HandleMQTTCallFunc 返回一个处理MQTT通话请求的Gin处理函数
func HandleMQTTCallFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewMQTTCallController(ctx, container)

		switch method {
		case "initiateCall":
			controller.InitiateCall()
		case "callerAction":
			controller.CallerAction()
		case "calleeAction":
			controller.CalleeAction()
		case "getCallSession":
			controller.GetCallSession()
		case "endCallSession":
			controller.EndCallSession()
		case "publishDeviceStatus":
			controller.PublishDeviceStatus()
		case "publishSystemMessage":
			controller.PublishSystemMessage()
		default:
			errcode.FailWithMessage(ctx, errcode.ErrBind, "无效的方法", nil)
		}
	}
}

// 1. InitiateCall 发起通话
// @Summary      发起MQTT通话
// @Description  通过MQTT向设备关联的住户发起视频通话请求。如果提供了household_number参数，则呼叫该户号下的所有住户；如果未提供，则呼叫该设备绑定的户号下的所有住户。
// @Tags         MQTT
// @Accept       json
// @Produce      json
// @Param        request body InitiateCallRequest true "通话请求参数：支持device_id(必填)、household_number(可选)、timestamp(可选)参数"
// @Success      200  {object}  InitiateCallResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /mqtt/call [post]
func (c *MQTTCallController) InitiateCall() {
	var req InitiateCallRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.HandleError(http.StatusBadRequest, "无效的请求参数", err)
		return
	}

	mqttCallService := c.Container.GetService("mqtt_call").(service.InterfaceMQTTCallService)

	var callID string
	var err error
	var targetResidentIDs []string

	if req.HouseholdNumber != "" {
		// 如果提供了户号，向该户号下的所有居民发起呼叫
		callID, targetResidentIDs, err = mqttCallService.InitiateCallToHousehold(req.DeviceID, req.HouseholdNumber)
	} else {
		// 否则，向关联该设备的所有户号下的所有居民发起呼叫
		callID, targetResidentIDs, err = mqttCallService.InitiateCallToAll(req.DeviceID)
	}

	if err != nil {
		c.HandleError(http.StatusInternalServerError, "发起通话失败", err)
		return
	}

	// 使用当前时间戳或者请求中提供的时间戳
	timestamp := time.Now().UnixMilli()
	if req.Timestamp > 0 {
		timestamp = req.Timestamp
	}

	// 创建响应
	response := InitiateCallResponse{
		CallID:            callID,
		DeviceDeviceID:    req.DeviceID,
		TargetResidentIDs: targetResidentIDs,
		Timestamp:         timestamp,
		CallInfo: &CallInfo{
			CallID:    callID,
			Action:    "ringing",
			Timestamp: timestamp,
		},
	}

	// 优先复用服务层实际会话中的房间号，避免与MQTT下发数据不一致。
	roomID := ""
	if session, exists := mqttCallService.GetCallSession(callID); exists {
		roomID = session.TRTCInfo.RoomID
	}

	// 如果系统配置了腾讯云RTC，添加RTC信息
	config := c.Container.GetService("config").(*config.Config)
	if config.TencentRTCEnabled {
		// 获取RTC服务
		rtcService := c.Container.GetService("tencent_rtc").(service.InterfaceTencentRTCService)

		// 为设备生成UserSig
		deviceUserID := req.DeviceID
		tokenInfo, err := rtcService.GetUserSig(deviceUserID)
		if err == nil {
			if roomID == "" && len(targetResidentIDs) > 0 {
				roomID = service.BuildSharedTRTCRoomID(req.DeviceID, targetResidentIDs[0], timestamp/1000)
			}

			response.TencentRTC = &TRTCInfo{
				SDKAppID:   config.TencentSDKAppID,
				UserID:     deviceUserID,
				UserSig:    tokenInfo.UserSig,
				RoomID:     roomID,
				RoomIDType: "string",
			}
		}
	}

	c.HandleSuccess(response)
}

// 2. CallerAction 处理呼叫方动作
// @Summary      处理MQTT呼叫方动作
// @Description  处理设备端通话动作，支持的动作类型包括：hangup(挂断)、cancelled(取消呼叫)
// @Tags         MQTT
// @Accept       json
// @Produce      json
// @Param        request body CallActionRequest true "设备通话动作请求，包含call_id和action字段"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /mqtt/controller/device [post]
func (c *MQTTCallController) CallerAction() {
	var req CallActionRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.HandleError(http.StatusBadRequest, "无效的请求参数", err)
		return
	}

	// 验证动作类型
	validActions := map[string]bool{"hangup": true, "cancelled": true}
	if !validActions[req.Action] {
		c.HandleError(http.StatusBadRequest, "不支持的动作类型", nil)
		return
	}

	mqttCallService := c.Container.GetService("mqtt_call").(service.InterfaceMQTTCallService)
	if err := mqttCallService.HandleCallerAction(req.CallID, req.Action, req.Reason); err != nil {
		c.HandleError(http.StatusInternalServerError, "处理呼叫方动作失败", err)
		return
	}

	c.HandleSuccess(nil)
}

// 3. CalleeAction 处理被呼叫方动作
// @Summary      处理MQTT被呼叫方动作
// @Description  处理居民端通话动作，支持的动作类型包括：rejected(拒绝)、answered(接听)、hangup(挂断)、timeout(超时)
// @Tags         MQTT
// @Accept       json
// @Produce      json
// @Param        request body CallActionRequest true "居民通话动作请求，包含call_id、action和可选的reason字段"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /mqtt/controller/resident [post]
func (c *MQTTCallController) CalleeAction() {
	var req CallActionRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.HandleError(http.StatusBadRequest, "无效的请求参数", err)
		return
	}

	// 验证动作类型
	validActions := map[string]bool{"rejected": true, "answered": true, "hangup": true, "timeout": true}
	if !validActions[req.Action] {
		c.HandleError(http.StatusBadRequest, "不支持的动作类型", nil)
		return
	}

	mqttCallService := c.Container.GetService("mqtt_call").(service.InterfaceMQTTCallService)
	if err := mqttCallService.HandleCalleeAction(req.CallID, req.Action, req.Reason, req.ResidentID); err != nil {
		c.HandleError(http.StatusInternalServerError, "处理被呼叫方动作失败", err)
		return
	}

	c.HandleSuccess(nil)
}

// 4. GetCallSession 获取通话会话
// @Summary      获取MQTT通话会话
// @Description  获取通话会话信息及TRTC房间详情，包括设备ID、住户ID、通话状态、开始时间等
// @Tags         MQTT
// @Accept       json
// @Produce      json
// @Param        call_id query string true "通话会话ID" example:"mqtt-call-20250510-abcdef123456"
// @Success      200  {object}  CallSessionResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /mqtt/session [get]
func (c *MQTTCallController) GetCallSession() {
	callID := c.Ctx.Query("call_id")
	if callID == "" {
		c.HandleError(http.StatusBadRequest, "缺少必要参数：call_id", nil)
		return
	}

	mqttCallService := c.Container.GetService("mqtt_call").(service.InterfaceMQTTCallService)
	session, exists := mqttCallService.GetCallSession(callID)
	if !exists {
		c.HandleError(http.StatusNotFound, "通话会话不存在", nil)
		return
	}

	// 创建响应对象
	response := CallSessionResponse{
		CallID:       session.CallID,
		DeviceID:     session.DeviceID,
		ResidentID:   session.ResidentID,
		StartTime:    session.StartTime,
		Status:       session.Status,
		LastActivity: session.LastActivity,
	}

	// 如果系统配置了腾讯云RTC，添加RTC信息
	config := c.Container.GetService("config").(*config.Config)
	if config.TencentRTCEnabled {
		// 获取RTC服务
		rtcService := c.Container.GetService("tencent_rtc").(service.InterfaceTencentRTCService)

		// 为设备生成UserSig
		deviceUserID := "device_" + session.DeviceID
		tokenInfo, err := rtcService.GetUserSig(deviceUserID)
		if err == nil {
			response.TencentRTC = &TRTCInfo{
				SDKAppID:   config.TencentSDKAppID,
				UserID:     deviceUserID,
				UserSig:    tokenInfo.UserSig,
				RoomID:     session.CallID, // 使用callID作为房间ID
				RoomIDType: "string",
			}
		}
	}

	c.HandleSuccess(response)
}

// 5. EndCallSession 结束通话会话
// @Summary      结束MQTT通话会话
// @Description  强制结束通话会话并通知所有参与方，适用于系统管理或异常情况下的通话强制终止
// @Tags         MQTT
// @Accept       json
// @Produce      json
// @Param        request body EndCallSessionRequest true "结束通话会话请求，包含call_id和可选的reason字段"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /mqtt/end-session [post]
func (c *MQTTCallController) EndCallSession() {
	var req EndCallSessionRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.HandleError(http.StatusBadRequest, "无效的请求参数", err)
		return
	}

	mqttCallService := c.Container.GetService("mqtt_call").(service.InterfaceMQTTCallService)
	if err := mqttCallService.EndCallSession(req.CallID, req.Reason); err != nil {
		c.HandleError(http.StatusInternalServerError, "结束通话会话失败", err)
		return
	}

	c.HandleSuccess(nil)
}

// 6. PublishDeviceStatus 发布设备状态
// @Summary      更新设备状态
// @Description  更新设备状态信息，包括在线状态、电池电量和其他自定义属性，无需MQTT连接，系统会通过MQTT推送给相关订阅方
// @Tags         Device
// @Accept       json
// @Produce      json
// @Param        request body PublishDeviceStatusRequest true "设备状态信息：必须包含device_id、online、battery字段，可选包含properties自定义属性"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /mqtt/device/status [post]
func (c *MQTTCallController) PublishDeviceStatus() {
	var req PublishDeviceStatusRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.HandleError(http.StatusBadRequest, "无效的请求参数", err)
		return
	}

	mqttCallService := c.Container.GetService("mqtt_call").(service.InterfaceMQTTCallService)
	status := map[string]interface{}{
		"device_id":   req.DeviceID,
		"online":      req.Online,
		"battery":     req.Battery,
		"properties":  req.Properties,
		"last_update": time.Now().UnixMilli(),
	}

	if err := mqttCallService.PublishDeviceStatus(req.DeviceID, status); err != nil {
		c.HandleError(http.StatusInternalServerError, "发布设备状态失败", err)
		return
	}

	c.HandleSuccess(nil)
}

// 7. PublishSystemMessage 发布系统消息
// @Summary      发布系统消息
// @Description  通过MQTT发布系统消息，支持info、warning、error三种级别，消息会推送给所有订阅相关主题的客户端
// @Tags         MQTT
// @Accept       json
// @Produce      json
// @Param        request body PublishSystemMessageRequest true "系统消息信息：必须包含type、level、message字段，level支持info、warning、error三种级别"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /mqtt/system/message [post]
func (c *MQTTCallController) PublishSystemMessage() {
	var req PublishSystemMessageRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.HandleError(http.StatusBadRequest, "无效的请求参数", err)
		return
	}

	// 验证消息级别
	validLevels := map[string]bool{"info": true, "warning": true, "error": true}
	if !validLevels[req.Level] {
		c.HandleError(http.StatusBadRequest, "无效的消息级别", nil)
		return
	}

	// 使用当前时间戳或者请求中提供的时间戳
	timestamp := time.Now().UnixMilli()
	if req.Timestamp > 0 {
		timestamp = req.Timestamp
	}

	mqttCallService := c.Container.GetService("mqtt_call").(service.InterfaceMQTTCallService)
	message := map[string]interface{}{
		"type":      req.Type,
		"level":     req.Level,
		"message":   req.Message,
		"data":      req.Data,
		"timestamp": timestamp,
	}

	if err := mqttCallService.PublishSystemMessage(req.Type, message); err != nil {
		c.HandleError(http.StatusInternalServerError, "发布系统消息失败", err)
		return
	}

	c.HandleSuccess(nil)
}

// HandleSuccess 处理成功响应
func (c *MQTTCallController) HandleSuccess(data interface{}) {
	errcode.Success(c.Ctx, data)
}

// HandleError 处理错误响应
func (c *MQTTCallController) HandleError(status int, message string, err error) {
	errMessage := message
	if err != nil {
		errMessage = message + ": " + err.Error()
	}

	switch status {
	case http.StatusBadRequest:
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, errMessage, nil)
	case http.StatusNotFound:
		errcode.NotFound(c.Ctx, errMessage)
	case http.StatusInternalServerError:
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, errMessage, nil)
	default:
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, errMessage, nil)
	}
}
