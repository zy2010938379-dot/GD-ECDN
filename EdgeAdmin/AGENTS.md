# AGENTS.md - EdgeAdmin Repository Guide

This document provides essential information for AI agents working with the EdgeAdmin codebase, part of the GoEdge CDN & WAF system.

## Project Overview

**EdgeAdmin** is the administrative web panel for GoEdge, a free open-source CDN & WAF system. It provides a management interface for clusters, nodes, servers, DNS, users, and WAF configurations.

- **Language**: Go 1.24.0
- **Web Framework**: TeaGo (github.com/iwind/TeaGo)
- **Communication**: gRPC to EdgeAPI backend
- **Frontend**: Traditional server-rendered views with Vue.js components
- **Database**: MySQL (managed by EdgeAPI)

## Directory Structure

```
EdgeAdmin/
├── cmd/edge-admin/         # Main application entry point
│   └── main.go            # CLI commands and app initialization
├── internal/
│   ├── apps/              # Application lifecycle and directives
│   ├── configs/           # Configuration loading and management
│   ├── const/             # Constants and build info
│   ├── encrypt/           # Encryption utilities
│   ├── errors/            # Error definitions
│   ├── events/            # Event system
│   ├── gen/               # Code generation
│   ├── goman/             # Goroutine management
│   ├── nodes/             # Admin node lifecycle
│   ├── oplogs/            # Operation logging
│   ├── rpc/               # gRPC client for EdgeAPI communication
│   ├── setup/             # Installation utilities
│   ├── tasks/             # Background tasks
│   ├── ttlcache/          # In-memory TTL cache
│   ├── utils/             # General utility functions
│   ├── web/
│   │   ├── actions/       # Web request handlers (MVC controllers)
│   │   │   └── default/   # Action implementations
│   │   │       ├── users/         # User management
│   │   │       ├── clusters/      # Node cluster management
│   │   │       ├── servers/       # HTTP/TCP/UDP server configs
│   │   │       ├── dns/           # DNS management
│   │   │       ├── settings/      # System settings
│   │   │       ├── login/         # Authentication
│   │   │       └── ...
│   │   └── helpers/       # View helper functions
│   └── waf/               # WAF injection detection (libinjection)
├── web/
│   ├── views/@default/    # HTML templates matching action structure
│   └── public/            # Static assets (CSS, JS, fonts)
│       ├── js/
│       ├── css/
│       └── components/    # Vue.js components
├── docker/                # Docker configuration
├── configs/               # Runtime configuration files (generated)
└── go.mod                # Go module dependencies
```

## Essential Commands

### Build and Run

```bash
# Build the application
go build -o edge-admin ./cmd/edge-admin

# Run directly
./edge-admin

# Run with specific commands
./edge-admin start        # Start the service
./edge-admin stop         # Stop the service
./edge-admin restart      # Restart the service
./edge-admin daemon       # Run as daemon (auto-restart)
./edge-admin service      # Register as systemd service
```

### Development Commands

```bash
# Switch between environments (requires running service)
./edge-admin dev          # Switch to development mode
./edge-admin prod         # Switch to production mode

# Code generation
./edge-admin generate     # Generate code (see internal/gen/)

# Reset configuration
./edge-admin reset        # Reset API configuration

# Recovery mode
./edge-admin recover      # Enter recovery mode

# Demo mode
./edge-admin demo         # Toggle demo mode (limits write operations)

# Security
./edge-admin security.reset  # Reset security configuration
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./internal/utils

# Run specific test
go test -run TestExtractIP ./internal/utils
```

### Linting

```bash
# Run golangci-lint (uses .golangci.yaml config)
golangci-lint run

# Lint specific package
golangci-lint run ./internal/web/actions/default/users
```

### Docker

```bash
# Build Docker image
cd docker
./build.sh        # Uses docker build --no-cache

# Run container
./run.sh          # Exposes ports 7788, 8001, 3306

# Manual build
docker build --no-cache -t goedge/edge-admin:latest .
docker run -d -p 7788:7788 -p 8001:8001 -p 3306:3306 --name edge-admin goedge/edge-admin:latest
```

### Upgrade

```bash
# Upgrade from official site
./edge-admin upgrade

# Upgrade from specific URL
./edge-admin upgrade --url=https://example.com/version.zip
```

## Code Patterns and Conventions

### Web Actions (Controllers)

Actions follow the TeaGo MVC pattern with these conventions:

**Location**: `internal/web/actions/default/[module]/`

**File Naming**:
- `index.go` - List view
- `create.go` - Create page
- `createPopup.go` - Create in modal/popup
- `update.go` - Update page
- `updatePopup.go` - Update in modal/popup
- `delete.go` - Delete action
- `init.go` - Package initialization

