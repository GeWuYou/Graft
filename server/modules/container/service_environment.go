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

// normalizedOrchestratorInfo normalizes the provided orchestrator information by validating the type, deriving managed status from type, normalizing scope kinds, trimming whitespace from string fields, applying default confidence based on managed status, and ensuring the warnings slice is initialized.
func normalizedOrchestratorInfo(info OrchestratorInfo) OrchestratorInfo {
	info.Type = effectiveOrchestratorTypeFromValue(info.Type)
	info.Managed = info.Type != containerOrchestratorStandalone
	info.GroupScopeKind = normalizeContainerSourceScopeKind(info.GroupScopeKind)
	info.MemberScopeKind = normalizeContainerSourceScopeKind(info.MemberScopeKind)
	info.GroupValue = strings.TrimSpace(info.GroupValue)
	info.MemberValue = strings.TrimSpace(info.MemberValue)
	info.GroupDisplayName = strings.TrimSpace(info.GroupDisplayName)
	info.MemberDisplayName = strings.TrimSpace(info.MemberDisplayName)
	if strings.TrimSpace(info.Confidence) == "" {
		if info.Managed {
			info.Confidence = orchestratorConfidenceMedium
		} else {
			info.Confidence = orchestratorConfidenceHigh
		}
	}
	if info.Warnings == nil {
		info.Warnings = []string{}
	}
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
			return LogQuery{}, fmt.Errorf("%w: %v", errInvalidLogQuery, err)
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

// summaryMatchesListQuery 确定容器摘要是否与列表查询的所有过滤条件相匹配。
func summaryMatchesListQuery(item Summary, query ListQuery, keyword string) bool {
	return summaryMatchesState(item, query.State) &&
		summaryMatchesHealth(item, query.Health) &&
		summaryMatchesOrchestrator(item, query.Orchestrator) &&
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
// 比较采用不区分大小写的方式，源作用域类型必须与容器的编排器类型兼容。
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
	for _, candidate := range sourceScopeCandidates(item, info, scopeKind) {
		if strings.EqualFold(candidate, scope) {
			return true
		}
	}
	return false
}

// SourceScopeCandidates returns candidate values from the container summary and orchestrator information for matching against the given scope kind.
func sourceScopeCandidates(item Summary, info OrchestratorInfo, scopeKind string) []string {
	switch scopeKind {
	case composeProjectScopeKind:
		return []string{item.ComposeProject, info.GroupValue}
	case composeServiceScopeKind:
		return []string{item.ComposeService, info.MemberValue}
	case swarmStackScopeKind, kubernetesNamespaceScopeKind:
		return []string{info.GroupValue}
	case swarmTaskScopeKind, kubernetesPodScopeKind:
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
