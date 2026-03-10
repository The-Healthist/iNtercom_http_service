package service

import (
	"errors"
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/config"

	"gorm.io/gorm"
)

// InterfaceHouseholdService 定义户号服务接口
type InterfaceHouseholdService interface {
	GetAllHouseholds(page, pageSize int) ([]model.Household, int64, error)
	GetHouseholdsByBuildingID(buildingID uint, page, pageSize int) ([]model.Household, int64, error)
	GetHouseholdByID(id uint) (*model.Household, error)
	CreateHousehold(household *model.Household) error
	UpdateHousehold(id uint, updates map[string]interface{}) (*model.Household, error)
	DeleteHousehold(id uint) error
	GetHouseholdDevices(householdID uint) ([]model.Device, error)
	GetHouseholdResidents(householdID uint) ([]model.Resident, error)
	AssociateHouseholdWithDevice(householdID, deviceID uint) error
	RemoveHouseholdDeviceAssociation(householdID, deviceID uint) error
}

// HouseholdService 提供户号相关的服务
type HouseholdService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewHouseholdService 创建一个新的户号服务
func NewHouseholdService(db *gorm.DB, cfg *config.Config) InterfaceHouseholdService {
	return &HouseholdService{
		DB:     db,
		Config: cfg,
	}
}

// 1. GetAllHouseholds 获取所有户号列表，支持分页
func (s *HouseholdService) GetAllHouseholds(page, pageSize int) ([]model.Household, int64, error) {
	var households []model.Household
	var total int64

	// 获取总数
	if err := s.DB.Model(&model.Household{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := s.DB.Preload("Building").Limit(pageSize).Offset(offset).Find(&households).Error; err != nil {
		return nil, 0, err
	}

	return households, total, nil
}

// 2. GetHouseholdsByBuildingID 获取指定楼号下的户号列表
func (s *HouseholdService) GetHouseholdsByBuildingID(buildingID uint, page, pageSize int) ([]model.Household, int64, error) {
	var households []model.Household
	var total int64

	// 获取总数
	if err := s.DB.Model(&model.Household{}).Where("building_id = ?", buildingID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := s.DB.Where("building_id = ?", buildingID).Preload("Building").Limit(pageSize).Offset(offset).Find(&households).Error; err != nil {
		return nil, 0, err
	}

	return households, total, nil
}

// 3. GetHouseholdByID 根据ID获取户号
func (s *HouseholdService) GetHouseholdByID(id uint) (*model.Household, error) {
	var household model.Household
	if err := s.DB.Preload("Building").Preload("Devices").Preload("Residents").First(&household, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("户号不存在")
		}
		return nil, err
	}
	return &household, nil
}

// 4. CreateHousehold 创建新户号
func (s *HouseholdService) CreateHousehold(household *model.Household) error {
	// 验证户号唯一性（同一楼号下户号编号不能重复）
	var count int64
	if err := s.DB.Model(&model.Household{}).Where("building_id = ? AND household_number = ?", household.BuildingID, household.HouseholdNumber).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该楼号下已存在相同户号")
	}

	// 设置默认状态
	if household.Status == "" {
		household.Status = "active"
	}

	return s.DB.Create(household).Error
}

// 5. UpdateHousehold 更新户号信息
func (s *HouseholdService) UpdateHousehold(id uint, updates map[string]interface{}) (*model.Household, error) {
	household, err := s.GetHouseholdByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新户号编号和楼号ID，需要检查唯一性
	buildingID, hasBuildingID := updates["building_id"].(uint)
	householdNumber, hasHouseholdNumber := updates["household_number"].(string)

	if (hasBuildingID || hasHouseholdNumber) && (hasBuildingID && buildingID != household.BuildingID || hasHouseholdNumber && householdNumber != household.HouseholdNumber) {
		// 确定要检查的楼号ID
		checkBuildingID := household.BuildingID
		if hasBuildingID {
			checkBuildingID = buildingID
		}

		// 确定要检查的户号编号
		checkHouseholdNumber := household.HouseholdNumber
		if hasHouseholdNumber {
			checkHouseholdNumber = householdNumber
		}

		// 检查唯一性
		var count int64
		if err := s.DB.Model(&model.Household{}).Where("building_id = ? AND household_number = ? AND id != ?", checkBuildingID, checkHouseholdNumber, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("该楼号下已存在相同户号")
		}
	}

	if err := s.DB.Model(household).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新获取更新后的户号信息
	return s.GetHouseholdByID(id)
}

// 6. DeleteHousehold 删除户号
func (s *HouseholdService) DeleteHousehold(id uint) error {
	household, err := s.GetHouseholdByID(id)
	if err != nil {
		return err
	}

	// 检查是否有关联的居民
	var residentCount int64
	if err := s.DB.Model(&model.Resident{}).Where("household_id = ?", id).Count(&residentCount).Error; err != nil {
		return err
	}
	if residentCount > 0 {
		return errors.New("该户号下存在居民，无法删除")
	}

	// 检查是否有关联的设备
	var deviceCount int64
	if err := s.DB.Model(&model.Device{}).Where("household_id = ?", id).Count(&deviceCount).Error; err != nil {
		return err
	}
	if deviceCount > 0 {
		return errors.New("该户号下存在关联设备，请先解除关联")
	}

	return s.DB.Delete(household).Error
}

// 7. GetHouseholdDevices 获取户号关联的设备
func (s *HouseholdService) GetHouseholdDevices(householdID uint) ([]model.Device, error) {
	// 检查户号是否存在
	if _, err := s.GetHouseholdByID(householdID); err != nil {
		return nil, err
	}

	// 查询household_id为指定值的设备
	var devices []model.Device
	if err := s.DB.Where("household_id = ?", householdID).Find(&devices).Error; err != nil {
		return nil, err
	}

	return devices, nil
}

// 8. GetHouseholdResidents 获取户号下的居民
func (s *HouseholdService) GetHouseholdResidents(householdID uint) ([]model.Resident, error) {
	// 检查户号是否存在
	if _, err := s.GetHouseholdByID(householdID); err != nil {
		return nil, err
	}

	var residents []model.Resident
	if err := s.DB.Where("household_id = ?", householdID).Find(&residents).Error; err != nil {
		return nil, err
	}

	return residents, nil
}

// 9. AssociateHouseholdWithDevice 关联户号与设备
func (s *HouseholdService) AssociateHouseholdWithDevice(householdID, deviceID uint) error {
	// 检查户号是否存在
	if _, err := s.GetHouseholdByID(householdID); err != nil {
		return err
	}

	// 检查设备是否存在
	var device model.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("设备不存在")
		}
		return err
	}

	// 直接更新设备的household_id字段
	if err := s.DB.Model(&device).Update("household_id", householdID).Error; err != nil {
		return err
	}

	return nil
}

// 10. RemoveHouseholdDeviceAssociation 解除户号与设备的关联
func (s *HouseholdService) RemoveHouseholdDeviceAssociation(householdID, deviceID uint) error {
	// 检查户号是否存在
	if _, err := s.GetHouseholdByID(householdID); err != nil {
		return err
	}

	// 检查设备是否存在
	var device model.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("设备不存在")
		}
		return err
	}

	// 检查设备是否属于该户号
	if device.HouseholdID != householdID {
		return errors.New("该设备不属于此户号")
	}

	// 将设备的household_id设为NULL，表示解除关联
	if err := s.DB.Model(&device).Update("household_id", nil).Error; err != nil {
		return err
	}

	return nil
}
