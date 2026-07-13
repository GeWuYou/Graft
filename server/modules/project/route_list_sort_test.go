package project

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/module"
)

func TestBindListParamsBindsRestrictedProjectSort(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/ops/projects?sort=created_at:asc", nil)

	params, ok := bindListParams(ginCtx, &module.Context{})
	if !ok {
		t.Fatalf("expected valid project sort to bind, response=%d", recorder.Code)
	}
	if params.Sort == nil || len(*params.Sort) != 1 || (*params.Sort)[0] != "created_at:asc" {
		t.Fatalf("unexpected bound sort %#v", params.Sort)
	}
	if got := projectListSortParamValue(params.Sort); got != string(generated.GetProjectsParamsSortCreatedAtAsc) {
		t.Fatalf("expected sort param value %q, got %q", generated.GetProjectsParamsSortCreatedAtAsc, got)
	}
}

func TestBindListParamsRejectsInvalidOrDuplicateProjectSort(t *testing.T) {
	t.Parallel()

	for _, query := range []string{
		"sort=updated_at:desc",
		"sort=created_at:asc&sort=created_at:desc",
		"sort[]=created_at:asc&sort=created_at:desc",
	} {
		t.Run(query, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/ops/projects?"+query, nil)

			_, ok := bindListParams(ginCtx, &module.Context{})
			if ok {
				t.Fatal("expected invalid project sort query to be rejected")
			}
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 response, got %d", recorder.Code)
			}
		})
	}
}
