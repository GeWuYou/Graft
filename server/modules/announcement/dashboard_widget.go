package announcement

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	announcementcontract "graft/server/modules/announcement/contract"
	announcementstore "graft/server/modules/announcement/store"
)

const (
	announcementTimelineWidgetID    = "announcement.current-user-timeline"
	announcementTimelineWidgetOrder = 200
	announcementTimelineLimit       = 5
)

// registerAnnouncementDashboardWidget 注册当前用户可见公告时间线；Dashboard Registry 缺失时跳过可选展示。
func registerAnnouncementDashboardWidget(ctx *module.Context, service *Service) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}

	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             announcementTimelineWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       "dashboard.widget.announcementTimeline.title",
		DescriptionKey: "dashboard.widget.announcementTimeline.description",
		Type:           dashboard.WidgetTypeTimeline,
		Size:           dashboard.WidgetSizeMedium,
		Category:       dashboard.WidgetCategoryOperation,
		Priority:       dashboard.WidgetPriorityInfo,
		Order:          announcementTimelineWidgetOrder,
		RouteLocation:  announcementcontract.MyAnnouncementMenuPath,
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.widget.announcementTimeline.action",
			Route:    announcementcontract.MyAnnouncementMenuPath,
		},
		Loader: dashboard.WidgetLoaderFunc(func(loadCtx context.Context, request dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			return loadAnnouncementTimelineWidget(loadCtx, request, service)
		}),
	}); err != nil {
		return fmt.Errorf("register announcement dashboard widget: %w", err)
	}
	return nil
}

// loadAnnouncementTimelineWidget 使用请求认证用户读取真实公告，不借用管理端权限或跨用户列表。
func loadAnnouncementTimelineWidget(ctx context.Context, request dashboard.WidgetRequest, service *Service) (dashboard.WidgetPayload, error) {
	if service == nil || request.RequestAuth.User == nil || request.RequestAuth.User.ID == 0 {
		return nil, errors.New("announcement dashboard user is unavailable")
	}
	result, err := service.ListCurrentUser(ctx, UserListQuery{
		UserID:   request.RequestAuth.User.ID,
		Page:     1,
		PageSize: announcementTimelineLimit,
	})
	if err != nil {
		return nil, err
	}

	itemCount := len(result.Items)
	if itemCount > announcementTimelineLimit {
		itemCount = announcementTimelineLimit
	}
	items := make([]map[string]any, 0, itemCount)
	for _, item := range result.Items[:itemCount] {
		items = append(items, announcementTimelineItem(item))
	}
	visible := len(items) > 0
	state := dashboard.WidgetStateHidden
	if visible {
		state = dashboard.WidgetStateNormal
	}
	return dashboard.WidgetPayload{
		"items":     items,
		"empty_key": "dashboard.widget.announcementTimeline.empty",
		"visible":   visible,
		"state":     string(state),
	}, nil
}

func announcementTimelineItem(item announcementstore.UserAnnouncement) map[string]any {
	announcement := item.Announcement
	return map[string]any{
		"id":             strconv.FormatUint(announcement.ID, 10),
		"title_key":      "dashboard.widget.announcementTimeline.item." + strconv.FormatUint(announcement.ID, 10),
		"title":          announcement.Title,
		"occurred_at":    announcementTimelineTime(announcement).Format(time.RFC3339Nano),
		"status":         announcementTimelineStatus(announcement.Level),
		"route_location": announcementcontract.MyAnnouncementMenuPath,
	}
}

func announcementTimelineTime(item announcementstore.Announcement) time.Time {
	if item.PublishedAt != nil {
		return item.PublishedAt.UTC()
	}
	if item.PublishAt != nil {
		return item.PublishAt.UTC()
	}
	return item.CreatedAt.UTC()
}

func announcementTimelineStatus(level string) string {
	switch announcementcontract.AnnouncementLevel(level) {
	case announcementcontract.AnnouncementLevelSuccess:
		return "success"
	case announcementcontract.AnnouncementLevelWarning:
		return "warning"
	case announcementcontract.AnnouncementLevelError:
		return "error"
	default:
		return "normal"
	}
}
