package model

// EmergencyContact 表示紧急联系人信息
type EmergencyContact struct {
	BaseModel
	Name         string `gorm:"type:varchar(50);not null" json:"name"`
	PhoneNumber  string `gorm:"type:varchar(20);not null" json:"phone_number"`
	Role         string `gorm:"type:varchar(30);not null" json:"role"` // 如：警察、消防、医院、物业经理等
	Priority     int    `gorm:"default:0" json:"priority"`             // 联系优先级，数字越大优先级越高
	PropertyID   *uint  `json:"property_id,omitempty"`                 // 关联的物业ID，可以为空表示全局联系人
	PropertyName string `gorm:"type:varchar(100)" json:"property_name,omitempty"`
	Remark       string `gorm:"type:text" json:"remark,omitempty"`
}
