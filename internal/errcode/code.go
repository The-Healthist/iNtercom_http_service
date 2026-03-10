package errcode

// HTTP状态码.
const (
	// StatusOK - 200: 成功.
	StatusOK = 200
	// StatusBadRequest - 400: 请求参数错误.
	StatusBadRequest = 400
	// StatusUnauthorized - 401: 未授权.
	StatusUnauthorized = 401
	// StatusForbidden - 403: 禁止访问.
	StatusForbidden = 403
	// StatusNotFound - 404: 资源不存在.
	StatusNotFound = 404
	// StatusInternalServerError - 500: 服务器内部错误.
	StatusInternalServerError = 500
	// StatusTooManyRequests - 429: 请求过多.
	StatusTooManyRequests = 429
)

// 通用错误码 (100xxx).
const (
	// ErrSuccess - 200: 成功.
	ErrSuccess int = iota + 100000
	// ErrUnknown - 500: 未知错误.
	ErrUnknown
	// ErrBind - 400: 请求参数绑定错误.
	ErrBind
	// ErrValidation - 400: 请求参数验证错误.
	ErrValidation
	// ErrTokenInvalid - 401: 令牌无效.
	ErrTokenInvalid
	// ErrTooManyRequests - 429: 请求频率过高.
	ErrTooManyRequests
)

// 用户相关错误码 (101xxx).
const (
	// ErrUserNotFound - 404: 用户不存在.
	ErrUserNotFound int = iota + 101000
	// ErrUserAlreadyExist - 400: 用户已存在.
	ErrUserAlreadyExist
	// ErrUserPasswordIncorrect - 401: 用户密码错误.
	ErrUserPasswordIncorrect
)

// 设备相关错误码 (102xxx).
const (
	// ErrDeviceNotFound - 404: 设备不存在.
	ErrDeviceNotFound int = iota + 102000
	// ErrDeviceAlreadyExist - 400: 设备已存在.
	ErrDeviceAlreadyExist
	// ErrDeviceOffline - 400: 设备离线.
	ErrDeviceOffline
	// ErrDeviceBusy - 400: 设备忙.
	ErrDeviceBusy
)

// 住户相关错误码 (103xxx).
const (
	// ErrResidentNotFound - 404: 住户不存在.
	ErrResidentNotFound int = iota + 103000
	// ErrResidentAlreadyExist - 400: 住户已存在.
	ErrResidentAlreadyExist
)

// 呼叫相关错误码 (104xxx).
const (
	// ErrCallNotFound - 404: 呼叫记录不存在.
	ErrCallNotFound int = iota + 104000
	// ErrCallTimeout - 400: 呼叫超时.
	ErrCallTimeout
)

// 数据库相关错误码 (105xxx).
const (
	// ErrDatabase - 500: 数据库错误.
	ErrDatabase int = iota + 105000
	// ErrRecordNotFound - 404: 记录不存在.
	ErrRecordNotFound
)

// 迁移相关错误码 (109xxx).
const (
	// ErrMigrationFailed - 500: 迁移失败.
	ErrMigrationFailed int = iota + 109000
	// ErrBackupFailed - 500: 备份失败.
	ErrBackupFailed
	// ErrRestoreFailed - 500: 恢复失败.
	ErrRestoreFailed
	// ErrConnectionFailed - 500: 连接失败.
	ErrConnectionFailed
)
