// Package project 定义 Compose 项目管理权威边界拥有的数据模型。
package project

import "time"

// Record 是项目注册表一行记录的模块拥有持久化模型。
//
// 第一阶段不在此记录运行时状态，只保存注册、所有权、文件、快照和漂移元数据。
type Record struct {
	ApplicationRecordID      uint64
	ApplicationID            string
	ApplicationName          *string
	WorkspacePath            string
	ComposeProjectName       string
	ComposeProjectNameSource string
	DisplayName              string
	SourceType               string
	OwnershipMode            string
	LifecycleStrategyKind    string
	LifecycleReviewStatus    string
	LifecycleConfigJSON      []byte
	LastObservedConfigHash   string
	WorkspaceAnnotationsJSON []byte
	LastDriftCheckedAt       *time.Time
	DriftStatus              string
	CreatedBy                *uint64
	UpdatedBy                *uint64
	DeletedBy                *uint64
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                int64
}

// FileRecord 保存项目中一个按顺序排列的 Compose 或环境文件引用。
type FileRecord struct {
	ID                  uint64
	ApplicationRecordID uint64
	Kind                string
	Role                string
	AbsolutePath        string
	DisplayPath         string
	OrderIndex          int
	LastObservedHash    string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// SnapshotRecord 保存项目最近一次成功规范化的 Compose 快照。
type SnapshotRecord struct {
	ApplicationRecordID    uint64
	NormalizedComposeJSON  []byte
	ConfigHash             string
	RefreshedAt            time.Time
	DeclaredServiceCount   int
	DeclaredServicesDigest string
}
