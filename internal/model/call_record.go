package model

import (
	"time"
)

// CallStatus represents the status of a call
type CallStatus string

const (
	CallStatusAnswered CallStatus = "answered"
	CallStatusMissed   CallStatus = "missed"
	CallStatusTimeout  CallStatus = "timeout"
)

// CallRecord represents call records between devices and residents
type CallRecord struct {
	BaseModel
	CallID     string     `gorm:"type:varchar(100);index" json:"call_id"` // 通话唯一标识
	DeviceID   uint       `json:"device_id"`
	ResidentID uint       `json:"resident_id"`
	CallStatus CallStatus `gorm:"type:varchar(20)" json:"call_status"`
	Timestamp  time.Time  `json:"timestamp"` // 通话开始时间
	Duration   int        `json:"duration"`  // 通话时长

	// Relations
	Device   *Device   `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	Resident *Resident `gorm:"foreignKey:ResidentID" json:"resident,omitempty"`
}
