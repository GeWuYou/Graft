package mcp

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/moduleapi"
)

func TestRunStdioUsesCompiledCapabilitiesAndRESTDispatcher(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/api/items/:id", func(ctx *gin.Context) {
		caller, ok := moduleapi.PersonalAccessTokenCallerFromContext(ctx.Request.Context())
		if !ok || caller.User.ID != 7 {
			t.Fatalf("stdio dispatcher lost authenticated caller: %#v", caller)
		}
		ctx.JSON(http.StatusOK, gin.H{"id": ctx.Param("id"), "transport": "rest"})
	})
	clientToServerReader, clientToServerWriter := io.Pipe()
	serverToClientReader, serverToClientWriter := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- RunStdio(ctx, StdioRegistration{
			Engine: engine,
			OpenAPISpec: compilerTestBundle(map[string]any{
				"/api/items/{id}": map[string]any{"get": compilerTestOperation("getItem", compilerTestMetadata("low", false), false)},
			}),
			Authorizer:           &foundationTestAuthorizer{},
			Caller:               moduleapi.PersonalAccessTokenCaller{TokenID: 42, User: moduleapi.CurrentUser{ID: 7, Username: "alice"}, Scopes: []string{"item.read"}, ExpiresAt: time.Now().Add(time.Hour)},
			ConfirmationTokenTTL: time.Minute,
			Limits:               testRuntimeLimits(),
			Reader:               clientToServerReader,
			Writer:               serverToClientWriter,
		})
	}()
	reader := bufio.NewReader(serverToClientReader)
	writeStdioMessage(t, clientToServerWriter, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`)
	if line := readStdioMessage(t, reader); !strings.Contains(line, `"tools"`) || !strings.Contains(line, `"resources"`) {
		t.Fatalf("stdio initialize did not advertise compiled capabilities: %s", line)
	}
	writeStdioMessage(t, clientToServerWriter, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	writeStdioMessage(t, clientToServerWriter, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	if line := readStdioMessage(t, reader); !strings.Contains(line, `"get_item"`) {
		t.Fatalf("stdio tools/list did not expose compiled tool: %s", line)
	}
	writeStdioMessage(t, clientToServerWriter, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_item","arguments":{"id":"item-7"}}}`)
	if line := readStdioMessage(t, reader); !strings.Contains(line, `"id":"item-7"`) || !strings.Contains(line, `"transport":"rest"`) {
		t.Fatalf("stdio tool result must preserve REST payload: %s", line)
	}
	cancel()
	_ = clientToServerWriter.Close()
	_ = serverToClientReader.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stdio runtime did not stop after cancellation")
	}
}

func writeStdioMessage(t *testing.T, writer *io.PipeWriter, message string) {
	t.Helper()
	if _, err := io.WriteString(writer, message+"\n"); err != nil {
		t.Fatalf("write stdio message: %v", err)
	}
}

func readStdioMessage(t *testing.T, reader *bufio.Reader) string {
	t.Helper()
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read stdio message: %v", err)
	}
	return line
}
