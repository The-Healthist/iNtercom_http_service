package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"intercom_http_service/internal/config"
	"intercom_http_service/internal/model"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InterfaceMQTTCallService 定义MQTT通话服务接口
type InterfaceMQTTCallService interface {
	Connect() error
	Disconnect()
	InitiateCall(deviceID, residentID string) (string, error)
	InitiateCallToAll(deviceID string) (string, []string, error)
	InitiateCallToHousehold(deviceID string, householdNumber string) (string, []string, error)
	InitiateCallByPhone(deviceID string, phone string) (string, []string, error)
	HandleCallerAction(callID, action, reason string) error
	HandleCalleeAction(callID, action, reason, residentID string) error
	GetCallSession(callID string) (*model.CallSession, bool)
	EndCallSession(callID, reason string) error
	CleanupTimedOutSessions() int
	SubscribeToTopics() error
	PublishDeviceStatus(deviceID string, status map[string]interface{}) error
	PublishSystemMessage(messageType string, message map[string]interface{}) error
}

// MQTTCallService 整合MQTT和通话服务的实现
type MQTTCallService struct {
	DB              *gorm.DB
	Config          *config.Config
	RTCService      InterfaceTencentRTCService
	Client          mqtt.Client
	IsConnected     bool
	connectedMutex  sync.RWMutex // 保护IsConnected字段的读写
	connectMutex    sync.Mutex   // 保护连接操作，避免并发连接
	CallManager     *model.CallManager
	TopicHandlers   map[string]mqtt.MessageHandler
	CallRecordMutex sync.Mutex   // 用于保护通话记录创建
	ProcessedMsgs   *sync.Map    // 用于记录已处理的消息，防止重复处理
	PublishMutex    sync.Mutex   // 用于保护MQTT消息发布
	SessionMutex    sync.RWMutex // 用于保护会话操作
	CallChannels    *sync.Map    // 用于存储每个通话的控制通道
}

// 主题常量
const (
	// 来电通知主题
	TopicIncoming = "mqtt_call/incoming"

	// 设备控制主题
	TopicDeviceController = "mqtt_call/controller/device"

	// 住户控制主题
	TopicResidentController = "mqtt_call/controller/resident"

	// 系统消息主题
	TopicSystemMessage = "mqtt_call/system"
)

// 消息结构体定义
type (
	// MQTTMessage MQTT消息基础结构
	MQTTMessage struct {
		Type      string         `json:"type"`
		Timestamp int64          `json:"timestamp"`
		Payload   map[string]any `json:"payload"`
	}

	// IncomingCallMessage 来电通知消息
	IncomingCallMessage struct {
		CallID           string   `json:"call_id"`
		DeviceDeviceID   string   `json:"device_device_id"`
		TargetResidentID string   `json:"target_resident_id"`
		Timestamp        int64    `json:"timestamp"`
		TencentRTC       TRTCInfo `json:"tencen_rtc"`
	}

	// ControlMessage 控制消息
	ControlMessage struct {
		MessageType    string `json:"message_type,omitempty"`
		Action         string `json:"action"`
		CallID         string `json:"call_id"`
		Timestamp      int64  `json:"timestamp"`
		Reason         string `json:"reason,omitempty"`
		DeviceDeviceID string `json:"device_device_id,omitempty"`
		ResidentID     string `json:"resident_id,omitempty"`
		RoomID         string `json:"room_id,omitempty"`
	}

	// CallRequest 呼叫请求结构
	CallRequest struct {
		DeviceID        string `json:"device_id"`        // 呼叫方设备ID
		HouseholdNumber string `json:"household_number"` // 目标户号
		Timestamp       int64  `json:"timestamp"`        // 发起呼叫的Unix毫秒时间戳
	}

	// CallResponse 呼叫响应结构
	CallResponse struct {
		CallID            string         `json:"call_id"`             // 本次呼叫的唯一ID
		DeviceDeviceID    string         `json:"device_device_id"`    // 呼叫方设备ID
		TargetResidentIDs []string       `json:"target_resident_ids"` // 目标住户ID列表
		Timestamp         int64          `json:"timestamp"`
		TencentRTC        TRTCInfo       `json:"tencen_rtc"` // 腾讯云TRTC信息
		CallInfo          ControlMessage `json:"call_info"`  // 通话信息
	}

	// TRTCInfo 腾讯云RTC信息
	TRTCInfo struct {
		RoomIDType string `json:"room_id_type"`
		RoomID     string `json:"room_id"`
		SDKAppID   int    `json:"sdk_app_id"`
		UserID     string `json:"user_id"`
		UserSig    string `json:"user_sig"`
	}

	// SystemMessage 系统消息
	SystemMessage struct {
		Type      string      `json:"type"`
		Level     string      `json:"level"` // info/warning/error
		Message   string      `json:"message"`
		Data      interface{} `json:"data,omitempty"`
		Timestamp int64       `json:"timestamp"`
	}
)

// buildControlPayload 构建兼容控制消息载荷。
// 顶层字段保持兼容，同时补充 call_info 便于客户端统一解析。
func buildControlPayload(controlMsg ControlMessage) map[string]interface{} {
	messageType := controlMsg.MessageType
	if messageType == "" {
		messageType = "call_control"
	}

	payload := map[string]interface{}{
		"message_type": messageType,
		"action":       controlMsg.Action,
		"call_id":      controlMsg.CallID,
		"timestamp":    controlMsg.Timestamp,
		"call_info":    controlMsg,
	}

	if controlMsg.Reason != "" {
		payload["reason"] = controlMsg.Reason
	}

	if controlMsg.DeviceDeviceID != "" {
		payload["device_device_id"] = controlMsg.DeviceDeviceID
	}

	if controlMsg.ResidentID != "" {
		payload["resident_id"] = controlMsg.ResidentID
	}

	if controlMsg.RoomID != "" {
		payload["room_id"] = controlMsg.RoomID
	}

	return payload
}

func buildControlMessage(callID, action, reason string, timestamp int64, session *model.CallSession, residentID string) ControlMessage {
	controlMsg := ControlMessage{
		MessageType: "call_control",
		Action:      action,
		CallID:      callID,
		Timestamp:   timestamp,
		Reason:      reason,
	}

	if session != nil {
		controlMsg.DeviceDeviceID = session.DeviceID
		controlMsg.RoomID = session.TRTCInfo.RoomID

		if residentID == "" {
			residentID = session.ResidentID
		}
	}

	if residentID != "" {
		controlMsg.ResidentID = residentID
	}

	return controlMsg
}

func (s *MQTTCallService) mustGetSession(callID string) *model.CallSession {
	session, _ := s.CallManager.GetSession(callID)
	return session
}

