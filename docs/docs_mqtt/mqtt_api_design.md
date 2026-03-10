# MQTT通讯接口设计

## 一、MQTT配置与连接

### 1. MQTT服务器配置

intercom_http_service 提供了两种 MQTT 连接选项：

#### 1.1 本地MQTT服务器（开发环境）
- **地址**: localhost
- **标准端口**: 1883
- **WebSocket端口**: 9001
- **配置**: 允许匿名连接，无需用户名和密码

#### 1.2 云端MQTT服务器（生产环境）
- **地址**: pe0f0116.ala.cn-hangzhou.emqxsl.cn
- **SSL/TLS端口**: 8883
- **WebSocket SSL端口**: 8084
- **API地址**: https://pe0f0116.ala.cn-hangzhou.emqxsl.cn:8443/api/v5

### 2. MQTTX客户端测试配置

为了测试系统通信，您可以创建三个MQTTX客户端连接，模拟三个角色：

#### 2.1 intercom_http_service 服务器（服务端）
```
名称: intercom_http_service Server
主机: localhost (本地开发) 或 pe0f0116.ala.cn-hangzhou.emqxsl.cn (云端)
端口: 1883 (本地) 或 8883 (云端)
客户端ID: intercom_server
主题订阅: mqtt_call/#
```

#### 2.2 Resident客户端（住户）
```
名称: intercom Resident
主机: localhost (本地开发) 或 pe0f0116.ala.cn-hangzhou.emqxsl.cn (云端)
端口: 1883 (本地) 或 8883 (云端)
客户端ID: resident_[唯一ID]
主题订阅: 
- mqtt_call/incoming
- mqtt_call/controller/resident  
- mqtt_call/system
```

#### 2.3 Device客户端（设备）
```
名称: intercom Device
主机: localhost (本地开发) 或 pe0f0116.ala.cn-hangzhou.emqxsl.cn (云端)
端口: 1883 (本地) 或 8883 (云端)
客户端ID: device_[唯一ID]
主题订阅:
- mqtt_call/controller/device
- mqtt_call/system
```

### 3. 通信测试流程

1. 首先启动本地MQTT服务（如果使用本地开发环境）
2. 运行 intercom_http_service 对讲机后端服务
3. 使用MQTTX连接MQTT服务器，创建三个客户端
4. 通过API或MQTTX客户端向`mqtt_call/incoming`主题发送消息，测试通信流程

## 二、主题结构

系统使用以下固定主题：

1. `mqtt_call/incoming`
   - 用途：发送来电通知
   - QoS：1
   - 消息格式：
     ```json
     {
         "call_id": "string",
         "device_device_id": "string",
         "target_resident_id": "string",
         "timestamp": 1746870072136,
         "tencen_rtc": {
             "room_id_type": "string",
             "room_id": "string",
             "sdk_app_id": 1600084384,
             "user_id": "string",
             "user_sig": "string"
    }
}
```

2. `mqtt_call/controller/device`
   - 用途：设备端控制消息
   - QoS：1
   - 消息格式：
     ```json
     {
         "action": "string",  // ringing/answered/hangup
         "call_id": "string",
         "timestamp": 1746870072167,
         "reason": "string"
     }
     ```

3. `mqtt_call/controller/resident`
   - 用途：住户端控制消息
   - QoS：1
   - 消息格式：
     ```json
     {
         "action": "string",  // answered/rejected/hangup/timeout
         "call_id": "string",
         "timestamp": 1746870108846,
         "reason": "string"
}
```

4. `mqtt_call/system`
   - 用途：系统消息广播
   - QoS：1
   - 消息格式：
     ```json
     {
         "type": "string",
         "level": "string",  // info/warning/error
         "message": "string",
         "data": {},
         "timestamp": 1746870072136
     }
     ```

## 三、通话流程与并发控制

### 1. 通话架构与并发安全设计

MQTT通话系统采用发布/订阅模式，结合Go语言的并发特性实现高效稳定的通话处理：

#### 1.1 核心组件
- **MQTT服务器**：消息代理，负责消息路由和分发
- **后端服务**：处理业务逻辑，管理通话会话
- **设备端**：门禁设备，发起呼叫并参与通话
- **住户App**：接收来电通知，响应通话请求

#### 1.2 并发控制机制
系统采用多层次的并发控制机制确保在高并发环境下的数据一致性：

1. **分类互斥锁**：
   - 连接状态保护锁：保护连接状态读写
   - 通话记录互斥锁：保护通话记录创建
   - 消息发布互斥锁：保护MQTT消息发布
   - 会话操作互斥锁：保护会话操作

2. **线程安全的数据结构**：
   - 通话控制通道映射：存储通话控制通道
   - 已处理消息记录：存储已处理消息记录

