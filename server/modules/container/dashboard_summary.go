package container

import (
	"cmp"
	"slices"
	"strings"
)

const (
	containerDashboardTopLimit     = 3
	containerDashboardAnomalyLimit = 5
	containerDashboardPercentScale = 100
	containerDashboardRankHighLoad = 1
	containerDashboardRankCPU      = 2
	containerDashboardRankState    = 3
	containerDashboardRankHealth   = 4
)

type dashboardSummaryResult struct {
	Overview  containerDashboardOverview
	Hotspots  containerDashboardHotspots
	Anomalies []containerDashboardAnomalyItem
}

type containerDashboardOverview struct {
	RunningContainers     int
	AbnormalContainers    int
	CPUTotalPercent       float64
	MemoryTotalUsageBytes int64
	MemoryTotalLimitBytes int64
	MemoryTotalPercent    *float64
}

type containerDashboardHotspots struct {
	CPUTop    []containerDashboardTopItem
	MemoryTop []containerDashboardTopItem
}

type containerDashboardTopItem struct {
	ID           string
	Name         string
	ShortID      string
	Image        string
	State        string
	Health       string
	RestartCount *int
	Resource     ResourceSummary
}

type containerDashboardAnomalyItem struct {
	ID           string
	Name         string
	ShortID      string
	Image        string
	State        string
	Health       string
	RestartCount *int
	Resource     ResourceSummary
}

func buildContainerDashboardSummary(items []Summary) dashboardSummaryResult {
	overview := accumulateDashboardOverview(items)
	return dashboardSummaryResult{
		Overview: overview,
		Hotspots: containerDashboardHotspots{
			CPUTop:    buildDashboardTopItems(items, dashboardSortByCPU),
			MemoryTop: buildDashboardTopItems(items, dashboardSortByMemory),
		},
		Anomalies: buildDashboardAnomalyItems(items),
	}
}

func accumulateDashboardOverview(items []Summary) containerDashboardOverview {
	overview := containerDashboardOverview{}
	for _, item := range items {
		accumulateDashboardOverviewItem(&overview, item)
	}
	if overview.MemoryTotalLimitBytes > 0 {
		value := (float64(overview.MemoryTotalUsageBytes) / float64(overview.MemoryTotalLimitBytes)) * containerDashboardPercentScale
		overview.MemoryTotalPercent = &value
	}
	return overview
}

func accumulateDashboardOverviewItem(overview *containerDashboardOverview, item Summary) {
	if overview == nil {
		return
	}
	if isDashboardRunningState(item.State) {
		overview.RunningContainers++
	}
	if isDashboardAbnormal(item) {
		overview.AbnormalContainers++
	}
	if item.Resource.CPUPercent != nil && *item.Resource.CPUPercent > 0 {
		overview.CPUTotalPercent += *item.Resource.CPUPercent
	}
	if item.Resource.MemoryUsageBytes != nil && *item.Resource.MemoryUsageBytes > 0 {
		overview.MemoryTotalUsageBytes += *item.Resource.MemoryUsageBytes
	}
	if item.Resource.MemoryLimitBytes != nil && *item.Resource.MemoryLimitBytes > 0 {
		overview.MemoryTotalLimitBytes += *item.Resource.MemoryLimitBytes
	}
}

func buildDashboardTopItems(items []Summary, less func(a Summary, b Summary) int) []containerDashboardTopItem {
	filtered := make([]Summary, 0, len(items))
	for _, item := range items {
		if hasUsableDashboardResource(item.Resource) {
			filtered = append(filtered, item)
		}
	}
	slices.SortStableFunc(filtered, less)
	if len(filtered) > containerDashboardTopLimit {
		filtered = filtered[:containerDashboardTopLimit]
	}
	result := make([]containerDashboardTopItem, 0, len(filtered))
	for _, item := range filtered {
		result = append(result, toDashboardTopItem(item))
	}
	return result
}

func buildDashboardAnomalyItems(items []Summary) []containerDashboardAnomalyItem {
	filtered := make([]Summary, 0, len(items))
	for _, item := range items {
		if dashboardAnomalyRank(item) > 0 {
			filtered = append(filtered, item)
		}
	}
	slices.SortStableFunc(filtered, dashboardSortByAnomaly)
	if len(filtered) > containerDashboardAnomalyLimit {
		filtered = filtered[:containerDashboardAnomalyLimit]
	}
	result := make([]containerDashboardAnomalyItem, 0, len(filtered))
	for _, item := range filtered {
		result = append(result, toDashboardAnomalyItem(item))
	}
	return result
}