// 通话控制信号类型
type CallControlSignal int

const (
	SignalRinging CallControlSignal = iota
	SignalAnswered
	SignalRejected
	SignalHangup
	SignalError
)

// 通话控制消息
type CallControlMessage struct {
	Signal   CallControlSignal
	CallID   string
	DeviceID string
	UserID   string
	Action   string
	Reason   string
	Data     interface{}
}

// NewMQTTCallService 创建一个新的MQTT通话服务实现
func NewMQTTCallService(db *gorm.DB, cfg *config.Config, rtcService InterfaceTencentRTCService) InterfaceMQTTCallService {
	service := &MQTTCallService{
		DB:            db,
		Config:        cfg,
		RTCService:    rtcService,
		CallManager:   model.NewCallManager(),
		TopicHandlers: make(map[string]mqtt.MessageHandler),
		IsConnected:   false,
		ProcessedMsgs: &sync.Map{},
		CallChannels:  &sync.Map{},
	}

	// 设置MQTT客户端
	service.setupMQTTClient()

	// 设置主题处理程序
	service.setupTopicHandlers()

	// 主动连接 MQTT broker
	if err := service.Connect(); err != nil {
		log.Printf("[MQTT] 初始化时连接 broker 失败: %v，将在首次发布时重试", err)
	}

	// 启动会话清理定时任务
	go service.startSessionCleanupTask()

	// 启动消息去重清理任务
	go service.startMsgCleanupTask()

	return service
}

// setupMQTTClient 设置MQTT客户端
func (s *MQTTCallService) setupMQTTClient() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(s.Config.MQTTBrokerURL)
	// 使用唯一的客户端ID，避免同一服务多实例冲突
	opts.SetClientID(fmt.Sprintf("%s-%s-%d", s.Config.MQTTClientID, uuid.New().String()[:8], time.Now().UnixNano()))
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(time.Second * 30)
	opts.SetKeepAlive(time.Second * 60)
	opts.SetPingTimeout(time.Second * 10)
	opts.SetCleanSession(true)
	opts.SetOrderMatters(true)

	// 设置QoS等级为1，确保消息至少传递一次
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("[MQTT] 收到未处理的消息: topic=%s", msg.Topic())
	})

	// 添加用户名和密码
	if s.Config.MQTTUsername != "" {
		opts.SetUsername(s.Config.MQTTUsername)
		opts.SetPassword(s.Config.MQTTPassword)
	}

	// 添加TLS配置，支持SSL连接
	if strings.HasPrefix(s.Config.MQTTBrokerURL, "ssl://") || strings.HasPrefix(s.Config.MQTTBrokerURL, "tls://") || s.Config.MQTTSSLEnabled {
		log.Println("[MQTT] 使用TLS连接")
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // 默认跳过验证，如有CA证书则使用
		}

		// 如果提供了CA证书路径，则加载证书
		if s.Config.MQTTCACertPath != "" {
			log.Printf("[MQTT] 使用CA证书: %s", s.Config.MQTTCACertPath)
			certpool := x509.NewCertPool()
			pemCerts, err := os.ReadFile(s.Config.MQTTCACertPath)
			if err == nil {
				certpool.AppendCertsFromPEM(pemCerts)
				tlsConfig.RootCAs = certpool
				tlsConfig.InsecureSkipVerify = false
				log.Println("[MQTT] CA证书加载成功")
			} else {
				log.Printf("[MQTT] 加载CA证书失败: %v，将使用InsecureSkipVerify", err)
			}
		}

		opts.SetTLSConfig(tlsConfig)
	}

	// 设置连接丢失回调
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("[MQTT] 连接丢失: %v", err)
		s.connectedMutex.Lock()
		s.IsConnected = false
		s.connectedMutex.Unlock()
	})

	// 设置连接建立回调
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("[MQTT] 成功连接到", s.Config.MQTTBrokerURL)
		s.connectedMutex.Lock()
		s.IsConnected = true
		s.connectedMutex.Unlock()

		// 订阅主题
		if err := s.SubscribeToTopics(); err != nil {
			log.Printf("[MQTT] 订阅主题失败: %v", err)
		}
	})

	// 设置重连回调
	opts.SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
		log.Println("[MQTT] 正在尝试重连...")
	})

	// 创建客户端
	s.Client = mqtt.NewClient(opts)
}

// setupTopicHandlers 设置主题处理程序
func (s *MQTTCallService) setupTopicHandlers() {
	s.TopicHandlers = map[string]mqtt.MessageHandler{
		TopicDeviceController:   s.handleDeviceControl,
		TopicResidentController: s.handleResidentControl,
		TopicSystemMessage:      s.handleSystemMessage,
	}
}

// Connect 连接到MQTT服务器，带有重试机制
func (s *MQTTCallService) Connect() error {
	log.Printf("[MQTT] 正在连接到 %s...", s.Config.MQTTBrokerURL)

	// 使用独立的连接锁，避免与 publishMessage 的 PublishMutex 死锁
	s.connectMutex.Lock()
	defer s.connectMutex.Unlock()

	// 如果已连接，直接返回
	s.connectedMutex.RLock()
	isConnected := s.IsConnected && s.Client.IsConnected()
	s.connectedMutex.RUnlock()

	if isConnected {
		return nil
	}

	// 添加最大重试次数和指数退避策略
	maxRetries := 5
	var err error

	for i := 0; i < maxRetries; i++ {
		token := s.Client.Connect()
		if token.WaitTimeout(5*time.Second) && token.Error() == nil {
			s.connectedMutex.Lock()
			s.IsConnected = true
			s.connectedMutex.Unlock()
			log.Printf("[MQTT] 成功连接到 %s", s.Config.MQTTBrokerURL)
			return nil
		}

		err = token.Error()
		backoffTime := time.Duration(1<<uint(i)) * time.Second // 指数退避: 1s, 2s, 4s, 8s, 16s
		log.Printf("[MQTT] 连接尝试 %d/%d 失败: %v, 将在 %v 后重试", i+1, maxRetries, err, backoffTime)
		time.Sleep(backoffTime)
	}

	return fmt.Errorf("[MQTT] 连接失败，已尝试 %d 次: %v", maxRetries, err)
}

// Disconnect 断开与MQTT服务器的连接
func (s *MQTTCallService) Disconnect() {
	if s.Client != nil && s.Client.IsConnected() {
		s.Client.Disconnect(250)
	}
}

