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

// InterfaceHouseholdController 定义户号控制器接口
type InterfaceHouseholdController interface {
	GetHouseholds()
	GetHousehold()
	CreateHousehold()
	UpdateHousehold()
	DeleteHousehold()
	GetHouseholdDevices()
	GetHouseholdResidents()
	AssociateHouseholdWithDevice()
	RemoveHouseholdDeviceAssociation()
}

// HouseholdController 处理户号相关的请求
type HouseholdController struct {
	Ctx       *gin.Context
	Container *container.ServiceContainer
}

// NewHouseholdController 创建一个新的户号控制器
func NewHouseholdController(ctx *gin.Context, container *container.ServiceContainer) *HouseholdController {
	return &HouseholdController{
		Ctx:       ctx,
		Container: container,
	}
}

// HouseholdRequest 表示户号请求
type HouseholdRequest struct {
	HouseholdNumber string `json:"household_number" binding:"required" example:"1-1-101"`
	BuildingID      uint   `json:"building_id" binding:"required" example:"1"`
	Status          string `json:"status" example:"active"` // active, inactive
}

// HouseholdDeviceRequest 表示户号与设备关联请求
type HouseholdDeviceRequest struct {
	DeviceID uint `json:"device_id" binding:"required" example:"1"`
}

// HandleHouseholdFunc 返回一个处理户号请求的Gin处理函数
func HandleHouseholdFunc(container *container.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewHouseholdController(ctx, container)

		switch method {
		case "getHouseholds":
			controller.GetHouseholds()
		case "getHousehold":
			controller.GetHousehold()
		case "createHousehold":
			controller.CreateHousehold()
		case "updateHousehold":
			controller.UpdateHousehold()
		case "deleteHousehold":
			controller.DeleteHousehold()
		case "getHouseholdDevices":
			controller.GetHouseholdDevices()
		case "getHouseholdResidents":
			controller.GetHouseholdResidents()
		case "associateHouseholdWithDevice":
			controller.AssociateHouseholdWithDevice()
		case "removeHouseholdDeviceAssociation":
			controller.RemoveHouseholdDeviceAssociation()
		default:
			response.FailWithMessage(ctx, code.ErrBind, "无效的方法", nil)
		}
	}
}

// 1. GetHouseholds 获取所有户号列表
// @Summary 获取所有户号
// @Description 获取系统中所有户号的列表
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认为1"
// @Param page_size query int false "每页条数，默认为10"
// @Param building_id query int false "楼号ID，用于筛选特定楼号下的户号"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /households [get]
func (c *HouseholdController) GetHouseholds() {
	// 获取分页参数
	page, _ := strconv.Atoi(c.Ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.Ctx.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取筛选参数
	buildingIDStr := c.Ctx.Query("building_id")
	var buildingID uint
	if buildingIDStr != "" {
		id, err := strconv.Atoi(buildingIDStr)
		if err == nil && id > 0 {
			buildingID = uint(id)
		}
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)

	var households []models.Household
	var total int64
	var err error

	// 根据是否提供楼号ID决定获取方式
	if buildingID > 0 {
		households, total, err = householdService.GetHouseholdsByBuildingID(buildingID, page, pageSize)
	} else {
		households, total, err = householdService.GetAllHouseholds(page, pageSize)
	}

	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "获取户号列表失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"data":        households,
	})
}

// 2. GetHousehold 获取单个户号详情
// @Summary 获取户号详情
// @Description 根据ID获取户号详细信息
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "户号ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /households/{id} [get]
func (c *HouseholdController) GetHousehold() {
	id := c.Ctx.Param("id")
	householdID, err := strconv.Atoi(id)
	if err != nil {
		response.ParamError(c.Ctx, "无效的户号ID")
		return
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)
	household, err := householdService.GetHouseholdByID(uint(householdID))
	if err != nil {
		response.NotFound(c.Ctx, "户号不存在: "+err.Error())
		return
	}

	response.Success(c.Ctx, household)
}

// 3. CreateHousehold 创建新户号
// @Summary 创建户号
// @Description 创建一个新的户号，需要关联到楼号
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param household body HouseholdRequest true "户号信息"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /households [post]
func (c *HouseholdController) CreateHousehold() {
	var req HouseholdRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	// 创建户号对象
	household := &models.Household{
		HouseholdNumber: req.HouseholdNumber,
		BuildingID:      req.BuildingID,
	}

	// 如果提供了状态，则设置状态
	if req.Status != "" {
		household.Status = req.Status
	} else {
		household.Status = "active"
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)

	// 验证楼号是否存在
	buildingService := c.Container.GetService("building").(services.InterfaceBuildingService)
	_, err := buildingService.GetBuildingByID(req.BuildingID)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrBind, "关联的楼号不存在: "+err.Error(), nil)
		return
	}

	if err := householdService.CreateHousehold(household); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "创建户号失败: "+err.Error(), nil)
		return
	}

	c.Ctx.Status(http.StatusCreated)
	response.Success(c.Ctx, household)
}

