package controllers

import (
	"intercom_http_service/internal/domain/models"
	"intercom_http_service/internal/domain/services"
	"intercom_http_service/internal/domain/services/container"
	"intercom_http_service/internal/error/code"
	"intercom_http_service/internal/error/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// InterfaceStaffController 定义物业员工控制器接口
type InterfaceStaffController interface {
	GetStaffs()
	GetStaff()
	CreateStaff()
	UpdateStaff()
	DeleteStaff()
	GetStaffsWithDevices()
}

// StaffController 处理物业员工相关的请求
type StaffController struct {
	Ctx       *gin.Context
	Container *container.ServiceContainer
}

// NewStaffController 创建一个新的物业员工控制器
func NewStaffController(ctx *gin.Context, container *container.ServiceContainer) *StaffController {
	return &StaffController{
		Ctx:       ctx,
		Container: container,
	}
}

// GetStaffs 获取物业员工列表
// @Summary      获取物业员工列表
// @Description  获取所有物业员工的列表，支持分页和搜索
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        page query int false "页码，默认为1" example:"1"
// @Param        page_size query int false "每页条数，默认为10" example:"10"
// @Param        search query string false "搜索关键词(姓名、电话等)" example:"manager"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  ErrorResponse
// @Router       /staffs [get]
// @Security     BearerAuth
func (c *StaffController) GetStaffs() {
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

	// 使用 StaffService 获取物业员工列表
	staffService := c.Container.GetService("staff").(services.InterfaceStaffService)

	staffs, total, err := staffService.GetAllStaff(page, pageSize, search)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "查询物业员工列表失败: "+err.Error(), nil)
		return
	}

	// 构建响应
	var staffResponses []gin.H
	for _, staff := range staffs {
		staffResponses = append(staffResponses, gin.H{
			"id":            staff.ID,
			"phone":         staff.Phone,
			"property_name": staff.PropertyName,
			"position":      staff.Position,
			"role":          staff.Role,
			"status":        staff.Status,
			"created_at":    staff.CreatedAt,
			"updated_at":    staff.UpdatedAt,
		})
	}

	response.Success(c.Ctx, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"data":        staffResponses,
	})
}

// GetStaff 获取单个物业员工详情
// @Summary      获取物业员工详情
// @Description  根据ID获取特定物业员工的详细信息
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        id path int true "物业员工ID" example:"1"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /staffs/{id} [get]
// @Security     BearerAuth
func (c *StaffController) GetStaff() {
	// 获取URL参数中的ID
	idStr := c.Ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ParamError(c.Ctx, "无效的ID参数")
		return
	}

	// 使用 StaffService 获取物业员工详情
	staffService := c.Container.GetService("staff").(services.InterfaceStaffService)
	staff, err := staffService.GetStaffByIDWithDevices(uint(id))
	if err != nil {
		if err.Error() == "物业员工不存在" {
			response.NotFound(c.Ctx, "物业员工不存在")
			return
		}
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "查询物业员工失败: "+err.Error(), nil)
		return
	}

	// 提取设备ID
	var deviceIDs []uint
	for _, device := range staff.Devices {
		deviceIDs = append(deviceIDs, device.ID)
	}

	// 返回物业员工信息
	response.Success(c.Ctx, gin.H{
		"id":            staff.ID,
		"phone":         staff.Phone,
		"property_name": staff.PropertyName,
		"position":      staff.Position,
		"role":          staff.Role,
		"status":        staff.Status,
		"username":      staff.Username,
		"remark":        staff.Remark,
		"device_ids":    deviceIDs,
		"created_at":    staff.CreatedAt,
		"updated_at":    staff.UpdatedAt,
	})
}

// CreateStaffRequest 表示创建物业员工的请求体
type CreateStaffRequest struct {
	Name         string `json:"name" example:"王物业"` // 注意: 已从模型中移除，但保留请求结构以兼容前端
	Phone        string `json:"phone" binding:"required" example:"13700001234"`
	PropertyName string `json:"property_name" example:"阳光花园小区"`
	Position     string `json:"position" example:"物业经理"`
	Role         string `json:"role" binding:"required" example:"manager"` // 可选值: manager, staff, security
	Status       string `json:"status" example:"active"`                   // 可选值: active, inactive, suspended
	Remark       string `json:"remark" example:"负责A区日常管理工作"`
	Username     string `json:"username" binding:"required" example:"wangwuye"`
	Password     string `json:"password" binding:"required" example:"Property@123"`
	DeviceIDs    []uint `json:"device_ids" example:"1,2,3"` // 关联的设备ID列表
}

