package rpc

import (
	"context"
	"time"

	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/TeaOSLab/EdgeUser/internal/consts"
	"github.com/TeaOSLab/EdgeUser/internal/const"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client gRPC客户端封装
type Client struct {
	userService          pb.UserServiceClient
	userAccessKeyService pb.UserAccessKeyServiceClient
	userIdentityService  pb.UserIdentityServiceClient
	serverService        pb.ServerServiceClient
	conn                 *grpc.ClientConn
}

// NewClient 创建新的gRPC客户端
func NewClient() (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	
	conn, err := grpc.DialContext(ctx, consts.EdgeAPIAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	
	return &Client{
		userService:          pb.NewUserServiceClient(conn),
		userAccessKeyService: pb.NewUserAccessKeyServiceClient(conn),
		userIdentityService:  pb.NewUserIdentityServiceClient(conn),
		serverService:        pb.NewServerServiceClient(conn),
		conn:                 conn,
	}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetUserService 获取用户服务客户端
func (c *Client) GetUserService() pb.UserServiceClient {
	return c.userService
}

// GetUserAccessKeyService 获取用户访问密钥服务客户端
func (c *Client) GetUserAccessKeyService() pb.UserAccessKeyServiceClient {
	return c.userAccessKeyService
}

// GetUserIdentityService 获取用户身份认证服务客户端
func (c *Client) GetUserIdentityService() pb.UserIdentityServiceClient {
	return c.userIdentityService
}

// GetServerService 获取服务器服务客户端
func (c *Client) GetServerService() pb.ServerServiceClient {
	return c.serverService
}
}