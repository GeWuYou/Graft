// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
	"graft/server/modules/container/terminal"
)

var shellWebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

func (r routeRuntime) handleShellWebSocket(ginCtx *gin.Context) {
	ref, ok := readRef(ginCtx, r)
	if !ok {
		return
	}
	requestCtx, requestAuth, handled := r.authenticateShellWebSocketRequest(ginCtx)
	if handled {
		return
	}
	handshakeValue, exists := ginCtx.Get("container.shell.handshake")
	if !exists {
		r.writeRouteError(ginCtx, errShellSessionFailed)
		return
	}
	handshake, ok := handshakeValue.(ShellHandshake)
	if !ok {
		r.writeRouteError(ginCtx, errShellSessionFailed)
		return
	}
	if requestAuth.User == nil || requestAuth.User.ID != handshake.UserID {
		r.writeRouteError(ginCtx, errShellForbidden)
		return
	}

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
		_ = conn.Close()
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

	userService, err := resolveUserService(r.ctx)
	if err != nil {
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
	userSummary, err := userService.GetUserByID(requestCtx, handshake.UserID)
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

func isShellDisconnectError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
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
