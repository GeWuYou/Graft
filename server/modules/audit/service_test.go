package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"graft/server/internal/drilldown"
	auditstore "graft/server/modules/audit/store"
)

type stubAuditRepository struct {
	createdInput  auditstore.CreateAuditLogInput
	listQuery     auditstore.ListAuditLogsQuery
	overviewWnd   auditstore.AuditTimePreset
	incidentID    uint64
	deletedBefore time.Time
	deletedRows   int64
	policyRules   []auditstore.AuditPolicyRule
	createResult  auditstore.AuditLog
	listResult    auditstore.ListAuditLogsResult
	overview      auditstore.AuditOverview
	incident      auditstore.AuditIncident
	createErr     error
	listErr       error
	overviewErr   error
	incidentErr   error
	policyErr     error
	deleteErr     error
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

func (r *stubAuditRepository) ReadAuditOverview(_ context.Context, window auditstore.AuditTimePreset) (auditstore.AuditOverview, error) {
	r.overviewWnd = window
	if r.overviewErr != nil {
		return auditstore.AuditOverview{}, r.overviewErr
	}
	return r.overview, nil
}

func (r *stubAuditRepository) ReadIncident(_ context.Context, eventID uint64) (auditstore.AuditIncident, error) {
	r.incidentID = eventID
	if r.incidentErr != nil {
		return auditstore.AuditIncident{}, r.incidentErr
	}
	return r.incident, nil
}

func (r *stubAuditRepository) ListAuditPolicyRules(_ context.Context) ([]auditstore.AuditPolicyRule, error) {
	if r.policyErr != nil {
		return nil, r.policyErr
	}
	return append([]auditstore.AuditPolicyRule(nil), r.policyRules...), nil
}

func (r *stubAuditRepository) DeleteAuditLogsBefore(_ context.Context, createdBefore time.Time) (int64, error) {
	r.deletedBefore = createdBefore
	if r.deleteErr != nil {
		return 0, r.deleteErr
	}
	return r.deletedRows, nil
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

func TestServiceDeleteBeforeDelegatesUTCToRepository(t *testing.T) {
	repo := &stubAuditRepository{deletedRows: 3}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	localCutoff := time.Date(2026, 5, 27, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	deleted, err := service.DeleteBefore(context.Background(), localCutoff)
	if err != nil {
		t.Fatalf("delete before: %v", err)
	}

	if deleted != 3 {
		t.Fatalf("expected deleted row count 3, got %d", deleted)
	}
	if !repo.deletedBefore.Equal(localCutoff.UTC()) || repo.deletedBefore.Location() != time.UTC {
		t.Fatalf("expected UTC cutoff %s, got %s", localCutoff.UTC(), repo.deletedBefore)
	}
}

func TestServiceDeleteBeforeRejectsZeroCutoff(t *testing.T) {
	service, err := NewService(&stubAuditRepository{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := service.DeleteBefore(context.Background(), time.Time{}); err == nil {
		t.Fatal("expected zero cutoff error")
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
	if repo.listQuery.TimePreset != "" {
		t.Fatalf("expected list query preset to stay empty by default, got %q", repo.listQuery.TimePreset)
	}
	if result.Page != defaultPage || result.PageSize != maxPageSize || result.Total != 42 || len(result.Items) != 1 {
		t.Fatalf("unexpected list result %#v", result)
	}
}

func TestServiceListPreservesExplicitPreset(t *testing.T) {
	repo := &stubAuditRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.List(context.Background(), ListQuery{
		TimePreset: auditstore.AuditTimePresetLast24Hours,
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}

	if repo.listQuery.TimePreset != auditstore.AuditTimePresetLast24Hours {
		t.Fatalf("expected explicit preset to be preserved, got %q", repo.listQuery.TimePreset)
	}
}

func TestServiceListNormalizesSortExpressions(t *testing.T) {
	repo := &stubAuditRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.List(context.Background(), ListQuery{
		Sorts: []string{" created_at:asc ", "created_at:asc", "invalid", "created_at:desc"},
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}

	if len(repo.listQuery.Sorts) != 1 {
		t.Fatalf("expected one normalized sort after field dedupe, got %#v", repo.listQuery.Sorts)
	}
	if repo.listQuery.Sorts[0] != "created_at:asc" {
		t.Fatalf("unexpected normalized sorts %#v", repo.listQuery.Sorts)
	}
}

type stubDrilldownRepo struct {
	metadata drilldown.ScopeMetadata
	err      error
}

func (r stubDrilldownRepo) GetScope(context.Context, string, string) (drilldown.ScopeMetadata, error) {
	if r.err != nil {
		return drilldown.ScopeMetadata{}, r.err
	}
	return r.metadata, nil
}

type stubListQueryResolver struct {
	resolved drilldown.ResolvedScope[ListQuery]
	err      error
}

func (r stubListQueryResolver) Resolve(_ context.Context, _ drilldown.ScopeMetadata, _ ListQuery) (drilldown.ResolvedScope[ListQuery], error) {
	if r.err != nil {
		return drilldown.ResolvedScope[ListQuery]{}, r.err
	}
	return r.resolved, nil
}

func TestServiceListAppliesScopePatch(t *testing.T) {
	scopeRepo := stubDrilldownRepo{metadata: drilldown.ScopeMetadata{
		Module:       "audit",
		Scope:        "high_risk_operations",
		Name:         "High Risk",
		Description:  "desc",
		Enabled:      true,
		TargetType:   "log_query",
		TargetModule: "audit",
		TargetPage:   "audit_logs",
	}}
	repo := &stubAuditRepository{}
	scopeResolver := stubListQueryResolver{
		resolved: drilldown.ResolvedScope[ListQuery]{
			Applied: drilldown.AppliedScope{Module: "audit", Scope: "high_risk_operations"},
			Projection: drilldown.ScopeProjection{
				Title: "High Risk",
			},
			ConvertibleFilters: drilldown.ConvertibleFilters{
				BusinessCategory: string(auditstore.AuditBusinessCategoryHighRiskOperations),
			},
			QueryPatch: ListQuery{
				BusinessCategory: auditstore.AuditBusinessCategoryHighRiskOperations,
				ActionKeywords:   []string{"sensitive"},
			},
		},
	}
	drilldownService, err := drilldown.NewService[ListQuery, ListQuery](scopeRepo, scopeResolver)
	if err != nil {
		t.Fatalf("new drilldown service: %v", err)
	}
	service, err := NewServiceWithDrilldown(repo, drilldownService)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.List(context.Background(), ListQuery{
		Scope:          "high_risk_operations",
		ActionKeywords: []string{"existing"},
	})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}

	if repo.listQuery.BusinessCategory != auditstore.AuditBusinessCategoryHighRiskOperations {
		t.Fatalf("expected scope business category to be applied, got %q", repo.listQuery.BusinessCategory)
	}
	if len(repo.listQuery.ActionKeywords) != 2 {
		t.Fatalf("expected merged action keywords, got %#v", repo.listQuery.ActionKeywords)
	}
	if result.AppliedScope == nil || result.ScopeProjection == nil || result.ConvertibleFilters == nil {
		t.Fatalf("expected scope metadata in list result, got %#v", result)
	}
}

func TestServiceOverviewNormalizesPreset(t *testing.T) {
	repo := &stubAuditRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := service.Overview(context.Background(), ""); err != nil {
		t.Fatalf("overview: %v", err)
	}
	if repo.overviewWnd != auditstore.AuditTimePresetLast24Hours {
		t.Fatalf("expected default overview preset, got %q", repo.overviewWnd)
	}
}

func TestServiceIncidentRequiresEventID(t *testing.T) {
	repo := &stubAuditRepository{}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.Incident(context.Background(), 0)
	if err == nil || err.Error() != "audit incident event id is required" {
		t.Fatalf("expected missing event id error, got %v", err)
	}
}

func TestServiceRecordCandidateAppliesPolicy(t *testing.T) {
	repo := &stubAuditRepository{
		policyRules: []auditstore.AuditPolicyRule{
			{
				Enabled:     true,
				Effect:      auditstore.AuditPolicyEffectExclude,
				Method:      "GET",
				PathPattern: "/api/users",
				MatchType:   auditstore.AuditPolicyMatchTypePrefix,
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, recorded, err := service.RecordCandidate(context.Background(), auditstore.AuditCandidate{
		Action:        "user.read",
		RequestMethod: "GET",
		RequestPath:   "/api/users/7",
	})
	if err != nil {
		t.Fatalf("record candidate: %v", err)
	}
	if recorded {
		t.Fatal("expected candidate to be filtered by policy")
	}
}

func TestParseOptionalUint64Param(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got, ok, err := parseOptionalUint64Param(stubParamGetter{value: ""}, "id")
		if err != nil || ok || got != 0 {
			t.Fatalf("expected empty result, got value=%d ok=%v err=%v", got, ok, err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, _, err := parseOptionalUint64Param(stubParamGetter{value: "x"}, "id")
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("ok", func(t *testing.T) {
		got, ok, err := parseOptionalUint64Param(stubParamGetter{value: "42"}, "id")
		if err != nil || !ok || got != 42 {
			t.Fatalf("expected parsed value, got value=%d ok=%v err=%v", got, ok, err)
		}
	})
}

type stubParamGetter struct {
	value string
}

func (s stubParamGetter) Param(string) string {
	return s.value
}

func TestPolicyEvaluatorReturnsRepositoryError(t *testing.T) {
	repo := &stubAuditRepository{policyErr: errors.New("policy failed")}
	evaluator, err := NewPolicyEvaluator(repo)
	if err != nil {
		t.Fatalf("new policy evaluator: %v", err)
	}

	_, err = evaluator.Evaluate(context.Background(), auditstore.AuditCandidate{})
	if err == nil || err.Error() != "policy failed" {
		t.Fatalf("expected repository error, got %v", err)
	}
}
