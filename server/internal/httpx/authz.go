package httpx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/i18n"
)

const (
	actorHeader       = "X-Graft-Actor"
	permissionsHeader = "X-Graft-Permissions"
)

// Session 表示 MVP 阶段请求携带的显式身份信息。
//
// 在真实 auth 与 RBAC 插件落地之前，受保护路由通过这些请求头进行可见
// 的后端权限校验，而不是隐式信任前端路由元数据。
type Session struct {
	Actor       string
	Permissions map[string]struct{}
}

// HasPermission 判断当前会话是否拥有所需权限码。
//
// 空权限码被视为无需额外授权，方便路由层复用同一守卫函数处理“仅要求
// 已登录”和“要求具体权限”两类场景。
func (s Session) HasPermission(code string) bool {
	if strings.TrimSpace(code) == "" {
		return true
	}

	_, ok := s.Permissions[code]
	return ok
}

// SessionFromRequest 从请求中解析 MVP 阶段的显式会话头。
//
// 解析过程会去除空白并忽略空权限项，保证后续权限判断只面对规范化后的
// 权限集合。
func SessionFromRequest(request *http.Request) Session {
	session := Session{
		Actor:       strings.TrimSpace(request.Header.Get(actorHeader)),
		Permissions: make(map[string]struct{}),
	}

	for _, raw := range strings.Split(request.Header.Get(permissionsHeader), ",") {
		permission := strings.TrimSpace(raw)
		if permission == "" {
			continue
		}
		session.Permissions[permission] = struct{}{}
	}

	return session
}

// RequirePermission 执行当前 MVP 阶段的显式权限校验。
//
// 这个中间件只负责后端显式鉴权，不负责构造真实登录态；缺少身份头返回
// 401，身份存在但权限不足返回 403。
func RequirePermission(localizer *i18n.Service, code string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := SessionFromRequest(ctx.Request)
		if session.Actor == "" {
			// 先拒绝匿名请求，避免后续处理链把“未提供调用者”误判成普通的
			// 权限不足场景。
			AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.missing_actor", nil)
			return
		}

		if !session.HasPermission(code) {
			AbortLocalizedError(ctx, localizer, http.StatusForbidden, "auth.missing_permission", map[string]any{
				"permission": code,
			})
			return
		}

		ctx.Next()
	}
}