// SubscribeToTopics 订阅相关主题
func (s *MQTTCallService) SubscribeToTopics() error {
	// 使用QoS 1确保消息至少被传递一次
	qos := byte(1)

	for topic, handler := range s.TopicHandlers {
		if token := s.Client.Subscribe(topic, qos, handler); token.Wait() && token.Error() != nil {
			return fmt.Errorf("订阅主题失败 [%s]: %v", topic, token.Error())
		}
		log.Printf("[MQTT] 已订阅主题: %s", topic)
	}
	return nil
}

// InitiateCall 发起通话
func (s *MQTTCallService) InitiateCall(deviceID, residentID string) (string, error) {
	// 使用互斥锁保护整个通话创建过程
	s.SessionMutex.Lock()
	defer s.SessionMutex.Unlock()

	// 检查是否已存在相同设备和住户的通话会话
	existingCallID := s.findActiveCallByParticipants(deviceID, residentID)
	if existingCallID != "" {
		log.Printf("[MQTT] 已存在设备 %s 和住户 %s 之间的通话: %s", deviceID, residentID, existingCallID)
		return existingCallID, nil
	}

	// 生成唯一的通话ID
	callID := uuid.New().String()
	callUnix := time.Now().Unix()
	roomID := BuildSharedTRTCRoomID(deviceID, residentID, callUnix)

	// 创建通话控制通道
	controlChan := make(chan CallControlMessage, 10) // 缓冲区大小10
	s.CallChannels.Store(callID, controlChan)

	// 启动独立的通话控制goroutine
	go s.handleCallSession(callID, deviceID, residentID, controlChan)

	// 为住户生成UserSig
	tokenInfo, err := s.RTCService.GetUserSig(residentID)
	if err != nil {
		// 发送错误信号并关闭通道
		controlChan <- CallControlMessage{Signal: SignalError, Reason: err.Error()}
		s.CallChannels.Delete(callID)
		return "", fmt.Errorf("生成UserSig失败: %v", err)
	}

	// 创建TRTC信息
	trtcInfo := model.TRTCInfo{
		RoomID:     roomID,
		RoomIDType: "string",
		SDKAppID:   tokenInfo.SDKAppID,
		UserID:     tokenInfo.UserID,
		UserSig:    tokenInfo.UserSig,
	}

	// 创建通话会话
	_, err = s.CallManager.CreateSession(callID, deviceID, residentID, trtcInfo)
	if err != nil {
		// 发送错误信号并关闭通道
		controlChan <- CallControlMessage{Signal: SignalError, Reason: err.Error()}
		s.CallChannels.Delete(callID)
		return "", fmt.Errorf("创建通话会话失败: %v", err)
	}

	// 发送呼入通知给住户
	incomingNotification := IncomingCallMessage{
		CallID:           callID,
		DeviceDeviceID:   deviceID,
		TargetResidentID: residentID,
		Timestamp:        time.Now().UnixMilli(),
		TencentRTC: TRTCInfo{
			RoomIDType: trtcInfo.RoomIDType,
			RoomID:     trtcInfo.RoomID,
			SDKAppID:   trtcInfo.SDKAppID,
			UserID:     trtcInfo.UserID,
			UserSig:    trtcInfo.UserSig,
		},
	}

	// 发布到住户的呼入通知主题
	if err := s.publishMessage(TopicIncoming, incomingNotification); err != nil {
		// 发送错误信号并关闭通道
		controlChan <- CallControlMessage{Signal: SignalError, Reason: err.Error()}
		s.CallManager.EndSession(callID, "发送通知失败")
		s.CallChannels.Delete(callID)
		return "", fmt.Errorf("发送呼入通知失败: %v", err)
	}

	// 更新会话状态为振铃中
	s.CallManager.UpdateSessionStatus(callID, "ringing")

	// 创建振铃控制消息
	timestamp := time.Now().UnixMilli()
	ringControl := buildControlMessage(callID, "ringing", "", timestamp, s.mustGetSession(callID), residentID)

	// 先标记此消息为已处理，防止我们自己发出的消息被重复处理
	s.markMessageProcessed(callID, "ringing", timestamp)

	// 同时发送振铃消息给设备和住户
	controlPayload := buildControlPayload(ringControl)
	if err := s.publishMessage(TopicDeviceController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送振铃控制消息给设备失败: %v", err)
	}

	if err := s.publishMessage(TopicResidentController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送振铃控制消息给住户失败: %v", err)
	}

	// 向通话控制通道发送振铃信号
	controlChan <- CallControlMessage{
		Signal:   SignalRinging,
		CallID:   callID,
		DeviceID: deviceID,
		UserID:   residentID,
	}

	// 构建呼叫响应
	callResponse := CallResponse{
		CallID:            callID,
		DeviceDeviceID:    deviceID,
		TargetResidentIDs: []string{residentID},
		Timestamp:         time.Now().UnixMilli(),
		TencentRTC: TRTCInfo{
			RoomIDType: trtcInfo.RoomIDType,
			RoomID:     trtcInfo.RoomID,
			SDKAppID:   trtcInfo.SDKAppID,
			UserID:     deviceID,
		},
		CallInfo: ringControl,
	}

	// 创建通话记录
	s.createCallRecord(callID, deviceID, residentID, "ringing")

	// 记录详细日志
	log.Printf("[MQTT] 成功发起通话，callID: %s, 设备: %s, 住户: %s, 响应: %+v",
		callID, deviceID, residentID, callResponse)

	return callID, nil
}

// handleCallSession 处理单个通话会话的控制流
func (s *MQTTCallService) handleCallSession(callID, deviceID, residentID string, controlChan chan CallControlMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MQTT] 通话控制处理panic: callID=%s, error=%v", callID, r)
		}

		// 通话结束时清理资源
		close(controlChan)
		s.CallChannels.Delete(callID)
	}()

	// 通话状态
	var status string = "ringing"
	// 超时计时器
	ringTimer := time.NewTimer(2 * time.Minute)
	defer ringTimer.Stop()

	callTimer := time.NewTimer(2 * time.Hour)
	defer callTimer.Stop()

	log.Printf("[MQTT] 开始处理通话: callID=%s, deviceID=%s, residentID=%s", callID, deviceID, residentID)

	// 创建一个合并的事件通道，避免使用for { select {} }模式
	for {
		select {
		case msg, ok := <-controlChan:
			if !ok {
				log.Printf("[MQTT] 通话控制通道已关闭: callID=%s", callID)
				return
			}

			// 处理控制消息
			switch msg.Signal {
			case SignalRinging:
				log.Printf("[MQTT] 通话振铃: callID=%s", callID)
				status = "ringing"

			case SignalAnswered:
				log.Printf("[MQTT] 通话已接听: callID=%s", callID)
				status = "connected"
				// 重置超时时间为2小时
				if !ringTimer.Stop() {
					select {
					case <-ringTimer.C:
					default:
					}
				}
				callTimer.Reset(2 * time.Hour)

			case SignalRejected:
				log.Printf("[MQTT] 通话被拒绝: callID=%s, reason=%s", callID, msg.Reason)
				return

			case SignalHangup:
				log.Printf("[MQTT] 通话挂断: callID=%s, reason=%s", callID, msg.Reason)
				return

			case SignalError:
				log.Printf("[MQTT] 通话错误: callID=%s, reason=%s", callID, msg.Reason)
				return
			}

		case <-ringTimer.C:
			// 振铃超时
			if status == "ringing" {
				log.Printf("[MQTT] 通话振铃超时: callID=%s", callID)
				s.EndCallSession(callID, "ring_timeout")
				return
			}

		case <-callTimer.C:
			// 通话超时
			log.Printf("[MQTT] 通话时间超时: callID=%s", callID)
			s.EndCallSession(callID, "call_timeout")
			return
		}
	}
}

