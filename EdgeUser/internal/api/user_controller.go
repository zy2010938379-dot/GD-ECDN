package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	consts "github.com/TeaOSLab/EdgeUser/internal/const"
	rpc "github.com/TeaOSLab/EdgeUser/internal/rpc"
	"github.com/gin-gonic/gin"
)

// UserController 用户控制器
type UserController struct {
	rpcClient *rpc.Client
}

// NewUserController 创建用户控制器实例
func NewUserController() (*UserController, error) {
	client, err := rpc.NewClient()
	if err != nil {
		return nil, err
	}

	return &UserController{
		rpcClient: client,
	}, nil
}

// Close 关闭连接
func (c *UserController) Close() error {
	if c.rpcClient != nil {
		return c.rpcClient.Close()
	}
	return nil
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录接口，直接调用EdgeAPI的gRPC接口
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param loginRequest body pb.LoginUserRequest true "登录请求参数"
// @Success 200 {object} pb.LoginUserResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /api/v1/users/login [post]
func (c *UserController) Login(ctx *gin.Context) {
	var req pb.LoginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetUserService().LoginUser(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "登录失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Register 用户注册
// @Summary 用户注册
// @Description 用户注册接口，直接调用EdgeAPI的gRPC接口
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param registerRequest body pb.RegisterUserRequest true "注册请求参数"
// @Success 201 {object} pb.RegisterUserResponse
// @Failure 400 {object} gin.H
// @Failure 409 {object} gin.H
// @Router /api/v1/users/register [post]
func (c *UserController) Register(ctx *gin.Context) {
	var req pb.RegisterUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetUserService().RegisterUser(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "注册失败"})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// GetUserInfo 获取用户信息
// @Summary 获取用户信息
// @Description 获取用户详细信息，直接调用EdgeAPI的gRPC接口
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param userId path int64 true "用户ID"
// @Success 200 {object} pb.FindEnabledUserResponse
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /api/v1/users/{userId} [get]
func (c *UserController) GetUserInfo(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.FindEnabledUserRequest{UserId: userID}
	response, err := c.rpcClient.GetUserService().FindEnabledUser(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateUserInfo 更新用户信息
// @Summary 更新用户信息
// @Description 更新用户基本信息，直接调用EdgeAPI的gRPC接口
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param updateRequest body pb.UpdateUserInfoRequest true "更新请求参数"
// @Success 200 {object} pb.RPCSuccess
// @Failure 400 {object} gin.H
// @Router /api/v1/users/info [put]
func (c *UserController) UpdateUserInfo(ctx *gin.Context) {
	var req pb.UpdateUserInfoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetUserService().UpdateUserInfo(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "更新失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// CreateAccessKey 创建访问密钥
// @Summary 创建访问密钥
// @Description 为用户创建新的访问密钥，直接调用EdgeAPI的gRPC接口
// @Tags 访问密钥管理
// @Accept json
// @Produce json
// @Param createRequest body pb.CreateUserAccessKeyRequest true "创建访问密钥请求参数"
// @Success 201 {object} pb.CreateUserAccessKeyResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/users/access-keys [post]
func (c *UserController) CreateAccessKey(ctx *gin.Context) {
	var req pb.CreateUserAccessKeyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetUserAccessKeyService().CreateUserAccessKey(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "创建失败"})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// ListAccessKeys 列出访问密钥
// @Summary 列出访问密钥
// @Description 列出用户的所有访问密钥，直接调用EdgeAPI的gRPC接口
// @Tags 访问密钥管理
// @Accept json
// @Produce json
// @Param userId query int64 true "用户ID"
// @Success 200 {object} pb.FindAllEnabledUserAccessKeysResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/users/access-keys [get]
func (c *UserController) ListAccessKeys(ctx *gin.Context) {
	userIDStr := ctx.Query("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.FindAllEnabledUserAccessKeysRequest{UserId: userID}
	response, err := c.rpcClient.GetUserAccessKeyService().FindAllEnabledUserAccessKeys(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "查询失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// DeleteAccessKey 删除访问密钥
// @Summary 删除访问密钥
// @Description 删除用户的访问密钥，直接调用EdgeAPI的gRPC接口
// @Tags 访问密钥管理
// @Accept json
// @Produce json
// @Param accessKeyId path int64 true "访问密钥ID"
// @Success 200 {object} pb.RPCSuccess
// @Failure 400 {object} gin.H
// @Router /api/v1/users/access-keys/{accessKeyId} [delete]
func (c *UserController) DeleteAccessKey(ctx *gin.Context) {
	accessKeyIDStr := ctx.Param("accessKeyId")
	accessKeyID, err := strconv.ParseInt(accessKeyIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的访问密钥ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.DeleteUserAccessKeyRequest{UserAccessKeyId: accessKeyID}
	response, err := c.rpcClient.GetUserAccessKeyService().DeleteUserAccessKey(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "删除失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetUserDashboard 获取用户仪表板数据
// @Summary 获取用户仪表板数据
// @Description 获取用户的服务器统计、流量统计等仪表板数据
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param userId path int64 true "用户ID"
// @Success 200 {object} pb.ComposeUserDashboardResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/users/dashboard/{userId} [get]
func (c *UserController) GetUserDashboard(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.ComposeUserDashboardRequest{UserId: userID}
	response, err := c.rpcClient.GetUserService().ComposeUserDashboard(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取仪表板数据失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetUserFeatures 获取用户功能列表
// @Summary 获取用户功能列表
// @Description 获取用户可以使用的功能列表
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param userId path int64 true "用户ID"
// @Success 200 {object} pb.FindUserFeaturesResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/users/features/{userId} [get]
func (c *UserController) GetUserFeatures(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.FindUserFeaturesRequest{UserId: userID}
	response, err := c.rpcClient.GetUserService().FindUserFeatures(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取功能列表失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// RenewUserServersState 更新用户服务可用状态
// @Summary 更新用户服务可用状态
// @Description 更新用户的服务可用状态，用于续费或激活服务
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param userId path int64 true "用户ID"
// @Success 200 {object} pb.RenewUserServersStateResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/users/servers-state/{userId} [put]
func (c *UserController) RenewUserServersState(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.RenewUserServersStateRequest{UserId: userID}
	response, err := c.rpcClient.GetUserService().RenewUserServersState(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新服务状态失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// CreateUserIdentity 创建用户身份认证信息
// @Summary 创建用户身份认证信息
// @Description 创建用户的实名认证信息
// @Tags 身份认证管理
// @Accept json
// @Produce json
// @Param createRequest body pb.CreateUserIdentityRequest true "创建身份认证请求参数"
// @Success 201 {object} pb.CreateUserIdentityResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/users/identity [post]
func (c *UserController) CreateUserIdentity(ctx *gin.Context) {
	var req pb.CreateUserIdentityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetUserIdentityService().CreateUserIdentity(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建身份认证失败"})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// GetUserIdentity 获取用户身份认证信息
// @Summary 获取用户身份认证信息
// @Description 获取用户的实名认证信息
// @Tags 身份认证管理
// @Accept json
// @Produce json
// @Param userId path int64 true "用户ID"
// @Success 200 {object} pb.FindEnabledUserIdentityResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/users/identity/{userId} [get]
func (c *UserController) GetUserIdentity(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.FindEnabledUserIdentityRequest{UserIdentityId: userID}
	response, err := c.rpcClient.GetUserIdentityService().FindEnabledUserIdentity(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "身份认证信息不存在"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateUserIdentity 更新用户身份认证信息
// @Summary 更新用户身份认证信息
// @Description 更新用户的实名认证信息
// @Tags 身份认证管理
// @Accept json
// @Produce json
// @Param identityId path int64 true "身份认证ID"
// @Param updateRequest body pb.UpdateUserIdentityRequest true "更新身份认证请求参数"
// @Success 200 {object} pb.RPCSuccess
// @Failure 400 {object} gin.H
// @Router /api/v1/users/identity/{identityId} [put]
func (c *UserController) UpdateUserIdentity(ctx *gin.Context) {
	identityIDStr := ctx.Param("identityId")
	identityID, err := strconv.ParseInt(identityIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的身份认证ID"})
		return
	}

	var req pb.UpdateUserIdentityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	req.UserIdentityId = identityID

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetUserIdentityService().UpdateUserIdentity(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新身份认证失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// SubmitUserIdentity 提交用户身份认证审核
// @Summary 提交用户身份认证审核
// @Description 提交用户的实名认证信息进行审核
// @Tags 身份认证管理
// @Accept json
// @Produce json
// @Param identityId path int64 true "身份认证ID"
// @Success 200 {object} pb.RPCSuccess
// @Failure 400 {object} gin.H
// @Router /api/v1/users/identity/{identityId}/submit [post]
func (c *UserController) SubmitUserIdentity(ctx *gin.Context) {
	identityIDStr := ctx.Param("identityId")
	identityID, err := strconv.ParseInt(identityIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的身份认证ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.SubmitUserIdentityRequest{UserIdentityId: identityID}
	response, err := c.rpcClient.GetUserIdentityService().SubmitUserIdentity(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "提交身份认证失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// ListUserServers 列出用户的所有服务器
// @Summary 列出用户的所有服务器
// @Description 获取用户创建的所有服务器列表
// @Tags 服务器管理
// @Accept json
// @Produce json
// @Param userId path int64 true "用户ID"
// @Success 200 {object} pb.ListEnabledServersResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/servers/user/{userId} [get]
func (c *UserController) ListUserServers(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.ListEnabledServersRequest{UserId: userID}
	response, err := c.rpcClient.GetServerService().ListEnabledServers(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取服务器列表失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetServerInfo 获取服务器详细信息
// @Summary 获取服务器详细信息
// @Description 获取指定服务器的详细信息
// @Tags 服务器管理
// @Accept json
// @Produce json
// @Param serverId path int64 true "服务器ID"
// @Success 200 {object} pb.FindEnabledServerResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/servers/{serverId} [get]
func (c *UserController) GetServerInfo(ctx *gin.Context) {
	serverIDStr := ctx.Param("serverId")
	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的服务器ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.FindEnabledServerRequest{ServerId: serverID}
	response, err := c.rpcClient.GetServerService().FindEnabledServer(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "服务器不存在"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// CreateServer 创建服务器
// @Summary 创建服务器
// @Description 为用户创建新的服务器
// @Tags 服务器管理
// @Accept json
// @Produce json
// @Param createRequest body pb.CreateServerRequest true "创建服务器请求参数"
// @Success 201 {object} pb.CreateServerResponse
// @Failure 400 {object} gin.H
// @Router /api/v1/servers [post]
func (c *UserController) CreateServer(ctx *gin.Context) {
	var req pb.CreateServerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetServerService().CreateServer(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建服务器失败"})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// UpdateServer 更新服务器信息
// @Summary 更新服务器信息
// @Description 更新服务器的基本信息
// @Tags 服务器管理
// @Accept json
// @Produce json
// @Param serverId path int64 true "服务器ID"
// @Param updateRequest body pb.UpdateServerBasicRequest true "更新服务器请求参数"
// @Success 200 {object} pb.RPCSuccess
// @Failure 400 {object} gin.H
// @Router /api/v1/servers/{serverId} [put]
func (c *UserController) UpdateServer(ctx *gin.Context) {
	serverIDStr := ctx.Param("serverId")
	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的服务器ID"})
		return
	}

	var req pb.UpdateServerBasicRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	req.ServerId = serverID

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetServerService().UpdateServerBasic(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新服务器失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// DeleteServer 删除服务器
// @Summary 删除服务器
// @Description 删除指定的服务器
// @Tags 服务器管理
// @Accept json
// @Produce json
// @Param serverId path int64 true "服务器ID"
// @Success 200 {object} pb.RPCSuccess
// @Failure 400 {object} gin.H
// @Router /api/v1/servers/{serverId} [delete]
func (c *UserController) DeleteServer(ctx *gin.Context) {
	serverIDStr := ctx.Param("serverId")
	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的服务器ID"})
		return
	}

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	req := &pb.DeleteServerRequest{ServerId: serverID}
	response, err := c.rpcClient.GetServerService().DeleteServer(rpcCtx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "删除服务器失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateAccessKeyStatus 更新访问密钥状态
// @Summary 更新访问密钥状态
// @Description 启用或禁用访问密钥
// @Tags 访问密钥管理
// @Accept json
// @Produce json
// @Param accessKeyId path int64 true "访问密钥ID"
// @Param updateRequest body pb.UpdateUserAccessKeyIsOnRequest true "状态更新请求参数"
// @Success 200 {object} pb.RPCSuccess
// @Failure 400 {object} gin.H
// @Router /api/v1/users/access-keys/{accessKeyId}/status [put]
func (c *UserController) UpdateAccessKeyStatus(ctx *gin.Context) {
	accessKeyIDStr := ctx.Param("accessKeyId")
	accessKeyID, err := strconv.ParseInt(accessKeyIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的访问密钥ID"})
		return
	}

	var req pb.UpdateUserAccessKeyIsOnRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	req.UserAccessKeyId = accessKeyID

	rpcCtx, cancel := context.WithTimeout(context.Background(), time.Duration(consts.RequestTimeout)*time.Second)
	defer cancel()

	response, err := c.rpcClient.GetUserAccessKeyService().UpdateUserAccessKeyIsOn(rpcCtx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "更新访问密钥状态失败"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
