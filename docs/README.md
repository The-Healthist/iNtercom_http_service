# intercom_http_service 对讲机后端服务

## 项目概述

intercom_http_service 是一个基于 Go 语言开发的对讲机后端服务，提供视频通话、设备联动、门禁控制和紧急情况处理能力。系统采用 Docker 容器化部署，便于在各种环境中快速安装和更新。

## 系统架构

- **后端**: Go + 标准库 net/http (Go 1.22 新版 ServeMux)
- **数据库**: MySQL 8.0
- **缓存**: Redis 7.0
- **消息队列**: MQTT (Eclipse Mosquitto 2.0)
- **部署**: Docker + Docker Compose
- **通讯**:
  - RESTful API: 基础业务操作
  - MQTT: 实时消息推送、视频通话信令
  - 音视频通话: 支持阿里云RTC和腾讯云TRTC

## 项目结构

项目采用清晰的分层架构设计：

```
intercom_http_service/
├── cmd/                   # 应用入口点
│   └── server/            # 主服务器
├── internal/              # 内部包，不对外暴露
│   ├── app/               # 应用层
│   │   ├── controllers/   # 控制器
│   │   ├── middleware/    # 中间件
│   │   └── routes/        # 路由定义
│   ├── domain/            # 领域层
│   │   ├── models/        # 数据模型
│   │   └── services/      # 业务服务
│   │       └── container/ # 服务容器
│   ├── error/             # 错误处理
│   │   ├── code/          # 错误码
│   │   └── response/      # 响应格式
│   ├── infrastructure/    # 基础设施层
│   │   ├── config/        # 配置管理
│   │   ├── database/      # 数据库连接池
│   │   └── mqtt/          # MQTT配置
│   └── test/              # 测试
│       └── benchmark/     # 性能测试
├── pkg/                   # 可共享的包
│   ├── logger/            # 日志工具
│   ├── utils/             # 通用工具
│   └── validator/         # 数据验证
├── docs/                  # 文档
│   ├── docs_api/          # API文档
│   ├── docs_code_test/    # 代码测试文档
│   ├── docs_deploy/       # 部署文档
│   ├── docs_error/        # 错误处理文档
│   ├── docs_mqtt/         # MQTT通信文档
│   ├── swagger.json       # Swagger API文档
│   └── swagger.yaml       # Swagger API文档(YAML格式)
├── scripts/               # 脚本文件
│   ├── deploy/            # 部署脚本
│   ├── migrate/           # 迁移脚本
│   └── lint.sh            # 代码检查脚本
├── logs/                  # 日志目录
└── docker-compose.yml     # Docker Compose配置
```

## MQTT 通信架构

### 主题设计

#### 视频通话相关主题

- **呼叫请求**: `mqtt_call/call`
- **来电通知**: `mqtt_call/incoming`
- **设备控制**: `mqtt_call/controller/device`
- **用户控制**: `mqtt_call/controller/resident`

### 消息格式

所有消息采用 JSON 格式，包含以下基本字段：

- 呼叫请求:
  ```json
  {
    "device_device_id": "设备ID",
    "target_resident_id": "目标用户ID",
    "timestamp": 1678886400000
  }
  ```

- 通话控制:
  ```json
  {
    "call_info": {
      "action": "hangup/rejected/reveive/ringing",
      "call_id": "呼叫ID",
      "timestamp": 1678886500000,
      "reason": "可选原因说明"
    }
  }
  ```

## 主要功能

- **用户管理**：管理员、物业人员、居民
- **设备管理**：智能门锁监控和控制
- **视频通话**：访客与居民之间的实时沟通
- **紧急情况处理**：火灾、入侵、医疗等紧急事件
- **完整的认证和权限管理**

## 部署指南

### 前置要求

- Docker 和 Docker Compose
- Linux 服务器（推荐 Ubuntu 20.04 或 CentOS 8）
- 开放端口：20033(HTTP), 3310(MySQL), 6380(Redis), 1883/8883/9001(MQTT)

### 使用部署脚本

我们提供了部署脚本，可以自动完成部署过程：

1. **配置部署脚本**:

   ```bash
   # 编辑脚本中的服务器信息
   vim scripts/deploy/deploy_to_server.sh
   ```

2. **执行部署脚本**:

   ```bash
   chmod +x scripts/deploy/deploy_to_server.sh
   ./scripts/deploy/deploy_to_server.sh -s 服务器IP -u root
   ```

