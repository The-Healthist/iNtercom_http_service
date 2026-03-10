package services

import (
	"errors"
	"intercom_http_service/internal/domain/models"
	"intercom_http_service/internal/infrastructure/config"
	"time"

	"gorm.io/gorm"
)

// CallStatistics 通话统计信息
type CallStatistics struct {
	TotalCalls      int64 `json:"total_calls"`
	AnsweredCalls   int64 `json:"answered_calls"`
	MissedCalls     int64 `json:"missed_calls"`
	TimeoutCalls    int64 `json:"timeout_calls"`
	AverageDuration int   `json:"average_duration"` // 秒
}

// CallFeedback 通话质量反馈
type CallFeedback struct {
	CallID    uint      `json:"call_id"`
	Rating    int       `json:"rating"`  // 1-5 星评分
	Comment   string    `json:"comment"` // 可选评论
	Issues    string    `json:"issues"`  // 问题描述
	Timestamp time.Time `json:"timestamp"`
}

// InterfaceCallRecordService defines the call record service interface
type InterfaceCallRecordService interface {
	GetAllCallRecords(page, pageSize int) ([]models.CallRecord, int64, error)
	GetCallRecordByID(id uint) (*models.CallRecord, error)
	GetCallRecordsByDeviceID(deviceID uint, page, pageSize int) ([]models.CallRecord, int64, error)
	GetCallRecordsByResidentID(residentID uint, page, pageSize int) ([]models.CallRecord, int64, error)
	GetCallStatistics() (*CallStatistics, error)
	SubmitCallFeedback(feedback *CallFeedback) error
	GetCallRecordByCallID(callID string) (*models.CallRecord, error)
}

// CallRecordService 提供通话记录相关的服务
type CallRecordService struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewCallRecordService 创建一个新的通话记录服务
func NewCallRecordService(db *gorm.DB, cfg *config.Config) InterfaceCallRecordService {
	return &CallRecordService{
		DB:     db,
		Config: cfg,
	}
}

// 1 GetAllCallRecords 获取所有通话记录，支持分页
func (s *CallRecordService) GetAllCallRecords(page, pageSize int) ([]models.CallRecord, int64, error) {
	var calls []models.CallRecord
	var total int64

	// 获取总数
	if err := s.DB.Model(&models.CallRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，并预加载关联
	offset := (page - 1) * pageSize
	if err := s.DB.Preload("Device").Preload("Residents").
		Order("timestamp DESC").
		Limit(pageSize).Offset(offset).
		Find(&calls).Error; err != nil {
		return nil, 0, err
	}

	return calls, total, nil
}

// 2 GetCallRecordByID 根据ID获取通话记录
func (s *CallRecordService) GetCallRecordByID(id uint) (*models.CallRecord, error) {
	var call models.CallRecord
	if err := s.DB.Preload("Device").Preload("Residents").First(&call, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("通话记录不存在")
		}
		return nil, err
	}
	return &call, nil
}

// 3 GetCallRecordsByDeviceID 获取指定设备的通话记录
func (s *CallRecordService) GetCallRecordsByDeviceID(deviceID uint, page, pageSize int) ([]models.CallRecord, int64, error) {
	var calls []models.CallRecord
	var total int64

	// 检查设备是否存在
	var device models.Device
	if err := s.DB.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("设备不存在")
		}
		return nil, 0, err
	}

	// 获取总数
	if err := s.DB.Model(&models.CallRecord{}).Where("device_id = ?", deviceID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，并预加载关联
	offset := (page - 1) * pageSize
	if err := s.DB.Preload("Device").Preload("Residents").
		Where("device_id = ?", deviceID).
		Order("timestamp DESC").
		Limit(pageSize).Offset(offset).
		Find(&calls).Error; err != nil {
		return nil, 0, err
	}

	return calls, total, nil
}

// 4 GetCallRecordsByResidentID 获取指定居民的通话记录
func (s *CallRecordService) GetCallRecordsByResidentID(residentID uint, page, pageSize int) ([]models.CallRecord, int64, error) {
	var calls []models.CallRecord
	var total int64

	// 检查居民是否存在
	var resident models.Resident
	if err := s.DB.First(&resident, residentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("居民不存在")
		}
		return nil, 0, err
	}

	// 获取总数
	if err := s.DB.Model(&models.CallRecord{}).Where("resident_id = ?", residentID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，并预加载关联
	offset := (page - 1) * pageSize
	if err := s.DB.Preload("Device").Preload("Residents").
		Where("resident_id = ?", residentID).
		Order("timestamp DESC").
		Limit(pageSize).Offset(offset).
		Find(&calls).Error; err != nil {
		return nil, 0, err
	}

	return calls, total, nil
}

// 5 GetCallStatistics 获取通话统计信息
func (s *CallRecordService) GetCallStatistics() (*CallStatistics, error) {
	var statistics CallStatistics
	var totalDuration int64

	// 获取总通话数
	if err := s.DB.Model(&models.CallRecord{}).Count(&statistics.TotalCalls).Error; err != nil {
		return nil, err
	}

	// 获取已接通话数
	if err := s.DB.Model(&models.CallRecord{}).Where("call_status = ?", models.CallStatusAnswered).Count(&statistics.AnsweredCalls).Error; err != nil {
		return nil, err
	}

	// 获取未接通话数
	if err := s.DB.Model(&models.CallRecord{}).Where("call_status = ?", models.CallStatusMissed).Count(&statistics.MissedCalls).Error; err != nil {
		return nil, err
	}

	// 获取超时通话数
	if err := s.DB.Model(&models.CallRecord{}).Where("call_status = ?", models.CallStatusTimeout).Count(&statistics.TimeoutCalls).Error; err != nil {
		return nil, err
	}

	// 计算平均通话时长
	if statistics.AnsweredCalls > 0 {
		var result struct {
			TotalDuration int64
		}
		if err := s.DB.Model(&models.CallRecord{}).
			Where("call_status = ?", models.CallStatusAnswered).
			Select("sum(duration) as total_duration").
			Scan(&result).Error; err != nil {
			return nil, err
		}
		totalDuration = result.TotalDuration
		statistics.AverageDuration = int(totalDuration / statistics.AnsweredCalls)
	}

	return &statistics, nil
}

// 6 SubmitCallFeedback 提交通话质量反馈
func (s *CallRecordService) SubmitCallFeedback(feedback *CallFeedback) error {
	// 验证通话记录是否存在
	var call models.CallRecord
	if err := s.DB.First(&call, feedback.CallID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通话记录不存在")
		}
		return err
	}

	// 验证评分范围
	if feedback.Rating < 1 || feedback.Rating > 5 {
		return errors.New("评分必须在1-5之间")
	}

	// 设置时间戳
	feedback.Timestamp = time.Now()

	// 这里可以存储反馈到数据库，或发送到其他服务处理
	// 例如：s.DB.Create(feedback)
	// 但由于模型中没有提供反馈表，这里仅做示例

	return nil
}

// GetCallRecordByCallID 根据通话ID获取通话记录
func (s *CallRecordService) GetCallRecordByCallID(callID string) (*models.CallRecord, error) {
	var call models.CallRecord

	// 查询字段名可能需要根据实际的数据表结构调整
	if err := s.DB.Preload("Device").Preload("Residents").
		Where("call_id = ?", callID).
		First(&call).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("通话记录不存在")
		}
		return nil, err
	}

	return &call, nil
}
