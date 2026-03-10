package services

import (
	"errors"
	"intercom_http_service/internal/domain/models"
	"intercom_http_service/internal/infrastructure/config"

	"gorm.io/gorm"
)

// InterfaceResidentService defines the resident service interface
type InterfaceResidentService interface {
	GetAllResidents(page int, pageSize int) ([]models.Resident, int64, error)
	GetResidentByID(id uint) (*models.Resident, error)
	CreateResident(resident *models.Resident) error
	UpdateResident(id uint, updates map[string]interface{}) (*models.Resident, error)
	DeleteResident(id uint) error
}

// ResidentService 提供居民相关的服务
type ResidentService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewResidentService 创建一个新的居民服务
func NewResidentService(db *gorm.DB, cfg *config.Config) InterfaceResidentService {
	return &ResidentService{
		DB:     db,
		Config: cfg,
	}
}

// 1 GetAllResidents 获取所有居民
func (s *ResidentService) GetAllResidents(page int, pageSize int) ([]models.Resident, int64, error) {
	var residents []models.Resident
	var total int64
	if err := s.DB.Model(&models.Resident{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := s.DB.Offset((page - 1) * pageSize).Limit(pageSize).Find(&residents).Error; err != nil {
		return nil, 0, err
	}
	return residents, total, nil
}

// 2 GetResidentByID 根据ID获取居民
func (s *ResidentService) GetResidentByID(id uint) (*models.Resident, error) {
	var resident models.Resident
	if err := s.DB.First(&resident, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("居民不存在")
		}
		return nil, err
	}
	return &resident, nil
}

// 3 CreateResident 创建新居民
func (s *ResidentService) CreateResident(resident *models.Resident) error {
	// 验证手机号唯一性
	var count int64
	if err := s.DB.Model(&models.Resident{}).Where("phone = ?", resident.Phone).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("手机号已被使用")
	}

	// 验证户号是否存在
	if resident.HouseholdID > 0 {
		var household models.Household
		if err := s.DB.First(&household, resident.HouseholdID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("户号不存在")
			}
			return err
		}
	} else {
		return errors.New("必须提供有效的户号ID")
	}

	return s.DB.Create(resident).Error
}

// 4 UpdateResident 更新居民信息
func (s *ResidentService) UpdateResident(id uint, updates map[string]interface{}) (*models.Resident, error) {
	resident, err := s.GetResidentByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新手机号，需要检查唯一性
	if phone, ok := updates["phone"].(string); ok && phone != resident.Phone {
		var count int64
		if err := s.DB.Model(&models.Resident{}).Where("phone = ? AND id != ?", phone, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("手机号已被其他居民使用")
		}
	}

	// 如果更新户号ID，需要验证户号是否存在
	if householdID, ok := updates["household_id"].(uint); ok {
		var household models.Household
		if err := s.DB.First(&household, householdID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("户号不存在")
			}
			return nil, err
		}
	}

	if err := s.DB.Model(resident).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新获取更新后的居民信息
	return s.GetResidentByID(id)
}

// 5 DeleteResident 删除居民
func (s *ResidentService) DeleteResident(id uint) error {
	resident, err := s.GetResidentByID(id)
	if err != nil {
		return err
	}
	return s.DB.Delete(resident).Error
}