// HandleCallerAction 处理呼叫方动作
func (s *MQTTCallService) HandleCallerAction(callID, action, reason string) error {
	// 获取会话
	session, exists := s.CallManager.GetSession(callID)
	if !exists {
		return fmt.Errorf("会话不存在: %s", callID)
	}

	// 通过通道发送控制消息
	callChannelObj, exists := s.CallChannels.Load(callID)
	if exists {
		callChannel, ok := callChannelObj.(chan CallControlMessage)
		if ok {
			// 确定信号类型
			var signal CallControlSignal
			switch action {
			case "hangup":
				signal = SignalHangup
			case "cancelled":
				signal = SignalHangup // 取消也视为挂断
			default:
				return fmt.Errorf("不支持的动作: %s", action)
			}

			// 向通道发送信号
			select {
			case callChannel <- CallControlMessage{
				Signal:   signal,
				CallID:   callID,
				Action:   action,
				Reason:   reason,
				DeviceID: "", // 可以从会话获取
			}:
				// 消息已发送
			default:
				// 通道已满或关闭，记录警告
				log.Printf("[MQTT] 无法发送控制消息到通话通道: callID=%s, action=%s", callID, action)
			}
		}
	}

	// 更新会话状态
	var newStatus string
	switch action {
	case "hangup":
		newStatus = "ended"
	case "cancelled":
		newStatus = "cancelled"
	default:
		return fmt.Errorf("不支持的动作: %s", action)
	}

	// 更新会话状态
	if err := s.CallManager.UpdateSessionStatus(callID, newStatus); err != nil {
		return err
	}

	// 创建控制消息
	timestamp := time.Now().UnixMilli()
	controlMsg := buildControlMessage(callID, action, reason, timestamp, session, "")

	// 先标记此消息为已处理，防止我们自己发出的消息被重复处理
	s.markMessageProcessed(callID, action, timestamp)

	// 同时发送控制消息给设备端和住户端，确保双方都收到消息
	controlPayload := buildControlPayload(controlMsg)

	if err := s.publishMessage(TopicDeviceController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送控制消息给设备方失败: %v", err)
	}

	if err := s.publishMessage(TopicResidentController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送控制消息给住户方失败: %v", err)
	}

	// 如果是结束通话的动作，结束会话
	if action == "hangup" || action == "cancelled" {
		if _, err := s.CallManager.EndSession(callID, reason); err != nil {
			return err
		}

		// 更新通话记录
		s.updateCallRecord(callID, "caller_"+action, reason)
	}

	return nil
}

// HandleCalleeAction 处理被呼叫方动作
func (s *MQTTCallService) HandleCalleeAction(callID, action, reason, residentID string) error {
	// 获取会话
	session, exists := s.CallManager.GetSession(callID)
	if !exists {
		return fmt.Errorf("会话不存在: %s", callID)
	}

	// 通过通道发送控制消息
	callChannelObj, exists := s.CallChannels.Load(callID)
	if exists {
		callChannel, ok := callChannelObj.(chan CallControlMessage)
		if ok {
			// 确定信号类型
			var signal CallControlSignal
			switch action {
			case "answered":
				signal = SignalAnswered
			case "rejected":
				signal = SignalRejected
			case "hangup":
				signal = SignalHangup
			case "timeout":
				signal = SignalHangup // 超时也视为挂断
			default:
				return fmt.Errorf("不支持的动作: %s", action)
			}

			// 向通道发送信号
			select {
			case callChannel <- CallControlMessage{
				Signal: signal,
				CallID: callID,
				Action: action,
				Reason: reason,
				UserID: "", // 可以从会话获取
			}:
				// 消息已发送
			default:
				// 通道已满或关闭，记录警告
				log.Printf("[MQTT] 无法发送控制消息到通话通道: callID=%s, action=%s", callID, action)
			}
		}
	}

	// 更新会话状态
	var newStatus string
	switch action {
	case "rejected":
		newStatus = "rejected"
	case "answered":
		newStatus = "connected"
	case "hangup":
		newStatus = "ended"
	case "timeout":
		newStatus = "timeout"
	default:
		return fmt.Errorf("不支持的动作: %s", action)
	}

	// 更新会话状态
	if err := s.CallManager.UpdateSessionStatus(callID, newStatus); err != nil {
		return err
	}

	// 创建控制消息
	timestamp := time.Now().UnixMilli()
	controlMsg := buildControlMessage(callID, action, reason, timestamp, session, residentID)

	// 先标记此消息为已处理，防止我们自己发出的消息被重复处理
	s.markMessageProcessed(callID, action, timestamp)

	// 同时发送控制消息给设备端和住户端，确保双方都收到消息
	controlPayload := buildControlPayload(controlMsg)

	if err := s.publishMessage(TopicDeviceController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送控制消息给设备方失败: %v", err)
	}

	if err := s.publishMessage(TopicResidentController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送控制消息给住户方失败: %v", err)
	}

	// 如果是结束通话的动作，结束会话
	if action == "rejected" || action == "hangup" || action == "timeout" {
		if _, err := s.CallManager.EndSession(callID, reason); err != nil {
			return err
		}

		// 更新通话记录
		s.updateCallRecord(callID, "callee_"+action, reason)
	}

	return nil
}