**Action Structure**:
```go
package users

import (
    "github.com/TeaOSLab/EdgeAdmin/internal/web/actions/actionutils"
    "github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
    "github.com/iwind/TeaGo/actions"
)

type IndexAction struct {
    actionutils.ParentAction
}

func (this *IndexAction) Init() {
    this.Nav("mainMenu", "tab", "firstMenu")
}

func (this *IndexAction) RunGet(params struct {
    Keyword string
    Offset  int64  `default:"0"`
    Size    int64  `default:"20"`
}) {
    // Fetch data via RPC
    usersResp, err := this.RPC().UserRPC().ListEnabledUsers(
        this.AdminContext(),
        &pb.ListEnabledUsersRequest{
            Keyword: params.Keyword,
            Offset:  params.Offset,
            Size:    params.Size,
        },
    )
    if err != nil {
        this.ErrorPage(err)
        return
    }

    // Set view data
    this.Data["users"] = usersResp.Users

    // Render view
    this.Show()
}
```

**Key Patterns**:
- All actions embed `actionutils.ParentAction`
- `Init()` sets up navigation context
- `RunGet(params struct{...})` handles GET requests
- `RunPost(params struct{...})` handles POST requests
- Use `params.Must.Field().Require()` for validation
- Access backend via `this.RPC().[Service]RPC()` methods
- Use `this.ErrorPage(err)` for error handling
- Use `this.Data["key"] = value` to pass data to views
- Call `this.Show()` to render the template

**Error Handling**:
```go
// Standard error handling
if err != nil {
    this.ErrorPage(err)
    return
}

// Field-specific error (for form validation)
this.FailField("fieldName", "error message")

// General failure message
this.Fail("error message")

// Not found
this.NotFound("ResourceName", itemId)
```

**Pagination**:
```go
page := this.NewPage(totalCount)
this.Data["page"] = page.AsHTML()
```

**Logging**:
```go
// Info log (automatically tracks success/failure based on response code)
defer func() {
    this.CreateLogInfo(codes.User_LogCreateUser, userId)
}()

// Custom log level
this.CreateLog(oplogs.LevelInfo, codes.User_LogUpdateUser, userId)
```

### RPC Communication

All backend communication goes through gRPC clients in `internal/rpc/`:

```go
// Get RPC client
rpcClient := this.RPC()

// Access various services
userClient := rpcClient.UserRPC()
nodeClusterClient := rpcClient.NodeClusterRPC()
serverClient := rpcClient.ServerRPC()

// Create admin context (includes authentication)
ctx := this.AdminContext()

// Make RPC call
resp, err := rpcClient.UserRPC().CreateUser(ctx, &pb.CreateUserRequest{
    Username: "testuser",
    // ...
})
```

**Available RPC Clients**: Look at `internal/rpc/rpc_client.go` for all available services. Common ones:
- `UserRPC()` - User management
- `ServerRPC()` - HTTP/TCP/UDP servers
- `NodeClusterRPC()` - Node clusters
- `NodeRPC()` - Individual nodes
- `SSLCertRPC()` - SSL certificates
- `DNSDomainRPC()` - DNS domains
- `HTTPFirewallPolicyRPC()` - WAF policies

### Configuration

**API Config** (`configs/api_admin.yaml`):
```yaml
rpc:
  endpoints:
    - "http://127.0.0.1:8001"  # EdgeAPI gRPC endpoint
  disableUpdate: false
nodeId: "admin-node-id"
secret: "shared-secret-key"
```

**Server Config** (`configs/server.yaml`):
```yaml
env: prod  # or "dev"

http:
  "on": true
  listen:
    - "0.0.0.0:7788"

https:
  "on": false
  listen:
    - "0.0.0.0:443"
  cert: ""
  key: ""
```

**Load configs in code**:
```go
import "github.com/TeaOSLab/EdgeAdmin/internal/configs"

apiConfig, err := configs.LoadAPIConfig()
adminConfig, err := configloaders.LoadAdminUIConfig()
```

### Testing Patterns

Tests follow standard Go conventions:

```go
package utils

import (
    "testing"
)

func TestExtractIP(t *testing.T) {
    result := ExtractIP("192.168.1.100")
    t.Log(result)
}

func TestExtractIP_CIDR(t *testing.T) {
    result := ExtractIP("192.168.2.100/24")
    t.Log(result)
}
```

**Test files**:
- Located alongside source files: `*_test.go`
- Use table-driven tests for multiple cases
- Focus on testing utility functions and RPC helpers
- Most logic is in RPC layer, so unit tests are limited

### Frontend Views

**HTML Templates**: `web/views/@default/[module]/`

**Naming Convention**: Matches action files (e.g., `index.html` for `IndexAction`)

**Layout System**:
- `@layout.html` - Main layout wrapper
- `@menu.html` - Sidebar menu
- `@left_menu.html` - Left navigation
- `@popup.html` - Popup/iframe layout

