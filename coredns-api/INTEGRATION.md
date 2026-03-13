# CoreDNS 集成指南

## 概述

本项目将 CoreDNS 源码拉取到 GoEdge 项目中，并创建了一个自定义的 API 插件，使 CoreDNS 能够通过 HTTP API 管理 DNS 记录。

## 项目结构

```
coredns-api/
├── coredns/                 # CoreDNS 源码
├── build/                   # 构建输出目录
├── plugin-1.go             # API 插件源代码
├── setup.sh                # 集成脚本
├── start.sh                # 启动脚本
├── Corefile                # CoreDNS 配置文件
├── zones.db                # DNS 区域文件示例
├── README.md               # 使用说明
└── INTEGRATION.md          # 本集成文档
```

## 集成步骤

### 1. 获取 CoreDNS 源码

```bash
cd coredns-api
git clone https://github.com/coredns/coredns.git
```

### 2. 集成 API 插件

运行集成脚本：

```bash
./integrate.sh
```

这个脚本会：
- 将 API 插件复制到 CoreDNS 插件目录
- 更新 plugin.cfg 文件包含 API 插件
- 重新构建 CoreDNS

### 3. 构建 CoreDNS

```bash
cd coredns-api/coredns
go generate && go build -o ../build/coredns-with-api
```

### 4. 配置 CoreDNS

编辑 `Corefile` 配置文件：

```
.:53 {
    errors
    health
    
    # API 插件配置
    api {
        address :8080
        zone_file ./zones.db
        # 可选：ECS 扩展日志开关，默认 off
        # ecs_log on
    }
    
    file ./zones.db
    cache
    forward . 8.8.8.8 1.1.1.1
}
```

### 5. 启动 CoreDNS

```bash
./start.sh
```

## API 功能

### 支持的端点

- `GET /domains` - 获取所有域名列表
- `GET /domains/{domain}/records` - 获取指定域名的 DNS 记录
- `POST /domains/{domain}/records` - 添加新的 DNS 记录
- `PUT /domains/{domain}/records/{id}` - 更新 DNS 记录
- `DELETE /domains/{domain}/records/{id}` - 删除 DNS 记录

### 认证支持

可选 API 密钥认证：

```
api {
    address :8080
    apikey "your-secret-key"
    zone_file ./zones.db
    # 可选：ECS 扩展日志开关，默认 off
    # ecs_log on
}
```

请求时需要包含头部：
```
X-API-Key: your-secret-key
```

## 与 GoEdge 集成

### 1. 在 GoEdge 中配置 CoreDNS 提供商

在 GoEdge 管理面板中：
1. 进入 DNS 提供商管理
2. 添加新的 DNS 提供商
3. 选择 CoreDNS 类型
4. 配置 API 端点：`http://localhost:8080`
5. 设置认证密钥（如果启用）

### 2. 启动流程

1. 启动 CoreDNS：`./start.sh`
2. 确保 CoreDNS 正常运行
3. 在 GoEdge 中配置 DNS 记录
4. GoEdge 将通过 HTTP API 管理 CoreDNS 的 DNS 记录

## 技术细节

### 插件架构

- **插件名称**: `api`
- **包路径**: `github.com/coredns/coredns/plugin/api`
- **HTTP 端口**: 默认 8080
- **DNS 端口**: 默认 53

### 数据存储

- 使用 CoreDNS 原生的 `file` 插件进行 DNS 记录存储
- 区域文件格式兼容标准 DNS 区域文件
- 支持实时重载配置

### 安全性

- 可选的 API 密钥认证
- 建议在生产环境中启用认证
- 考虑使用 HTTPS 和防火墙规则限制访问

## 故障排除

### 常见问题

1. **插件未加载**
   - 检查 plugin.cfg 文件中是否包含 `api:github.com/coredns/coredns/plugin/api`
   - 重新运行 `go generate && go build`

2. **API 无法访问**
   - 检查 CoreDNS 是否正在运行
   - 验证端口 8080 是否被占用
   - 查看 CoreDNS 日志输出

3. **DNS 查询失败**
   - 检查 zones.db 文件格式是否正确
   - 验证 Corefile 配置
   - 检查防火墙设置

### 日志调试

启动 CoreDNS 时添加 `-log` 参数查看详细日志：

```bash
./build/coredns-with-api -conf Corefile -log
```

## 性能优化

- 启用缓存插件提高查询性能
- 调整线程数和连接数限制
- 使用负载均衡器处理高并发请求
- 定期清理和优化区域文件

## 扩展开发

要扩展 API 功能，可以修改 `plugin/api/setup.go` 文件：

1. 添加新的 HTTP 端点处理器
2. 实现更复杂的 DNS 记录操作逻辑
3. 添加监控和统计功能
4. 支持更多 DNS 记录类型

## 部署建议

### 开发环境
- 直接使用 `./start.sh` 脚本启动
- 使用本地文件系统存储区域文件

### 生产环境
- 使用 systemd 服务管理 CoreDNS
- 配置日志轮转和监控
- 启用 TLS 加密 API 通信
- 设置备份和恢复策略
