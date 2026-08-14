package moduleapi

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

// WorkspaceSnapshot 是 Build 执行消费的不可变源码输入。MaterializedRoot 仅是来源
// adapter 交接时的临时输入；Build 持久化或执行时必须只使用 MaterializationRef。
type WorkspaceSnapshot struct {
	ID                 string
	WorkspaceID        string
	SourceKind         string
	SourceReference    string
	ContentDigest      string
	MaterializedRoot   string
	MaterializationRef string
	CreatedAt          time.Time
}

// BuildWorkspace 是 Build 所有的可复用来源定义；实际源码内容由 Snapshot 冻结，
// Workspace 本身只保存来源身份和生命周期策略，不保存任意主机路径。
type BuildWorkspace struct {
	ID              string
	Name            string
	SourceKind      string
	SourceReference string
	RetentionPolicy string
	CreatedBy       uint64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// BuilderProfile 是 Build-owned 的 Driver 配置和执行策略，不携带 Runtime endpoint 或凭据。
type BuilderProfile struct {
	ID            string
	DisplayName   string
	DriverRef     string
	DriverVersion string
	Policy        json.RawMessage
}

// BuilderInstance 是 Profile 绑定 Runtime Target build capability 后的可执行实例。
type BuilderInstance struct {
	ID              string
	ProfileID       string
	RuntimeTargetID int64
	Status          string
	Labels          map[string]string
	DriverRef       string
	DriverVersion   string
}

// BuildCapabilityRequirement 是由冻结 Template、Driver、平台和 Snapshot 交付要求推导出的静态要求。
// 它不携带 Runtime Target 连接、凭据或动态遥测事实。
type BuildCapabilityRequirement struct {
	DriverRef             string
	TemplateRef           string
	DestinationKind       string
	CachePolicy           string
	SecurityPolicy        string
	Platforms             []string
	SnapshotDeliveryModes []string
	RequiredFeatures      []string
	FeatureRequirements   []BuildCapabilityFeatureRequirement
}

// BuildCapabilityFeatureRequirement 保留单项 Provider feature 的冻结请求意图，供执行重试复放协商结论。
type BuildCapabilityFeatureRequirement struct {
	Feature string
	Mode    string
}

const (
	// BuildCapabilityFeatureRequired requires the provider capability.
	BuildCapabilityFeatureRequired = "required"
	// BuildCapabilityFeaturePreferred records a preferred provider capability.
	BuildCapabilityFeaturePreferred = "preferred"
	// BuildCapabilityFeatureOptional permits omission of the provider capability.
	BuildCapabilityFeatureOptional = "optional"
)

// BuildExecutionCapability 是 Runtime provider 对单个 Builder 的版本化静态能力事实。
// ProviderCapabilityVersion 才是能力语义版本；DriverVersion 仅用于诊断。
type BuildExecutionCapability struct {
	ProviderCapabilityProfile string
	ProviderCapabilityVersion string
	SupportedDrivers          []string
	SupportedPlatforms        []string
	SnapshotDeliveryModes     []string
	Features                  []string
}

// NegotiatedCapability 是 CapabilityMatcher 对一次静态匹配的可重放结果。
type NegotiatedCapability struct {
	ProviderCapabilityProfile string
	ProviderCapabilityVersion string
	DriverRef                 string
	SnapshotDeliveryMode      string
	SatisfiedFeatures         []string
	UnsatisfiedFeatures       []string
	PreferredMissReasons      map[string]string
	OptionalOmissionReasons   map[string]string
}

// CapabilityMatcher 是 Build-owned 的纯能力协商边界；实现不得读取遥测或重新选择 Target。
type CapabilityMatcher interface {
	MatchBuildCapability(BuildCapabilityRequirement, BuildExecutionCapability) (NegotiatedCapability, error)
}

// WorkspaceMaterializationRequest 描述一次按不可变 Snapshot 身份隔离的执行物化。
type WorkspaceMaterializationRequest struct {
	ExecutionID string
}

// WorkspaceMaterialization 是 Build-owned 执行字节的私有引用，不能进入 Task metadata 或 HTTP。
type WorkspaceMaterialization struct {
	SnapshotID         string
	ContentDigest      string
	MaterializationRef string
}

// WorkspaceMaterializer 负责 Snapshot -> execution workspace -> cleanup 生命周期。
type WorkspaceMaterializer interface {
	MaterializeSnapshot(context.Context, WorkspaceSnapshot, WorkspaceMaterializationRequest) (WorkspaceMaterialization, error)
	ReleaseMaterialization(context.Context, WorkspaceMaterialization) error
}

// BuilderPool 是 Builder Instance 的选择策略集合，不拥有 Task 状态或独立调度循环。
type BuilderPool struct {
	ID               string
	DisplayName      string
	SchedulingPolicy string
	Selector         json.RawMessage
}

// BuilderPoolMember 是 Pool 对 Instance 的 Build-owned 调度关系，不复制 Runtime
// Target 的连接信息，也不拥有 Task 执行状态。
type BuilderPoolMember struct {
	PoolID     string
	InstanceID string
	Priority   int
}

// BuilderPoolSelection 是持久化 Pool 策略返回的静态选择事实。Cursor 只在
// RoundRobin 策略中有值，调用方必须把它冻结为 Placement Evidence。
type BuilderPoolSelection struct {
	Instance BuilderInstance
	Cursor   *int64
}

// BuilderPlacement 是冻结计划中一个目标平台到 Builder Instance/Runtime Target 的可重放分配。
// 它由 Build scheduler 在提交前解析，Task Runtime 只消费其稳定身份而不重新调度。
type BuilderPlacement struct {
	Platform           string
	BuilderInstanceID  string
	RuntimeTargetID    int64
	SchedulingPolicy   string
	SchedulingEvidence json.RawMessage
}

// BuilderReservation 是 Build 对单个 Builder Instance 的容量租约事实。
// FenceToken 只用于拒绝旧执行尝试，不能替代 Task Runtime 的执行状态。
type BuilderReservation struct {
	ID             string
	InstanceID     string
	PlanID         string
	TaskID         uint64
	Attempt        int
	LegID          string
	FenceToken     string
	State          string
	LeaseExpiresAt time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

const (
	// BuilderReservationReserved 表示容量已被 Build 原子占用。
	BuilderReservationReserved = "reserved"
	// BuilderReservationAccepted 表示 Task 已接受该容量租约。
	BuilderReservationAccepted = "accepted"
	// BuilderReservationRunning 表示执行器已开始使用该租约。
	BuilderReservationRunning = "running"
	// BuilderReservationReleased 表示执行终态已释放容量。
	BuilderReservationReleased = "released"
	// BuilderReservationExpired 表示租约过期且不可再使用。
	BuilderReservationExpired = "expired"
	// BuilderReservationAbandoned 表示外部结果未知，等待人工恢复。
	BuilderReservationAbandoned = "abandoned"
)

// BuilderReservationRepository 是 Build-owned 的最小容量 lease 持久化边界。
// 实现不得启动 scheduler，也不得写入 PlatformAvailabilityStore。
type BuilderReservationRepository interface {
	ReserveBuilder(ctx context.Context, tx *sql.Tx, reservation BuilderReservation) (BuilderReservation, error)
	ReserveBuilderAttempt(ctx context.Context, reservation BuilderReservation) (BuilderReservation, error)
	MarkBuilderReservationRunning(ctx context.Context, taskID uint64, legID, fenceToken string) error
	RenewBuilderReservation(ctx context.Context, taskID uint64, legID, fenceToken string, leaseExpiresAt time.Time) error
	ReleaseBuilderReservation(ctx context.Context, taskID uint64, legID, fenceToken, state string) error
}

// BuilderResourceRepository 是 Builder Profile、Instance、Pool 及成员关系的持久化边界。
type BuilderResourceRepository interface {
	CreateBuilderProfile(context.Context, BuilderProfile, uint64) error
	CreateBuilderInstance(context.Context, BuilderInstance, uint64) error
	CreateBuilderPool(context.Context, BuilderPool, uint64) error
	ReplaceBuilderPoolMembers(context.Context, string, []BuilderPoolMember, uint64) error
	GetBuilderPool(context.Context, string) (BuilderPool, error)
	ListBuilderPoolMembers(context.Context, string) ([]BuilderInstance, error)
	SelectRoundRobinBuilderInstance(context.Context, string) (BuilderPoolSelection, error)
}

const (
	// WorkspaceSourceApplication 表示由 Application Workspace adapter 提供来源。
	WorkspaceSourceApplication = "application_workspace"
	// WorkspaceSourceGit 表示由受控 Git provider 提供来源。
	WorkspaceSourceGit = "git"
	// WorkspaceSourceArchive 表示由平台上传归档提供来源。
	WorkspaceSourceArchive = "uploaded_archive"
	// WorkspaceSourceGenerated 表示由 Build 规则生成来源。
	WorkspaceSourceGenerated = "generated"
	// WorkspaceSourceTargetLocal 表示经 Runtime Target 授权的本地来源。
	WorkspaceSourceTargetLocal = "target_local"
)

// ApplicationWorkspaceSnapshotResolver 在 Build 创建执行计划前冻结已授权的
// Application Workspace。
type ApplicationWorkspaceSnapshotResolver interface {
	FreezeApplicationWorkspaceSnapshot(context.Context, string) (WorkspaceSnapshot, ApplicationBuildContext, error)
}

// BuildDestination 是非秘密、provider-neutral 的输出绑定。Phase 1 支持 OCI
// Registry 发布；凭据和端点仍由 Infrastructure 与所选 Runtime Target 持有。
type BuildDestination struct {
	Kind          string
	ConnectionRef string
	RepositoryRef string
	Reference     string
}

// AuthorizedArtifactDestination 是 Registry 解析后的非秘密产物目的地；它可冻结进执行计划，提供方端点和凭据引用仍由 Registry 私有持有。
type AuthorizedArtifactDestination struct {
	Kind          string
	ConnectionRef string
	RepositoryRef string
	Reference     string
}

// RegistryDestinationResolver 是 Build 提交时使用的窄基础设施能力，负责校验操作者对仓库的发布授权并规范化非秘密目的地身份。
type RegistryDestinationResolver interface {
	ResolveArtifactDestination(context.Context, uint64, BuildDestination) (AuthorizedArtifactDestination, error)
}

// RegistryPublicationBinding 是仅供执行器使用的提供方绑定。它在执行期解析，
// 绝不能序列化进 Plan、Task、HTTP 响应或日志。
type RegistryPublicationBinding struct {
	Destination   AuthorizedArtifactDestination
	Endpoint      string
	CredentialRef string
	AuthExecution RegistryAuthExecution
}

// RegistryAuthExecution 描述 Runtime adapter 使用 Registry 凭据的非明文方式。
// 新执行只允许 ephemeral session；Docker credential store 仅保留历史事实读取。
type RegistryAuthExecution struct {
	Mode string
}

const (
	// RegistryAuthExecutionDockerStore 表示由 Docker credential store 负责认证。
	RegistryAuthExecutionDockerStore = "docker-runtime-store"
	// RegistryAuthExecutionEphemeral 表示本次操作只使用隔离的短期凭据会话。
	RegistryAuthExecutionEphemeral = "ephemeral-credential"
)

// RegistryPublicationResolver 解析所选 Runtime Target 发布已构建镜像所需的
// 私有提供方绑定。
type RegistryPublicationResolver interface {
	ResolvePublicationBinding(context.Context, AuthorizedArtifactDestination) (RegistryPublicationBinding, error)
}

// EphemeralCredentialSession 是一次 Registry 操作的短期凭据会话。
// Secret 只允许在 CredentialProvider 与 RuntimeExecutionAdapter 的进程内边界流转。
type EphemeralCredentialSession struct {
	ID        string
	ExpiresAt time.Time
}

// CredentialRequest 描述一次最小作用域的 Registry 凭据申请。
type CredentialRequest struct {
	CredentialRef string
	Endpoint      string
	RepositoryRef string
	Operation     string
	ExpiresAt     time.Time
}

// CredentialEligibilityRequest 描述一次不签发凭据的最小 scope 评估。
// 它只允许消费者询问已知的 opaque reference，不能枚举 Provider 的凭据目录。
type CredentialEligibilityRequest struct {
	CredentialRef string
	Endpoint      string
	RepositoryRef string
	Operation     string
}

// CredentialEligibilityStatus 是 Provider 对已知 scope 的非秘密结论。
type CredentialEligibilityStatus string

const (
	// CredentialEligibilityEligible 表示 Provider 当前可为该 scope 签发短期会话。
	CredentialEligibilityEligible CredentialEligibilityStatus = "eligible"
	// CredentialEligibilityIneligible 表示 reference、scope 或有效期不满足签发条件。
	CredentialEligibilityIneligible CredentialEligibilityStatus = "ineligible"
)

// CredentialEligibility 只返回当前 scope 是否可签发，不暴露 secret、来源、过期时间或 Provider 内部信息。
type CredentialEligibility struct {
	Status CredentialEligibilityStatus
}

// CredentialProvider 从 Secret authority 申请短期凭据，并负责终态撤销。
type CredentialProvider interface {
	Assess(context.Context, CredentialEligibilityRequest) (CredentialEligibility, error)
	Prepare(context.Context, CredentialRequest) (EphemeralCredentialSession, error)
	Inject(context.Context, EphemeralCredentialSession, CredentialInjectionTarget) error
	Revoke(context.Context, EphemeralCredentialSession) error
}

// CredentialInjectionTarget 是 Runtime adapter 提供的隔离注入位置，不携带凭据原文。
type CredentialInjectionTarget struct {
	ConfigDir     string
	Endpoint      string
	RepositoryRef string
}

// OCIRegistryVerificationRequest 描述一次由指定 Runtime Target 执行的非变更 OCI V2 认证探测。
// 它只接受已知的私有执行 binding，不能作为 HTTP 或 Registry 管理输入使用。
type OCIRegistryVerificationRequest struct {
	RuntimeTargetID int64
	CredentialRef   string
	Endpoint        string
	RepositoryRef   string
	Operation       string
}

// OCIRegistryVerificationResult 记录认证探测的最小事实，不包含响应内容、认证头或凭据细节。
// AuthenticationSucceeded 仅表示 V2 根端点接受了本次认证，绝不代表 Repository pull 或 push 已获授权。
type OCIRegistryVerificationResult struct {
	Reachable                bool
	ProtocolCompatible       bool
	AuthenticationChallenged bool
	AuthenticationSucceeded  bool
	ProviderScopeConforms    bool
}

// RuntimeOCIRegistryVerifier 是 Runtime Target 对 Registry 暴露的最小认证验证 capability。
type RuntimeOCIRegistryVerifier interface {
	VerifyOCIRegistry(context.Context, OCIRegistryVerificationRequest) (OCIRegistryVerificationResult, error)
}

// RuntimeExecutionAdapter 是 Runtime Target 唯一拥有的隔离 Registry 执行边界。
// 调用方只提交非秘密 binding，adapter 自己申请、注入并清理 ephemeral session。
type RuntimeExecutionAdapter interface {
	PublishImage(context.Context, int64, DockerImageBuildResult, RegistryPublicationBinding, DockerImageBuildLogSink) (DockerImageBuildResult, error)
	PublishManifest(context.Context, int64, OCIManifestPublicationInput, RegistryPublicationBinding, DockerImageBuildLogSink) (OCIManifestPublicationResult, error)
	CopyOCIArtifact(context.Context, int64, OCIArtifactCopyInput, RegistryArtifactCopyBinding, DockerImageBuildLogSink) (OCIArtifactCopyResult, error)
	RuntimeOCIRegistryVerifier
}

// ArtifactPublicationSource 是从可变 Publication 选择的 Build-owned 摘要源。
// 它不携带 Publication tag，OCI copy provider 必须读取 Build 记录的不可变 digest。
type ArtifactPublicationSource struct {
	ArtifactID      string
	PublicationID   string
	Digest          string
	MediaType       string
	DestinationKind string
	ConnectionRef   string
	RepositoryRef   string
}

// AuthorizedArtifactCopy 在 Registry 完成请求操作者的双端授权后冻结非秘密的源和目的地身份。
type AuthorizedArtifactCopy struct {
	Source      ArtifactPublicationSource
	Destination AuthorizedArtifactDestination
}

// RegistryArtifactCopyBinding 是仅供执行器使用的基础设施状态；端点和凭据引用不得进入
// Plan、Task、HTTP 响应或日志。
type RegistryArtifactCopyBinding struct {
	SourceEndpoint      string
	SourceCredentialRef string
	SourceAuthExecution RegistryAuthExecution
	Destination         RegistryPublicationBinding
}

// RegistryArtifactCopyResolver 将来源读取和目的地写入授权留在 Registry，并只为
// provider-owned copy execution 解析私有 binding。
type RegistryArtifactCopyResolver interface {
	AuthorizeArtifactCopy(context.Context, uint64, ArtifactPublicationSource, BuildDestination) (AuthorizedArtifactCopy, error)
	ResolveArtifactCopyBinding(context.Context, AuthorizedArtifactCopy) (RegistryArtifactCopyBinding, error)
}

// BuildExecutionPlan 在 Task 提交前冻结每一项调用者可见的 Build 输入；它刻意不
// 包含 workspace path、endpoint、credential 或自由形式的 executor command。
type BuildExecutionPlan struct {
	ID                string
	Digest            string
	Workspace         WorkspaceSnapshot
	BuilderPoolID     string
	BuilderInstanceID string
	RuntimeTargetID   int64
	BuilderPlacements []BuilderPlacement
	Driver            string
	TemplateRef       string
	CachePolicy       string
	SecurityPolicy    string
	Platforms         []string
	Destination       BuildDestination
	CreatedAt         time.Time
}

// PlacementForPlatform 返回冻结计划中指定平台的 Builder placement。旧计划尚未持久化 placement
// 时使用计划级 Runtime Target 作为兼容读取；新提交必须显式冻结每个平台的 placement。
func (p BuildExecutionPlan) PlacementForPlatform(platform string) (BuilderPlacement, bool) {
	for _, placement := range p.BuilderPlacements {
		if placement.Platform == platform {
			return placement, placement.RuntimeTargetID > 0
		}
	}
	if len(p.BuilderPlacements) == 0 && p.RuntimeTargetID > 0 {
		return BuilderPlacement{Platform: platform, BuilderInstanceID: p.BuilderInstanceID, RuntimeTargetID: p.RuntimeTargetID}, true
	}
	return BuilderPlacement{}, false
}

// PlatformArtifact 是协调构建中单个平台产物的不可变事实；最终 OCI Manifest 由 Build 在收齐全部 leg 后单独发布。
type PlatformArtifact struct {
	LegID      string
	Platform   string
	Digest     string
	MediaType  string
	SizeBytes  int64
	ProducedAt time.Time
}

// OCIManifestPublicationInput 是由 Build 完整性校验后的不可变平台 Artifact 集合；它禁止使用 mutable tag 作为合并输入。
type OCIManifestPublicationInput struct {
	Destination       AuthorizedArtifactDestination
	PlatformArtifacts []PlatformArtifact
}

// OCIManifestPublicationResult 是 Driver 发布 OCI Image Index 或 Manifest List 后返回的摘要寻址结果。
type OCIManifestPublicationResult struct {
	Digest    string
	MediaType string
	SizeBytes int64
}

// OCIArtifactCopyInput 包含不可变 source digest 和已授权的目的地；它刻意排除 source tag、
// Registry endpoint 与 credential，防止 provider 将 Promotion 降级为可变 tag copy。
type OCIArtifactCopyInput struct {
	Source      ArtifactPublicationSource
	Destination AuthorizedArtifactDestination
}

// OCIArtifactCopyResult 是 provider 对已复制 digest 的证据；Build 会基于该 digest
// 结算新的 Publication，而不改变 Artifact。
type OCIArtifactCopyResult struct {
	Digest    string
	MediaType string
	SizeBytes int64
}

// ArtifactPromotionTaskInput 是 Task Runtime 持久化的 Promotion 执行输入。来源必须是
// Build 读取到的 Publication identity，Registry endpoint、凭据和 provider 命令不得进入 Task。
type ArtifactPromotionTaskInput struct {
	Source          ArtifactPublicationSource     `json:"source"`
	Destination     AuthorizedArtifactDestination `json:"destination"`
	RuntimeTargetID int64                         `json:"runtime_target_id"`
}

// BuildPlanTaskInput 是唯一允许进入 Task metadata 的 Build v2 payload。执行器
// 从 Build-owned plan storage 解析本地物化输入，绝不接收客户端给出的 source path。
type BuildPlanTaskInput struct {
	BuildID         string `json:"build_id"`
	ExecutionPlanID string `json:"execution_plan_id"`
	Platform        string `json:"platform,omitempty"`
	LegID           string `json:"leg_id,omitempty"`
}

// ApplicationBuildContext 是 Project 模块为构建消费者冻结的、经授权的应用来源上下文。
// 它只包含构建所需的公开身份和受控 workspace 根目录，不暴露 Application entity 或仓储。
type ApplicationBuildContext struct {
	ApplicationID       string
	ApplicationRecordID uint64
	DisplayName         string
	WorkspaceRoot       string
	RuntimeTargetID     uint64
	RuntimeTargetName   string
	RuntimeProvider     string
	CanBuild            bool
}

// ApplicationBuildContextResolver 解析当前操作者可访问的 Application 构建上下文。
type ApplicationBuildContextResolver interface {
	ResolveApplicationBuildContext(context.Context, string) (ApplicationBuildContext, error)
}

// DockerImageBuildInput 是 Container 模块接受的受控 Docker 构建请求。
// 路径必须相对于已授权 workspace，调用方不能传入 daemon、host 或任意 CLI 参数。
type DockerImageBuildInput struct {
	WorkspaceRoot      string
	MaterializationRef string
	ContextPath        string
	DockerfilePath     string
	ImageRepository    string
	ImageTag           string
	Platform           string
	BuildArgs          []DockerImageBuildArg
}

// DockerImageBuildArg 表示非敏感 Docker 构建参数。
type DockerImageBuildArg struct {
	Name  string
	Value string
}

// DockerImageBuildResult 是 Docker executor 生成的规范化镜像事实。
type DockerImageBuildResult struct {
	ImageID      string
	Digest       string
	Repository   string
	Tag          string
	SizeBytes    int64
	OS           string
	Architecture string
	Variant      string
}

// BuildDriverLogSink 接收经过 executor 限长和脱敏的逐行 Driver 输出。
// 它属于 Build/Task 的通用执行边界，不能携带 Docker 专有的命令或连接事实。
type BuildDriverLogSink func(context.Context, TaskLogEntry) error

// DockerImageBuildLogSink 保留 Docker capability 的兼容别名。
// 新的 provider-neutral Driver contract 必须使用 BuildDriverLogSink。
type DockerImageBuildLogSink = BuildDriverLogSink

// DockerImageBuildCapability 由 Container 模块提供 Docker image build 执行能力。
type DockerImageBuildCapability interface {
	BuildImage(context.Context, DockerImageBuildInput, DockerImageBuildLogSink) (DockerImageBuildResult, error)
}

// TargetBoundDockerImageBuildCapability 只在提供方 owner 重新校验所选 Runtime
// Target 后执行 Docker build；冻结的 Execution Plan 指定 target 时，Build 必须
// 优先使用此边界。
type TargetBoundDockerImageBuildCapability interface {
	BuildImageOnTarget(context.Context, int64, DockerImageBuildInput, DockerImageBuildLogSink) (DockerImageBuildResult, error)
}

// WorkspaceSnapshotDeliveryRequest 是 Build 将冻结 Snapshot 交给 Runtime provider 的受控请求。
// MaterializationRef 是 Build-owned opaque reference；provider 不接收宿主机路径。
type WorkspaceSnapshotDeliveryRequest struct {
	TargetID           int64
	SnapshotID         string
	ContentDigest      string
	MaterializationRef string
	DeliveryMode       string
}

// WorkspaceSnapshotDeliveryResult 是 provider 对 Snapshot 消费前校验的不可变证明。
// 它不回传 endpoint、凭据或新的宿主机路径。
type WorkspaceSnapshotDeliveryResult struct {
	TargetID      int64
	SnapshotID    string
	ContentDigest string
	DeliveryMode  string
}

// NewWorkspaceSnapshotMaterializationReference 生成绑定 Snapshot 身份、内容摘要和物化目录的 opaque 引用。
func NewWorkspaceSnapshotMaterializationReference(snapshotID, contentDigest, root string) (string, error) {
	rootName := filepath.Base(filepath.Clean(strings.TrimSpace(root)))
	if strings.TrimSpace(snapshotID) == "" || strings.TrimSpace(contentDigest) == "" || rootName == "." || rootName == ".." || !strings.HasPrefix(rootName, "snapshot-") {
		return "", errors.New("workspace snapshot materialization reference input is invalid")
	}
	encode := base64.RawURLEncoding.EncodeToString
	return "build-snapshot:v1:" + encode([]byte(rootName)) + ":" + encode([]byte(snapshotID)) + ":" + encode([]byte(contentDigest)), nil
}

// ParseWorkspaceSnapshotMaterializationReference 解析并返回物化目录、Snapshot 身份和内容摘要。
//
//nolint:revive,cyclop // 解析结果是同一 opaque capability 的完整三元组，拆分会增加调用方错配风险。
func ParseWorkspaceSnapshotMaterializationReference(reference string) (rootName, snapshotID, contentDigest string, err error) {
	parts := strings.Split(strings.TrimSpace(reference), ":")
	if len(parts) != 5 || parts[0] != "build-snapshot" || parts[1] != "v1" {
		return "", "", "", errors.New("workspace snapshot materialization reference is invalid")
	}
	decode := base64.RawURLEncoding.DecodeString
	rootBytes, rootErr := decode(parts[2])
	idBytes, idErr := decode(parts[3])
	digestBytes, digestErr := decode(parts[4])
	if rootErr != nil || idErr != nil || digestErr != nil || len(rootBytes) == 0 || len(idBytes) == 0 || len(digestBytes) == 0 {
		return "", "", "", errors.New("workspace snapshot materialization reference is invalid")
	}
	rootName, snapshotID, contentDigest = string(rootBytes), string(idBytes), string(digestBytes)
	if rootName != filepath.Base(rootName) || strings.HasPrefix(rootName, ".") || !strings.HasPrefix(rootName, "snapshot-") {
		return "", "", "", errors.New("workspace snapshot materialization reference is invalid")
	}
	return rootName, snapshotID, contentDigest, nil
}

// TargetBoundWorkspaceSnapshotDeliveryCapability 负责证明选定 Runtime 能消费冻结 Snapshot。
type TargetBoundWorkspaceSnapshotDeliveryCapability interface {
	DeliverWorkspaceSnapshot(context.Context, WorkspaceSnapshotDeliveryRequest) (WorkspaceSnapshotDeliveryResult, error)
}

// ProviderDriverExecutionRequest 是 provider-neutral Driver 执行请求。
// 它只携带冻结的 Snapshot 身份、已验证的交付证明和平台选择，不携带端点、凭据、宿主机路径或 CLI 命令。
type ProviderDriverExecutionRequest struct {
	TargetID      int64
	DriverRef     string
	Platform      string
	SnapshotID    string
	ContentDigest string
	DeliveryProof WorkspaceSnapshotDeliveryResult
}

// ProviderDriverExecutionResult 是 Driver 对不可变平台产物的执行证明。
// Provider 必须返回与请求匹配的目标、Driver、平台和 Snapshot 身份，以及摘要寻址的产物事实。
type ProviderDriverExecutionResult struct {
	ProviderID     string
	TargetID       int64
	DriverRef      string
	Platform       string
	SnapshotID     string
	ContentDigest  string
	ArtifactDigest string
	MediaType      string
	SizeBytes      int64
}

// TargetBoundProviderDriverExecutionCapability 是 Runtime Target 所有的通用 Driver 执行边界。
// 非 Docker provider 必须先实现该边界并通过 Phase 9C conformance gate，才能被 Runtime Target 声明为可执行。
type TargetBoundProviderDriverExecutionCapability interface {
	ExecuteProviderDriver(context.Context, ProviderDriverExecutionRequest, BuildDriverLogSink) (ProviderDriverExecutionResult, error)
}

// ProviderExecutionConformanceRequest 是 Build 在执行前提交给 Runtime provider 的能力证明请求。
// 请求只携带冻结身份和 provider-neutral 选择，不允许携带 endpoint、凭据值或宿主机路径。
type ProviderExecutionConformanceRequest struct {
	TargetID      int64
	DriverRef     string
	Platform      string
	SnapshotID    string
	ContentDigest string
	DeliveryMode  string
}

// ProviderExecutionConformanceResult 是 provider 对当前目标执行资格的可重放证明。
// Executable 为 false 时，Build 必须在任何 Snapshot 交付或 Driver 调用前 fail-closed。
type ProviderExecutionConformanceResult struct {
	ProviderID            string
	ConformanceVersion    string
	Executable            bool
	SnapshotDeliveryProof bool
	DriverExecutionProof  bool
	PublicationProof      bool
	CancellationProof     bool
	CleanupProof          bool
}

// ProviderExecutionEvidence 是 Build 持久化的非秘密 provider 执行资格事实。
// 它不改变 Artifact 身份，也不承载连接、凭据或物化路径。
type ProviderExecutionEvidence struct {
	TaskID      uint64
	StageID     uint64
	TargetID    int64
	Platform    string
	Conformance ProviderExecutionConformanceResult
}

// TargetBoundProviderExecutionConformanceCapability 是 Runtime Target 唯一拥有的 provider 执行门槛。
// 它把 capability 声明与可执行 provider 注册、连接健康和生命周期证据绑定起来。
type TargetBoundProviderExecutionConformanceCapability interface {
	ConformProviderExecution(context.Context, ProviderExecutionConformanceRequest) (ProviderExecutionConformanceResult, error)
}

// TargetBoundDockerBuildProvider 是 Runtime Target 注册的完整 Docker 适配器边界。
// 它只约束 Docker reference provider；未来非 Docker provider 必须定义自己的 provider-neutral Driver contract。
type TargetBoundDockerBuildProvider interface {
	TargetBoundDockerImageBuildCapability
	TargetBoundWorkspaceSnapshotDeliveryCapability
	TargetBoundProviderExecutionConformanceCapability
	ProviderID() string
}

// TargetBoundDockerImagePublicationCapability 通过所选 Runtime Target，使用
// Infrastructure 所有的 provider binding 发布镜像。
type TargetBoundDockerImagePublicationCapability interface {
	PublishImageOnTarget(context.Context, int64, DockerImageBuildResult, RegistryPublicationBinding, DockerImageBuildLogSink) (DockerImageBuildResult, error)
}

// TargetBoundOCIManifestPublicationCapability 由支持多平台的 Builder Driver 实现。Build 只能传入已经验证的
// 平台 digest 集合与 Infrastructure 解析后的绑定，不能传递 Docker 命令、端点或凭据。
type TargetBoundOCIManifestPublicationCapability interface {
	PublishOCIManifestOnTarget(context.Context, int64, OCIManifestPublicationInput, RegistryPublicationBinding, DockerImageBuildLogSink) (OCIManifestPublicationResult, error)
}

// TargetBoundOCIArtifactCopyCapability 归 Runtime Target provider 所有。Build 只提供
// 不可变身份和 Registry 提供的私有 binding；provider 拥有协议命令、认证和 digest 校验。
type TargetBoundOCIArtifactCopyCapability interface {
	CopyOCIArtifactOnTarget(context.Context, int64, OCIArtifactCopyInput, RegistryArtifactCopyBinding, DockerImageBuildLogSink) (OCIArtifactCopyResult, error)
}

// TaskBatchQueryService 为列表型消费者提供批量 Task 读取能力，避免其自行逐行查询。
// 它独立于 TaskQueryService，便于现有实现渐进接入而不破坏旧消费者。
type TaskBatchQueryService interface {
	GetTasksByIDs(context.Context, []uint64) ([]TaskView, error)
}

// TaskTransactionalSubmissionAdapter 绑定调用方拥有的 SQL transaction，写入 Task 及其 stages。
// 实现不得提交、回滚或在调用完成后继续使用 transaction。
type TaskTransactionalSubmissionAdapter interface {
	SubmitTask(context.Context, SubmitTaskInput) (TaskReceipt, error)
}

// TaskTransactionalSubmissionFactory 将 caller-owned transaction 绑定为 Task 写参与者。
type TaskTransactionalSubmissionFactory interface {
	BindTaskTransaction(*sql.Tx) (TaskTransactionalSubmissionAdapter, error)
}

// BuildTaskInput 返回 Build executor 在 Task metadata 中使用的稳定、非敏感输入载荷。
// 该辅助类型刻意不包含 build arg 值、凭据或绝对路径。
type BuildTaskInput struct {
	BuildID         string          `json:"build_id"`
	ApplicationID   string          `json:"application_id"`
	ContextPath     string          `json:"context_path"`
	DockerfilePath  string          `json:"dockerfile_path"`
	ImageRepository string          `json:"image_repository"`
	ImageTag        string          `json:"image_tag"`
	BuildArgs       json.RawMessage `json:"build_args,omitempty"`
}
