package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	containercontract "graft/server/modules/container/contract"
)

func (s *service) applyEnvironmentPolicy(ctx context.Context, detail Detail) Detail {
	policy := s.environmentDisplayPolicy(ctx)
	if policy == containercontract.ContainerEnvironmentPolicyPlain && !environmentPlainAccessAllowed(ctx) {
		policy = containercontract.ContainerEnvironmentPolicyMasked
	}
	detail.EnvironmentPolicy = policy.String()
	detail.EnvironmentMaskedCopyEnabled = s.maskedEnvironmentCopyEnabled(ctx)
	detail.Environment = applyEnvironmentPolicy(detail.Environment, environmentPolicyOptions{
		maskedCopyEnabled: policy == containercontract.ContainerEnvironmentPolicyMasked &&
			environmentPlainAccessAllowed(ctx) &&
			s.maskedEnvironmentCopyEnabled(ctx),
		policy: policy,
	})
	return detail
}

// withEnvironmentPlainAccess 将上下文标记为允许访问明文环境变量。
func withEnvironmentPlainAccess(ctx context.Context) context.Context {
	return context.WithValue(ctx, environmentPlainAccessContextKey{}, true)
}

// environmentPlainAccessAllowed 检查请求上下文是否允许查看明文环境变量。
func environmentPlainAccessAllowed(ctx context.Context) bool {
	allowed, _ := ctx.Value(environmentPlainAccessContextKey{}).(bool)
	return allowed
}

func (s *service) environmentDisplayPolicy(ctx context.Context) containercontract.EnvironmentPolicy {
	// System Config 是运行时策略的权威来源；读取失败时保留启动时的默认值，避免详情接口因配置中心暂时不可用而失败。
	fallback := defaultContainerEnvironmentPolicy
	if s != nil && s.environmentPolicy != "" {
		fallback = s.environmentPolicy
	}
	if s == nil || s.systemConfig == nil {
		return fallback
	}
	raw, err := s.systemConfig.ResolveDefaultConfig(
		ctx,
		containercontract.ContainerEnvironmentPolicyConfig.String(),
	)
	if err != nil {
		return fallback
	}
	var value string
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fallback
	}
	return normalizeEnvironmentPolicy(value)
}

type environmentPolicyOptions struct {
	policy            containercontract.EnvironmentPolicy
	maskedCopyEnabled bool
}

// applyEnvironmentPolicy applies environment display and masking policy to variables.
// Each variable is marked sensitive if its key matches known sensitive patterns. The
// returned payload always carries explicit display-state fields so downstream consumers
// applyEnvironmentPolicy modifies environment variables to enforce the specified display policy, controlling value visibility through masking, hiding, or plaintext modes.
func applyEnvironmentPolicy(environment []EnvironmentVariable, options environmentPolicyOptions) []EnvironmentVariable {
	if len(environment) == 0 {
		return nil
	}
	mapped := make([]EnvironmentVariable, 0, len(environment))
	for _, item := range environment {
		item.Sensitive = item.Sensitive || isSensitiveEnvironmentKey(item.Key)
		item.CopyValue = ""
		item.DisplayValue = item.Value
		item.ValueMasked = false
		item.ValueHidden = false
		switch options.policy {
		case containercontract.ContainerEnvironmentPolicyHidden:
			item.Value = ""
			item.DisplayValue = "[HIDDEN]"
			item.ValueHidden = true
			item.Masked = true
		case containercontract.ContainerEnvironmentPolicyPlain:
			item.Masked = false
		default:
			if item.Sensitive {
				if options.maskedCopyEnabled && strings.TrimSpace(item.Value) != "" {
					item.CopyValue = item.Value
				}
				item.Value = ""
				item.DisplayValue = maskedEnvironmentPlaceholder
				item.ValueMasked = true
				item.Masked = true
			} else {
				item.Masked = false
			}
		}
		mapped = append(mapped, item)
	}
	return mapped
}

// normalizeEnvironmentPolicy 将字符串规范化为环境策略类型。
// 识别 Hidden 和 Plain 策略；若输入不匹配任何已知策略，则默认返回 Masked。
func normalizeEnvironmentPolicy(value string) containercontract.EnvironmentPolicy {
	switch containercontract.EnvironmentPolicy(strings.ToLower(strings.TrimSpace(value))) {
	case containercontract.ContainerEnvironmentPolicyHidden:
		return containercontract.ContainerEnvironmentPolicyHidden
	case containercontract.ContainerEnvironmentPolicyPlain:
		return containercontract.ContainerEnvironmentPolicyPlain
	default:
		return containercontract.ContainerEnvironmentPolicyMasked
	}
}

