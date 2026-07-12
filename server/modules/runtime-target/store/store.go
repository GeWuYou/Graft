// Package store owns runtime-target persistence queries.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// ErrNotFound indicates that no live runtime target matches a lookup.
var ErrNotFound = errors.New("runtime target not found")

// LocalDockerProbe records the bounded Local Docker discovery result.
type LocalDockerProbe struct {
	Endpoint  string
	Available bool
	Error     string
	CheckedAt time.Time
}

// SQLRepository persists runtime target records.
type SQLRepository struct{ db *sql.DB }

// NewSQLRepository constructs the module-owned SQL repository.
func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db: db} }

// Target is the persisted runtime-target read projection.
type Target struct {
	ID             uint64     `json:"id"`
	Provider       string     `json:"provider"`
	DisplayName    string     `json:"displayName"`
	EndpointLabel  string     `json:"endpointLabel"`
	ConnectionKind string     `json:"connectionKind"`
	Capabilities   []string   `json:"capabilities"`
	Availability   bool       `json:"availability"`
	LastError      string     `json:"lastError"`
	CheckedAt      *time.Time `json:"checkedAt"`
}

// List returns all live runtime targets in stable display order.
func (r *SQLRepository) List(ctx context.Context) ([]Target, error) {
	if r == nil || r.db == nil {
		return []Target{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE deleted_at = 0 ORDER BY provider, display_name, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []Target{}
	for rows.Next() {
		var item Target
		var capabilities []byte
		if err := rows.Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Get returns one live runtime target by identifier.
func (r *SQLRepository) Get(ctx context.Context, id uint64) (Target, error) {
	if r == nil || r.db == nil {
		return Target{}, ErrNotFound
	}
	var item Target
	var capabilities []byte
	err := r.db.QueryRowContext(ctx, `SELECT id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at FROM runtime_targets WHERE id = $1 AND deleted_at = 0`, id).Scan(&item.ID, &item.Provider, &item.DisplayName, &item.EndpointLabel, &item.ConnectionKind, &capabilities, &item.Availability, &item.LastError, &item.CheckedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrNotFound
	}
	if err != nil {
		return Target{}, err
	}
	if err := json.Unmarshal(capabilities, &item.Capabilities); err != nil {
		return Target{}, err
	}
	return item, nil
}

// UpsertLocalDocker records the system-managed Local Docker discovery result.
func (r *SQLRepository) UpsertLocalDocker(ctx context.Context, probe LocalDockerProbe) error {
	if r == nil || r.db == nil {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO runtime_targets (provider, endpoint, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at, system_managed, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by) VALUES ('docker', $1, 'Local Docker', 'unix:///var/run/docker.sock', 'unix_socket', '["containers"]'::jsonb, $2, $3, $4, true, NOW(), 0, NOW(), 0, 0, 0) ON CONFLICT (provider, endpoint) WHERE deleted_at = 0 DO UPDATE SET availability = EXCLUDED.availability, last_error = EXCLUDED.last_error, checked_at = EXCLUDED.checked_at, updated_at = NOW(), updated_by = 0`, probe.Endpoint, probe.Available, probe.Error, probe.CheckedAt)
	return err
}
