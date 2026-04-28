# intercom_http_service API 文档

## 目录

- [认证接口](01_auth_api.md)
- [管理员接口](02_admin_api.md)
- [设备接口](03_device_api.md)
- [居民接口](04_resident_api.md)
- [物业员工接口](05_staff_api.md)
- [通话记录接口](06_call_record_api.md)
- [紧急情况接口](07_emergency_api.md)
- [楼号接口](08_building_api.md)
- [户号接口](09_household_api.md)
- [音视频通话接口](10_rtc_api.md)
- [健康检查接口](11_health_api.md)

## 简介

本文档提供了 intercom_http_service 对讲机后端服务的 API 接口说明，包括认证、管理员、设备、居民、物业员工、通话记录、紧急情况、楼号、户号、音视频通话和健康检查等模块的接口。

## 认证说明

系统当前文档包含两套认证方式：

### 1. 后台账号认证

适用于后台管理接口，登录后在请求头中携带：

```text
Authorization: Bearer <your_token>
```

### 2. 开放平台应用认证

适用于第三方对接场景：

1. 先调用 `/api/open/auth/token`
2. 使用 `Access ID + Access Key` 本地签名换取短期 Token
3. 再通过以下方式访问开放接口：

```text
Authorization: Bearer <app_token>
```

## 响应格式

所有 API 响应都遵循以下格式：

```json
{
	"code": 0, // 0 表示成功，非 0 表示错误
	"message": "成功", // 响应消息
	"data": {} // 响应数据，可能是对象或数组
}
```

## 错误码说明

- 0: 成功
- 400: 请求参数错误
- 401: 未授权或令牌无效
- 403: 权限不足
- 404: 资源不存在
- 500: 服务器内部错误
