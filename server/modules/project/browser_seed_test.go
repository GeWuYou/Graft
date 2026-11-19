// 浏览器验收确定性 fixture：通过 Application 模块自身的导入写路径创建固定的
// fixture 应用，保证 schema 正确性由模块权威代码负责。仅在设置
// GRAFT_BROWSER_SEED_DATABASE_URL 时执行；本文件是测试专用代码。
package project

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// browserFixtureApplicationID 是 task 浏览器套件使用的应用公开标识。
// 固定为 app_ 前缀加 26 个 Crockford Base32 字符，保证跨模块 fixture 一致且幂等。
const browserFixtureApplicationID = "app_APPBR0WSERFXTR000000000000"

// TestSeedBrowserFixture 通过模块自有 ImportApplication 写路径写入确定性应用。
// 未提供种子数据库地址时跳过，保证常规 `go test` 与 CI 不依赖任何真实数据库。
func TestSeedBrowserFixture(t *testing.T) {
	databaseURL := os.Getenv("GRAFT_BROWSER_SEED_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("GRAFT_BROWSER_SEED_DATABASE_URL is required for the browser acceptance fixture seed")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open browser fixture database: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping browser fixture database: %v", err)
	}

	repository, err := projectstore.NewSQLRepository(db)
	if err != nil {
		t.Fatalf("create application store for browser fixture: %v", err)
	}
	_, err = repository.ImportApplication(context.Background(), projectstore.ImportApplicationInput{
		ApplicationID:            browserFixtureApplicationID,
		DeploymentAdapterKind:    projectcontract.DeploymentAdapterKindCompose.String(),
		WorkspacePath:            "/srv/graft-browser-fixture/workspace",
		ComposeProjectName:       "graft-browser-fixture",
		ComposeProjectNameSource: projectcontract.ComposeProjectNameSourceComputed.String(),
		DisplayName:              "Browser Fixture Application",
		SourceType:               projectcontract.SourceTypeImported.String(),
		OwnershipMode:            projectcontract.OwnershipModeExternal.String(),
		SourceMetadata:           map[string]string{"fixture": "browser-acceptance"},
		LifecycleStrategyKind:    projectcontract.LifecycleStrategyKindStandard.String(),
		LifecycleReviewStatus:    projectcontract.LifecycleReviewStatusConfirmed.String(),
		DriftStatus:              projectcontract.DriftStatusUnknown.String(),
		Files: []projectstore.ApplicationFile{{
			Kind:         projectcontract.FileKindCompose.String(),
			Role:         projectcontract.FileRolePrimary.String(),
			AbsolutePath: "/srv/graft-browser-fixture/workspace/compose.yml",
			DisplayPath:  "compose.yml",
		}},
	})
	if err != nil {
		t.Fatalf("import browser fixture application: %v", err)
	}
}

// TestBrowserFixtureApplicationIDFormat 校验 fixture 应用标识满足公开标识契约，
// 使 task fixture 与 HTTP 路由在无需数据库时即可静态验证一致性。
func TestBrowserFixtureApplicationIDFormat(t *testing.T) {
	if !isApplicationID(browserFixtureApplicationID) {
		t.Fatalf("browser fixture application id %q is not a valid application id", browserFixtureApplicationID)
	}
}
