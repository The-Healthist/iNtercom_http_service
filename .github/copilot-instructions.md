# GitHub Copilot Instructions for intercom_http_service

## 项目背景

这是一个基于 Go 的**智能社区门禁对讲系统**后端服务，模块名为 `intercom_http_service`，采用扁平化分层结构。

系统服务四类客户端：

- **Web管理后台**：物业管理员操作，需 JWT 认证（`role=admin`）
- **门禁设备端**：嵌入式硬件，通过 HTTP 发起呼叫/上报状态，通过 MQTT 接收下行消息
- **居民App端**：接听通话、查看记录（当前待完善）
- **物业员工App端**：处理告警、查看分配设备（当前待完善）

### 当前目录结构

```
cmd/server/          # 服务入口 main.go
internal/
  config/            # 配置加载（Config 结构体 + 单例）
  database/          # GORM MySQL 连接池
  errcode/           # 统一错误码 + HTTP 响应工具
  handler/           # HTTP 控制器（对应各业务模块）
  logger/            # Zap 日志封装
  middleware/        # JWT 认证、限流、响应缓存
  model/             # GORM 数据模型
  mqtt/              # MQTT 证书与配置文件（无 Go 代码）
  router/            # Gin 路由注册
  service/           # 业务逻辑层 + ServiceContainer 依赖注入
  test/              # 基准测试
  utils/             # 通用工具（hash、compress、sign、random）
docs/                # API 文档、设计文档
```

### 核心数据层级关系

```
Building（楼号）
  └── Household（户号）
        ├── Resident（居民）
        └── Device（门禁设备）
              ├── PropertyStaff（物业员工，多对多）
              ├── CallRecord（通话记录）
              └── EmergencyLog（紧急事件日志）
```

---

## 代码生成与修改原则

1. 始终保持当前扁平化结构：`handler` / `service` / `model` / `errcode` 各司其职，不要混合。
2. 变更尽量小且聚焦，不要做无关重构。
3. 保持 Go 风格一致：命名清晰、职责单一、错误处理明确。
4. 优先兼容现有配置加载方式、环境变量命名和启动流程。
5. 修改配置相关逻辑时，必须同时考虑 `LOCAL_` / `SERVER_` 两类环境前缀。
6. 涉及数据库、Redis、MQTT、RTC、JWT 的配置项时，沿用现有 `Config` 结构体集中管理。
7. 不要随意移除 `panic` 型必填配置校验，除非需求明确要求改为可恢复错误。
8. 新增环境变量时：
   - 优先放入 `internal/config/config.go`
   - 为非必须项提供合理默认值，为必须项使用必填校验
   - 命名风格与现有字段保持一致
9. 修改接口或配置行为时，同步考虑 `docs/` 下文档的更新需求。

---

## 各层职责说明

### handler/（控制器层）

- 只负责请求解析、参数校验、调用 service、返回响应
- 使用 `errcode.Response*` 系列函数统一格式化响应
- 不包含任何业务逻辑或数据库操作

### service/（业务逻辑层）

- 所有服务通过 `ServiceContainer` 统一管理依赖
- `container.go` 在 `service` 包内，不单独成子包
- 新增服务时在 `ServiceContainer` 中注册，并提供 `Get*Service()` 方法

### model/（数据模型层）

- 包名为 `model`（不是 `models`）
- 所有 GORM 模型嵌入 `BaseModel`（含 `id`、`created_at`、`updated_at`）
- 密码字段统一加 `json:"-"` 防止序列化泄露

### errcode/（错误码与响应层）

- 包名为 `errcode`
- 包含：`code.go`（错误码常量）、`message.go`（中文消息映射）、`response.go`（HTTP 响应工具）
- `response.go` 内直接引用同包符号，不加 `errcode.` 前缀

### middleware/（中间件层）

- `auth.go`：JWT 认证（原 jwt.go）
- `cache.go`：响应缓存（Redis）
- `rate_limiter.go`：IP 维度 + 路径维度双重限流

---

## 配置模块特别说明

配置文件位于 `internal/config/config.go`。

