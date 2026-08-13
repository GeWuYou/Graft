package user

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReadUserListQueryAppliesDefaultsAndLimitCap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest("GET", "/api/users?keyword=admin&status=enabled&role_id=7&limit=999&offset=4", nil)

	query, ok := readUserListQuery(ginCtx)
	if !ok {
		t.Fatal("readUserListQuery() returned invalid")
	}
	if query.Keyword != "admin" || query.Status != "enabled" || query.Limit != maximumUserListLimit || query.Offset != 4 || query.RoleID == nil || *query.RoleID != 7 {
		t.Fatalf("readUserListQuery() = %#v", query)
	}
}

func TestReadUserListQueryRejectsInvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, rawQuery := range []string{"limit=0", "offset=-1", "status=unknown", "role_id=0"} {
		t.Run(rawQuery, func(t *testing.T) {
			ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ginCtx.Request = httptest.NewRequest("GET", "/api/users?"+rawQuery, nil)
			if _, ok := readUserListQuery(ginCtx); ok {
				t.Fatalf("readUserListQuery(%q) accepted invalid query", rawQuery)
			}
		})
	}
}
