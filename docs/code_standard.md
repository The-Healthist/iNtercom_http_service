# intercom_http_service Go 后端开发规范

## 1. 适用范围

本规范适用于 `intercom_http_service` 项目下全部 Go 后端代码，涵盖以下目录:

- `cmd/server`
- `internal/router`
- `internal/handler`
- `internal/service`
- `internal/model`
- `internal/middleware`
- `internal/config`
- `internal/database`
- `internal/errcode`
- `internal/logger`
- `internal/utils`

所有新增代码、重构代码、缺陷修复代码，均应遵循本规范。

## 2. 项目架构规范

### 2.1 分层结构

项目当前采用以下后端分层:

- `cmd/server`: 服务启动、初始化、迁移、系统级装配。
- `internal/router`: 路由注册、中间件挂载、缓存与限流策略编排。
- `internal/handler`: HTTP 请求解析、基础参数校验、响应输出。
- `internal/service`: 业务逻辑、数据库读写、事务处理、第三方服务编排。
- `internal/model`: GORM 模型、分页结构、基础数据结构。
- `internal/middleware`: 认证、限流、缓存、上下文注入等横切能力。
- `internal/config`: 配置读取与配置对象管理。
- `internal/database`: 数据库连接池与数据库初始化。
- `internal/errcode`: 统一响应格式、错误码、错误消息。
- `internal/logger`: 项目统一日志入口。
- `internal/utils`: 通用且无业务状态的工具函数。

### 2.2 依赖方向

代码依赖方向必须保持清晰，统一为:

`router -> handler -> service -> model/database`

约束如下:

- `handler` 不得直接操作数据库。
- `handler` 不得承载完整业务流程。
- `service` 不得直接输出 HTTP 响应。
- `service` 不得依赖 `gin.Context`。
- `utils` 不得承载业务规则。
- `middleware` 不得写资源级业务逻辑。

### 2.3 实现原则

后端实现必须满足以下要求:

- 满足业务目标的前提下，优先采用依赖更少、抽象更少、调用链更短的实现方式。
- 无明确复用价值时，不新增接口层、不新增包装层、不做预设型通用设计。
- 新代码必须与现有项目分层和目录结构保持一致。

## 3. 文件组织与命名规范

### 3.1 包命名

- 包名使用全小写英文单词。
- 包名与目录名保持一致。
- 包名不得使用下划线、混合大小写、无意义缩写。

示例:

- `handler`
- `service`
- `middleware`
- `config`

### 3.2 文件命名

- 文件名统一使用小写字母与下划线。
- 文件名应体现资源或职责。
- 同一资源的 handler、service、model 建议使用一致命名。

示例:

- `admin_controller.go`
- `admin_service.go`
- `mqtt_call_service.go`
- `health_controller.go`

### 3.3 类型命名

- 结构体使用大驼峰命名。
- 请求体统一使用 `XxxRequest`。
- 返回体需要独立复用时使用 `XxxResponse`。
- 领域模型名称与业务实体保持一致。

示例:

- `AdminController`
- `AdminService`
- `CreateAdminRequest`
- `CallRecord`

### 3.4 接口命名

- 接口仅在确有抽象边界、替换需求、测试隔离需求时定义。
- 新代码中的接口命名不得再使用 `InterfaceXxx` 前缀。
- 接口名称应直接表达职责。

推荐命名:

- `AdminService`
- `TokenValidator`
- `MQTTPublisher`

历史代码中已存在的 `InterfaceXxx` 命名可保留，但新代码不得继续扩散。

## 4. 注释规范

### 4.1 文件顶部多行注释规范

以下文件必须在文件顶部增加统一的多行注释，放置于 `package` 之前:

- `cmd/server` 下的入口文件
- `internal/router` 下的路由文件
- `internal/handler` 下的控制器文件
- `internal/service` 下的业务服务文件
- `internal/middleware` 下的中间件文件
- `internal/model` 下的核心模型文件
- `internal/config` 下的配置文件
- `internal/database` 下的数据库文件

要求如下:

- 使用 `/* ... */` 多行注释。
- 内容保持简洁，一般 3 至 6 行。
- 必须说明文件职责、主要内容、边界说明。
- 不得写空泛描述，不得只写“工具类”“处理文件”之类无效说明。

标准格式:

