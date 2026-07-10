package task

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/moduleapi"
)

func TestTaskListFilterParsesOptionalTypeAndStatus(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/tasks?type=project.compose.redeploy&status=failed", nil)
	filter, err := taskListFilter(context, moduleapi.TaskOwner{Type: "compose_project", ID: "42"})
	if err != nil {
		t.Fatalf("parse task list filter: %v", err)
	}
	if filter.Type == nil || *filter.Type != "project.compose.redeploy" || filter.Status == nil || *filter.Status != moduleapi.TaskStatusFailed {
		t.Fatalf("filter = %#v", filter)
	}
}

func TestTaskListFilterRejectsUnknownStatus(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/tasks?status=unsupported", nil)
	if _, err := taskListFilter(context, moduleapi.TaskOwner{Type: "compose_project", ID: "42"}); err == nil {
		t.Fatal("unknown task status accepted")
	}
}
