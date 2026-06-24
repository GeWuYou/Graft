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
	memoryHub, ok := hub.(*memoryHub)
	if !ok {
		t.Fatal("expected memory hub implementation")
	}
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

	waitForTopicSubscriberCount(t, memoryHub, topic, 1)

	if err := conn.Close(); err != nil {
		t.Fatalf("close websocket client: %v", err)
	}

	waitForTopicSubscriberCount(t, memoryHub, topic, 0)
}

func waitForTopicSubscriberCount(t *testing.T, hub *memoryHub, topic string, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := topicSubscriberCount(hub, topic); got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected %d subscribers for topic %q, got %d", want, topic, topicSubscriberCount(hub, topic))
}

func topicSubscriberCount(hub *memoryHub, topic string) int {
	if hub == nil {
		return 0
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.topics[topic])
}
