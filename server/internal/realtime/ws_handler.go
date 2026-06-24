package realtime

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/realtimeauth"
)

const (
	// WebSocketTopicScope is the bounded permission scope embedded in websocket topic tickets.
	WebSocketTopicScope = "realtime.topic.subscribe"
	// WebSocketTopicResourceType is the canonical resource type for unified realtime topics.
	WebSocketTopicResourceType = "realtime.topic"
	websocketBufferSize        = 4096
)

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  websocketBufferSize,
	WriteBufferSize: websocketBufferSize,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// GatewayRegistration defines unified realtime websocket gateway dependencies.
type GatewayRegistration struct {
	Hub                   Hub
	I18n                  *i18n.Service
	Tickets               realtimeauth.Service
	WebSocketAllowOrigins []string
}

// RegisterWebSocketGateway mounts the canonical unified realtime websocket route.
func RegisterWebSocketGateway(router gin.IRouter, registration GatewayRegistration) error {
	if router == nil {
		return errors.New("realtime router is unavailable")
	}
	if registration.Hub == nil {
		return errors.New("realtime hub is unavailable")
	}
	if registration.Tickets == nil {
		return errors.New("realtime ticket service is unavailable")
	}

	router.GET("/ws", func(ctx *gin.Context) {
		request, ok := parseGatewayRequest(ctx, registration)
		if !ok {
			return
		}
		if !consumeGatewayTicket(ctx, registration, request) {
			return
		}
		conn, err := websocketUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}
		_ = streamTopicEvents(ctx.Request.Context(), conn, registration.Hub, request.topic)
	})

	return nil
}

type gatewayRequest struct {
	topic  string
	ticket string
}

func parseGatewayRequest(ctx *gin.Context, registration GatewayRegistration) (gatewayRequest, bool) {
	request := gatewayRequest{
		topic:  NormalizeTopic(ctx.Query("topic")),
		ticket: strings.TrimSpace(ctx.Query("ticket")),
	}
	if request.topic == "" || request.ticket == "" {
		httpx.WriteLocalizedError(ctx, registration.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
			"field": "topic",
		})
		return gatewayRequest{}, false
	}
	if err := realtimeauth.ValidateOrigin(ctx.GetHeader("Origin"), registration.WebSocketAllowOrigins); err != nil {
		httpx.WriteLocalizedError(ctx, registration.I18n, http.StatusForbidden, messagecontract.AuthForbidden.String(), map[string]any{
			"topic": request.topic,
		})
		return gatewayRequest{}, false
	}
	return request, true
}

func consumeGatewayTicket(ctx *gin.Context, registration GatewayRegistration, request gatewayRequest) bool {
	_, err := registration.Tickets.Consume(ctx.Request.Context(), realtimeauth.ConsumeRequest{
		Ticket:       request.ticket,
		ResourceType: WebSocketTopicResourceType,
		ResourceID:   request.topic,
		Scope:        WebSocketTopicScope,
	})
	if err == nil {
		return true
	}
	httpx.WriteLocalizedError(ctx, registration.I18n, websocketTicketErrorStatus(err), messagecontract.AuthForbidden.String(), map[string]any{
		"topic": request.topic,
	})
	return false
}

func websocketTicketErrorStatus(err error) int {
	switch {
	case errors.Is(err, realtimeauth.ErrTicketRequired), errors.Is(err, realtimeauth.ErrInvalidTicket):
		return http.StatusBadRequest
	case errors.Is(err, realtimeauth.ErrExpiredTicket), errors.Is(err, realtimeauth.ErrUsedTicket):
		return http.StatusConflict
	default:
		return http.StatusForbidden
	}
}

func closeWebSocketConnection(conn *websocket.Conn) {
	if conn == nil {
		return
	}
	_ = conn.Close()
}

func streamTopicEvents(parent context.Context, conn *websocket.Conn, hub Hub, topic string) error {
	if conn == nil || hub == nil {
		return nil
	}
	defer closeWebSocketConnection(conn)

	eventCh, unsubscribe := hub.Subscribe(topic)
	defer unsubscribe()

	connectionCtx, cancel := context.WithCancel(parent)
	defer cancel()
	go watchWebSocketReads(conn, cancel)

	for {
		select {
		case <-connectionCtx.Done():
			return connectionCtx.Err()
		case event := <-eventCh:
			if err := conn.WriteJSON(event); err != nil {
				return err
			}
		}
	}
}

func watchWebSocketReads(conn *websocket.Conn, cancel context.CancelFunc) {
	if conn == nil || cancel == nil {
		return
	}

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			cancel()
			return
		}
	}
}
