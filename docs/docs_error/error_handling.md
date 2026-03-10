# intercom_http_service 错误处理规范

## 1. 错误码设计

错误码采用6位数字格式，格式为：`SCMXXX`

- **S**: 服务标识，固定为 `1`
- **C**: 组件类别
  - `0`: 通用错误
  - `1`: 用户相关
  - `2`: 设备相关
  - `3`: 住户相关
  - `4`: 呼叫相关
  - `5`: 数据库相关
  - `9`: 迁移相关
- **M**: 模块标识
  - `0`: 通用模块
  - `1`-`9`: 具体模块
- **XXX**: 具体错误码，从 `000` 开始递增

### 通用错误码 (100xxx)

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 100000 | 成功 | 200 |
| 100001 | 未知错误 | 500 |
| 100002 | 请求参数绑定错误 | 400 |
| 100003 | 请求参数验证错误 | 400 |
| 100004 | 令牌无效 | 401 |

### 用户相关错误码 (101xxx)

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 101000 | 用户不存在 | 404 |
| 101001 | 用户已存在 | 400 |
| 101002 | 用户密码错误 | 401 |

### 设备相关错误码 (102xxx)

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 102000 | 设备不存在 | 404 |
| 102001 | 设备已存在 | 400 |
| 102002 | 设备离线 | 400 |
| 102003 | 设备忙 | 400 |

### 住户相关错误码 (103xxx)

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 103000 | 住户不存在 | 404 |
| 103001 | 住户已存在 | 400 |

### 呼叫相关错误码 (104xxx)

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 104000 | 呼叫记录不存在 | 404 |
| 104001 | 呼叫超时 | 400 |

### 数据库相关错误码 (105xxx)

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 105000 | 数据库错误 | 500 |
| 105001 | 记录不存在 | 404 |

### 迁移相关错误码 (109xxx)

| 错误码 | 描述 | HTTP状态码 |
|--------|------|------------|
| 109000 | 迁移失败 | 500 |
| 109001 | 备份失败 | 500 |
| 109002 | 恢复失败 | 500 |
| 109003 | 连接失败 | 500 |

## 2. 统一响应格式

所有API响应采用统一的JSON格式：

```json
{
  "code": 100000,
  "message": "成功",
  "data": { ... } // 可选，成功时返回数据
}
```

- `code`: 错误码，成功为 100000，失败为对应错误码
- `message`: 错误消息，描述错误原因
- `data`: 返回数据，可选字段，成功时包含返回数据

## 3. 使用方法

### 成功响应

```go
import "intercom_http_service/internal/error/response"

func (c *Controller) HandleRequest(ctx *gin.Context) {
    // 业务逻辑...
    data := gin.H{
        "id": 1,
        "name": "测试",
    }
    response.Success(ctx, data)
}
```

### 失败响应

```go
import (
    "intercom_http_service/internal/error/code"
    "intercom_http_service/internal/error/response"
)

func (c *Controller) HandleRequest(ctx *gin.Context) {
    // 业务逻辑...
    if err != nil {
        response.Fail(ctx, code.ErrDatabase, nil)
        return
    }
    
    // 或者自定义消息
    response.FailWithMessage(ctx, code.ErrDatabase, "数据库连接失败", nil)
}
```

### 常用响应方法

- `response.Success(ctx, data)`: 成功响应
- `response.Fail(ctx, errorCode, data)`: 失败响应
- `response.FailWithMessage(ctx, errorCode, message, data)`: 自定义消息的失败响应
- `response.ParamError(ctx, message)`: 参数错误响应
- `response.ServerError(ctx)`: 服务器错误响应
- `response.NotFound(ctx, message)`: 资源不存在响应
- `response.Unauthorized(ctx)`: 未授权响应 
graph TD
    A[Analyze Controllers] --> B[Identify Response Patterns]
    B --> C[Update Imports]
    C --> D[Standardize Error Responses]
    D --> E[Standardize Success Responses]
    E --> F[Test API Endpoints]
    
    D --> D1[Use response.ParamError for 400]
    D --> D2[Use response.NotFound for 404]
    D --> D3[Use response.FailWithMessage for other errors]
    D --> D4[Use appropriate error codes from code package]
    
    E --> E1[Use response.Success for all success responses]