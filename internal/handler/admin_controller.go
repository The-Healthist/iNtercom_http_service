package handler

import (
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/service"
	"intercom_http_service/internal/errcode"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// InterfaceAdminController 定义管理员控制器接口
type InterfaceAdminController interface {
	GetAdmins()
	GetAdmin()
	CreateAdmin()
	UpdateAdmin()
	DeleteAdmin()
}

// AdminController 管理员控制器
type AdminController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewAdminController 创建一个新的管理员控制器
func NewAdminController(ctx *gin.Context, container *service.ServiceContainer) *AdminController {
	return &AdminController{
		Ctx:       ctx,
		Container: container,
	}
}

// CreateAdminRequest 创建管理员请求
type CreateAdminRequest struct {
	Username string `json:"username" binding:"required" example:"admin123"`
	Password string `json:"password" binding:"required" example:"Admin@123"`
	Phone    string `json:"phone" binding:"required" example:"13800138000"`
	Email    string `json:"email" binding:"required,email" example:"admin@example.com"`
}

// UpdateAdminRequest 更新管理员请求
type UpdateAdminRequest struct {
	Phone    string `json:"phone" example:"13800138000"`
	Email    string `json:"email" binding:"omitempty,email" example:"admin@example.com"`
	Password string `json:"password" example:"NewPassword@123"`
}

// HandleAdminFunc 返回一个处理管理员请求的Gin处理函数
func HandleAdminFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewAdminController(ctx, container)

		switch method {
		case "getAdmins":
			controller.GetAdmins()
		case "getAdmin":
			controller.GetAdmin()
		case "createAdmin":
			controller.CreateAdmin()
		case "updateAdmin":
			controller.UpdateAdmin()
		case "deleteAdmin":
			controller.DeleteAdmin()
		default:
			errcode.FailWithMessage(ctx, errcode.ErrBind, "无效的方法", nil)
		}
	}
}

// 1. GetAdmins 获取管理员列表
// @Summary      获取管理员列表
// @Description  分页获取所有管理员用户列表
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        page query int false "页码, 默认为1"
// @Param        page_size query int false "每页条数, 默认为10"
// @Param        search query string false "搜索关键词(用户名、电话等)"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  ErrorResponse
// @Router       /admins [get]
// @Security     BearerAuth
func (c *AdminController) GetAdmins() {
	// 获取分页参数
	page, _ := strconv.Atoi(c.Ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.Ctx.DefaultQuery("page_size", "10"))
	search := c.Ctx.Query("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 使用 AdminService 获取管理员列表
	adminService := c.Container.GetService("admin").(service.InterfaceAdminService)
	admins, total, err := adminService.GetAllAdmins(page, pageSize, search)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "查询管理员列表失败: "+err.Error(), nil)
		return
	}

	// 处理敏感信息，不返回密码
	var adminResponses []gin.H
	for _, admin := range admins {
		adminResponses = append(adminResponses, gin.H{
			"id":         admin.ID,
			"username":   admin.Username,
			"phone":      admin.Phone,
			"email":      admin.Email,
			"created_at": admin.CreatedAt,
			"updated_at": admin.UpdatedAt,
		})
	}

	errcode.Success(c.Ctx, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"data":        adminResponses,
	})
}

// 2. GetAdmin 获取管理员详情
// @Summary      获取管理员详情
// @Description  根据ID获取特定管理员的详细信息
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id path int true "管理员ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /admins/{id} [get]
// @Security     BearerAuth
func (c *AdminController) GetAdmin() {
	// 获取URL参数中的ID
	idStr := c.Ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的ID参数")
		return
	}

	// 使用 AdminService 获取管理员详情
	adminService := c.Container.GetService("admin").(service.InterfaceAdminService)
	admin, err := adminService.GetAdminByID(uint(id))
	if err != nil {
		errcode.NotFound(c.Ctx, "管理员不存在")
		return
	}

	// 处理敏感信息，不返回密码
	errcode.Success(c.Ctx, gin.H{
		"id":         admin.ID,
		"username":   admin.Username,
		"phone":      admin.Phone,
		"email":      admin.Email,
		"created_at": admin.CreatedAt,
		"updated_at": admin.UpdatedAt,
	})
}

