package services

import (
	"errors"
	"intercom_http_service/internal/domain/models"
	"intercom_http_service/internal/infrastructure/config"
	"intercom_http_service/pkg/utils"

	"gorm.io/gorm"
)

// InterfaceStaffService defines the staff service interface
type InterfaceStaffService interface {
	GetAllStaff(page, pageSize int, search string) ([]models.PropertyStaff, int64, error)
	GetStaffByID(id uint) (*models.PropertyStaff, error)
	CreateStaff(staff *models.PropertyStaff) error
	UpdateStaff(id uint, updates map[string]interface{}) (*models.PropertyStaff, error)
	DeleteStaff(id uint) error
	GetStaffDevices(staffID uint) ([]models.Device, error)
	GetStaffByIDWithDevices(id uint) (*models.PropertyStaff, error)
}

// StaffService 提供物业人员相关的服务
type StaffService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewStaffService 创建一个新的物业人员服务
func NewStaffService(db *gorm.DB, cfg *config.Config) InterfaceStaffService {
	return &StaffService{
		DB:     db,
		Config: cfg,
	}
}

// 1 GetAllStaff 获取所有物业人员，支持分页和搜索
func (s *StaffService) GetAllStaff(page, pageSize int, search string) ([]models.PropertyStaff, int64, error) {
	var staff []models.PropertyStaff
	var total int64

	query := s.DB.Model(&models.PropertyStaff{})

	// 如果有搜索关键词，添加搜索条件
	if search != "" {
		query = query.Where("name LIKE ? OR phone LIKE ? OR property_name LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Limit(pageSize).Offset(offset).Find(&staff).Error; err != nil {
		return nil, 0, err
	}

	return staff, total, nil
}

// 2 GetStaffByID 根据ID获取物业人员
func (s *StaffService) GetStaffByID(id uint) (*models.PropertyStaff, error) {
	var staff models.PropertyStaff
	if err := s.DB.First(&staff, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("物业员工不存在")
		}
		return nil, err
	}
	return &staff, nil
}

// 3 CreateStaff 创建新物业人员
func (s *StaffService) CreateStaff(staff *models.PropertyStaff) error {
	// 验证手机号唯一性
	var count int64
	if err := s.DB.Model(&models.PropertyStaff{}).Where("phone = ?", staff.Phone).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("手机号已被使用")
	}

	// 验证用户名唯一性
	if err := s.DB.Model(&models.PropertyStaff{}).Where("username = ?", staff.Username).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("用户名已存在")
	}

	// 处理密码（在实际应用中应该进行哈希处理）
	hashedPassword, err := utils.HashPassword(staff.Password)
	if err != nil {
		return errors.New("密码加密失败")
	}
	staff.Password = hashedPassword

	return s.DB.Create(staff).Error
}

// 4 UpdateStaff 更新物业人员信息
func (s *StaffService) UpdateStaff(id uint, updates map[string]interface{}) (*models.PropertyStaff, error) {
	staff, err := s.GetStaffByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新手机号，需要检查唯一性
	if phone, ok := updates["phone"].(string); ok && phone != staff.Phone {
		var count int64
		if err := s.DB.Model(&models.PropertyStaff{}).Where("phone = ? AND id != ?", phone, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("手机号已被其他物业员工使用")
		}
	}

	// 如果更新用户名，需要检查唯一性
	if username, ok := updates["username"].(string); ok && username != staff.Username {
		var count int64
		if err := s.DB.Model(&models.PropertyStaff{}).Where("username = ? AND id != ?", username, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("用户名已被其他物业员工使用")
		}
	}

	// 如果更新密码，需要进行哈希处理
	if password, ok := updates["password"].(string); ok {
		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			return nil, errors.New("密码加密失败")
		}
		updates["password"] = hashedPassword
	}

	if err := s.DB.Model(staff).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新获取更新后的物业人员信息
	return s.GetStaffByID(id)
}

// 5 DeleteStaff 删除物业人员
func (s *StaffService) DeleteStaff(id uint) error {
	staff, err := s.GetStaffByID(id)
	if err != nil {
		return err
	}
	return s.DB.Delete(staff).Error
}

// 6 GetStaffDevices 获取物业人员管理的设备列表
func (s *StaffService) GetStaffDevices(staffID uint) ([]models.Device, error) {
	// 检查物业人员是否存在
	staff, err := s.GetStaffByID(staffID)
	if err != nil {
		return nil, err
	}

	// 查询所有关联该物业人员的设备
	var devices []models.Device
	if err := s.DB.Where("property_id = ?", staff.ID).Find(&devices).Error; err != nil {
		return nil, err
	}

	return devices, nil
}

// 7 GetStaffByIDWithDevices 获取物业人员信息及其管理的设备
func (s *StaffService) GetStaffByIDWithDevices(id uint) (*models.PropertyStaff, error) {
	staff, err := s.GetStaffByID(id)
	if err != nil {
		return nil, err
	}

	// 手动加载设备信息
	var devices []models.Device
	if err := s.DB.Where("property_id = ?", staff.ID).Find(&devices).Error; err != nil {
		return nil, err
	}
	staff.Devices = devices

	return staff, nil
}
