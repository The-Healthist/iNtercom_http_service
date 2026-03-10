# MQTT通话控制器设计文档

## 一、架构概述

MQTT通话控制器作为 intercom_http_service 对讲机后端服务的核心组件，负责处理实时通话请求、管理通话状态以及协调设备与住户之间的通信。本文档详细说明控制器的设计理念、实现方式和并发控制机制。

## 二、设计原则

1. **分层设计**：严格遵循MVC架构，将控制器、服务层和数据模型分离。
2. **接口驱动**：所有组件通过接口交互，便于单元测试和模块替换。
3. **并发安全**：采用多重锁机制和线程安全数据结构，确保高并发环境下的数据一致性。
4. **幂等性**：所有操作设计为幂等的，确保重复操作不会导致系统状态异常。
5. **容错性**：通过精细的错误处理和恢复机制，保证系统在各种异常情况下的稳定性。

## 三、控制器实现

### 1. 接口定义

控制器实现`InterfaceMQTTCallController`接口，提供以下核心方法：

```go
type InterfaceMQTTCallController interface {
    InitiateCall()            // 发起通话
    CallerAction()            // 处理呼叫方动作
    CalleeAction()            // 处理被呼叫方动作
    GetCallSession()          // 获取通话会话
    EndCallSession()          // 结束通话会话
    PublishDeviceStatus()     // 发布设备状态
    PublishSystemMessage()    // 发布系统消息
}
```

### 2. 依赖注入

控制器通过依赖注入获取所需服务：

```go
type MQTTCallController struct {
    Ctx       *gin.Context
    Container *container.ServiceContainer
}
```

通过服务容器获取MQTT通话服务：

```go
mqttCallService := c.Container.GetService("mqtt_call").(services.InterfaceMQTTCallService)
```

### 3. 路由处理

使用工厂函数生成针对不同方法的处理函数：

```go
func HandleMQTTCallFunc(container *container.ServiceContainer, method string) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        controller := NewMQTTCallController(ctx, container)
        // 根据method调用不同的控制器方法
    }
}
```

## 四、并发控制机制

MQTT通话服务在高并发环境下需要处理多个同时进行的通话会话，我们采用多层次的并发控制机制：

### 1. 互斥锁分类

系统使用多个专用互斥锁，针对不同资源实施精细的并发控制：

```go
type MQTTCallService struct {
    // ...其他字段
    connectedMutex  sync.RWMutex // 保护IsConnected字段的读写
    CallRecordMutex sync.Mutex   // 用于保护通话记录创建
    PublishMutex    sync.Mutex   // 用于保护MQTT消息发布
    SessionMutex    sync.RWMutex // 用于保护会话操作
    // ...其他字段
}
```

### 2. 读写锁使用策略

对于读多写少的场景，如连接状态和会话访问，使用读写锁提高并发性能：

```go
// 读取连接状态示例
s.connectedMutex.RLock()
isConnected := s.IsConnected && s.Client.IsConnected()
s.connectedMutex.RUnlock()
```

### 3. 基于通道的会话控制

每个通话会话使用独立的Go通道进行控制，实现了更细粒度的并发控制：

```go
// 创建会话控制通道
controlChan := make(chan CallControlMessage, 10) // 缓冲区大小10
s.CallChannels.Store(callID, controlChan)

// 启动独立的控制goroutine
go s.handleCallSession(callID, deviceID, residentID, controlChan)
```

### 4. 线程安全的Map

使用`sync.Map`存储会话和已处理消息信息，确保线程安全：

```go
// 会话通道存储
s.CallChannels.Store(callID, controlChan)

// 标记消息为已处理
s.ProcessedMsgs.Store(key, time.Now().Unix())
```

### 5. 消息去重机制

为防止MQTT消息循环处理，实现了基于消息ID和时间戳的去重机制：

```go
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
```

### 6. 定时清理机制

两个独立的后台goroutine负责清理过期的会话和消息记录：

```go
// 会话清理任务
go s.startSessionCleanupTask()

// 消息去重记录清理任务
go s.startMsgCleanupTask()
```

## 五、安全设计

### 1. MQTT连接安全

支持TLS/SSL加密连接，确保通信内容安全：

```go
// TLS配置示例
tlsConfig := &tls.Config{
    InsecureSkipVerify: true, // 在生产环境中应该验证证书
}
opts.SetTLSConfig(tlsConfig)
```

### 2. 重连策略

实现了指数退避的重连机制，避免重连风暴：

```go
backoffTime := time.Duration(1<<uint(i)) * time.Second // 指数退避: 1s, 2s, 4s, 8s, 16s
```

### 3. 消息防重放

通过时间戳和消息ID的组合，防止消息重放攻击：

