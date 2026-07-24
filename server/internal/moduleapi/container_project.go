package moduleapi

import "context"

// UpdateComposeRuntimeCandidate 描述容器模块为平台更新推导出的宿主机 Compose 根目录候选。
// Root 只在服务端模块间传递，HTTP 请求只能提交 CandidateKey；候选不写入持久配置。
type UpdateComposeRuntimeCandidate struct {
	CandidateKey string
	Root         string
	WorkingDir   string
	ConfigFiles  []string
	ProjectName  string
	Confidence   string
	Warnings     []string
}

// UpdateComposeRuntimeReader 暴露当前 server 容器的受限 Compose 运行时发现能力。
type UpdateComposeRuntimeReader interface {
	DiscoverCurrentServerCompose(ctx context.Context) ([]UpdateComposeRuntimeCandidate, error)
}

// ContainerProjectMember 描述 Project 模块可消费的单个容器窄化运行时投影。
//
// 它有意排除日志、事件、统计、Shell、inspect 载荷等容器详情，避免 Project 模块形成第二份运行时真相。
type ContainerProjectMember struct {
	ContainerID    string
	ContainerName  string
	ServiceName    string
	CanonicalState string
}

// ContainerProjectRuntimeSummary 描述一个 Compose 项目标识的有界运行时聚合结果。
type ContainerProjectRuntimeSummary struct {
	CanonicalProjectName string
	RunningCount         int
	StoppedCount         int
	Members              []ContainerProjectMember
}

// ContainerProjectServiceResourceSummary 描述单个服务的有界运行时资源聚合结果。
type ContainerProjectServiceResourceSummary struct {
	ServiceName                  string
	ContainerCount               int
	RunningCount                 int
	StoppedCount                 int
	TransitioningCount           int
	IssueCount                   int
	HealthyContainerCount        int
	UnhealthyContainerCount      int
	StartingContainerCount       int
	RestartCount                 int
	StatsAvailable               bool
	StatsAvailableContainerCount int
	CPUPercent                   float64
	MemoryUsageBytes             int64
	MemoryLimitBytes             int64
}

// ContainerProjectResourceSummary 描述项目级有界运行时资源聚合结果。
type ContainerProjectResourceSummary struct {
	CanonicalProjectName         string
	CollectedAt                  string
	StatsAvailable               bool
	StatsAvailableContainerCount int
	HealthyContainerCount        int
	UnhealthyContainerCount      int
	StartingContainerCount       int
	RestartCount                 int
	CPUPercent                   float64
	MemoryUsageBytes             int64
	MemoryLimitBytes             int64
	RxBytes                      int64
	TxBytes                      int64
	Services                     []ContainerProjectServiceResourceSummary
}

// ContainerProjectRuntimeContainerCounts 描述有界的运行时容器计数聚合结果。
type ContainerProjectRuntimeContainerCounts struct {
	Running int
	Stopped int
	Total   int
}

// ContainerProjectRuntimeCandidate 描述根据容器运行时元数据投影出的一个 Compose 导入候选。
//
// 容器模块拥有运行时元数据提取以及与运行时真相绑定的状态、原因码；Project 模块可以消费候选，
// 但不得自行解析 Docker labels。
type ContainerProjectRuntimeCandidate struct {
	CandidateKey           string
	CanonicalProjectName   string
	Status                 string
	StatusReasonCodes      []string
	Importable             bool
	RuntimeType            string
	RuntimeVersion         string
	WorkingDirectory       string
	WorkingDirectorySource string
	ConfigFiles            []string
	ServiceNames           []string
	ContainerCounts        ContainerProjectRuntimeContainerCounts
	Warnings               []string
}

// ContainerProjectRuntimeReader 暴露供 Project 模块聚合服务状态的窄化共享边界。
//
// 该边界允许 Project 模块聚合运行时计数而不导入 container 模块内部实现；container 仍是运行时真相拥有者。
type ContainerProjectRuntimeReader interface {
	ListProjectMembers(ctx context.Context, hostScope string, canonicalProjectName string) (ContainerProjectRuntimeSummary, error)
	ListImportCandidates(ctx context.Context, hostScope string) ([]ContainerProjectRuntimeCandidate, error)
	ListImportCandidateMembers(
		ctx context.Context,
		hostScope string,
		candidate ContainerProjectRuntimeCandidate,
	) ([]ContainerProjectMember, error)
}

// ContainerProjectResourceReader 暴露供 Project 模块聚合概览资源的窄化共享边界。
//
// container 仍拥有统计采集、规范化、缓存和实时主题语义；Project 模块只能消费该有界聚合结果，
// 不得直接读取容器统计快照或运行时内部实现。
type ContainerProjectResourceReader interface {
	ReadProjectResourceSummary(
		ctx context.Context,
		hostScope string,
		canonicalProjectName string,
	) (ContainerProjectResourceSummary, error)
}

// ContainerProjectLogQuery 描述 Project 模块可向 container 运行时真相请求的有界日志查询。
type ContainerProjectLogQuery struct {
	Tail       int
	Since      string
	Timestamps bool
	Stdout     bool
	Stderr     bool
	// FollowOnly 在查询用于实时流时抑制运行时尾部重放；历史条目必须在开始 follow 前通过 ReadProjectLogs 获取。
	FollowOnly bool
}

// ContainerProjectLogEntry 保留一条带有明确运行时来源归属的项目日志条目。
type ContainerProjectLogEntry struct {
	ContainerID   string
	ContainerName string
	ServiceName   string
	Line          string
	Stream        string
	OccurredAt    string
}

// ContainerProjectLogSnapshot 描述从项目成员容器聚合出的有界项目日志快照。
type ContainerProjectLogSnapshot struct {
	CanonicalProjectName string
	Tail                 int
	Since                *string
	Timestamps           bool
	Stdout               bool
	Stderr               bool
	Truncated            bool
	Entries              []ContainerProjectLogEntry
}

// ContainerProjectLogReader 暴露供 Project 模块聚合日志并接入实时 fan-in 的窄化共享边界。
//
// container 仍拥有运行时日志传输、规范化和单容器日志语义；Project 模块只能消费该有界多容器投影。
type ContainerProjectLogReader interface {
	ReadProjectLogs(
		ctx context.Context,
		hostScope string,
		canonicalProjectName string,
		query ContainerProjectLogQuery,
	) (ContainerProjectLogSnapshot, error)
	StreamProjectLogs(
		ctx context.Context,
		hostScope string,
		canonicalProjectName string,
		query ContainerProjectLogQuery,
		emit func(ContainerProjectLogEntry) error,
	) error
}
