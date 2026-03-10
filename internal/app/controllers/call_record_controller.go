package controllers

import (
	"intercom_http_service/internal/domain/services"
	"intercom_http_service/internal/domain/services/container"
	"intercom_http_service/internal/error/code"
	"intercom_http_service/internal/error/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// InterfaceCallRecordController 定义通话记录控制器接口
type InterfaceCallRecordController interface {
	GetCallRecords()
	GetCallRecord()
	GetCallStatistics()
	GetDeviceCallRecords()
	GetResidentCallRecords()
	SubmitCallFeedback()
	GetCallSession()
}

// CallRecordController 处理通话记录相关的请求
type CallRecordController struct {
	Ctx       *gin.Context
	Container *container.ServiceContainer
}

// NewCallRecordController 创建一个新的通话记录控制器
func NewCallRecordController(ctx *gin.Context, container *container.ServiceContainer) *CallRecordController {
	return &CallRecordController{
		Ctx:       ctx,
		Container: container,
	}
}

// CallFeedbackRequest 表示通话质量反馈请求
type CallFeedbackRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5" example:"4"` // 1-5 星评分
	Comment string `json:"comment" example:"通话质量良好，声音清晰"`                     // 可选评论
	Issues  string `json:"issues" example:"偶尔有一点延迟"`                          // 问题描述
}

// HandleCallRecordFunc 返回一个处理通话记录请求的Gin处理函数
func HandleCallRecordFunc(container *container.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewCallRecordController(ctx, container)

		switch method {
		case "getCallRecords":
			controller.GetCallRecords()
		case "getCallRecord":
			controller.GetCallRecord()
		case "getCallStatistics":
			controller.GetCallStatistics()
		case "getDeviceCallRecords":
			controller.GetDeviceCallRecords()
		case "getResidentCallRecords":
			controller.GetResidentCallRecords()
		case "submitCallFeedback":
			controller.SubmitCallFeedback()
		case "getCallSession":
			controller.GetCallSession()
		default:
			response.FailWithMessage(ctx, code.ErrBind, "无效的方法", nil)
		}
	}
}

// 1. GetCallRecords 获取通话记录列表
// @Summary      获取通话记录列表
// @Description  获取系统中所有通话记录，支持分页
// @Tags         CallRecord
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "页码，默认为1" example:"1"
// @Param        page_size query int false "每页条数，默认为10" example:"10"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  ErrorResponse
// @Router       /call_records [get]
func (c *CallRecordController) GetCallRecords() {
	// 获取分页参数
	page, _ := strconv.Atoi(c.Ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.Ctx.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	callRecordService := c.Container.GetService("call_record").(services.InterfaceCallRecordService)

	calls, total, err := callRecordService.GetAllCallRecords(page, pageSize)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "获取通话记录失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"records":     calls,
	})
}

// 2. GetCallRecord 获取单个通话记录
// @Summary      获取通话记录详情
// @Description  根据ID获取特定通话记录的详细信息
// @Tags         CallRecord
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "通话记录ID" example:"1"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /call_records/{id} [get]
func (c *CallRecordController) GetCallRecord() {
	id := c.Ctx.Param("id")
	recordID, err := strconv.Atoi(id)
	if err != nil {
		response.ParamError(c.Ctx, "无效的通话记录ID")
		return
	}

	callRecordService := c.Container.GetService("call_record").(services.InterfaceCallRecordService)

	record, err := callRecordService.GetCallRecordByID(uint(recordID))
	if err != nil {
		response.NotFound(c.Ctx, err.Error())
		return
	}

	response.Success(c.Ctx, record)
}

// 3. GetCallStatistics 获取通话统计信息
// @Summary      获取通话统计信息
// @Description  获取通话统计信息，包括总数、已接、未接等
// @Tags         CallRecord
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  ErrorResponse
// @Router       /call_records/statistics [get]
func (c *CallRecordController) GetCallStatistics() {
	callRecordService := c.Container.GetService("call_record").(services.InterfaceCallRecordService)

	statistics, err := callRecordService.GetCallStatistics()
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "获取通话统计信息失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, statistics)
}

