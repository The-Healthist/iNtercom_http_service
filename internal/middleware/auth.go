package middleware

import (
	"intercom_http_service/internal/config"
	"intercom_http_service/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

var jwtService service.InterfaceJWTService

// InitAuthMiddleware 初始化认证中间件
func InitAuthMiddleware(cfg *config.Config, db *gorm.DB) {
	jwtService = service.NewJWTService(cfg, db)
}

// extractToken 从授权头中提取token
func extractToken(authHeader string) string {
	// 检查并移除 "Bearer " 前缀
	if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
		return authHeader[7:]
	}
	return authHeader
}

// AuthenticateSystemAdmin 验证系统管理员权限
func AuthenticateSystemAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header is required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 提取token
		tokenString := extractToken(authHeader)
		token, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token: " + err.Error(),
				"data":    nil,
			})
			c.Abort()
			return
		}

		if token.Valid {
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "Invalid token claims",
					"data":    nil,
				})
				c.Abort()
				return
			}

			// 检查是否是系统管理员
			if role, exists := claims["role"].(string); !exists || role != "admin" {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "Insufficient permissions: requires system admin role",
					"data":    nil,
				})
				c.Abort()
				return
			}

			// 存储claims到上下文
			c.Set("userID", claims["user_id"])
			c.Set("role", claims["role"])
			c.Set("claims", claims)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token",
				"data":    nil,
			})
			c.Abort()
			return
		}
	}
}

// AuthenticatePropertyStaff 验证物业人员权限
func AuthenticatePropertyStaff() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header is required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 提取token
		tokenString := extractToken(authHeader)
		token, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token: " + err.Error(),
				"data":    nil,
			})
			c.Abort()
			return
		}

		if token.Valid {
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "Invalid token claims",
					"data":    nil,
				})
				c.Abort()
				return
			}

			// 检查是否是物业人员
			role, exists := claims["role"].(string)
			if !exists || (role != "staff" && role != "admin") { // 管理员也可以访问物业人员的接口
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "Insufficient permissions: requires property staff role",
					"data":    nil,
				})
				c.Abort()
				return
			}

			// 存储claims到上下文
			c.Set("userID", claims["user_id"])
			c.Set("role", role)
			// propertyID可能不存在，所以只有在claims中有值时才设置
			if propID, exists := claims["property_id"]; exists && propID != nil {
				c.Set("propertyID", propID)
			}
			c.Set("claims", claims)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token",
				"data":    nil,
			})
			c.Abort()
			return
		}
	}
}

// AuthenticateUser 验证普通用户权限
func AuthenticateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header is required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 提取token
		tokenString := extractToken(authHeader)
		token, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token: " + err.Error(),
				"data":    nil,
			})
			c.Abort()
			return
		}

		if token.Valid {
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "Invalid token claims",
					"data":    nil,
				})
				c.Abort()
				return
			}

			// 检查是否有任何有效角色
			role, exists := claims["role"].(string)
			if !exists || (role != "user" && role != "staff" && role != "admin") {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "Insufficient permissions: requires valid user role",
					"data":    nil,
				})
				c.Abort()
				return
			}

			// 存储claims到上下文
			c.Set("userID", claims["user_id"])
			c.Set("role", role)
			c.Set("claims", claims)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token",
				"data":    nil,
			})
			c.Abort()
			return
		}
	}
}

// Authentication 通用的认证中间件
func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header is required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 检查是否是Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header format must be Bearer {token}",
				"data":    nil,
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token format",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 验证token
		token, err := jwtService.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid or expired token",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 提取claims并设置到上下文中
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token claims",
				"data":    nil,
			})
			c.Abort()
			return
		}

		c.Set("userID", claims["user_id"])
		c.Set("role", claims["role"])
		c.Set("claims", claims)
		c.Next()
	}
}