// EndCallSession 结束通话会话
func (s *MQTTCallService) EndCallSession(callID, reason string) error {
	session, _ := s.CallManager.GetSession(callID)

	// 通过通道发送结束信号
	callChannelObj, exists := s.CallChannels.Load(callID)
	if exists {
		callChannel, ok := callChannelObj.(chan CallControlMessage)
		if ok {
			// 向通道发送挂断信号
			select {
			case callChannel <- CallControlMessage{
				Signal: SignalHangup,
				CallID: callID,
				Reason: reason,
			}:
				// 消息已发送
			default:
				// 通道已满或关闭，记录警告
				log.Printf("[MQTT] 无法发送结束信号到通话通道: callID=%s", callID)
			}
		}
	}

	if _, err := s.CallManager.EndSession(callID, reason); err != nil {
		return err
	}

	// 向双方发送通话结束通知
	timestamp := time.Now().UnixMilli()
	endInfo := buildControlMessage(callID, "hangup", reason, timestamp, session, "")

	// 先标记此消息为已处理，防止我们自己发出的消息被重复处理
	s.markMessageProcessed(callID, "hangup", timestamp)

	// 同时发送结束通知给设备端和住户端
	endPayload := buildControlPayload(endInfo)

	if err := s.publishMessage(TopicDeviceController, endPayload); err != nil {
		log.Printf("[MQTT] 发送结束通知给设备方失败: %v", err)
	}

	if err := s.publishMessage(TopicResidentController, endPayload); err != nil {
		log.Printf("[MQTT] 发送结束通知给住户方失败: %v", err)
	}

	// 更新通话记录
	s.updateCallRecord(callID, "system_ended", reason)

	return nil
}

// publishMessage 发布消息到指定主题
func (s *MQTTCallService) publishMessage(topic string, payload interface{}) error {
	// 加锁保护发布过程，避免并发发布冲突
	s.PublishMutex.Lock()
	defer s.PublishMutex.Unlock()

	// 检查连接状态
	s.connectedMutex.RLock()
	isConnected := s.IsConnected && s.Client.IsConnected()
	s.connectedMutex.RUnlock()

	if !isConnected {
		log.Printf("[MQTT] 客户端未连接，尝试重新连接...")
		if err := s.Connect(); err != nil {
			return fmt.Errorf("MQTT客户端未连接: %v", err)
		}
	}

	// 序列化消息
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 发布消息，使用QoS 1确保消息至少被传递一次
	qos := byte(1)
	retained := false // 非持久消息

	// 创建发布令牌并等待完成
	token := s.Client.Publish(topic, qos, retained, jsonData)

	// 设置超时时间，避免无限等待
	if !token.WaitTimeout(3 * time.Second) {
		return fmt.Errorf("发布消息超时")
	}

	if token.Error() != nil {
		return fmt.Errorf("发布消息失败: %v", token.Error())
	}

	// 打印简化的日志，不输出完整消息内容
	payloadType := fmt.Sprintf("%T", payload)
	log.Printf("[MQTT] 已发布%s类型消息到主题: %s", payloadType, topic)
	return nil
}

// findActiveCallByParticipants 根据参与者查找活跃的通话
func (s *MQTTCallService) findActiveCallByParticipants(deviceID, residentID string) string {
	// 获取所有活跃会话
	activeSessions := s.CallManager.GetAllActiveSessions()

	// 遍历所有会话，查找匹配的活跃通话
	for _, session := range activeSessions {
		if session.DeviceID == deviceID && session.ResidentID == residentID {
			return session.CallID
		}
	}
	return ""
}

// GetAllSessions 安全地获取所有会话的只读副本
func (s *MQTTCallService) GetAllSessions() map[string]*model.CallSession {
	// 无法直接访问sessions字段，需要使用现有方法获取所有会话
	sessions := make(map[string]*model.CallSession)
	activeSessions := s.CallManager.GetAllActiveSessions()

	// 转换为map格式
	for _, session := range activeSessions {
		sessions[session.CallID] = session
	}

	return sessions
}

// GetCallSession 获取指定通话会话
func (s *MQTTCallService) GetCallSession(callID string) (*model.CallSession, bool) {
	return s.CallManager.GetSession(callID)
}

// CleanupTimedOutSessions 清理超时会话
func (s *MQTTCallService) CleanupTimedOutSessions() int {
	// 呼叫中状态超时时间: 2分钟
	ringTimeout := 2 * time.Minute
	// 通话中状态超时时间: 2小时
	callTimeout := 2 * time.Hour

	return s.CallManager.CleanupTimedOutSessions(callTimeout, ringTimeout)
}

// startSessionCleanupTask 启动会话清理定时任务
func (s *MQTTCallService) startSessionCleanupTask() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cleanedCount := s.CleanupTimedOutSessions()
		if cleanedCount > 0 {
			log.Printf("[MQTT] 清理超时会话: %d 个", cleanedCount)
		}
	}
}

// startMsgCleanupTask 启动消息去重清理定时任务
func (s *MQTTCallService) startMsgCleanupTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// 清理超过5分钟的消息记录
		now := time.Now().Unix()
		count := 0

		s.ProcessedMsgs.Range(func(key, value interface{}) bool {
			if timestamp, ok := value.(int64); ok {
				// 清理超过5分钟的消息
				if now-timestamp > 300 {
					s.ProcessedMsgs.Delete(key)
					count++
				}
			}
			return true
		})

		if count > 0 {
			log.Printf("[MQTT] 清理了 %d 条历史消息记录", count)
		}
	}
}

// 生成消息唯一标识
func generateMsgKey(callID, action string, timestamp int64) string {
	return fmt.Sprintf("%s:%s:%d", callID, action, timestamp)
}

// 判断消息是否已处理
func (s *MQTTCallService) isMessageProcessed(callID, action string, timestamp int64) bool {
	key := generateMsgKey(callID, action, timestamp)
	_, exists := s.ProcessedMsgs.Load(key)
	return exists
}

// 标记消息为已处理
func (s *MQTTCallService) markMessageProcessed(callID, action string, timestamp int64) {
	key := generateMsgKey(callID, action, timestamp)
	s.ProcessedMsgs.Store(key, time.Now().Unix())
}

// handleDeviceControl 处理设备控制消息
func (s *MQTTCallService) handleDeviceControl(_ mqtt.Client, msg mqtt.Message) {
	// 使用defer和recover防止处理程序panic导致整个服务崩溃
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MQTT] 处理设备控制消息发生panic: %v", r)
		}
	}()

	var controlMsg ControlMessage
	if err := json.Unmarshal(msg.Payload(), &controlMsg); err != nil {
		log.Printf("[MQTT] 解析设备控制消息失败: %v", err)
		return
	}

	// 跳过处理我们自己发出的"ringing"消息
	if controlMsg.Action == "ringing" {
		log.Printf("[MQTT] 收到ringing消息，正常状态更新")
		return
	}

	// 消息去重，避免重复处理相同的消息
	if s.isMessageProcessed(controlMsg.CallID, controlMsg.Action, controlMsg.Timestamp) {
		log.Printf("[MQTT] 跳过重复处理的设备控制消息: %s, callID=%s, timestamp=%d",
			controlMsg.Action, controlMsg.CallID, controlMsg.Timestamp)
		return
	}

	// 标记消息为已处理
	s.markMessageProcessed(controlMsg.CallID, controlMsg.Action, controlMsg.Timestamp)

	// 处理控制消息
	if err := s.HandleCallerAction(controlMsg.CallID, controlMsg.Action, controlMsg.Reason); err != nil {
		log.Printf("[MQTT] 处理设备控制消息失败: %v", err)
	}
}

