# CoreDNS + API CentOS 7.9 独立运行包

## 🚀 快速部署指南

### 1. 构建独立运行包
```bash
# 在开发环境执行
chmod +x build-standalone-centos7.sh
./build-standalone-centos7.sh
```

### 2. 上传到CentOS 7.9服务器
将生成的 `coredns-api-standalone-centos7-YYYYMMDD.tar.gz` 文件上传到目标服务器。

### 3. 一键部署
```bash
# 在CentOS 7.9服务器执行
tar -xzf coredns-api-standalone-centos7-20240211.tar.gz
cd coredns-api-standalone-centos7-20240211
./start.sh
```

## 📦 包内容说明

独立运行包包含以下文件：

```
coredns-api-standalone-centos7-20240211/
├── coredns          # CoreDNS二进制文件（静态编译）
├── Corefile         # 配置文件
├── zones.db         # DNS区域文件
├── start.sh         # 启动脚本
├── stop.sh          # 停止脚本
├── status.sh        # 状态检查脚本
├── test.sh          # 功能测试脚本
└── README.txt       # 使用说明
```

## 🔧 管理命令

| 命令 | 功能 | 说明 |
|------|------|------|
| `./start.sh` | 启动服务 | 启动CoreDNS服务 |
| `./stop.sh` | 停止服务 | 停止CoreDNS服务 |
| `./status.sh` | 查看状态 | 检查服务运行状态 |
| `./test.sh` | 功能测试 | 测试DNS和API功能 |

## 🌐 服务信息

- **DNS服务端口**: 8053 (TCP/UDP)
- **API管理端口**: 18080 (HTTP)
- **日志文件**: coredns.log
- **PID文件**: coredns.pid

## 📋 部署流程

### 步骤1: 解压包
```bash
tar -xzf coredns-api-standalone-centos7-20240211.tar.gz
cd coredns-api-standalone-centos7-20240211
```

### 步骤2: 启动服务
```bash
./start.sh
```

输出示例：
```
启动CoreDNS服务...
✅ CoreDNS启动成功 (PID: 12345)
📊 服务信息:
   • DNS端口: 8053
   • API端口: 18080
   • 日志文件: coredns.log
   • 停止命令: ./stop.sh

🌐 测试命令:
   dig @localhost -p 8053 example.com
   curl http://localhost:18080/domains
```

### 步骤3: 验证功能
```bash
./test.sh
```

### 步骤4: 日常管理
```bash
# 查看状态
./status.sh

# 停止服务
./stop.sh

# 重启服务
./stop.sh && ./start.sh
```

## 🔍 功能验证

### DNS功能测试
```bash
# 查询example.com
dig @localhost -p 8053 example.com

# 查询www.example.com  
dig @localhost -p 8053 www.example.com
```

### API功能测试
```bash
# 获取域名列表
curl http://localhost:18080/domains

# 获取example.com记录
curl http://localhost:18080/domains/example.com/records

# 添加DNS记录
curl -X POST http://localhost:18080/domains/test.com/records \\
  -H "Content-Type: application/json" \\
  -d '{"name":"www","type":"A","value":"192.168.1.100","ttl":3600}'
```

## ⚙️ 自定义配置

### 修改DNS配置
编辑 `Corefile` 文件：
```
.:8053 {
    errors
    health
    api {
        address :18080
        zone_file ./zones.db
    }
    file ./zones.db
    cache
    forward . 8.8.8.8 1.1.1.1
}
```

### 修改DNS记录
编辑 `zones.db` 文件：
```
; 添加新的DNS记录
test    IN      A       192.168.1.200
api     IN      A       192.168.1.201
```

## 🛠️ 故障排除

### 常见问题

#### 1. 端口被占用
```bash
# 检查端口占用
netstat -tlnp | grep -E '(8053|18080)'

# 终止占用进程
pkill -f coredns
```

#### 2. 权限问题
```bash
# 确保所有脚本可执行
chmod +x *.sh
chmod +x coredns
```

#### 3. 服务无法启动
```bash
# 查看详细日志
tail -f coredns.log

# 检查二进制文件
file coredns
./coredns -version
```

#### 4. 防火墙阻止
```bash
# CentOS 7防火墙命令
firewall-cmd --permanent --add-port=8053/tcp
firewall-cmd --permanent --add-port=8053/udp  
firewall-cmd --permanent --add-port=18080/tcp
firewall-cmd --reload
```

### 日志分析

查看实时日志：
```bash
tail -f coredns.log
```

常见日志信息：
- `[INFO] Starting server` - 服务启动成功
- `[ERROR]` - 错误信息
- `address already in use` - 端口被占用

## 🔄 生产环境建议

### 1. 使用systemd服务（可选）
```bash
# 创建systemd服务文件
sudo vi /etc/systemd/system/coredns-api.service

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable coredns-api
sudo systemctl start coredns-api
```

### 2. 配置日志轮转
```bash
# 创建日志轮转配置
sudo vi /etc/logrotate.d/coredns
```

### 3. 监控和告警
- 监控8053和18080端口状态
- 设置进程监控
- 配置日志监控

## 📞 技术支持

### 基础检查清单
1. ✅ 包文件完整
2. ✅ 脚本可执行权限
3. ✅ 端口未被占用
4. ✅ 防火墙配置正确
5. ✅ 查看日志无错误

### 获取帮助
如果遇到问题：
1. 查看 `coredns.log` 日志文件
2. 运行 `./status.sh` 检查状态
3. 执行 `./test.sh` 进行功能测试
4. 检查系统资源使用情况

## 🎯 核心优势

- **零依赖**: 无需安装Go、Docker或其他依赖
- **开箱即用**: 解压即可运行
- **简单管理**: 提供完整的管理脚本
- **兼容性强**: 专为CentOS 7.9优化
- **生产就绪**: 包含完整的运维工具

---

**注意**: 此包专为CentOS 7.9设计，在其他Linux发行版上可能需要调整。