package services

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"intercom_http_service/internal/infrastructure/config"
	"intercom_http_service/pkg/utils"
	"io"
	"sort"
	"time"
)

// InterfaceRTCService defines the RTC service interface
type InterfaceRTCService interface {
	GetToken(channelID, userID string) (*RTCTokenInfo, error)
	CreateVideoCall(deviceID, residentID string) (string, error)
	ParseToken(tokenStr string) (AppToken, error)
}

// 常量
const (
	VENSION_LENGTH       = 3
	BUFFER_CAPACITY_BASE = 256
	VERSION_0            = "000"
	WILDCARD_CHARACTERS  = "*"
)

// 权限常量
const (
	PRIVILEGE_ENABLED        int32 = 1
	PRIVILEGE_AUDIO_PUBLISH  int32 = 2
	PRIVILEGE_VIDEO_PUBLISH  int32 = 4
	PRIVILEGE_SCREEN_PUBLISH int32 = 8
)

// RTCService 处理与阿里云RTC的实时通信
type RTCService struct {
	Config *config.Config
}

// RTCTokenInfo 表示需要进行身份验证的令牌
type RTCTokenInfo struct {
	AppID       string    `json:"app_id"`
	ChannelID   string    `json:"channel_id"`
	UserID      string    `json:"user_id"`
	Token       string    `json:"token"`
	ExpireTime  time.Time `json:"expire_time"`
	RequestTime time.Time `json:"request_time"`
}

// Service 表示用于令牌生成的服务信息
type Service struct {
	ChannelId string
	UserId    string
	Privilege *int32
}

// TokenOptions 表示用于令牌生成的附加选项
type TokenOptions struct {
	EngineOptions map[string]string
}

// AppToken 表示阿里云RTC令牌
type AppToken struct {
	AppId          string
	AppKey         string
	IssueTimestamp int32
	Salt           int32
	Timestamp      int32
	Service        *Service
	Options        *TokenOptions
	Signature      []byte
}

// NewRTCService 创建一个新的阿里云RTC服务
func NewRTCService(cfg *config.Config) InterfaceRTCService {
	return &RTCService{
		Config: cfg,
	}
}

// 1 GetToken 生成一个用于阿里云RTC的令牌
func (s *RTCService) GetToken(channelID, userID string) (*RTCTokenInfo, error) {
	// 检查是否需要配置
	if s.Config.AliyunAccessKey == "" || s.Config.AliyunRTCAppID == "" {
		return nil, fmt.Errorf("missing required Aliyun RTC configuration")
	}

	// 打印配置信息用于调试
	fmt.Printf("GetToken Debug - AppID: %s, AppKey: %s\n", s.Config.AliyunRTCAppID, s.Config.AliyunAccessKey)
	fmt.Printf("GetToken Debug - ChannelID: %s, UserID: %s\n", channelID, userID)

	// Create token
	now := time.Now()
	// 添加一些随机毫秒以增加时间戳的随机性
	expireTime := now.Add(24*time.Hour + time.Duration(utils.RandomInt32()%1000)*time.Millisecond)
	timestamp := int32(expireTime.Unix())

	// 打印时间戳用于调试
	fmt.Printf("GetToken Debug - Timestamp: %d, Now: %d\n", timestamp, now.Unix())

	// Initialize token - 使用AliyunRTCAppID作为AppId，AliyunAccessKey作为AppKey
	token := CreateAppToken(s.Config.AliyunRTCAppID, s.Config.AliyunAccessKey, timestamp)

	// 打印Salt值用于调试
	fmt.Printf("GetToken Debug - Salt: %d\n", token.Salt)

	// Set service
	service := CreateService(channelID, userID)
	service.AddAudioPublishPrivilege()
	service.AddVideoPublishPrivilege()
	token.SetService(&service)

	// Build token
	tokenString, err := token.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build token: %w", err)
	}

	// 打印生成的Token用于调试
	fmt.Printf("GetToken Debug - Generated Token: %s\n", tokenString)

	tokenInfo := &RTCTokenInfo{
		AppID:       s.Config.AliyunRTCAppID,
		ChannelID:   channelID,
		UserID:      userID,
		Token:       tokenString,
		ExpireTime:  expireTime,
		RequestTime: now,
	}

	return tokenInfo, nil
}

// 2 CreateVideoCall initiates a video call between a door device and a resident
func (s *RTCService) CreateVideoCall(deviceID, residentID string) (string, error) {
	// Generate a unique channel ID based on device ID and timestamp
	channelID := fmt.Sprintf("%s_%s_%d", deviceID, residentID, time.Now().Unix())

	// In a real implementation, you would create a call session in your database
	// and potentially notify the client devices to prepare for an incoming call

	return channelID, nil
}

