package audit

import (
	"context"
	"strings"

	auditstore "graft/server/modules/audit/store"
)

// PolicyEvaluator applies plugin-owned audit policy rules to normalized candidates.
type PolicyEvaluator struct {
	repo auditstore.AuditRepository
}

// NewPolicyEvaluator creates an evaluator backed by the audit plugin repository.
func NewPolicyEvaluator(repo auditstore.AuditRepository) (*PolicyEvaluator, error) {
	if repo == nil {
		return nil, ErrNilAuditRepository
	}

	return &PolicyEvaluator{repo: repo}, nil
}

// Evaluate loads policy rules and returns the first matching decision.
func (e *PolicyEvaluator) Evaluate(ctx context.Context, candidate auditstore.AuditCandidate) (auditstore.AuditPolicyDecision, error) {
	if e == nil || e.repo == nil {
		return auditstore.AuditPolicyDecision{}, ErrAuditServiceUnavailable
	}

	rules, err := e.repo.ListAuditPolicyRules(ctx)
	if err != nil {
		return auditstore.AuditPolicyDecision{}, err
	}

	for index := range rules {
		rule := rules[index]
		if !rule.Enabled {
			continue
		}
		if !ruleMatchesCandidate(rule, candidate) {
			continue
		}

		return auditstore.AuditPolicyDecision{
			Matched: true,
			Allowed: rule.Effect == auditstore.AuditPolicyEffectInclude,
			Rule:    &rule,
		}, nil
	}

	return auditstore.AuditPolicyDecision{}, nil
}

func ruleMatchesCandidate(rule auditstore.AuditPolicyRule, candidate auditstore.AuditCandidate) bool {
	if !methodMatches(rule.Method, candidate.RequestMethod) {
		return false
	}
	if !sourceMatches(rule.Source, candidate.Source) {
		return false
	}
	if !pathMatches(rule.MatchType, rule.PathPattern, candidate.RequestPath) {
		return false
	}
	if !fieldMatches(rule.EventType, candidate.EventType) {
		return false
	}
	if !fieldMatches(strings.ToUpper(rule.TargetType), strings.ToUpper(strings.TrimSpace(candidate.TargetType))) {
		return false
	}
	if rule.RiskLevel != "" && normalizeAuditRiskLevel(rule.RiskLevel) != normalizeAuditRiskLevel(classifyCandidateRiskLevel(candidate)) {
		return false
	}

	return true
}

func sourceMatches(ruleSource auditstore.AuditSource, candidateSource auditstore.AuditSource) bool {
	ruleSource = auditstore.AuditSource(strings.ToUpper(strings.TrimSpace(string(ruleSource))))
	candidateSource = auditstore.AuditSource(strings.ToUpper(strings.TrimSpace(string(candidateSource))))
	return ruleSource == "" || ruleSource == candidateSource
}

func methodMatches(ruleMethod string, requestMethod string) bool {
	ruleMethod = strings.ToUpper(strings.TrimSpace(ruleMethod))
	requestMethod = strings.ToUpper(strings.TrimSpace(requestMethod))

	return ruleMethod == "" || ruleMethod == "*" || ruleMethod == requestMethod
}

func pathMatches(matchType auditstore.AuditPolicyMatchType, pattern string, path string) bool {
	pattern = strings.TrimSpace(pattern)
	path = strings.TrimSpace(path)
	if pattern == "" {
		return true
	}

	switch matchType {
	case auditstore.AuditPolicyMatchTypePrefix:
		return strings.HasPrefix(path, pattern)
	default:
		return path == pattern
	}
}

func fieldMatches(ruleValue string, candidateValue string) bool {
	ruleValue = strings.TrimSpace(ruleValue)
	candidateValue = strings.TrimSpace(candidateValue)
	return ruleValue == "" || strings.EqualFold(ruleValue, candidateValue)
}
