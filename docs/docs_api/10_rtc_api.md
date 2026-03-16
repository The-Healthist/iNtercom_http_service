# 音视频通话接口

## 发起 MQTT 通话

- **路径**: `/api/mqtt/call`
- **方法**: POST
- **描述**: 通过 MQTT 向关联设备的所有居民发起视频通话请求
- **参数**:
  ```json
  {
  	"device_device_id": "1",
  	"target_resident_id": "2",
  	"timestamp": 1651234567890
  }
  ```
- **响应**: 通话会话信息

## 处理 MQTT 呼叫方动作

- **路径**: `/api/mqtt/controller/device`
- **方法**: POST
- **描述**: 处理设备端通话动作(挂断、取消等)
- **参数**:
  ```json
  {
  	"call_info": {
  		"call_id": "call-20250510-abcdef123456",
  		"action": "answered",
  		"reason": "user_busy",
  		"timestamp": 1651234567890
  	}
  }
  ```
- **响应**: 处理结果

## 处理 MQTT 被呼叫方动作

- **路径**: `/api/mqtt/controller/resident`
- **方法**: POST
- **描述**: 处理居民端通话动作(接听、拒绝、挂断、超时等)
- **参数**: 同处理 MQTT 呼叫方动作
- **响应**: 处理结果

## 获取 MQTT 通话会话

- **路径**: `/api/mqtt/session`
- **方法**: GET
- **描述**: 获取通话会话信息及 TRTC 房间详情
- **参数**:
  - `call_id`: 通话会话 ID
- **响应**: 通话会话详情

## 结束 MQTT 通话会话

- **路径**: `/api/mqtt/end-session`
- **方法**: POST
- **描述**: 强制结束通话会话并通知所有参与方
- **参数**:
  ```json
  {
  	"call_id": "call-20250510-abcdef123456",
  	"reason": "call_completed"
  }
  ```
- **响应**: 结束结果

## 更新设备状态

- **路径**: `/api/mqtt/device/status`
- **方法**: POST
- **描述**: 更新设备状态信息，包括在线状态、电池电量和其他自定义属性
- **参数**:
  ```json
  {
  	"device_id": "1",
  	"online": true,
  	"battery": 85,
  	"properties": {}
  }
  ```
- **响应**: 更新结果

## 发布系统消息

- **路径**: `/api/mqtt/system/message`
- **方法**: POST
- **描述**: 通过 MQTT 发布系统消息
- **参数**:
  ```json
  {
  	"type": "notification",
  	"level": "info",
  	"message": "系统将于今晚22:00进行升级维护",
  	"timestamp": 1651234567890,
  	"data": {}
  }
  ```
- **响应**: 发布结果

## 获取腾讯云 UserSig

- **路径**: `/api/trtc/usersig`
- **方法**: POST
- **描述**: 获取腾讯云实时通信的 UserSig 凭证
- **参数**:
  ```json
  {
  	"user_id": "user123"
  }
  ```
- **响应**: UserSig 信息

## 开始腾讯视频通话

- **路径**: `/api/trtc/call`
- **方法**: POST
- **描述**: 在设备和居民之间发起腾讯云视频通话
- **参数**:
  ```json
  {
  	"device_id": "1",
  	"resident_id": "2"
  }
  ```
- **响应**: 通话会话信息
