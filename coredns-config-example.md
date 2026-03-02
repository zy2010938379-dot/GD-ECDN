# CoreDNS 配置和API网关实现

要让GoEdge能够与CoreDNS集成，您需要部署一个API网关来提供HTTP API接口。以下是完整的解决方案。

## 方案一：使用提供的Go语言API网关

### 1. CoreDNS基础配置 (Corefile)

```
.:53 {
    # 基础插件
    errors
    health
    
    # 文件方式管理DNS记录
    file /etc/coredns/zones.db
    
    # 自动重新加载zone文件
    reload 10s
    
    # 缓存
    cache
    
    # 转发到上游DNS
    forward . 8.8.8.8 1.1.1.1
}
```

### 2. Zone文件格式 (zones.db)

```
; 示例zone文件
$ORIGIN example.com.
$TTL 3600

@       IN      SOA     ns1.example.com. admin.example.com. (
                        2024010101 ; serial
                        3600       ; refresh
                        1800       ; retry
                        604800     ; expire
                        86400 )    ; minimum

@       IN      NS      ns1.example.com.
@       IN      NS      ns2.example.com.

@       IN      A       192.168.1.1
www     IN      A       192.168.1.1
mail    IN      A       192.168.1.2
```

### 3. 部署API网关

我们提供了一个简单的Go语言API网关，用于管理CoreDNS的zone文件：

```bash
# 下载并编译API网关
git clone https://github.com/your-repo/coredns-api-gateway.git
cd coredns-api-gateway
go build -o coredns-api

# 启动API网关
./coredns-api --config config.yaml
```

## 方案二：使用Python Flask API网关

### 1. 安装依赖

```bash
pip install flask flask-httpauth
```

### 2. Python API网关代码 (coredns_api.py)

```python
#!/usr/bin/env python3
from flask import Flask, request, jsonify
from flask_httpauth import HTTPTokenAuth
import os
import re
import tempfile
import shutil

app = Flask(__name__)
auth = HTTPTokenAuth(scheme='Bearer')

# 配置
ZONE_FILE_PATH = '/etc/coredns/zones.db'
API_TOKENS = {'your-api-key-here': 'admin'}

@auth.verify_token
def verify_token(token):
    return API_TOKENS.get(token)

class ZoneFileParser:
    def __init__(self, zone_file_path):
        self.zone_file_path = zone_file_path
    
    def get_domains(self):
        """获取所有域名列表"""
        domains = []
        try:
            with open(self.zone_file_path, 'r') as f:
                content = f.read()
                # 查找所有$ORIGIN定义
                origins = re.findall(r'\$ORIGIN\s+(\S+)\.', content)
                domains.extend(origins)
        except FileNotFoundError:
            pass
        return domains
    
    def get_records(self, domain):
        """获取指定域名的记录"""
        records = []
        try:
            with open(self.zone_file_path, 'r') as f:
                lines = f.readlines()
            
            in_domain = False
            for line in lines:
                line = line.strip()
                if not line or line.startswith(';'):
                    continue
                
                if f'$ORIGIN {domain}.' in line:
                    in_domain = True
                    continue
                
                if in_domain and line.startswith('$ORIGIN'):
                    break
                
                if in_domain and not line.startswith('$'):
                    # 解析DNS记录
                    parts = line.split()
                    if len(parts) >= 4:
                        record = {
                            'name': parts[0],
                            'type': parts[2],
                            'ttl': parts[1] if parts[1].isdigit() else 3600,
                            'value': ' '.join(parts[3:])
                        }
                        records.append(record)
            
        except FileNotFoundError:
            pass
        return records
    
    def add_record(self, domain, record_data):
        """添加记录"""
        # 创建临时文件
        temp_file = tempfile.NamedTemporaryFile(mode='w', delete=False)
        
        try:
            with open(self.zone_file_path, 'r') as f:
                content = f.read()
            
            # 检查域名是否存在
            if f'$ORIGIN {domain}.' not in content:
                # 添加新的zone
                new_zone = f'''
; {domain} zone
$ORIGIN {domain}.
$TTL 3600

@       IN      SOA     ns1.{domain}. admin.{domain}. (
                        2024010101 ; serial
                        3600       ; refresh
                        1800       ; retry
                        604800     ; expire
                        86400 )    ; minimum

@       IN      NS      ns1.{domain}.
@       IN      NS      ns2.{domain}.
'''
                content += new_zone
            
            # 添加记录
            record_line = f"{record_data['name']}\t{record_data.get('ttl', 3600)}\tIN\t{record_data['type']}\t{record_data['value']}\n"
            
            # 在zone的适当位置插入记录
            pattern = f'(\\$ORIGIN {domain}\\.\\n.*?\\n)(?=\\$ORIGIN|\\Z)'
            replacement = f'\\1{record_line}'
            content = re.sub(pattern, replacement, content, flags=re.DOTALL)
            
            # 写入临时文件
            temp_file.write(content)
            temp_file.close()
            
            # 替换原文件
            shutil.move(temp_file.name, self.zone_file_path)
            
            return True
        except Exception as e:
            if os.path.exists(temp_file.name):
                os.unlink(temp_file.name)
            raise e

zone_parser = ZoneFileParser(ZONE_FILE_PATH)

@app.route('/domains', methods=['GET'])
@auth.login_required
def get_domains():
    """获取所有域名列表"""
    domains = zone_parser.get_domains()
    return jsonify({'domains': domains})

@app.route('/domains/<domain>/records', methods=['GET'])
@auth.login_required
def get_records(domain):
    """获取指定域名的记录"""
    records = zone_parser.get_records(domain)
    return jsonify({'records': records})

@app.route('/domains/<domain>/records', methods=['POST'])
@auth.login_required
def add_record(domain):
    """添加记录"""
    data = request.get_json()
    
    required_fields = ['name', 'type', 'value']
    for field in required_fields:
        if field not in data:
            return jsonify({'error': f'Missing required field: {field}'}), 400
    
    try:
        zone_parser.add_record(domain, data)
        return jsonify({'message': 'Record added successfully'})
    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080, debug=True)
```

