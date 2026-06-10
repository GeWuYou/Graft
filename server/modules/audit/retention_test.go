// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/cronx"
)

func TestNewAuditLogRetentionPolicyRejectsNonPositiveRetention(t *testing.T) {
	_, err := newAuditLogRetentionPolicy(config.AuditConfig{})
	if err == nil {
		t.Fatal("expected invalid retention policy error")
	}
}

func TestAuditLogRetentionPolicyCutoff(t *testing.T) {
	policy, err := newAuditLogRetentionPolicy(config.AuditConfig{LogRetention: 30 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}

	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	cutoff, err := auditLogRetentionCutoff(now, policy.retention)
	if err != nil {
		t.Fatalf("cutoff: %v", err)
	}

	want := now.Add(-30 * 24 * time.Hour)
	if !cutoff.Equal(want) {
		t.Fatalf("expected cutoff %s, got %s", want, cutoff)
	}
}

func TestAuditLogRetentionCleanerInvokesServiceWithCutoff(t *testing.T) {
	repo := &stubAuditRepository{deletedRows: 5}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	cleaner, err := newAuditLogRetentionCleaner(
		zap.NewNop(),
		service,
		config.AuditConfig{LogRetention: 7 * 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	}

	result, err := cleaner.cleanup(context.Background(), retentionJobConfig{RetentionDays: 14, BatchSize: 1000})
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	wantCutoff := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	if result.Metrics["deletedCount"] != int64(5) {
		t.Fatalf("expected deleted row count 5, got %#v", result)
	}
	if result.Details["retentionDays"] != 14 {
		t.Fatalf("expected configured retention days in result, got %#v", result)
	}
	if _, ok := result.Details["dryRun"]; ok {
		t.Fatalf("did not expect dryRun in persistent cleanup result details: %#v", result.Details)
	}
	if !repo.deletedBefore.Equal(wantCutoff) {
		t.Fatalf("expected cutoff %s, got %s", wantCutoff, repo.deletedBefore)
	}
}

func TestAuditLogRetentionCleanerLogsFailure(t *testing.T) {
	repo := &stubAuditRepository{deleteErr: errors.New("boom")}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	core, logs := observer.New(zap.InfoLevel)
	cleaner, err := newAuditLogRetentionCleaner(
		zap.New(core),
		service,
		config.AuditConfig{LogRetention: 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("new cleaner: %v", err)
	}
	cleaner.now = func() time.Time {
		return time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	}

	if result, err := cleaner.cleanup(context.Background(), retentionJobConfig{BatchSize: 1000}); err == nil {
		t.Fatal("expected cleanup failure")
	} else if result.Stage != "failed" || len(result.Warnings) == 0 {
		t.Fatalf("expected failed structured result, got %#v", result)
	}

	if logs.FilterMessage("audit log retention cleanup started").Len() != 1 {
		t.Fatalf("expected start log, got %#v", logs.All())
	}
	if logs.FilterMessage("audit log retention cleanup failed").Len() != 1 {
		t.Fatalf("expected failure log, got %#v", logs.All())
	}
}

func TestRegisterAuditLogRetentionCleanupJob(t *testing.T) {
	registry := cronx.NewRegistry()
	repo := &stubAuditRepository{deletedRows: 2}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if err := registerAuditLogRetentionCleanupJob(
		registry,
		zap.NewNop(),
		service,
		config.AuditConfig{LogRetention: 30 * 24 * time.Hour},
	); err != nil {
		t.Fatalf("register retention job: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one registered retention job, got %d", len(items))
	}
	assertAuditLogRetentionJobMetadata(t, items[0])
	if err := items[0].Validate(); err != nil {
		t.Fatalf("validate registered job: %v", err)
	}
	if !repo.deletedBefore.IsZero() {
		t.Fatal("expected startup registration to avoid cleanup execution")
	}

	result, err := items[0].Handler(context.Background(), items[0].RuntimeDefaultConfig())
	if err != nil {
		t.Fatalf("run registered job: %v", err)
	}
	if result.AffectedResource != "audit_log" || result.Metrics["deletedCount"] != int64(2) {
		t.Fatalf("expected structured audit cleanup result, got %#v", result)
	}
	if repo.deletedBefore.IsZero() {
		t.Fatal("expected job run to invoke cleanup")
	}
	assertAuditLogRetentionDryRunAction(t, repo, items[0])
}

func TestRegisterAuditLogRetentionConfigDefinition(t *testing.T) {
	registry := configregistry.NewRegistry()

	if err := registerAuditLogRetentionConfigDefinition(registry); err != nil {
		t.Fatalf("register config definition: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one config definition, got %d", len(items))
	}
	assertAuditLogRetentionConfigDefinition(t, items[0])
}

func assertAuditLogRetentionConfigDefinition(t *testing.T, definition configregistry.Definition) {
	t.Helper()

	if definition.Key != auditLogRetentionCleanupJobName ||
		definition.Module != moduleID ||
		definition.Type != configregistry.ValueTypeObject {
		t.Fatalf("unexpected audit log config definition: %#v", definition)
	}
	if definition.GroupKey != auditLogRetentionConfigGroupKey ||
		definition.DomainKey != auditLogRetentionConfigDomainKey ||
		definition.GroupDescriptionKey != auditLogRetentionConfigGroupDescKey ||
		definition.TitleKey != auditLogRetentionConfigTitleKey ||
		definition.DescriptionKey != auditLogRetentionConfigDescriptionKey {
		t.Fatalf("expected localized audit log config metadata, got %#v", definition)
	}
	if definition.GroupLabel == "audit / log.retention" {
		t.Fatalf("group label must be product-facing fallback, got %q", definition.GroupLabel)
	}
	if string(definition.DefaultValue) != auditLogRetentionCleanupDefaultConfig {
		t.Fatalf("expected default config %s, got %s", auditLogRetentionCleanupDefaultConfig, definition.DefaultValue)
	}
	if !strings.Contains(string(definition.Schema), `"x-i18n"`) ||
		!strings.Contains(string(definition.Schema), `"unitKey":"systemConfig.units.days"`) ||
		!strings.Contains(string(definition.Schema), `"batchSize":{"type":"integer","minimum":1,"maximum":10000`) {
		t.Fatalf("expected x-i18n schema metadata, got %s", string(definition.Schema))
	}
}

func assertAuditLogRetentionJobMetadata(t *testing.T, job cronx.Job) {
	t.Helper()

	if job.Name != auditLogRetentionCleanupJobName {
		t.Fatalf("expected job name %q, got %q", auditLogRetentionCleanupJobName, job.Name)
	}
	if job.Module != moduleID {
		t.Fatalf("expected job module %q, got %q", moduleID, job.Module)
	}
	if job.Schedule != auditLogRetentionCleanupJobSchedule {
		t.Fatalf("expected job schedule %q, got %q", auditLogRetentionCleanupJobSchedule, job.Schedule)
	}
	if job.DefaultConfig != auditLogRetentionCleanupDefaultConfig {
		t.Fatalf("expected default config %s, got %s", auditLogRetentionCleanupDefaultConfig, job.DefaultConfig)
	}
	assertAuditLogRetentionJobConfig(t, job)
	assertAuditLogRetentionJobActions(t, job)
}

func assertAuditLogRetentionJobConfig(t *testing.T, job cronx.Job) {
	t.Helper()

	if strings.Contains(job.DefaultConfig, "dryRun") || strings.Contains(job.ConfigSchema, "dryRun") {
		t.Fatalf("did not expect dryRun in persistent audit log job config: default=%s schema=%s", job.DefaultConfig, job.ConfigSchema)
	}
	if !strings.Contains(job.DefaultConfig, "retentionDays") || !strings.Contains(job.ConfigSchema, "retentionDays") {
		t.Fatalf("expected retentionDays in audit log job config: default=%s schema=%s", job.DefaultConfig, job.ConfigSchema)
	}
}

func assertAuditLogRetentionJobActions(t *testing.T, job cronx.Job) {
	t.Helper()

	if len(job.Actions) != 1 || job.Actions[0].Key != auditLogRetentionDryRunActionKey {
		t.Fatalf("expected dryRun action, got %#v", job.Actions)
	}
}

func assertAuditLogRetentionDryRunAction(t *testing.T, repo *stubAuditRepository, job cronx.Job) {
	t.Helper()

	repo.deletedBefore = time.Time{}
	actionResult, err := job.Actions[0].Handler(context.Background(), `{"retentionDays":7,"batchSize":500}`)
	if err != nil {
		t.Fatalf("run dryRun action: %v", err)
	}
	if actionResult.Metrics["deletedCount"] != int64(0) {
		t.Fatalf("expected dryRun action to avoid deletion, got %#v", actionResult)
	}
	if !repo.deletedBefore.IsZero() {
		t.Fatalf("expected dryRun action to skip deletion, got cutoff %s", repo.deletedBefore)
	}
}
