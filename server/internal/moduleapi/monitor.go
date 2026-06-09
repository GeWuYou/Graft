// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package moduleapi

import (
	"context"
	"time"
)

// MonitorEvidenceAvailability describes whether monitor-owned evidence can be attached to an audit incident.
type MonitorEvidenceAvailability string

const (
	// MonitorEvidenceAvailable means monitor evidence was resolved for the incident window.
	MonitorEvidenceAvailable MonitorEvidenceAvailability = "available"
	// MonitorEvidenceModuleDisabled means the monitor module is disabled.
	MonitorEvidenceModuleDisabled MonitorEvidenceAvailability = "module_disabled"
	// MonitorEvidenceNoMatch means no monitor anomaly matched the bounded incident context.
	MonitorEvidenceNoMatch MonitorEvidenceAvailability = "no_match"
	// MonitorEvidenceExpired means the incident window is older than monitor evidence retention.
	MonitorEvidenceExpired MonitorEvidenceAvailability = "expired"
	// MonitorEvidenceCapabilityUnavailable means the monitor capability could not serve evidence.
	MonitorEvidenceCapabilityUnavailable MonitorEvidenceAvailability = "capability_unavailable"
)

// MonitorEvidenceLinkTimeWindow describes the time range covered by an evidence link.
type MonitorEvidenceLinkTimeWindow struct {
	CreatedFrom time.Time
	CreatedTo   time.Time
}

// MonitorAuditEvidenceContext narrows an evidence link to related audit search dimensions.
type MonitorAuditEvidenceContext struct {
	Action       string
	ActionPrefix string
	Source       string
	ResourceType string
	ResourceID   string
	ResourceName string
	RequestID    string
	Result       string
	RiskLevel    string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
}

// MonitorIncidentSeedLink points back to the seed audit event for the incident.
type MonitorIncidentSeedLink struct {
	EventID uint64
}

// MonitorEvidenceLink describes one monitor-owned drilldown target for an audit incident.
type MonitorEvidenceLink struct {
	TargetKind   string
	LinkState    string
	Title        string
	Reason       string
	TimeWindow   *MonitorEvidenceLinkTimeWindow
	AuditContext *MonitorAuditEvidenceContext
	IncidentSeed *MonitorIncidentSeedLink
}

// ResolveAuditIncidentMonitorEvidenceInput carries the bounded incident context exposed to monitor.
type ResolveAuditIncidentMonitorEvidenceInput struct {
	IncidentSeedEventID uint64
	IncidentStartedAt   time.Time
	IncidentEndedAt     time.Time
	RequestID           string
	ResourceType        string
	ResourceID          string
	ResourceName        string
	AuditSource         string
	AuditResult         string
	AuditRiskLevel      string
}

// ResolvedAuditIncidentMonitorEvidence is the monitor capability response for an audit incident.
type ResolvedAuditIncidentMonitorEvidence struct {
	Availability  MonitorEvidenceAvailability
	Summary       string
	Reason        string
	AnomalyKey    string
	ScopeKind     string
	ScopeRef      string
	ObservedAt    *time.Time
	EvidenceLinks []MonitorEvidenceLink
}

// MonitorIncidentEvidenceService resolves monitor-owned anomaly evidence for audit incidents.
type MonitorIncidentEvidenceService interface {
	ResolveAuditIncidentMonitorEvidence(
		ctx context.Context,
		input ResolveAuditIncidentMonitorEvidenceInput,
	) (ResolvedAuditIncidentMonitorEvidence, error)
}
