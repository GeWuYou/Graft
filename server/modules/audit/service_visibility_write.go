package audit

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
)

type candidateRecordingPolicy struct {
	decision     auditstore.AuditPolicyDecision
	strategy     auditstore.AuditVisibilityStrategy
	shouldRecord bool
}

// resolveCandidateRecordingPolicy 集中决定候选事件是否记录以及最终可见性。
// 可见性策略写请求必须绕过普通排除规则留存证据，并在全局忽略策略下至少以隐藏记录落库。
func (s *Service) resolveCandidateRecordingPolicy(
	ctx context.Context,
	evaluator *PolicyEvaluator,
	candidate auditstore.AuditCandidate,
) (candidateRecordingPolicy, error) {
	mandatoryPolicyWrite := isAuditVisibilityPolicyWriteCandidate(candidate)
	decision := auditstore.AuditPolicyDecision{Matched: true, Allowed: true}
	if !mandatoryPolicyWrite {
		var err error
		decision, err = evaluator.Evaluate(ctx, candidate)
		if err != nil {
			return candidateRecordingPolicy{}, err
		}
		if !decision.Matched || !decision.Allowed {
			return candidateRecordingPolicy{decision: decision}, nil
		}
	}

	strategy, err := s.resolveCandidateVisibilityStrategy(ctx, candidate)
	if err != nil {
		return candidateRecordingPolicy{}, err
	}
	if strategy != auditstore.AuditVisibilityStrategyIgnore {
		return candidateRecordingPolicy{decision: decision, strategy: strategy, shouldRecord: true}, nil
	}
	if mandatoryPolicyWrite {
		return candidateRecordingPolicy{
			decision:     decision,
			strategy:     auditstore.AuditVisibilityStrategyHidden,
			shouldRecord: true,
		}, nil
	}
	return candidateRecordingPolicy{decision: decision, strategy: strategy}, nil
}

func normalizeVisibilityOverrideInput(
	input auditstore.UpsertAuditVisibilityOverrideInput,
) (auditstore.UpsertAuditVisibilityOverrideInput, error) {
	source, actionKey, err := normalizeVisibilityOverrideRef(input.Source, input.ActionKey)
	if err != nil {
		return auditstore.UpsertAuditVisibilityOverrideInput{}, err
	}
	strategy := normalizeAuditVisibilityStrategy(input.Strategy)
	if strategy == "" {
		return auditstore.UpsertAuditVisibilityOverrideInput{}, fmt.Errorf("%w: audit visibility override strategy is required", auditstore.ErrAuditValidation)
	}
	if strategy == auditstore.AuditVisibilityStrategyIgnore && isProtectedAuditVisibilityPolicyWriteAction(source, actionKey) {
		return auditstore.UpsertAuditVisibilityOverrideInput{}, fmt.Errorf("%w: audit visibility policy writes cannot be ignored", auditstore.ErrAuditValidation)
	}
	return auditstore.UpsertAuditVisibilityOverrideInput{
		Source:      source,
		ActionKey:   actionKey,
		Strategy:    strategy,
		Description: strings.TrimSpace(input.Description),
		Actor: auditstore.AuditVisibilityActor{
			UserID:   input.Actor.UserID,
			Username: strings.TrimSpace(input.Actor.Username),
		},
	}, nil
}

func isProtectedAuditVisibilityPolicyWriteAction(source auditstore.AuditSource, actionKey string) bool {
	if source != auditstore.AuditSourceRequest {
		return false
	}
	return isAuditVisibilityPolicyWriteAction(strings.TrimSpace(actionKey))
}

func isAuditVisibilityPolicyWriteCandidate(candidate auditstore.AuditCandidate) bool {
	if normalizeAuditSource(candidate.Source) != auditstore.AuditSourceRequest {
		return false
	}
	action := strings.ToUpper(strings.TrimSpace(candidate.RequestMethod)) + " " + strings.TrimSpace(candidate.RequestPath)
	return isAuditVisibilityPolicyWriteAction(action)
}

func isAuditVisibilityPolicyWriteAction(action string) bool {
	switch action {
	case http.MethodPut + " " + "/api" + auditcontract.AuditVisibilityPolicyAPIPath,
		http.MethodPut + " " + "/api" + auditcontract.AuditVisibilityOverrideAPIPath,
		http.MethodDelete + " " + "/api" + auditcontract.AuditVisibilityOverrideAPIPath,
		http.MethodPut + " " + "/api" + auditcontract.AuditVisibilityOverrideBatchAPIPath:
		return true
	default:
		return false
	}
}
