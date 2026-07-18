// Package requestctx 为 server 内部调用链保存请求关联快照。
package requestctx

import "context"

type auditContextKey struct{}

// AuditContext 描述随 context.Context 传递给 service 和 runtime 的关联字段。
type AuditContext struct {
	RequestID string
	TraceID   string
	Route     string
	Method    string
	ClientIP  string
	UserAgent string
}

// WithAuditContext 将一份 canonical 请求关联快照附加到 ctx。
func WithAuditContext(ctx context.Context, auditCtx AuditContext) context.Context {
	return context.WithValue(ctx, auditContextKey{}, auditCtx)
}

// AuditContextFromContext 从上下文链读取 canonical 请求关联快照。
func AuditContextFromContext(ctx context.Context) (AuditContext, bool) {
	if ctx == nil {
		return AuditContext{}, false
	}

	auditCtx, ok := ctx.Value(auditContextKey{}).(AuditContext)
	return auditCtx, ok
}
