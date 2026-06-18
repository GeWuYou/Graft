// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package monitor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/container"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	monitorcontract "graft/server/modules/monitor/contract"
)

type incidentEvidenceCapability struct {
	module *Module
	ctx    *module.Context
}

func registerIncidentEvidenceCapability(ctx *module.Context, instance *Module) error {
	if ctx == nil || ctx.Services == nil {
		return errors.New("module context services are unavailable")
	}
	if instance == nil {
		return errors.New("monitor module instance is unavailable")
	}

	return ctx.Services.RegisterSingleton((*moduleapi.MonitorIncidentEvidenceService)(nil), func(_ container.Resolver) (any, error) {
		return incidentEvidenceCapability{module: instance, ctx: ctx}, nil
	})
}

func (c incidentEvidenceCapability) ResolveAuditIncidentMonitorEvidence(
	ctx context.Context,
	input moduleapi.ResolveAuditIncidentMonitorEvidenceInput,
) (moduleapi.ResolvedAuditIncidentMonitorEvidence, error) {
	if c.module == nil || c.ctx == nil {
		return moduleapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: moduleapi.MonitorEvidenceCapabilityUnavailable,
			Summary:      "Monitor capability is unavailable for this incident.",
			Reason:       "Monitor capability is unavailable.",
		}, errors.New("monitor incident evidence capability is unavailable")
	}
	if input.IncidentStartedAt.IsZero() || input.IncidentEndedAt.IsZero() {
		return moduleapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: moduleapi.MonitorEvidenceCapabilityUnavailable,
			Summary:      "Monitor capability requires a bounded incident window.",
			Reason:       "Monitor capability is unavailable.",
		}, errors.New("incident time window is required")
	}

	now := time.Now().UTC()
	if input.IncidentEndedAt.Before(now.Add(-maxTrendRetentionWindow)) {
		return moduleapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: moduleapi.MonitorEvidenceExpired,
			Summary:      "Monitor evidence expired from the bounded short-retention window.",
			Reason:       "Matching monitor evidence has expired due to short retention.",
			EvidenceLinks: []moduleapi.MonitorEvidenceLink{
				unavailableMonitorEvidenceLink(input.IncidentStartedAt, input.IncidentEndedAt, "Short-retention monitor samples are no longer available for this incident window."),
			},
		}, nil
	}

	response, err := buildServerStatusResponse(ctx, c.ctx, c.module, monitorcontract.TrendRange1Hour)
	if err != nil {
		return moduleapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: moduleapi.MonitorEvidenceCapabilityUnavailable,
			Summary:      "Monitor capability could not reconstruct current anomaly context.",
			Reason:       "Monitor capability is unavailable.",
		}, fmt.Errorf("resolve audit incident monitor evidence: %w", err)
	}

	anomaly, ok := matchIncidentAnomaly(response.Anomalies, input)
	if !ok {
		return moduleapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: moduleapi.MonitorEvidenceNoMatch,
			Summary:      "No matching monitor-owned anomaly is attached to this audit incident.",
			Reason:       "No matching anomaly exists for the current bounded incident context.",
			EvidenceLinks: []moduleapi.MonitorEvidenceLink{
				unavailableMonitorEvidenceLink(input.IncidentStartedAt, input.IncidentEndedAt, "Monitor owns no matching anomaly for the bounded incident context."),
			},
		}, nil
	}

	observedAt := anomaly.ObservedAt.UTC()
	return moduleapi.ResolvedAuditIncidentMonitorEvidence{
		Availability:  moduleapi.MonitorEvidenceAvailable,
		Summary:       anomaly.Summary,
		AnomalyKey:    string(anomaly.AnomalyKey),
		ScopeKind:     string(anomaly.ScopeKind),
		ScopeRef:      anomaly.ScopeRef,
		ObservedAt:    &observedAt,
		EvidenceLinks: toMonitorEvidenceLinks(anomaly.EvidenceLinks, input),
	}, nil
}

func matchIncidentAnomaly(
	anomalies []generated.ServerStatusAnomaly,
	input moduleapi.ResolveAuditIncidentMonitorEvidenceInput,
) (generated.ServerStatusAnomaly, bool) {
	for _, anomaly := range anomalies {
		if anomaly.ObservedAt.Before(input.IncidentStartedAt) || anomaly.ObservedAt.After(input.IncidentEndedAt.Add(trendSampleInterval)) {
			continue
		}
		if scopeMatchesResource(string(anomaly.ScopeKind), anomaly.ScopeRef, input.ResourceType, input.ResourceID, input.ResourceName) {
			return anomaly, true
		}
	}
	return generated.ServerStatusAnomaly{}, false
}