- 从环境变量加载，根据 `ENV_TYPE` 切换 `LOCAL_` / `SERVER_` 前缀
- 统一管理数据库、Redis、RTC（阿里云 + 腾讯云）、MQTT、JWT、默认管理员密码等
- 通过 `sync.Once` 单例暴露 `GetConfig()`
- 提供 `GetDSN()` 和 `GetRedisAddr()` 等派生方法

### 配置约定

| 配置项               | 必填性                                  |
| -------------------- | --------------------------------------- |
| 数据库配置           | 必填，缺失则 panic                      |
| 阿里云 RTC           | 必填                                    |
| 腾讯云 TRTC          | 可选，通过 `TencentRTCEnabled` 开关控制 |
| MQTT                 | 可选，带合理默认值                      |
| JWT Secret           | 允许使用默认值，生产环境应显式配置      |
| DefaultAdminPassword | 必填                                    |

修改 `config.go` 时：

- 保持 `LoadConfig()`、`GetConfig()`、`GetDSN()`、`GetRedisAddr()` 调用签名不变
- 复用现有辅助函数：`getEnv()`、`getEnvAsInt()`、`getEnvAsBool()`、`getEnvRequired()`

---

## 路由与权限说明

| 路由组                       | 认证要求         | 限流     |
| ---------------------------- | ---------------- | -------- |
| `/api/ping`、`/api/health/*` | 无               | 10 req/s |
| `/api/auth/login`            | 无               | 10 req/s |
| `/api/rtc/*`、`/api/trtc/*`  | 无               | 5 req/s  |
| `/api/mqtt/*`                | 无               | 20 req/s |
| `/api/device/status`         | 无（设备心跳）   | 10 req/s |
| 其余 `/api/*`                | JWT `role=admin` | 30 req/s |

新增路由时，在 `internal/router/router.go` 的对应分组下注册。公开路由（设备/居民端）放 `registerPublicRoutes`，管理员路由放 `registerAuthenticatedRoutes`。

---

## 当前业务模块完成度

| 模块                   | 状态   | 说明                         |
| ---------------------- | ------ | ---------------------------- |
| 管理员（Admin）        | 完成   | CRUD + 分页搜索              |
| 楼号（Building）       | 完成   | CRUD + 设备/户号关联查询     |
| 户号（Household）      | 完成   | CRUD + 设备绑定/居民查询     |
| 设备（Device）         | 完成   | CRUD + 心跳上报 + 关联管理   |
| 居民（Resident）       | 完成   | CRUD（管理员视角）           |
| 物业员工（Staff）      | 完成   | CRUD + 设备批量绑定          |
| 通话记录（CallRecord） | 完成   | 查询 + 统计 + 质量反馈       |
| MQTT 通话（MQTTCall）  | 完成   | 发起/接听/挂断/会话管理      |
| 阿里云 RTC             | 完成   | Token 生成 + 发起通话        |
| 腾讯云 TRTC            | 完成   | UserSig 生成 + 发起通话      |
| 紧急情况（Emergency）  | 完成   | 告警触发 + 联系人 + 全员通知 |
| 健康检查（Health）     | 完成   | ping + 系统状态 + 缓存统计   |
| 居民端登录             | 待开发 | 目前只有 admin/staff 能登录  |
| 居民端个人 API         | 待开发 | 居民查询自己的通话记录等     |
| 设备拉取户号列表       | 待开发 | 单元门口机选号呼叫场景       |

---

## 生成代码时的输出偏好

1. 优先给出可直接落地的 Go 实现。
2. handler 只做参数校验和响应，业务逻辑放 service。
3. 新增配置项时补充注释说明用途、默认值和必填性。
4. 需求不明确时，延续现有工程风格，不引入新框架习惯。

---

## 文档与注释偏好

- 注释优先解释"为什么"，避免重复描述"做了什么"
- 环境变量、部署相关内容使用面向维护者的表达方式

---

## 安全与稳定性要求

- 日志中不输出密码、密钥、Token 等敏感信息
- 不为了"方便运行"移除必要的配置校验
- 修改数据库和认证配置时，优先保持向后兼容

---

## 回答风格

- 默认使用中文
- 先理解当前实现，再提出修改
- 若存在潜在风险，明确指出风险点与影响范围
- 优先给出最小可维护方案