```go
/*
文件职责: 管理员 HTTP 接口处理文件。
主要内容:
1. 管理员列表、详情、创建、更新、删除接口。
2. 请求参数绑定与基础校验。
3. 统一响应输出。
*/
package handler
```

补充要求:

- `internal/utils` 下如文件逻辑简单，可不强制添加。
- 如果文件承担核心业务流程，即使位于 `utils` 或 `test`，也建议添加顶部多行注释。

### 4.2 主要函数编号注释规范

每个文件中的主要函数，必须在函数定义上方增加简短编号注释，便于快速定位。

适用范围:

- 所有导出函数。
- 所有路由注册函数。
- 所有 handler 主处理函数。
- 所有 service 主业务函数。
- 逻辑较长或承担关键路径的私有函数。

格式要求:

- 统一采用 `// 1. 函数说明` 这种格式。
- 同一文件内按出现顺序连续编号。
- 注释内容必须直接描述函数职责。
- 编号注释必须简短，不写冗长说明。

标准格式:

```go
// 1. GetAdmins 获取管理员列表
func (c *AdminController) GetAdmins() {
	// ...
}

// 2. CreateAdmin 创建管理员
func (c *AdminController) CreateAdmin() {
	// ...
}
```

补充要求:

- 新增函数时必须同步维护编号顺序。
- 拆分文件后，编号从 `1` 重新开始。
- 仅包含一两个极短辅助函数的文件，可不对每个辅助函数强制编号，但主函数必须编号。

## 5. Router 规范

`router` 层仅负责 HTTP 路由组织，不承载业务处理。

必须遵守:

- 仅注册路由、路由组、中间件、缓存、限流。
- 不在 `router` 中写数据库查询。
- 不在 `router` 中写具体业务判断。
- 路由路径使用资源名词，保持 REST 风格。

推荐方式:

- 公共路由与鉴权路由分组注册。
- 同一资源的路由集中注册。
- 路由结构应与 `docs/docs_api` 保持一致。

## 6. Handler 规范

`handler` 层负责请求接入与响应输出。

必须遵守:

- 只负责参数提取、参数绑定、基础格式处理、调用 service、返回响应。
- 基础格式处理包括: `TrimSpace`、默认值修正、分页参数边界修正。
- 业务规则校验放在 `service`，不得在 `handler` 中扩展为完整业务流程。
- 响应统一通过 `internal/errcode` 输出。
- 新代码不得直接散落 `c.JSON(...)`。

推荐流程:

1. 读取 `path`、`query`、`body` 参数。
2. 完成基础语法级校验。
3. 调用对应 service。
4. 将错误翻译为统一错误码与响应消息。
5. 输出统一 JSON 响应。

补充要求:

- 请求结构体优先定义在对应 handler 文件中。
- 当请求体、响应体、转换逻辑明显增多时，可拆分为 `*_request.go`、`*_response.go` 或 `*_dto.go`。
- `handler` 中不得返回敏感字段，如密码、密钥、完整 token。

## 7. Service 规范

`service` 层负责承载业务逻辑，是项目主要实现层。

必须遵守:

- 所有数据库读写均在 `service` 层完成。
- 跨表写入、状态流转、补偿操作必须显式控制事务。
- `service` 不得依赖 `gin.Context`、HTTP 状态码、`gin.H`。
- `service` 返回业务结果与 `error`，由上层负责转换响应。
- 原生 SQL 仅用于迁移场景或明确的性能优化场景，并应写明原因。

推荐方式:

- 查询条件显式声明，避免隐式副作用。
- 更新逻辑优先更新必要字段，避免整对象覆盖。
- 唯一性校验、存在性校验、状态迁移校验集中在 `service`。

禁止事项:

- 通过 `panic` 处理运行期业务错误。
- 将 HTTP 语义写入业务错误。
- 将日志、响应、数据库逻辑混写为单个超长流程函数。

## 8. Middleware 规范

`middleware` 用于处理横切逻辑。

必须遵守:

- 仅处理认证、鉴权、上下文透传、限流、缓存、审计等通用能力。
- 统一从请求中提取认证信息并注入上下文。
- 中间件返回错误时，响应格式应尽量与 `errcode` 保持一致。

禁止事项:

