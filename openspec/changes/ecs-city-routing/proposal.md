## Why

当前上游已经通过 EDNS Client Subnet (ECS) 在 DNS 请求中携带客户端原始网络信息，但现有 GoEdge/CoreDNS 链路仍主要按递归解析器来源做调度，导致跨地市命中错误节点。需要引入“按 ECS 原始 IP 做地市级映射”的能力，提升解析就近性与回源效率。

## What Changes

- 在 CoreDNS 请求处理链中解析 ECS 扩展，提取客户端原始 IP（优先 IPv4 `/32`，兼容 IPv6 前缀）。
- 新增基于原始 IP 的地理映射流程，将 IP 映射到省/市（如 `61.142.56.193 -> 广州市`）。
- 将城市维度结果接入 GoEdge 节点选择，优先返回该城市可用边缘节点；无匹配时按既有策略降级。
- 增加配置开关与观测字段（日志/指标），用于验证 ECS 命中率、城市命中率和降级原因。
- 补充测试与回归覆盖，确保在无 ECS、非法 ECS、城市无节点等场景下行为稳定。

## Capabilities

### New Capabilities
- `ecs-city-routing`: 从 DNS 请求 ECS 扩展提取客户端原始 IP，并基于 IP 库映射地市后执行城市优先的边缘节点调度。

### Modified Capabilities
- None.

## Impact

- Affected code:
  - `coredns-api/coredns/` 中请求解析/插件链路（ECS 读取与上下文透传）。
  - GoEdge DNS 调度与节点选择相关模块（按城市筛选候选节点与降级）。
  - IP 库读取与地理映射相关组件（复用现有 region/city 数据）。
- APIs & config:
  - 可能新增/扩展 DNS 提供商配置项（是否启用 ECS 城市路由、降级策略）。
  - 可能扩展内部调度接口字段（携带 `clientIP`/`cityId` 等上下文）。
- Operational impact:
  - 需要更新发布与配置文档，明确 ECS 生效前提（上游透传、隐私策略、缓存行为）。
