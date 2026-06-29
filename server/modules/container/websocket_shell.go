package container

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
	"graft/server/modules/container/terminal"
)

const shellWebSocketBufferSize = 4096

var shellWebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  shellWebSocketBufferSize,
	WriteBufferSize: shellWebSocketBufferSize,
	CheckOrigin:     func(*http.Request) bool { return true },
}

func (r routeRuntime) handleShellWebSocket(ginCtx *gin.Context) {
	requestCtx, requestAuth, handled := r.authenticateShellWebSocketRequest(ginCtx)
	if handled {
		return
	}
	ref, handshake, ok := r.resolveShellWebSocketContext(ginCtx, requestAuth)
	if !ok {
		return
	}
	r.runShellWebSocketBridge(requestCtx, ginCtx, ref, handshake)
}

// shellHandshakeFromContext 从 Gin 上下文中检索 shell 握手。
// 若握手存在且类型正确，返回握手和 true；否则返回零值握手和 false。
func shellHandshakeFromContext(ginCtx *gin.Context) (ShellHandshake, bool) {
	handshakeValue, exists := ginCtx.Get("container.shell.handshake")
	if !exists {
		return ShellHandshake{}, false
	}
	handshake, ok := handshakeValue.(ShellHandshake)
	if !ok {
		return ShellHandshake{}, false
	}
	return handshake, true
}

func (r routeRuntime) resolveShellWebSocketContext(
	ginCtx *gin.Context,
	requestAuth moduleapi.RequestAuthContext,
) (Ref, ShellHandshake, bool) {
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return Ref{}, ShellHandshake{}, false
	}
	handshake, ok := shellHandshakeFromContext(ginCtx)
	if !ok {
		return Ref{}, ShellHandshake{}, false
	}
	if requestAuth.User == nil || requestAuth.User.ID != handshake.UserID {
		r.writeRouteError(ginCtx, errShellForbidden)
		return Ref{}, ShellHandshake{}, false
	}
	return ref, handshake, true
}

func (r routeRuntime) runShellWebSocketBridge(
	requestCtx context.Context,
	ginCtx *gin.Context,
	ref Ref,
	handshake ShellHandshake,
) {
	conn, err := shellWebSocketUpgrader.Upgrade(ginCtx.Writer, ginCtx.Request, nil)
	if err != nil {
		r.service.publishShellSessionFailed(
			requestCtx,
			handshake,
			"websocket_upgrade_failed",
			errShellSessionFailed,
		)
		return
	}

	session, err := r.service.OpenShellTerminalSession(requestCtx, ref, handshake)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		r.service.publishShellSessionFailed(requestCtx, handshake, "session_open_failed", err)
		return
	}

	bridge := terminal.NewBridge(conn, session)
	bridgeCtx, cancel := context.WithCancel(requestCtx)
	defer cancel()
	startedAt := time.Now().UTC()
	runErr := bridge.Run(bridgeCtx, terminal.Size{
		Cols: uint(handshake.Cols),
		Rows: uint(handshake.Rows),
	})
	if runErr != nil && !errors.Is(runErr, context.Canceled) && !isShellDisconnectError(runErr) {
		r.service.publishShellSessionFailed(requestCtx, handshake, "bridge_failed", runErr)
		return
	}
	r.service.publishShellSessionClosed(requestCtx, handshake, startedAt, "client_closed", nil)
}

func (r routeRuntime) authenticateShellWebSocketRequest(ginCtx *gin.Context) (context.Context, moduleapi.RequestAuthContext, bool) {
	requestID := httpx.EnsureRequestID(ginCtx)
	traceID := httpx.EnsureTraceID(ginCtx)
	requestCtx := httpx.WithRequestAuditContext(ginCtx.Request.Context(), httpx.RequestAuditContext{
		RequestID: requestID,
		TraceID:   traceID,
		Route:     ginCtx.FullPath(),
		Method:    ginCtx.Request.Method,
		ClientIP:  ginCtx.ClientIP(),
		UserAgent: ginCtx.Request.UserAgent(),
	})

	if r.userService == nil {
		httpx.WriteLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
		return nil, moduleapi.RequestAuthContext{}, true
	}
	authorizer, err := resolveAuthorizer(r.ctx)
	if err != nil {
		httpx.WriteLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
		return nil, moduleapi.RequestAuthContext{}, true
	}
	params := bindGetContainerShellWebSocketParams(ginCtx)
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return nil, moduleapi.RequestAuthContext{}, true
	}
	handshake, err := r.service.ConsumeShellSessionTicket(
		requestCtx,
		ref,
		params.Ticket,
		ginCtx.GetHeader("Origin"),
	)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return nil, moduleapi.RequestAuthContext{}, true
	}
	userSummary, err := r.userService.GetUserByID(requestCtx, handshake.UserID)
	if err != nil {
		httpx.WriteLocalizedError(ginCtx, r.ctx.I18n, http.StatusForbidden, messagecontract.AuthForbidden.String(), nil)
		return nil, moduleapi.RequestAuthContext{}, true
	}
	requestAuth := moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{
			ID:          userSummary.ID,
			Username:    userSummary.Username,
			DisplayName: userSummary.Display,
		},
	}
	requestCtx = moduleapi.WithRequestAuthContext(requestCtx, requestAuth)
	if err := authorizer.Authorize(requestCtx, requestAuth, containercontract.ContainerShellPermission.String()); err != nil {
		httpx.WriteLocalizedError(ginCtx, r.ctx.I18n, http.StatusForbidden, messagecontract.AuthForbidden.String(), map[string]any{
			"permission": containercontract.ContainerShellPermission.String(),
		})
		return nil, moduleapi.RequestAuthContext{}, true
	}
	ginCtx.Set("container.shell.handshake", handshake)
	return requestCtx, requestAuth, false
}

// isShellDisconnectError reports whether an error is a shell disconnection.
// It returns true if the error is EOF, a closed connection, or a WebSocket close code,
// and false otherwise.
func isShellDisconnectError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	return websocket.IsCloseError(
		err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
		websocket.CloseAbnormalClosure,
	)
}
