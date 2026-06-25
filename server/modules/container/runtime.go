package container

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"graft/server/modules/container/terminal"
)

const (
	runtimeNameDocker = "docker"

	containerActionStart   = "start"
	containerActionStop    = "stop"
	containerActionRestart = "restart"
	containerActionRemove  = "remove"

	actionResultCompleted = "completed"
	actionResultUnchanged = "unchanged"
	actionStatusRemoved   = "removed"

	defaultContainerListLimit     = 20
	maxContainerListLimit         = 100
	maxContainerBatchActionIDs    = 100
	containerListKeywordMaxLength = 128
	containerShortIDLength        = 12

	containerHealthHealthy     = "healthy"
	containerHealthUnhealthy   = "unhealthy"
	containerHealthStarting    = "starting"
	containerHealthNone        = "none"
	containerHealthUnavailable = "unavailable"

	containerOrchestratorStandalone = "standalone"
	containerOrchestratorCompose    = "compose"
	containerOrchestratorSwarm      = "swarm"
	containerOrchestratorKubernetes = "kubernetes"
	containerOrchestratorUnknown    = "unknown"

	orchestratorConfidenceHigh   = "high"
	orchestratorConfidenceMedium = "medium"
	orchestratorConfidenceLow    = "low"

	orchestratorWarningManagedActionRisk = "managed_action_risk"
	orchestratorWarningBatchBlocked      = "batch_action_blocked"
	orchestratorWarningReadonly          = "readonly"

	recommendedActionOpenController = "open_controller"

	containerStatsNotCollectedReason          = "stats_not_collected"
	containerStatsIncompleteReason            = "stats_incomplete"
	containerStatsTimeoutReason               = "stats_timeout"
	containerStatsUnavailableReason           = "stats_unavailable"
	containerMountUsageUnsupportedReason      = "mount_usage_unsupported"
	containerMountUsageStatusNotMeasured      = "not_measured"
	containerMountUsageStatusMeasured         = "measured"
	containerMountUsageStatusUnsupported      = "unsupported"
	containerMountUsageStatusPermissionDenied = "permission_denied"
	containerMountUsageStatusNotFound         = "not_found"
	containerMountUsageStatusTimeout          = "timeout"
	containerMountUsageStatusError            = "error"

	composeProjectLabel = "com.docker.compose.project"
	composeServiceLabel = "com.docker.compose.service"

	composeProjectScopeKind      = "compose_project"
	composeServiceScopeKind      = "compose_service"
	swarmStackScopeKind          = "swarm_stack"
	swarmTaskScopeKind           = "swarm_task"
	kubernetesNamespaceScopeKind = "kubernetes_namespace"
	kubernetesPodScopeKind       = "kubernetes_pod"
)

var mountIDPattern = regexp.MustCompile(`^m_[A-Za-z0-9_-]{1,62}$`)

var validContainerStates = []string{"created", "running", "paused", "restarting", "removing", "exited", "dead", "unknown"}
var validContainerOrchestrators = []string{
	containerOrchestratorStandalone,
	containerOrchestratorCompose,
	containerOrchestratorSwarm,
	containerOrchestratorKubernetes,
	containerOrchestratorUnknown,
}

var validContainerSourceScopeKinds = []string{
	composeProjectScopeKind,
	composeServiceScopeKind,
	swarmStackScopeKind,
	swarmTaskScopeKind,
	kubernetesNamespaceScopeKind,
	kubernetesPodScopeKind,
}