3. **基于通道的会话控制**：
   每个通话会话使用独立的通信通道进行控制，实现隔离的状态管理

4. **消息去重机制**：
   通过callID、action和timestamp的组合唯一标识每条消息，防止消息重复处理

### 2. 详细通话流程

#### 2.1 发起呼叫流程

1. **请求处理与会话锁定**：使用互斥锁保护整个通话创建过程
2. **会话创建**：生成唯一的通话ID和创建通话控制通道
3. **启动独立控制流程**：为每个通话启动独立的控制流程
4. **TRTC房间创建**：创建TRTC房间并生成签名
5. **消息预处理与去重标记**：提前标记消息为已处理，防止重复处理
6. **发布消息**：向住户发送来电通知

#### 2.2 通话控制与状态管理

通话状态由独立的控制流程管理，处理多种事件：
- 处理通话控制消息（振铃、接听、拒绝、挂断等）
- 处理振铃超时
- 处理通话超时

#### 2.3 消息处理与去重

系统实现了严格的消息去重机制，防止消息循环处理：
- 接收消息时检查是否已处理
- 跳过重复消息
- 处理新消息并标记为已处理

#### 2.4 资源释放与清理

系统通过多种机制确保资源正确释放：
- 通话结束时清理资源
- 定期清理过期会话
- 定期清理过期消息记录

### 3. 安全设计

#### 3.1 通信安全

系统采用多层次的安全措施保护通信安全：
- **TLS/SSL加密**：支持加密连接
- **重连策略**：实现指数退避的重连机制，防止重连风暴
- **消息防重放**：通过时间戳和消息ID的组合，防止消息重放攻击

#### 3.2 超时控制

系统实现了多级超时控制，防止资源泄露：
- 振铃状态超时时间: 2分钟
- 通话中状态超时时间: 2小时

#### 3.3 错误恢复

所有关键处理过程都有异常恢复机制，确保系统稳定性：
- 异常捕获与日志记录
- 资源正确释放
- 会话状态恢复

### 4. 通话状态转换图

```
            ┌─────────────┐
            │   初始化    │
            └──────┬──────┘
                   ▼
            ┌─────────────┐
            │    振铃     │──────────────┐
            └──────┬──────┘              │
                   │                     │
      ┌────────────┼─────────────┐       │
      ▼            ▼             ▼       │
┌──────────┐ ┌──────────┐ ┌──────────┐   │
│  已接听  │ │  已拒绝  │ │  已超时  │   │
└────┬─────┘ └────┬─────┘ └────┬─────┘   │
     │            │            │         │
     ▼            ▼            ▼         │
┌─────────────────────────────────────┐  │
│              已结束                 │◀─┘
└─────────────────────────────────────┘
```

### 5. 完整通话示例流程

以下是一次完整的通话流程示例，包含并发控制和安全措施：

1. **发起呼叫**：
   - 设备发送HTTP请求：`POST /api/mqtt/call`
   - 请求体：
   ```json
   {
     "device_id": "5",
     "household_number": "MQTT-101"
   }
   ```
   - 后端加锁保护会话创建过程
   - 生成唯一通话ID并创建通话控制通道
   - 启动独立通话控制流程
   - 创建TRTC房间并生成签名
   - 标记消息为已处理，防止自我循环处理
   - 通过MQTT向住户发送来电通知
   - 释放会话创建锁

2. **住户响应**：
   - 住户接听通话：`POST /api/mqtt/controller/resident`
   - 请求体：
   ```json
   {
     "action": "answered",
     "call_id": "mqtt-call-20250510-abcdef123456"
   }
   ```
   - 后端检查消息是否已处理，避免重复处理
   - 标记消息为已处理
   - 向通话控制通道发送接听信号
   - 控制流程接收信号，更新状态为"connected"
   - 重置通话超时计时器
   - 通过MQTT向设备发送状态更新(发送前加锁)

3. **通话中控制**：
   - 独立的控制流程持续监控通话状态
   - 处理各种控制信号和超时事件
   - 确保状态转换的一致性

4. **结束通话**：
   - 当收到挂断信号或通话超时：
   - 向双方发送通话结束通知
   - 关闭通话控制通道
   - 删除会话记录
   - 更新通话记录(加锁)
   - 释放TRTC资源

## 四、HTTP API接口