### 3. 启动Python API网关

```bash
# 启动API网关
python3 coredns_api.py

# 或者使用生产服务器
pip install gunicorn
gunicorn -b 0.0.0.0:8080 coredns_api:app
```

## 方案三：使用Shell脚本 + Webhook

### 1. 创建管理脚本 (manage_zone.sh)

```bash
#!/bin/bash

ZONE_FILE="/etc/coredns/zones.db"
BACKUP_DIR="/var/backups/coredns"

# 备份zone文件
backup_zone_file() {
    mkdir -p "$BACKUP_DIR"
    cp "$ZONE_FILE" "$BACKUP_DIR/zones.db.$(date +%Y%m%d%H%M%S)"
}

# 添加记录
add_record() {
    local domain=$1
    local name=$2
    local type=$3
    local value=$4
    local ttl=${5:-3600}
    
    backup_zone_file
    
    # 检查域名是否存在
    if ! grep -q "\\$ORIGIN $domain\." "$ZONE_FILE"; then
        # 添加新的zone
        cat >> "$ZONE_FILE" << EOF

; $domain zone
\$ORIGIN $domain.
\$TTL 3600

@       IN      SOA     ns1.$domain. admin.$domain. (
                        2024010101 ; serial
                        3600       ; refresh
                        1800       ; retry
                        604800     ; expire
                        86400 )    ; minimum

@       IN      NS      ns1.$domain.
@       IN      NS      ns2.$domain.
EOF
    fi
    
    # 添加记录
    echo "$name\t$ttl\tIN\t$type\t$value" >> "$ZONE_FILE"
    
    # 重新加载CoreDNS
    pkill -SIGHUP coredns
}

case "$1" in
    add)
        add_record "$2" "$3" "$4" "$5" "$6"
        ;;
    *)
        echo "Usage: $0 add <domain> <name> <type> <value> [ttl]"
        exit 1
        ;;
esac
```

### 2. 创建Webhook接口

使用简单的HTTP服务器调用shell脚本：

```python
#!/usr/bin/env python3
from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import subprocess
import os

class CoreDNSHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        data = json.loads(post_data)
        
        # 验证API密钥
        api_key = self.headers.get('Authorization', '').replace('Bearer ', '')
        if api_key != 'your-api-key-here':
            self.send_response(401)
            self.end_headers()
            return
        
        try:
            # 调用shell脚本添加记录
            cmd = [
                '/path/to/manage_zone.sh', 'add',
                data['domain'], data['name'], data['type'], data['value']
            ]
            if 'ttl' in data:
                cmd.append(str(data['ttl']))
            
            result = subprocess.run(cmd, capture_output=True, text=True)
            
            if result.returncode == 0:
                self.send_response(200)
                self.send_header('Content-type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({'message': 'Record added successfully'}).encode())
            else:
                self.send_response(500)
                self.end_headers()
                self.wfile.write(json.dumps({'error': result.stderr}).encode())
        except Exception as e:
            self.send_response(500)
            self.end_headers()
            self.wfile.write(json.dumps({'error': str(e)}).encode())
    
    def do_GET(self):
        self.send_response(404)
        self.end_headers()

if __name__ == '__main__':
    server = HTTPServer(('0.0.0.0', 8080), CoreDNSHandler)
    server.serve_forever()
```

## GoEdge中的配置

在GoEdge中添加CoreDNS服务商时，需要配置以下参数：

- **URL**: API网关地址，例如：`http://localhost:8080`
- **API Key**: 如果在API网关中配置了认证

## HTTP API接口说明

所有方案都提供相同的REST API接口：

- `GET /domains` - 获取所有域名列表
- `GET /domains/{domain}/records` - 获取指定域名的记录
- `POST /domains/{domain}/records` - 添加记录

## 注意事项

1. **权限管理**: 确保API网关有权限修改CoreDNS的zone文件
2. **安全性**: 在生产环境中启用API认证
3. **备份**: 修改zone文件前自动备份
4. **重载**: 修改后向CoreDNS发送SIGHUP信号使其重载配置
5. **错误处理**: 实现完善的错误处理和日志记录