// handleResidentControl 处理住户控制消息
func (s *MQTTCallService) handleResidentControl(_ mqtt.Client, msg mqtt.Message) {
	// 使用defer和recover防止处理程序panic导致整个服务崩溃
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MQTT] 处理住户控制消息发生panic: %v", r)
		}
	}()

	var controlMsg ControlMessage
	if err := json.Unmarshal(msg.Payload(), &controlMsg); err != nil {
		log.Printf("[MQTT] 解析住户控制消息失败: %v", err)
		return
	}

	// 跳过处理我们自己发出的"ringing"消息
	if controlMsg.Action == "ringing" {
		log.Printf("[MQTT] 收到住户ringing消息，正常状态更新")
		return
	}

	// 消息去重，避免重复处理相同的消息
	if s.isMessageProcessed(controlMsg.CallID, controlMsg.Action, controlMsg.Timestamp) {
		log.Printf("[MQTT] 跳过重复处理的住户控制消息: %s, callID=%s, timestamp=%d",
			controlMsg.Action, controlMsg.CallID, controlMsg.Timestamp)
		return
	}

	// 标记消息为已处理
	s.markMessageProcessed(controlMsg.CallID, controlMsg.Action, controlMsg.Timestamp)

	// 处理控制消息
	if err := s.HandleCalleeAction(controlMsg.CallID, controlMsg.Action, controlMsg.Reason, controlMsg.ResidentID); err != nil {
		log.Printf("[MQTT] 处理住户控制消息失败: %v", err)
	}
}

// handleSystemMessage 处理系统消息
func (s *MQTTCallService) handleSystemMessage(_ mqtt.Client, msg mqtt.Message) {
	var systemMsg SystemMessage
	if err := json.Unmarshal(msg.Payload(), &systemMsg); err != nil {
		log.Printf("[MQTT] 解析系统消息失败: %v", err)
		return
	}

	// 记录系统消息
	log.Printf("[MQTT] 收到系统消息: 类型=%s, 级别=%s, 消息=%s",
		systemMsg.Type, systemMsg.Level, systemMsg.Message)
}

// InitiateCallToAll 向设备关联的户号下的所有居民发起通话
func (s *MQTTCallService) InitiateCallToAll(deviceID string) (string, []string, error) {
	// 查询设备信息及其关联的户号
	var device model.Device
	if err := s.DB.Preload("Household.Residents").First(&device, deviceID).Error; err != nil {
		return "", nil, fmt.Errorf("查询设备失败: %v", err)
	}

	// 如果设备没有关联户号，返回错误
	if device.Household == nil || device.HouseholdID == 0 {
		return "", nil, fmt.Errorf("设备未关联户号")
	}

	// 如果户号没有关联居民，返回错误
	if len(device.Household.Residents) == 0 {
		return "", nil, fmt.Errorf("户号未关联任何居民")
	}

	// 生成唯一的通话ID
	callID := uuid.New().String()
	callUnix := time.Now().Unix()
	primaryResidentID := fmt.Sprintf("%d", device.Household.Residents[0].ID)
	roomID := BuildSharedTRTCRoomID(deviceID, primaryResidentID, callUnix)

	// 收集所有居民ID
	residentIDs := make([]string, 0, len(device.Household.Residents))

	// 多住户呼叫共享同一个会话和房间号，避免相同callID重复建会话。
	sessionTRTCInfo := model.TRTCInfo{
		RoomID:     roomID,
		RoomIDType: "string",
		SDKAppID:   s.Config.TencentSDKAppID,
	}
	if _, err := s.CallManager.CreateSession(callID, deviceID, "", sessionTRTCInfo); err != nil {
		return "", nil, fmt.Errorf("创建通话会话失败: %v", err)
	}

	// 向每个居民发送呼叫通知
	for _, resident := range device.Household.Residents {
		residentID := fmt.Sprintf("%d", resident.ID)

		// 为住户生成UserSig
		tokenInfo, err := s.RTCService.GetUserSig(residentID)
		if err != nil {
			log.Printf("[MQTT] 为居民 %s 生成UserSig失败: %v", residentID, err)
			continue
		}

		// 创建TRTC信息
		trtcInfo := model.TRTCInfo{
			RoomID:     roomID,
			RoomIDType: "string",
			SDKAppID:   tokenInfo.SDKAppID,
			UserID:     tokenInfo.UserID,
			UserSig:    tokenInfo.UserSig,
		}

		// 发送呼入通知给住户
		incomingNotification := IncomingCallMessage{
			CallID:           callID,
			DeviceDeviceID:   deviceID,
			TargetResidentID: residentID,
			Timestamp:        time.Now().UnixMilli(),
			TencentRTC: TRTCInfo{
				RoomIDType: trtcInfo.RoomIDType,
				RoomID:     trtcInfo.RoomID,
				SDKAppID:   trtcInfo.SDKAppID,
				UserID:     trtcInfo.UserID,
				UserSig:    trtcInfo.UserSig,
			},
		}

		// 发布到住户的呼入通知主题
		if err := s.publishMessage(TopicIncoming, incomingNotification); err != nil {
			log.Printf("[MQTT] 发送呼入通知给居民 %s 失败: %v", residentID, err)
			continue
		}

		residentIDs = append(residentIDs, residentID)

		// 创建通话记录
		s.createCallRecord(callID, deviceID, residentID, "ringing")
	}

	if len(residentIDs) == 0 {
		if _, err := s.CallManager.EndSession(callID, "no_available_resident"); err != nil {
			log.Printf("[MQTT] 清理空通话会话失败: %v", err)
		}
		return "", nil, fmt.Errorf("没有成功向任何居民发起呼叫")
	}

	// 更新会话状态为振铃中
	s.CallManager.UpdateSessionStatus(callID, "ringing")

	// 创建振铃控制消息
	timestamp := time.Now().UnixMilli()
	ringControl := buildControlMessage(callID, "ringing", "", timestamp, s.mustGetSession(callID), "")

	// 先标记此消息为已处理，防止我们自己发出的消息被重复处理
	s.markMessageProcessed(callID, "ringing", timestamp)

	// 同时发送振铃消息给设备和住户
	controlPayload := buildControlPayload(ringControl)

	if err := s.publishMessage(TopicDeviceController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送振铃控制消息给设备失败: %v", err)
	}

	if err := s.publishMessage(TopicResidentController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送振铃控制消息给住户失败: %v", err)
	}

	// 构建呼叫响应
	callResponse := CallResponse{
		CallID:            callID,
		DeviceDeviceID:    deviceID,
		TargetResidentIDs: residentIDs,
		Timestamp:         time.Now().UnixMilli(),
		TencentRTC: TRTCInfo{
			RoomIDType: "string",
			RoomID:     roomID,
			SDKAppID:   s.Config.TencentSDKAppID,
			UserID:     deviceID,
		},
		CallInfo: ringControl,
	}

	// 记录详细的呼叫信息
	log.Printf("[MQTT] 成功发起通话，callID: %s, 设备: %s, 目标居民: %v, 响应: %+v",
		callID, deviceID, residentIDs, callResponse)

	return callID, residentIDs, nil
}

