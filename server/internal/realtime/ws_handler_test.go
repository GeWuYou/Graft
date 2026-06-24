package realtime

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"graft/server/internal/realtimeauth"
)

func TestRegisterWebSocketGatewayStopsSubscriptionOnClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tickets := realtimeauth.NewMemoryService()
	topic := "topic.runtime.disconnect"
	issued, err := tickets.Issue(t.Context(), realtimeauth.IssueRequest{
		UserID:       1,
		ResourceType: WebSocketTopicResourceType,
		ResourceID:   topic,
		Scope:        WebSocketTopicScope,
	})
	if err != nil {
		t.Fatalf("issue websocket ticket: %v", err)
	}

	hub := NewHub()
	engine := gin.New()
	if err := RegisterWebSocketGateway(engine, GatewayRegistration{
		Hub:                   hub,
		Tickets:               tickets,
		WebSocketAllowOrigins: []string{"http://client.example"},
	}); err != nil {
		t.Fatalf("register websocket gateway: %v", err)
	}

	server := httptest.NewServer(engine)
	defer server.Close()

	wsURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server url: %v", err)
	}
	wsURL.Scheme = "ws"
	wsURL.Path = "/ws"
	query := wsURL.Query()
	query.Set("topic", topic)
	query.Set("ticket", issued.Ticket)
	wsURL.RawQuery = query.Encode()

	headers := http.Header{}
	headers.Set("Origin", "http://client.example")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), headers)
	if err != nil {
		t.Fatalf("dial websocket gateway: %v", err)
	}

	_, unsubscribe := hub.Subscribe(topic)
	unsubscribe()

	if err := conn.Close(); err != nil {
		t.Fatalf("close websocket client: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			hub.Publish(topic, i)
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected websocket disconnect to stop gateway subscription without blocking publishers")
	}
}
