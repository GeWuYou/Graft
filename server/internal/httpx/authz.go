package httpx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	actorHeader       = "X-Graft-Actor"
	permissionsHeader = "X-Graft-Permissions"
)

// Session captures the explicit request identity carried by the MVP shell.
//
// Until the auth and RBAC plugins land, protected routes use these headers as a
// visible server-side guard instead of silently trusting frontend route meta.
type Session struct {
	Actor       string
	Permissions map[string]struct{}
}

// HasPermission reports whether the current session owns the required code.
func (s Session) HasPermission(code string) bool {
	if strings.TrimSpace(code) == "" {
		return true
	}

	_, ok := s.Permissions[code]
	return ok
}

// SessionFromRequest parses the explicit MVP session headers from the request.
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

// RequirePermission enforces the explicit MVP authorization contract.
func RequirePermission(code string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := SessionFromRequest(ctx.Request)
		if session.Actor == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing request actor",
			})
			return
		}

		if !session.HasPermission(code) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":      "missing permission",
				"permission": code,
			})
			return
		}

		ctx.Next()
	}
}
