package handler

import (
	"intercom_http_service/internal/errcode"
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// InterfaceBuildingController 定义楼号控制器接口
type InterfaceBuildingController interface {
	GetBuildings()
	GetBuilding()
	CreateBuilding()
	UpdateBuilding()
	DeleteBuilding()
	GetBuildingDevices()
	GetBuildingHouseholds()
	GetHouseholdTemplate()
	SaveHouseholdTemplate()
	BatchCreateHouseholds()
	RollbackBatchHouseholds()
}

// BuildingController 处理楼号相关的请求
type BuildingController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewBuildingController 创建一个新的楼号控制器
func NewBuildingController(ctx *gin.Context, container *service.ServiceContainer) *BuildingController {
	return &BuildingController{
		Ctx:       ctx,
		Container: container,
	}
}

// BuildingRequest 表示楼号请求
type BuildingRequest struct {
	BuildingName string `json:"building_name" binding:"required,max=50" example:"1号楼"`
	BuildingCode string `json:"building_code" binding:"required,max=20" example:"B001"`
	Address      string `json:"address" binding:"max=200" example:"小区东南角"`
	Status       string `json:"status" example:"active"` // active, inactive
}

// SaveHouseholdTemplateRequest 保存楼栋户号模板请求
type SaveHouseholdTemplateRequest struct {
	TemplateName string `json:"template_name"`
	TemplateJSON string `json:"template_json" binding:"required"`
}

// BatchCreateHouseholdsRequest 批量创建户号请求
type BatchCreateHouseholdsRequest struct {
	HouseholdItems   []BatchCreateHouseholdItem `json:"household_items"`
	HouseholdNumbers []string                   `json:"household_numbers"`
}

// BatchCreateHouseholdItem 批量创建户号结构化条目
type BatchCreateHouseholdItem struct {
	HouseholdNumber string `json:"household_number"`
	HouseCode       string `json:"house"`
	FloorCode       string `json:"floor"`
	UnitCode        string `json:"unit"`
	HouseholdExtID  string `json:"household_id"`
}

// RollbackBatchHouseholdsRequest 批量回滚户号请求
type RollbackBatchHouseholdsRequest struct {
	CreatedIDs []uint `json:"created_ids" binding:"required"`
}

// HandleBuildingFunc 返回一个处理楼号请求的Gin处理函数
func HandleBuildingFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewBuildingController(ctx, container)

		switch method {
		case "getBuildings":
			controller.GetBuildings()
		case "getBuilding":
			controller.GetBuilding()
		case "createBuilding":
			controller.CreateBuilding()
		case "updateBuilding":
			controller.UpdateBuilding()
		case "deleteBuilding":
			controller.DeleteBuilding()
		case "getBuildingDevices":
			controller.GetBuildingDevices()
		case "getBuildingHouseholds":
			controller.GetBuildingHouseholds()
		case "getHouseholdTemplate":
			controller.GetHouseholdTemplate()
		case "saveHouseholdTemplate":
			controller.SaveHouseholdTemplate()
		case "batchCreateHouseholds":
			controller.BatchCreateHouseholds()
		case "rollbackBatchHouseholds":
			controller.RollbackBatchHouseholds()
		default:
			errcode.FailWithMessage(ctx, errcode.ErrBind, "无效的方法", nil)
		}
	}
}

// 1. GetBuildings 获取所有楼号列表
// @Summary 获取所有楼号
// @Description 获取系统中所有楼号的列表
// @Tags Building
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认为1"
// @Param page_size query int false "每页条数，默认为10"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /buildings [get]
func (c *BuildingController) GetBuildings() {
	// 获取分页参数
	page, _ := strconv.Atoi(c.Ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.Ctx.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取楼号服务
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	buildings, total, err := buildingService.GetAllBuildings(page, pageSize)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "获取楼号列表失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, gin.H{
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		"data":        buildings,
	})
}

// 2. GetBuilding 获取单个楼号详情
// @Summary 获取楼号详情
// @Description 根据ID获取楼号详细信息
// @Tags Building
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "楼号ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /buildings/{id} [get]
func (c *BuildingController) GetBuilding() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	// 获取楼号服务
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	building, err := buildingService.GetBuildingByID(uint(buildingID))
	if err != nil {
		errcode.NotFound(c.Ctx, "楼号不存在: "+err.Error())
		return
	}

	errcode.Success(c.Ctx, building)
}

