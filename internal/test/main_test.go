//go:build integration

package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestConfig 测试配置
type TestConfig struct {
	BaseURL     string `json:"base_url"`
	AdminUser   string `json:"admin_user"`
	AdminPass   string `json:"admin_pass"`
	Concurrency int    `json:"concurrency"`
	Requests    int    `json:"requests"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

var (
	config    TestConfig
	authToken string
)

// TestMain 测试主函数
func TestMain(m *testing.M) {
	// 加载测试配置
	if err := loadConfig(); err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 获取认证令牌
	token, err := getAuthToken()
	if err != nil {
		// 连接被拒绝说明服务器未启动，跳过而非失败
		var netErr *net.OpError
		if errors.As(err, &netErr) {
			fmt.Printf("服务器不可用，跳过集成测试: %v\n", err)
			os.Exit(0)
		}
		fmt.Printf("获取认证令牌失败: %v\n", err)
		os.Exit(1)
	}
	authToken = token

	// 运行测试
	os.Exit(m.Run())
}

// loadConfig 加载测试配置
func loadConfig() error {
	config = TestConfig{
		BaseURL:     "http://localhost:20033/api",
		AdminUser:   "admin",
		AdminPass:   "admin123",
		Concurrency: 10,
		Requests:    100,
	}
	return nil
}

// getAuthToken 发起真实登录请求，返回解析后的 JWT token
func getAuthToken() (string, error) {
	body, err := json.Marshal(LoginRequest{
		Username: config.AdminUser,
		Password: config.AdminPass,
	})
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(config.BaseURL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("登录请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if loginResp.Data.Token == "" {
		return "", fmt.Errorf("登录失败 (code=%d): %s", loginResp.Code, loginResp.Message)
	}

	return loginResp.Data.Token, nil
}

// TestDeviceList 测试设备列表接口
func TestDeviceList(t *testing.T) {
	benchmark := NewAPIBenchmark(config.BaseURL, config.Concurrency, config.Requests, authToken)
	result := benchmark.RunGET("/devices")
	result.PrintResult()

	// 验证结果
	if result.FailureCount > 0 {
		t.Errorf("设备列表接口测试失败: 成功率 %.2f%%", float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	}
}

// TestDeviceDetail 测试设备详情接口
func TestDeviceDetail(t *testing.T) {
	benchmark := NewAPIBenchmark(config.BaseURL, config.Concurrency, config.Requests, authToken)
	result := benchmark.RunGET("/devices/1") // 假设ID为1的设备存在
	result.PrintResult()

	// 验证结果
	if result.FailureCount > 0 {
		t.Errorf("设备详情接口测试失败: 成功率 %.2f%%", float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	}
}

// TestBuildingList 测试楼号列表接口
func TestBuildingList(t *testing.T) {
	benchmark := NewAPIBenchmark(config.BaseURL, config.Concurrency, config.Requests, authToken)
	result := benchmark.RunGET("/buildings")
	result.PrintResult()

	// 验证结果
	if result.FailureCount > 0 {
		t.Errorf("楼号列表接口测试失败: 成功率 %.2f%%", float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	}
}

// TestResidentList 测试住户列表接口
func TestResidentList(t *testing.T) {
	benchmark := NewAPIBenchmark(config.BaseURL, config.Concurrency, config.Requests, authToken)
	result := benchmark.RunGET("/residents")
	result.PrintResult()

	// 验证结果
	if result.FailureCount > 0 {
		t.Errorf("住户列表接口测试失败: 成功率 %.2f%%", float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	}
}

// TestCallRecordList 测试通话记录列表接口
func TestCallRecordList(t *testing.T) {
	benchmark := NewAPIBenchmark(config.BaseURL, config.Concurrency, config.Requests, authToken)
	result := benchmark.RunGET("/call-records")
	result.PrintResult()

	// 验证结果
	if result.FailureCount > 0 {
		t.Errorf("通话记录列表接口测试失败: 成功率 %.2f%%", float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	}
}

// TestMQTTCallInitiate 测试MQTT通话发起接口
func TestMQTTCallInitiate(t *testing.T) {
	benchmark := NewAPIBenchmark(config.BaseURL, config.Concurrency, config.Requests, authToken)

	// 通话请求数据
	callRequest := map[string]interface{}{
		"device_id":    "SN12345678",
		"household_id": 1,
	}

	result := benchmark.RunPOST("/mqtt/call", callRequest)
	result.PrintResult()

	// 验证结果
	if result.FailureCount > 0 {
		t.Errorf("MQTT通话发起接口测试失败: 成功率 %.2f%%", float64(result.SuccessCount)/float64(result.TotalRequests)*100)
	}
}