**Components**: Vue.js components in `web/public/js/components/`

**Accessing backend data**:
```html
{{users}}
{{.page}}
```

**Language**:
```html
{{.lang "AdminUsers_Username"}}
```

### Naming Conventions

- **Files**: lowercase_with_underscores (e.g., `user_manager.go`)
- **Packages**: lowercase, single word when possible (e.g., `utils`, `configs`)
- **Actions**: PascalCase ending with `Action` (e.g., `IndexAction`, `CreatePopupAction`)
- **Functions/Methods**: PascalCase for exported, camelCase for unexported
- **Variables/Constants**: PascalCase for exported constants, camelCase for locals
- **Database/Config**: snake_case for YAML keys (e.g., `rpc.endpoints`)

## Important Gotchas

### Dependency Management

- **EdgeCommon is a local dependency**: The module uses `replace github.com/TeaOSLab/EdgeCommon => ../EdgeCommon`
- **EdgeCommon must exist** in the parent directory or the build will fail
- Check `go.mod` for the exact path if you encounter import errors

### Action Registration

- **All action packages must be imported** in `internal/web/import.go`
- Adding a new action package requires updating this import file
- Use blank imports: `_ "github.com/TeaOSLab/EdgeAdmin/internal/web/actions/default/newmodule"`

### Context Management

- **Each request needs a fresh context**: Always call `this.AdminContext()` for RPC calls
- **Never share context between requests**
- Admin ID is automatically included in the context via session

### RPC Client Lifecycle

- **RPC client is shared**: `rpc.SharedRPC()` returns a singleton
- **Connections are pooled**: The client manages multiple gRPC connections
- **Local endpoints are auto-detected**: If `RPCEndpoints` contains a local IP, it's replaced with 127.0.0.1

### Security

- **CSRF protection**: Forms need `params.CSRF *actionutils.CSRF` in POST handlers
- **Demo mode check**: Write operations should check `teaconst.IsDemoMode` and fail with `teaconst.ErrorDemoOperation`
- **Admin authentication**: All actions automatically check admin authentication via session
- **Secret key generation**: Generated at runtime and stored in temp file (`/tmp/edge-admin-secret.tmp`)

### Configuration Paths

The app searches configs in multiple locations (in order):
1. `configs/api_admin.yaml` (local)
2. `~/.edge-admin/api_admin.yaml` (home directory)
3. `/etc/edge-admin/api_admin.yaml` (system config)

When a config is found, it's automatically copied to `configs/` directory.

### Linter Configuration

**Many linters are disabled** in `.golangci.yaml`:
- Check the `disable:` list before adding new lint rules
- Common disabled linters include: `gosec`, `exhaustivestruct`, `godot`, `lll`
- Don't re-enable linters without understanding why they were disabled

### Database Access

- **No direct database access**: All data access goes through EdgeAPI via gRPC
- **EdgeAPI handles MySQL**: This is just the admin panel, not the data layer
- **Check EdgeCommon** for DAO definitions and protobuf service definitions

### Session Management

- **Custom session manager**: `internal/nodes/session_manager.go`
- **Session ID**: `edge-admin` prefix with cookie name from `teaconst.CookieSID`
- **Admin ID stored in session**: Accessed via `this.AdminId()`

### Event System

- **Events are published** via `internal/events/` package
- **Common events**: `events.EventStart`, `events.EventQuit`
- **Register handlers** with `events.On(events.EventName, func(){})`

### Background Tasks

- **Tasks run in goroutines** managed by `internal/goman/`
- **Example**: EdgeAPI is started as a subprocess in `adminNode.startAPINode()`
- **Use goman.Go()** instead of `go func()` for better tracking

### WAF Integration

- **Uses libinjection** for SQLi and XSS detection
- **C code compiled into Go**: See `internal/waf/injectionutils/`
- **Fallback implementations** exist for systems without CGO

### View Data

- **Use this.Data** to pass data to templates
- **Type**: `maps.Map` (custom map wrapper from TeaGo)
- **Access in templates**: `{{.dataKey}}`

### Error Pages

- **Standard error handling**: `this.ErrorPage(err)` renders error template
- **Detects API node status**: Shows "API node starting" message if API is unavailable
- **Reads local logs**: Displays last error from `edge-api/logs/issues.log` for debugging

## Code Generation

The `./edge-admin generate` command runs `gen.Generate()` which:
- Generates code based on protobuf definitions (if applicable)
- Regenerates boilerplate files
- Check `internal/gen/generate_test.go` for usage

## Docker Deployment

**Multi-container setup**:
- EdgeAdmin (port 7788)
- EdgeAPI (port 8001) - automatically started by EdgeAdmin
- MySQL (port 3306) - embedded in Docker container