// CreateStaff 创建新物业员工
// @Summary      创建物业员工
// @Description  创建一个新的物业员工
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        request body CreateStaffRequest true "物业员工信息"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /staffs [post]
// @Security     BearerAuth
func (c *StaffController) CreateStaff() {
	var req CreateStaffRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 创建物业员工对象
	staff := &models.PropertyStaff{
		Phone:        req.Phone,
		PropertyName: req.PropertyName,
		Position:     req.Position,
		Role:         req.Role,
		Status:       req.Status,
		Remark:       req.Remark,
		Username:     req.Username,
		Password:     req.Password,
	}

	// 使用 StaffService 创建物业员工
	staffService := c.Container.GetService("staff").(services.InterfaceStaffService)
	if err := staffService.CreateStaff(staff); err != nil {
		if err.Error() == "手机号已被使用" || err.Error() == "用户名已存在" {
			c.Ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建物业员工失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 关联设备
	if len(req.DeviceIDs) > 0 {
		// 获取设备服务
		deviceService := c.Container.GetService("device").(services.InterfaceDeviceService)

		// 为每个设备设置物业ID
		for _, deviceID := range req.DeviceIDs {
			updates := map[string]interface{}{
				"property_id": staff.ID,
			}
			if _, err := deviceService.UpdateDevice(deviceID, updates); err != nil {
				c.Ctx.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "关联设备失败: " + err.Error(),
					"data":    nil,
				})
				return
			}
		}
	}

	c.Ctx.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "成功创建物业员工",
		"data": gin.H{
			"id":            staff.ID,
			"phone":         staff.Phone,
			"property_name": staff.PropertyName,
			"position":      staff.Position,
			"role":          staff.Role,
			"status":        staff.Status,
			"username":      staff.Username,
			"remark":        staff.Remark,
			"device_ids":    req.DeviceIDs,
			"created_at":    staff.CreatedAt,
		},
	})
}

// UpdateStaffRequest 表示更新物业员工的请求体
type UpdateStaffRequest struct {
	Name         string `json:"name" example:"李物业"`
	Phone        string `json:"phone" example:"13700005678"`
	PropertyName string `json:"property_name" example:"幸福家园小区"`
	Position     string `json:"position" example:"前台客服"`
	Role         string `json:"role" example:"staff"`    // 可选值: manager, staff, security
	Status       string `json:"status" example:"active"` // 可选值: active, inactive, suspended
	Remark       string `json:"remark" example:"负责接待访客和处理居民投诉"`
	Username     string `json:"username" example:"liwuye"`
	Password     string `json:"password" example:"NewProperty@456"`
	DeviceIDs    []uint `json:"device_ids" example:"1,3,5"` // 更新关联的设备ID列表
}

// UpdateStaff 更新物业员工信息
// @Summary      更新物业员工
// @Description  更新现有物业员工的信息
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        id path int true "物业员工ID" example:"1"
// @Param        request body UpdateStaffRequest true "更新的物业员工信息"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /staffs/{id} [put]
// @Security     BearerAuth
func (c *StaffController) UpdateStaff() {
	// 获取URL参数中的ID
	idStr := c.Ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的ID参数",
			"data":    nil,
		})
		return
	}

	var req UpdateStaffRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 构建更新字段映射
	updates := make(map[string]interface{})
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.PropertyName != "" {
		updates["property_name"] = req.PropertyName
	}
	if req.Position != "" {
		updates["position"] = req.Position
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Password != "" {
		updates["password"] = req.Password
	}

	// 使用 StaffService 更新物业员工
	staffService := c.Container.GetService("staff").(services.InterfaceStaffService)
	staff, err := staffService.UpdateStaff(uint(id), updates)
	if err != nil {
		if err.Error() == "物业员工不存在" {
			c.Ctx.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "物业员工不存在",
				"data":    nil,
			})
			return
		}
		if err.Error() == "手机号已被其他物业员工使用" || err.Error() == "用户名已被其他物业员工使用" {
			c.Ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新物业员工失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 如果请求中包含设备ID列表，更新关联设备
	if req.DeviceIDs != nil {
		// 获取设备服务
		deviceService := c.Container.GetService("device").(services.InterfaceDeviceService)

		// 清除现有关联 - 将属于该员工的所有设备的property_id设为NULL
		devices, err := staffService.GetStaffDevices(uint(id))
		if err != nil {
			c.Ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取设备关联失败: " + err.Error(),
				"data":    nil,
			})
			return
		}

		for _, device := range devices {
			updates := map[string]interface{}{
				"property_id": nil,
			}
			if _, err := deviceService.UpdateDevice(device.ID, updates); err != nil {
				c.Ctx.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "清除设备关联失败: " + err.Error(),
					"data":    nil,
				})
				return
			}
		}

		// 添加新关联
		for _, deviceID := range req.DeviceIDs {
			updates := map[string]interface{}{
				"property_id": uint(id),
			}
			if _, err := deviceService.UpdateDevice(deviceID, updates); err != nil {
				c.Ctx.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "关联设备失败: " + err.Error(),
					"data":    nil,
				})
				return
			}
		}
	}

	// 获取更新后的设备ID列表
	devices, err := staffService.GetStaffDevices(uint(id))
	if err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询关联设备失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	var deviceIDs []uint
	for _, device := range devices {
		deviceIDs = append(deviceIDs, device.ID)
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功更新物业员工",
		"data": gin.H{
			"id":            staff.ID,
			"phone":         staff.Phone,
			"property_name": staff.PropertyName,
			"position":      staff.Position,
			"role":          staff.Role,
			"status":        staff.Status,
			"username":      staff.Username,
			"remark":        staff.Remark,
			"device_ids":    deviceIDs,
			"updated_at":    staff.UpdatedAt,
		},
	})
}

