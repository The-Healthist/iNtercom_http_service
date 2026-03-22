package service

import (
	"fmt"
	"intercom_http_service/internal/config"
	"time"

	"github.com/tencentyun/tls-sig-api-v2-golang/tencentyun"
)

// InterfaceTencentRTCService defines the Tencent RTC service interface
type InterfaceTencentRTCService interface {
	GetUserSig(userID string) (*TencentRTCTokenInfo, error)
	CreateVideoCall(deviceID, residentID string) (string, error)
	GenPrivateMapKey(userID string, roomID string, expire int) (string, error)
}

// TencentRTCService 处理与腾讯云TRTC的实时通信
type TencentRTCService struct {
	Config *config.Config
}

// TencentRTCTokenInfo 表示腾讯云TRTC的令牌信息
type TencentRTCTokenInfo struct {
	SDKAppID    int       `json:"sdk_app_id"`
	UserID      string    `json:"user_id"`
	UserSig     string    `json:"user_sig"`
	ExpireTime  time.Time `json:"expire_time"`
	RequestTime time.Time `json:"request_time"`
}

// BuildSharedTRTCRoomID 为同一通呼叫生成双方共享的房间号。
// 保持原有 room_<device>_<resident>_<unix> 风格，避免前端联调格式变化。
func BuildSharedTRTCRoomID(deviceID, residentID string, unixSeconds int64) string {
	return fmt.Sprintf("room_%s_%s_%d", deviceID, residentID, unixSeconds)
}

// NewTencentRTCService 创建一个新的腾讯云TRTC服务
func NewTencentRTCService(cfg *config.Config) InterfaceTencentRTCService {
	return &TencentRTCService{
		Config: cfg,
	}
}

// 1 GetUserSig 使用服务端方式生成腾讯云TRTC的UserSig
// 这是推荐的正式环境使用方式，密钥只存储在服务端
func (s *TencentRTCService) GetUserSig(userID string) (*TencentRTCTokenInfo, error) {
	// 检查是否配置了必要的参数
	if s.Config.TencentSDKAppID == 0 || s.Config.TencentSecretKey == "" {
		return nil, fmt.Errorf("缺少必要的腾讯云TRTC配置")
	}

	// 默认UserSig有效期为24小时
	expireSeconds := 86400
	now := time.Now()
	expireTime := now.Add(time.Duration(expireSeconds) * time.Second)

	// 使用腾讯云官方SDK生成UserSig
	userSig, err := tencentyun.GenUserSig(
		s.Config.TencentSDKAppID,
		s.Config.TencentSecretKey,
		userID,
		expireSeconds,
	)
	if err != nil {
		return nil, fmt.Errorf("生成UserSig失败: %w", err)
	}

	tokenInfo := &TencentRTCTokenInfo{
		SDKAppID:    s.Config.TencentSDKAppID,
		UserID:      userID,
		UserSig:     userSig,
		ExpireTime:  expireTime,
		RequestTime: now,
	}

	return tokenInfo, nil
}

// 2 CreateVideoCall 创建一个视频通话，返回一个唯一的房间ID
func (s *TencentRTCService) CreateVideoCall(deviceID, residentID string) (string, error) {
	// 为设备和居民生成一个通话房间号（RoomID）
	// 在腾讯云TRTC中，可以使用同一个房间号让用户进入同一个通话
	roomID := fmt.Sprintf("room_%s_%s_%d", deviceID, residentID, time.Now().Unix())

	// 实际应用中，你可能需要在数据库中记录这个通话
	// 并通过其他方式（如推送通知）通知居民有来电

	return roomID, nil
}

// 3 GenPrivateMapKey 生成用于权限控制的PrivateMapKey (可选功能)
func (s *TencentRTCService) GenPrivateMapKey(userID string, roomID string, expire int) (string, error) {
	if s.Config.TencentSDKAppID == 0 || s.Config.TencentSecretKey == "" {
		return "", fmt.Errorf("缺少必要的腾讯云TRTC配置")
	}

	// 默认权限设置：音视频全部权限
	// 1(创建房间) + 2(加入房间) + 4(发送语音) + 8(接收语音) + 16(发送视频) + 32(接收视频) + 64(发送屏幕共享) + 128(接收屏幕共享)
	privilegeMap := uint32(255)

	// 生成带字符串房间号的PrivateMapKey
	privateMapKey, err := tencentyun.GenPrivateMapKeyWithStringRoomID(
		s.Config.TencentSDKAppID,
		s.Config.TencentSecretKey,
		userID,
		expire,
		roomID,
		privilegeMap,
	)

	if err != nil {
		return "", fmt.Errorf("生成PrivateMapKey失败: %w", err)
	}

	return privateMapKey, nil
}
