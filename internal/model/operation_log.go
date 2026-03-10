package model

import (
	"time"
)

// OperationLog 表示设备操作日志
type OperationLog struct {
	BaseModel
	OperationType string    `gorm:"type:varchar(100);not null" json:"operation_type"` // 如: door_unlock, emergency_unlock, configuration_change
	DeviceID      uint      `json:"device_id"`
	UserID        uint      `json:"user_id"` // 执行操作的用户ID，0表示系统自动操作
	Details       string    `gorm:"type:text" json:"details"`
	Timestamp     time.Time `json:"timestamp"`
	Success       bool      `gorm:"default:true" json:"success"` // 操作是否成功
	IPAddress     string    `gorm:"type:varchar(45)" json:"ip_address"`

	// 关联关系
	Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}
