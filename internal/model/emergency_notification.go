package model

import (
	"time"
)

// EmergencyNotification 表示紧急通知信息
type EmergencyNotification struct {
	BaseModel
	Title      string    `gorm:"type:varchar(100);not null" json:"title"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	Severity   string    `gorm:"type:varchar(10);not null" json:"severity"`    // 如：high(高)、medium(中)、low(低)
	Timestamp  time.Time `json:"timestamp"`                                    // 发送时间
	ExpiresAt  time.Time `json:"expires_at"`                                   // 过期时间
	TargetType string    `gorm:"type:varchar(20);not null" json:"target_type"` // 如：all(所有人)、residents(居民)、staff(物业人员)
	SenderID   uint      `json:"sender_id"`                                    // 发送者ID
	SenderRole string    `gorm:"type:varchar(20)" json:"sender_role"`          // 发送者角色
	PropertyID *uint     `json:"property_id,omitempty"`                        // 关联的物业ID，可以为空表示全局通知
	IsPublic   bool      `gorm:"default:false" json:"is_public"`               // 是否为公开通知
}
