package update

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

//go:embed migrations/202607230001_update_discovery_cache.sql
var discoveryCacheBaseSQL string

//go:embed migrations/202611160001_update_discovery_catalog.sql
var discoveryCacheCatalogSQL string

func discoveryCacheDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	applyDiscoveryMigration(t, db, discoveryCacheBaseSQL)
	return db
}

func applyDiscoveryMigration(t *testing.T, db *sql.DB, source string) {
	t.Helper()
	// 复用真实迁移列定义，仅适配 SQLite 的时间类型、默认值与注释语法。
	for _, statement := range strings.Split(source, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" || strings.HasPrefix(statement, "COMMENT ON") {
			continue
		}
		statement = strings.ReplaceAll(statement, "TIMESTAMPTZ", "TIMESTAMP")
		statement = strings.ReplaceAll(statement, "NOW()", "CURRENT_TIMESTAMP")
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDiscoveryCatalogMigrationPreservesExistingLatestOnlyCache(t *testing.T) {
	db := discoveryCacheDatabase(t)
	latest := Release{Version: "1.1.0-beta.2", Channel: "beta"}
	payload, err := json.Marshal(latest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO update_discovery_cache(cache_key,latest_release_json) VALUES('release_catalog',?)", payload); err != nil {
		t.Fatal(err)
	}
	cache, err := newSQLDiscoveryCache(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Load(t.Context()); err == nil || !strings.Contains(err.Error(), "catalog_json") {
		t.Fatalf("historical schema did not reproduce the missing catalog column: %v", err)
	}
	applyDiscoveryMigration(t, db, discoveryCacheCatalogSQL)
	loaded, err := cache.Load(t.Context())
	if err != nil || loaded.Latest == nil || loaded.Latest.Version != latest.Version || loaded.Catalog != nil {
		t.Fatalf("legacy cache changed during additive upgrade: %#v err=%v", loaded, err)
	}
}

func TestDiscoveryCatalogRepositoryRoundTripsTheMigratedColumn(t *testing.T) {
	db := discoveryCacheDatabase(t)
	applyDiscoveryMigration(t, db, discoveryCacheCatalogSQL)
	cache, err := newSQLDiscoveryCache(db)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	catalog := []Release{{Version: "1.1.0-beta.2", Channel: "beta"}, {Version: "1.1.0", Channel: "stable"}}
	snapshot := DiscoverySnapshot{Latest: &catalog[0], Catalog: catalog, LastSuccessfulAt: &now, LastAttemptAt: &now}
	if err := cache.Save(t.Context(), snapshot); err != nil {
		t.Fatal(err)
	}
	loaded, err := cache.Load(t.Context())
	if err != nil || len(loaded.Catalog) != 2 || loaded.Catalog[1].Version != catalog[1].Version || loaded.Latest.Version != catalog[0].Version {
		t.Fatalf("catalog round trip=%#v err=%v", loaded, err)
	}
	if loaded.LastSuccessfulAt == nil || !loaded.LastSuccessfulAt.Equal(now) || loaded.LastAttemptAt == nil || !loaded.LastAttemptAt.Equal(now) {
		t.Fatalf("catalog timestamps changed: %#v", loaded)
	}
}
