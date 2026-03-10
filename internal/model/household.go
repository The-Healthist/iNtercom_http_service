package model

// Household 表示户号信息
type Household struct {
	BaseModel
	HouseholdNumber string `gorm:"type:varchar(50);not null" json:"household_number"` // 户号编号，如"1-1-101"
	BuildingID      uint   `json:"building_id"`                                       // 关联的楼号ID
	Status          string `gorm:"type:varchar(20);default:'active'" json:"status"`   // 状态：active, inactive

	// Relations - 关联关系
	Building  *Building  `gorm:"foreignKey:BuildingID" json:"building,omitempty"`   // 关联的楼号（多对一）
	Residents []Resident `gorm:"foreignKey:HouseholdID" json:"residents,omitempty"` // 关联的居民（一对多）
	Devices   []Device   `gorm:"foreignKey:HouseholdID" json:"devices,omitempty"`   // 关联的设备（一对多）
}