// normalizeOrchestratorActionLevel normalizes a string to an orchestrator action level,
// returning Readonly or Allow if matched, or Warn as the default.
func normalizeOrchestratorActionLevel(value string) containercontract.OrchestratorActionLevel {
	switch containercontract.OrchestratorActionLevel(strings.ToLower(strings.TrimSpace(value))) {
	case containercontract.ContainerOrchestratorActionLevelReadonly:
		return containercontract.ContainerOrchestratorActionLevelReadonly
	case containercontract.ContainerOrchestratorActionLevelAllow:
		return containercontract.ContainerOrchestratorActionLevelAllow
	default:
		return containercontract.ContainerOrchestratorActionLevelWarn
	}
}

// normalizedOrchestratorInfo 标准化编排器信息，补全作用域信息并设置管理状态、置信度和告警列表。
func normalizedOrchestratorInfo(info OrchestratorInfo) OrchestratorInfo {
	info.Type = effectiveOrchestratorTypeFromValue(info.Type)
	info.Managed = info.Type != containerOrchestratorStandalone
	info = normalizeOrchestratorIdentityFields(info)
	info = normalizeOrchestratorScopeKinds(info)
	info = normalizeOrchestratorScopeValues(info)
	info = normalizeOrchestratorConfidence(info)
	if info.Warnings == nil {
		info.Warnings = []string{}
	}
	return info
}

// normalizeOrchestratorIdentityFields 标准化编排器身份和作用域字段。
//
// 该函数会修剪字符串字段的空白、规范化作用域类型，并清理配置文件列表中的空值和重复项。
func normalizeOrchestratorIdentityFields(info OrchestratorInfo) OrchestratorInfo {
	info.GroupScopeKind = normalizeContainerSourceScopeKind(info.GroupScopeKind)
	info.MemberScopeKind = normalizeContainerSourceScopeKind(info.MemberScopeKind)
	info.Project = strings.TrimSpace(info.Project)
	info.Service = strings.TrimSpace(info.Service)
	info.Stack = strings.TrimSpace(info.Stack)
	info.Namespace = strings.TrimSpace(info.Namespace)
	info.Pod = strings.TrimSpace(info.Pod)
	info.Task = strings.TrimSpace(info.Task)
	info.WorkingDir = strings.TrimSpace(info.WorkingDir)
	info.ConfigFiles = normalizedStringSlice(info.ConfigFiles)
	info.GroupValue = strings.TrimSpace(info.GroupValue)
	info.MemberValue = strings.TrimSpace(info.MemberValue)
	info.GroupDisplayName = strings.TrimSpace(info.GroupDisplayName)
	info.MemberDisplayName = strings.TrimSpace(info.MemberDisplayName)
	return info
}

// normalizeOrchestratorScopeKinds infers missing group and member scope kinds from the corresponding orchestrator identity fields.
func normalizeOrchestratorScopeKinds(info OrchestratorInfo) OrchestratorInfo {
	if info.GroupScopeKind == "" {
		switch {
		case info.Project != "":
			info.GroupScopeKind = composeProjectScopeKind
		case info.Stack != "":
			info.GroupScopeKind = swarmStackScopeKind
		case info.Namespace != "":
			info.GroupScopeKind = kubernetesNamespaceScopeKind
		}
	}
	if info.MemberScopeKind == "" {
		switch {
		case info.Service != "":
			info.MemberScopeKind = composeServiceScopeKind
		case info.Task != "":
			info.MemberScopeKind = swarmTaskScopeKind
		case info.Pod != "":
			info.MemberScopeKind = kubernetesPodScopeKind
		}
	}
	return info
}

// normalizeOrchestratorScopeValues derives missing group and member scope values and fills their display names.
func normalizeOrchestratorScopeValues(info OrchestratorInfo) OrchestratorInfo {
	info.GroupValue = normalizedGroupScopeValue(info)
	info.MemberValue = normalizedMemberScopeValue(info)
	if info.GroupDisplayName == "" {
		info.GroupDisplayName = info.GroupValue
	}
	if info.MemberDisplayName == "" {
		info.MemberDisplayName = info.MemberValue
	}
	return info
}

// normalizedGroupScopeValue returns the group scope value, deriving it from the corresponding identity field when necessary.
func normalizedGroupScopeValue(info OrchestratorInfo) string {
	if info.GroupValue != "" {
		return info.GroupValue
	}
	switch info.GroupScopeKind {
	case composeProjectScopeKind:
		return info.Project
	case swarmStackScopeKind:
		return info.Stack
	case kubernetesNamespaceScopeKind:
		return info.Namespace
	default:
		return ""
	}
}