var (
	errRuntimeDisabled             = errors.New("container runtime disabled")
	errRuntimeSocketMissing        = errors.New("container runtime socket missing")
	errRuntimePermissionDenied     = errors.New("container runtime permission denied")
	errRuntimeDaemonUnavailable    = errors.New("container runtime daemon unavailable")
	errContainerNotFound           = errors.New("container not found")
	errInvalidRef                  = errors.New("invalid container reference")
	errInvalidListQuery            = errors.New("invalid container list query")
	errInvalidBatchAction          = errors.New("invalid container batch action")
	errInvalidContainerState       = errors.New("invalid container state")
	errLogsTooLarge                = errors.New("container logs tail exceeds limit")
	errContainerRuntimeTimeout     = errors.New("container runtime timeout")
	errDangerousActionsDisabled    = errors.New("dangerous container actions disabled")
	errUnsupportedContainerRuntime = errors.New("unsupported container runtime")
	errMountUsageUnsupported       = errors.New("container mount usage unsupported")
	errContainerMountNotFound      = errors.New("container mount not found")
	errShellDisabled               = errors.New("container shell disabled")
	errShellForbidden              = errors.New("container shell forbidden")
	errShellTicketInvalid          = errors.New("container shell ticket invalid")
	errShellTicketExpired          = errors.New("container shell ticket expired")
	errShellTicketUsed             = errors.New("container shell ticket used")
	errShellOriginDenied           = errors.New("container shell origin denied")
	errContainerNotRunning         = errors.New("container not running")
	errShellCommandNotFound        = errors.New("container shell command not found")
	errShellInvalidSize            = errors.New("container shell invalid size")
	errShellSessionFailed          = errors.New("container shell session failed")
)

// Runtime is the module-owned boundary between API/service code and a concrete container runtime adapter.
type Runtime interface {
	Info(ctx context.Context) (RuntimeInfo, error)
	List(ctx context.Context, query ListQuery) ([]Summary, error)
	Detail(ctx context.Context, id Ref) (Detail, error)
	Mounts(ctx context.Context, id Ref) ([]Mount, error)
	MountUsage(ctx context.Context, id Ref, mountID string) (MountUsage, error)
	Logs(ctx context.Context, id Ref, query LogQuery) (Logs, error)
	Shell(ctx context.Context, ref Ref, command string) (terminal.Session, error)
	Start(ctx context.Context, id Ref) (ActionResult, error)
	Stop(ctx context.Context, id Ref) (ActionResult, error)
	Restart(ctx context.Context, id Ref) (ActionResult, error)
	Remove(ctx context.Context, id Ref, options RemoveOptions) (ActionResult, error)
	Close() error
}

// StatsCollectorRuntime exposes background stats snapshot collection for runtimes
// that support collector-driven sampling and publish flows.
type StatsCollectorRuntime interface {
	Runtime
	CollectStatsSnapshots(ctx context.Context) ([]StatsSnapshot, error)
}

// StatsSnapshot is one collector-produced resource snapshot ready for publish.
type StatsSnapshot struct {
	ContainerID  string
	Name         string
	ShortID      string
	Image        string
	Runtime      string
	State        string
	Status       string
	Health       string
	RestartCount *int
	Resource     ResourceSummary
	CollectedAt  time.Time
}

// Ref is a validated Docker-compatible container id or name.
type Ref struct {
	Value string
}

// ListQuery describes bounded list pagination and low-cost runtime filters.
type ListQuery struct {
	Limit           int
	Offset          int
	Keyword         string
	State           string
	Health          string
	Orchestrator    string
	SourceScopeKind string
	SourceScope     string
}

// ListResult is the service-owned list response model.
type ListResult struct {
	Runtime RuntimeInfo
	Items   []Summary
	Total   int
	Limit   int
	Offset  int
	Summary ListSummary
}

type dashboardSummaryQuery struct{}

// LogQuery describes bounded container log retrieval options.
type LogQuery struct {
	Tail       int
	Since      string
	Timestamps bool
	Stdout     bool
	Stderr     bool
}

// RuntimeInfo is sanitized runtime metadata exposed by the container API.
type RuntimeInfo struct {
	Runtime           string
	Status            string
	Endpoint          string
	APIVersion        string
	ServerVersion     string
	OperatingSystem   string
	Architecture      string
	ContainersTotal   int
	ContainersRunning int
}

