package model

import (
	"time"
)

// EmergencyAlarm 表示紧急警报信息
type EmergencyAlarm struct {
	BaseModel
	Type        string     `gorm:"type:varchar(30);not null" json:"type"` // 如：fire(火灾)、intrusion(入侵)、medical(医疗)等
	Location    string     `gorm:"type:varchar(100);not null" json:"location"`
	Description string     `gorm:"type:text" json:"description"`
	Status      string     `gorm:"type:varchar(20);default:'triggered'" json:"status"` // 如：triggered(已触发)、processing(处理中)、resolved(已解决)
	Timestamp   time.Time  `json:"timestamp"`
	ReportedBy  uint       `json:"reported_by"` // 报告人ID，0表示系统自动报警
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy  *uint      `json:"resolved_by,omitempty"`
	Resolution  string     `gorm:"type:text" json:"resolution,omitempty"`
	PropertyID  *uint      `json:"property_id,omitempty"` // 可为空，非外键
}
