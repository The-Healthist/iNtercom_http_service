package model

// DeviceStatus represents the status of a door access device
type DeviceStatus string

const (
	DeviceStatusOnline  DeviceStatus = "online"
	DeviceStatusOffline DeviceStatus = "offline"
	DeviceStatusFault   DeviceStatus = "fault"
)

// Device represents door access devices
type Device struct {
	BaseModel
	Name         string       `gorm:"type:varchar(50);not null" json:"name"`
	SerialNumber string       `gorm:"type:varchar(50);uniqueIndex;not null" json:"serial_number"`
	Location     string       `gorm:"type:varchar(100)" json:"location"`
	Status       DeviceStatus `gorm:"type:varchar(20);default:'offline'" json:"status"`
	BuildingID   uint         `gorm:"index:idx_devices_building_id" json:"building_id,omitempty"`   // 关联的楼号ID
	HouseholdID  uint         `gorm:"index:idx_devices_household_id" json:"household_id,omitempty"` // 关联的户号ID

	// Relations - 关联关系
	Staff         []PropertyStaff `gorm:"many2many:staff_device_relations;" json:"staff,omitempty"` // 通过关系表关联的物业人员列表
	Building      *Building       `gorm:"foreignKey:BuildingID" json:"building,omitempty"`          // 关联的楼号（多对一）
	Household     *Household      `gorm:"foreignKey:HouseholdID" json:"household,omitempty"`        // 关联的户号（多对一）
	CallRecords   []CallRecord    `gorm:"foreignKey:DeviceID" json:"call_records,omitempty"`
	AccessLogs    []AccessLog     `gorm:"foreignKey:DeviceID" json:"access_logs,omitempty"`
	EmergencyLogs []EmergencyLog  `gorm:"foreignKey:DeviceID" json:"emergency_logs,omitempty"`
}
