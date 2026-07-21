package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
)

func TestScopeGateRunsRBACBeforeTokenScope(t *testing.T) {
	caller := caller{tokenID: 9, user: moduleapi.CurrentUser{ID: 7}, scopes: nil, expiresAt: time.Now().Add(time.Hour).Unix()}
	ctx := withCaller(context.Background(), caller)
	authorizer := &foundationTestAuthorizer{err: moduleapi.ErrPermissionDenied}
	gate := ScopeGate{authorizer: authorizer}

	err := gate.Authorize(ctx, "audit.read", "audit.read")
	if !errors.Is(err, moduleapi.ErrPermissionDenied) {
		t.Fatalf("authorize = %v, want RBAC denial", err)
	}
	if authorizer.calls != 1 {
		t.Fatalf("expected RBAC authorizer to run before scope check, got %d calls", authorizer.calls)
	}

	authorizer.err = nil
	err = gate.Authorize(ctx, "audit.read", "audit.read")
	if !errors.Is(err, moduleapi.ErrPermissionDenied) {
		t.Fatalf("authorize without token scope = %v, want scope denial", err)
	}
	if authorizer.calls != 2 {
		t.Fatalf("expected RBAC authorizer before each scope check, got %d calls", authorizer.calls)
	}
}

func TestConfirmationTokensAreCallerBoundAndSingleUse(t *testing.T) {
	now := time.Date(2026, time.July, 21, 10, 0, 0, 0, time.UTC)
	confirmations, err := newConfirmationTokens(time.Minute)
	if err != nil {
		t.Fatalf("new confirmation tokens: %v", err)
	}
	confirmations.now = func() time.Time { return now }
	first := withCaller(context.Background(), caller{tokenID: 3})
	second := withCaller(context.Background(), caller{tokenID: 4})

	token, err := confirmations.Issue(first, "task.delete", "request-1")
	if err != nil {
		t.Fatalf("issue confirmation: %v", err)
	}
	if err := confirmations.Consume(second, token, "task.delete", "request-1"); !errors.Is(err, errConfirmationTokenInvalid) {
		t.Fatalf("cross-caller consume = %v, want invalid confirmation", err)
	}
	if err := confirmations.Consume(first, token, "task.delete", "request-1"); !errors.Is(err, errConfirmationTokenInvalid) {
		t.Fatalf("replayed confirmation = %v, want invalid confirmation", err)
	}
}

func TestConfirmationTokensExpireBeforeConsumption(t *testing.T) {
	now := time.Date(2026, time.July, 21, 10, 0, 0, 0, time.UTC)
	confirmations, err := newConfirmationTokens(time.Minute)
	if err != nil {
		t.Fatalf("new confirmation tokens: %v", err)
	}
	confirmations.now = func() time.Time { return now }
	callerContext := withCaller(context.Background(), caller{tokenID: 3})

	token, err := confirmations.Issue(callerContext, "task.delete", "request-1")
	if err != nil {
		t.Fatalf("issue confirmation: %v", err)
	}
	now = now.Add(time.Minute)

	if err := confirmations.Consume(callerContext, token, "task.delete", "request-1"); !errors.Is(err, errConfirmationTokenExpired) {
		t.Fatalf("consume expired confirmation = %v, want expired error", err)
	}
}

func TestRegisterServesStreamableReadToolsFromOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	tokenService := foundationPersonalAccessTokenService{
		caller: moduleapi.PersonalAccessTokenCaller{
			TokenID:   42,
			User:      moduleapi.CurrentUser{ID: 7, Username: "alice"},
			Scopes:    []string{"audit.read"},
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	if err := Register(HTTPRegistration{
		Engine:               engine,
		OpenAPISpec:          compilerTestBundle(map[string]any{"/api/items/{id}": map[string]any{"get": compilerTestOperation("getItem", compilerTestMetadata("low", false), false)}}),
		PersonalTokenService: tokenService,
		Authorizer:           &foundationTestAuthorizer{},
		ConfirmationTokenTTL: time.Minute,
	}); err != nil {
		t.Fatalf("register streamable MCP foundation: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, HTTPPath, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	request.Header.Set("Authorization", "Bearer gpat_verified_by_httpx")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("initialize status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Result struct {
			Capabilities map[string]json.RawMessage `json:"capabilities"`
		} `json:"result"`
	}
	if err := json.Unmarshal(streamableResponsePayload(recorder.Body.String()), &response); err != nil {
		t.Fatalf("decode initialize response: %v; body=%s", err, recorder.Body.String())
	}
	if _, ok := response.Result.Capabilities["tools"]; !ok {
		t.Fatalf("compiled read tools must advertise tools capability: %s", recorder.Body.String())
	}
	if _, ok := response.Result.Capabilities["resources"]; !ok {
		t.Fatalf("OpenAPI resource projection must advertise resources capability: %s", recorder.Body.String())
	}
	if _, ok := response.Result.Capabilities["prompts"]; ok {
		t.Fatalf("resource/action batch must not expose prompts capability: %s", recorder.Body.String())
	}

	sessionID := recorder.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("initialize response must create a streamable session")
	}
	listRequest := httptest.NewRequest(http.MethodPost, HTTPPath, strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`))
	listRequest.Header.Set("Authorization", "Bearer gpat_verified_by_httpx")
	listRequest.Header.Set("Content-Type", "application/json")
	listRequest.Header.Set("Accept", "application/json, text/event-stream")
	listRequest.Header.Set("Mcp-Session-Id", sessionID)
	listRecorder := httptest.NewRecorder()
	engine.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d, want %d: %s", listRecorder.Code, http.StatusOK, listRecorder.Body.String())
	}
	var toolList struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(streamableResponsePayload(listRecorder.Body.String()), &toolList); err != nil {
		t.Fatalf("decode tools/list response: %v; body=%s", err, listRecorder.Body.String())
	}
	if len(toolList.Result.Tools) != 1 || toolList.Result.Tools[0].Name != "get_item" {
		t.Fatalf("tools/list must contain only OpenAPI-compiled get_item: %#v", toolList.Result.Tools)
	}
	closeRequest := httptest.NewRequest(http.MethodDelete, HTTPPath, nil)
	closeRequest.Header.Set("Authorization", "Bearer gpat_verified_by_httpx")
	closeRequest.Header.Set("Mcp-Session-Id", sessionID)
	closeRecorder := httptest.NewRecorder()
	engine.ServeHTTP(closeRecorder, closeRequest)
	if closeRecorder.Code != http.StatusNoContent {
		t.Fatalf("close status = %d, want %d: %s", closeRecorder.Code, http.StatusNoContent, closeRecorder.Body.String())
	}
}

func TestStreamableToolCallPreservesRESTAuthorizationAndAuditBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	authorizer := &foundationTestAuthorizer{}
	auditEvents := 0
	engine.GET("/api/items/:id", httpx.RequirePermission(nil, nil, authorizer, "item.read"), func(ginCtx *gin.Context) {
		caller, ok := moduleapi.PersonalAccessTokenCallerFromContext(ginCtx.Request.Context())
		if !ok || caller.User.ID != 7 {
			t.Fatalf("REST handler lost personal token caller: %#v", caller)
		}
		requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
		if !ok || requestAuth.User == nil || requestAuth.User.ID != caller.User.ID {
			t.Fatalf("REST handler lost request auth context: %#v", requestAuth)
		}
		auditEvents++
		ginCtx.JSON(http.StatusOK, gin.H{"id": ginCtx.Param("id")})
	})
	tokenService := foundationPersonalAccessTokenService{caller: moduleapi.PersonalAccessTokenCaller{
		TokenID:   42,
		User:      moduleapi.CurrentUser{ID: 7, Username: "alice"},
		Scopes:    []string{"item.read"},
		ExpiresAt: time.Now().Add(time.Hour),
	}}
	if err := Register(HTTPRegistration{
		Engine:               engine,
		OpenAPISpec:          compilerTestBundle(map[string]any{"/api/items/{id}": map[string]any{"get": compilerTestOperation("getItem", compilerTestMetadata("low", false), false)}}),
		PersonalTokenService: tokenService,
		Authorizer:           authorizer,
		ConfirmationTokenTTL: time.Minute,
	}); err != nil {
		t.Fatalf("register streamable MCP read tool: %v", err)
	}

	initialize := httptest.NewRequest(http.MethodPost, HTTPPath, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	initialize.Header.Set("Authorization", "Bearer gpat_verified_by_httpx")
	initialize.Header.Set("Content-Type", "application/json")
	initialize.Header.Set("Accept", "application/json, text/event-stream")
	initializeRecorder := httptest.NewRecorder()
	engine.ServeHTTP(initializeRecorder, initialize)
	sessionID := initializeRecorder.Header().Get("Mcp-Session-Id")
	if initializeRecorder.Code != http.StatusOK || sessionID == "" {
		t.Fatalf("initialize = %d session=%q body=%s", initializeRecorder.Code, sessionID, initializeRecorder.Body.String())
	}

	call := httptest.NewRequest(http.MethodPost, HTTPPath, strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_item","arguments":{"id":"item-7"}}}`))
	call.Header.Set("Authorization", "Bearer gpat_verified_by_httpx")
	call.Header.Set("Content-Type", "application/json")
	call.Header.Set("Accept", "application/json, text/event-stream")
	call.Header.Set("Mcp-Session-Id", sessionID)
	callRecorder := httptest.NewRecorder()
	engine.ServeHTTP(callRecorder, call)
	if callRecorder.Code != http.StatusOK {
		t.Fatalf("tools/call status = %d, want %d: %s", callRecorder.Code, http.StatusOK, callRecorder.Body.String())
	}
	if authorizer.calls != 1 || auditEvents != 1 {
		t.Fatalf("REST authorization and audit behavior calls = %d, %d; want 1, 1", authorizer.calls, auditEvents)
	}
	if !strings.Contains(string(streamableResponsePayload(callRecorder.Body.String())), `"id":"item-7"`) {
		t.Fatalf("tools/call must return canonical REST response: %s", callRecorder.Body.String())
	}
}

