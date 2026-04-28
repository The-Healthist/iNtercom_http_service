package service

import (
	"errors"
	"intercom_http_service/internal/config"
	"intercom_http_service/internal/model"

	"gorm.io/gorm"
)

// InterfaceBuildingService 定义楼号服务接口
type InterfaceBuildingService interface {
	GetAllBuildings(page, pageSize int) ([]model.Building, int64, error)
	GetBuildingByID(id uint) (*model.Building, error)
	CreateBuilding(building *model.Building) error
	UpdateBuilding(id uint, updates map[string]interface{}) (*model.Building, error)
	DeleteBuilding(id uint) error
	GetBuildingDevices(buildingID uint) ([]model.Device, error)
	GetBuildingHouseholds(buildingID uint) ([]model.Household, error)
	GetHouseholdTemplate(buildingID uint) (*model.BuildingHouseholdTemplate, error)
	SaveHouseholdTemplate(buildingID uint, templateName, templateJSON, operator string) (*model.BuildingHouseholdTemplate, error)
}

// BuildingService 提供楼号相关的服务
type BuildingService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewBuildingService 创建一个新的楼号服务
func NewBuildingService(db *gorm.DB, cfg *config.Config) InterfaceBuildingService {
	return &BuildingService{
		DB:     db,
		Config: cfg,
	}
}

// 1. GetAllBuildings 获取所有楼号列表，支持分页
func (s *BuildingService) GetAllBuildings(page, pageSize int) ([]model.Building, int64, error) {
	var buildings []model.Building
	var total int64

	// 获取总数
	if err := s.DB.Model(&model.Building{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := s.DB.Limit(pageSize).Offset(offset).Find(&buildings).Error; err != nil {
		return nil, 0, err
	}

	return buildings, total, nil
}

// 2. GetBuildingByID 根据ID获取楼号
func (s *BuildingService) GetBuildingByID(id uint) (*model.Building, error) {
	var building model.Building
	if err := s.DB.First(&building, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("楼号不存在")
		}
		return nil, err
	}
	return &building, nil
}

// 3. CreateBuilding 创建新楼号
func (s *BuildingService) CreateBuilding(building *model.Building) error {
	// 验证楼号编码唯一性
	exists, err := existsByQuery(s.DB, &model.Building{}, "building_code = ?", building.BuildingCode)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("楼号编码已存在")
	}

	// 设置默认状态
	if building.Status == "" {
		building.Status = "active"
	}

	return s.DB.Create(building).Error
}

// 4. UpdateBuilding 更新楼号信息
func (s *BuildingService) UpdateBuilding(id uint, updates map[string]interface{}) (*model.Building, error) {
	building, err := s.GetBuildingByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新楼号编码，需要检查唯一性
	if buildingCode, ok := updates["building_code"].(string); ok && buildingCode != building.BuildingCode {
		exists, err := existsByQuery(s.DB, &model.Building{}, "building_code = ? AND id != ?", buildingCode, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("楼号编码已存在")
		}
	}

	if err := s.DB.Model(building).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新获取更新后的楼号信息
	return s.GetBuildingByID(id)
}

// 5. DeleteBuilding 删除楼号
func (s *BuildingService) DeleteBuilding(id uint) error {
	building := &model.Building{}
	if err := s.DB.First(building, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("楼号不存在")
		}
		return err
	}

	// 检查是否有关联的户号
	hasHouseholds, err := existsByQuery(s.DB, &model.Household{}, "building_id = ?", id)
	if err != nil {
		return err
	}
	if hasHouseholds {
		return errors.New("该楼号下存在户号，无法删除")
	}

	// 检查是否有关联的设备
	hasDevices, err := existsByQuery(s.DB, &model.Device{}, "building_id = ?", id)
	if err != nil {
		return err
	}
	if hasDevices {
		return errors.New("该楼号下存在设备，无法删除")
	}

	return s.DB.Delete(building).Error
}

// 6. GetBuildingDevices 获取楼号关联的设备
func (s *BuildingService) GetBuildingDevices(buildingID uint) ([]model.Device, error) {
	// 检查楼号是否存在
	if _, err := s.GetBuildingByID(buildingID); err != nil {
		return nil, err
	}

	var devices []model.Device
	if err := s.DB.Where("building_id = ?", buildingID).Find(&devices).Error; err != nil {
		return nil, err
	}

	return devices, nil
}

// 7. GetBuildingHouseholds 获取楼号下的户号
func (s *BuildingService) GetBuildingHouseholds(buildingID uint) ([]model.Household, error) {
	// 检查楼号是否存在
	if _, err := s.GetBuildingByID(buildingID); err != nil {
		return nil, err
	}

	var households []model.Household
	if err := s.DB.Where("building_id = ?", buildingID).Find(&households).Error; err != nil {
		return nil, err
	}

	return households, nil
}

// GetHouseholdTemplate 获取楼栋户号模板
func (s *BuildingService) GetHouseholdTemplate(buildingID uint) (*model.BuildingHouseholdTemplate, error) {
	if _, err := s.GetBuildingByID(buildingID); err != nil {
		return nil, err
	}

	var tpl model.BuildingHouseholdTemplate
	if err := s.DB.Where("building_id = ?", buildingID).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &tpl, nil
}

// SaveHouseholdTemplate 保存楼栋户号模板
func (s *BuildingService) SaveHouseholdTemplate(buildingID uint, templateName, templateJSON, operator string) (*model.BuildingHouseholdTemplate, error) {
	if _, err := s.GetBuildingByID(buildingID); err != nil {
		return nil, err
	}

	var tpl model.BuildingHouseholdTemplate
	err := s.DB.Where("building_id = ?", buildingID).First(&tpl).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			createObj := model.BuildingHouseholdTemplate{
				BuildingID:   buildingID,
				TemplateName: templateName,
				TemplateJSON: templateJSON,
				TemplateVer:  "v1",
				LastOperator: operator,
			}
			if err := s.DB.Create(&createObj).Error; err != nil {
				return nil, err
			}
			return &createObj, nil
		}
		return nil, err
	}

	updates := map[string]interface{}{
		"template_name": templateName,
		"template_json": templateJSON,
		"template_ver":  "v1",
		"last_operator": operator,
	}

	if err := s.DB.Model(&tpl).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := s.DB.Where("id = ?", tpl.ID).First(&tpl).Error; err != nil {
		return nil, err
	}

	return &tpl, nil
}
