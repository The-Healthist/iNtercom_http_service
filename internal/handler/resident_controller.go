package handler

import (
	"strings"

	"intercom_http_service/internal/errcode"
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// InterfaceResidentController 定义居民控制器接口
type InterfaceResidentController interface {
	GetResidents()
	GetResident()
	CreateResident()
	UpdateResident()
	DeleteResident()
}

// ResidentController 处理居民相关的请求
type ResidentController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewResidentController 创建一个新的居民控制器
func NewResidentController(ctx *gin.Context, container *service.ServiceContainer) *ResidentController {
	return &ResidentController{
		Ctx:       ctx,
		Container: container,
	}
}

// ResidentRequest 表示居民请求
type ResidentRequest struct {
	Name        string `json:"name" binding:"required" example:"张三"`
	Email       string `json:"email" binding:"omitempty,email" example:"zhangsan@resident.com"`
	Phone       string `json:"phone" binding:"omitempty,numeric" example:"13812345678"`
	HouseholdID uint   `json:"household_id" binding:"required" example:"1"` // 必填，关联的户号ID
}

// UpdateResidentRequest 表示更新居民请求
type UpdateResidentRequest struct {
	Name        string `json:"name" example:"李四"`
	Email       string `json:"email" binding:"omitempty,email" example:"lisi@resident.com"`
	Phone       string `json:"phone" binding:"omitempty,numeric" example:"13987654321"`
	HouseholdID *uint  `json:"household_id" example:"1"` // 可选，关联的户号ID，使用指针允许设置为null
}

// GetResidents 获取所有居民
// @Summary      获取居民列表
// @Description  获取系统中所有居民的列表
// @Tags         Resident
// @Accept       json
// @Produce      json
// @Param        page query int false "页码，默认为1"
// @Param        page_size query int false "每页条数，默认为10"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /residents [get]
func (c *ResidentController) GetResidents() {
	// 获取分页参数
	page, _ := strconv.Atoi(c.Ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.Ctx.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 使用 ResidentService 获取居民列表
	residentService := c.Container.GetService("resident").(service.InterfaceResidentService)
	residents, total, err := residentService.GetAllResidents(page, pageSize)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "获取居民列表失败", nil)
		return
	}

	errcode.Success(c.Ctx, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"data":        residents,
	})
}

// GetResident 获取单个居民
// @Summary      获取居民详情
// @Description  根据ID获取特定居民的详细信息
// @Tags         Resident
// @Accept       json
// @Produce      json
// @Param        id path int true "居民ID"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /residents/{id} [get]
func (c *ResidentController) GetResident() {
	id := c.Ctx.Param("id")
	if id == "" {
		errcode.ParamError(c.Ctx, "居民ID不能为空")
		return
	}

	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的居民ID")
		return
	}

	// 使用 ResidentService 获取居民详情
	residentService := c.Container.GetService("resident").(service.InterfaceResidentService)
	resident, err := residentService.GetResidentByID(uint(idUint))
	if err != nil {
		if err.Error() == "居民不存在" {
			errcode.NotFound(c.Ctx, err.Error())
			return
		}
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "获取居民信息失败", nil)
		return
	}

	errcode.Success(c.Ctx, resident)
}

// CreateResident 创建新居民
// @Summary      创建居民
// @Description  创建新的居民账户，需要关联到特定户号
// @Tags         Resident
// @Accept       json
// @Produce      json
// @Param        request body ResidentRequest true "居民信息 - 姓名和户号ID为必填，手机号可选但只能包含数字"
// @Security     BearerAuth
// @Success      201  {object}  map[string]interface{} "成功响应，包含创建的居民详情"
// @Failure      400  {object}  ErrorResponse "请求错误，户号不存在或电话号码已被使用"
// @Failure      500  {object}  ErrorResponse "服务器错误"
// @Router       /residents [post]
func (c *ResidentController) CreateResident() {
	var req ResidentRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数", nil)
		return
	}

	// 创建居民对象
	resident := &model.Resident{
		Name:        req.Name,
		Email:       req.Email,
		Phone:       strings.TrimSpace(req.Phone),
		HouseholdID: req.HouseholdID,
		// 密码将在 ResidentService 中处理
	}

	// 使用 ResidentService 创建居民
	residentService := c.Container.GetService("resident").(service.InterfaceResidentService)
	if err := residentService.CreateResident(resident); err != nil {
		if err.Error() == "手机号已被使用" || err.Error() == "户号不存在" {
			errcode.FailWithMessage(c.Ctx, errcode.ErrBind, err.Error(), nil)
			return
		}
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "创建居民失败: "+err.Error(), nil)
		return
	}

	c.Ctx.Status(http.StatusCreated)
	errcode.Success(c.Ctx, resident)
}

// UpdateResident 更新居民信息
// @Summary      更新居民
// @Description  更新现有居民的信息
// @Tags         Resident
// @Accept       json
// @Produce      json
// @Param        id path int true "居民ID"
// @Param        request body UpdateResidentRequest true "更新的居民信息"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /residents/{id} [put]
func (c *ResidentController) UpdateResident() {
	id := c.Ctx.Param("id")
	if id == "" {
		errcode.ParamError(c.Ctx, "无效的居民ID")
		return
	}

	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的居民ID")
		return
	}

	var req UpdateResidentRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数", nil)
		return
	}

	// 构建更新字段映射
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	updates["phone"] = strings.TrimSpace(req.Phone)
	if req.HouseholdID != nil {
		updates["household_id"] = *req.HouseholdID
	}

	// 使用 ResidentService 更新居民
	residentService := c.Container.GetService("resident").(service.InterfaceResidentService)
	resident, err := residentService.UpdateResident(uint(idUint), updates)
	if err != nil {
		if err.Error() == "居民不存在" {
			errcode.NotFound(c.Ctx, err.Error())
			return
		}
		if err.Error() == "手机号已被使用" || err.Error() == "户号不存在" {
			errcode.FailWithMessage(c.Ctx, errcode.ErrBind, err.Error(), nil)
			return
		}
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "更新居民失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, resident)
}

// DeleteResident 删除居民
// @Summary      删除居民
// @Description  删除指定ID的居民
// @Tags         Resident
// @Accept       json
// @Produce      json
// @Param        id path int true "居民ID"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /residents/{id} [delete]
func (c *ResidentController) DeleteResident() {
	id := c.Ctx.Param("id")
	if id == "" {
		errcode.ParamError(c.Ctx, "无效的居民ID")
		return
	}

	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的居民ID")
		return
	}

	// 使用 ResidentService 删除居民
	residentService := c.Container.GetService("resident").(service.InterfaceResidentService)
	if err := residentService.DeleteResident(uint(idUint)); err != nil {
		if err.Error() == "居民不存在" {
			errcode.NotFound(c.Ctx, err.Error())
			return
		}
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "删除居民失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, nil)
}

// HandleResidentFunc 返回一个处理居民请求的Gin处理函数
func HandleResidentFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewResidentController(ctx, container)

		switch method {
		case "getResidents":
			controller.GetResidents()
		case "getResident":
			controller.GetResident()
		case "createResident":
			controller.CreateResident()
		case "updateResident":
			controller.UpdateResident()
		case "deleteResident":
			controller.DeleteResident()
		default:
			errcode.FailWithMessage(ctx, errcode.ErrBind, "无效的方法", nil)
		}
	}
}
