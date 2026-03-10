package model

// PropertyStaff 表示物业员工
type PropertyStaff struct {
	BaseModel
	Phone        string `gorm:"type:varchar(20);unique;not null" json:"phone"`
	PropertyName string `gorm:"type:varchar(100)" json:"property_name"`
	Position     string `gorm:"type:varchar(50)" json:"position"`
	Role         string `gorm:"type:varchar(20);not null" json:"role"` // manager, staff, etc.
	Status       string `gorm:"type:varchar(20);default:'active'" json:"status"`
	Remark       string `gorm:"type:text" json:"remark"`
	Username     string `gorm:"type:varchar(50);unique;not null" json:"username"`
	Password     string `gorm:"type:varchar(100);not null" json:"-"` // Password not exposed in JSON

	// 关联关系 - 使用多对多关系替代直接关联
	Devices []Device `gorm:"many2many:staff_device_relations;" json:"devices,omitempty"` // 通过关系表关联的设备列表
}