3. **验证部署**:
   - 访问 `http://服务器IP:20033/api/ping` 检查服务运行状态
   - 访问 `http://服务器IP:20033/swagger/index.html` 查看API文档

### 手动部署

如果需要手动部署，可以按照以下步骤操作：

1. **克隆代码到本地**:

   ```bash
   git clone <repository-url>
  cd intercom_http_service
   ```

2. **配置服务**:

   ```bash
   # 编辑docker-compose.yml文件，根据需要修改配置
   vim docker-compose.yml
   ```

3. **启动服务**:
   ```bash
   docker-compose up -d
   ```

## 迁移指南

当需要将系统从一台服务器迁移到另一台服务器时，可以使用我们提供的迁移脚本：

1. **备份源服务器数据**:

   ```bash
   # 编辑备份脚本中的源服务器信息
   vim scripts/migrate/backup.sh
   # 执行备份
   ./scripts/migrate/backup.sh
   ```

2. **迁移到目标服务器**:

   ```bash
   # 编辑迁移脚本中的目标服务器信息
   vim scripts/migrate/migrate.sh
   # 执行迁移
   ./scripts/migrate/migrate.sh
   ```

3. **验证迁移**:
   - 检查服务状态: `docker-compose ps`
   - 测试API接口: `curl http://服务器IP:20033/api/ping`

如果迁移失败，可以使用回滚脚本:
```bash
./scripts/migrate/rollback.sh
```

## API 文档

系统集成了 Swagger 文档，部署后可以通过以下地址访问：

http://服务器IP:20033/swagger/index.html

主要 API 端点包括：

- **健康检查**: `/api/ping`, `/api/health/status`
- **认证**: `/api/auth/login`
- **管理员**: `/api/admin/*`
- **物业人员**: `/api/staffs/*`
- **居民**: `/api/residents/*`
- **设备**: `/api/devices/*`
- **通话记录**: `/api/call-records/*`
- **紧急情况**: `/api/emergency/*`
- **楼号管理**: `/api/buildings/*`
- **户号管理**: `/api/households/*`
- **MQTT通信**: `/api/mqtt/*`
- **RTC服务**: `/api/rtc/*`, `/api/trtc/*`

## 系统特性

- **自动迁移**: 支持数据库自动迁移，包括alter和drop模式
- **基于角色的访问控制**: 不同角色拥有不同权限
- **安全通信**: 基于JWT的API认证
- **性能优化**:
  - 高效的数据库连接池管理
  - 响应缓存中间件，支持多种缓存策略
  - 多级限流保护，支持IP、路径和自定义限流
- **视频通话集成**: 支持阿里云RTC和腾讯云TRTC
- **容器化部署**: 使用Docker和Docker Compose简化部署和维护
- **健康检查**: 服务健康状态监控，确保系统稳定运行

## 故障排除

1. **服务无法启动**:

   - 检查Docker和Docker Compose是否正确安装
   - 检查端口是否被占用: `netstat -tunlp`
   - 查看容器日志: `docker-compose logs app`

2. **数据库连接失败**:

   - 检查数据库配置是否正确
   - 确认数据库服务是否运行: `docker-compose ps db`
   - 检查数据库日志: `docker-compose logs db`
   - 检查连接池状态: `curl http://服务器IP:20033/api/health/cache-stats`

3. **MQTT服务问题**:

  - 检查MQTT配置文件: `/root/intercom_http_service/mqtt/config/mosquitto.conf`
  - 确认ACL配置正确: `/root/intercom_http_service/mqtt/config/acl.conf`
   - 查看MQTT日志: `docker-compose logs mqtt`

4. **视频通话失败**:
   - 检查阿里云/腾讯云配置是否正确
   - 确认MQTT服务正常运行
   - 检查防火墙是否开放了必要的端口

## 规范设置

### 1. 代码风格规范

- **Go 语言规范**: 遵循官方 Go 语言规范和最佳实践
- **命名规则**:
  - 包名: 小写单词，不使用下划线或混合大小写
  - 文件名: 小写，使用下划线分隔多个单词
  - 结构体名: 驼峰命名法，首字母大写
  - 接口名: 通常以 "er" 结尾，如 Reader, Writer
  - 方法和函数: 驼峰命名法，公开方法首字母大写，私有方法首字母小写
- **注释规范**: 所有导出的函数、类型和变量必须有注释

### 2. 代码静态检查

- **Linter 工具**: 使用 `golangci-lint` 进行代码静态分析
  ```bash
  # 执行代码检查
  ./scripts/lint.sh
  ```