// InitiateCallToHousehold 向指定户号下的所有居民发起通话
func (s *MQTTCallService) InitiateCallToHousehold(deviceID string, householdNumber string) (string, []string, error) {
	log.Printf("[MQTT] 向户号 %s 发起通话，设备ID: %s", householdNumber, deviceID)

	// 查询户号
	var household model.Household
	if err := s.DB.Where("household_number = ?", householdNumber).First(&household).Error; err != nil {
		return "", nil, fmt.Errorf("查询户号失败: %v", err)
	}

	if household.ID == 0 {
		return "", nil, fmt.Errorf("户号不存在: %s", householdNumber)
	}

	// 查询该户号下的所有居民
	var residents []model.Resident
	if err := s.DB.Where("household_id = ?", household.ID).Find(&residents).Error; err != nil {
		return "", nil, fmt.Errorf("查询户号下的居民失败: %v", err)
	}

	// 如果没有关联的居民，返回错误
	if len(residents) == 0 {
		return "", nil, fmt.Errorf("户号未关联任何居民")
	}

	log.Printf("[MQTT] 户号 %s 下有 %d 个居民", householdNumber, len(residents))

	// 生成唯一的通话ID
	callID := uuid.New().String()
	callUnix := time.Now().Unix()
	primaryResidentID := fmt.Sprintf("%d", residents[0].ID)
	roomID := BuildSharedTRTCRoomID(deviceID, primaryResidentID, callUnix)

	// 收集所有居民ID
	residentIDs := make([]string, 0, len(residents))

	// 多住户呼叫共享同一个会话和房间号，避免相同callID重复建会话。
	sessionTRTCInfo := model.TRTCInfo{
		RoomID:     roomID,
		RoomIDType: "string",
		SDKAppID:   s.Config.TencentSDKAppID,
	}
	if _, err := s.CallManager.CreateSession(callID, deviceID, "", sessionTRTCInfo); err != nil {
		return "", nil, fmt.Errorf("创建通话会话失败: %v", err)
	}

	// 向每个居民发送呼叫通知
	for _, resident := range residents {
		residentID := fmt.Sprintf("%d", resident.ID)

		// 为住户生成UserSig
		tokenInfo, err := s.RTCService.GetUserSig(residentID)
		if err != nil {
			log.Printf("[MQTT] 为居民 %s 生成UserSig失败: %v", residentID, err)
			continue
		}

		// 创建TRTC信息
		trtcInfo := model.TRTCInfo{
			RoomID:     roomID,
			RoomIDType: "string",
			SDKAppID:   tokenInfo.SDKAppID,
			UserID:     tokenInfo.UserID,
			UserSig:    tokenInfo.UserSig,
		}

		// 发送呼入通知给住户
		incomingNotification := IncomingCallMessage{
			CallID:           callID,
			DeviceDeviceID:   deviceID,
			TargetResidentID: residentID,
			Timestamp:        time.Now().UnixMilli(),
			TencentRTC: TRTCInfo{
				RoomIDType: trtcInfo.RoomIDType,
				RoomID:     trtcInfo.RoomID,
				SDKAppID:   trtcInfo.SDKAppID,
				UserID:     trtcInfo.UserID,
				UserSig:    trtcInfo.UserSig,
			},
		}

		// 发布到住户的呼入通知主题
		if err := s.publishMessage(TopicIncoming, incomingNotification); err != nil {
			log.Printf("[MQTT] 发送呼入通知给居民 %s 失败: %v", residentID, err)
			continue
		}

		residentIDs = append(residentIDs, residentID)

		// 创建通话记录
		s.createCallRecord(callID, deviceID, residentID, "ringing")
	}

	if len(residentIDs) == 0 {
		if _, err := s.CallManager.EndSession(callID, "no_available_resident"); err != nil {
			log.Printf("[MQTT] 清理空通话会话失败: %v", err)
		}
		return "", nil, fmt.Errorf("没有成功向任何居民发起呼叫")
	}

	// 更新会话状态为振铃中
	s.CallManager.UpdateSessionStatus(callID, "ringing")

	// 创建振铃控制消息
	timestamp := time.Now().UnixMilli()
	ringControl := buildControlMessage(callID, "ringing", "", timestamp, s.mustGetSession(callID), "")

	// 先标记此消息为已处理，防止我们自己发出的消息被重复处理
	s.markMessageProcessed(callID, "ringing", timestamp)

	// 同时发送振铃消息给设备和住户
	controlPayload := buildControlPayload(ringControl)

	if err := s.publishMessage(TopicDeviceController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送振铃控制消息给设备失败: %v", err)
	}

	if err := s.publishMessage(TopicResidentController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送振铃控制消息给住户失败: %v", err)
	}

	// 构建呼叫响应
	callResponse := CallResponse{
		CallID:            callID,
		DeviceDeviceID:    deviceID,
		TargetResidentIDs: residentIDs,
		Timestamp:         time.Now().UnixMilli(),
		TencentRTC: TRTCInfo{
			RoomIDType: "string",
			RoomID:     roomID,
			SDKAppID:   s.Config.TencentSDKAppID,
			UserID:     deviceID,
		},
		CallInfo: ringControl,
	}

	// 记录详细的呼叫信息
	log.Printf("[MQTT] 成功发起通话，callID: %s, 目标居民: %v, 响应: %+v",
		callID, residentIDs, callResponse)

	return callID, residentIDs, nil
}

