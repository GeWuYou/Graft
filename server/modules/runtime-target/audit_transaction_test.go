package runtimetarget

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/event"
	store "graft/server/modules/runtime-target/store"
)

type runtimeTargetRecordingPublisher struct {
	published  []event.Event
	publishErr error
	tx         *sql.Tx
}

func (p *runtimeTargetRecordingPublisher) PublishTx(_ context.Context, tx *sql.Tx, current event.Event, _ event.PublishOptions) (event.Receipt, error) {
	p.tx = tx
	p.published = append(p.published, current)
	return event.Receipt{EventID: current.ID, Delivery: event.DeliveryDurable}, p.publishErr
}

func TestRunRefreshAuditTransactionCommitsTargetAndDurableEvent(t *testing.T) {
	db, mock := newRuntimeTargetAuditMock(t)
	publisher := &runtimeTargetRecordingPublisher{}
	module := &Module{repository: store.NewSQLRepository(db), events: publisher}
	probe := store.LocalDockerProbe{Endpoint: localDockerEndpoint, Available: true, CheckedAt: time.Now().UTC()}

	mock.ExpectBegin()
	expectRuntimeTargetUpsert(mock, probe)
	mock.ExpectCommit()
	target, err := module.runRefreshAuditTransaction(context.Background(), func(ctx context.Context) (store.Target, error) {
		if err := module.repository.UpsertLocalDocker(ctx, probe); err != nil {
			return store.Target{}, err
		}
		return store.Target{ID: 9, Provider: "docker", DisplayName: "Local Docker"}, nil
	})
	if err != nil {
		t.Fatalf("run refresh audit transaction: %v", err)
	}
	if target.ID != 9 {
		t.Fatalf("target id = %d, want 9", target.ID)
	}
	if publisher.tx == nil || len(publisher.published) != 1 {
		t.Fatalf("expected one durable event in the transaction, got tx=%v events=%d", publisher.tx != nil, len(publisher.published))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestRunRefreshAuditTransactionRollsBackTargetWhenDurableEventFails(t *testing.T) {
	db, mock := newRuntimeTargetAuditMock(t)
	publishErr := errors.New("durable event unavailable")
	publisher := &runtimeTargetRecordingPublisher{publishErr: publishErr}
	module := &Module{repository: store.NewSQLRepository(db), events: publisher}
	probe := store.LocalDockerProbe{Endpoint: localDockerEndpoint, Available: false, Error: "Docker Unix socket is unavailable", CheckedAt: time.Now().UTC()}

	mock.ExpectBegin()
	expectRuntimeTargetUpsert(mock, probe)
	mock.ExpectRollback()
	_, err := module.runRefreshAuditTransaction(context.Background(), func(ctx context.Context) (store.Target, error) {
		if err := module.repository.UpsertLocalDocker(ctx, probe); err != nil {
			return store.Target{}, err
		}
		return store.Target{ID: 9, Provider: "docker", DisplayName: "Local Docker"}, nil
	})
	if !errors.Is(err, publishErr) {
		t.Fatalf("expected publish error %v, got %v", publishErr, err)
	}
	if publisher.tx == nil || len(publisher.published) != 1 {
		t.Fatalf("expected one failed durable event write inside transaction, got tx=%v events=%d", publisher.tx != nil, len(publisher.published))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func newRuntimeTargetAuditMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func expectRuntimeTargetUpsert(mock sqlmock.Sqlmock, probe store.LocalDockerProbe) {
	mock.ExpectExec(`INSERT INTO runtime_targets.*image_build`).
		WithArgs(probe.Endpoint, probe.Available, probe.Error, probe.CheckedAt).
		WillReturnResult(sqlmock.NewResult(9, 1))
}
