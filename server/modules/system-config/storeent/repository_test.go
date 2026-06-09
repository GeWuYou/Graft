// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	systemconfigstore "graft/server/modules/system-config/store"
)

func TestRepositorySetOverrideWrapsUserIDConversionError(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	}()

	repo := &repository{db: db}
	overflow := uint64(math.MaxInt64) + 1
	_, err = repo.SetOverride(context.Background(), "scheduler.timeout", json.RawMessage(`"60s"`), &overflow)
	if err == nil {
		t.Fatalf("expected user id range error")
	}
	if !strings.Contains(err.Error(), "set system config override:") {
		t.Fatalf("expected set override operation context, got %v", err)
	}
	if !strings.Contains(err.Error(), "system config override user id exceeds database range") {
		t.Fatalf("expected user id conversion error, got %v", err)
	}
	if _, ok := err.(interface{ Unwrap() error }); !ok {
		t.Fatalf("expected wrapped error, got %T", err)
	}
}

var _ systemconfigstore.Repository = (*repository)(nil)
