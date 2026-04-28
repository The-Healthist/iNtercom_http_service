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
│   ├── code_standard.md   # 当前项目的 Go 后端开发规范
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
- **RTC服务**: `/api/trtc/*`

## 系统特性

- **自动迁移**: 支持数据库自动迁移，包括alter和drop模式
- **基于角色的访问控制**: 不同角色拥有不同权限
- **安全通信**: 基于JWT的API认证
- **性能优化**:
  - 高效的数据库连接池管理
  - 响应缓存中间件，支持多种缓存策略
  - 多级限流保护，支持IP、路径和自定义限流
- **视频通话集成**: 支持腾讯云TRTC
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

当前仓库以 [`code_standard.md`](./code_standard.md) 作为 Go 后端开发的主规范。新增代码、重构代码、缺陷修复代码，均以该文档为准。

规范文件用于知识库归档、代码评审、日常开发与重构执行。涉及架构、注释、命名、响应、错误、日志、测试等要求时，以该文件为唯一标准来源。

## 许可证

版权所有 © 2024 intercom_http_service 开发团队