- 针对单一业务资源编写耦合型业务中间件。
- 在中间件中编排完整业务流程。

## 9. Model 与数据库规范

### 9.1 Model 规范

- `model` 主要用于承载 GORM 模型与基础数据结构。
- 字段名必须与业务语义一致。
- `json` 标签、`gorm` 标签必须明确。
- 模型中不放置复杂业务流程。

### 9.2 数据库操作规范

- 所有数据库访问必须通过 `service` 层。
- 查询、更新、删除必须控制筛选条件，禁止不带条件的风险操作。
- 迁移逻辑集中在启动流程或专用迁移逻辑中，不得散落在业务 handler 中。
- 涉及批量操作时，应明确影响范围并记录必要日志。

## 10. 响应与错误处理规范

### 10.1 响应格式

项目统一响应格式如下:

```json
{
  "code": 100000,
  "message": "成功",
  "data": {}
}
```

### 10.2 响应规则

- 成功统一使用 `errcode.Success`。
- 参数错误统一使用 `errcode.ParamError` 或 `errcode.FailWithMessage(..., errcode.ErrBind, ...)`。
- 资源不存在统一使用 `errcode.NotFound`。
- 权限错误、限流错误、数据库错误统一走 `errcode` 定义。
- 错误消息必须可读、稳定、可定位，不得暴露内部堆栈、SQL 细节、密钥信息。

### 10.3 错误处理分工

- `handler` 负责错误到响应的转换。
- `service` 负责返回业务错误。
- `middleware` 负责处理鉴权与通用入口错误。
- `config` 与启动阶段可采用 fail-fast，但运行期业务代码不得滥用 `panic`。

## 11. 日志规范

必须遵守:

- 项目统一使用 `internal/logger` 输出日志。
- 新增代码不得继续扩散 `log.Printf` 风格。
- 日志必须记录关键上下文，例如 `admin_id`、`device_id`、`resident_id`、`call_id`。
- 禁止打印密码、JWT、密钥、数据库密码、完整认证信息。

记录原则:

- 成功路径日志从简。
- 失败路径日志准确。
- 同一错误只在最合适的一层记录一次，避免重复刷屏。

## 12. 配置规范

必须遵守:

- 所有环境变量统一在 `internal/config/config.go` 中读取。
- 业务代码、中间件、handler、service 中不得直接使用 `os.Getenv(...)`。
- 必填配置使用显式必填策略。
- 默认值仅用于本地可控场景，生产配置必须明确给出。

## 13. 文件规模与复杂度规范

建议控制标准如下:

- 单个函数尽量控制在 80 行以内。
- 单个文件尽量控制在 400 行以内。
- 同一文件中若职责明显分叉，必须拆分。

拆分优先级如下:

1. 先拆私有辅助函数。
2. 再按动作拆分文件，例如查询类、写入类、流程类。
3. 最后按子域拆 service。

结合当前项目现状，以下文件后续新增功能前应优先拆分:

- `internal/service/mqtt_call_service.go`
- `internal/handler/device_controller.go`
- `internal/handler/staff_controller.go`
- `internal/handler/building_controller.go`

要求如下:

- 对上述文件新增逻辑时，禁止继续无边界追加。
- 新增代码应优先落入拆分后的新文件。

## 14. 测试与文档规范

必须遵守:

- 新增业务逻辑时，至少补充正常路径测试。
- 修复缺陷时，至少补充回归测试。
- 修改 API 请求或响应结构时，必须同步更新 `docs/docs_api`。
- 修改错误码、错误语义、错误响应方式时，必须同步更新 `docs/docs_error`。

提交前最低检查:

```bash
gofmt -w ./...
go test ./...
```

如项目后续启用 `golangci-lint`，则增加:

```bash
golangci-lint run
```

## 15. 开发执行清单

每次新增或修改后端代码时，必须检查以下事项:

1. 路由是否仅承担注册职责。
2. handler 是否仅承担接入与响应职责。
3. service 是否完整承载业务与数据库逻辑。
4. 响应是否统一走 `errcode`。
5. 是否遵守文件顶部多行注释规范。
6. 是否遵守主要函数编号注释规范。
7. 是否避免引入无明确收益的新抽象层。
8. 是否同步补充测试与文档。
9. 对于logger的包和errcode的包的封装，按照业务判断是否需要封装即可。