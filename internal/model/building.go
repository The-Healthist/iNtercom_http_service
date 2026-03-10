package model

// Building 表示楼号信息
type Building struct {
	BaseModel
	BuildingName string `gorm:"type:varchar(50);not null" json:"building_name"`        // 楼号名称，如"1号楼"
	BuildingCode string `gorm:"type:varchar(20);unique;not null" json:"building_code"` // 楼号编码，如"B001"
	Address      string `gorm:"type:varchar(200)" json:"address"`                      // 楼号地址，如"小区东南角"
	Status       string `gorm:"type:varchar(20);default:'active'" json:"status"`       // 状态：active, inactive

	// 关联关系
	Households []Household `gorm:"foreignKey:BuildingID" json:"households,omitempty"` // 楼号下的户号（一对多）
	Devices    []Device    `gorm:"foreignKey:BuildingID" json:"devices,omitempty"`    // 楼号关联的设备（一对多）
}