// ListSummary carries aggregate counts across the filtered list.
type ListSummary struct {
	Total             int
	Running           int
	Stopped           int
	Error             int
	Healthy           int
	Unhealthy         int
	HealthUnavailable int
}

// ResourceSummary is the container module's latest-known stats projection.
// HTTP consumers receive it as a seed snapshot; canonical stats authority remains
// the backend collector/cache/topic chain rather than page-local frontend state.
type ResourceSummary struct {
	Available                  bool
	UnavailableReason          string
	StatsAvailable             bool
	StatsErrorKey              string
	StatsErrorMessage          string
	CollectedAt                string
	CPUPercent                 *float64
	OnlineCPUs                 *int64
	SystemCPUUsage             *int64
	TotalCPUUsage              *int64
	CPUUsageInUsermode         *int64
	CPUUsageInKernelmode       *int64
	ThrottlingPeriods          *int64
	ThrottlingThrottledPeriods *int64
	ThrottlingThrottledTime    *int64
	MemoryUsageBytes           *int64
	MemoryLimitBytes           *int64
	MemoryPercent              *float64
	MemoryCache                *int64
	MemoryRSS                  *int64
	MemoryActiveFile           *int64
	MemoryInactiveFile         *int64
	MemoryPgfault              *int64
	MemoryPgmajfault           *int64
	RxBytes                    *int64
	TxBytes                    *int64
	RxPackets                  *int64
	TxPackets                  *int64
	RxErrors                   *int64
	TxErrors                   *int64
	RxDropped                  *int64
	TxDropped                  *int64
	PIDsCurrent                *int64
	PIDsLimit                  *int64
}

// Summary is a sanitized row for container list responses.
type Summary struct {
	ID             string
	ShortID        string
	Name           string
	Names          []string
	Image          string
	ImageID        string
	Labels         map[string]string
	Ports          []Port
	PrimaryIP      string
	Networks       []Network
	NetworkSummary string
	Resource       ResourceSummary
	RestartCount   *int
	RestartPolicy  string
	Runtime        string
	CreatedAt      string
	StartedAt      string
	State          string
	Status         string
	Health         string
	ComposeProject string
	ComposeService string
	Orchestrator   OrchestratorInfo
	CanStart       bool
	CanStop        bool
	CanRestart     bool
	CanRemove      bool
}

// Detail is a sanitized container inspect view.
type Detail struct {
	Summary
	Command                      []string
	Entrypoint                   []string
	Environment                  []EnvironmentVariable
	EnvironmentPolicy            string
	EnvironmentMaskedCopyEnabled bool
	Healthcheck                  *Healthcheck
	LastExitCode                 *int
	Mounts                       []Mount
	Networks                     []Network
	OOMKilled                    *bool
	RuntimeInfo                  RuntimeInfo
	InspectUpdatedAt             string
	WorkingDir                   string
}

// Healthcheck describes Docker healthcheck diagnostics from container inspect.
type Healthcheck struct {
	Configured     bool
	Status         string
	Command        []string
	ExitCode       *int
	Output         string
	CheckedAt      string
	FailingStreak  *int
	FailureMessage string
}

// Port describes one exposed or published container port.
type Port struct {
	IP          string
	PrivatePort int
	PublicPort  *int
	Type        string
}

// Mount describes one mounted path without exposing raw inspect payloads.
type Mount struct {
	ID          string
	Type        string
	Name        string
	Source      string
	Destination string
	Mode        string
	ReadOnly    bool
	Usage       *MountUsage
}

// MountUsage describes filesystem usage for one inspect-derived mount source.
type MountUsage struct {
	MountID     string
	ContainerID string
	Type        string
	Name        string
	Source      string
	Destination string
	SizeBytes   int64
	SizeDisplay string
	Status      string
	Message     string
	SharedHint  string
	Cached      bool
	MeasuredAt  string
}

// EnvironmentVariable describes one container environment entry after policy application.
type EnvironmentVariable struct {
	Key          string
	Value        string
	CopyValue    string
	DisplayValue string
	ValueMasked  bool
	ValueHidden  bool
	Masked       bool
	Sensitive    bool
	Source       string
}

