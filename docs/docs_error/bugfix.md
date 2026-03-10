# intercom_http_service 错误处理方案

## 1. 问题背景

在当前的 intercom_http_service 项目中，错误处理存在以下问题：

1. 错误码不统一，不同模块使用不同的错误返回格式
2. 缺少错误堆栈信息，导致问题定位困难
3. 错误日志分散，格式不一致
4. 缺乏对外统一的错误响应格式
5. 从阿里云迁移到京东云过程中，错误处理不完善导致迁移失败难以定位

## 2. 错误处理方案设计

### 2.1 错误码设计规范

采用统一的错误码格式：`SCMMM`

- **S**: 服务标识(1-9)
    - `1`: intercom_http_service
    - `2`: intercom MQTT Service
  - `3`: 其他服务
- **C**: 模块标识(0-9)
  - `0`: 通用模块
  - `1`: 用户模块
  - `2`: 设备模块
  - `3`: 权限模块
  - `4`: 日志模块
  - `5`: 配置模块
  - `6`: 数据库模块
  - `7`: Redis模块
  - `8`: MQTT模块
  - `9`: 迁移模块
- **MMM**: 模块内错误序号(000-999)

### 2.2 HTTP 状态码映射

限制使用以下6种HTTP状态码：

- `200`: 成功
- `400`: 客户端错误
- `401`: 认证失败
- `403`: 授权失败
- `404`: 资源不存在
- `500`: 服务器错误

### 2.3 错误响应格式

统一的JSON响应格式：

```json
{
  "code": 100101,
  "message": "用户未找到",
    "reference": "https://github.com/username/intercom_http_service/blob/master/docs/errors.md",
  "data": null
}
```

## 3. 实施步骤

### 步骤1：创建错误包和错误码包

1. 安装依赖包：

```bash
go get -u github.com/marmotedu/errors
go get -u github.com/marmotedu/log
```

2. 创建错误码包结构：

```
internal/
  └── pkg/
      └── code/
          ├── base.go       # 通用错误码
          ├── http.go       # HTTP服务错误码
          ├── device.go     # 设备相关错误码
          ├── user.go       # 用户相关错误码
          ├── migration.go  # 迁移相关错误码
          └── register.go   # 错误码注册
```

### 步骤2：定义基础错误码

在 `internal/pkg/code/base.go` 中定义：

```go
package code

import "github.com/marmotedu/errors"

// 通用错误码
const (
    // ErrSuccess - 200: 成功
    ErrSuccess int = iota + 100001

    // ErrUnknown - 500: 服务器内部错误
    ErrUnknown

    // ErrBind - 400: 请求参数绑定错误
    ErrBind

    // ErrValidation - 400: 参数验证失败
    ErrValidation

    // ErrTokenInvalid - 401: 令牌无效
    ErrTokenInvalid

    // ErrDatabase - 500: 数据库错误
    ErrDatabase

    // ErrRedis - 500: Redis错误
    ErrRedis
)
```

### 步骤3：定义模块错误码

在 `internal/pkg/code/http.go` 中定义：

```go
package code

// HTTP服务错误码
const (
    // ErrRequestTimeout - 500: 请求超时
    ErrRequestTimeout int = iota + 100101

    // ErrTooManyRequests - 429: 请求过于频繁
    ErrTooManyRequests
)
```

在 `internal/pkg/code/device.go` 中定义：

```go
package code

// 设备错误码
const (
    // ErrDeviceNotFound - 404: 设备未找到
    ErrDeviceNotFound int = iota + 102001

    // ErrDeviceAlreadyExist - 400: 设备已存在
    ErrDeviceAlreadyExist

    // ErrDeviceOffline - 400: 设备离线
    ErrDeviceOffline
)
```

### 步骤4：实现错误码注册

在 `internal/pkg/code/register.go` 中实现：

```go
package code

import "github.com/marmotedu/errors"

// 注册所有错误码
func init() {
    // 注册通用错误码
    errors.MustRegister(ErrSuccess, 200, "成功")
    errors.MustRegister(ErrUnknown, 500, "服务器内部错误")
    errors.MustRegister(ErrBind, 400, "请求参数绑定错误")
    errors.MustRegister(ErrValidation, 400, "参数验证失败")
    errors.MustRegister(ErrTokenInvalid, 401, "令牌无效")
    errors.MustRegister(ErrDatabase, 500, "数据库错误")
    errors.MustRegister(ErrRedis, 500, "Redis错误")

    // 注册HTTP服务错误码
    errors.MustRegister(ErrRequestTimeout, 500, "请求超时")
    errors.MustRegister(ErrTooManyRequests, 429, "请求过于频繁")

    // 注册设备错误码
    errors.MustRegister(ErrDeviceNotFound, 404, "设备未找到")
    errors.MustRegister(ErrDeviceAlreadyExist, 400, "设备已存在")
    errors.MustRegister(ErrDeviceOffline, 400, "设备离线")

    // 其他模块错误码注册...
}
```

### 步骤5：创建统一响应处理

在 `internal/pkg/response/response.go` 中实现：

