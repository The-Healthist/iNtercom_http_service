package service

import (
	"errors"
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/config"

	"gorm.io/gorm"
)

// InterfaceDeviceService defines the device service interface
type InterfaceDeviceService interface {
	GetAllDevices() ([]model.Device, error)
	GetDevicesByBuilding(buildingID uint) ([]model.Device, error)
	GetDeviceByID(id uint) (*model.Device, error)
	CreateDevice(device *model.Device) error
	UpdateDevice(id uint, updates map[string]interface{}) (*model.Device, error)
	DeleteDevice(id uint) error
	GetDeviceStatus(id uint) (string, error)
	UpdateDeviceConfiguration(id uint, config map[string]interface{}) error
	RebootDevice(id uint) error
	UnlockDevice(id uint) error
	GetDeviceHouseholds(deviceID uint) ([]model.Household, error)
	GetDeviceBuilding(deviceID uint) (*model.Building, error)
}

// DeviceService 提供设备相关的服务
type DeviceService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewDeviceService 创建一个新的设备服务
func NewDeviceService(db *gorm.DB, cfg *config.Config) InterfaceDeviceService {
	return &DeviceService{
		DB:     db,
		Config: cfg,
	}
}

// 1 GetAllDevices 获取所有设备列表
func (s *DeviceService) GetAllDevices() ([]model.Device, error) {
	var devices []model.Device
	if err := s.DB.Preload("Staff").Preload("Building").Find(&devices).Error; err != nil {
		return nil, err
	}

	return devices, nil
}

// 1.2 GetDevicesByBuilding 根据楼号获取设备列表
func (s *DeviceService) GetDevicesByBuilding(buildingID uint) ([]model.Device, error) {
	var devices []model.Device
	if err := s.DB.Where("building_id = ?", buildingID).Preload("Staff").Preload("Building").Find(&devices).Error; err != nil {
		return nil, err
	}

	return devices, nil
}

// 2 GetDeviceByID 根据ID获取设备
func (s *DeviceService) GetDeviceByID(id uint) (*model.Device, error) {
	var device model.Device
	if err := s.DB.Preload("Staff").
		Preload("Building").
		Preload("Household").
		First(&device, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("设备不存在")
		}
		return nil, err
	}

	return &device, nil
}

// 3 CreateDevice 创建新设备
func (s *DeviceService) CreateDevice(device *model.Device) error {
	// 验证序列号唯一性
	var count int64
	if err := s.DB.Model(&model.Device{}).Where("serial_number = ?", device.SerialNumber).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("设备序列号已存在")
	}

	// 设置默认状态
	if device.Status == "" {
		device.Status = model.DeviceStatusOffline
	}

	return s.DB.Create(device).Error
}

// 4 UpdateDevice 更新设备信息
func (s *DeviceService) UpdateDevice(id uint, updates map[string]interface{}) (*model.Device, error) {
	device, err := s.GetDeviceByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新序列号，需要检查唯一性
	if serialNumber, ok := updates["serial_number"].(string); ok && serialNumber != device.SerialNumber {
		var count int64
		if err := s.DB.Model(&model.Device{}).Where("serial_number = ? AND id != ?", serialNumber, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("设备序列号已存在")
		}
	}

	if err := s.DB.Model(device).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新获取更新后的设备信息
	return s.GetDeviceByID(id)
}

// 5 DeleteDevice 删除设备
func (s *DeviceService) DeleteDevice(id uint) error {
	device, err := s.GetDeviceByID(id)
	if err != nil {
		return err
	}

	// 不再需要删除多对多关系表中的记录
	// 直接删除设备即可
	return s.DB.Delete(device).Error
}

// 6 GetDeviceStatus 获取设备状态 (TODO: 硬件集成)
func (s *DeviceService) GetDeviceStatus(id uint) (string, error) {
	device, err := s.GetDeviceByID(id)
	if err != nil {
		return "", err
	}
	// TODO: 与硬件集成，获取实时设备状态
	return string(device.Status), nil
}

// 7 UpdateDeviceConfiguration 更新设备配置 (TODO: 硬件集成)
func (s *DeviceService) UpdateDeviceConfiguration(id uint, config map[string]interface{}) error {
	_, err := s.GetDeviceByID(id)
	if err != nil {
		return err
	}
	// TODO: 与硬件集成，更新设备配置
	return errors.New("功能尚未实现，需要硬件集成")
}

// 8 RebootDevice 重启设备 (TODO: 硬件集成)
func (s *DeviceService) RebootDevice(id uint) error {
	_, err := s.GetDeviceByID(id)
	if err != nil {
		return err
	}
	// TODO: 与硬件集成，发送重启指令
	return errors.New("功能尚未实现，需要硬件集成")
}

// 9 UnlockDevice 远程开门 (TODO: 硬件集成)
func (s *DeviceService) UnlockDevice(id uint) error {
	_, err := s.GetDeviceByID(id)
	if err != nil {
		return err
	}
	// TODO: 与硬件集成，发送开门指令
	return errors.New("功能尚未实现，需要硬件集成")
}

// 10 GetDeviceHouseholds 获取设备关联的户号
func (s *DeviceService) GetDeviceHouseholds(deviceID uint) ([]model.Household, error) {
	// 检查设备是否存在
	device, err := s.GetDeviceByID(deviceID)
	if err != nil {
		return nil, err
	}

	// 如果设备没有关联户号
	if device.HouseholdID == 0 {
		return nil, errors.New("设备未关联户号")
	}

	var household model.Household
	if err := s.DB.First(&household, device.HouseholdID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("关联的户号不存在")
		}
		return nil, err
	}

	return []model.Household{household}, nil
}

// 11 GetDeviceBuilding 获取设备所属的楼号
func (s *DeviceService) GetDeviceBuilding(deviceID uint) (*model.Building, error) {
	// 检查设备是否存在
	device, err := s.GetDeviceByID(deviceID)
	if err != nil {
		return nil, err
	}

	// 如果设备没有关联楼号
	if device.BuildingID == 0 {
		return nil, errors.New("设备未关联楼号")
	}

	var building model.Building
	if err := s.DB.First(&building, device.BuildingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("关联的楼号不存在")
		}
		return nil, err
	}

	return &building, nil
}
