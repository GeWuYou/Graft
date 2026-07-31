package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SSEEventName 是统一 SSE 网关向客户端发送实时主题事件时使用的事件名。
const SSEEventName = "message"

const sseHeartbeatInterval = 10 * time.Second

// RegisterSSEGateway 注册统一实时 SSE 入口。它与 WebSocket 网关使用相同的 topic、票据和来源校验规则。
func RegisterSSEGateway(router gin.IRouter, registration GatewayRegistration) error {
	if router == nil {
		return errors.New("realtime router is unavailable")
	}
	if registration.Hub == nil {
		return errors.New("realtime hub is unavailable")
	}
	if registration.Tickets == nil {
		return errors.New("realtime ticket service is unavailable")
	}
	router.GET("/sse", func(ctx *gin.Context) {
		request, ok := parseGatewayRequest(ctx, registration)
		if !ok || !consumeGatewayTicket(ctx, registration, request) {
			return
		}
		_ = StreamSSE(ctx.Request.Context(), ctx.Writer, registration.Hub, request.topic)
	})
	return nil
}

// StreamSSE 将一个已获授权的主题订阅写入标准 Server-Sent Events 响应。
// 路由、认证和主题授权由统一网关拥有；此函数只负责安全地桥接 Hub 与 HTTP 流。
func StreamSSE(ctx context.Context, writer http.ResponseWriter, hub Hub, topic string) error {
	if ctx == nil || writer == nil || hub == nil || NormalizeTopic(topic) == "" {
		return errors.New("realtime SSE stream is unavailable")
	}

	events, unsubscribe := hub.Subscribe(topic)
	defer unsubscribe()

	flusher, err := prepareSSEStream(ctx, writer, hub, topic)
	if err != nil {
		return err
	}

	return streamSSEEvents(ctx, writer, flusher, events)
}

func prepareSSEStream(ctx context.Context, writer http.ResponseWriter, hub Hub, topic string) (http.Flusher, error) {
	if ctx == nil || writer == nil || hub == nil || NormalizeTopic(topic) == "" {
		return nil, errors.New("realtime SSE stream is unavailable")
	}
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return nil, errors.New("realtime SSE response does not support flushing")
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache, no-transform")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)
	flusher.Flush()
	return flusher, nil
}

func streamSSEEvents(ctx context.Context, writer http.ResponseWriter, flusher http.Flusher, events <-chan Event) error {
	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-heartbeat.C:
			if err := writeSSEHeartbeat(writer, flusher); err != nil {
				return err
			}
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if err := writeSSEEvent(writer, flusher, event); err != nil {
				return err
			}
		}
	}
}

func writeSSEHeartbeat(writer http.ResponseWriter, flusher http.Flusher) error {
	if _, err := fmt.Fprint(writer, ": keep-alive\n\n"); err != nil {
		return fmt.Errorf("write realtime SSE heartbeat: %w", err)
	}
	flusher.Flush()
	return nil
}

func writeSSEEvent(writer http.ResponseWriter, flusher http.Flusher, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode realtime SSE event: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", SSEEventName, payload); err != nil {
		return fmt.Errorf("write realtime SSE event: %w", err)
	}
	flusher.Flush()
	return nil
}