func scopeMatchesResource(scopeKind string, scopeRef string, resourceType string, resourceID string, resourceName string) bool {
	scopeKind = strings.TrimSpace(scopeKind)
	scopeRef = strings.TrimSpace(scopeRef)
	resourceType = strings.TrimSpace(resourceType)
	resourceID = strings.TrimSpace(resourceID)
	resourceName = strings.TrimSpace(resourceName)

	switch scopeKind {
	case scopeKindDependency:
		return resourceMatchesExactKind(resourceType, "dependency", scopeRef, resourceID, resourceName)
	case scopeKindModule:
		return resourceMatchesExactKind(resourceType, "module", scopeRef, resourceID, resourceName)
	case scopeKindRuntime:
		return strings.EqualFold(resourceType, "runtime")
	case scopeKindResource:
		return resourceMatchesGenericScope(resourceType, scopeRef, resourceID, resourceName)
	default:
		return false
	}
}

func resourceMatchesExactKind(resourceType string, expectedType string, scopeRef string, resourceID string, resourceName string) bool {
	return strings.EqualFold(resourceType, expectedType) && resourceMatchesIdentifier(scopeRef, resourceID, resourceName)
}

func resourceMatchesGenericScope(resourceType string, scopeRef string, resourceID string, resourceName string) bool {
	if strings.EqualFold(resourceType, "resource") && resourceMatchesIdentifier(scopeRef, resourceID, resourceName) {
		return true
	}
	return resourceType != "" && strings.Contains(strings.ToLower(scopeRef), strings.ToLower(resourceType))
}

func resourceMatchesIdentifier(scopeRef string, resourceID string, resourceName string) bool {
	return strings.EqualFold(resourceID, scopeRef) || strings.EqualFold(resourceName, scopeRef)
}

func toMonitorEvidenceLinks(
	links []generated.EvidenceLink,
	input moduleapi.ResolveAuditIncidentMonitorEvidenceInput,
) []moduleapi.MonitorEvidenceLink {
	if len(links) == 0 {
		return []moduleapi.MonitorEvidenceLink{
			unavailableMonitorEvidenceLink(input.IncidentStartedAt, input.IncidentEndedAt, "Monitor anomaly exists but no drilldown evidence links are available."),
		}
	}

	converted := make([]moduleapi.MonitorEvidenceLink, 0, len(links))
	for _, link := range links {
		entry := moduleapi.MonitorEvidenceLink{
			TargetKind: string(link.TargetKind),
			LinkState:  string(link.LinkState),
			TitleKey:   valueOrEmpty(link.TitleKey),
			Title:      link.Title,
		}
		if link.Reason != nil {
			entry.Reason = *link.Reason
		}
		if link.TimeWindow != nil {
			entry.TimeWindow = &moduleapi.MonitorEvidenceLinkTimeWindow{
				CreatedFrom: link.TimeWindow.CreatedFrom,
				CreatedTo:   link.TimeWindow.CreatedTo,
			}
		}
		if link.AuditContext != nil {
			entry.AuditContext = &moduleapi.MonitorAuditEvidenceContext{
				CreatedFrom:  link.AuditContext.CreatedFrom,
				CreatedTo:    link.AuditContext.CreatedTo,
				ResourceType: valueOrEmpty(link.AuditContext.ResourceType),
				ResourceID:   valueOrEmpty(link.AuditContext.ResourceId),
				ResourceName: valueOrEmpty(link.AuditContext.ResourceName),
				RequestID:    valueOrEmpty(link.AuditContext.RequestId),
				Result:       valueOrEmpty(link.AuditContext.Result),
				RiskLevel:    valueOrEmpty(link.AuditContext.RiskLevel),
				Source:       valueOrEmpty(link.AuditContext.Source),
			}
		}
		if link.IncidentSeed != nil {
			if incidentSeed, ok := monitorIncidentSeedLink(link.IncidentSeed.EventId); ok {
				entry.IncidentSeed = incidentSeed
			}
		}
		converted = append(converted, entry)
	}

	return converted
}

func unavailableMonitorEvidenceLink(start time.Time, end time.Time, reason string) moduleapi.MonitorEvidenceLink {
	return moduleapi.MonitorEvidenceLink{
		TargetKind: evidenceTargetAudit,
		LinkState:  evidenceStateUnavailable,
		TitleKey:   "monitor.evidence.unavailable.title",
		Title:      "",
		Reason:     reason,
		TimeWindow: &moduleapi.MonitorEvidenceLinkTimeWindow{
			CreatedFrom: start,
			CreatedTo:   end,
		},
	}
}

func valueOrEmpty[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func monitorIncidentSeedLink(eventID int64) (*moduleapi.MonitorIncidentSeedLink, bool) {
	if eventID < 0 {
		return nil, false
	}
	return &moduleapi.MonitorIncidentSeedLink{EventID: uint64(eventID)}, true
}