```go
package response

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/marmotedu/errors"
    
    "github.com/username/intercom_http_service/internal/pkg/code"
)

// Response 定义统一的响应格式
type Response struct {
    Code      int         `json:"code,omitempty"`
    Message   string      `json:"message,omitempty"`
    Reference string      `json:"reference,omitempty"`
    Data      interface{} `json:"data,omitempty"`
}

// WriteResponse 写入响应
func WriteResponse(c *gin.Context, err error, data interface{}) {
    if err != nil {
        coder := errors.ParseCoder(err)
        c.JSON(coder.HTTPStatus(), Response{
            Code:      coder.Code(),
            Message:   coder.String(),
            Reference: "https://github.com/username/intercom_http_service/blob/master/docs/errors.md",
            Data:      data,
        })
        return
    }

    c.JSON(http.StatusOK, Response{
        Code:    code.ErrSuccess,
        Message: "成功",
        Data:    data,
    })
}
```

### 步骤6：错误处理示例

在业务代码中使用错误处理：

```go
package handler

import (
    "github.com/gin-gonic/gin"
    "github.com/marmotedu/errors"
    "github.com/marmotedu/log"
    
    "github.com/username/intercom_http_service/internal/pkg/code"
    "github.com/username/intercom_http_service/internal/pkg/response"
    "github.com/username/intercom_http_service/internal/service"
)

// GetDevice 获取设备信息
func GetDevice(c *gin.Context) {
    deviceID := c.Param("id")
    
    // 参数验证
    if deviceID == "" {
        response.WriteResponse(c, errors.WithCode(code.ErrValidation, "设备ID不能为空"), nil)
        return
    }
    
    // 调用服务层
    device, err := service.GetDevice(deviceID)
    if err != nil {
        // 直接返回服务层的错误，不需要再包装
        response.WriteResponse(c, err, nil)
        return
    }
    
    // 成功响应
    response.WriteResponse(c, nil, device)
}
```

在服务层中使用错误处理：

```go
package service

import (
    "github.com/marmotedu/errors"
    "github.com/marmotedu/log"
    
    "github.com/username/intercom_http_service/internal/model"
    "github.com/username/intercom_http_service/internal/pkg/code"
    "github.com/username/intercom_http_service/internal/store"
)

// GetDevice 获取设备信息
func GetDevice(deviceID string) (*model.Device, error) {
    // 从数据库获取设备
    device, err := store.GetDevice(deviceID)
    if err != nil {
        if store.IsNotFound(err) {
            return nil, errors.WithCode(code.ErrDeviceNotFound, "设备 %s 不存在", deviceID)
        }
        
        // 记录内部错误
        log.Errorf("查询设备失败: %v", err)
        return nil, errors.WrapC(err, code.ErrDatabase, "查询设备数据库失败")
    }
    
    // 检查设备状态
    if !device.IsOnline {
        return nil, errors.WithCode(code.ErrDeviceOffline, "设备 %s 当前离线", deviceID)
    }
    
    return device, nil
}
```

### 步骤7：生成错误码文档

1. 安装 `codegen` 工具：

```bash
go get -u github.com/marmotedu/codegen
```

2. 在项目根目录创建 `code_gen.go` 文件：

```go
package main

//go:generate codegen -type=int -doc -output ./docs/errors.md
```

3. 执行生成命令：

```bash
go generate ./...
```

## 4. 迁移功能错误处理增强

为迁移功能增加专门的错误处理：

1. 在 `migration_scripts` 目录中添加错误处理：

```go
// 在迁移脚本中使用错误处理
func backupDatabase() error {
    cmd := exec.Command("mysqldump", "-u", "root", "-p"+password, "intercom")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return errors.WrapC(err, code.ErrBackupFailed, "数据库备份失败: %s", string(output))
    }
    return nil
}
```

2. 在迁移脚本中添加详细的日志记录：

```bash
function handle_error() {
    local exit_code=$1
    local error_message=$2
    local error_code=$3
    
    if [ $exit_code -ne 0 ]; then
        print_error "$error_message (错误码: $error_code)"
        echo "$(date '+%Y-%m-%d %H:%M:%S') - 错误: $error_message (错误码: $error_code)" >> "$LOG_FILE"
        exit $exit_code
    fi
}
```

## 5. 实施计划

1. **第一阶段**: 创建错误包和错误码包结构
   - 预计时间: 1天
   - 负责人: [开发人员]

2. **第二阶段**: 实现基础错误码和响应处理
   - 预计时间: 2天
   - 负责人: [开发人员]

3. **第三阶段**: 在关键模块中应用新的错误处理方案
   - 预计时间: 3天
   - 负责人: [开发人员]

4. **第四阶段**: 完善迁移脚本的错误处理
   - 预计时间: 2天
   - 负责人: [开发人员]

5. **第五阶段**: 测试和文档完善
   - 预计时间: 2天
   - 负责人: [测试人员]

## 6. 预期效果

1. 统一的错误处理机制，提高代码可维护性
2. 详细的错误堆栈信息，便于问题定位
3. 规范的错误码体系，便于客户端处理
4. 完善的错误文档，便于开发和维护
5. 增强的迁移脚本错误处理，提高迁移成功率

## 7. 参考资料

1. [Go 错误处理最佳实践](https://github.com/marmotedu/errors)
2. [IAM 项目错误码设计](https://github.com/marmotedu/iam/tree/master/internal/pkg/code)
3. [Gin 框架错误处理](https://github.com/gin-gonic/gin#model-binding-and-validation) 