```go
// 标记消息为已处理时包含时间戳
s.markMessageProcessed(callID, action, timestamp)
```

### 4. 会话超时控制

实现了多级超时控制，防止资源泄露：

```go
// 振铃状态超时时间: 2分钟
ringTimeout := 2 * time.Minute
// 通话中状态超时时间: 2小时
callTimeout := 2 * time.Hour
```

### 5. 状态一致性保障

发布消息前进行预处理，确保一致性：

```go
// 先标记此消息为已处理，防止我们自己发出的消息被重复处理
s.markMessageProcessed(callID, action, timestamp)
```

### 6. 资源释放保证

使用defer确保资源正确释放：

```go
defer func() {
    // 通话结束时清理资源
    close(controlChan)
    s.CallChannels.Delete(callID)
}()
```

## 六、通话生命周期管理

### 1. 会话创建

```go
// 创建通话会话
_, err = s.CallManager.CreateSession(callID, deviceID, residentID, trtcInfo)
```

### 2. 状态转换

通话状态转换由独立的goroutine管理，确保状态一致性：

```go
switch msg.Signal {
case SignalRinging:
    status = "ringing"
case SignalAnswered:
    status = "connected"
    // 重置超时时间
    callTimer.Reset(2 * time.Hour)
case SignalRejected:
    return
// ... 其他状态
}
```

### 3. 会话终止

提供多种终止机制，确保资源正确释放：

```go
// 发送挂断信号并结束会话
if err := s.EndCallSession(callID, reason); err != nil {
    return err
}
```

## 七、性能优化

### 1. 连接池管理

使用连接池和自动重连机制，提高系统稳定性：

```go
opts.SetAutoReconnect(true)
opts.SetMaxReconnectInterval(time.Second * 30)
```

### 2. 消息缓存

使用缓冲通道管理消息，提高高并发下的性能：

```go
controlChan := make(chan CallControlMessage, 10) // 缓冲区大小10
```

### 3. 定时清理

定时清理过期数据，防止内存泄露：

```go
ticker := time.NewTicker(5 * time.Minute)
for range ticker.C {
    // 清理过期数据
}
```

## 八、错误处理

### 1. 层级化错误处理

控制器层处理HTTP相关错误，服务层处理业务逻辑错误：

```go
// 控制器层错误处理
func (c *MQTTCallController) HandleError(status int, message string, err error) {
    errMessage := message
    if err != nil {
        errMessage = message + ": " + err.Error()
    }
    // 返回错误响应
}
```

### 2. Panic恢复

所有关键处理函数都有panic恢复机制，确保系统稳定性：

```go
defer func() {
    if r := recover(); r != nil {
        log.Printf("[MQTT] 处理panic: %v", r)
    }
}()
```

## 九、设计模式应用

### 1. 工厂模式

使用工厂函数创建控制器和服务实例：

```go
func NewMQTTCallController(ctx *gin.Context, container *container.ServiceContainer) InterfaceMQTTCallController
```

### 2. 依赖注入

通过依赖注入提供服务：

```go
func NewMQTTCallService(db *gorm.DB, cfg *config.Config, rtcService InterfaceTencentRTCService) InterfaceMQTTCallService
```

### 3. 观察者模式

MQTT的发布/订阅本质上是观察者模式的实现，系统中多处使用此模式。

### 4. 命令模式

通话控制消息实际上是命令模式的应用，将控制请求封装为对象：

```go
type CallControlMessage struct {
    Signal   CallControlSignal
    CallID   string
    DeviceID string
    UserID   string
    Action   string
    Reason   string
    Data     interface{}
}
```

## 十、测试策略

### 1. 单元测试

对关键组件进行单元测试，如消息解析、状态转换等：

```go
func TestMQTTCallService_InitiateCall(t *testing.T) {
    // 测试发起通话
}
```

### 2. 集成测试

模拟MQTT服务器，测试完整的通话流程：

```go
func TestFullCallFlow(t *testing.T) {
    // 测试完整通话流程
}
```

### 3. 负载测试

使用工具模拟高并发场景，验证系统稳定性：

```go
func BenchmarkMQTTCallService_InitiateCall(b *testing.B) {
    // 性能测试
}
```

## 十一、部署与扩展

### 1. 水平扩展

服务设计支持水平扩展，多个实例可共享同一个MQTT服务器。

### 2. 高可用配置

建议部署多个MQTT服务器节点，实现高可用。

### 3. 监控与告警

集成Prometheus和Grafana，实时监控系统状态。

## 十二、未来优化方向

1. 增加服务发现机制
2. 实现完整的事件溯源架构
3. 添加全链路追踪
4. 增强安全性，如加密通话内容
5. 实现更高效的消息序列化方案
