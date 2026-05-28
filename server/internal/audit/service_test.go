package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	auditstore "graft/server/plugins/audit/store"
)

type stubAuditRepository struct {
	createdInput auditstore.CreateAuditLogInput
	listQuery    auditstore.ListAuditLogsQuery
	overviewWnd  auditstore.OverviewWindow
	policyRules  []auditstore.AuditPolicyRule
	createResult auditstore.AuditLog
	listResult   auditstore.ListAuditLogsResult
	overview     auditstore.AuditOverview
	createErr    error
	listErr      error
	overviewErr  error
	policyErr    error
}

func (r *stubAuditRepository) CreateAuditLog(_ context.Context, input auditstore.CreateAuditLogInput) (auditstore.AuditLog, error) {
	r.createdInput = input
	if r.createErr != nil {
		return auditstore.AuditLog{}, r.createErr
	}
	if r.createResult.ID == 0 {
		r.createResult = auditstore.AuditLog{ID: 1}
	}
	return r.createResult, nil
}

func (r *stubAuditRepository) ListAuditLogs(_ context.Context, query auditstore.ListAuditLogsQuery) (auditstore.ListAuditLogsResult, error) {
	r.listQuery = query
	if r.listErr != nil {
		return auditstore.ListAuditLogsResult{}, r.listErr
	}
	return r.listResult, nil
}

func (r *stubAuditRepository) ReadAuditOverview(_ context.Context, window auditstore.OverviewWindow) (auditstore.AuditOverview, error) {
	r.overviewWnd = window
	if r.overviewErr != nil {
		return auditstore.AuditOverview{}, r.overviewErr
	}
	return r.overview, nil
}

func (r *stubAuditRepository) ListAuditPolicyRules(_ context.Context) ([]auditstore.AuditPolicyRule, error) {
	if r.policyErr != nil {
		return nil, r.policyErr
	}
	return append([]auditstore.AuditPolicyRule(nil), r.policyRules...), nil
}

