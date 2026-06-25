package container

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

const (
	containerDashboardTopLimit     = 3
	containerDashboardAnomalyLimit = 5
	containerDashboardPercentScale = 100
	containerDashboardRankState    = 1
	containerDashboardRankHealth   = 2
)

type dashboardSummaryResult struct {
	CollectedAt string
	Overview    containerDashboardOverview
	Hotspots    containerDashboardHotspots
	Anomalies   []containerDashboardAnomalyItem
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
	Status       string
	Health       string
	RestartCount *int
	ReasonCode   string
	ReasonLabel  string
	Resource     ResourceSummary
}

const (
	containerDashboardReasonHealthUnhealthy = "health.unhealthy"
	containerDashboardReasonStateRestarting = "state.restarting"
	containerDashboardReasonStateExited     = "state.exited"
	containerDashboardReasonStateDead       = "state.dead"
	containerDashboardReasonStateUnknown    = "state.unknown"
)

// buildContainerDashboardSummary 构建容器仪表盘汇总结果。
// 结果包含采集时间、概览信息、CPU 与内存热点列表以及异常列表。
func buildContainerDashboardSummary(items []Summary) dashboardSummaryResult {
	overview := accumulateDashboardOverview(items)
	return dashboardSummaryResult{
		CollectedAt: dashboardSummaryCollectedAt(items),
		Overview:    overview,
		Hotspots: containerDashboardHotspots{
			CPUTop:    buildDashboardTopItems(items, dashboardSortByCPU),
			MemoryTop: buildDashboardTopItems(items, dashboardSortByMemory),
		},
		Anomalies: buildDashboardAnomalyItems(items),
	}
}

// accumulateDashboardOverview 汇总所有容器摘要的概览数据。
// 当累计到内存限制值大于 0 时，还会计算内存使用占比并写入 MemoryTotalPercent。
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

// accumulateDashboardOverviewItem 汇总单个容器摘要的概览指标。
// 当 overview 为 nil 时直接返回；否则更新运行容器数、异常容器数，以及 CPU 总百分比、内存使用量和内存上限总和。
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

// buildDashboardTopItems 构建按指定指标排序的容器 Top 列表。
// 仅包含具有可用资源数据的项，并将结果限制在仪表盘 Top 数量上限内。
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

// buildDashboardAnomalyItems 构建容器异常项列表，并按优先级截取前 N 项。
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

// toDashboardTopItem 将 Summary 映射为仪表盘热点列表项。
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

// toDashboardAnomalyItem 将 Summary 映射为容器异常列表项。
//
// toDashboardAnomalyItem 将摘要转换为异常项，并补充异常原因信息与资源摘要。
// 它会保留身份、状态、健康状况和重启次数字段。
func toDashboardAnomalyItem(item Summary) containerDashboardAnomalyItem {
	reasonCode, reasonLabel := dashboardAnomalyReason(item)
	return containerDashboardAnomalyItem{
		ID:           item.ID,
		Name:         item.Name,
		ShortID:      item.ShortID,
		Image:        item.Image,
		State:        item.State,
		Status:       item.Status,
		Health:       item.Health,
		RestartCount: item.RestartCount,
		ReasonCode:   reasonCode,
		ReasonLabel:  reasonLabel,
		Resource:     item.Resource,
	}
}

// dashboardSortByCPU 按 CPU 占用对容器摘要进行降序排序。
func dashboardSortByCPU(a Summary, b Summary) int {
	return compareDashboardMetric(resourceCPUPercent(a.Resource), resourceCPUPercent(b.Resource), a, b)
}

// dashboardSortByMemory 按内存使用量对两个摘要进行降序比较，
// 并在内存使用量相同时按容器标识进行稳定排序。
func dashboardSortByMemory(a Summary, b Summary) int {
	return compareDashboardMetric(resourceMemoryUsage(a.Resource), resourceMemoryUsage(b.Resource), a, b)
}

// dashboardSortByAnomaly 按异常等级、重启次数、CPU、内存和标识信息对容器摘要进行排序。
// 排序优先级依次为异常等级、重启次数、CPU 使用率、内存使用量，最后按名称和 ID 作为稳定性兜底。
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

// compareDashboardMetric 按指标值降序比较两个容器摘要，并在相等时按名称和 ID 作为稳定性兜底。
// 返回值遵循比较函数约定：前者应排在后者之前时返回负数，之后返回正数，相等返回 0。
func compareDashboardMetric(metricA float64, metricB float64, a Summary, b Summary) int {
	if diff := cmp.Compare(metricB, metricA); diff != 0 {
		return diff
	}
	return compareSummaryIdentity(a, b)
}

// compareSummaryIdentity 按名称和 ID 为两个 Summary 提供稳定的比较结果。
// 先比较去除首尾空白后的名称，再比较 ID。
func compareSummaryIdentity(a Summary, b Summary) int {
	if diff := cmp.Compare(strings.TrimSpace(a.Name), strings.TrimSpace(b.Name)); diff != 0 {
		return diff
	}
	return cmp.Compare(a.ID, b.ID)
}