// CreateAppToken creates a new app token
func CreateAppToken(appId string, appKey string, timestamp int32) AppToken {
	return AppToken{
		AppId:          appId,
		AppKey:         appKey,
		Salt:           utils.RandomInt32(),
		IssueTimestamp: int32(time.Now().Unix()),
		Timestamp:      timestamp,
	}
}

// CreateService creates a new service
func CreateService(channelId string, userId string) Service {
	return Service{
		ChannelId: channelId,
		UserId:    userId,
	}
}

// CreateServiceOnlyWithUserId creates a service with only user ID
func CreateServiceOnlyWithUserId(userId string) Service {
	return Service{
		ChannelId: WILDCARD_CHARACTERS,
		UserId:    userId,
	}
}

// CreateServiceOnlyWithChannelId creates a service with only channel ID
func CreateServiceOnlyWithChannelId(channelId string) Service {
	return Service{
		ChannelId: channelId,
		UserId:    WILDCARD_CHARACTERS,
	}
}

// AddAudioPublishPrivilege adds audio publish privilege to the service
func (service *Service) AddAudioPublishPrivilege() {
	if service.Privilege == nil {
		service.Privilege = new(int32)
		*service.Privilege = PRIVILEGE_ENABLED
	}
	*service.Privilege = *service.Privilege | PRIVILEGE_AUDIO_PUBLISH
}

// AddVideoPublishPrivilege adds video publish privilege to the service
func (service *Service) AddVideoPublishPrivilege() {
	if service.Privilege == nil {
		service.Privilege = new(int32)
		*service.Privilege = PRIVILEGE_ENABLED
	}
	*service.Privilege = *service.Privilege | PRIVILEGE_VIDEO_PUBLISH
}

// AddScreenPublishPrivilege adds screen publish privilege to the service
func (service *Service) AddScreenPublishPrivilege() {
	if service.Privilege == nil {
		service.Privilege = new(int32)
		*service.Privilege = PRIVILEGE_ENABLED
	}
	*service.Privilege = *service.Privilege | PRIVILEGE_SCREEN_PUBLISH
}

// Validate validates the service
func (service *Service) Validate() {
	if service.ChannelId == "" || service.UserId == "" {
		panic("illegal ChannelId or UserId")
	}
}

