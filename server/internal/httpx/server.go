// Package httpx 提供 `server` 运行时使用的 HTTP 服务封装。
package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
)

const (
	defaultServerReadHeaderTimeout = 5 * time.Second
	defaultServerShutdownTimeout   = 5 * time.Second
)

// Server 封装运行时使用的 Gin 引擎与 HTTP 服务实例。
//
// Server 负责把 `Run` / `Shutdown` 的生命周期归属集中到一个显式对象中，
// 避免并发启动或停止时出现状态竞争。Server 支持并发调用生命周期方法。
//
// Server 只管理 HTTP 外壳本身，不负责模块路由装配策略或业务中间件语义；
// 这些职责仍留在 app 与各模块边界内。
type Server struct {
	engine *gin.Engine
	mu     sync.Mutex
	repo   AccessLogRepository
	// server 持有当前运行中的 http.Server 指针，用于串行化 Run/Shutdown
	// 的所有权切换，避免重复关闭或重复启动同一个生命周期槽位。
	server *http.Server
}

// AccessLogOptions configures HTTP access-log persistence and process-log emission.
type AccessLogOptions struct {
	ConsolePolicy config.AccessLogConsolePolicy
	SlowThreshold time.Duration
}

// ServerOptions carries optional HTTP runtime behavior for NewServerWithOptions.
type ServerOptions struct {
	AccessLog AccessLogOptions
}

// NewServer 创建 MVP 运行时使用的最小 Gin 服务外壳。
//
// 返回的服务默认挂载全局 request-id、中台统一 access log 与恢复中间件，
// 便于 core 和模块在统一入口上继续注册路由。
func NewServer(logger *zap.Logger, repo ...AccessLogRepository) *Server {
	return NewServerWithOptions(logger, ServerOptions{
		AccessLog: AccessLogOptions{
			ConsolePolicy: config.AccessLogConsoleAlways,
			SlowThreshold: time.Second,
		},
	}, repo...)
}

// NewServerWithOptions creates the Gin server shell with explicit runtime options.
func NewServerWithOptions(logger *zap.Logger, options ServerOptions, repo ...AccessLogRepository) *Server {
	engine := gin.New()

	var accessLogRepo AccessLogRepository
	for _, candidate := range repo {
		if candidate != nil {
			accessLogRepo = candidate
			break
		}
	}

	engine.Use(RequestIDMiddleware(), newAccessLogMiddleware(logger, accessLogRepo, options.AccessLog), gin.Recovery())
	return &Server{engine: engine, repo: accessLogRepo}
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

// Run 启动 HTTP 服务，并把服务生命周期绑定到给定上下文。
//
// 当监听提前失败时直接返回错误；当上下文取消时，Run 会触发一次优雅关闭，
// 并等待监听 goroutine 退出后再返回。
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: defaultServerReadHeaderTimeout,
	}
	if err := s.bindRunningServer(srv); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		// ListenAndServe 正常关闭时会返回 http.ErrServerClosed，这里把它视为
		// 生命周期正常收敛，而不是需要继续向上传播的失败。
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err, ok := <-errCh:
		s.clearRunningServer(srv)
		if !ok {
			return nil
		}
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultServerShutdownTimeout)
		defer cancel()

		shutdownErr := s.Shutdown(shutdownCtx)
		<-errCh
		return shutdownErr
	}
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
