package model

// Admin represents system administrators
type Admin struct {
	BaseModel
	Username string `gorm:"type:varchar(50);unique;not null" json:"username"`
	Password string `gorm:"type:varchar(100);not null" json:"-"` // Password not exposed in JSON
	Email    string `gorm:"type:varchar(100);unique" json:"email"`
	Phone    string `gorm:"type:varchar(20)" json:"phone"`
	Role     string `gorm:"type:varchar(50);default:'admin'" json:"role"`    // Role: system_admin, admin
	Status   string `gorm:"type:varchar(20);default:'active'" json:"status"` // Status: active, inactive, locked
}
