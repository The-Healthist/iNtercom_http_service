package handler

import (
	"intercom_http_service/internal/errcode"
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InterfaceDeviceController 定义设备控制器接口
type InterfaceDeviceController interface {
	GetDevices()
	GetDevice()
	CreateDevice()
	UpdateDevice()
	DeleteDevice()
	GetDeviceStatus()
	CheckDeviceHealth()
	AssociateDeviceWithBuilding()
	AssociateDeviceWithHousehold()
	GetDeviceHouseholds()
	RemoveDeviceHouseholdAssociation()
}

// DeviceController 处理设备相关的请求
type DeviceController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewDeviceController 创建一个新的设备控制器
func NewDeviceController(ctx *gin.Context, container *service.ServiceContainer) *DeviceController {
	return &DeviceController{
		Ctx:       ctx,
		Container: container,
	}
}

// DeviceRequest 表示旧版设备请求结构（为了兼容性保留）
type DeviceRequest struct {
	Name         string `json:"name" binding:"required" example:"门禁1号"`
	SerialNumber string `json:"serial_number" binding:"required" example:"SN2024050001"`
	Status       string `json:"status" example:"online"` // online, offline, fault
	Location     string `json:"location" example:"小区北门入口"`
	StaffIDs     []uint `json:"staff_ids" example:"1,2"` // 关联的物业员工ID列表
}

// DeviceRequestInput 表示新版设备请求结构
type DeviceRequestInput struct {
	Name         string `json:"name" binding:"required" example:"门禁1号"`
	SerialNumber string `json:"serial_number" binding:"required" example:"SN12345678"`
	Status       string `json:"status" example:"online"` // online, offline, fault
	Location     string `json:"location" example:"小区北门入口"`
	BuildingID   uint   `json:"building_id" example:"1"`     // 关联的楼号ID
	HouseholdIDs []uint `json:"household_ids" example:"1,2"` // 关联的户号ID列表
	StaffIDs     []uint `json:"staff_ids" example:"1,2,3"`   // 关联的物业员工ID列表(可选)
}

// DeviceBuildingRequest 设备关联楼号请求
type DeviceBuildingRequest struct {
	BuildingID uint `json:"building_id" binding:"required" example:"1"`
}

// DeviceHouseholdRequest 设备关联户号请求
type DeviceHouseholdRequest struct {
	HouseholdID uint `json:"household_id" binding:"required" example:"1"`
}

// DeviceHealthRequest 设备健康检测请求
type DeviceHealthRequest struct {
	DeviceID string `json:"device_id" binding:"required" example:"1"`
}

// DeviceStatusRequest 设备状态请求
type DeviceStatusRequest struct {
	DeviceID   uint                   `json:"device_id" binding:"required" example:"1"`
	Status     string                 `json:"status" example:"online"`
	Battery    int                    `json:"battery" example:"85"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// HandleDeviceFunc 返回一个处理设备请求的Gin处理函数
func HandleDeviceFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewDeviceController(ctx, container)

		switch method {
		case "getDevices":
			controller.GetDevices()
		case "getDevice":
			controller.GetDevice()
		case "createDevice":
			controller.CreateDevice()
		case "updateDevice":
			controller.UpdateDevice()
		case "deleteDevice":
			controller.DeleteDevice()
		case "getDeviceStatus":
			controller.GetDeviceStatus()
		case "checkDeviceHealth":
			controller.CheckDeviceHealth()
		case "associateDeviceWithBuilding":
			controller.AssociateDeviceWithBuilding()
		case "associateDeviceWithHousehold":
			controller.AssociateDeviceWithHousehold()
		case "getDeviceHouseholds":
			controller.GetDeviceHouseholds()
		case "removeDeviceHouseholdAssociation":
			controller.RemoveDeviceHouseholdAssociation()
		default:
			errcode.FailWithMessage(ctx, errcode.ErrBind, "无效的方法", nil)
		}
	}
}

// 1. GetDevices 获取所有设备列表
// @Summary 获取所有设备
// @Description 获取所有设备的列表，支持按楼号筛选
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param building_id query int false "楼号ID"
// @Success 200 {array} model.Device
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices [get]
func (c *DeviceController) GetDevices() {
	// 获取筛选参数
	buildingIDStr := c.Ctx.Query("building_id")

	var buildingID uint
	if buildingIDStr != "" {
		id, err := strconv.Atoi(buildingIDStr)
		if err == nil && id > 0 {
			buildingID = uint(id)
		}
	}

	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)

	var devices []model.Device
	var err error

	// 根据筛选条件获取设备
	if buildingID > 0 {
		devices, err = deviceService.GetDevicesByBuilding(buildingID)
	} else {
		devices, err = deviceService.GetAllDevices()
	}

	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "获取设备列表失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, devices)
}

// 2. GetDevice 获取单个设备详情
// @Summary 获取单个设备
// @Description 根据ID获取设备信息
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "设备ID"
// @Success 200 {object} model.Device
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id} [get]
func (c *DeviceController) GetDevice() {
	id := c.Ctx.Param("id")
	deviceID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的设备ID")
		return
	}

	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)

	device, err := deviceService.GetDeviceByID(uint(deviceID))
	if err != nil {
		c.Ctx.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data":    device,
	})
}

// 3. CreateDevice 创建新设备
// @Summary 创建新设备
// @Description 创建一个新的门禁设备，支持设备类型和关联
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param device body DeviceRequestInput true "设备信息：包含名称、类型、位置、关联楼号/户号等"
// @Success 201 {object} model.Device "成功创建的设备信息"
// @Failure 400 {object} ErrorResponse "请求参数错误，如缺少必填字段或格式不正确"
// @Failure 500 {object} ErrorResponse "服务器内部错误，可能是数据库操作失败等"
// @Router /devices [post]
func (c *DeviceController) CreateDevice() {
	var req DeviceRequestInput
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数: " + err.Error(),
			"data":    nil,
		})
		return
	}

	device := &model.Device{
		Name:         req.Name,
		SerialNumber: req.SerialNumber,
		Location:     req.Location,
	}

	// 设置关联的楼号
	if req.BuildingID > 0 {
		device.BuildingID = req.BuildingID
	}

	// 如果提供了状态，则设置状态
	if req.Status != "" {
		switch req.Status {
		case "online":
			device.Status = model.DeviceStatusOnline
		case "offline":
			device.Status = model.DeviceStatusOffline
		case "fault":
			device.Status = model.DeviceStatusFault
		default:
			device.Status = model.DeviceStatusOffline
		}
	} else {
		device.Status = model.DeviceStatusOffline
	}

	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)

	// 如果提供了楼号ID，验证楼号是否存在
	if req.BuildingID > 0 {
		buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
		_, err := buildingService.GetBuildingByID(req.BuildingID)
		if err != nil {
			c.Ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "关联的楼号不存在: " + err.Error(),
				"data":    nil,
			})
			return
		}
	}

	// 创建设备 - 这里不设置household_id
	if err := deviceService.CreateDevice(device); err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建设备失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 如果提供了户号ID列表，关联户号
	if len(req.HouseholdIDs) > 0 {
		householdService := c.Container.GetService("household").(service.InterfaceHouseholdService)

		// 关联第一个户号
		householdID := req.HouseholdIDs[0]
		if err := householdService.AssociateHouseholdWithDevice(householdID, device.ID); err != nil {
			c.Ctx.Error(err)
		} else {
			device.HouseholdID = householdID
		}
	}

	// 设置关联的物业员工(如果有提供)
	if len(req.StaffIDs) > 0 {
		for _, staffID := range req.StaffIDs {
			updates := map[string]interface{}{
				"property_id": staffID,
			}
			if _, err := deviceService.UpdateDevice(device.ID, updates); err != nil {
				// 这里只记录错误，不中断流程
				c.Ctx.Error(err)
			}
		}
	}

	c.Ctx.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "成功",
		"data":    device,
	})
}

// 4. UpdateDevice 更新设备信息
// @Summary 更新设备信息
// @Description 根据ID更新设备信息，支持更新关联
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "设备ID"
// @Param device body DeviceRequestInput true "设备信息：包含需要更新的字段"
// @Success 200 {object} model.Device "更新后的设备信息"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 404 {object} ErrorResponse "设备不存在"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /devices/{id} [put]
func (c *DeviceController) UpdateDevice() {
	id := c.Ctx.Param("id")
	deviceID, err := strconv.Atoi(id)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的设备ID",
			"data":    nil,
		})
		return
	}

	var req DeviceRequestInput
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 创建更新映射
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.SerialNumber != "" {
		updates["serial_number"] = req.SerialNumber
	}
	if req.Location != "" {
		updates["location"] = req.Location
	}
	if req.BuildingID > 0 {
		// 验证楼号是否存在
		buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
		_, err := buildingService.GetBuildingByID(req.BuildingID)
		if err != nil {
			c.Ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "关联的楼号不存在: " + err.Error(),
				"data":    nil,
			})
			return
		}
		updates["building_id"] = req.BuildingID
	}

	// 处理状态更新
	if req.Status != "" {
		switch req.Status {
		case "online":
			updates["status"] = model.DeviceStatusOnline
		case "offline":
			updates["status"] = model.DeviceStatusOffline
		case "fault":
			updates["status"] = model.DeviceStatusFault
		}
	}

	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)

	// 更新设备基本信息
	device, err := deviceService.UpdateDevice(uint(deviceID), updates)
	if err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新设备失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 如果提供了户号ID列表，更新关联
	if len(req.HouseholdIDs) > 0 {
		// 先清除现有关联
		householdService := c.Container.GetService("household").(service.InterfaceHouseholdService)

		// 获取当前关联的户号
		households, err := deviceService.GetDeviceHouseholds(uint(deviceID))
		if err != nil {
			c.Ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取设备关联户号失败: " + err.Error(),
				"data":    nil,
			})
			return
		}

		// 清除现有关联
		for _, household := range households {
			if err := householdService.RemoveHouseholdDeviceAssociation(household.ID, uint(deviceID)); err != nil {
				// 这里只记录错误，不中断流程
				c.Ctx.Error(err)
			}
		}

		// 添加新关联
		for _, householdID := range req.HouseholdIDs {
			if err := householdService.AssociateHouseholdWithDevice(householdID, uint(deviceID)); err != nil {
				// 这里只记录错误，不中断流程
				c.Ctx.Error(err)
			}
		}
	}

	// 更新关联的物业员工(如果有提供)
	if len(req.StaffIDs) > 0 {
		// 这里需要实现物业员工关联更新逻辑
		// TODO: 实现物业员工关联更新
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data":    device,
	})
}

// 5. DeleteDevice 删除设备
// @Summary 删除设备
// @Description 根据ID删除设备
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "设备ID"
// @Success 204 {object} nil
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id} [delete]
func (c *DeviceController) DeleteDevice() {
	id := c.Ctx.Param("id")
	deviceID, err := strconv.Atoi(id)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的设备ID",
			"data":    nil,
		})
		return
	}

	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)

	if err := deviceService.DeleteDevice(uint(deviceID)); err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除设备失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data":    nil,
	})
}

// 6. GetDeviceStatus 获取设备状态
// @Summary      获取设备状态
// @Description  获取设备的当前状态信息，包括在线状态、最后更新时间等
// @Tags         device
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "设备ID" example:"1"
// @Success      200  {object}  map[string]interface{} "设备状态信息，包含ID、名称、状态、位置、最后在线时间等"
// @Failure      404  {object}  ErrorResponse "设备不存在"
// @Failure      500  {object}  ErrorResponse "服务器内部错误，可能是数据库查询失败等"
// @Router       /devices/{id}/status [get]
func (c *DeviceController) GetDeviceStatus() {
	// 获取URL参数中的ID
	idStr := c.Ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的ID参数",
			"data":    nil,
		})
		return
	}

	// 查询数据库
	var device model.Device
	db := c.Container.GetService("db").(*gorm.DB)
	result := db.First(&device, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.Ctx.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "设备未找到",
				"data":    nil,
			})
		} else {
			c.Ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "查询设备失败: " + result.Error.Error(),
				"data":    nil,
			})
		}
		return
	}

	// 返回设备状态信息
	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data": gin.H{
			"id":            device.ID,
			"name":          device.Name,
			"serial_number": device.SerialNumber,
			"status":        device.Status,
			"location":      device.Location,
			"last_online":   device.UpdatedAt,
		},
	})
}

// CheckDeviceHealth 设备健康检测API
// @Summary 设备健康检测
// @Description 设备用于报告在线状态的简单健康检测接口
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DeviceHealthRequest true "设备健康检测请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /device/status [post]
func (c *DeviceController) CheckDeviceHealth() {
	var req DeviceHealthRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 转换设备ID
	deviceID, err := strconv.Atoi(req.DeviceID)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的设备ID",
			"data":    nil,
		})
		return
	}

	// 获取设备服务
	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)

	// 检查设备是否存在
	_, err = deviceService.GetDeviceByID(uint(deviceID))
	if err != nil {
		c.Ctx.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "设备不存在: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 更新设备状态为在线
	updates := map[string]interface{}{
		"status": model.DeviceStatusOnline,
	}

	_, err = deviceService.UpdateDevice(uint(deviceID), updates)
	if err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新设备状态失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "设备状态更新成功",
		"data": gin.H{
			"device_id": req.DeviceID,
			"status":    "online",
			"timestamp": time.Now(),
		},
	})
}

// 8. AssociateDeviceWithBuilding 将设备关联到楼号
// @Summary 关联设备与楼号
// @Description 将指定设备关联到楼号
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "设备ID"
// @Param request body DeviceBuildingRequest true "楼号信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id}/building [post]
func (c *DeviceController) AssociateDeviceWithBuilding() {
	id := c.Ctx.Param("id")
	deviceID, err := strconv.Atoi(id)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的设备ID",
			"data":    nil,
		})
		return
	}

	var req DeviceBuildingRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 验证楼号是否存在
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	_, err = buildingService.GetBuildingByID(req.BuildingID)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "楼号不存在: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 更新设备关联的楼号
	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)
	updates := map[string]interface{}{
		"building_id": req.BuildingID,
	}

	device, err := deviceService.UpdateDevice(uint(deviceID), updates)
	if err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "关联设备与楼号失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "设备与楼号关联成功",
		"data":    device,
	})
}

// 9. AssociateDeviceWithHousehold 将设备关联到户号
// @Summary 关联设备与户号
// @Description 将指定设备关联到户号
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "设备ID"
// @Param request body DeviceHouseholdRequest true "户号信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id}/households [post]
func (c *DeviceController) AssociateDeviceWithHousehold() {
	id := c.Ctx.Param("id")
	deviceID, err := strconv.Atoi(id)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的设备ID",
			"data":    nil,
		})
		return
	}

	var req DeviceHouseholdRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的请求参数: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 验证设备是否存在
	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)
	_, err = deviceService.GetDeviceByID(uint(deviceID))
	if err != nil {
		c.Ctx.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "设备不存在: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 关联设备到户号
	householdService := c.Container.GetService("household").(service.InterfaceHouseholdService)
	if err := householdService.AssociateHouseholdWithDevice(req.HouseholdID, uint(deviceID)); err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "关联设备与户号失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "设备与户号关联成功",
		"data":    nil,
	})
}

// 10. GetDeviceHouseholds 获取设备关联的户号
// @Summary 获取设备关联的户号
// @Description 获取指定设备关联的所有户号
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "设备ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id}/households [get]
func (c *DeviceController) GetDeviceHouseholds() {
	id := c.Ctx.Param("id")
	deviceID, err := strconv.Atoi(id)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的设备ID",
			"data":    nil,
		})
		return
	}

	// 验证设备是否存在
	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)
	_, err = deviceService.GetDeviceByID(uint(deviceID))
	if err != nil {
		c.Ctx.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "设备不存在: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 获取设备关联的户号
	households, err := deviceService.GetDeviceHouseholds(uint(deviceID))
	if err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取设备关联户号失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data":    households,
	})
}

// 11. RemoveDeviceHouseholdAssociation 解除设备与户号的关联
// @Summary 解除设备与户号的关联
// @Description 解除指定设备与其当前关联的户号的关联
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "设备ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id}/households [delete]
func (c *DeviceController) RemoveDeviceHouseholdAssociation() {
	// 获取设备ID
	id := c.Ctx.Param("id")
	deviceID, err := strconv.Atoi(id)
	if err != nil {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的设备ID",
			"data":    nil,
		})
		return
	}

	// 验证设备是否存在
	deviceService := c.Container.GetService("device").(service.InterfaceDeviceService)
	device, err := deviceService.GetDeviceByID(uint(deviceID))
	if err != nil {
		c.Ctx.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "设备不存在: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 检查设备是否关联了户号
	if device.HouseholdID == 0 {
		c.Ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "设备未关联任何户号",
			"data":    nil,
		})
		return
	}

	// 解除设备与户号的关联
	householdService := c.Container.GetService("household").(service.InterfaceHouseholdService)
	if err := householdService.RemoveHouseholdDeviceAssociation(device.HouseholdID, uint(deviceID)); err != nil {
		c.Ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解除设备与户号关联失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.Ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "设备与户号关联已解除",
		"data":    nil,
	})
}