// normalizedMemberScopeValue returns the member scope value, deriving it from the member scope kind when necessary.
func normalizedMemberScopeValue(info OrchestratorInfo) string {
	if info.MemberValue != "" {
		return info.MemberValue
	}
	switch info.MemberScopeKind {
	case composeServiceScopeKind:
		return info.Service
	case swarmTaskScopeKind:
		return info.Task
	case kubernetesPodScopeKind:
		return info.Pod
	default:
		return ""
	}
}

// normalizeOrchestratorConfidence ensures that orchestrator information has a confidence level.
// It preserves an existing confidence value, or assigns medium confidence to managed
// orchestrators and high confidence to standalone orchestrators.
func normalizeOrchestratorConfidence(info OrchestratorInfo) OrchestratorInfo {
	if strings.TrimSpace(info.Confidence) != "" {
		return info
	}
	if info.Managed {
		info.Confidence = orchestratorConfidenceMedium
		return info
	}
	info.Confidence = orchestratorConfidenceHigh
	return info
}

// EffectiveOrchestratorType returns the normalized orchestrator type from the container summary.
func effectiveOrchestratorType(item Summary) string {
	return effectiveOrchestratorTypeFromValue(item.Orchestrator.Type)
}

// effectiveOrchestratorTypeFromValue returns the normalized orchestrator type for the given value, defaulting to standalone if the value is invalid.
func effectiveOrchestratorTypeFromValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if isValidContainerOrchestrator(value) {
		return value
	}
	return containerOrchestratorStandalone
}

// orchestratorWarningsFor returns a deduplicated slice of warnings for an orchestrator, combining base warnings with those derived from managed status and action level constraints.
func orchestratorWarningsFor(
	info OrchestratorInfo,
	level containercontract.OrchestratorActionLevel,
) []string {
	const extraOrchestratorWarnings = 2

	seen := map[string]struct{}{}
	warnings := make([]string, 0, len(info.Warnings)+extraOrchestratorWarnings)
	appendWarning := func(code string) {
		code = strings.TrimSpace(code)
		if code == "" {
			return
		}
		if _, ok := seen[code]; ok {
			return
		}
		seen[code] = struct{}{}
		warnings = append(warnings, code)
	}
	for _, code := range info.Warnings {
		appendWarning(code)
	}
	if info.Managed {
		appendWarning(orchestratorWarningManagedActionRisk)
	}
	switch level {
	case containercontract.ContainerOrchestratorActionLevelReadonly:
		appendWarning(orchestratorWarningReadonly)
		appendWarning(orchestratorWarningBatchBlocked)
	case containercontract.ContainerOrchestratorActionLevelWarn:
		appendWarning(orchestratorWarningBatchBlocked)
	}
	return warnings
}

func normalizedStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// isSensitiveEnvironmentKey 判断环境变量键是否表示敏感值。
func isSensitiveEnvironmentKey(key string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(key))
	for _, marker := range sensitiveEnvironmentKeyMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

var sensitiveEnvironmentKeyMarkers = []string{
	"PASSWORD",
	"PASSWD",
	"TOKEN",
	"SECRET",
	"KEY",
	"AUTH",
	"CREDENTIAL",
	"PRIVATE",
	"CERT",
	"COOKIE",
	"SESSION",
}

func (s *service) normalizeLogQuery(ctx context.Context, query LogQuery) (LogQuery, error) {
	// 日志尾部数量受 System Config 与模块硬上限共同约束，未指定输出流时默认同时返回 stdout 和 stderr。
	defaultTail, maxTail := s.effectiveLogTailBounds(ctx)
	if query.Tail == 0 {
		query.Tail = defaultTail
	}
	if query.Tail < 0 || query.Tail > maxTail || query.Tail > defaultContainerLogsMaxTail {
		return LogQuery{}, errLogsTooLarge
	}
	if !query.Stdout && !query.Stderr {
		query.Stdout = true
		query.Stderr = true
	}
	if query.Since != "" {
		if _, err := parseLogSince(query.Since); err != nil {
			return LogQuery{}, fmt.Errorf("%w: %w", errInvalidLogQuery, err)
		}
	}
	return query, nil
}

