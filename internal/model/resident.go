package model

// Resident represents home residents
type Resident struct {
	BaseModel
	Name        string `gorm:"type:varchar(50);not null" json:"name"`
	Email       string `gorm:"type:varchar(100)" json:"email"`
	Phone       string `gorm:"type:varchar(20);not null" json:"phone"`
	Password    string `gorm:"type:varchar(100);not null" json:"-"` // 不在JSON中暴露密码
	HouseholdID uint   `gorm:"index" json:"household_id"`           // 关联的户号ID

	// Relations
	Household     *Household     `gorm:"foreignKey:HouseholdID" json:"household,omitempty"` // 所属户号
	CallRecords   []CallRecord   `gorm:"foreignKey:ResidentID" json:"call_records,omitempty"`
	AccessLogs    []AccessLog    `gorm:"foreignKey:ResidentID" json:"access_logs,omitempty"`
	EmergencyLogs []EmergencyLog `gorm:"foreignKey:ResidentID" json:"emergency_logs,omitempty"`
}
