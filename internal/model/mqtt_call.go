package model

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// CallSession 表示一个通话会话
type CallSession struct {
	CallID       string     // 通话唯一标识
	DeviceID     string     // 设备ID
	ResidentID   string     // 住户ID
	StartTime    time.Time  // 开始时间
	EndTime      time.Time  // 结束时间
	Status       string     // 状态: calling, ringing, connected, ended
	TRTCInfo     TRTCInfo   // 腾讯云TRTC房间信息
	LastActivity time.Time  // 最后活动时间
	mu           sync.Mutex // 互斥锁，保护会话状态修改
}

// TRTCInfo 包含TRTC相关信息
type TRTCInfo struct {
	RoomID     string
	RoomIDType string
	SDKAppID   int
	UserID     string
	UserSig    string
}

// CallManager 管理所有通话会话
type CallManager struct {
	sessions map[string]*CallSession // 以callID为键的会话映射
	mu       sync.RWMutex            // 读写锁保护会话映射
}

// NewCallManager 创建一个新的通话会话管理器
func NewCallManager() *CallManager {
	return &CallManager{
		sessions: make(map[string]*CallSession),
	}
}

// CreateSession 创建一个新的通话会话
func (m *CallManager) CreateSession(callID, deviceID, residentID string, trtcInfo TRTCInfo) (*CallSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查会话是否已存在
	if _, exists := m.sessions[callID]; exists {
		return nil, errors.New("会话已存在")
	}

	// 创建新会话
	session := &CallSession{
		CallID:       callID,
		DeviceID:     deviceID,
		ResidentID:   residentID,
		StartTime:    time.Now(),
		Status:       "calling", // 初始状态为呼叫中
		TRTCInfo:     trtcInfo,
		LastActivity: time.Now(),
	}

	m.sessions[callID] = session

	// 记录会话创建
	log.Printf("创建通话会话: ID=%s, 设备=%s, 住户=%s", callID, deviceID, residentID)

	return session, nil
}

// GetSession 获取指定通话会话
func (m *CallManager) GetSession(callID string) (*CallSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[callID]
	return session, exists
}

// UpdateSessionStatus 更新会话状态
func (m *CallManager) UpdateSessionStatus(callID, status string) error {
	m.mu.RLock()
	session, exists := m.sessions[callID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("会话不存在: %s", callID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Status = status
	session.LastActivity = time.Now()

	log.Printf("更新会话状态: ID=%s, 状态=%s", callID, status)
	return nil
}

// EndSession 结束通话会话
func (m *CallManager) EndSession(callID, reason string) (*CallSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[callID]
	if !exists {
		return nil, fmt.Errorf("会话不存在: %s", callID)
	}

	session.mu.Lock()
	// 更新会话状态
	session.Status = "ended"
	session.EndTime = time.Now()
	session.LastActivity = time.Now()
	session.mu.Unlock()

	// 记录会话结束
	duration := session.EndTime.Sub(session.StartTime)
	log.Printf("结束通话会话: ID=%s, 原因=%s, 持续时间=%v", callID, reason, duration)

	// 从映射中移除会话
	delete(m.sessions, callID)

	return session, nil
}

// GetAllActiveSessions 获取所有活动会话
func (m *CallManager) GetAllActiveSessions() []*CallSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeSessions := make([]*CallSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		activeSessions = append(activeSessions, session)
	}

	return activeSessions
}

// CleanupTimedOutSessions 清理超时会话
func (m *CallManager) CleanupTimedOutSessions(callTimeout, ringTimeout time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	var cleanedCount int
	now := time.Now()

	for callID, session := range m.sessions {
		session.mu.Lock()
		lastActivity := session.LastActivity
		status := session.Status
		session.mu.Unlock()

		var timeout time.Duration
		if status == "calling" || status == "ringing" {
			// 呼叫中或振铃中状态的超时较短
			timeout = ringTimeout
		} else {
			// 已连接状态的超时较长
			timeout = callTimeout
		}

		if now.Sub(lastActivity) > timeout {
			// 会话超时，记录并删除
			log.Printf("会话超时: ID=%s, 状态=%s, 最后活动=%v", callID, status, lastActivity)
			delete(m.sessions, callID)
			cleanedCount++
		}
	}

	return cleanedCount
}

// UpdateSessionActivity 更新会话最后活动时间
func (m *CallManager) UpdateSessionActivity(callID string) error {
	m.mu.RLock()
	session, exists := m.sessions[callID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("会话不存在: %s", callID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.LastActivity = time.Now()
	return nil
}

// SessionExists 检查会话是否存在
func (m *CallManager) SessionExists(callID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.sessions[callID]
	return exists
}