// filterContainerSummaries returns the summaries that match the query criteria.
func filterContainerSummaries(items []Summary, query ListQuery) []Summary {
	filtered := make([]Summary, 0, len(items))
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	for _, item := range items {
		if !summaryMatchesListQuery(item, query, keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func normalizeDockerImageListQuery(query DockerImageListQuery) (DockerImageListQuery, error) {
	if query.Limit == 0 {
		query.Limit = defaultContainerListLimit
	}
	if query.Limit < 1 || query.Limit > maxContainerListLimit || query.Offset < 0 {
		return DockerImageListQuery{}, errInvalidListQuery
	}
	query.Keyword = strings.TrimSpace(query.Keyword)
	if len(query.Keyword) > containerListKeywordMaxLength {
		return DockerImageListQuery{}, errInvalidListQuery
	}
	return query, nil
}

func filterDockerImages(items []DockerImage, keyword string) []DockerImage {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return items
	}
	filtered := make([]DockerImage, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.ID), keyword) ||
			dockerImageFieldContains(item.RepositoryTags, keyword) || dockerImageFieldContains(item.RepositoryDigests, keyword) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterUnusedDockerImages(items []DockerImage) []DockerImage {
	filtered := make([]DockerImage, 0, len(items))
	for _, item := range items {
		if len(item.ContainerReferences) == 0 {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func dockerImageFieldContains(values []string, keyword string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func pageDockerImages(items []DockerImage, offset, limit int) []DockerImage {
	if offset >= len(items) {
		return []DockerImage{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

// summaryMatchesListQuery 确定容器摘要是否与列表查询的所有过滤条件相匹配。
func summaryMatchesListQuery(item Summary, query ListQuery, keyword string) bool {
	return summaryMatchesState(item, query.State) &&
		summaryMatchesHealth(item, query.Health) &&
		summaryMatchesOrchestrator(item, firstNonEmpty(query.DeploymentType, query.Orchestrator)) &&
		summaryMatchesSourceScopeFilter(item, query.SourceScopeKind, query.SourceScope) &&
		summaryMatchesKeywordFilter(item, keyword)
}

// summaryMatchesState 检查容器摘要的状态是否与给定的状态匹配，空字符串表示接受任何状态。
func summaryMatchesState(item Summary, state string) bool {
	return state == "" || item.State == state
}

// summaryMatchesHealth reports whether a container summary matches the given health filter.
func summaryMatchesHealth(item Summary, health string) bool {
	return health == "" || effectiveHealth(item) == health
}

// summaryMatchesOrchestrator reports whether a container summary matches the
// given orchestrator filter.
func summaryMatchesOrchestrator(item Summary, orchestrator string) bool {
	return orchestrator == "" || effectiveOrchestratorType(item) == orchestrator
}

// summaryMatchesSourceScopeFilter 检查容器摘要是否与源作用域过滤条件匹配。
// 当 scopeKind 为空时返回 true，表示不应用该过滤；否则检查摘要是否与指定作用域相匹配。
func summaryMatchesSourceScopeFilter(item Summary, scopeKind string, scope string) bool {
	return scopeKind == "" || summaryMatchesSourceScope(item, scopeKind, scope)
}

// SummaryMatchesKeywordFilter reports whether a Summary matches the keyword filter, where an empty keyword matches all summaries.
func summaryMatchesKeywordFilter(item Summary, keyword string) bool {
	return keyword == "" || summaryMatchesKeyword(item, keyword)
}

// pageContainerSummaries 根据查询条件对容器摘要进行分页。
// 返回从指定偏移开始、不超过指定限制数量的摘要切片，若偏移超过总项数则返回空切片。
func pageContainerSummaries(items []Summary, query ListQuery) []Summary {
	if query.Offset >= len(items) {
		return []Summary{}
	}
	end := query.Offset + query.Limit
	if end > len(items) {
		end = len(items)
	}
	return items[query.Offset:end]
}

// summarizeContainers computes aggregate counts of containers grouped by state and health status.
func summarizeContainers(items []Summary) ListSummary {
	summary := ListSummary{Total: len(items)}
	for _, item := range items {
		switch item.State {
		case "running":
			summary.Running++
		case "created", "exited", "paused", "restarting":
			summary.Stopped++
		case "dead", "unknown", "removing":
			summary.Error++
		}
		switch effectiveHealth(item) {
		case containerHealthHealthy:
			summary.Healthy++
		case containerHealthUnhealthy:
			summary.Unhealthy++
		default:
			summary.HealthUnavailable++
		}
	}
	return summary
}

// applyActionAvailability 根据编排器策略和容器状态对容器摘要应用动作可用性限制，禁用危险操作被禁用或编排器操作级别为只读时的所有可变动作。
func applyActionAvailability(items []Summary, policy effectiveActionPolicy) []Summary {
	adjusted := make([]Summary, 0, len(items))
	for _, item := range items {
		item.CanRemove = canRemoveState(item.State)
		item.Orchestrator = policy.decorate(item.Orchestrator)
		if !policy.dangerousAllowed || item.Orchestrator.ActionLevel == containercontract.ContainerOrchestratorActionLevelReadonly.String() {
			item.CanStart = false
			item.CanStop = false
			item.CanRestart = false
			item.CanRemove = false
		}
		adjusted = append(adjusted, item)
	}
	return adjusted
}

// summaryMatchesKeyword reports whether the keyword matches any of the container summary's searchable fields.
func summaryMatchesKeyword(item Summary, keyword string) bool {
	values := []string{
		item.ID,
		item.ShortID,
		item.Name,
		item.Image,
		item.ImageID,
		item.Status,
		item.State,
		item.Runtime,
		item.RestartPolicy,
		item.PrimaryIP,
		item.NetworkSummary,
		item.ComposeProject,
		item.ComposeService,
	}
	values = append(values, item.Names...)
	for _, port := range item.Ports {
		values = append(values, port.IP, strconv.Itoa(port.PrivatePort), port.Type)
		if port.PublicPort != nil {
			values = append(values, strconv.Itoa(*port.PublicPort))
		}
	}
	for _, network := range item.Networks {
		values = append(values, network.Name, network.NetworkID, network.EndpointID, network.Gateway, network.IPAddress, network.MacAddress)
	}
	for key, value := range item.Labels {
		values = append(values, key, value)
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

// NormalizeContainerSourceScopeKind 规范化容器源作用域类型值，转换为小写并去除空白。返回规范化后的值（如果为支持的作用域类型）或空字符串。
func normalizeContainerSourceScopeKind(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if !isValidContainerSourceScopeKind(value) {
		return ""
	}
	return value
}

// sourceScopeKindCompatibleWithOrchestrator reports whether a source scope kind is compatible with an orchestrator type.
func sourceScopeKindCompatibleWithOrchestrator(orchestrator string, scopeKind string) bool {
	scopeKind = normalizeContainerSourceScopeKind(scopeKind)
	if scopeKind == "" {
		return false
	}
	switch scopeKind {
	case composeProjectScopeKind, composeServiceScopeKind:
		return orchestrator == "" || orchestrator == containerOrchestratorCompose
	case swarmStackScopeKind, swarmTaskScopeKind:
		return orchestrator == "" || orchestrator == containerOrchestratorSwarm
	case kubernetesNamespaceScopeKind, kubernetesPodScopeKind:
		return orchestrator == "" || orchestrator == containerOrchestratorKubernetes
	default:
		return false
	}
}

// summaryMatchesSourceScope 判断容器摘要是否与指定的源作用域类型和值相匹配。
// summaryMatchesSourceScope 判断容器是否匹配指定的源作用域类型和值，并以不区分大小写的方式比较作用域值；作用域类型必须与容器的编排器类型兼容。
func summaryMatchesSourceScope(item Summary, scopeKind string, scope string) bool {
	scopeKind = normalizeContainerSourceScopeKind(scopeKind)
	scope = strings.TrimSpace(scope)
	if scopeKind == "" || scope == "" {
		return false
	}
	info := normalizedOrchestratorInfo(item.Orchestrator)
	if info.Type != "" && !sourceScopeKindCompatibleWithOrchestrator(info.Type, scopeKind) {
		return false
	}
	for _, candidate := range sourceScopeCandidates(info, scopeKind) {
		if strings.EqualFold(candidate, scope) {
			return true
		}
	}
	return false
}

// SourceScopeCandidates returns candidate values from orchestrator information for matching against the given scope kind.
func sourceScopeCandidates(info OrchestratorInfo, scopeKind string) []string {
	switch scopeKind {
	case composeProjectScopeKind, swarmStackScopeKind, kubernetesNamespaceScopeKind:
		return []string{info.GroupValue}
	case composeServiceScopeKind, swarmTaskScopeKind, kubernetesPodScopeKind:
		return []string{info.MemberValue}
	default:
		return nil
	}
}

// effectiveHealth 返回项目的有效健康状态，若未设定则默认为不可用。
func effectiveHealth(item Summary) string {
	if item.Health == "" {
		return containerHealthUnavailable
	}
	return item.Health
}

func parseLogSince(raw string) (string, error) {
	// since 同时接受 RFC3339 时间和相对时长；相对时长转换为 UTC Unix 秒供 Docker API 使用。
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
		return timestamp.UTC().Format(time.RFC3339), nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return "", fmt.Errorf("invalid since value")
	}
	return strconv.FormatInt(time.Now().UTC().Add(-duration).Unix(), 10), nil
}
