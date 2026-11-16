package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/moduleapi"
)

func TestUpdateAdmissionReturnsActionableUnavailableAndBusyErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, scenario := range []struct {
		failure error
		status  int
		code    string
	}{
		{moduleapi.ErrTaskExecutionUnavailable, http.StatusServiceUnavailable, messagecontract.CommonTaskExecutionUnavailable.String()},
		{moduleapi.ErrTaskOwnerBusy, http.StatusConflict, messagecontract.CommonTaskResourceBusy.String()},
	} {
		t.Run(scenario.code, func(t *testing.T) {
			response := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(response)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/platform/updates/operations", nil)
			updateRouteHandlers{}.writeStartFailure(ctx, 1, "0.0.0-beta.2", "", scenario.failure)
			var body struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != scenario.status || body.Code != scenario.code {
				t.Fatalf("admission response = %d / %s, want %d / %s", response.Code, body.Code, scenario.status, scenario.code)
			}
		})
	}
}

func TestPlatformUpdateClaimsTheTargetExecutionDomain(t *testing.T) {
	expectation := updateControllerExecutionExpectation(7, "payload")
	if len(expectation.ResourceKeys) != 1 || expectation.ResourceKeys[0] != "platform-update" {
		t.Fatalf("platform replacement lacks exclusive target coordination: %#v", expectation)
	}
}