// 4. UpdateHousehold 更新户号信息
// @Summary 更新户号
// @Description 更新户号信息
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "户号ID"
// @Param household body HouseholdRequest true "户号信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /households/{id} [put]
func (c *HouseholdController) UpdateHousehold() {
	id := c.Ctx.Param("id")
	householdID, err := strconv.Atoi(id)
	if err != nil {
		response.ParamError(c.Ctx, "无效的户号ID")
		return
	}

	var req HouseholdRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	// 创建更新映射
	updates := make(map[string]interface{})
	if req.HouseholdNumber != "" {
		updates["household_number"] = req.HouseholdNumber
	}
	if req.BuildingID > 0 {
		// 验证楼号是否存在
		buildingService := c.Container.GetService("building").(services.InterfaceBuildingService)
		_, err := buildingService.GetBuildingByID(req.BuildingID)
		if err != nil {
			response.FailWithMessage(c.Ctx, code.ErrBind, "关联的楼号不存在: "+err.Error(), nil)
			return
		}
		updates["building_id"] = req.BuildingID
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)
	household, err := householdService.UpdateHousehold(uint(householdID), updates)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "更新户号失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, household)
}

// 5. DeleteHousehold 删除户号
// @Summary 删除户号
// @Description 删除指定的户号
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "户号ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /households/{id} [delete]
func (c *HouseholdController) DeleteHousehold() {
	id := c.Ctx.Param("id")
	householdID, err := strconv.Atoi(id)
	if err != nil {
		response.ParamError(c.Ctx, "无效的户号ID")
		return
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)
	if err := householdService.DeleteHousehold(uint(householdID)); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "删除户号失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, nil)
}

// 6. GetHouseholdDevices 获取户号关联的设备
// @Summary 获取户号关联的设备
// @Description 获取指定户号关联的所有设备
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "户号ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /households/{id}/devices [get]
func (c *HouseholdController) GetHouseholdDevices() {
	id := c.Ctx.Param("id")
	householdID, err := strconv.Atoi(id)
	if err != nil {
		response.ParamError(c.Ctx, "无效的户号ID")
		return
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)
	devices, err := householdService.GetHouseholdDevices(uint(householdID))
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "获取户号关联设备失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, devices)
}

// 7. GetHouseholdResidents 获取户号下的居民
// @Summary 获取户号下的居民
// @Description 获取指定户号下的所有居民
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "户号ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /households/{id}/residents [get]
func (c *HouseholdController) GetHouseholdResidents() {
	id := c.Ctx.Param("id")
	householdID, err := strconv.Atoi(id)
	if err != nil {
		response.ParamError(c.Ctx, "无效的户号ID")
		return
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)
	residents, err := householdService.GetHouseholdResidents(uint(householdID))
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "获取户号下居民失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, residents)
}

// 8. AssociateHouseholdWithDevice 将户号关联到设备
// @Summary 关联户号与设备
// @Description 将指定户号关联到设备
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "户号ID"
// @Param request body HouseholdDeviceRequest true "设备信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /households/{id}/devices [post]
func (c *HouseholdController) AssociateHouseholdWithDevice() {
	id := c.Ctx.Param("id")
	householdID, err := strconv.Atoi(id)
	if err != nil {
		response.ParamError(c.Ctx, "无效的户号ID")
		return
	}

	var req HouseholdDeviceRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)

	// 验证设备是否存在
	deviceService := c.Container.GetService("device").(services.InterfaceDeviceService)
	_, err = deviceService.GetDeviceByID(req.DeviceID)
	if err != nil {
		response.FailWithMessage(c.Ctx, code.ErrBind, "设备不存在: "+err.Error(), nil)
		return
	}

	if err := householdService.AssociateHouseholdWithDevice(uint(householdID), req.DeviceID); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "关联户号与设备失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, nil)
}

// 9. RemoveHouseholdDeviceAssociation 解除户号与设备的关联
// @Summary 解除户号与设备的关联
// @Description 解除指定户号与设备的关联
// @Tags Household
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "户号ID"
// @Param device_id path int true "设备ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /households/{id}/devices/{device_id} [delete]
func (c *HouseholdController) RemoveHouseholdDeviceAssociation() {
	// 获取户号ID
	id := c.Ctx.Param("id")
	householdID, err := strconv.Atoi(id)
	if err != nil {
		response.ParamError(c.Ctx, "无效的户号ID")
		return
	}

	// 获取设备ID
	deviceIDStr := c.Ctx.Param("device_id")
	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		response.ParamError(c.Ctx, "无效的设备ID")
		return
	}

	// 获取户号服务
	householdService := c.Container.GetService("household").(services.InterfaceHouseholdService)
	if err := householdService.RemoveHouseholdDeviceAssociation(uint(householdID), uint(deviceID)); err != nil {
		response.FailWithMessage(c.Ctx, code.ErrDatabase, "解除户号与设备关联失败: "+err.Error(), nil)
		return
	}

	response.Success(c.Ctx, nil)
}
