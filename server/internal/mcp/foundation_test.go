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

func TestRegisterServesStreamableFoundationWithoutBusinessCapabilities(t *testing.T) {
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
	for _, capability := range []string{"tools", "resources", "prompts"} {
		if _, ok := response.Result.Capabilities[capability]; ok {
			t.Fatalf("foundation must not expose %s capability: %s", capability, recorder.Body.String())
		}
	}

	sessionID := recorder.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("initialize response must create a streamable session")
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