// DeleteStaff 删除物业员工
// @Summary      删除物业员工
// @Description  删除指定ID的物业员工
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        id path int true "物业员工ID" example:"2"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /staffs/{id} [delete]
// @Security     BearerAuth
func (c *StaffController) DeleteStaff() {
	// 获取URL参数中的ID
	idStr := c.Ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的ID参数",
			"data":    nil,
		})
		return
	}

	// 使用 StaffService 删除物业员工
	staffService := c.Container.GetService("staff").(services.InterfaceStaffService)
	if err := staffService.DeleteStaff(uint(id)); err != nil {
		if err.Error() == "物业员工不存在" {
			c.Ctx.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "物业员工不存在",
				"data":    nil,
			})
			return
		}
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除物业员工失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功删除物业员工",
		"data":    nil,
	})
}

// GetStaffsWithDevices 获取包含关联设备的物业员工列表
// @Summary      获取带设备信息的物业员工列表
// @Description  获取所有物业员工的列表及其关联的设备信息
// @Tags         Staff
// @Accept       json
// @Produce      json
// @Param        page query int false "页码，默认为1" example:"1"
// @Param        page_size query int false "每页条数，默认为10" example:"10"
// @Param        search query string false "搜索关键词(姓名、电话等)" example:"manager"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  ErrorResponse
// @Router       /staffs/with-devices [get]
// @Security     BearerAuth
func (c *StaffController) GetStaffsWithDevices() {
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

	// 使用 StaffService 获取物业员工列表
	staffService := c.Container.GetService("staff").(services.InterfaceStaffService)

	// 这里假设 StaffService 有一个获取带设备的员工列表的方法
	// 如果没有，可以先获取员工列表，然后针对每个员工查询其设备
	staffs, total, err := staffService.GetAllStaff(page, pageSize, search)
	if err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询物业员工列表失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 构建响应，包含设备信息
	var staffResponses []gin.H
	for _, staff := range staffs {
		// 获取每个员工的设备
		devices, err := staffService.GetStaffDevices(staff.ID)
		if err != nil {
			c.Ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "查询员工设备失败: " + err.Error(),
				"data":    nil,
			})
			return
		}

		// 提取设备ID和基本信息
		var deviceList []gin.H
		for _, device := range devices {
			deviceList = append(deviceList, gin.H{
				"id":            device.ID,
				"name":          device.Name,
				"serial_number": device.SerialNumber,
				"location":      device.Location,
				"status":        device.Status,
			})
		}

		staffResponses = append(staffResponses, gin.H{
			"id":            staff.ID,
			"phone":         staff.Phone,
			"property_name": staff.PropertyName,
			"position":      staff.Position,
			"role":          staff.Role,
			"status":        staff.Status,
			"username":      staff.Username,
			"created_at":    staff.CreatedAt,
			"updated_at":    staff.UpdatedAt,
			"devices":       deviceList,
		})
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data": gin.H{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
			"data":        staffResponses,
		},
	})
}

// HandleStaffFunc 返回一个处理物业员工请求的Gin处理函数
func HandleStaffFunc(container *container.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewStaffController(ctx, container)

		switch method {
		case "getStaffs":
			controller.GetStaffs()
		case "getStaff":
			controller.GetStaff()
		case "createStaff":
			controller.CreateStaff()
		case "updateStaff":
			controller.UpdateStaff()
		case "deleteStaff":
			controller.DeleteStaff()
		case "getStaffsWithDevices":
			controller.GetStaffsWithDevices()
		default:
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "无效的方法",
				"data":    nil,
			})
		}
	}
}
