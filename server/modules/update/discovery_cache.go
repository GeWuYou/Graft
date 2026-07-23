package update

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// DiscoverySnapshot 是 Update 模块持久化的最近发现结果，不承载未验证 manifest 正文。
type DiscoverySnapshot struct {
	Latest           *Release
	LastSuccessfulAt *time.Time
	LastAttemptAt    *time.Time
	CheckError       string
}

// DiscoveryCache 是 Update 自有的已验证 release catalog 快照边界。
type DiscoveryCache interface {
	Load(context.Context) (DiscoverySnapshot, error)
	Save(context.Context, DiscoverySnapshot) error
}

type sqlDiscoveryCache struct{ db *sql.DB }

func newSQLDiscoveryCache(db *sql.DB) (DiscoveryCache, error) {
	if db == nil {
		return nil, errors.New("update discovery cache database is unavailable")
	}
	return &sqlDiscoveryCache{db: db}, nil
}

func (s *sqlDiscoveryCache) Load(ctx context.Context) (DiscoverySnapshot, error) {
	if s == nil || s.db == nil {
		return DiscoverySnapshot{}, errors.New("update discovery cache is unavailable")
	}
	var latestJSON []byte
	var successful, attempted sql.NullTime
	var checkError sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT latest_release_json, last_successful_at, last_attempt_at, check_error
FROM update_discovery_cache WHERE cache_key = 'release_catalog'`).Scan(&latestJSON, &successful, &attempted, &checkError)
	if errors.Is(err, sql.ErrNoRows) {
		return DiscoverySnapshot{}, nil
	}
	if err != nil {
		return DiscoverySnapshot{}, fmt.Errorf("load update discovery cache: %w", err)
	}
	snapshot := DiscoverySnapshot{CheckError: checkError.String}
	if len(latestJSON) > 0 && string(latestJSON) != "null" {
		var latest Release
		if err := json.Unmarshal(latestJSON, &latest); err != nil {
			return DiscoverySnapshot{}, fmt.Errorf("decode cached update release: %w", err)
		}
		snapshot.Latest = &latest
	}
	if successful.Valid {
		value := successful.Time.UTC()
		snapshot.LastSuccessfulAt = &value
	}
	if attempted.Valid {
		value := attempted.Time.UTC()
		snapshot.LastAttemptAt = &value
	}
	return snapshot, nil
}

func (s *sqlDiscoveryCache) Save(ctx context.Context, snapshot DiscoverySnapshot) error {
	if s == nil || s.db == nil {
		return errors.New("update discovery cache is unavailable")
	}
	var latest any
	if snapshot.Latest != nil {
		encoded, err := json.Marshal(snapshot.Latest)
		if err != nil {
			return fmt.Errorf("encode cached update release: %w", err)
		}
		latest = encoded
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO update_discovery_cache
(cache_key, latest_release_json, last_successful_at, last_attempt_at, check_error, updated_at)
VALUES ('release_catalog', $1, $2, $3, $4, CURRENT_TIMESTAMP)
ON CONFLICT (cache_key) DO UPDATE SET latest_release_json = EXCLUDED.latest_release_json,
last_successful_at = EXCLUDED.last_successful_at, last_attempt_at = EXCLUDED.last_attempt_at,
check_error = EXCLUDED.check_error, updated_at = CURRENT_TIMESTAMP`, latest, nullableTime(snapshot.LastSuccessfulAt), nullableTime(snapshot.LastAttemptAt), nullableString(snapshot.CheckError))
	if err != nil {
		return fmt.Errorf("save update discovery cache: %w", err)
	}
	return nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}