func TestResourceAndConfirmedActionPreserveCanonicalRESTBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	auditEvents := 0
	engine.GET("/api/items/:id", func(ginCtx *gin.Context) {
		auditEvents++
		ginCtx.JSON(http.StatusOK, gin.H{"id": ginCtx.Param("id"), "result": "read"})
	})
	engine.POST("/api/items/:id/restart", func(ginCtx *gin.Context) {
		auditEvents++
		ginCtx.JSON(http.StatusOK, gin.H{"id": ginCtx.Param("id"), "result": "restarted"})
	})
	dispatcher, err := newDispatcher(engine)
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	read := toolDefinition{name: "get_item", method: http.MethodGet, path: "/api/items/{id}", inputs: []inputBinding{{name: "id", location: "path", required: true}}, metadata: mcpMetadata{resourceURIParameterBindings: map[string]string{"id": "id"}}}
	resource := resourceDefinition{uriTemplate: "graft://docker/containers/{id}", tool: read}
	ctx := withCaller(context.Background(), caller{tokenID: 42})
	resourceResult, err := dispatcher.resourceHandler(resource)(ctx, &mcpsdk.ReadResourceRequest{Params: &mcpsdk.ReadResourceParams{URI: "graft://docker/containers/item-7"}})
	if err != nil || len(resourceResult.Contents) != 1 || resourceResult.Contents[0].Text != `{"id":"item-7","result":"read"}` {
		t.Fatalf("resource projection = %#v, %v; want canonical REST JSON", resourceResult, err)
	}

	confirmations, err := newConfirmationTokens(time.Minute)
	if err != nil {
		t.Fatalf("new confirmations: %v", err)
	}
	action := toolDefinition{name: "post_item_restart", method: http.MethodPost, path: "/api/items/{id}/restart", inputs: []inputBinding{{name: "id", location: "path", required: true}}, metadata: mcpMetadata{confirmation: mcpConfirmation{required: true, strategy: "two_phase", ttl: "PT1M"}}}
	handler := dispatcher.actionHandler(action, confirmations)
	first, err := handler(ctx, &mcpsdk.CallToolRequest{Params: &mcpsdk.CallToolParamsRaw{Arguments: json.RawMessage(`{"id":"item-7"}`)}})
	if err != nil || !first.IsError || auditEvents != 1 {
		t.Fatalf("unconfirmed action = %#v, %v; it must not invoke REST", first, err)
	}
	var confirmation map[string]any
	if err := json.Unmarshal([]byte(toolResultText(first)), &confirmation); err != nil {
		t.Fatalf("decode confirmation: %v", err)
	}
	token, _ := confirmation[confirmationTokenInputName].(string)
	arguments, _ := json.Marshal(map[string]any{"id": "item-7", confirmationTokenInputName: token})
	confirmed, err := handler(ctx, &mcpsdk.CallToolRequest{Params: &mcpsdk.CallToolParamsRaw{Arguments: arguments}})
	if err != nil || confirmed.IsError || toolResultText(confirmed) != `{"id":"item-7","result":"restarted"}` || auditEvents != 2 {
		t.Fatalf("confirmed action = %#v, %v; must preserve REST response and audit", confirmed, err)
	}
}

