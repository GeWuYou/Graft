package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSharedBuildProviderPreservesOriginalMaterialLeaseFence(t *testing.T) {
	lease := executionLease{ID: "original-lease", FenceToken: "original-fence", Capability: buildExecutionCapability, Protocol: buildExecutionProtocol, OperationID: buildImageLocalOperation, Input: json.RawMessage(`{"build_id":"build-transport"}`)}
	observed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = true
		if r.Method != http.MethodPost || r.URL.Path != "/agent/v1/execution-leases/original-lease/material" {
			t.Errorf("unexpected material request %s %s", r.Method, r.URL.Path)
		}
		var request executionHandleRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.FenceToken != lease.FenceToken {
			t.Errorf("material fence=%q", request.FenceToken)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"protocol":"build-execution-material/v1","payload":{}}`))
	}))
	defer server.Close()
	result := executeProviderOperation(context.Background(), server.Client(), config{AgentURL: server.URL, DockerSocket: "unix:///missing"}, lease)
	if !observed || result.Outcome != "failed" || result.FailureCode != failureInvalidIntent {
		t.Fatalf("observed=%v result=%#v", observed, result)
	}
}