// Network describes one network attachment.
type Network struct {
	Name       string
	NetworkID  string
	EndpointID string
	Gateway    string
	IPAddress  string
	MacAddress string
}

// OrchestratorInfo describes the resolved container source and policy semantics.
type OrchestratorInfo struct {
	Type               string
	Managed            bool
	DisplayName        string
	Confidence         string
	GroupScopeKind     string
	GroupDisplayName   string
	GroupValue         string
	MemberScopeKind    string
	MemberDisplayName  string
	MemberValue        string
	Project            string
	Service            string
	Stack              string
	Namespace          string
	Pod                string
	Container          string
	Task               string
	WorkingDir         string
	ConfigFiles        []string
	Warnings           []string
	RecommendedAction  string
	ActionLevel        string
	BatchActionAllowed bool
}

// Logs contains bounded log lines and the effective log options.
type Logs struct {
	ID         string
	Name       string
	Runtime    string
	Lines      []string
	Tail       int
	Since      string
	Timestamps bool
	Stdout     bool
	Stderr     bool
	Truncated  bool
}

// ActionResult describes a start, stop, or restart result for audit and API responses.
type ActionResult struct {
	ID           string
	Name         string
	Image        string
	Action       string
	Result       string
	Runtime      string
	StatusBefore string
	StatusAfter  string
	MessageKey   string
	Message      string
	Orchestrator OrchestratorInfo
}

// RemoveOptions describes guarded remove behavior passed to runtime adapters.
type RemoveOptions struct {
	Force bool
}

// ActionOptions describes service-layer action behavior shared by single and batch actions.
type ActionOptions struct {
	Force bool
}

// BatchActionCommand describes a bounded batch action request.
type BatchActionCommand struct {
	Action string
	IDs    []string
	Force  bool
}

// BatchActionResult aggregates per-container action outcomes without hiding partial failures.
type BatchActionResult struct {
	Action       string
	Total        int
	SuccessCount int
	FailedCount  int
	MessageKey   string
	Message      string
	RequestID    string
	Items        []BatchActionItem
}

// BatchActionItem carries one container action outcome.
type BatchActionItem struct {
	ID         string
	Name       string
	Action     string
	Success    bool
	ErrorCode  string
	MessageKey string
	Message    string
	Result     ActionResult
}

// parseRef validates and normalizes a container reference from a raw string. It returns a Ref containing the normalized reference on success, or an error if the input is invalid.
func parseRef(raw string) (Ref, error) {
	unescaped, err := url.PathUnescape(raw)
	if err != nil {
		return Ref{}, errInvalidRef
	}
	value := strings.TrimSpace(unescaped)
	if value == "" || strings.Contains(value, "/") {
		return Ref{}, errInvalidRef
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return Ref{}, errInvalidRef
		}
	}
	return Ref{Value: value}, nil
}

// normalizeListQuery 对容器列表查询应用验证约束。
// 验证分页边界、关键词长度、枚举字段值和源作用域兼容性。
// 如果任何验证失败，返回错误；否则返回规范化的查询。
func normalizeListQuery(query ListQuery) (ListQuery, error) {
	if err := normalizeListPagination(&query); err != nil {
		return ListQuery{}, err
	}
	if err := normalizeListKeyword(&query); err != nil {
		return ListQuery{}, err
	}
	if err := normalizeListEnums(&query); err != nil {
		return ListQuery{}, err
	}
	if err := normalizeListSourceScope(&query); err != nil {
		return ListQuery{}, err
	}
	return query, nil
}

