package service

import (
	"errors"
	"fmt"
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// InterfaceJWTService 定义JWT服务接口
type InterfaceJWTService interface {
	GenerateToken(userID uint, role string, propertyID, deviceID *uint) (string, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	ExtractClaims(tokenString string) (*JWTClaims, error)
	Login(username, password string) (*LoginResult, error)
}

// LoginResult 表示登录结果
type LoginResult struct {
	Token     string      `json:"token"`
	UserID    uint        `json:"user_id"`
	Role      string      `json:"role"`
	Username  string      `json:"username"`
	Phone     string      `json:"phone,omitempty"`
	CreatedAt interface{} `json:"created_at"`
}

// JWTService 提供JWT相关服务
type JWTService struct {
	secretKey string
	issuer    string
	DB        *gorm.DB
}

// JWTClaims 定义JWT令牌的声明结构
type JWTClaims struct {
	UserID     uint   `json:"user_id"`
	Role       string `json:"role"`
	PropertyID *uint  `json:"property_id,omitempty"` // 物业ID，用于标识用户所属物业
	DeviceID   *uint  `json:"device_id,omitempty"`
	jwt.RegisteredClaims
}

// NewJWTService 创建一个新的JWT服务
func NewJWTService(cfg *config.Config, db *gorm.DB) InterfaceJWTService {
	return &JWTService{
		secretKey: cfg.JWTSecretKey,
		issuer:    "intercom_http_service",
		DB:        db,
	}
}

// GenerateToken 生成JWT令牌
func (s *JWTService) GenerateToken(userID uint, role string, propertyID, deviceID *uint) (string, error) {
	// 令牌有效期为24小时
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &JWTClaims{
		UserID:     userID,
		Role:       role,
		PropertyID: propertyID,
		DeviceID:   deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// ValidateToken 验证JWT令牌
func (s *JWTService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})
}

// ExtractClaims 从令牌中提取声明
func (s *JWTService) ExtractClaims(tokenString string) (*JWTClaims, error) {
	token, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// 将map claims转换为JWTClaims结构
		jwtClaims := &JWTClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer: claims["iss"].(string),
			},
		}

		// 提取用户ID
		if userID, ok := claims["user_id"].(float64); ok {
			jwtClaims.UserID = uint(userID)
		}

		// 提取角色
		if role, ok := claims["role"].(string); ok {
			jwtClaims.Role = role
		}

		// 提取物业ID（如果存在）
		if propertyID, ok := claims["property_id"].(float64); ok {
			propID := uint(propertyID)
			jwtClaims.PropertyID = &propID
		}

		// 提取设备ID（如果存在）
		if deviceID, ok := claims["device_id"].(float64); ok {
			devID := uint(deviceID)
			jwtClaims.DeviceID = &devID
		}

		return jwtClaims, nil
	}

	return nil, errors.New("invalid token claims")
}

// Login 处理用户登录请求
func (s *JWTService) Login(username, password string) (*LoginResult, error) {
	// 尝试查找管理员用户
	var admin model.Admin
	if err := s.DB.Where("username = ?", username).First(&admin).Error; err == nil {
		// 比较密码
		if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err == nil {
			// 生成管理员令牌
			token, err := s.GenerateToken(admin.ID, "admin", nil, nil)
			if err != nil {
				return nil, err
			}

			return &LoginResult{
				Token:     token,
				UserID:    admin.ID,
				Role:      "admin",
				Username:  admin.Username,
				CreatedAt: admin.CreatedAt,
			}, nil
		}
	}

	// 尝试查找物业人员
	var staff model.PropertyStaff
	if err := s.DB.Where("username = ?", username).First(&staff).Error; err == nil {
		// 获取密码字段
		var dbPassword string

		// 使用原始查询获取所需字段
		row := s.DB.Table("property_staffs").
			Select("password").
			Where("id = ?", staff.ID).
			Row()

		if err := row.Scan(&dbPassword); err != nil {
			return nil, err
		}

		// 比较密码
		if err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password)); err == nil {
			// 生成物业人员令牌
			token, err := s.GenerateToken(staff.ID, "staff", nil, nil)
			if err != nil {
				return nil, err
			}

			// 获取用户名
			var username string
			s.DB.Table("property_staffs").
				Select("username").
				Where("id = ?", staff.ID).
				Row().
				Scan(&username)

			return &LoginResult{
				Token:     token,
				UserID:    staff.ID,
				Role:      "staff",
				Username:  username,
				CreatedAt: staff.CreatedAt,
			}, nil
		}
	}

	// 尝试查找普通居民
	var resident model.Resident
	if err := s.DB.Where("phone = ?", username).First(&resident).Error; err == nil {
		// 获取密码字段和其他信息
		var dbPassword string
		var name string
		var phone string

		// 使用原始查询获取所需字段
		row := s.DB.Table("residents").
			Select("password, name, phone").
			Where("id = ?", resident.ID).
			Row()

		if err := row.Scan(&dbPassword, &name, &phone); err != nil {
			return nil, err
		}

		// 比较密码
		if err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password)); err == nil {
			// 生成居民令牌
			token, err := s.GenerateToken(resident.ID, "user", nil, nil)
			if err != nil {
				return nil, err
			}

			return &LoginResult{
				Token:     token,
				UserID:    resident.ID,
				Role:      "user",
				Username:  name,
				Phone:     phone,
				CreatedAt: resident.CreatedAt,
			}, nil
		}
	}

	// 用户名或密码无效
	return nil, errors.New("invalid username or password")
}
