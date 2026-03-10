package errcode

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 定义统一的响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrSuccess,
		Message: GetMessage(ErrSuccess),
		Data:    data,
	})
}

// Fail 失败响应
func Fail(c *gin.Context, errorCode int, data interface{}) {
	httpStatus := GetStatus(errorCode)
	message := GetMessage(errorCode)

	c.JSON(httpStatus, Response{
		Code:    errorCode,
		Message: message,
		Data:    data,
	})
}

// FailWithMessage 失败响应（自定义消息）
func FailWithMessage(c *gin.Context, errorCode int, message string, data interface{}) {
	httpStatus := GetStatus(errorCode)

	c.JSON(httpStatus, Response{
		Code:    errorCode,
		Message: message,
		Data:    data,
	})
}

// ParamError 参数错误响应
func ParamError(c *gin.Context, message string) {
	Fail(c, ErrValidation, nil)
}

// ServerError 服务器错误响应
func ServerError(c *gin.Context) {
	Fail(c, ErrUnknown, nil)
}

// NotFound 资源不存在响应
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "资源不存在"
	}
	FailWithMessage(c, StatusNotFound, message, nil)
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context) {
	Fail(c, ErrTokenInvalid, nil)
}
