package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresExecutionOwnerRejectsAnotherServerBeforeRecovery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db, SQLDialectPostgres)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT pg_try_advisory_lock").WithArgs(executionOwnershipKey).WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(false))
	if _, err := repository.AcquireExecutionOwnership(context.Background()); err == nil {
		t.Fatal("second active execution owner accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExecutionOwnershipAdmissionStateMachine(t *testing.T) {
	// 一次瞬时探测失败不得锁定准入；N 次连续失败进入退化并拒绝新请求；
	// 探测恢复即自动回到就绪；只有真实的所有权冲突才永久失去。
	lost := make(chan struct{})
	ownership := &ExecutionOwnership{Lost: lost, lost: lost}
	if ready, diagnostic := ownership.Admission(); !ready || diagnostic != "execution_ready" {
		t.Fatalf("fresh ownership admission = %v/%s", ready, diagnostic)
	}
	for failures := 1; failures < ExecutionOwnershipDegradedAfter; failures++ {
		if degraded := ownership.ProbeFailed(); degraded {
			t.Fatalf("transient ping failure %d degraded admission", failures)
		}
		if ready, diagnostic := ownership.Admission(); !ready || diagnostic != "execution_ready" {
			t.Fatalf("admission degraded after %d failures = %v/%s", failures, ready, diagnostic)
		}
	}
	if !ownership.ProbeFailed() {
		t.Fatal("expected degraded admission after the failure threshold")
	}
	if ready, diagnostic := ownership.Admission(); ready || diagnostic != ExecutionOwnershipAdmissionDegraded {
		t.Fatalf("degraded admission = %v/%s", ready, diagnostic)
	}
	// 探测恢复：退化自动清除，无需重启。
	ownership.ProbeSucceeded()
	if ready, diagnostic := ownership.Admission(); !ready || diagnostic != "execution_ready" {
		t.Fatalf("admission did not self-heal = %v/%s", ready, diagnostic)
	}
}

func TestExecutionOwnershipLatchesOnlyOnGenuineConflict(t *testing.T) {
	lost := make(chan struct{})
	ownership := &ExecutionOwnership{Lost: lost, lost: lost}
	for i := 0; i < ExecutionOwnershipDegradedAfter; i++ {
		ownership.ProbeFailed()
	}
	ownership.OwnershipConflict()
	if ready, diagnostic := ownership.Admission(); ready || diagnostic != "execution_ownership_lost" {
		t.Fatalf("conflicted admission = %v/%s", ready, diagnostic)
	}
	select {
	case <-ownership.Lost:
	default:
		t.Fatal("genuine ownership conflict must close the lost channel")
	}
	// lost 之后的探测不得复活所有权。
	ownership.ProbeSucceeded()
	if ready, diagnostic := ownership.Admission(); ready || diagnostic != "execution_ownership_lost" {
		t.Fatalf("lost ownership resurrected = %v/%s", ready, diagnostic)
	}
}

func TestExecutionOwnershipReacquireBackoffIsBounded(t *testing.T) {
	lost := make(chan struct{})
	ownership := &ExecutionOwnership{Lost: lost, lost: lost}
	for i := 0; i < ExecutionOwnershipDegradedAfter; i++ {
		ownership.ProbeFailed()
	}
	if !ownership.ShouldReacquire() {
		t.Fatal("first reacquire attempt must happen immediately at degraded")
	}
	ownership.failures = ExecutionOwnershipDegradedAfter + executionOwnershipReacquireGap - 1
	if ownership.ShouldReacquire() {
		t.Fatal("reacquire attempt fired inside the bounded gap")
	}
	ownership.failures = ExecutionOwnershipDegradedAfter + executionOwnershipReacquireGap
	if !ownership.ShouldReacquire() {
		t.Fatal("reacquire attempt must fire at the gap boundary")
	}
}

func TestExecutionOwnershipCloseWithoutConnection(t *testing.T) {
	lost := make(chan struct{})
	ownership := &ExecutionOwnership{Lost: lost, lost: lost}
	if err := ownership.Close(); err != nil {
		t.Fatalf("close without connection: %v", err)
	}
}

func TestPostgresExecutionOwnerDegradesOnTransientPingFailure(t *testing.T) {
	// 单次探测失败进入退化但不永久失去所有权；退化可自动恢复（既有的 1 次失败即失去语义已废止）。
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db, SQLDialectPostgres)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT pg_try_advisory_lock").WithArgs(executionOwnershipKey).WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(true))
	// 后续探测不再设置期望：sqlmock 使其失败，监视循环按连续失败计数进入退化。
	mock.ExpectPing().WillReturnError(errors.New("ownership connection lost"))
	ownership, err := repository.AcquireExecutionOwnership(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		ready, diagnostic := ownership.Admission()
		if !ready {
			if diagnostic != ExecutionOwnershipAdmissionDegraded {
				t.Fatalf("degraded diagnostic = %s, want %s", diagnostic, ExecutionOwnershipAdmissionDegraded)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ownership did not degrade after repeated ping failures")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// 瞬时退化不得关闭 Lost；永久失去仅由真实的所有权冲突触发。
	select {
	case <-ownership.Lost:
		t.Fatal("transient ping failure must not report ownership loss")
	default:
	}
	// 监视循环仍在运行（未放弃自我恢复），关闭后随运行时一起释放。
	if err := ownership.Close(); err != nil {
		t.Fatalf("close degraded ownership: %v", err)
	}
}
