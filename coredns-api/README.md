# CoreDNS API Plugin for GoEdge

This is a CoreDNS plugin that provides HTTP API for DNS record management, designed to work with GoEdge system.

## Features

- RESTful HTTP API for DNS record management
- Support for multiple zones (domains)
- File-based zone storage (compatible with CoreDNS file plugin)
- Authentication support via API keys
- Real-time zone file reloading

## Integration with GoEdge

To integrate this plugin with GoEdge:

1. **Build the plugin**:
   ```bash
   cd coredns-api
   ./build.sh
   ```

2. **Install CoreDNS** (if not already installed):
   ```bash
   # Download CoreDNS
   wget https://github.com/coredns/coredns/releases/download/v1.11.1/coredns_1.11.1_linux_amd64.tgz
   tar xzf coredns_1.11.1_linux_amd64.tgz
   sudo mv coredns /usr/local/bin/
   ```

3. **Configure CoreDNS**:
   - Copy the generated `api.so` plugin to CoreDNS plugins directory
   - Use the provided `Corefile` configuration
   - Ensure zone file directory exists:
     ```bash
     sudo mkdir -p /etc/coredns
     sudo cp zones.db /etc/coredns/
     ```

4. **Start CoreDNS**:
   ```bash
   coredns -conf Corefile
   ```

5. **Configure GoEdge to use CoreDNS**:
   - In GoEdge admin panel, add CoreDNS as a DNS provider
   - Set the API endpoint to `http://localhost:8080`
   - Configure authentication if needed

## API Endpoints

### Get all domains
```
GET /domains
```

### Get records for a domain
```
GET /domains/{domain}/records
```

### Add a record
```
POST /domains/{domain}/records
Content-Type: application/json

{
    "name": "www",
    "type": "A",
    "value": "192.168.1.1",
    "ttl": 3600
}
```

### Update a record
```
PUT /domains/{domain}/records/{id}
Content-Type: application/json

{
    "name": "www",
    "type": "A",
    "value": "192.168.1.2",
    "ttl": 3600
}
```

### Delete a record
```
DELETE /domains/{domain}/records/{id}
```

## Configuration

Add the following to your Corefile:

```
.:53 {
    # CoreDNS plugins
    errors
    health

    # Enable API plugin
    api {
        address :8080
        # Optional: enable authentication
        # apikey "your-secret-key"
        zone_file /etc/coredns/zones.db
        # Optional: ECS extension logging switch, default off
        # ecs_log on
    }

    # File-based zone storage
    file /etc/coredns/zones.db

    cache
    forward . 8.8.8.8 1.1.1.1
}
```

## Building

```bash
# Build the plugin
./build.sh

# The plugin will be compiled as api.so
```

## Testing

```bash
# Test the API endpoints
./test_api.sh
```

## Troubleshooting

1. **Plugin not loading**: Ensure `api.so` is in the correct plugins directory
2. **Permission denied**: Run CoreDNS with appropriate permissions for zone file access
3. **API not responding**: Check if CoreDNS is running and the API port is accessible

## Security Considerations

- Always use authentication (`apikey`) in production environments
- Consider using HTTPS with TLS certificates
- Restrict API access to trusted IP addresses
- Regularly update CoreDNS and the plugin
