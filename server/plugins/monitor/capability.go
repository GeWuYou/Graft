package monitor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/container"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	monitorcontract "graft/server/plugins/monitor/contract"
)

type incidentEvidenceCapability struct {
	plugin *Plugin
	ctx    *plugin.Context
}

func registerIncidentEvidenceCapability(ctx *plugin.Context, instance *Plugin) error {
	if ctx == nil || ctx.Services == nil {
		return errors.New("plugin context services are unavailable")
	}
	if instance == nil {
		return errors.New("monitor plugin instance is unavailable")
	}

	return ctx.Services.RegisterSingleton((*pluginapi.MonitorIncidentEvidenceService)(nil), func(_ container.Resolver) (any, error) {
		return incidentEvidenceCapability{plugin: instance, ctx: ctx}, nil
	})
}

func (c incidentEvidenceCapability) ResolveAuditIncidentMonitorEvidence(
	ctx context.Context,
	input pluginapi.ResolveAuditIncidentMonitorEvidenceInput,
) (pluginapi.ResolvedAuditIncidentMonitorEvidence, error) {
	if c.plugin == nil || c.ctx == nil {
		return pluginapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: pluginapi.MonitorEvidenceCapabilityUnavailable,
			Summary:      "Monitor capability is unavailable for this incident.",
			Reason:       "Monitor capability is unavailable.",
		}, errors.New("monitor incident evidence capability is unavailable")
	}
	if input.IncidentStartedAt.IsZero() || input.IncidentEndedAt.IsZero() {
		return pluginapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: pluginapi.MonitorEvidenceCapabilityUnavailable,
			Summary:      "Monitor capability requires a bounded incident window.",
			Reason:       "Monitor capability is unavailable.",
		}, errors.New("incident time window is required")
	}

	now := time.Now().UTC()
	if input.IncidentEndedAt.Before(now.Add(-maxTrendRetentionWindow)) {
		return pluginapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: pluginapi.MonitorEvidenceExpired,
			Summary:      "Monitor evidence expired from the bounded short-retention window.",
			Reason:       "Matching monitor evidence has expired due to short retention.",
			EvidenceLinks: []pluginapi.MonitorEvidenceLink{
				unavailableMonitorEvidenceLink(input.IncidentStartedAt, input.IncidentEndedAt, "Short-retention monitor samples are no longer available for this incident window."),
			},
		}, nil
	}

	response, err := buildServerStatusResponse(ctx, c.ctx, c.plugin, monitorcontract.TrendRange1Hour)
	if err != nil {
		return pluginapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: pluginapi.MonitorEvidenceCapabilityUnavailable,
			Summary:      "Monitor capability could not reconstruct current anomaly context.",
			Reason:       "Monitor capability is unavailable.",
		}, fmt.Errorf("resolve audit incident monitor evidence: %w", err)
	}

	anomaly, ok := matchIncidentAnomaly(response.Anomalies, input)
	if !ok {
		return pluginapi.ResolvedAuditIncidentMonitorEvidence{
			Availability: pluginapi.MonitorEvidenceNoMatch,
			Summary:      "No matching monitor-owned anomaly is attached to this audit incident.",
			Reason:       "No matching anomaly exists for the current bounded incident context.",
			EvidenceLinks: []pluginapi.MonitorEvidenceLink{
				unavailableMonitorEvidenceLink(input.IncidentStartedAt, input.IncidentEndedAt, "Monitor owns no matching anomaly for the bounded incident context."),
			},
		}, nil
	}

	observedAt := anomaly.ObservedAt.UTC()
	return pluginapi.ResolvedAuditIncidentMonitorEvidence{
		Availability:  pluginapi.MonitorEvidenceAvailable,
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
	input pluginapi.ResolveAuditIncidentMonitorEvidenceInput,
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
	case scopeKindPlugin:
		return resourceMatchesExactKind(resourceType, "plugin", scopeRef, resourceID, resourceName)
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
	input pluginapi.ResolveAuditIncidentMonitorEvidenceInput,
) []pluginapi.MonitorEvidenceLink {
	if len(links) == 0 {
		return []pluginapi.MonitorEvidenceLink{
			unavailableMonitorEvidenceLink(input.IncidentStartedAt, input.IncidentEndedAt, "Monitor anomaly exists but no drilldown evidence links are available."),
		}
	}

	converted := make([]pluginapi.MonitorEvidenceLink, 0, len(links))
	for _, link := range links {
		entry := pluginapi.MonitorEvidenceLink{
			TargetKind: string(link.TargetKind),
			LinkState:  string(link.LinkState),
			Title:      link.Title,
		}
		if link.Reason != nil {
			entry.Reason = *link.Reason
		}
		if link.TimeWindow != nil {
			entry.TimeWindow = &pluginapi.MonitorEvidenceLinkTimeWindow{
				CreatedFrom: link.TimeWindow.CreatedFrom,
				CreatedTo:   link.TimeWindow.CreatedTo,
			}
		}
		if link.AuditContext != nil {
			entry.AuditContext = &pluginapi.MonitorAuditEvidenceContext{
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

func unavailableMonitorEvidenceLink(start time.Time, end time.Time, reason string) pluginapi.MonitorEvidenceLink {
	return pluginapi.MonitorEvidenceLink{
		TargetKind: evidenceTargetAudit,
		LinkState:  evidenceStateUnavailable,
		Title:      "Monitor evidence is unavailable",
		Reason:     reason,
		TimeWindow: &pluginapi.MonitorEvidenceLinkTimeWindow{
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

func monitorIncidentSeedLink(eventID int64) (*pluginapi.MonitorIncidentSeedLink, bool) {
	if eventID < 0 {
		return nil, false
	}
	return &pluginapi.MonitorIncidentSeedLink{EventID: uint64(eventID)}, true
}
