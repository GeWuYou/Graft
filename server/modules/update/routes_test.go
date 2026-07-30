package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"graft/server/internal/httpx"
	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
	updatecontract "graft/server/modules/update/contract"
)

type failingOperationStore struct{ err error }

func (s failingOperationStore) Create(context.Context, ComposeUpdateOperation) error { return s.err }
func (s failingOperationStore) Get(context.Context, string) (ComposeUpdateOperation, error) {
	return ComposeUpdateOperation{}, s.err
}
func (s failingOperationStore) List(context.Context, int) ([]ComposeUpdateOperation, error) {
	return nil, s.err
}
func (s failingOperationStore) Settle(context.Context, ComposeUpdateOperation) error { return s.err }

type updateAuthorizerStub struct{ err error }

func (s updateAuthorizerStub) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permission string) error {
	if permission != updatecontract.UpdateManagePermission.String() {
		panic("unexpected update authorizer permission: " + permission)
	}

	return s.err
}

func TestMayViewComposeCandidatesRequiresUpdateManagePermission(t *testing.T) {
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})

	if !mayViewComposeCandidates(ctx, updateAuthorizerStub{}) {
		t.Fatal("expected manage-authorized request to view compose candidates")
	}
	if mayViewComposeCandidates(ctx, updateAuthorizerStub{err: errors.New("denied")}) {
		t.Fatal("expected denied request to hide compose candidates")
	}
	if mayViewComposeCandidates(context.Background(), updateAuthorizerStub{}) {
		t.Fatal("expected request without auth context to hide compose candidates")
	}
}

func TestStatusForUpdateViewerRedactsEveryNonManageResponse(t *testing.T) {
	status := Status{Profile: InstallationProfile{ComposeCandidates: []ComposeRootCandidate{{CandidateKey: "compose-a", Root: "/srv/graft"}}}}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})

	if got := statusForUpdateViewer(ctx, updateAuthorizerStub{}, status); len(got.Profile.ComposeCandidates) != 1 {
		t.Fatalf("expected manage response to preserve candidates, got %#v", got.Profile.ComposeCandidates)
	}
	if got := statusForUpdateViewer(ctx, updateAuthorizerStub{err: errors.New("denied")}, status); len(got.Profile.ComposeCandidates) != 0 {
		t.Fatalf("expected non-manage response to redact candidates, got %#v", got.Profile.ComposeCandidates)
	}
}

func TestListOperationsLogsRequestCorrelatedStoreFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zap.ErrorLevel)
	response := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(response)
	request := httptest.NewRequest(http.MethodGet, "/api/platform/updates/operations", nil)
	request = request.WithContext(httpx.WithRequestAuditContext(request.Context(), httpx.RequestAuditContext{RequestID: "request-history-91"}))
	ginCtx.Request = request
	handler := updateRouteHandlers{
		rollout:   &RolloutService{operations: failingOperationStore{err: errors.New("relation update_failure_diagnostics does not exist")}},
		appLogger: logger.NewAppLogger(zap.New(core)),
	}

	handler.list(ginCtx)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("list response code = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one failure log, got %#v", entries)
	}
	fields := entries[0].ContextMap()
	if fields[logger.FieldOperation] != "platform_update.operations.list" || fields[logger.FieldRequestID] != "request-history-91" {
		t.Fatalf("missing operation or request correlation: %#v", fields)
	}
}