func TestServiceRecordSanitizesSensitiveFields(t *testing.T) {
	repo := &stubAuditRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	actorID := uint64(7)
	_, err = service.Record(context.Background(), RecordInput{
		ActorUserID:      &actorID,
		ActorUsername:    " alice ",
		ActorDisplayName: " Alice ",
		Action:           " user.password.reset ",
		ResourceType:     " user ",
		ResourceID:       " 7 ",
		ResourceName:     " Alice ",
		RequestID:        " req-1 ",
		IP:               " 127.0.0.1 ",
		UserAgent:        " curl/8 ",
		Message:          `password=plain authorization: Bearer token`,
		Metadata: map[string]any{
			"password":      "plain-text",
			"nested":        map[string]any{"refresh_token": "secret-token"},
			"request_body":  map[string]any{"username": "alice"},
			"free_text_log": `cookie: session=abc`,
		},
	})
	if err != nil {
		t.Fatalf("record audit log: %v", err)
	}

	if repo.createdInput.ActorUsername != "alice" || repo.createdInput.ActorDisplayName != "Alice" {
		t.Fatalf("expected trimmed actor identity, got %#v", repo.createdInput)
	}
	if repo.createdInput.Message != "password=[REDACTED] authorization: [REDACTED]" {
		t.Fatalf("expected sensitive message redaction, got %q", repo.createdInput.Message)
	}

	var metadata map[string]any
	if err := json.Unmarshal(repo.createdInput.Metadata, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata["password"] != redactedValue {
		t.Fatalf("expected password to be redacted, got %#v", metadata["password"])
	}
	nested, ok := metadata["nested"].(map[string]any)
	if !ok || nested["refresh_token"] != redactedValue {
		t.Fatalf("expected nested token to be redacted, got %#v", metadata["nested"])
	}
	if metadata["free_text_log"] != "cookie: [REDACTED]" {
		t.Fatalf("expected free-text cookie redaction, got %#v", metadata["free_text_log"])
	}
}

func TestServiceRecordRedactsAllCookiePairs(t *testing.T) {
	repo := &stubAuditRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.Record(context.Background(), RecordInput{
		Action:   "audit.test",
		Metadata: map[string]any{"free_text_log": "cookie: a=1; b=2"},
	})
	if err != nil {
		t.Fatalf("record audit log: %v", err)
	}

	var metadata map[string]any
	if err := json.Unmarshal(repo.createdInput.Metadata, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata["free_text_log"] != "cookie: [REDACTED]" {
		t.Fatalf("expected all cookie pairs to be redacted, got %#v", metadata["free_text_log"])
	}
}

func TestServiceRecordRejectsMissingAction(t *testing.T) {
	repo := &stubAuditRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.Record(context.Background(), RecordInput{})
	if err == nil || err.Error() != "audit action is required" {
		t.Fatalf("expected missing action error, got %v", err)
	}
}

func TestServiceListNormalizesPagination(t *testing.T) {
	repo := &stubAuditRepository{
		listResult: auditstore.ListAuditLogsResult{
			Items: []auditstore.AuditLog{{ID: 9}},
			Total: 42,
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	success := true
	start := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	result, err := service.List(context.Background(), ListQuery{
		Page:         0,
		PageSize:     999,
		Action:       " user.create ",
		ResourceType: " user ",
		Success:      &success,
		RequestID:    " req-1 ",
		CreatedFrom:  &start,
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}

	if repo.listQuery.Limit != maxPageSize || repo.listQuery.Offset != 0 {
		t.Fatalf("expected capped pagination, got %#v", repo.listQuery)
	}
	if repo.listQuery.Action != "user.create" || repo.listQuery.ResourceType != "user" || repo.listQuery.RequestID != "req-1" {
		t.Fatalf("expected trimmed filters, got %#v", repo.listQuery)
	}
	if result.Page != defaultPage || result.PageSize != maxPageSize || result.Total != 42 || len(result.Items) != 1 {
		t.Fatalf("unexpected list result %#v", result)
	}
}

func TestServiceListPropagatesRepositoryError(t *testing.T) {
	repo := &stubAuditRepository{listErr: errors.New("boom")}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.List(context.Background(), ListQuery{})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestServiceOverviewDelegatesWindowWithoutNormalization(t *testing.T) {
	repo := &stubAuditRepository{
		overview: auditstore.AuditOverview{Window: "custom"},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.Overview(context.Background(), "custom")
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if repo.overviewWnd != "custom" {
		t.Fatalf("expected raw window to be delegated, got %q", repo.overviewWnd)
	}
}

func TestServiceRecordCandidateAppliesMatchingPolicy(t *testing.T) {
	repo := &stubAuditRepository{
		policyRules: []auditstore.AuditPolicyRule{
			{
				ID:        7,
				Name:      "domain.user.create",
				Source:    auditstore.AuditSourceDomainEvent,
				Enabled:   true,
				Priority:  10,
				Effect:    auditstore.AuditPolicyEffectInclude,
				EventType: "user.create",
				MatchType: auditstore.AuditPolicyMatchTypeExact,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	actorID := uint64(9)
	record, recorded, err := service.RecordCandidate(context.Background(), auditstore.AuditCandidate{
		Source:           auditstore.AuditSourceDomainEvent,
		ActorUserID:      &actorID,
		ActorUsername:    "bob",
		ActorDisplayName: "Bob",
		Action:           "user.create",
		EventType:        "user.create",
		ResourceType:     "user",
		ResourceID:       "9",
		RequestID:        "req-9",
		Success:          true,
		Message:          "created",
	})
	if err != nil {
		t.Fatalf("record candidate: %v", err)
	}
	if !recorded {
		t.Fatal("expected candidate to be recorded")
	}
	if record.ID == 0 || repo.createdInput.Action != "user.create" {
		t.Fatalf("unexpected persisted record %#v %#v", record, repo.createdInput)
	}
}

func TestServiceRecordCandidateSkipsUnmatchedPolicy(t *testing.T) {
	repo := &stubAuditRepository{
		policyRules: []auditstore.AuditPolicyRule{
			{
				Name:        "request.login",
				Source:      auditstore.AuditSourceRequest,
				Enabled:     true,
				Priority:    10,
				Effect:      auditstore.AuditPolicyEffectInclude,
				Method:      "POST",
				PathPattern: "/api/auth/login",
				MatchType:   auditstore.AuditPolicyMatchTypeExact,
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, recorded, err := service.RecordCandidate(context.Background(), auditstore.AuditCandidate{
		Source:        auditstore.AuditSourceRequest,
		Action:        "GET /api/users/:id",
		RequestMethod: "GET",
		RequestPath:   "/api/users/:id",
		Success:       true,
	})
	if err != nil {
		t.Fatalf("record candidate: %v", err)
	}
	if recorded {
		t.Fatal("expected candidate to be skipped")
	}
	if repo.createdInput.Action != "" {
		t.Fatalf("expected no record write, got %#v", repo.createdInput)
	}
}

func TestServiceRecordCandidateSkipsExcludedPolicy(t *testing.T) {
	repo := &stubAuditRepository{
		policyRules: []auditstore.AuditPolicyRule{
			{
				Name:        "request.monitor.exclude",
				Source:      auditstore.AuditSourceRequest,
				Enabled:     true,
				Priority:    1,
				Effect:      auditstore.AuditPolicyEffectExclude,
				Method:      "GET",
				PathPattern: "/api/monitor",
				MatchType:   auditstore.AuditPolicyMatchTypePrefix,
			},
			{
				Name:        "request.monitor.include.fallback",
				Source:      auditstore.AuditSourceRequest,
				Enabled:     true,
				Priority:    20,
				Effect:      auditstore.AuditPolicyEffectInclude,
				Method:      "GET",
				PathPattern: "/api/monitor/server-status",
				MatchType:   auditstore.AuditPolicyMatchTypeExact,
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, recorded, err := service.RecordCandidate(context.Background(), auditstore.AuditCandidate{
		Source:        auditstore.AuditSourceRequest,
		Action:        "GET /api/monitor/server-status",
		RequestMethod: "GET",
		RequestPath:   "/api/monitor/server-status",
		Success:       true,
	})
	if err != nil {
		t.Fatalf("record candidate: %v", err)
	}
	if recorded {
		t.Fatal("expected candidate to be excluded")
	}
	if repo.createdInput.Action != "" {
		t.Fatalf("expected no record write, got %#v", repo.createdInput)
	}
}

func TestServiceRecordCandidateWritesPolicyDecisionMetadata(t *testing.T) {
	repo := &stubAuditRepository{
		policyRules: []auditstore.AuditPolicyRule{
			{
				ID:        11,
				Name:      "security.auth.permission_denied",
				Source:    auditstore.AuditSourceSecurityEvent,
				Enabled:   true,
				Priority:  10,
				Effect:    auditstore.AuditPolicyEffectInclude,
				EventType: "auth.permission.denied",
				MatchType: auditstore.AuditPolicyMatchTypeExact,
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, recorded, err := service.RecordCandidate(context.Background(), auditstore.AuditCandidate{
		Source:        auditstore.AuditSourceSecurityEvent,
		Action:        "auth.permission.denied",
		EventType:     "auth.permission.denied",
		RequestMethod: "GET",
		RequestPath:   "/api/roles",
		RequestID:     "req-denied",
		Success:       false,
		Message:       "auth.forbidden",
		StatusCode:    403,
	})
	if err != nil {
		t.Fatalf("record candidate: %v", err)
	}
	if !recorded {
		t.Fatal("expected candidate to be recorded")
	}

	var metadata map[string]any
	if err := json.Unmarshal(repo.createdInput.Metadata, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata["audit_source"] != string(auditstore.AuditSourceSecurityEvent) {
		t.Fatalf("expected audit source metadata, got %#v", metadata)
	}
	if metadata["request_path"] != "/api/roles" {
		t.Fatalf("expected request path metadata, got %#v", metadata)
	}
	if metadata["policy_rule_name"] != "security.auth.permission_denied" {
		t.Fatalf("expected matched rule metadata, got %#v", metadata)
	}
}