**Data persistence**:
- Config files: `/usr/local/goedge/edge-admin/configs/`
- Logs: `/usr/local/goedge/edge-admin/logs/`
- EdgeAPI: `/usr/local/goedge/edge-api/`

## Development Workflow

1. **Adding a new feature**:
   - Create action file in `internal/web/actions/default/[module]/`
   - Import package in `internal/web/import.go`
   - Create corresponding HTML template in `web/views/@default/[module]/`
   - Add Vue.js components if needed in `web/public/js/components/`

2. **Modifying existing code**:
   - Actions follow standard patterns (Init, RunGet, RunPost)
   - Use RPC clients for all data access
   - Check teaconst.IsDemoMode for write operations
   - Add operation logging with `this.CreateLogInfo()`

3. **Testing**:
   - Write unit tests for utility functions
   - Test actions by running the app and accessing the web UI
   - Check logs in `logs/run.log` for errors

4. **Debugging**:
   - Check `logs/run.log` for application logs
   - Check `edge-api/logs/run.log` for API logs
   - Check `edge-api/logs/issues.log` for API startup issues
   - Use `edge-admin dev` to enable development mode with more verbose logging

## Common Issues

**"process is already running"**:
- Another instance is running
- Check with `edge-admin` (without args) or check processes
- Use `edge-admin stop` to stop existing instance

**"no valid 'rpc.endpoints'"**:
- Config file is missing or malformed
- Run `edge-admin reset` to reinitialize config
- Check `configs/api_admin.yaml` exists and has valid endpoints

**RPC connection errors**:
- EdgeAPI is not running or unreachable
- Check EdgeAPI logs at `edge-api/logs/run.log`
- Verify `RPCEndpoints` in config

**Build failures**:
- EdgeCommon dependency is missing
- Check that `../EdgeCommon` exists and is properly initialized
- Run `cd ../EdgeCommon && go mod tidy` if needed

**404 on new pages**:
- Action package not imported in `internal/web/import.go`
- Check import is present with blank import syntax
- Restart application after adding import

**CSRF token errors**:
- Form POST handler missing `CSRF *actionutils.CSRF` parameter
- Check all POST actions include the CSRF field
- Verify CSRF protection is initialized (automatic in ParentAction)

## External Dependencies

**Key Libraries**:
- `github.com/iwind/TeaGo` - Web framework (routing, actions, templates)
- `google.golang.org/grpc` - gRPC client library
- `github.com/TeaOSLab/EdgeCommon` - Shared definitions (protobuf, configs)
- `github.com/miekg/dns` - DNS operations
- `github.com/tealeg/xlsx/v3` - Excel export
- `gopkg.in/yaml.v3` - YAML config parsing

**Frontend**:
- Vue.js - JavaScript framework
- Semantic UI - CSS framework
- ECharts - Charts and graphs
- CodeMirror - Code editor
- Quill - Rich text editor
- SweetAlert2 - Alert dialogs
- Moment.js - Date/time handling
- Pikaday - Date picker

## Project-Specific Context

**Multi-tenant Architecture**:
- Admin panel manages multiple users
- Each user can have their own clusters and servers
- Admin actions are logged with operation logs

**Cluster Management**:
- Nodes are organized into clusters
- Clusters can be assigned to users
- Nodes communicate with each other via gRPC

**WAF Features**:
- IP lists (allow, deny, grey)
- HTTP firewall rules (SQLi, XSS, etc.)
- Country/province blocking
- Provider-based blocking
- Rate limiting

**DNS Integration**:
- Automatic DNS record management
- Support for multiple DNS providers
- Route-based DNS management

**SSL/TLS**:
- ACME/Let's Encrypt integration
- Custom certificate upload
- Automatic renewal
- OCSP stapling support

**Caching**:
- HTTP caching policies
- Purge cache operations
- Cache statistics
- Batch cache operations

## When Working with This Codebase

1. **Always check existing patterns** before writing new code - the codebase is consistent
2. **Use the RPC client** for all data operations - never bypass it
3. **Follow the action structure** (Init, RunGet/RunPost) - framework depends on it
4. **Add operation logging** for user-facing actions - audit trail is important
5. **Handle errors with ErrorPage** - provides user-friendly error messages
6. **Check demo mode** for write operations - prevents accidental modifications in demo
7. **Import new action packages** in `internal/web/import.go` - or routes won't be registered
8. **Test in dev mode first** - switch to prod only after verification
9. **Check both application logs and API logs** when debugging issues
10. **Respect the disabled linters** - they were disabled for good reasons

## Related Repositories

- [EdgeAPI](https://github.com/TeaOSLab/EdgeAPI) - Backend API service
- [EdgeNode](https://github.com/TeaOSLab/EdgeNode) - Edge node binary
- [EdgeCommon](https://github.com/TeaOSLab/EdgeCommon) - Shared code and definitions

These are managed as a monorepo in production but separate repos on GitHub.
