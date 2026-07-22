package mcp

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"graft/server/internal/httpx"
)

var errRuntimeOverloaded = errors.New("mcp runtime is busy")

// RuntimeMetrics 是 adapter 级别、可由进程监控采集的累计运行指标。
// 它不记录 Tool 参数、Token 或 REST 响应，避免把敏感业务输入写入日志或指标标签。
type RuntimeMetrics struct {
	RequestsTotal     uint64
	RequestsRejected  uint64
	InvocationsTotal  uint64
	InvocationsFailed uint64
	ActiveRequests    int64
	ActiveSessions    int64
}

type runtimeLifecycle struct {
	limits   RuntimeLimits
	logger   *zap.Logger
	sem      chan struct{}
	closed   atomic.Bool
	requests atomic.Uint64
	rejected atomic.Uint64
	calls    atomic.Uint64
	failures atomic.Uint64
	active   atomic.Int64

	mu       sync.Mutex
	sessions map[string]time.Time
}

func newRuntimeLifecycle(limits RuntimeLimits, logger *zap.Logger) (*runtimeLifecycle, error) {
	if limits.SessionTimeout <= 0 || limits.RequestTimeout <= 0 || limits.MaxRequestBytes <= 0 || limits.MaxSessions <= 0 || limits.MaxConcurrentRequests <= 0 {
		return nil, errors.New("mcp runtime limits must be greater than zero")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &runtimeLifecycle{
		limits:   limits,
		logger:   logger,
		sem:      make(chan struct{}, limits.MaxConcurrentRequests),
		sessions: make(map[string]time.Time),
	}, nil
}

func (l *runtimeLifecycle) Close() {
	if l == nil {
		return
	}
	l.closed.Store(true)
}

func (l *runtimeLifecycle) Metrics() RuntimeMetrics {
	if l == nil {
		return RuntimeMetrics{}
	}
	l.mu.Lock()
	l.removeExpiredSessionsLocked(time.Now())
	sessions := len(l.sessions)
	l.mu.Unlock()
	return RuntimeMetrics{
		RequestsTotal: l.requests.Load(), RequestsRejected: l.rejected.Load(),
		InvocationsTotal: l.calls.Load(), InvocationsFailed: l.failures.Load(),
		ActiveRequests: l.active.Load(), ActiveSessions: int64(sessions),
	}
}

func (l *runtimeLifecycle) httpHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if l == nil || l.closed.Load() {
			http.Error(w, "MCP runtime unavailable", http.StatusServiceUnavailable)
			return
		}
		l.requests.Add(1)
		l.active.Add(1)
		defer l.active.Add(-1)
		if request.Method == http.MethodPost && request.Header.Get("Mcp-Session-Id") == "" && !l.canOpenSession() {
			l.reject("session_limit")
			http.Error(w, "MCP session limit reached", http.StatusTooManyRequests)
			return
		}
		select {
		case l.sem <- struct{}{}:
			defer func() { <-l.sem }()
		default:
			l.reject("concurrency_limit")
			http.Error(w, "MCP runtime is busy", http.StatusTooManyRequests)
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, l.limits.MaxRequestBytes)
		ctx, cancel := context.WithTimeout(request.Context(), l.limits.RequestTimeout)
		defer cancel()
		recorder := &responseStatusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, request.WithContext(ctx))
		l.trackHTTPSession(request, recorder)
	})
}

func (l *runtimeLifecycle) canOpenSession() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.removeExpiredSessionsLocked(time.Now())
	return len(l.sessions) < l.limits.MaxSessions
}

func (l *runtimeLifecycle) trackHTTPSession(request *http.Request, response *responseStatusWriter) {
	if l == nil || request == nil || response == nil {
		return
	}
	sessionID := response.Header().Get("Mcp-Session-Id")
	if request.Method == http.MethodDelete {
		sessionID = request.Header.Get("Mcp-Session-Id")
		l.mu.Lock()
		delete(l.sessions, sessionID)
		l.mu.Unlock()
		return
	}
	if request.Method != http.MethodPost || response.status >= http.StatusMultipleChoices || sessionID == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.removeExpiredSessionsLocked(now)
	if _, exists := l.sessions[sessionID]; exists {
		l.sessions[sessionID] = now.Add(l.limits.SessionTimeout)
		return
	}
	l.sessions[sessionID] = now.Add(l.limits.SessionTimeout)
}

func (l *runtimeLifecycle) removeExpiredSessionsLocked(now time.Time) {
	for id, expiresAt := range l.sessions {
		if !expiresAt.After(now) {
			delete(l.sessions, id)
		}
	}
}

func (l *runtimeLifecycle) toolHandler(name string, next mcpsdk.ToolHandler) mcpsdk.ToolHandler {
	return func(ctx context.Context, request *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		return l.invoke(ctx, "tool", name, func(ctx context.Context) (*mcpsdk.CallToolResult, error) { return next(ctx, request) })
	}
}

func (l *runtimeLifecycle) resourceHandler(name string, next mcpsdk.ResourceHandler) mcpsdk.ResourceHandler {
	return func(ctx context.Context, request *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		if err := l.acquire(ctx); err != nil {
			return nil, err
		}
		defer l.release()
		callCtx, cancel := context.WithTimeout(ctx, l.limits.RequestTimeout)
		defer cancel()
		result, err := next(callCtx, request)
		l.complete(callCtx, "resource", name, err != nil)
		return result, err
	}
}

func (l *runtimeLifecycle) invoke(ctx context.Context, kind, name string, call func(context.Context) (*mcpsdk.CallToolResult, error)) (*mcpsdk.CallToolResult, error) {
	if err := l.acquire(ctx); err != nil {
		return toolErrorResult(err), nil
	}
	defer l.release()
	callCtx, cancel := context.WithTimeout(ctx, l.limits.RequestTimeout)
	defer cancel()
	result, err := call(callCtx)
	failed := err != nil || result == nil || result.IsError
	l.complete(callCtx, kind, name, failed)
	return result, err
}

func (l *runtimeLifecycle) acquire(ctx context.Context) error {
	if l == nil || l.closed.Load() {
		return errors.New("mcp runtime unavailable")
	}
	select {
	case l.sem <- struct{}{}:
		l.calls.Add(1)
		return nil
	case <-ctx.Done():
		l.reject("deadline")
		return ctx.Err()
	default:
		l.reject("concurrency_limit")
		return errRuntimeOverloaded
	}
}

func (l *runtimeLifecycle) release() { <-l.sem }

func (l *runtimeLifecycle) complete(ctx context.Context, kind, name string, failed bool) {
	if failed {
		l.failures.Add(1)
	}
	fields := []zap.Field{zap.String("transport", "adapter"), zap.String("kind", kind), zap.String("capability", name), zap.Bool("failed", failed)}
	if audit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		fields = append(fields, zap.String("request_id", audit.RequestID), zap.String("trace_id", audit.TraceID))
	}
	l.logger.Info("mcp invocation completed", fields...)
}

func (l *runtimeLifecycle) reject(reason string) {
	l.rejected.Add(1)
	l.logger.Warn("mcp request rejected", zap.String("reason", reason))
}

type responseStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseStatusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStatusWriter) Write(body []byte) (int, error) {
	return w.ResponseWriter.Write(body)
}
