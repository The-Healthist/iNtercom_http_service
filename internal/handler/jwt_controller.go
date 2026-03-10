package handler

import (
	"intercom_http_service/internal/model"
	"intercom_http_service/internal/service"
	"intercom_http_service/internal/errcode"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// InterfaceJWTController 定义认证控制器接口
type InterfaceJWTController interface {
	Login()
}

// JWTController 处理身份验证请求
type JWTController struct {
	Ctx       *gin.Context
	Container *service.ServiceContainer
}

// NewJWTController 创建一个新的认证控制器
func NewJWTController(ctx *gin.Context, container *service.ServiceContainer) *JWTController {
	return &JWTController{
		Ctx:       ctx,
		Container: container,
	}
}

// LoginRequest 表示登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"admin123"`
}

// LoginResponse 表示登录响应
type LoginResponse struct {
	Code    int         `json:"code" example:"0"`
	Message string      `json:"message" example:"Login successful"`
	Data    interface{} `json:"data"`
}

// LoginData 表示登录成功后返回的数据
type LoginData struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	UserID    uint   `json:"user_id" example:"1"`
	Role      string `json:"role" example:"admin"`
	Username  string `json:"username" example:"admin"`
	CreatedAt string `json:"created_at" example:"2023-01-01T00:00:00Z"`
}

// ErrorResponse 表示错误响应
type ErrorResponse struct {
	Code    int         `json:"code" example:"401"`
	Message string      `json:"message" example:"Invalid username or password"`
	Data    interface{} `json:"data"`
}

// HandleJWTFunc 返回一个处理JWT认证请求的Gin处理函数
func HandleJWTFunc(container *service.ServiceContainer, method string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		controller := NewJWTController(ctx, container)

		switch method {
		case "login":
			controller.Login()
		default:
			errcode.FailWithMessage(ctx, errcode.ErrBind, "无效的方法", nil)
		}
	}
}

// Login 处理用户登录
// @Summary      User Login
// @Description  Process user login and return JWT token with different permissions based on user role
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login request parameters"
// @Success      200  {object}  LoginResponse{data=LoginData}  "Success response with token"
// @Failure      400  {object}  ErrorResponse  "Bad request"
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /auth/login [post]
func (c *JWTController) Login() {
	var req LoginRequest
	if err := c.Ctx.ShouldBindJSON(&req); err != nil {
		errcode.FailWithMessage(c.Ctx, errcode.ErrBind, "无效的请求参数", nil)
		return
	}

	// 获取数据库连接
	db := c.Container.GetDB()
	// 获取JWT服务
	jwtService := c.Container.GetService("jwt").(*service.JWTService)

	// 尝试查找管理员用户
	var admin model.Admin
	if err := db.Where("username = ?", req.Username).First(&admin).Error; err == nil {
		// 比较密码
		if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)); err == nil {
			// 生成管理员令牌
			token, err := jwtService.GenerateToken(admin.ID, "admin", nil, nil)
			if err != nil {
				errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "生成令牌失败", nil)
				return
			}

			errcode.Success(c.Ctx, gin.H{
				"token":      token,
				"user_id":    admin.ID,
				"role":       "admin",
				"username":   admin.Username,
				"created_at": admin.CreatedAt,
			})
			return
		}
	}

	// 尝试查找物业人员
	var staff model.PropertyStaff
	if err := db.Where("username = ?", req.Username).First(&staff).Error; err == nil {
		// 获取密码字段
		var password string

		// 使用原始查询获取所需字段，移除对不存在的property_id的引用
		row := db.Table("property_staffs").
			Select("password").
			Where("id = ?", staff.ID).
			Row()

		if err := row.Scan(&password); err != nil {
			errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "数据库错误: "+err.Error(), nil)
			return
		}

		// 比较密码
		if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err == nil {
			// 生成物业人员令牌，不再传递propertyID
			token, err := jwtService.GenerateToken(staff.ID, "staff", nil, nil)
			if err != nil {
				errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "生成令牌失败", nil)
				return
			}

			// 获取用户名
			var username string
			db.Table("property_staffs").
				Select("username").
				Where("id = ?", staff.ID).
				Row().
				Scan(&username)

			errcode.Success(c.Ctx, gin.H{
				"token":      token,
				"user_id":    staff.ID,
				"role":       "staff",
				"username":   username,
				"created_at": staff.CreatedAt,
			})
			return
		}
	}

	// 尝试查找普通居民
	var resident model.Resident
	if err := db.Where("phone = ?", req.Username).First(&resident).Error; err == nil {
		// 获取密码字段
		var password string
		var name string
		var phone string

		// 使用原始查询获取所需字段
		row := db.Table("residents").
			Select("password, name, phone").
			Where("id = ?", resident.ID).
			Row()

		if err := row.Scan(&password, &name, &phone); err != nil {
			errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "数据库错误: "+err.Error(), nil)
			return
		}

		// 比较密码
		if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err == nil {
			// 生成居民令牌
			token, err := jwtService.GenerateToken(resident.ID, "user", nil, nil)
			if err != nil {
				errcode.FailWithMessage(c.Ctx, errcode.ErrDatabase, "生成令牌失败", nil)
				return
			}

			errcode.Success(c.Ctx, gin.H{
				"token":      token,
				"user_id":    resident.ID,
				"role":       "user",
				"username":   name,
				"phone":      phone,
				"created_at": resident.CreatedAt,
			})
			return
		}
	}

	// 用户名或密码无效
	errcode.Unauthorized(c.Ctx)
}