// InitiateCallByPhone 通过住户电话发起通话
func (s *MQTTCallService) InitiateCallByPhone(deviceID string, phone string) (string, []string, error) {
	log.Printf("[MQTT] 通过电话 %s 发起通话，设备ID: %s", phone, deviceID)

	// 通过电话号码查询住户
	var resident model.Resident
	if err := s.DB.Where("phone = ?", phone).First(&resident).Error; err != nil {
		return "", nil, fmt.Errorf("未找到电话为 %s 的住户: %v", phone, err)
	}

	// 生成唯一的通话ID
	callID := uuid.New().String()
	// 获取住户ID
	residentID := fmt.Sprintf("%d", resident.ID)
	callUnix := time.Now().Unix()
	roomID := BuildSharedTRTCRoomID(deviceID, residentID, callUnix)

	// 为住户生成UserSig
	tokenInfo, err := s.RTCService.GetUserSig(residentID)
	if err != nil {
		return "", nil, fmt.Errorf("生成UserSig失败: %v", err)
	}

	// 创建TRTC信息
	trtcInfo := model.TRTCInfo{
		RoomID:     roomID,
		RoomIDType: "string",
		SDKAppID:   tokenInfo.SDKAppID,
		UserID:     tokenInfo.UserID,
		UserSig:    tokenInfo.UserSig,
	}

	// 创建通话会话
	_, err = s.CallManager.CreateSession(callID, deviceID, residentID, trtcInfo)
	if err != nil {
		return "", nil, fmt.Errorf("创建通话会话失败: %v", err)
	}

	// 发送呼入通知给住户
	incomingNotification := IncomingCallMessage{
		CallID:           callID,
		DeviceDeviceID:   deviceID,
		TargetResidentID: residentID,
		Timestamp:        time.Now().UnixMilli(),
		TencentRTC: TRTCInfo{
			RoomIDType: trtcInfo.RoomIDType,
			RoomID:     trtcInfo.RoomID,
			SDKAppID:   trtcInfo.SDKAppID,
			UserID:     trtcInfo.UserID,
			UserSig:    trtcInfo.UserSig,
		},
	}

	// 发布到住户的呼入通知主题
	if err := s.publishMessage(TopicIncoming, incomingNotification); err != nil {
		return "", nil, fmt.Errorf("发送呼入通知失败: %v", err)
	}

	// 更新会话状态为振铃中
	s.CallManager.UpdateSessionStatus(callID, "ringing")

	// 创建振铃控制消息
	timestamp := time.Now().UnixMilli()
	ringControl := buildControlMessage(callID, "ringing", "", timestamp, s.mustGetSession(callID), residentID)

	// 先标记此消息为已处理，防止我们自己发出的消息被重复处理
	s.markMessageProcessed(callID, "ringing", timestamp)

	// 同时发送振铃消息给设备和住户
	controlPayload := buildControlPayload(ringControl)

	if err := s.publishMessage(TopicDeviceController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送振铃控制消息给设备失败: %v", err)
	}

	if err := s.publishMessage(TopicResidentController, controlPayload); err != nil {
		log.Printf("[MQTT] 发送振铃控制消息给住户失败: %v", err)
	}

	// 创建通话记录
	s.createCallRecord(callID, deviceID, residentID, "ringing")

	// 构建呼叫响应
	callResponse := CallResponse{
		CallID:            callID,
		DeviceDeviceID:    deviceID,
		TargetResidentIDs: []string{residentID},
		Timestamp:         time.Now().UnixMilli(),
		TencentRTC: TRTCInfo{
			RoomIDType: trtcInfo.RoomIDType,
			RoomID:     trtcInfo.RoomID,
			SDKAppID:   trtcInfo.SDKAppID,
			UserID:     deviceID,
		},
		CallInfo: ringControl,
	}

	log.Printf("[MQTT] 成功通过电话发起通话，callID: %s, 目标居民: %s, 响应: %+v",
		callID, residentID, callResponse)

	return callID, []string{residentID}, nil
}

// createCallRecord 创建通话记录
func (s *MQTTCallService) createCallRecord(callID, deviceID, residentID, status string) {
	s.CallRecordMutex.Lock()
	defer s.CallRecordMutex.Unlock()

	log.Printf("[MQTT] 创建通话记录: ID=%s, 设备=%s, 住户=%s, 状态=%s",
		callID, deviceID, residentID, status)

	devID, err := strconv.ParseUint(deviceID, 10, 64)
	if err != nil {
		log.Printf("[MQTT] 解析设备ID失败: %v", err)
		return
	}
	resID, err := strconv.ParseUint(residentID, 10, 64)
	if err != nil {
		log.Printf("[MQTT] 解析住户ID失败: %v", err)
		return
	}

	record := model.CallRecord{
		CallID:     callID,
		DeviceID:   uint(devID),
		ResidentID: uint(resID),
		CallStatus: model.CallStatus(status),
		Timestamp:  time.Now(),
	}
	if err := s.DB.Create(&record).Error; err != nil {
		log.Printf("[MQTT] 保存通话记录失败: %v", err)
	}
}

// updateCallRecord 更新通话记录
func (s *MQTTCallService) updateCallRecord(callID, status, reason string) {
	s.CallRecordMutex.Lock()
	defer s.CallRecordMutex.Unlock()

	log.Printf("[MQTT] 更新通话记录: ID=%s, 状态=%s, 原因=%s", callID, status, reason)

	updates := map[string]interface{}{
		"call_status": status,
	}

	// 如果是终态（挂断/结束），计算通话时长
	if status == "caller_hangup" || status == "callee_hangup" || status == "system_ended" {
		var record model.CallRecord
		if err := s.DB.Where("call_id = ?", callID).First(&record).Error; err == nil {
			duration := int(time.Since(record.Timestamp).Seconds())
			updates["duration"] = duration
		}
	}
	// 如果是接听，更新状态为 answered
	if status == "callee_answered" {
		updates["call_status"] = string(model.CallStatusAnswered)
	}

	if err := s.DB.Model(&model.CallRecord{}).Where("call_id = ?", callID).Updates(updates).Error; err != nil {
		log.Printf("[MQTT] 更新通话记录失败: %v", err)
	}
}

// PublishDeviceStatus 发布设备状态
func (s *MQTTCallService) PublishDeviceStatus(deviceID string, status map[string]interface{}) error {
	return s.publishMessage(TopicDeviceController, status)
}

// PublishSystemMessage 发布系统消息
func (s *MQTTCallService) PublishSystemMessage(messageType string, message map[string]interface{}) error {
	// 创建标准格式的系统消息
	systemMsg := SystemMessage{
		Type:      messageType,
		Level:     message["level"].(string),
		Message:   message["message"].(string),
		Data:      message["data"],
		Timestamp: time.Now().UnixMilli(),
	}

	return s.publishMessage(TopicSystemMessage, systemMsg)
}