func TestSDKTokenInfoBindsStreamableSessionToPersonalTokenID(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	request := httptest.NewRequest(http.MethodPost, HTTPPath, nil)
	request.Header.Set("Authorization", "Bearer gpat_verified_by_httpx")
	request = request.WithContext(withCaller(request.Context(), caller{
		tokenID:   42,
		scopes:    []string{"audit.read"},
		expiresAt: expiresAt.Unix(),
	}))
	recorder := httptest.NewRecorder()
	withSDKTokenInfo(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		info := mcpauth.TokenInfoFromContext(request.Context())
		if info == nil || info.UserID != "42" || len(info.Scopes) != 1 || info.Scopes[0] != "audit.read" || !info.Expiration.Equal(expiresAt) {
			t.Fatalf("unexpected SDK token info: %#v", info)
		}
	})).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

type foundationTestAuthorizer struct {
	calls int
	err   error
}

func (a *foundationTestAuthorizer) Authorize(context.Context, moduleapi.RequestAuthContext, string) error {
	a.calls++
	return a.err
}

var _ moduleapi.Authorizer = (*foundationTestAuthorizer)(nil)

type foundationPersonalAccessTokenService struct {
	caller moduleapi.PersonalAccessTokenCaller
}

func (s foundationPersonalAccessTokenService) CreateCurrentUserPersonalAccessToken(context.Context, moduleapi.PersonalAccessTokenCreateInput) (moduleapi.PersonalAccessTokenIssued, error) {
	return moduleapi.PersonalAccessTokenIssued{}, errors.New("not implemented")
}

func (s foundationPersonalAccessTokenService) ListCurrentUserPersonalAccessTokens(context.Context, int) ([]moduleapi.PersonalAccessTokenSummary, error) {
	return nil, errors.New("not implemented")
}

func (s foundationPersonalAccessTokenService) RevokeCurrentUserPersonalAccessToken(context.Context, uint64) error {
	return errors.New("not implemented")
}

func (s foundationPersonalAccessTokenService) AuthenticatePersonalAccessToken(_ context.Context, token string) (moduleapi.PersonalAccessTokenCaller, error) {
	if token != "gpat_verified_by_httpx" {
		return moduleapi.PersonalAccessTokenCaller{}, moduleapi.ErrInvalidPersonalAccessToken
	}
	return s.caller, nil
}

var _ moduleapi.PersonalAccessTokenService = foundationPersonalAccessTokenService{}

func streamableResponsePayload(body string) []byte {
	for _, line := range strings.Split(body, "\n") {
		if payload, ok := strings.CutPrefix(line, "data: "); ok {
			return []byte(payload)
		}
	}
	return []byte(body)
}
