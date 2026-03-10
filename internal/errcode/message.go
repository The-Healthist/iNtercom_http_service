package errcode

// 错误码消息映射.
var codeMessageMap = map[int]string{
	// 通用错误码
	ErrSuccess:      "成功",
	ErrUnknown:      "未知错误",
	ErrBind:         "请求参数绑定错误",
	ErrValidation:   "请求参数验证错误",
	ErrTokenInvalid: "无效的认证令牌",

	// 用户相关错误码
	ErrUserNotFound:          "用户不存在",
	ErrUserAlreadyExist:      "用户已存在",
	ErrUserPasswordIncorrect: "用户密码错误",

	// 设备相关错误码
	ErrDeviceNotFound:     "设备不存在",
	ErrDeviceAlreadyExist: "设备已存在",
	ErrDeviceOffline:      "设备当前离线",
	ErrDeviceBusy:         "设备忙，请稍后再试",

	// 住户相关错误码
	ErrResidentNotFound:     "住户不存在",
	ErrResidentAlreadyExist: "住户已存在",

	// 呼叫相关错误码
	ErrCallNotFound: "呼叫记录不存在",
	ErrCallTimeout:  "呼叫超时",

	// 数据库相关错误码
	ErrDatabase:       "数据库错误",
	ErrRecordNotFound: "记录不存在",

	// 迁移相关错误码
	ErrMigrationFailed:  "迁移失败",
	ErrBackupFailed:     "备份失败",
	ErrRestoreFailed:    "恢复失败",
	ErrConnectionFailed: "连接失败",
}

// 错误码HTTP状态码映射.
var codeStatusMap = map[int]int{
	// 通用错误码
	ErrSuccess:      StatusOK,
	ErrUnknown:      StatusInternalServerError,
	ErrBind:         StatusBadRequest,
	ErrValidation:   StatusBadRequest,
	ErrTokenInvalid: StatusUnauthorized,

	// 用户相关错误码
	ErrUserNotFound:          StatusNotFound,
	ErrUserAlreadyExist:      StatusBadRequest,
	ErrUserPasswordIncorrect: StatusUnauthorized,

	// 设备相关错误码
	ErrDeviceNotFound:     StatusNotFound,
	ErrDeviceAlreadyExist: StatusBadRequest,
	ErrDeviceOffline:      StatusBadRequest,
	ErrDeviceBusy:         StatusBadRequest,

	// 住户相关错误码
	ErrResidentNotFound:     StatusNotFound,
	ErrResidentAlreadyExist: StatusBadRequest,

	// 呼叫相关错误码
	ErrCallNotFound: StatusNotFound,
	ErrCallTimeout:  StatusBadRequest,

	// 数据库相关错误码
	ErrDatabase:       StatusInternalServerError,
	ErrRecordNotFound: StatusNotFound,

	// 迁移相关错误码
	ErrMigrationFailed:  StatusInternalServerError,
	ErrBackupFailed:     StatusInternalServerError,
	ErrRestoreFailed:    StatusInternalServerError,
	ErrConnectionFailed: StatusInternalServerError,
}

// GetMessage 获取错误码对应的消息.
func GetMessage(code int) string {
	if msg, ok := codeMessageMap[code]; ok {
		return msg
	}
	return "未知错误"
}

// GetStatus 获取错误码对应的HTTP状态码.
func GetStatus(code int) int {
	if status, ok := codeStatusMap[code]; ok {
		return status
	}
	return StatusInternalServerError
}
