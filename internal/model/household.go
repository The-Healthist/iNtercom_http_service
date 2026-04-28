package model

// Household 表示户号信息
type Household struct {
	BaseModel
	HouseholdNumber string `gorm:"type:varchar(50);not null;index:idx_household_number;uniqueIndex:idx_building_household_number" json:"household_number"` // 户号编号，如"1-1-101"
	HouseCode       string `gorm:"type:varchar(20);default:'';index:idx_house_code" json:"house_code"`                                                     // 楼号编码，如08
	FloorCode       string `gorm:"type:varchar(20);default:'';index:idx_floor_code" json:"floor_code"`                                                     // 楼层编码，如01-02
	UnitCode        string `gorm:"type:varchar(50);default:'';index:idx_unit_code" json:"unit_code"`                                                       // 单元编码，如A-B
	HouseholdExtID  string `gorm:"type:varchar(120);default:'';index:idx_household_ext_id" json:"household_ext_id"`                                        // 扩展户号ID，如080102AB
	BuildingID      uint   `gorm:"index:idx_building_id;uniqueIndex:idx_building_household_number" json:"building_id"`                                     // 关联的楼号ID
	Status          string `gorm:"type:varchar(20);default:'active';index:idx_household_status" json:"status"`                                             // 状态：active, inactive

	// Relations - 关联关系
	Building  *Building  `gorm:"foreignKey:BuildingID" json:"building,omitempty"`   // 关联的楼号（多对一）
	Residents []Resident `gorm:"foreignKey:HouseholdID" json:"residents,omitempty"` // 关联的居民（一对多）
	Devices   []Device   `gorm:"foreignKey:HouseholdID" json:"devices,omitempty"`   // 关联的设备（一对多）
}