- **检查规则**:
  - 格式检查: `gofmt`
  - 语法检查: `golint`
  - 错误检查: `errcheck`
  - 简单性能问题: `gosimple`
  - 未使用的代码: `unused`
  - 代码复杂度: `gocyclo`
- **CI/CD 集成**: 在提交代码前和 CI 流程中自动运行静态检查

### 3. 接口设计规范

- **RESTful API 设计**:
  - 使用恰当的 HTTP 方法: GET(查询), POST(创建), PUT(更新), DELETE(删除)
  - 路径使用名词而非动词: `/api/devices` 而非 `/api/getDevices`
  - 使用复数形式表示资源集合: `/api/residents` 而非 `/api/resident`
  - 使用嵌套结构表示资源关系: `/api/buildings/:id/households`
- **响应格式统一**:
  ```json
  {
    "code": 200,
    "message": "操作成功",
    "data": { ... }
  }
  ```
- **版本控制**: API 路径中包含版本信息或通过 Accept 头指定版本
- **文档化**: 使用 Swagger/OpenAPI 记录所有 API 端点

### 4. 日志管理规范

- **日志级别**:
  - DEBUG: 开发环境中的详细信息
  - INFO: 常规操作信息
  - WARNING: 潜在问题警告
  - ERROR: 错误但不影响系统运行
  - FATAL: 严重错误导致系统无法继续运行
- **日志格式**: 统一的结构化日志格式，包含时间戳、级别、模块、消息和上下文
  ```
  2023-05-15T14:30:45.123Z [INFO] [auth] 用户登录成功 {"user_id": "123", "ip": "192.168.1.1"}
  ```
- **日志轮转**: 按大小或时间自动轮转日志文件，防止单个日志文件过大
- **敏感信息处理**: 确保密码、令牌等敏感信息不会被记录到日志中

### 5. 错误管理规范

- **错误码系统**: 使用统一的错误码体系，便于问题定位和客户端处理
  - 1xxxx: 系统级错误
  - 2xxxx: 业务逻辑错误
  - 3xxxx: 第三方服务错误
- **错误处理流程**:
  - 在发生错误的地方捕获并记录详细信息
  - 向上层返回有意义的错误信息
  - 在 API 层统一格式化错误响应
- **错误日志**: 记录完整的错误堆栈和上下文信息，便于问题排查
- **优雅降级**: 当部分服务不可用时，实现合理的降级策略

### 6. 代码测试规范

- **测试类型**:
  - 单元测试: 测试独立功能单元
  - 集成测试: 测试组件间交互
  - 端到端测试: 测试完整流程
  - 性能测试: 测试系统性能和并发能力
- **测试覆盖率**: 核心业务逻辑的测试覆盖率应达到 80% 以上
- **测试命名**: `TestXxx` 格式，清晰描述测试目的
- **模拟和存根**: 使用 `gomock` 或类似工具模拟外部依赖
- **测试自动化**: 集成到 CI/CD 流程中，自动运行测试

### 7. 性能分析规范

- **性能指标**:
  - 响应时间: API 请求的平均响应时间
  - 吞吐量: 系统每秒处理的请求数
  - 错误率: 请求失败的百分比
  - 资源利用率: CPU、内存、磁盘、网络等资源使用情况
- **分析工具**:
  - `pprof`: Go 内置的性能分析工具
  - `benchstat`: 统计基准测试结果
  - 负载测试: 使用 `wrk`、`hey` 等工具进行负载测试
- **性能测试脚本**: 位于 `internal/test/benchmark/` 目录
- **监控系统**: 使用 Prometheus + Grafana 监控生产环境性能

### 8. 部署维护规范

- **环境隔离**:
  - 开发环境: 用于日常开发和测试
  - 测试环境: 用于集成测试和验收测试
  - 生产环境: 面向最终用户的环境
- **配置管理**:
  - 使用环境变量进行配置
  - 敏感配置使用加密存储
  - 不同环境使用不同的配置文件
- **版本控制**:
  - 使用语义化版本号: `主版本.次版本.修订号`
  - 每个版本有明确的变更日志
- **部署流程**:
  - 代码审查 → 自动化测试 → 构建 → 部署
  - 使用蓝绿部署或金丝雀发布策略
- **监控和告警**:
  - 实时监控系统状态
  - 设置合理的告警阈值
  - 建立明确的问题响应流程

## 许可证

版权所有 © 2024 intercom_http_service 开发团队
