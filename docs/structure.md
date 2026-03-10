# intercom_http_service 项目结构

本文档介绍了 intercom_http_service（对讲机后端服务）的文件目录结构。

## 目录结构概览

```
intercom_http_service/
├── cmd/                    # 应用程序入口点
│   └── server/             # HTTP 服务入口
│       └── main.go         # 主程序入口
├── configs/                # 配置文件
├── docs/                   # 项目文档
├── internal/               # 私有应用程序代码
│   ├── app/                # 应用层
│   │   ├── controllers/    # 控制器
│   │   ├── middleware/     # 中间件
│   │   └── routes/         # 路由
│   ├── domain/             # 领域层
│   │   ├── models/         # 数据模型
│   │   └── services/       # 业务服务
│   ├── error/              # 错误处理
│   │   ├── code/           # 错误码
│   │   └── response/       # 响应格式
│   └── infrastructure/     # 基础设施层
│       ├── config/         # 配置
│       ├── database/       # 数据库
│       └── mqtt/           # MQTT客户端
├── logs/                   # 日志文件
├── pkg/                    # 公共代码包
│   ├── logger/             # 日志工具
│   ├── utils/              # 通用工具函数
│   └── validator/          # 验证工具
├── scripts/                # 脚本文件
│   ├── build.sh            # 构建脚本
│   ├── deploy.sh           # 部署脚本
│   └── migration.sh        # 迁移脚本
├── test/                   # 测试文件
├── .env                    # 环境变量
├── .gitignore              # Git忽略文件
├── go.mod                  # Go模块定义
├── go.sum                  # Go依赖校验和
└── README.md               # 项目说明
```

## 目录说明

### `/cmd`

包含项目的主要应用程序入口。每个应用程序都有自己的目录。

- `server/` - HTTP服务入口
  - `main.go` - 主程序入口点

### `/configs`

配置文件模板或默认配置。

### `/docs`

项目文档，包括API文档、错误处理文档、部署文档等。

### `/internal`

私有应用程序代码，不希望被外部项目导入。

- `app/` - 应用层
  - `controllers/` - 控制器，处理HTTP请求
  - `middleware/` - 中间件，如认证、日志等
  - `routes/` - 路由定义
- `domain/` - 领域层
  - `models/` - 数据模型定义
  - `services/` - 业务逻辑服务
- `error/` - 错误处理
  - `code/` - 错误码定义
  - `response/` - 统一响应格式
- `infrastructure/` - 基础设施层
  - `config/` - 配置管理
  - `database/` - 数据库操作
  - `mqtt/` - MQTT客户端实现

### `/logs`

应用程序生成的日志文件。

### `/pkg`

可以被外部应用安全导入的代码。

- `logger/` - 日志工具
- `utils/` - 通用工具函数
- `validator/` - 数据验证工具

### `/scripts`

各种构建、安装、分析等操作的脚本。

### `/test`

额外的外部测试应用程序和测试数据。

## 设计原则

1. **关注点分离**: 将不同功能的代码分开，便于维护和理解
2. **依赖注入**: 通过依赖注入实现组件间的解耦
3. **分层架构**: 清晰的分层有助于理解代码职责
4. **封装变化**: 将可能变化的部分封装起来，减少修改对系统的影响

## 导入路径约定

- 内部包: `intercom_http_service/internal/...`
- 公共包: `intercom_http_service/pkg/...`
- 主程序: `intercom_http_service/cmd/...` 