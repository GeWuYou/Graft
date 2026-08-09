package httpx

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/apperror"
	"graft/server/internal/config"
	"graft/server/internal/contract/errorcode"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/moduleapi"
)

const agentBootstrapCertificatePath = "/bootstrap/v1/certificate"
const maxAgentBootstrapRequestBytes int64 = 1 << 20

// AgentBootstrapServer 是首次证书签发使用的独立 server-authenticated TLS listener。
// 它只接受一次性 token 和 CSR，不能承载 bearer、cookie 或后续 mTLS Agent 流量。
type AgentBootstrapServer struct {
	engine     *gin.Engine
	tlsConfig  *tls.Config
	logger     *zap.Logger
	mu         sync.Mutex
	server     *http.Server
	configured bool
}

// NewAgentBootstrapServer 从部署挂载的 server TLS 证书构造 bootstrap listener。
func NewAgentBootstrapServer(cfg config.AgentBootstrapTLSConfig, logger *zap.Logger) (*AgentBootstrapServer, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	certificate, err := tls.LoadX509KeyPair(cfg.CertificateFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load agent bootstrap TLS certificate: %w", err)
	}
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), newRecoveryMiddleware(logger, nil), agentBootstrapNoStore())
	return &AgentBootstrapServer{
		engine:    engine,
		tlsConfig: &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}},
		logger:    logger,
	}, nil
}

// Configure 注册唯一的 Runtime Target bootstrap authority。
// 配置必须在 listener 启动前完成，避免未绑定业务授权的 TLS 暴露。
func (s *AgentBootstrapServer) Configure(authority moduleapi.AgentBootstrapAuthority) error {
	if s == nil || s.engine == nil {
		return errors.New("agent bootstrap server is unavailable")
	}
	if authority == nil {
		return errors.New("agent bootstrap authority is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.configured {
		return errors.New("agent bootstrap server is already configured")
	}
	s.engine.POST(agentBootstrapCertificatePath, agentBootstrapCertificateHandler(authority, s.logger))
	s.configured = true
	return nil
}

// Engine 返回 bootstrap 专用路由树，仅用于受控 listener 测试与运行时装配。
func (s *AgentBootstrapServer) Engine() *gin.Engine {
	if s == nil {
		return nil
	}
	return s.engine
}

// Start 启动专用 bootstrap TLS listener。
func (s *AgentBootstrapServer) Start(addr string) (<-chan error, error) {
	if s == nil || s.engine == nil || s.tlsConfig == nil {
		return nil, errors.New("agent bootstrap server is unavailable")
	}
	s.mu.Lock()
	configured := s.configured
	s.mu.Unlock()
	if !configured {
		return nil, errors.New("agent bootstrap server is not configured")
	}
	listener, err := tls.Listen("tcp", addr, s.tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("listen agent bootstrap TLS: %w", err)
	}
	server := &http.Server{Handler: s.engine, ReadHeaderTimeout: defaultServerReadHeaderTimeout, ReadTimeout: defaultServerReadTimeout, WriteTimeout: defaultServerWriteTimeout, IdleTimeout: defaultServerIdleTimeout}
	if err := s.bindRunningServer(server); err != nil {
		_ = listener.Close()
		return nil, err
	}
	errCh := make(chan error, 1)
	go func() {
		defer s.clearRunningServer(server)
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve agent bootstrap TLS: %w", err)
		}
		close(errCh)
	}()
	return errCh, nil
}

// Shutdown 停止专用 bootstrap listener。
func (s *AgentBootstrapServer) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	server := s.detachRunningServer()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

func (s *AgentBootstrapServer) bindRunningServer(server *http.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return errors.New("agent bootstrap server already running")
	}
	s.server = server
	return nil
}

func (s *AgentBootstrapServer) clearRunningServer(server *http.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server == server {
		s.server = nil
	}
}

func (s *AgentBootstrapServer) detachRunningServer() *http.Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	server := s.server
	s.server = nil
	return server
}

type agentBootstrapCertificateRequest struct {
	BootstrapToken string `json:"bootstrap_token"`
	CSRDER         []byte `json:"csr_der"`
}

type agentBootstrapCertificateResponse struct {
	CertificateChainDER [][]byte                       `json:"certificate_chain_der"`
	TrustBundle         moduleapi.TrustBundleReference `json:"trust_bundle"`
	ExpiresAt           string                         `json:"expires_at"`
}

func agentBootstrapNoStore() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("Cache-Control", "no-store")
		ctx.Next()
	}
}

func agentBootstrapCertificateHandler(authority moduleapi.AgentBootstrapAuthority, runtimeLogger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.GetHeader(httpheader.Authorization.String()) != "" || ctx.GetHeader(httpheader.Cookie.String()) != "" {
			abortInvalidAgentBootstrapRequest(ctx, runtimeLogger)
			return
		}
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxAgentBootstrapRequestBytes)
		request, err := decodeAgentBootstrapCertificateRequest(ctx.Request)
		if err != nil {
			abortInvalidAgentBootstrapRequest(ctx, runtimeLogger)
			return
		}
		result, err := authority.BootstrapAgent(ctx.Request.Context(), request)
		if err != nil {
			abortAgentBootstrapAuthorityError(ctx, runtimeLogger, err)
			return
		}
		ctx.JSON(http.StatusOK, agentBootstrapCertificateResponse{CertificateChainDER: result.CertificateChainDER, TrustBundle: result.TrustBundle, ExpiresAt: result.ExpiresAt.UTC().Format(time.RFC3339)})
	}
}

func abortAgentBootstrapAuthorityError(ctx *gin.Context, runtimeLogger *zap.Logger, err error) {
	if errors.Is(err, moduleapi.ErrAgentBootstrapRejected) {
		AbortAppError(ctx, nil, runtimeLogger, apperror.New(apperror.Descriptor{Kind: apperror.KindUnauthenticated, Code: errorcode.AuthTokenInvalid, MessageKey: messagecontract.AuthTokenInvalid}))
		return
	}
	AbortAppError(ctx, nil, runtimeLogger, apperror.Wrap(err, apperror.Descriptor{Kind: apperror.KindInternal, Code: errorcode.CommonInternalError, MessageKey: messagecontract.CommonInternalError}))
}

func decodeAgentBootstrapCertificateRequest(request *http.Request) (moduleapi.AgentBootstrapRequest, error) {
	if request == nil || request.Body == nil || !isAgentBootstrapJSONContentType(request.Header.Get("Content-Type")) {
		return moduleapi.AgentBootstrapRequest{}, errors.New("bootstrap request content type is invalid")
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload agentBootstrapCertificateRequest
	if err := decoder.Decode(&payload); err != nil {
		return moduleapi.AgentBootstrapRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return moduleapi.AgentBootstrapRequest{}, errors.New("bootstrap request must contain one JSON value")
	}
	if strings.TrimSpace(payload.BootstrapToken) == "" || len(payload.CSRDER) == 0 {
		return moduleapi.AgentBootstrapRequest{}, errors.New("bootstrap request is incomplete")
	}
	return moduleapi.AgentBootstrapRequest{BootstrapToken: payload.BootstrapToken, CSRDER: payload.CSRDER}, nil
}

func isAgentBootstrapJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && strings.EqualFold(mediaType, "application/json")
}

func abortInvalidAgentBootstrapRequest(ctx *gin.Context, runtimeLogger *zap.Logger) {
	AbortAppError(ctx, nil, runtimeLogger, apperror.New(apperror.Descriptor{Kind: apperror.KindInvalidArgument, Code: errorcode.CommonInvalidArgument, MessageKey: messagecontract.CommonInvalidArgument}))
}