// dashboardAnomalyRank 返回容器异常候选的优先级。
// 当前仅将健康异常和状态异常视为 anomaly；正常资源活跃度由热点列表承载。
func dashboardAnomalyRank(item Summary) int {
	switch {
	case strings.EqualFold(item.Health, containerHealthUnhealthy):
		return containerDashboardRankHealth
	case isDashboardAbnormalState(item.State):
		return containerDashboardRankState
	default:
		return 0
	}
}

// isDashboardAbnormal 判断容器是否处于异常状态。
// 当容器健康状态为不健康，或者状态属于重启、退出、死亡时，返回 true。
func isDashboardAbnormal(item Summary) bool {
	return strings.EqualFold(item.Health, containerHealthUnhealthy) || isDashboardAbnormalState(item.State)
}

// isDashboardAbnormalState 判断容器状态是否属于异常状态。
//
// 当状态归一化后为 `restarting`、`exited` 或 `dead` 时，返回 `true`。
func isDashboardAbnormalState(state string) bool {
	switch normalizeContainerState(state) {
	case "restarting", "exited", "dead":
		return true
	default:
		return false
	}
}

// isDashboardRunningState 判断容器状态是否为运行中。
// 当归一化后的状态为 "running" 时返回 `true`，否则返回 `false`。
func isDashboardRunningState(state string) bool {
	return normalizeContainerState(state) == "running"
}

// dashboardSummaryCollectedAt 返回汇总数据的最新采集时间。
// 它会从各项资源的采集时间中选择最新的有效值；如果没有可用时间，则使用当前 UTC 时间。
// 返回 RFC3339 格式的时间字符串。
func dashboardSummaryCollectedAt(items []Summary) string {
	var latest time.Time
	for _, item := range items {
		parsed, ok := parseResourceCollectedAt(item.Resource.CollectedAt)
		if !ok {
			continue
		}
		if parsed.After(latest) {
			latest = parsed
		}
	}
	if latest.IsZero() {
		latest = time.Now().UTC()
	}
	return latest.Format(time.RFC3339)
}

// dashboardAnomalyReason 为容器异常项生成原因编码和标签。
// 当容器健康状态为不健康时，优先返回健康异常原因；否则按容器状态返回重启、退出或死亡原因。
// 其他情况返回未知原因，并使用状态文本作为标签，若状态为空则使用“Unknown”。
// @return 第一个值为原因编码，第二个值为原因标签。
func dashboardAnomalyReason(item Summary) (string, string) {
	switch {
	case strings.EqualFold(item.Health, containerHealthUnhealthy):
		return containerDashboardReasonHealthUnhealthy, "Unhealthy"
	case normalizeContainerState(item.State) == "restarting":
		return containerDashboardReasonStateRestarting, "Restarting"
	case normalizeContainerState(item.State) == "exited":
		return containerDashboardReasonStateExited, "Exited"
	case normalizeContainerState(item.State) == "dead":
		return containerDashboardReasonStateDead, "Dead"
	default:
		return containerDashboardReasonStateUnknown, firstNonEmpty(strings.TrimSpace(item.Status), "Unknown")
	}
}

// 该函数会裁剪字符串字段空白，规范化容器状态，并在健康状态无效时清空 Health。
func summaryFromStatsSnapshot(snapshot StatsSnapshot) Summary {
	health := strings.TrimSpace(snapshot.Health)
	if !isValidContainerHealth(health) {
		health = ""
	}
	return Summary{
		ID:           strings.TrimSpace(snapshot.ContainerID),
		ShortID:      strings.TrimSpace(snapshot.ShortID),
		Name:         strings.TrimSpace(snapshot.Name),
		Image:        strings.TrimSpace(snapshot.Image),
		Runtime:      strings.TrimSpace(snapshot.Runtime),
		State:        normalizeContainerState(snapshot.State),
		Status:       strings.TrimSpace(snapshot.Status),
		Health:       health,
		RestartCount: snapshot.RestartCount,
		Resource:     snapshot.Resource,
	}
}

// hasUsableDashboardResource 判断资源摘要是否包含可用于仪表盘展示的 CPU 或内存数据。
// 当 CPU 使用率或内存使用量大于 0 时返回 true。
func hasUsableDashboardResource(resource ResourceSummary) bool {
	return resourceCPUPercent(resource) > 0 || resourceMemoryUsage(resource) > 0
}

// resourceCPUPercent 返回资源配置中的 CPU 百分比，若未设置或为负值则返回 0。
func resourceCPUPercent(resource ResourceSummary) float64 {
	if resource.CPUPercent == nil || *resource.CPUPercent < 0 {
		return 0
	}
	return *resource.CPUPercent
}

// resourceMemoryUsage 返回资源使用内存字节数。
//
// 当 MemoryUsageBytes 为空或小于 0 时，返回 0。
func resourceMemoryUsage(resource ResourceSummary) float64 {
	if resource.MemoryUsageBytes == nil || *resource.MemoryUsageBytes < 0 {
		return 0
	}
	return float64(*resource.MemoryUsageBytes)
}

// dashboardRestartCount 返回容器的重启次数。
// 当重启次数未设置或小于 0 时，返回 0。
func dashboardRestartCount(item Summary) int {
	if item.RestartCount == nil || *item.RestartCount < 0 {
		return 0
	}
	return *item.RestartCount
}