// normalizeListPagination 验证列表查询的分页参数是否有效，并在未指定时应用默认的限制数。如果查询为 nil、限制值超出允许范围或偏移量为负数，返回 errInvalidListQuery。
func normalizeListPagination(query *ListQuery) error {
	if query == nil {
		return errInvalidListQuery
	}
	if query.Limit == 0 {
		query.Limit = defaultContainerListLimit
	}
	if query.Limit < 1 || query.Limit > maxContainerListLimit {
		return errInvalidListQuery
	}
	if query.Offset < 0 {
		return errInvalidListQuery
	}
	return nil
}

// normalizeListKeyword validates the keyword field, ensuring it does not exceed the maximum length after trimming whitespace.
func normalizeListKeyword(query *ListQuery) error {
	if query == nil {
		return errInvalidListQuery
	}
	query.Keyword = strings.TrimSpace(query.Keyword)
	if len(query.Keyword) > containerListKeywordMaxLength {
		return errInvalidListQuery
	}
	return nil
}

// normalizeListEnums normalizes and validates enumerated filter fields in a list query.
func normalizeListEnums(query *ListQuery) error {
	if query == nil {
		return errInvalidListQuery
	}
	var err error
	query.State, err = normalizeOptionalListEnum(query.State, isValidContainerState)
	if err != nil {
		return err
	}
	query.Health, err = normalizeOptionalListEnum(query.Health, isValidContainerHealth)
	if err != nil {
		return err
	}
	query.Orchestrator, err = normalizeOptionalListEnum(query.Orchestrator, isValidContainerOrchestrator)
	if err != nil {
		return err
	}
	query.SourceScopeKind, err = normalizeOptionalListEnum(query.SourceScopeKind, isValidContainerSourceScopeKind)
	return err
}

// normalizeListSourceScope validates source scope constraints in a list query.
// It ensures that SourceScopeKind and SourceScope are paired (both non-empty or both empty),
// and validates that SourceScopeKind is compatible with the specified orchestrator.
func normalizeListSourceScope(query *ListQuery) error {
	if query == nil {
		return errInvalidListQuery
	}
	query.SourceScope = strings.TrimSpace(query.SourceScope)
	if (query.SourceScopeKind == "") != (query.SourceScope == "") {
		return errInvalidListQuery
	}
	if query.SourceScope == "" {
		return nil
	}
	if !sourceScopeKindCompatibleWithOrchestrator(query.Orchestrator, query.SourceScopeKind) {
		return errInvalidListQuery
	}
	return nil
}

// normalizeOptionalListEnum 规范化并验证可选的枚举字符串值。若非空字符串经验证函数检查无效，则返回错误；否则返回规范化后的字符串。
func normalizeOptionalListEnum(value string, valid func(string) bool) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value != "" && !valid(value) {
		return "", errInvalidListQuery
	}
	return value, nil
}

// isValidContainerState 报告 state 是否为有效的容器状态。
func isValidContainerState(state string) bool {
	return slices.Contains(validContainerStates, state)
}

// isValidContainerHealth reports whether a health state is valid.
func isValidContainerHealth(health string) bool {
	return slices.Contains([]string{
		containerHealthHealthy,
		containerHealthUnhealthy,
		containerHealthStarting,
		containerHealthNone,
		containerHealthUnavailable,
	}, health)
}

// isValidContainerOrchestrator 报告给定的值是否为有效的容器编排器。
func isValidContainerOrchestrator(value string) bool {
	return slices.Contains(validContainerOrchestrators, value)
}

// isValidContainerSourceScopeKind reports whether value is a valid container source scope kind.
func isValidContainerSourceScopeKind(value string) bool {
	return slices.Contains(validContainerSourceScopeKinds, value)
}

// isValidMountID reports whether value is a valid mount ID.
func isValidMountID(value string) bool {
	return mountIDPattern.MatchString(strings.TrimSpace(value))
}

// stableMountID generates a stable identifier for a mount based on its destination, source, and type.
func stableMountID(mount Mount) string {
	parts := []string{
		strings.TrimSpace(mount.Destination),
		strings.TrimSpace(mount.Source),
		strings.TrimSpace(mount.Type),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "m_" + hex.EncodeToString(sum[:16])
}
