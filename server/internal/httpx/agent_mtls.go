package httpx

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/apperror"
	"graft/server/internal/config"
	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/moduleapi"
)

const agentIdentityContextKey = "httpx.agent_mtls_identity"

type agentIdentityContextValue struct{}

// AgentMTLSIdentity 是已验证客户端证书导出的 Agent 身份与证书证据。
// 证书内容不会进入请求上下文，后续存储只可使用序列号和公钥指纹。
type AgentMTLSIdentity struct {
	moduleapi.AgentIdentity
}

// AgentMTLSIdentityFromContext 返回 mTLS 中间件写入的可信 Agent 身份。
func AgentMTLSIdentityFromContext(ctx context.Context) (AgentMTLSIdentity, bool) {
	if ctx == nil {
		return AgentMTLSIdentity{}, false
	}
	identity, ok := ctx.Value(agentIdentityContextValue{}).(AgentMTLSIdentity)
	return identity, ok
}

// RequireAgentMTLSIdentity 从 TLS 已验证链提取精确 URI SAN 身份。
// 身份绝不接受 HTTP header、bearer token 或请求 payload 的替代值。
func RequireAgentMTLSIdentity(runtimeLoggers ...*zap.Logger) gin.HandlerFunc {
	runtimeLogger := zap.NewNop()
	if len(runtimeLoggers) > 0 && runtimeLoggers[0] != nil {
		runtimeLogger = runtimeLoggers[0]
	}
	return func(ctx *gin.Context) {
		identity, err := agentMTLSIdentityFromRequest(ctx.Request)
		if err != nil {
			abortAgentMTLSIdentity(ctx, runtimeLogger)
			return
		}
		ctx.Set(agentIdentityContextKey, identity)
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), agentIdentityContextValue{}, identity))
		ctx.Next()
	}
}

func abortAgentMTLSIdentity(ctx *gin.Context, runtimeLogger *zap.Logger) {
	logsafe.Warn(runtimeLogger, "agent mTLS identity rejected",
		zap.String("request_id", EnsureRequestID(ctx)),
		zap.String("trace_id", EnsureTraceID(ctx)),
		zap.String("method", currentRequestMethod(ctx)),
		zap.String("route", currentRequestRoute(ctx)),
		zap.String("path", currentRequestPath(ctx)),
		zap.String("reason", "unverified_or_invalid_client_certificate"),
	)
	AbortAppError(ctx, nil, runtimeLogger, apperror.New(apperror.Descriptor{
		Kind:       apperror.KindUnauthenticated,
		Code:       errorcode.AuthTokenInvalid,
		MessageKey: messagecontract.AuthTokenInvalid,
	}))
}

// AgentMTLSIdentityFromGinContext 返回当前 Agent handler 可消费的证书身份。
func AgentMTLSIdentityFromGinContext(ctx *gin.Context) (AgentMTLSIdentity, bool) {
	if ctx == nil || ctx.Request == nil {
		return AgentMTLSIdentity{}, false
	}
	if identity, ok := AgentMTLSIdentityFromContext(ctx.Request.Context()); ok {
		return identity, true
	}
	identity, ok := ctx.Get(agentIdentityContextKey)
	result, typed := identity.(AgentMTLSIdentity)
	return result, ok && typed
}

func agentMTLSIdentityFromRequest(request *http.Request) (AgentMTLSIdentity, error) {
	if request == nil || request.TLS == nil || len(request.TLS.VerifiedChains) == 0 || len(request.TLS.VerifiedChains[0]) == 0 {
		return AgentMTLSIdentity{}, errors.New("verified client certificate is required")
	}
	certificate := request.TLS.VerifiedChains[0][0]
	identity, err := parseAgentCertificateIdentity(certificate)
	if err != nil {
		return AgentMTLSIdentity{}, err
	}
	fingerprint := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
	identity.CertificateSerial = certificate.SerialNumber.String()
	identity.PublicKeyFingerprint = "sha256:" + hex.EncodeToString(fingerprint[:])
	return identity, nil
}

func parseAgentCertificateIdentity(certificate *x509.Certificate) (AgentMTLSIdentity, error) {
	if certificate == nil || len(certificate.URIs) != 1 || len(certificate.DNSNames) != 0 || len(certificate.IPAddresses) != 0 || len(certificate.EmailAddresses) != 0 {
		return AgentMTLSIdentity{}, errors.New("certificate SAN identity is invalid")
	}
	return parseAgentIdentityURI(certificate.URIs[0])
}