func toDashboardTopItem(item Summary) containerDashboardTopItem {
	return containerDashboardTopItem{
		ID:           item.ID,
		Name:         item.Name,
		ShortID:      item.ShortID,
		Image:        item.Image,
		State:        item.State,
		Health:       item.Health,
		RestartCount: item.RestartCount,
		Resource:     item.Resource,
	}
}

func toDashboardAnomalyItem(item Summary) containerDashboardAnomalyItem {
	return containerDashboardAnomalyItem{
		ID:           item.ID,
		Name:         item.Name,
		ShortID:      item.ShortID,
		Image:        item.Image,
		State:        item.State,
		Health:       item.Health,
		RestartCount: item.RestartCount,
		Resource:     item.Resource,
	}
}

func dashboardSortByCPU(a Summary, b Summary) int {
	return compareDashboardMetric(resourceCPUPercent(a.Resource), resourceCPUPercent(b.Resource), a, b)
}

func dashboardSortByMemory(a Summary, b Summary) int {
	return compareDashboardMetric(resourceMemoryUsage(a.Resource), resourceMemoryUsage(b.Resource), a, b)
}

func dashboardSortByAnomaly(a Summary, b Summary) int {
	if diff := cmp.Compare(dashboardAnomalyRank(b), dashboardAnomalyRank(a)); diff != 0 {
		return diff
	}
	if diff := cmp.Compare(dashboardRestartCount(b), dashboardRestartCount(a)); diff != 0 {
		return diff
	}
	if diff := cmp.Compare(resourceCPUPercent(b.Resource), resourceCPUPercent(a.Resource)); diff != 0 {
		return diff
	}
	if diff := cmp.Compare(resourceMemoryUsage(b.Resource), resourceMemoryUsage(a.Resource)); diff != 0 {
		return diff
	}
	return compareSummaryIdentity(a, b)
}

func compareDashboardMetric(metricA float64, metricB float64, a Summary, b Summary) int {
	if diff := cmp.Compare(metricB, metricA); diff != 0 {
		return diff
	}
	return compareSummaryIdentity(a, b)
}

func compareSummaryIdentity(a Summary, b Summary) int {
	if diff := cmp.Compare(strings.TrimSpace(a.Name), strings.TrimSpace(b.Name)); diff != 0 {
		return diff
	}
	return cmp.Compare(a.ID, b.ID)
}

func dashboardAnomalyRank(item Summary) int {
	switch {
	case strings.EqualFold(item.Health, containerHealthUnhealthy):
		return containerDashboardRankHealth
	case isDashboardAbnormalState(item.State):
		return containerDashboardRankState
	case resourceCPUPercent(item.Resource) > 0:
		return containerDashboardRankCPU
	case resourceMemoryUsage(item.Resource) > 0:
		return containerDashboardRankHighLoad
	default:
		return 0
	}
}

func isDashboardAbnormal(item Summary) bool {
	return strings.EqualFold(item.Health, containerHealthUnhealthy) || isDashboardAbnormalState(item.State)
}

func isDashboardAbnormalState(state string) bool {
	switch normalizeContainerState(state) {
	case "restarting", "exited", "dead":
		return true
	default:
		return false
	}
}

func isDashboardRunningState(state string) bool {
	return normalizeContainerState(state) == "running"
}

func hasUsableDashboardResource(resource ResourceSummary) bool {
	return resourceCPUPercent(resource) > 0 || resourceMemoryUsage(resource) > 0
}

func resourceCPUPercent(resource ResourceSummary) float64 {
	if resource.CPUPercent == nil || *resource.CPUPercent < 0 {
		return 0
	}
	return *resource.CPUPercent
}

func resourceMemoryUsage(resource ResourceSummary) float64 {
	if resource.MemoryUsageBytes == nil || *resource.MemoryUsageBytes < 0 {
		return 0
	}
	return float64(*resource.MemoryUsageBytes)
}

func dashboardRestartCount(item Summary) int {
	if item.RestartCount == nil || *item.RestartCount < 0 {
		return 0
	}
	return *item.RestartCount
}
