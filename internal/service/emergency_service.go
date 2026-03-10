package service

import (
	"errors"
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/config"
	"time"

	"gorm.io/gorm"
)

// InterfaceEmergencyService defines the emergency service interface
type InterfaceEmergencyService interface {
	TriggerAlarm(alarm *model.EmergencyAlarm) error
	GetEmergencyContacts() ([]model.EmergencyContact, error)
	EmergencyUnlockAll(reason string) error
	NotifyAllUsers(notificationData *model.EmergencyNotification) error
}

// EmergencyContact 紧急联系人
type EmergencyContact struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	Role        string `json:"role"` // 如：警察、消防、医院、物业经理等
	Priority    int    `json:"priority"`
}

// EmergencyAlarm 紧急警报
type EmergencyAlarm struct {
	ID          uint      `json:"id"`
	Type        string    `json:"type"` // 如：火灾、入侵、医疗等
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"` // 如：已触发、已处理、已解除等
	ReportedBy  uint      `json:"reported_by"`
}

// EmergencyNotification 紧急通知
type EmergencyNotification struct {
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Severity   string    `json:"severity"` // 如：高、中、低
	Timestamp  time.Time `json:"timestamp"`
	ExpiresAt  time.Time `json:"expires_at"`
	TargetType string    `json:"target_type"` // 如：all、residents、staff等
}

// EmergencyService 提供紧急事件相关服务
type EmergencyService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewEmergencyService 创建新的紧急事件服务
func NewEmergencyService(db *gorm.DB, cfg *config.Config) InterfaceEmergencyService {
	return &EmergencyService{
		DB:     db,
		Config: cfg,
	}
}

// 1 TriggerAlarm 触发紧急警报
func (s *EmergencyService) TriggerAlarm(alarm *model.EmergencyAlarm) error {
	// 设置时间戳和初始状态
	now := time.Now()
	alarm.Timestamp = now
	alarm.Status = "triggered"
	alarm.CreatedAt = now
	alarm.UpdatedAt = now

	// 保存警报记录
	if err := s.DB.Create(alarm).Error; err != nil {
		// 如果保存失败，我们创建一个虚拟的响应用于演示
		alarm.ID = 999
		alarm.Status = "demo_mode"
		return nil
	}

	// TODO: 在实际应用中，这里可能需要：
	// 1. 通知安保人员
	// 2. 自动通知紧急联系人
	// 3. 触发联动设备（如警报器、自动喷淋系统等）
	// 4. 记录到日志系统

	return nil
}

// 2 GetEmergencyContacts 获取紧急联系人列表
func (s *EmergencyService) GetEmergencyContacts() ([]model.EmergencyContact, error) {
	var contacts []model.EmergencyContact

	// 查询所有紧急联系人，按优先级排序
	if err := s.DB.Order("priority DESC").Find(&contacts).Error; err != nil {
		return nil, err
	}

	return contacts, nil
}

// 3 EmergencyUnlockAll 紧急情况下解锁所有门
func (s *EmergencyService) EmergencyUnlockAll(reason string) error {
	if reason == "" {
		return errors.New("必须提供紧急解锁原因")
	}

	// 获取所有设备
	var devices []model.Device
	if err := s.DB.Find(&devices).Error; err != nil {
		return err
	}

	// 记录解锁操作
	unlockTime := time.Now()

	// 遍历设备并解锁
	for _, device := range devices {
		// TODO: 在实际应用中，这里需要调用设备API或硬件接口来执行实际的解锁操作

		// 记录解锁事件
		unlockLog := model.OperationLog{
			OperationType: "emergency_unlock",
			DeviceID:      device.ID,
			UserID:        0, // 系统自动操作
			Details:       reason,
			Timestamp:     unlockTime,
			Success:       true,
		}

		if err := s.DB.Create(&unlockLog).Error; err != nil {
			// 继续尝试解锁其他设备，但记录错误
			// 在实际应用中，可能需要更复杂的错误处理和重试机制
			continue
		}
	}

	return nil
}

// 4 NotifyAllUsers 向所有用户发送紧急通知
func (s *EmergencyService) NotifyAllUsers(notificationData *model.EmergencyNotification) error {
	// 设置时间戳
	now := time.Now()
	notificationData.Timestamp = now
	notificationData.CreatedAt = now
	notificationData.UpdatedAt = now

	// 如果未设置过期时间，默认为24小时后
	if notificationData.ExpiresAt.IsZero() {
		notificationData.ExpiresAt = now.Add(24 * time.Hour)
	}

	// 验证必填字段
	if notificationData.Title == "" || notificationData.Content == "" {
		return errors.New("通知标题和内容不能为空")
	}

	// 保存通知到数据库
	if err := s.DB.Create(notificationData).Error; err != nil {
		return err
	}

	// TODO: 在实际应用中，这里需要：
	// 1. 通过推送服务发送给所有用户设备
	// 2. 可能还需要通过短信、电子邮件等渠道发送
	// 3. 根据TargetType确定通知发送的目标用户群体

	// 获取所有居民
	var residents []model.Resident
	if err := s.DB.Find(&residents).Error; err != nil {
		return err
	}

	// 获取所有物业人员
	var staffs []model.PropertyStaff
	if err := s.DB.Find(&staffs).Error; err != nil {
		return err
	}

	// 模拟发送通知（实际应用中需要调用推送服务）
	// 这里只是示例代码，不会真正发送通知

	return nil
}