//nolint:cyclop // URI SAN 的每段都属于安全边界，保持在同一函数中便于审计。
func parseAgentIdentityURI(identityURI *url.URL) (AgentMTLSIdentity, error) {
	if identityURI == nil || identityURI.Scheme != "spiffe" || identityURI.Host != "graft" || identityURI.RawQuery != "" || identityURI.Fragment != "" {
		return AgentMTLSIdentity{}, errors.New("agent URI SAN is invalid")
	}
	parts := strings.Split(strings.TrimPrefix(identityURI.EscapedPath(), "/"), "/")
	if len(parts) != 6 || parts[0] != "runtime-target" || parts[2] != "builder-agent" || parts[4] != "generation" {
		return AgentMTLSIdentity{}, errors.New("agent URI SAN path is invalid")
	}
	targetID, err := parseCanonicalPositiveInt64(parts[1])
	if err != nil {
		return AgentMTLSIdentity{}, fmt.Errorf("parse agent URI SAN target: %w", err)
	}
	if !isCanonicalAgentID(parts[3]) {
		return AgentMTLSIdentity{}, errors.New("agent URI SAN agent id is invalid")
	}
	generation, err := parseCanonicalPositiveInt64(parts[5])
	if err != nil {
		return AgentMTLSIdentity{}, fmt.Errorf("parse agent URI SAN generation: %w", err)
	}
	return AgentMTLSIdentity{AgentIdentity: moduleapi.AgentIdentity{TargetID: targetID, AgentID: parts[3], Generation: generation}}, nil
}

func parseCanonicalPositiveInt64(value string) (int64, error) {
	if value == "" || strings.HasPrefix(value, "+") || (len(value) > 1 && value[0] == '0') {
		return 0, errors.New("not a canonical positive integer")
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 1 {
		return 0, errors.New("not a canonical positive integer")
	}
	return parsed, nil
}

//nolint:cyclop // 逐字符 allowlist 避免把 Agent 标识解析委托给宽松的通用规则。
func isCanonicalAgentID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for index, current := range value {
		if (current >= 'a' && current <= 'z') || (current >= '0' && current <= '9') {
			continue
		}
		if current == '-' && index > 0 && index < len(value)-1 {
			continue
		}
		return false
	}
	return true
}

// AgentServer 是专用于 Agent mTLS 路由的独立监听器，不复用用户 HTTP listener。
type AgentServer struct {
	engine                 *gin.Engine
	tlsConfig              *tls.Config
	logger                 *zap.Logger
	mu                     sync.Mutex
	server                 *http.Server
	ledgerRoutesConfigured bool
}

// NewAgentServer 从部署挂载的证书文件构造 Agent 专用 TLS 监听器。
func NewAgentServer(cfg config.AgentTLSConfig, logger *zap.Logger) (*AgentServer, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	tlsConfig, err := newAgentTLSConfig(cfg)
	if err != nil {
		return nil, err
	}
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), newRecoveryMiddleware(logger, nil), RequireAgentMTLSIdentity(logger))
	return &AgentServer{engine: engine, tlsConfig: tlsConfig, logger: logger}, nil
}

// Engine 返回 Agent 专用路由树；只有 Vault-backed handler 可在此注册路由。
func (s *AgentServer) Engine() *gin.Engine {
	if s == nil {
		return nil
	}
	return s.engine
}

// Start 启动专用 TLS listener。
func (s *AgentServer) Start(addr string) (<-chan error, error) {
	if s == nil || s.engine == nil || s.tlsConfig == nil {
		return nil, errors.New("agent mTLS server is unavailable")
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen agent mTLS: %w", err)
	}
	return s.StartListener(listener)
}

// StartListener 在已绑定的原始 listener 上启动 Agent 专用 mTLS 服务。
// 调用方不能自行绕过 TLS；该方法始终使用 AgentServer 创建时固定的 TLS 配置。
func (s *AgentServer) StartListener(listener net.Listener) (<-chan error, error) {
	if s == nil || s.engine == nil || s.tlsConfig == nil || listener == nil {
		return nil, errors.New("agent mTLS server is unavailable")
	}
	tlsListener := tls.NewListener(listener, s.tlsConfig)
	server := &http.Server{Handler: s.engine, ReadHeaderTimeout: defaultServerReadHeaderTimeout, ReadTimeout: defaultServerReadTimeout, WriteTimeout: defaultServerWriteTimeout, IdleTimeout: defaultServerIdleTimeout}
	if err := s.bindRunningServer(server); err != nil {
		_ = tlsListener.Close()
		return nil, err
	}
	errCh := make(chan error, 1)
	go func() {
		defer s.clearRunningServer(server)
		err := server.Serve(tlsListener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve agent mTLS: %w", err)
		}
		close(errCh)
	}()
	return errCh, nil
}

// Shutdown 停止专用 Agent listener。
func (s *AgentServer) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	server := s.detachRunningServer()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

func (s *AgentServer) bindRunningServer(server *http.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return errors.New("agent mTLS server already running")
	}
	s.server = server
	return nil
}

func (s *AgentServer) clearRunningServer(server *http.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server == server {
		s.server = nil
	}
}

func (s *AgentServer) detachRunningServer() *http.Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	server := s.server
	s.server = nil
	return server
}

func newAgentTLSConfig(cfg config.AgentTLSConfig) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.CertificateFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load agent TLS certificate: %w", err)
	}
	clientCA, err := os.ReadFile(cfg.ClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read agent client CA: %w", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(clientCA) {
		return nil, errors.New("parse agent client CA")
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: clientCAs}, nil
}
