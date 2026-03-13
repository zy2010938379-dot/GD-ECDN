# 会话记录（2026-03-10）

## 背景

- 目标：理解 CDN 基础，并分析本仓库（CoreDNS + GoEdge）里 ECS 地市路由相关实现。
- 用户背景：Java Web 开发者，刚接触 CDN 和 Go。

## 本次调研内容

1. 阅读项目结构与关键文档：
   - `coredns-config-example.md`
   - `coredns-config-example-1.md`
   - `coredns-api/README.md`
   - `coredns-api/INTEGRATION.md`
   - `openspec/changes/ecs-city-routing/` 下变更文档
2. 追踪 ECS/地市路由代码路径：
   - CoreDNS 请求侧 ECS 解析
   - CoreDNS `geoip` 插件与 GoEdge 城市映射
   - 基于 `metadata`/`view` 的路由选择
3. 追踪 GoEdge 与 CoreDNS 集成路径：
   - `EdgeAPI/internal/dnsclients/provider_coredns.go`
   - `coredns-api/coredns/plugin/api/api.go`
4. 执行关键测试验证：
   - `go test ./request -run EffectiveClientIP`
   - `go test ./plugin/geoip`
   - `go test ./test -run TestGeoIPECSCityRoutingCityFirstAndFallback`

## 关键技术发现

### 1) ECS effective client IP 已在 CoreDNS 请求层实现

- 文件：`coredns-api/coredns/request/client_ip.go`
- 行为：
  - 开启时优先使用 ECS。
  - ECS 缺失/异常时回退到 resolver source IP。
  - 支持 IPv4 与 IPv6 前缀。
  - 输出 `source` 与 `fallback reason`。

### 2) GoEdge 城市映射目前在 CoreDNS `geoip` 插件路径中实现

- 文件：
  - `coredns-api/coredns/plugin/geoip/geoip.go`
  - `coredns-api/coredns/plugin/geoip/goedge_city.go`
  - `coredns-api/coredns/plugin/geoip/setup.go`
- 行为：
  - `goedge-city` 选项会加载 `EdgeCommon/pkg/iplibrary`。
  - 将 `effective client IP -> country/province/city/provider`。
  - 写入 `metadata`，如 `geoip/goedge/city/id`、`geoip/goedge/city/name`。
  - 支持 `ecs-fallback` 策略：`resolver-ip|disabled`。

### 3) 路由决策由 `metadata + view` 完成

- 文件：
  - `coredns-api/coredns/core/dnsserver/server.go`
  - `coredns-api/coredns/test/geoip_ecs_city_routing_test.go`
- 行为：
  - 在 `view` 过滤前先收集 `metadata`。
  - `view expr` 可匹配 `geoip/goedge/city/id` 选择城市专属 server block。
  - 默认 server block 作为 fallback。

### 4) GoEdge 对接 CoreDNS API 使用 `/zones` 风格 endpoints

- 文件：
  - `EdgeAPI/internal/dnsclients/provider_coredns.go`
  - `coredns-api/coredns/plugin/api/api.go`
- 行为：
  - EdgeAPI 通过 `/zones` 与 `/zones/{domain}/records` 调用。
  - CoreDNS API 插件提供对应 `/zones` 接口。
  - 记录更新采用 `delete + add`。

### 5) IP 库来源与映射模型

- 文件：
  - `EdgeCommon/pkg/iplibrary/default_ip_library.go`
  - `EdgeCommon/pkg/iplibrary/reader_result.go`
- 行为：
  - 通过 `InitDefault` 加载内嵌 DB（`internal-ip-library.db`）。
  - 查询结果包含 `Country/Province/City/Provider` 的 ID 与名称。

## 验证结果

- 已通过：
  - `coredns-api/coredns/request` 测试（ECS effective IP 解析）
  - `coredns-api/coredns/plugin/geoip` 测试
  - E2E 城市路由测试（`TestGeoIPECSCityRoutingCityFirstAndFallback`）
- 备注：
  - E2E 测试首次在 sandbox 因 listen 权限失败，提升权限后通过。

## 当前缺口/风险

1. 文档不一致：
   - 仍有部分文档/脚本写的是 `/domains`，但当前 provider/plugin 实现使用 `/zones`。
2. `coredns-api/start.sh` 为空文件。
3. `openspec/changes/ecs-city-routing/tasks.md` 仍有 1 项未完成：
   - 5.1 在测试环境联调上游 ECS passthrough 并验证效果。

## 建议的学习顺序（本仓库）

1. CoreDNS request + plugin chain 基础：
   - `request/request.go`
   - `request/client_ip.go`
   - `plugin/metadata/*`
   - `plugin/view/*`
2. ECS 地市路由实现：
   - `plugin/geoip/*`
   - `test/geoip_ecs_city_routing_test.go`
3. GoEdge DNS provider 集成：
   - `EdgeAPI/internal/dnsclients/provider_interface.go`
   - `EdgeAPI/internal/dnsclients/provider_coredns.go`
   - `coredns-api/coredns/plugin/api/api.go`
4. IP library 内部机制：
   - `EdgeCommon/pkg/iplibrary/*`

## 下一步实操建议

- 搭一个最小可运行实验环境：
  - 使用 `geoip { edns-subnet ecs-fallback resolver-ip goedge-city }` 启动 CoreDNS。
  - 配置一个城市专属 `view` block + 一个默认 fallback block。
  - 用 `dig +subnet=<test-ip>/32` 做验证。
