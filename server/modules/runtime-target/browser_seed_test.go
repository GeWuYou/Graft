// 浏览器验收确定性 fixture：仅在设置 GRAFT_BROWSER_SEED_DATABASE_URL 时，
// 由 scripts/browser_acceptance.py 对隔离的 PostgreSQL 环境执行幂等种子写入。
// 本文件是测试专用代码，不承载任何生产业务逻辑。
package runtimetarget

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// browserFixtureTargets 是 runtime-target 浏览器套件消费的确定性事实。
// endpoint 必须指向不存在的本地套接字，避免与宿主机真实 Docker 探测事实冲突。
var browserFixtureTargets = []struct {
	DisplayName   string
	Endpoint      string
	EndpointLabel string
	Available     bool
}{
	{DisplayName: "Browser Fixture Target A", Endpoint: "unix:///run/graft-browser-fixture/docker-a.sock", EndpointLabel: "/run/graft-browser-fixture/docker-a.sock", Available: true},
	{DisplayName: "Browser Fixture Target B", Endpoint: "unix:///run/graft-browser-fixture/docker-b.sock", EndpointLabel: "/run/graft-browser-fixture/docker-b.sock", Available: false},
}

// TestSeedBrowserFixture 向隔离环境写入确定性 runtime targets；未提供种子数据库
// 地址时跳过，保证常规 `go test` 与 CI 不依赖任何真实数据库。
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

	checkedAt := time.Now().UTC()
	for _, target := range browserFixtureTargets {
		_, err := db.ExecContext(context.Background(), `
			INSERT INTO runtime_targets
				(provider, endpoint, display_name, endpoint_label, connection_kind,
				 capabilities_json, availability, last_error, checked_at,
				 system_managed, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
			VALUES ('docker', $1, $2, $3, 'unix_socket', '[]'::jsonb, $4, '', $5,
					FALSE, NOW(), 0, NOW(), 0, 0, 0)
			ON CONFLICT (provider, endpoint) WHERE deleted_at = 0 DO UPDATE SET
				display_name = EXCLUDED.display_name,
				endpoint_label = EXCLUDED.endpoint_label,
				availability = EXCLUDED.availability,
				checked_at = EXCLUDED.checked_at,
				updated_at = NOW(),
				deleted_at = 0,
				deleted_by = 0`,
			target.Endpoint, target.DisplayName, target.EndpointLabel, target.Available, checkedAt)
		if err != nil {
			t.Fatalf("seed runtime target %s: %v", target.DisplayName, err)
		}
	}
}

// TestBrowserFixtureTargetsAreDeterministic 校验 fixture 数据与生产探测事实隔离，
// 避免种子写入与本地 Docker 自动发现互相覆盖。
func TestBrowserFixtureTargetsAreDeterministic(t *testing.T) {
	seen := make(map[string]bool, len(browserFixtureTargets))
	for _, target := range browserFixtureTargets {
		if target.Endpoint == "unix:///var/run/docker.sock" {
			t.Fatalf("fixture endpoint %s must not shadow the local Docker probe target", target.Endpoint)
		}
		if seen[target.Endpoint] {
			t.Fatalf("duplicate fixture endpoint: %s", target.Endpoint)
		}
		seen[target.Endpoint] = true
		if target.DisplayName == "" {
			t.Fatalf("fixture target %s requires a display name", target.Endpoint)
		}
	}
	if len(browserFixtureTargets) != 2 {
		t.Fatalf("fixture target count = %d, want exactly two deterministic targets", len(browserFixtureTargets))
	}
	if _, exists := seen[browserFixtureTargets[0].Endpoint]; !exists {
		t.Fatal("deterministic fixture endpoints must remain stable")
	}
}
