package dockerexecution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type pullProgressObservation struct {
	created atomic.Bool
	started atomic.Bool
}

func newPullProgressEngine(t *testing.T, body string) (*httptest.Server, *pullProgressObservation) {
	t.Helper()
	observation := &pullProgressObservation{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.Header().Set("API-Version", "1.55")
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/images/create"):
			_, _ = w.Write([]byte(body))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/create"):
			observation.created.Store(true)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id":"controller","Warnings":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/containers/controller/start"):
			observation.started.Store(true)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected Docker request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server, observation
}

func controllerProgressEnvironment(t *testing.T, socket string) Environment {
	t.Helper()
	material := updateControllerMaterial{
		ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64),
		InputBase64:         "encoded", ComposeRoot: "/opt/graft", DockerSocket: "/var/run/docker.sock", StateVolume: "graft-update-state", OperationID: "update-progress",
	}
	payload, err := json.Marshal(material)
	if err != nil {
		t.Fatal(err)
	}
	return Environment{DockerSocket: socket, ResolveMaterial: func(context.Context, Request) (Material, error) {
		return Material{Protocol: updateControllerMaterialProtocol, Payload: payload}, nil
	}}
}

func TestImagePullRejectsHTTP200ErrorProgress(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		body   string
		failed bool
	}{
		{"completed", "{\"status\":\"Pulling\"}\n{\"status\":\"Download complete\"}\n", false},
		{"error-detail", `{"status":"Pulling"}` + "\n" + `{"errorDetail":{"message":"synthetic-secret"},"error":"synthetic-secret"}` + "\n", true},
		{"legacy-error", `{"error":"synthetic-secret"}` + "\n", true},
		{"malformed", "{", true},
	} {
		t.Run(testCase.name+"/image", func(t *testing.T) { assertPullProgressOutcome(t, testCase.body, testCase.failed, false) })
		t.Run(testCase.name+"/controller", func(t *testing.T) { assertPullProgressOutcome(t, testCase.body, testCase.failed, true) })
	}
}

func assertPullProgressOutcome(t *testing.T, body string, failed, controller bool) {
	t.Helper()
	server, observed := newPullProgressEngine(t, body)
	defer server.Close()
	environment := Environment{DockerSocket: server.URL}
	request := Request{Capability: "container_execution", Protocol: containerExecutionProtocol, OperationID: "container.image.pull.v1", Input: json.RawMessage(`{"image_ref":"example/image:latest"}`)}
	if controller {
		environment = controllerProgressEnvironment(t, server.URL)
		request = Request{Capability: updateControllerCapability, Protocol: updateControllerProtocol, OperationID: updateControllerOperation, Input: json.RawMessage(`{"operation_id":"update-progress"}`)}
	}
	result := Execute(context.Background(), environment, request)
	if (result.Outcome != "success") != failed {
		t.Fatalf("HTTP200 progress result=%#v failed=%v", result, failed)
	}
	if strings.Contains(fmt.Sprint(result), "synthetic-secret") {
		t.Fatalf("raw provider error escaped result: %#v", result)
	}
	if controller && (observed.created.Load() != !failed || observed.started.Load() != !failed) {
		t.Fatalf("controller effects after pull created=%v started=%v", observed.created.Load(), observed.started.Load())
	}
}

type progressReadCloser struct {
	io.Reader
	closed   bool
	closeErr error
}

func (r *progressReadCloser) Close() error { r.closed = true; return r.closeErr }

func TestDockerProgressClosesStreamsAndRedactsFailures(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		body     string
		closeErr error
		failed   bool
	}{
		{"completed", `{"status":"complete"}`, nil, false},
		{"error-detail", `{"errorDetail":{"message":"synthetic-secret"}}`, nil, true},
		{"legacy-error", `{"error":"synthetic-secret"}`, nil, true},
		{"close-failure", `{"status":"complete"}`, errors.New("synthetic-secret"), true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			body := &progressReadCloser{Reader: strings.NewReader(testCase.body), closeErr: testCase.closeErr}
			err := consumeDockerProgress(body)
			if !body.closed || (err != nil) != testCase.failed {
				t.Fatalf("closed=%v err=%v", body.closed, err)
			}
			if err != nil && strings.Contains(err.Error(), "synthetic-secret") {
				t.Fatal("provider text escaped diagnostic")
			}
		})
	}
}

func TestImagePullProgressRespondsToCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("API-Version", "1.55")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"Pulling"}` + "\n"))
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan Result, 1)
	go func() {
		result <- Execute(ctx, Environment{DockerSocket: server.URL}, Request{Capability: "container_execution", Protocol: containerExecutionProtocol, OperationID: "container.image.pull.v1", Input: json.RawMessage(`{"image_ref":"example/image:latest"}`)})
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("pull progress did not start")
	}
	cancel()
	select {
	case outcome := <-result:
		if outcome.Outcome == "success" {
			t.Fatal("cancelled pull returned success")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled pull did not return")
	}
}
