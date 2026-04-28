# 认证接口

## 认证模式

系统支持两类认证模式：

1. **管理后台认证**
   - 面向管理员、物业员工、居民账号
   - 通过用户名密码登录获取 JWT
   - 适用于后台管理和业务操作接口

2. **开放平台认证**
   - 面向第三方应用、合作方系统
   - 通过 `Access ID + Access Key` 签名获取短期 Token
   - 适用于对外开放接口调用

---

## 1. `/api/auth/login` [POST]

- **简介**: 后台账号登录，获取 JWT 令牌
- **请求参数**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

- **响应参数**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 1,
    "username": "admin",
    "role": "admin",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "created_at": "2026-04-03T10:00:00Z"
  }
}
```

---

## 2. `/api/open/auth/token` [POST]

- **简介**: 第三方应用通过 `Access ID + Access Key` 签名换取短期访问 Token
- **说明**
  - `Access ID` 用于标识调用方身份
  - `Access Key` 仅用于调用方本地计算签名，**不能直接放到请求中**
  - 服务端通过 `Access ID` 找到对应的 `Access Key`，校验签名后签发短期 Token
  - 该流程参考云厂商开放平台常见做法，适合服务对服务调用

### 请求头

```json
{
  "X-Access-Id": "ilock_partner_app",
  "X-Timestamp": "1775198400",
  "X-Nonce": "6f5b4b3a2c1d",
  "X-Signature": "P9Z2H6X4mC1zJ4m7dGm7LQvP2rQXrN2QqfN6a8YzYhQ="
}
```

### 请求参数

```json
{
  "grant_type": "client_credentials",
  "ttl_seconds": 7200
}
```

### 响应参数

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "app_id": 1,
    "access_id": "ilock_partner_app",
    "token": "app_token_xxxxxxxxx",
    "token_type": "Bearer",
    "expires_in": 7200,
    "expires_at": "2026-04-03T12:00:00Z"
  }
}
```

### 字段说明

#### 请求头字段

- `X-Access-Id`: 调用方分配的应用标识
- `X-Timestamp`: Unix 时间戳，单位秒，建议服务端允许 `±300` 秒时钟偏差
- `X-Nonce`: 单次随机串，建议长度 `8~64` 位，用于防重放
- `X-Signature`: 使用 `Access Key` 对待签名串进行 `HMAC-SHA256` 后再 `Base64` 编码得到的签名

#### 请求体字段

- `grant_type`: 固定为 `client_credentials`
- `ttl_seconds`: 申请的 Token 有效期，建议范围 `300 ~ 7200` 秒，超出范围时按服务端策略裁剪

#### 响应字段

- `app_id`: 开放平台应用主键 ID
- `access_id`: 应用唯一访问标识
- `token`: 短期访问 Token
- `token_type`: 固定为 `Bearer`
- `expires_in`: Token 剩余有效期，单位秒
- `expires_at`: Token 失效时间，UTC 时间

---

## 开放平台签名规则

### 1. 请求体摘要

对原始请求体做 `SHA256`，得到十六进制小写摘要：

```text
body_sha256 = SHA256(raw_body)
```

请求体示例：

```json
{"grant_type":"client_credentials","ttl_seconds":7200}
```

### 2. 待签名串

按如下顺序拼接，使用换行符 `\n` 连接：

```text
POST
/api/open/auth/token
ilock_partner_app
1775198400
6f5b4b3a2c1d
<body_sha256>
```

格式说明：

```text
HTTP_METHOD
REQUEST_PATH
X-Access-Id
X-Timestamp
X-Nonce
body_sha256
```

### 3. 计算签名

```text
signature = Base64( HMAC-SHA256(AccessKey, stringToSign) )
```

### 4. 签名校验建议

- 时间戳超出允许窗口时拒绝请求
- `nonce` 在有效窗口内不得重复使用
- 签名不通过时返回鉴权失败
- `Access Key` 只允许服务端保存密文或安全存储，不写入日志

---

## 开放平台 Token 使用方式

第三方应用获取到 Token 后，后续调用开放接口时在请求头中携带：

```text
Authorization: Bearer app_token_xxxxxxxxx
```

---

## 开放平台错误响应示例

### 签名错误

```json
{
  "code": 401,
  "message": "invalid signature",
  "data": null
}
```

### 时间戳过期

```json
{
  "code": 401,
  "message": "timestamp expired",
  "data": null
}
```

### nonce 重复

```json
{
  "code": 401,
  "message": "nonce already used",
  "data": null
}
```

### Access ID 无效

```json
{
  "code": 404,
  "message": "access application not found",
  "data": null
}
```
