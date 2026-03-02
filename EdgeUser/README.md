# EdgeUser 模块

EdgeUser是一个独立的用户管理模块，提供用户相关的HTTP API接口。该模块不直接操作数据库，而是通过gRPC调用EdgeAPI的现有服务接口。

## 功能特性

- ✅ 用户登录认证
- ✅ 用户注册
- ✅ 用户信息查询和更新
- ✅ 访问密钥管理
- ✅ RESTful API设计
- ✅ CORS支持
- ✅ 优雅关闭

## 架构设计

EdgeUser模块采用轻量级设计，主要职责包括：

1. **HTTP API网关**：提供RESTful API接口
2. **gRPC客户端**：连接EdgeAPI服务
3. **请求转发**：将HTTP请求转换为gRPC调用
4. **响应处理**：将gRPC响应转换为HTTP响应

## 项目结构

```
EdgeUser/
├── cmd/edge-user/main.go          # 主程序入口
├── internal/
│   ├── api/user_controller.go     # HTTP控制器
│   ├── const/const.go             # 常量定义
│   └── rpc/client.go              # gRPC客户端封装
├── go.mod                         # 依赖管理
└── README.md                      # 说明文档
```

## 快速开始

### 1. 环境要求

- Go 1.22+
- EdgeAPI服务运行在localhost:8003

### 2. 编译运行

```bash
# 进入EdgeUser目录
cd EdgeUser

# 下载依赖
go mod tidy

# 编译
go build -o edge-user ./cmd/edge-user

# 运行
./edge-user
```

### 3. 验证服务

```bash
# 健康检查
curl http://localhost:8081/api/v1/health

# 用户登录
curl -X POST http://localhost:8081/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"password123"}'
```

## API接口

### 用户管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/users/login` | 用户登录 |
| POST | `/api/v1/users/register` | 用户注册 |
| GET | `/api/v1/users/{userId}` | 获取用户信息 |
| PUT | `/api/v1/users/info` | 更新用户信息 |

### 访问密钥管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/users/access-keys` | 创建访问密钥 |
| GET | `/api/v1/users/access-keys` | 列出访问密钥 |
| DELETE | `/api/v1/users/access-keys/{accessKeyId}` | 删除访问密钥 |

### 系统管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/health` | 健康检查 |

## 配置说明

### gRPC连接配置

在 `internal/const/const.go` 中配置EdgeAPI服务地址：

```go
const (
    // EdgeAPIAddress EdgeAPI服务地址
    EdgeAPIAddress = "localhost:8003"
    
    // RequestTimeout 请求超时时间（秒）
    RequestTimeout = 30
)
```

### HTTP服务器配置

在 `cmd/edge-user/main.go` 中配置HTTP服务器：

```go
server := &http.Server{
    Addr:    ":8081",  // 监听端口
    Handler: router,
}
```

## 集成说明

EdgeUser模块与现有系统的集成方式：

1. **数据源**：通过gRPC调用EdgeAPI服务
2. **认证授权**：使用EdgeAPI现有的认证机制
3. **业务逻辑**：复用EdgeAPI的业务逻辑
4. **数据一致性**：确保与EdgeAPI的数据同步

## 开发指南

### 添加新的API接口

1. 在 `internal/api/user_controller.go` 中添加新的控制器方法
2. 在 `cmd/edge-user/main.go` 中注册新的路由
3. 通过gRPC客户端调用对应的EdgeAPI服务

### 错误处理

- 使用标准的HTTP状态码
- 返回统一的错误格式
- 记录详细的错误日志

### 性能优化

- 使用连接池管理gRPC连接
- 实现请求缓存机制
- 监控API响应时间

## 部署说明

### Docker部署

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o edge-user ./cmd/edge-user

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/edge-user .
EXPOSE 8081
CMD ["./edge-user"]
```

### 系统服务

创建systemd服务文件：

```ini
[Unit]
Description=EdgeUser Service
After=network.target

[Service]
Type=simple
User=edgeuser
WorkingDirectory=/opt/edgeuser
ExecStart=/opt/edgeuser/edge-user
Restart=always

[Install]
WantedBy=multi-user.target
```

## 监控和日志

- 使用Prometheus进行指标收集
- 集成Grafana进行可视化监控
- 结构化日志输出到stdout
- 支持日志轮转和归档

## 故障排除

### 常见问题

1. **gRPC连接失败**：检查EdgeAPI服务是否正常运行
2. **端口冲突**：修改默认端口8081
3. **依赖缺失**：运行 `go mod tidy` 下载依赖

### 调试技巧

- 启用Gin的调试模式：`gin.SetMode(gin.DebugMode)`
- 查看详细的gRPC调用日志
- 使用Postman测试API接口

## 贡献指南

欢迎提交Issue和Pull Request来改进EdgeUser模块。