// 4. GetDeviceCallRecords 获取指定设备的通话记录
// @Summary      获取设备通话记录
// @Description  获取特定设备的通话记录，支持分页
// @Tags         CallRecord
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        deviceId path int true "设备ID" example:"1"
// @Param        page query int false "页码，默认为1" example:"1"
// @Param        page_size query int false "每页条数，默认为10" example:"10"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /call_records/device/{deviceId} [get]
func (c *CallRecordController) GetDeviceCallRecords() {
	deviceID := c.Ctx.Param("deviceId")
	id, err := strconv.Atoi(deviceID)
	if err != nil {
		response.ParamError(c.Ctx, "无效的设备ID")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.Ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.Ctx.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	callRecordService := c.Container.GetService("call_record").(services.InterfaceCallRecordService)

	calls, total, err := callRecordService.GetCallRecordsByDeviceID(uint(id), page, pageSize)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "获取设备通话记录失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"records":     calls,
	})
}

// 5. GetResidentCallRecords 获取指定居民的通话记录
// @Summary      获取居民通话记录
// @Description  获取特定居民的通话记录，支持分页
// @Tags         CallRecord
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        residentId path int true "居民ID" example:"1"
// @Param        page query int false "页码，默认为1" example:"1"
// @Param        page_size query int false "每页条数，默认为10" example:"10"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /call_records/resident/{residentId} [get]
func (c *CallRecordController) GetResidentCallRecords() {
	residentID := c.Ctx.Param("residentId")
	id, err := strconv.Atoi(residentID)
	if err != nil {
		response.ParamError(c.Ctx, "无效的居民ID")
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.Ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.Ctx.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	callRecordService := c.Container.GetService("call_record").(services.InterfaceCallRecordService)

	calls, total, err := callRecordService.GetCallRecordsByResidentID(uint(id), page, pageSize)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "获取居民通话记录失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"records":     calls,
	})
}

// 6. SubmitCallFeedback 提交通话质量反馈
// @Summary      提交通话反馈
// @Description  为特定通话记录提交质量反馈
// @Tags         CallRecord
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "通话记录ID" example:"1"
// @Param        request body CallFeedbackRequest true "反馈信息"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /call_records/{id}/feedback [post]
func (c *CallRecordController) SubmitCallFeedback() {
	callID := c.Ctx.Param("id")
	id, err := strconv.Atoi(callID)
	if err != nil {
		response.ParamError(c.Ctx, "无效的通话记录ID")
		return
	}

	var req CallFeedbackRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	callRecordService := c.Container.GetService("call_record").(services.InterfaceCallRecordService)

	feedback := &services.CallFeedback{
		CallID:  uint(id),
		Rating:  req.Rating,
		Comment: req.Comment,
		Issues:  req.Issues,
	}

	if err := callRecordService.SubmitCallFeedback(feedback); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "提交反馈失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, nil)
}

// GetCallSession 通过MQTT会话ID获取通话记录
// @Summary      通过CallID获取通话记录
// @Description  通过CallID（MQTT会话ID）获取特定通话记录的详细信息
// @Tags         CallRecord
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        call_id query string true "通话会话ID（UUID）" example:"call-20250510-abcdef123456"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /call_records/session [get]
func (c *CallRecordController) GetCallSession() {
	callID := c.Ctx.Query("call_id")
	if callID == "" {
		response.ParamError(c.Ctx, "缺少必要的call_id参数")
		return
	}

	callRecordService := c.Container.GetService("call_record").(services.InterfaceCallRecordService)

	record, err := callRecordService.GetCallRecordByCallID(callID)
	if err != nil {
		response.NotFound(c.Ctx, "未找到通话记录: "+err.Error())
		return
	}

	response.Success(c.Ctx, record)
}
