package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestActiveRequestReaderExcludesCallingRequest(t *testing.T) {
	server := NewServer(zap.NewNop())
	reader := server.ActiveRequestReader()
	selfCount := make(chan int64, 1)
	release := make(chan struct{})
	done := make(chan struct{})
	server.Engine().GET("/hold", func(ctx *gin.Context) {
		selfCount <- reader.ReadActiveRequests(ctx.Request.Context())
		<-release
		ctx.Status(http.StatusNoContent)
	})

	go func() {
		defer close(done)
		server.Engine().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/hold", nil))
	}()
	if got := <-selfCount; got != 0 {
		t.Fatalf("expected the calling request to be excluded, got %d", got)
	}
	if got := reader.ReadActiveRequests(context.Background()); got != 1 {
		t.Fatalf("expected one externally visible active request, got %d", got)
	}
	close(release)
	<-done
	if got := reader.ReadActiveRequests(context.Background()); got != 0 {
		t.Fatalf("expected request completion to decrement the count, got %d", got)
	}
}

func TestActiveRequestTrackerCleansUpPanicAndNeverGoesNegative(t *testing.T) {
	server := NewServer(zap.NewNop())
	server.Engine().GET("/panic", func(*gin.Context) {
		panic("test panic")
	})
	server.Engine().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/panic", nil))

	tracker := server.activeRequests
	if got := tracker.ReadActiveRequests(context.Background()); got != 0 {
		t.Fatalf("expected panic recovery to release the active request, got %d", got)
	}
	_, done := tracker.begin(context.Background())
	done()
	done()
	tracker.decrement()
	if got := tracker.ReadActiveRequests(context.Background()); got != 0 {
		t.Fatalf("expected repeated cleanup to remain at zero, got %d", got)
	}
}

func TestActiveRequestTrackerExcludesWebSocketUpgrade(t *testing.T) {
	server := NewServer(zap.NewNop())
	reader := server.ActiveRequestReader()
	observed := make(chan int64, 1)
	server.Engine().GET("/ws", func(ctx *gin.Context) {
		observed <- reader.ReadActiveRequests(ctx.Request.Context())
		ctx.Status(http.StatusUnauthorized)
	})
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")
	server.Engine().ServeHTTP(httptest.NewRecorder(), request)

	if got := <-observed; got != 0 {
		t.Fatalf("expected websocket upgrade request to be excluded, got %d", got)
	}
}

func TestActiveRequestTrackerCountsConcurrentRequests(t *testing.T) {
	const requestCount = 32
	tracker := newActiveRequestTracker()
	started := make(chan struct{}, requestCount)
	release := make(chan struct{})
	var requests sync.WaitGroup
	requests.Add(requestCount)
	for range requestCount {
		go func() {
			defer requests.Done()
			_, done := tracker.begin(context.Background())
			defer done()
			started <- struct{}{}
			<-release
		}()
	}
	for range requestCount {
		<-started
	}
	if got := tracker.ReadActiveRequests(context.Background()); got != requestCount {
		t.Fatalf("expected %d concurrent requests, got %d", requestCount, got)
	}
	close(release)
	requests.Wait()
	if got := tracker.ReadActiveRequests(context.Background()); got != 0 {
		t.Fatalf("expected all concurrent requests to finish, got %d", got)
	}
}