// 3. CreateBuilding 创建新楼号
// @Summary 创建楼号
// @Description 创建一个新的楼号
// @Tags Building
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param building body BuildingRequest true "楼号信息"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /buildings [post]
func (c *BuildingController) CreateBuilding() {
	var req BuildingRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	// 创建楼号对象
	building := &model.Building{
		BuildingName: req.BuildingName,
		BuildingCode: req.BuildingCode,
		Address:      req.Address,
	}

	// 如果提供了状态，则设置状态
	if req.Status != "" {
		building.Status = req.Status
	} else {
		building.Status = "active"
	}

	// 获取楼号服务
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	if err := buildingService.CreateBuilding(building); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "创建楼号失败: "+err.Error(), nil)
		return
	}

	c.Ctx.Status(http.StatusCreated)
	errcode.Success(c.Ctx, building)
}

// 4. UpdateBuilding 更新楼号信息
// @Summary 更新楼号
// @Description 更新楼号信息
// @Tags Building
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "楼号ID"
// @Param building body BuildingRequest true "楼号信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /buildings/{id} [put]
func (c *BuildingController) UpdateBuilding() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	var req BuildingRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	// 创建更新映射
	updates := make(map[string]interface{})
	if req.BuildingName != "" {
		updates["building_name"] = req.BuildingName
	}
	if req.BuildingCode != "" {
		updates["building_code"] = req.BuildingCode
	}
	if req.Address != "" {
		updates["address"] = req.Address
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	// 获取楼号服务
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	building, err := buildingService.UpdateBuilding(uint(buildingID), updates)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "更新楼号失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, building)
}

// 5. DeleteBuilding 删除楼号
// @Summary 删除楼号
// @Description 删除指定的楼号
// @Tags Building
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "楼号ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /buildings/{id} [delete]
func (c *BuildingController) DeleteBuilding() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	// 获取楼号服务
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	if err := buildingService.DeleteBuilding(uint(buildingID)); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "删除楼号失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, nil)
}

// 6. GetBuildingDevices 获取楼号关联的设备
// @Summary 获取楼号关联的设备
// @Description 获取指定楼号关联的所有设备
// @Tags Building
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "楼号ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /buildings/{id}/devices [get]
func (c *BuildingController) GetBuildingDevices() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	// 获取楼号服务
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	devices, err := buildingService.GetBuildingDevices(uint(buildingID))
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "获取楼号关联设备失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, devices)
}

// 7. GetBuildingHouseholds 获取楼号下的户号
// @Summary 获取楼号下的户号
// @Description 获取指定楼号下的所有户号
// @Tags Building
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "楼号ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /buildings/{id}/households [get]
func (c *BuildingController) GetBuildingHouseholds() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	// 获取楼号服务
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	households, err := buildingService.GetBuildingHouseholds(uint(buildingID))
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "获取楼号下户号失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, households)
}

// GetHouseholdTemplate 获取楼栋户号模板
func (c *BuildingController) GetHouseholdTemplate() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	tpl, err := buildingService.GetHouseholdTemplate(uint(buildingID))
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "获取模板失败: "+err.Error(), nil)
		return
	}

	if tpl == nil {
		errcode.Success(c.Ctx, gin.H{"exists": false})
		return
	}

	errcode.Success(c.Ctx, gin.H{
		"exists":        true,
		"id":            tpl.ID,
		"template_name": tpl.TemplateName,
		"template_json": tpl.TemplateJSON,
		"template_ver":  tpl.TemplateVer,
		"updated_at":    tpl.UpdatedAt,
	})
}

// SaveHouseholdTemplate 保存楼栋户号模板
func (c *BuildingController) SaveHouseholdTemplate() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	var req SaveHouseholdTemplateRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	templateJSON := strings.TrimSpace(req.TemplateJSON)
	if templateJSON == "" {
		errcode.ParamError(c.Ctx, "模板内容不能为空")
		return
	}

	operator := "admin"
	buildingService := c.Container.GetService("building").(service.InterfaceBuildingService)
	tpl, err := buildingService.SaveHouseholdTemplate(uint(buildingID), req.TemplateName, templateJSON, operator)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "保存模板失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, gin.H{
		"id":            tpl.ID,
		"template_name": tpl.TemplateName,
		"updated_at":    tpl.UpdatedAt,
	})
}

