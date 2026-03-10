package model

import (
	"time"
)

// SystemLog represents system operation logs
type SystemLog struct {
	BaseModel
	AdminID   uint      `json:"admin_id"`
	Action    string    `gorm:"type:varchar(100);not null" json:"action"`
	Target    string    `gorm:"type:varchar(100)" json:"target"` // Target of action (device, user, config)
	IPAddress string    `gorm:"type:varchar(45)" json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`

	// Relations
	Admin *Admin `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}