### 1. 发起通话
- **路径**: `/api/mqtt/call`
- **方法**: POST
- **请求体**:
```json
{
      "device_id": "5",
      "household_number": "MQTT-101",
      "timestamp": 1746870072136
}
```
- **响应**:
```json
{
      "code": 200,
  "message": "成功",
  "data": {
          "call_id": "mqtt-call-20250510-abcdef123456",
          "device_device_id": "5",
          "target_resident_ids": ["6", "7"],
          "timestamp": 1746870072136,
          "tencen_rtc": {
              "room_id_type": "string",
              "room_id": "room_5_6_1746870072",
              "sdk_app_id": 1600084384,
              "user_id": "5",
              "user_sig": "eAEAowBc-3siVExTLnZlciI6IjIuMCIsIlRMUy5pZGVudGlmaWVyIjoiNSIsIlRMUy5zZGthcHBpZCI6MTYwMDA4NDM4NCwiVExTLmV4cGlyZSI6ODY0MDAsIlRMUy50aW1lIjoxNzQ2ODcwMDcyLCJUTFMuc2lnIjoieDcwTHQ2ZmxxWkZSendoLzJDMlpwakt4UXQvTkRtcDV5eFlTRXZkYlBGRT0ifQoBAAD--892L3E_"
          },
          "call_info": {
              "action": "ringing",
              "call_id": "mqtt-call-20250510-abcdef123456",
              "timestamp": 1746870072136
    }
  }
}
```

### 2. 设备端通话控制
- **路径**: `/api/mqtt/controller/device`
- **方法**: POST
- **请求体**:
  ```json
  {
      "action": "hangup",
      "call_id": "mqtt-call-20250510-abcdef123456",
      "timestamp": 1746870072136,
      "reason": "device_cancelled"
  }
  ```

### 3. 住户端通话控制
- **路径**: `/api/mqtt/controller/resident`
- **方法**: POST
- **请求体**:
```json
{
      "action": "answered",
      "call_id": "mqtt-call-20250510-abcdef123456",
      "timestamp": 1746870072136,
      "reason": ""
}
```

### 4. 获取通话会话
- **路径**: `/api/mqtt/session`
- **方法**: GET
- **参数**: `call_id=mqtt-call-20250510-abcdef123456`
- **响应**:
```json
{
      "code": 200,
  "message": "成功",
      "data": {
          "call_id": "mqtt-call-20250510-abcdef123456",
          "status": "connected",
          "device_id": "5",
          "resident_id": "6",
          "start_time": 1746870072136,
          "end_time": 1746870238164
      }
  }
  ```

### 5. 结束通话会话
- **路径**: `/api/mqtt/end-session`
- **方法**: POST
- **请求体**:
  ```json
  {
      "call_id": "string",
      "reason": "string"
  }
  ```

### 6. 更新设备状态
- **路径**: `/api/mqtt/device/status`
- **方法**: POST
- **请求体**:
  ```json
  {
      "device_id": "string",
      "status": {
          "online": true,
          "timestamp": 1746870072136
      }
  }
  ```

### 7. 发送系统消息
- **路径**: `/api/mqtt/system/message`
- **方法**: POST
- **请求体**:
  ```json
  {
      "type": "string",
      "level": "string",
      "message": "string",
      "target": ["string"]
  }
  ```

## 五、消息动作说明

### 1. 设备端动作
- `ringing`: 呼叫振铃中
- `hangup`: 挂断通话
- `cancelled`: 取消呼叫

### 2. 住户端动作
- `answered`: 接听通话
- `rejected`: 拒绝通话
- `hangup`: 挂断通话
- `timeout`: 呼叫超时

## 六、错误处理

所有API响应均使用统一的错误响应格式：
```json
{
    "code": 400,
    "message": "错误描述",
  "data": null
}
```

常见错误码：
- 400: 请求参数错误
- 404: 资源不存在
- 500: 服务器内部错误

## 七、高并发场景最佳实践

### 1. 客户端实现建议

为确保在高并发环境下系统的稳定性，客户端实现应遵循以下最佳实践：

#### 设备端
1. **重连机制**：实现指数退避的重连策略，避免重连风暴
2. **幂等性处理**：处理可能的重复消息，根据timestamp和callID过滤
3. **连接状态监控**：定期检查MQTT连接状态，在连接断开时主动重连
4. **QoS级别使用**：对关键消息使用QoS 1，确保至少一次送达

#### 住户端
1. **消息去重**：实现客户端消息去重机制
2. **会话状态管理**：本地维护通话状态，处理异常情况
3. **超时处理**：实现客户端超时机制，防止无限等待
4. **网络切换适配**：在网络切换时保持会话连续性

### 2. 通话会话限制

为防止资源耗尽，系统实施以下限制：

1. **单设备最大通话数**：每个设备同时最多支持1个活跃通话
2. **振铃超时**：通话振铃状态最长持续2分钟
3. **通话时长**：单次通话最长持续2小时
4. **消息频率**：每秒最多处理50条控制消息

### 3. 容错设计

系统设计了多层次的容错机制：

1. **消息重传**：关键状态变更消息在未收到确认时自动重传
2. **状态恢复**：服务重启后可从数据库恢复未完成的通话会话
3. **会话清理**：定期清理僵尸会话，释放资源
4. **日志追踪**：完整的消息日志，支持问题追踪和诊断