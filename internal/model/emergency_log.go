package model

import (
	"time"
)

// 紧急状态
type EmergencyStatus string

const (
	EmergencyStatusPending   EmergencyStatus = "pending"
	EmergencyStatusResponded EmergencyStatus = "responded"
	EmergencyStatusEscalated EmergencyStatus = "escalated"
	EmergencyStatusResolved  EmergencyStatus = "resolved"
)

// 紧急事件日志
type EmergencyLog struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	ResidentID  uint            `json:"resident_id"`
	DeviceID    uint            `json:"device_id"`
	Status      EmergencyStatus `gorm:"type:varchar(20)" json:"status"`
	TriggeredAt time.Time       `json:"triggered_at"`
	ResolvedAt  *time.Time      `json:"resolved_at"` // 可空字段

	// Relations
	Device   *Device   `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	Resident *Resident `gorm:"foreignKey:ResidentID" json:"resident,omitempty"`
}
