// Package httpx 提供 `server` 运行时使用的 HTTP 服务封装。
package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/i18n"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/moduleapi"
)

const (
	defaultServerReadHeaderTimeout = 5 * time.Second
	defaultServerReadTimeout       = 30 * time.Second
	defaultServerWriteTimeout      = 30 * time.Second
	defaultServerIdleTimeout       = 60 * time.Second
)

// Server 封装运行时使用的 Gin 引擎与 HTTP 服务实例。
//
// Server 负责把 `Run` / `Shutdown` 的生命周期归属集中到一个显式对象中，
// 避免并发启动或停止时出现状态竞争。Server 支持并发调用生命周期方法。
//
// Server 只管理 HTTP 外壳本身，不负责模块路由装配策略或业务中间件语义；
// 这些职责仍留在 app 与各模块边界内。
type Server struct {
	engine         *gin.Engine
	mu             sync.Mutex
	repo           AccessLogRepository
	activeRequests *activeRequestTracker
	// server 持有当前运行中的 http.Server 指针，用于串行化 Run/Shutdown
	// 的所有权切换，避免重复关闭或重复启动同一个生命周期槽位。
	server *http.Server
}

// AccessLogOptions 配置 HTTP access log 持久化与进程日志输出策略。
type AccessLogOptions struct {
	ConsolePolicy config.AccessLogConsolePolicy
	SlowThreshold time.Duration
	// PersistTimeout 限制单次访问日志持久化操作的耗时，非正值会回退到默认超时。
	PersistTimeout time.Duration
}

// ServerOptions 承载 NewServerWithOptions 使用的可选 HTTP runtime 行为。
type ServerOptions struct {
	AccessLog AccessLogOptions
	I18n      *i18n.Service
}

// NewServer 创建 MVP 运行时使用的最小 Gin 服务外壳。
//
// 返回的服务默认挂载全局 request-id、中台统一 access log 与恢复中间件，
// 便于 core 和模块在统一入口上继续注册路由。
func NewServer(logger *zap.Logger, repo ...AccessLogRepository) *Server {
	return NewServerWithOptions(logger, ServerOptions{
		AccessLog: AccessLogOptions{
			ConsolePolicy:  config.AccessLogConsoleAlways,
			SlowThreshold:  time.Second,
			PersistTimeout: time.Second,
		},
	}, repo...)
}

// NewServerWithOptions 使用显式 runtime 选项创建 Gin 服务外壳。
func NewServerWithOptions(logger *zap.Logger, options ServerOptions, repo ...AccessLogRepository) *Server {
	engine := gin.New()
	activeRequests := newActiveRequestTracker()

	var accessLogRepo AccessLogRepository
	for _, candidate := range repo {
		if candidate != nil {
			accessLogRepo = candidate
			break
		}
	}

	engine.Use(
		RequestIDMiddleware(),
		newAccessLogMiddleware(logger, accessLogRepo, activeRequests, options.AccessLog),
		newRecoveryMiddleware(logger, options.I18n),
	)
	return &Server{engine: engine, repo: accessLogRepo, activeRequests: activeRequests}
}

func newRecoveryMiddleware(runtimeLogger *zap.Logger, localizer *i18n.Service) gin.HandlerFunc {
	if runtimeLogger == nil {
		runtimeLogger = zap.NewNop()
	}
	return func(ctx *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			fields := []zap.Field{
				zap.String("request_id", EnsureRequestID(ctx)),
				zap.String("trace_id", EnsureTraceID(ctx)),
				zap.String("method", currentRequestMethod(ctx)),
				zap.String("route", currentRequestRoute(ctx)),
				zap.String("path", currentRequestPath(ctx)),
				zap.String("panic", fmt.Sprint(recovered)),
				zap.Strings("stacktrace", strings.Split(strings.TrimSpace(string(debug.Stack())), "\n")),
			}
			if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx.Request.Context()); ok && requestAuth.User != nil {
				fields = append(fields, zap.Uint64("user_id", requestAuth.User.ID))
			}
			logsafe.Error(runtimeLogger, "panic recovered", fields...)

			if !ctx.Writer.Written() {
				WriteLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			}
			ctx.Abort()
		}()
		ctx.Next()
	}
}

// Engine 返回供 core 和模块注册路由使用的根路由。
//
// 调用方应只在服务启动前完成长期稳定路由注册，避免运行期动态改写根路由
// 带来不可预测的行为。
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// AccessLogRepository 返回当前 HTTP 运行时绑定的访问日志仓储。
//
// 该方法用于让 core runtime 在装配 access-log explorer 时复用同一份
// access-log authority，而不是在其它边界重新构造第二个仓储实例。
func (s *Server) AccessLogRepository() AccessLogRepository {
	if s == nil {
		return nil
	}
	return s.repo
}

// ActiveRequestReader 返回当前 Server 唯一持有的进程内活动请求读取能力。
func (s *Server) ActiveRequestReader() moduleapi.ActiveRequestReader {
	if s == nil {
		return nil
	}
	return s.activeRequests
}

// Start 启动 HTTP 服务，并返回监听 goroutine 的错误通道。
//
// 返回通道会在监听 goroutine 退出时关闭；若监听因非正常错误退出，会先写入
// 一条错误。优雅关闭由上层 runtime 统一编排。
func (s *Server) Start(addr string) (<-chan error, error) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: defaultServerReadHeaderTimeout,
		ReadTimeout:       defaultServerReadTimeout,
		WriteTimeout:      defaultServerWriteTimeout,
		IdleTimeout:       defaultServerIdleTimeout,
	}
	if err := s.bindRunningServer(srv); err != nil {
		return nil, err
	}

	errCh := make(chan error, 1)
	go func() {
		defer s.clearRunningServer(srv)
		// ListenAndServe 正常关闭时会返回 http.ErrServerClosed，这里把它视为
		// 生命周期正常收敛，而不是需要继续向上传播的失败。
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
		close(errCh)
	}()

	return errCh, nil
}

// Shutdown 在服务运行时执行优雅关闭。
//
// 如果当前没有运行中的服务，Shutdown 会返回 nil；这让调用方可以在失败
// 清理路径中无条件调用，而不用额外维护外部状态。
func (s *Server) Shutdown(ctx context.Context) error {
	server := s.detachRunningServer()
	if server == nil {
		return nil
	}

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}

func (s *Server) bindRunningServer(server *http.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 这里显式串行化运行中服务指针的所有权，避免并发 Run / Shutdown
	// 在半完成状态上竞争，导致重复启动或错误清理。
	if s.server != nil {
		return errors.New("http server already running")
	}

	s.server = server
	return nil
}

func (s *Server) detachRunningServer() *http.Server {
	s.mu.Lock()
	defer s.mu.Unlock()

	server := s.server
	s.server = nil
	return server
}

func (s *Server) clearRunningServer(server *http.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 只清理由当前 Run 绑定的实例，避免并发失败路径误清除后来接管槽位的
	// 新服务指针。
	if s.server == server {
		s.server = nil
	}
}
