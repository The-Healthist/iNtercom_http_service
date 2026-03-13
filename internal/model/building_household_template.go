package model

// BuildingHouseholdTemplate 保存楼栋户号生成模板，便于后续一键复用
type BuildingHouseholdTemplate struct {
	BaseModel
	BuildingID    uint   `gorm:"uniqueIndex;not null" json:"building_id"`
	TemplateName  string `gorm:"type:varchar(100);default:''" json:"template_name"`
	TemplateJSON  string `gorm:"type:longtext;not null" json:"template_json"`
	TemplateVer   string `gorm:"type:varchar(20);default:'v1'" json:"template_ver"`
	LastOperator  string `gorm:"type:varchar(50);default:''" json:"last_operator"`
}
