package mcp

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
)

// StdioRegistration 将已认证的个人 API Token 调用者绑定到一个 stdio MCP 进程。
// stdio 不解析凭据；调用者必须由使用现有 Token service 的受信任启动器提供。
type StdioRegistration struct {
	Engine               *gin.Engine
	OpenAPISpec          []byte
	Authorizer           moduleapi.Authorizer
	Caller               moduleapi.PersonalAccessTokenCaller
	ConfirmationTokenTTL time.Duration
	Limits               RuntimeLimits
	Logger               *zap.Logger
	Reader               io.ReadCloser
	Writer               io.WriteCloser
}

// RunStdio 使用与 Streamable HTTP 相同的 OpenAPI 编译结果和 Gin route dispatcher 运行一个 stdio 会话。
func RunStdio(ctx context.Context, registration StdioRegistration) error {
	caller, err := stdioCaller(registration)
	if err != nil {
		return err
	}
	capabilities, err := CompileCapabilities(registration.OpenAPISpec)
	if err != nil {
		return fmt.Errorf("compile MCP capabilities: %w", err)
	}
	adapter, err := newAdapterWithCapabilities(registration.Authorizer, registration.ConfirmationTokenTTL, registration.Engine, capabilities, registration.Limits, registration.Logger)
	if err != nil {
		return err
	}
	defer func() { _ = adapter.Close() }()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = withCaller(ctx, caller)
	transport, err := stdioTransport(registration)
	if err != nil {
		return err
	}
	return adapter.server.Run(ctx, transport)
}

func stdioCaller(registration StdioRegistration) (caller, error) {
	if registration.Engine == nil || registration.Authorizer == nil {
		return caller{}, fmt.Errorf("mcp stdio runtime dependencies are unavailable")
	}
	resolved, err := newCaller(registration.Caller)
	if err != nil {
		return caller{}, fmt.Errorf("mcp stdio caller: %w", err)
	}
	if time.Now().Unix() >= resolved.expiresAt {
		return caller{}, fmt.Errorf("mcp stdio caller token is expired")
	}
	return resolved, nil
}

func stdioTransport(registration StdioRegistration) (mcpsdk.Transport, error) {
	transport := mcpsdk.Transport(&mcpsdk.StdioTransport{})
	if registration.Reader != nil || registration.Writer != nil {
		if registration.Reader == nil || registration.Writer == nil {
			return nil, fmt.Errorf("mcp stdio reader and writer must be provided together")
		}
		transport = &mcpsdk.IOTransport{Reader: registration.Reader, Writer: registration.Writer}
	}
	return transport, nil
}
