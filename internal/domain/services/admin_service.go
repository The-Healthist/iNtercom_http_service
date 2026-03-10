package services

import (
	"errors"
	"fmt"
	"intercom_http_service/internal/domain/models"
	"intercom_http_service/internal/infrastructure/config"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// InterfaceAdminService Admin服务接口
type InterfaceAdminService interface {
	CheckPassword(password, hash string) bool
	GetAdminByID(id uint) (*models.Admin, error)
	GetAdminByUsername(username string) (*models.Admin, error)
	GetAllAdmins(page, pageSize int, search string) ([]models.Admin, int64, error)
	CreateAdmin(admin *models.Admin) error
	UpdateAdmin(id uint, updates map[string]interface{}) (*models.Admin, error)
	DeleteAdmin(id uint) error
}

// AdminService 提供管理员相关的服务
type AdminService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewAdminService 创建一个新的管理员服务
func NewAdminService(db *gorm.DB, cfg *config.Config) InterfaceAdminService {
	return &AdminService{
		DB:     db,
		Config: cfg,
	}
}

// 1  CheckPassword 验证密码是否匹配
func (s *AdminService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// 2 GetAllAdmins 获取所有管理员，支持分页
func (s *AdminService) GetAllAdmins(page, pageSize int, search string) ([]models.Admin, int64, error) {
	var admins []models.Admin
	var total int64

	query := s.DB.Model(&models.Admin{})

	// 添加搜索条件
	if search != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR phone LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&admins).Error; err != nil {
		return nil, 0, err
	}

	// 返回数据和总数
	return admins, total, nil
}

// 3  GetAdminByID 根据ID获取管理员
func (s *AdminService) GetAdminByID(id uint) (*models.Admin, error) {
	var admin models.Admin
	if err := s.DB.First(&admin, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("管理员不存在")
		}
		return nil, err
	}
	return &admin, nil
}

// 4  GetAdminByUsername 根据用户名获取管理员
func (s *AdminService) GetAdminByUsername(username string) (*models.Admin, error) {
	var admin models.Admin
	if err := s.DB.Where("username = ?", username).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

// 5  CreateAdmin 创建新管理员
func (s *AdminService) CreateAdmin(admin *models.Admin) error {
	// 验证用户名唯一性
	var count int64
	if err := s.DB.Model(&models.Admin{}).Where("username = ?", admin.Username).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("用户名已存在")
	}

	// 设置密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(admin.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return fmt.Errorf("密码加密失败: %v", err)
	}
	admin.Password = string(hashedPassword)

	return s.DB.Create(admin).Error
}

// 6  UpdateAdmin 更新管理员信息
func (s *AdminService) UpdateAdmin(id uint, updates map[string]interface{}) (*models.Admin, error) {
	// 首先获取管理员
	admin, err := s.GetAdminByID(id)
	if err != nil {
		return nil, err
	}

	// 如果更新用户名，需要检查唯一性
	if username, ok := updates["username"].(string); ok && username != admin.Username {
		var count int64
		if err := s.DB.Model(&models.Admin{}).Where("username = ? AND id != ?", username, admin.ID).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("用户名已被其他管理员使用")
		}
	}

	// 如果更新密码，需要进行哈希处理
	if password, ok := updates["password"].(string); ok {
		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return nil, fmt.Errorf("密码加密失败: %v", err)
		}
		updates["password"] = string(hashedPassword)
	}

	if err := s.DB.Model(admin).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新获取更新后的管理员信息
	return s.GetAdminByID(id)
}

// 7  DeleteAdmin 删除管理员
func (s *AdminService) DeleteAdmin(id uint) error {
	// 确保系统中至少有一个管理员
	var count int64
	if err := s.DB.Model(&models.Admin{}).Count(&count).Error; err != nil {
		return err
	}
	if count <= 1 {
		return errors.New("系统必须至少有一个管理员，无法删除最后一个管理员")
	}

	result := s.DB.Delete(&models.Admin{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("未找到要删除的记录")
	}
	return nil
}

// 8  DeleteAdmins 批量删除管理员
func (s *AdminService) DeleteAdmins(ids []uint) error {
	// 确保系统中至少有一个管理员
	var count int64
	if err := s.DB.Model(&models.Admin{}).Count(&count).Error; err != nil {
		return err
	}

	// 检查删除操作是否会导致没有管理员
	var deleteCount int64
	if err := s.DB.Model(&models.Admin{}).Where("id IN ?", ids).Count(&deleteCount).Error; err != nil {
		return err
	}

	if count <= deleteCount {
		return errors.New("系统必须至少有一个管理员，无法删除所有管理员")
	}

	result := s.DB.Delete(&models.Admin{}, ids)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("未找到要删除的记录")
	}
	return nil
}
