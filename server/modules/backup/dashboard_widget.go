package backup

import (
	"context"
	"fmt"
	"time"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	backupcontract "graft/server/modules/backup/contract"
)

const (
	backupHealthWidgetID    = "platform-backup.latest-health"
	backupHealthWidgetOrder = 130
)

// registerBackupDashboardWidget 注册最近备份事实；Dashboard Registry 缺失时跳过可选展示。
func registerBackupDashboardWidget(ctx *module.Context, service *Service) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}
	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             backupHealthWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       "dashboard.widget.backupHealth.title",
		DescriptionKey: "dashboard.widget.backupHealth.description",
		Type:           dashboard.WidgetTypeHealth,
		Size:           dashboard.WidgetSizeSmall,
		Category:       dashboard.WidgetCategoryOperation,
		Priority:       dashboard.WidgetPriorityNormal,
		Order:          backupHealthWidgetOrder,
		RouteLocation:  backupcontract.BackupMenuPath,
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.widget.backupHealth.action",
			Route:    backupcontract.BackupMenuPath,
		},
		RequiredPermissions: []string{backupcontract.BackupReadPermission},
		Loader: dashboard.WidgetLoaderFunc(func(loadCtx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			return loadBackupHealthWidget(loadCtx, service, time.Now().UTC())
		}),
	}); err != nil {
		return fmt.Errorf("register backup dashboard widget: %w", err)
	}
	return nil
}

// loadBackupHealthWidget 只读取最近一条备份摘要，不读取工件内容，也不推断未被记录的保护能力。
func loadBackupHealthWidget(ctx context.Context, service *Service, now time.Time) (dashboard.WidgetPayload, error) {
	items, _, err := service.ListSummaries(ctx, 1, 0)
	if err != nil {
		return nil, err
	}
	return backupHealthPayload(items, now), nil
}

func backupHealthPayload(items []moduleapi.BackupSummary, now time.Time) dashboard.WidgetPayload {
	status, key, label := backupHealthStatus(items, now)
	state := dashboard.WidgetStateNormal
	priority := dashboard.WidgetPriorityNormal
	if status == dashboard.HealthStatusDegraded {
		state = dashboard.WidgetStateWarning
		priority = dashboard.WidgetPriorityWarning
	}
	return dashboard.WidgetPayload{
		"summary": dashboard.HealthSummaryItem{
			Status:   status,
			LabelKey: "dashboard.widget.backupHealth." + key + ".summary",
			Label:    label,
		},
		"items": []dashboard.HealthItem{{
			Key:            "latest-backup",
			LabelKey:       "dashboard.widget.backupHealth." + key + ".label",
			Label:          label,
			Status:         status,
			DescriptionKey: "dashboard.widget.backupHealth." + key + ".description",
			RouteLocation:  backupcontract.BackupMenuPath,
		}},
		"state":    string(state),
		"priority": string(priority),
	}
}

func backupHealthStatus(items []moduleapi.BackupSummary, now time.Time) (dashboard.HealthStatus, string, string) {
	if len(items) == 0 {
		return dashboard.HealthStatusUnknown, "none", "No backup record"
	}
	latest := items[0]
	switch latest.Status {
	case moduleapi.BackupStatusAvailable:
		if latest.RetainUntil.IsZero() {
			return dashboard.HealthStatusUnknown, "unknown", "Backup status unknown"
		}
		if !latest.RetainUntil.After(now) {
			return dashboard.HealthStatusDegraded, "expired", "Backup expired"
		}
		return dashboard.HealthStatusHealthy, "available", "Backup available"
	case moduleapi.BackupStatusExpired:
		return dashboard.HealthStatusDegraded, "expired", "Backup expired"
	case moduleapi.BackupStatusRestored:
		return dashboard.HealthStatusUnknown, "restored", "Restore recorded"
	default:
		return dashboard.HealthStatusUnknown, "unknown", "Backup status unknown"
	}
}
