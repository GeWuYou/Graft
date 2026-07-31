package realtime

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"graft/server/internal/realtimeauth"
)

func TestRegisterSSEGatewayStreamsAuthorizedTopicEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tickets := realtimeauth.NewMemoryService()
	topic := "topic.runtime.sse"
	issued, err := tickets.Issue(t.Context(), realtimeauth.IssueRequest{UserID: 1, ResourceType: WebSocketTopicResourceType, ResourceID: topic, Scope: WebSocketTopicScope})
	if err != nil {
		t.Fatalf("issue SSE ticket: %v", err)
	}
	hub := NewHub()
	memoryHub, ok := hub.(*memoryHub)
	if !ok {
		t.Fatal("expected memory hub implementation")
	}
	engine := gin.New()
	if err := RegisterSSEGateway(engine, GatewayRegistration{Hub: hub, Tickets: tickets, WebSocketAllowOrigins: []string{"http://client.example"}}); err != nil {
		t.Fatalf("register SSE gateway: %v", err)
	}
	server := httptest.NewServer(engine)
	defer server.Close()
	requestContext, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, server.URL+"/sse?topic="+url.QueryEscape(topic)+"&ticket="+url.QueryEscape(issued.Ticket), nil)
	if err != nil {
		t.Fatalf("create SSE request: %v", err)
	}
	request.Header.Set("Origin", "http://client.example")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("open SSE stream: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected SSE response: %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	waitForTopicSubscriberCount(t, memoryHub, topic, 1)
	hub.Publish(topic, map[string]string{"status": "PULLING"})
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil || line != "event: message\n" {
		t.Fatalf("event line = %q, %v", line, err)
	}
	line, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("data line: %v", err)
	}
	var event Event
	if err := json.Unmarshal([]byte(line[len("data: "):]), &event); err != nil {
		t.Fatalf("decode SSE event: %v", err)
	}
	if event.Topic != topic {
		t.Fatalf("event topic = %q", event.Topic)
	}
}

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
