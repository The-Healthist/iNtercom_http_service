package services

import (
	"errors"
	"intercom_http_service/internal/domain/models"
	"intercom_http_service/internal/infrastructure/config"

	"gorm.io/gorm"
)

// InterfaceBuildingService 定义楼号服务接口
type InterfaceBuildingService interface {
	GetAllBuildings(page, pageSize int) ([]models.Building, int64, error)
	GetBuildingByID(id uint) (*models.Building, error)
	CreateBuilding(building *models.Building) error
	UpdateBuilding(id uint, updates map[string]interface{}) (*models.Building, error)
	DeleteBuilding(id uint) error
	GetBuildingDevices(buildingID uint) ([]models.Device, error)
	GetBuildingHouseholds(buildingID uint) ([]models.Household, error)
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
func (s *BuildingService) GetAllBuildings(page, pageSize int) ([]models.Building, int64, error) {
	var buildings []models.Building
	var total int64

	// 获取总数
	if err := s.DB.Model(&models.Building{}).Count(&total).Error; err != nil {
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
func (s *BuildingService) GetBuildingByID(id uint) (*models.Building, error) {
	var building models.Building
	if err := s.DB.First(&building, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("楼号不存在")
		}
		return nil, err
	}
	return &building, nil
}

// 3. CreateBuilding 创建新楼号
func (s *BuildingService) CreateBuilding(building *models.Building) error {
	// 验证楼号编码唯一性
	var count int64
	if err := s.DB.Model(&models.Building{}).Where("building_code = ?", building.BuildingCode).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("楼号编码已存在")
	}

	// 设置默认状态
	if building.Status == "" {
		building.Status = "active"
	}

	return s.DB.Create(building).Error
}

// 4. UpdateBuilding 更新楼号信息
func (s *BuildingService) UpdateBuilding(id uint, updates map[string]interface{}) (*models.Building, error) {
	building, err := s.GetBuildingByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新楼号编码，需要检查唯一性
	if buildingCode, ok := updates["building_code"].(string); ok && buildingCode != building.BuildingCode {
		var count int64
		if err := s.DB.Model(&models.Building{}).Where("building_code = ? AND id != ?", buildingCode, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
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
	building, err := s.GetBuildingByID(id)
	if err != nil {
		return err
	}

	// 检查是否有关联的户号
	var householdCount int64
	if err := s.DB.Model(&models.Household{}).Where("building_id = ?", id).Count(&householdCount).Error; err != nil {
		return err
	}
	if householdCount > 0 {
		return errors.New("该楼号下存在户号，无法删除")
	}

	// 检查是否有关联的设备
	var deviceCount int64
	if err := s.DB.Model(&models.Device{}).Where("building_id = ?", id).Count(&deviceCount).Error; err != nil {
		return err
	}
	if deviceCount > 0 {
		return errors.New("该楼号下存在设备，无法删除")
	}

	return s.DB.Delete(building).Error
}

// 6. GetBuildingDevices 获取楼号关联的设备
func (s *BuildingService) GetBuildingDevices(buildingID uint) ([]models.Device, error) {
	// 检查楼号是否存在
	if _, err := s.GetBuildingByID(buildingID); err != nil {
		return nil, err
	}

	var devices []models.Device
	if err := s.DB.Where("building_id = ?", buildingID).Find(&devices).Error; err != nil {
		return nil, err
	}

	return devices, nil
}

// 7. GetBuildingHouseholds 获取楼号下的户号
func (s *BuildingService) GetBuildingHouseholds(buildingID uint) ([]models.Household, error) {
	// 检查楼号是否存在
	if _, err := s.GetBuildingByID(buildingID); err != nil {
		return nil, err
	}

	var households []models.Household
	if err := s.DB.Where("building_id = ?", buildingID).Find(&households).Error; err != nil {
		return nil, err
	}

	return households, nil
}