// Pack packs the service into bytes
func (service *Service) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)

	// channelId
	channelId := []byte(service.ChannelId)
	if err := binary.Write(buf, binary.BigEndian, int32(len(channelId))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(channelId); err != nil {
		return nil, err
	}

	// userId
	userId := []byte(service.UserId)
	if err := binary.Write(buf, binary.BigEndian, int32(len(userId))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(userId); err != nil {
		return nil, err
	}

	// hasPrivilege
	hasPrivilege := service.Privilege != nil
	if err := binary.Write(buf, binary.BigEndian, hasPrivilege); err != nil {
		return nil, err
	}
	// privilege
	if hasPrivilege {
		if err := binary.Write(buf, binary.BigEndian, *service.Privilege); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// CreateTokenOptions creates new token options
func CreateTokenOptions() TokenOptions {
	return TokenOptions{
		EngineOptions: make(map[string]string),
	}
}

// SetEngineOptions sets the engine options
func (options *TokenOptions) SetEngineOptions(engineOptions map[string]string) {
	options.EngineOptions = engineOptions
}

// Pack packs the options into bytes
func (options *TokenOptions) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)

	// hasEngineOptions
	hasEngineOptions := options.EngineOptions != nil
	if err := binary.Write(buf, binary.BigEndian, hasEngineOptions); err != nil {
		return nil, err
	}
	if hasEngineOptions {
		if err := binary.Write(buf, binary.BigEndian, int32(len(options.EngineOptions))); err != nil {
			return nil, err
		}

		if len(options.EngineOptions) > 0 {
			// sort by key
			keys := make([]string, 0, len(options.EngineOptions))
			for k := range options.EngineOptions {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, key := range keys {
				value := options.EngineOptions[key]
				if key == "" || value == "" {
					return nil, errors.New("illegal engineOptions entry")
				}
				if err := binary.Write(buf, binary.BigEndian, int32(len(key))); err != nil {
					return nil, err
				}
				if _, err := buf.Write([]byte(key)); err != nil {
					return nil, err
				}

				if err := binary.Write(buf, binary.BigEndian, int32(len(value))); err != nil {
					return nil, err
				}
				if _, err := buf.Write([]byte(value)); err != nil {
					return nil, err
				}
			}
		}
	}

	return buf.Bytes(), nil
}

// SetService sets the service for the token
func (token *AppToken) SetService(service *Service) {
	token.Service = service
}

// SetOptions sets the options for the token
func (token *AppToken) SetOptions(options *TokenOptions) {
	token.Options = options
}

// nextMultiple calculates the next multiple for buffer size
func nextMultiple(n, base int) int {
	if base <= 0 || n <= 0 {
		return 0
	}
	result := base
	for result < n {
		result *= 2
	}
	return result
}

// buildSignBody builds the sign body for the token
func (token *AppToken) buildSignBody() ([]byte, error) {
	buf := new(bytes.Buffer)
	// appId
	appId := []byte(token.AppId)
	if err := binary.Write(buf, binary.BigEndian, int32(len(appId))); err != nil {
		return nil, errors.New("illegal AppId")
	}
	if _, err := buf.Write(appId); err != nil {
		return nil, errors.New("illegal AppId")
	}

	// issueTimestamp
	if err := binary.Write(buf, binary.BigEndian, token.IssueTimestamp); err != nil {
		return nil, errors.New("illegal IssueTimestamp")
	}

	// salt
	if err := binary.Write(buf, binary.BigEndian, token.Salt); err != nil {
		return nil, errors.New("illegal Salt")
	}

	// timestamp
	if err := binary.Write(buf, binary.BigEndian, token.Timestamp); err != nil {
		return nil, errors.New("illegal Timestamp")
	}

	// service
	service, err := token.Service.Pack()
	if err != nil {
		return nil, errors.New("illegal Service")
	}
	if err := binary.Write(buf, binary.BigEndian, service); err != nil {
		return nil, errors.New("illegal Service")
	}

	// options
	if token.Options == nil {
		token.Options = &TokenOptions{
			EngineOptions: make(map[string]string),
		}
	}
	options, err := token.Options.Pack()
	if err != nil {
		return nil, errors.New("illegal TokenOptions")
	}
	if err := binary.Write(buf, binary.BigEndian, options); err != nil {
		return nil, errors.New("illegal TokenOptions")
	}

	len := nextMultiple(buf.Len(), BUFFER_CAPACITY_BASE)
	result := make([]byte, len)

	copy(result, buf.Bytes())
	return result, nil
}

// Build builds the token string
func (token *AppToken) Build() (string, error) {
	if token.AppKey == "" {
		return "", errors.New("illegal AppKey")
	}
	if token.Service == nil {
		return "", errors.New("illegal Service")
	}
	token.Service.Validate()

	generatedSign, err := utils.GenerateSign(token.AppKey, token.IssueTimestamp, token.Salt)

	if err != nil {
		return "", errors.New("generate sign failed")
	}

	buf, err := token.buildSignBody()
	if buf == nil || err != nil {
		return "", errors.New("build sign body failed")
	}

	// sign
	sign, err := utils.Sign(generatedSign, buf)

	if err != nil {
		return "", errors.New("sign failed")
	}

	tokenBuf := new(bytes.Buffer)
	// signLength
	if err := binary.Write(tokenBuf, binary.BigEndian, int32(len(sign))); err != nil {
		return "", errors.New("illegal sign")
	}
	// signBody
	if err := binary.Write(tokenBuf, binary.BigEndian, sign); err != nil {
		return "", errors.New("illegal sign")
	}
	// buf
	if err := binary.Write(tokenBuf, binary.BigEndian, buf); err != nil {
		return "", errors.New("illegal buf")
	}

	tokenCompress, err := utils.Compress(tokenBuf.Bytes())
	if err != nil {
		return "", errors.New("token compress failed")
	}

	return VERSION_0 + base64.StdEncoding.EncodeToString(tokenCompress), nil
}

// UnpackService unpacks service from bytes
func UnpackService(buf io.Reader) (*Service, error) {
	service := Service{}

	// channelId
	var channelIdLength int32
	if err := binary.Read(buf, binary.BigEndian, &channelIdLength); err != nil {
		return nil, err
	}
	channelId := make([]byte, channelIdLength)
	if _, err := io.ReadFull(buf, channelId); err != nil {
		return nil, err
	}
	service.ChannelId = string(channelId)

	// userId
	var userIdLength int32
	if err := binary.Read(buf, binary.BigEndian, &userIdLength); err != nil {
		return nil, err
	}
	userId := make([]byte, userIdLength)
	if _, err := io.ReadFull(buf, userId); err != nil {
		return nil, err
	}
	service.UserId = string(userId)

	// privilege
	var hasPrivilege bool
	if err := binary.Read(buf, binary.BigEndian, &hasPrivilege); err != nil {
		return nil, err
	}
	if hasPrivilege {
		var privilege int32
		if err := binary.Read(buf, binary.BigEndian, &privilege); err != nil {
			return nil, err
		}
		service.Privilege = &privilege
	}

	return &service, nil
}

// UnpackTokenOptions unpacks options from bytes
func UnpackTokenOptions(buf io.Reader) (*TokenOptions, error) {
	options := &TokenOptions{
		EngineOptions: make(map[string]string),
	}
	// hasEngineOptions
	var hasEngineOptions bool
	if err := binary.Read(buf, binary.BigEndian, &hasEngineOptions); err != nil {
		return nil, err
	}

	if hasEngineOptions {
		var size int32
		if err := binary.Read(buf, binary.BigEndian, &size); err != nil {
			return nil, err
		}

		for i := int32(0); i < size; i++ {
			var keyLength int32
			if err := binary.Read(buf, binary.BigEndian, &keyLength); err != nil {
				return nil, err
			}
			key := make([]byte, keyLength)
			if _, err := buf.Read(key); err != nil {
				return nil, err
			}

			var valueLength int32
			if err := binary.Read(buf, binary.BigEndian, &valueLength); err != nil {
				return nil, err
			}
			value := make([]byte, valueLength)
			if _, err := buf.Read(value); err != nil {
				return nil, err
			}

			options.EngineOptions[string(key)] = string(value)
		}
	}

	return options, nil
}

// 3 ParseToken parses a token string back into an AppToken
func (s *RTCService) ParseToken(tokenStr string) (AppToken, error) {
	appToken := AppToken{}
	if len(tokenStr) <= VENSION_LENGTH || tokenStr[0:VENSION_LENGTH] != VERSION_0 {
		return appToken, errors.New("illegal appToken length")
	}

	tokenOri := tokenStr[VENSION_LENGTH:]
	token, err := base64.StdEncoding.DecodeString(tokenOri)
	if err != nil {
		return appToken, errors.New("base64.decode appToken failed")
	}
	tokenDecompress, err := utils.Decompress(token)
	if err != nil {
		return appToken, errors.New("token decompress failed")
	}

	// sign
	tokenBuf := bytes.NewReader(tokenDecompress)

	// signLegth
	var signLegth int32
	if err := binary.Read(tokenBuf, binary.BigEndian, &signLegth); err != nil {
		return appToken, err
	}
	// signBody
	signature := make([]byte, signLegth)
	if _, err := io.ReadFull(tokenBuf, signature); err != nil {
		return appToken, errors.New("parse sign failed")
	}
	appToken.Signature = signature

	// appId
	var appIdLength int32
	if err := binary.Read(tokenBuf, binary.BigEndian, &appIdLength); err != nil {
		return appToken, err
	}
	appId := make([]byte, appIdLength)
	if _, err := io.ReadFull(tokenBuf, appId); err != nil {
		return appToken, errors.New("parse appId failed")
	}
	appToken.AppId = string(appId)

	// issueTimestamp
	var issueTimestamp int32
	if err := binary.Read(tokenBuf, binary.BigEndian, &issueTimestamp); err != nil {
		return appToken, errors.New("parse issueTimestamp failed")
	}
	appToken.IssueTimestamp = issueTimestamp

	// salt
	var salt int32
	if err := binary.Read(tokenBuf, binary.BigEndian, &salt); err != nil {
		return appToken, errors.New("parse salt failed")
	}
	appToken.Salt = salt

	// timestamp
	var timestamp int32
	if err := binary.Read(tokenBuf, binary.BigEndian, &timestamp); err != nil {
		return appToken, errors.New("parse timestamp failed")
	}
	appToken.Timestamp = timestamp

	// service
	service, err := UnpackService(tokenBuf)
	if err != nil {
		return appToken, errors.New("parse service failed")
	}
	appToken.Service = service

	// options
	options, err := UnpackTokenOptions(tokenBuf)
	if err != nil {
		return appToken, errors.New("parse tokenOptions failed")
	}
	appToken.Options = options

	return appToken, nil
}

// When implementing the actual Aliyun RTC integration, you would need to:
// 1. Import the official Aliyun SDK for Go
// 2. Implement proper token generation using the SDK
// 3. Add error handling and retry logic
// 4. Add detailed logging for debugging purposes