// 3. CreateAdmin 创建管理员
// @Summary      创建管理员
// @Description  创建一个新的管理员用户
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        request body CreateAdminRequest true "管理员信息"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /admins [post]
// @Security     BearerAuth
func (c *AdminController) CreateAdmin() {
	var req CreateAdminRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	// 数据预处理
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)

	// 创建管理员对象
	admin := &model.Admin{
		Username: req.Username,
		Password: req.Password, // 密码加密将在 Service 层处理
		Email:    req.Email,
		Phone:    req.Phone,
	}

	// 使用 AdminService 创建管理员
	adminService := c.Container.GetService("admin").(service.InterfaceAdminService)
	if err := adminService.CreateAdmin(admin); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "创建管理员失败: "+err.Error(), nil)
		return
	}

	// 成功创建后返回管理员信息（不含密码）
	admin.Password = ""
	errcode.Success(c.Ctx, gin.H{
		"id":         admin.ID,
		"username":   admin.Username,
		"phone":      admin.Phone,
		"email":      admin.Email,
		"created_at": admin.CreatedAt,
	})
}

// 4. UpdateAdmin 更新管理员
// @Summary      更新管理员
// @Description  更新现有管理员用户的信息
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id path int true "管理员ID"
// @Param        request body UpdateAdminRequest true "更新的管理员信息"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /admins/{id} [put]
// @Security     BearerAuth
func (c *AdminController) UpdateAdmin() {
	// 获取URL参数中的ID
	idStr := c.Ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的ID参数")
		return
	}

	var req UpdateAdminRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	// 构建更新字段映射
	updates := make(map[string]interface{})
	if req.Phone != "" {
		updates["phone"] = strings.TrimSpace(req.Phone)
	}
	if req.Email != "" {
		updates["email"] = strings.TrimSpace(req.Email)
	}
	if req.Password != "" {
		updates["password"] = req.Password
	}

	// 使用 AdminService 更新管理员
	adminService := c.Container.GetService("admin").(service.InterfaceAdminService)
	admin, err := adminService.UpdateAdmin(uint(id), updates)
	if err != nil {
		if err.Error() == "管理员不存在" {
			errcode.NotFound(c.Ctx, err.Error())
			return
		}
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "更新管理员失败: "+err.Error(), nil)
		return
	}

	// 返回更新后的管理员信息（不含密码）
	errcode.Success(c.Ctx, gin.H{
		"id":         admin.ID,
		"username":   admin.Username,
		"phone":      admin.Phone,
		"email":      admin.Email,
		"updated_at": admin.UpdatedAt,
	})
}

// 5. DeleteAdmin 删除管理员
// @Summary      删除管理员
// @Description  删除指定ID的管理员用户
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id path int true "管理员ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /admins/{id} [delete]
// @Security     BearerAuth
func (c *AdminController) DeleteAdmin() {
	// 获取URL参数中的ID
	idStr := c.Ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的ID参数")
		return
	}

	// 使用 AdminService 删除管理员
	adminService := c.Container.GetService("admin").(service.InterfaceAdminService)
	if err := adminService.DeleteAdmin(uint(id)); err != nil {
		// 区分不同类型的错误，返回适当的状态码
		if err.Error() == "管理员不存在" {
			errcode.NotFound(c.Ctx, err.Error())
			return
		}
		if err.Error() == "系统必须至少有一个管理员，无法删除最后一个管理员" {
			errcode.FailWithMessage(c.Ctx, errcode.ErrBind, err.Error(), nil)
			return
		}
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "删除管理员失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, nil)
}