// BatchCreateHouseholds 批量创建楼栋户号（自动去重）
func (c *BuildingController) BatchCreateHouseholds() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	var req BatchCreateHouseholdsRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	normalized := make([]string, 0, len(req.HouseholdNumbers))
	seen := make(map[string]bool)
	for _, item := range req.HouseholdNumbers {
		n := strings.TrimSpace(item)
		if n == "" {
			continue
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		normalized = append(normalized, n)
	}

	if len(normalized) == 0 && len(req.HouseholdItems) == 0 {
		errcode.ParamError(c.Ctx, "至少提供一个户号")
		return
	}

	items := make([]service.BatchHouseholdInput, 0, len(normalized)+len(req.HouseholdItems))
	for _, item := range req.HouseholdItems {
		householdNumber := strings.TrimSpace(item.HouseholdNumber)
		if householdNumber == "" {
			householdNumber = strings.TrimSpace(item.HouseholdExtID)
		}
		if householdNumber == "" {
			continue
		}

		items = append(items, service.BatchHouseholdInput{
			HouseholdNumber: householdNumber,
			HouseCode:       strings.TrimSpace(item.HouseCode),
			FloorCode:       strings.TrimSpace(item.FloorCode),
			UnitCode:        strings.TrimSpace(item.UnitCode),
			HouseholdExtID:  strings.TrimSpace(item.HouseholdExtID),
		})
	}

	for _, number := range normalized {
		items = append(items, service.BatchHouseholdInput{HouseholdNumber: number})
	}

	if len(items) == 0 {
		errcode.ParamError(c.Ctx, "至少提供一个有效户号")
		return
	}

	if len(items) > 5000 {
		errcode.ParamError(c.Ctx, "单次最多创建5000个户号")
		return
	}

	householdService := c.Container.GetService("household").(service.InterfaceHouseholdService)
	created, skipped, failed, err := householdService.BatchCreateHouseholds(uint(buildingID), items)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "批量创建失败: "+err.Error(), nil)
		return
	}

	createdIDs := make([]uint, 0, len(created))
	createdNumbers := make([]string, 0, len(created))
	for _, item := range created {
		createdIDs = append(createdIDs, item.ID)
		createdNumbers = append(createdNumbers, item.HouseholdNumber)
	}

	createdStructured := make([]gin.H, 0, len(created))
	for _, item := range created {
		createdStructured = append(createdStructured, gin.H{
			"id":               item.ID,
			"household_number": item.HouseholdNumber,
			"house":            item.HouseCode,
			"floor":            item.FloorCode,
			"unit":             item.UnitCode,
			"household_id":     item.HouseholdExtID,
		})
	}

	errcode.Success(c.Ctx, gin.H{
		"request_id":      strconv.FormatInt(time.Now().UnixNano(), 10),
		"created_count":   len(createdIDs),
		"skipped_count":   len(skipped),
		"failed_count":    len(failed),
		"created_ids":     createdIDs,
		"created_numbers": createdNumbers,
		"created_items":   createdStructured,
		"skipped_numbers": skipped,
		"failed_numbers":  failed,
	})
}

// RollbackBatchHouseholds 回滚指定批次创建的户号
func (c *BuildingController) RollbackBatchHouseholds() {
	id := c.Ctx.Param("id")
	buildingID, err := strconv.Atoi(id)
	if err != nil {
		errcode.ParamError(c.Ctx, "无效的楼号ID")
		return
	}

	var req RollbackBatchHouseholdsRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数: "+err.Error(), nil)
		return
	}

	if len(req.CreatedIDs) == 0 {
		errcode.ParamError(c.Ctx, "created_ids 不能为空")
		return
	}

	householdService := c.Container.GetService("household").(service.InterfaceHouseholdService)
	deletedIDs, blocked, err := householdService.RollbackBatchHouseholds(uint(buildingID), req.CreatedIDs)
	if err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "回滚失败: "+err.Error(), nil)
		return
	}

	errcode.Success(c.Ctx, gin.H{
		"deleted_count": len(deletedIDs),
		"deleted_ids":   deletedIDs,
		"blocked":       blocked,
	})
}
