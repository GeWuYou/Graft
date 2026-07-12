package audit

import (
	"context"

	"graft/server/internal/moduleapi"
	auditstore "graft/server/modules/audit/store"
)

type auditSecurityReader struct {
	reader *Service
}

func (r auditSecurityReader) ReadSecuritySnapshot(ctx context.Context, preset moduleapi.AuditOverviewPreset) (moduleapi.AuditSecuritySnapshot, error) {
	overview, err := r.reader.Overview(ctx, auditstore.AuditTimePreset(preset))
	if err != nil {
		return moduleapi.AuditSecuritySnapshot{}, err
	}
	snapshot := moduleapi.AuditSecuritySnapshot{
		TimePreset:          preset,
		TotalLogs:           overview.Summary.TotalLogs,
		FailedOperations:    overview.Summary.FailedOperations,
		HighRiskEvents:      overview.Summary.HighRiskEvents,
		SensitiveOperations: overview.Summary.SensitiveOperations,
		RiskGroups:          make([]moduleapi.AuditSecurityRiskGroup, 0, len(overview.RiskGroups)),
		RecentEvents:        make([]moduleapi.AuditSecurityEvent, 0, len(overview.SecurityTimeline)),
	}
	for _, group := range overview.RiskGroups {
		snapshot.RiskGroups = append(snapshot.RiskGroups, moduleapi.AuditSecurityRiskGroup{
			Key: string(group.Key), LabelKey: group.LabelKey, Count: group.Count, RiskLevel: string(group.RiskLevel),
		})
	}
	for _, event := range overview.SecurityTimeline {
		snapshot.RecentEvents = append(snapshot.RecentEvents, moduleapi.AuditSecurityEvent{
			ID: event.ID, CreatedAt: event.CreatedAt, Action: event.Action, Resource: event.ResourceName,
			RiskLevel: string(event.RiskLevel), Result: string(event.Result), RequestID: event.RequestID,
		})
	}
	return snapshot, nil
}

var _ moduleapi.AuditSecurityReader = auditSecurityReader{}
