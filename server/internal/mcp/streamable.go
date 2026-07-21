package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"graft/server/internal/buildinfo"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
)

const (
	// HTTPPath 是产品 MCP Streamable HTTP transport 的唯一公开路径。
	HTTPPath = "/mcp"

	streamableSessionTimeout = 15 * time.Minute
)

// HTTPRegistration 描述 MCP transport 装配所需的 core 与 auth 依赖。
//
// 业务 Tool 不在此处注册；后续 compiler 只能在同一个 adapter 上投影
// OpenAPI 已声明的 operation，不能把 transport 变成第二套业务 API。
type HTTPRegistration struct {
	Engine                 *gin.Engine
	I18n                   *i18n.Service
	PersonalTokenService   moduleapi.PersonalAccessTokenService
	Authorizer             moduleapi.Authorizer
	SecurityAuditPublisher httpx.SecurityAuditPublisher
	ConfirmationTokenTTL   time.Duration
}

// adapter 持有 MCP Streamable HTTP handler 与后续 Tool 投影需要的安全基础。
type adapter struct {
	handler       http.Handler
	scopes        ScopeGate
	confirmations *ConfirmationTokens
}

// Register 在根 Gin engine 注册受个人 API Token 保护的 MCP transport。
//
// 注册仅在 app 确认部署开关开启后调用。认证仍由 httpx 和 auth 模块负责，
// 这里不解析 token、不调用 loopback REST，也不注册业务 Tool。
func Register(registration HTTPRegistration) error {
	if registration.Engine == nil {
		return errors.New("mcp runtime engine is unavailable")
	}
	if registration.PersonalTokenService == nil {
		return errors.New("mcp personal API token service is unavailable")
	}
	if registration.Authorizer == nil {
		return errors.New("mcp authorizer is unavailable")
	}

	adapter, err := newAdapter(registration.Authorizer, registration.ConfirmationTokenTTL)
	if err != nil {
		return err
	}
	registration.Engine.Any(
		HTTPPath,
		httpx.RequirePersonalAccessToken(registration.I18n, registration.PersonalTokenService, registration.SecurityAuditPublisher),
		adapter.serveHTTP,
	)
	return nil
}

func newAdapter(authorizer moduleapi.Authorizer, confirmationTTL time.Duration) (*adapter, error) {
	confirmations, err := newConfirmationTokens(confirmationTTL)
	if err != nil {
		return nil, err
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "graft",
		Title:   "Graft",
		Version: buildinfo.Current().Version,
	}, &mcpsdk.ServerOptions{
		// Foundation batch intentionally exposes no tools, resources, prompts, or logging capability.
		Capabilities: &mcpsdk.ServerCapabilities{},
	})
	streamable := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, &mcpsdk.StreamableHTTPOptions{
		CrossOriginProtection: &http.CrossOriginProtection{},
		SessionTimeout:        streamableSessionTimeout,
	})

	return &adapter{
		handler:       withSDKTokenInfo(streamable),
		scopes:        ScopeGate{authorizer: authorizer},
		confirmations: confirmations,
	}, nil
}

func (a *adapter) serveHTTP(ginCtx *gin.Context) {
	if a == nil || a.handler == nil {
		http.Error(ginCtx.Writer, "MCP runtime unavailable", http.StatusServiceUnavailable)
		ginCtx.Abort()
		return
	}

	source, ok := moduleapi.PersonalAccessTokenCallerFromContext(ginCtx.Request.Context())
	if !ok {
		http.Error(ginCtx.Writer, "MCP caller unavailable", http.StatusUnauthorized)
		ginCtx.Abort()
		return
	}
	caller, err := newCaller(source)
	if err != nil {
		http.Error(ginCtx.Writer, "MCP caller unavailable", http.StatusUnauthorized)
		ginCtx.Abort()
		return
	}

	ginCtx.Request = ginCtx.Request.WithContext(withCaller(ginCtx.Request.Context(), caller))
	ginCtx.Abort()
	a.handler.ServeHTTP(ginCtx.Writer, ginCtx.Request)
}

func withSDKTokenInfo(next http.Handler) http.Handler {
	return mcpauth.RequireBearerToken(func(ctx context.Context, _ string, _ *http.Request) (*mcpauth.TokenInfo, error) {
		caller, ok := callerFromContext(ctx)
		if !ok {
			return nil, fmt.Errorf("%w: MCP caller context is missing", mcpauth.ErrInvalidToken)
		}
		return &mcpauth.TokenInfo{
			Scopes:     append([]string(nil), caller.scopes...),
			Expiration: time.Unix(caller.expiresAt, 0).UTC(),
			// SDK 使用该字段绑定 Streamable 会话。这里刻意使用 Token ID 而非用户 ID，
			// 避免低权限 scope 的 Token 恢复同一用户另一枚 Token 建立的会话。
			UserID: strconv.FormatUint(caller.tokenID, 10),
		}, nil
	}, nil)(next)
}
