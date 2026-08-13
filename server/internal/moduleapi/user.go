// Package moduleapi 定义稳定的跨模块能力契约。
package moduleapi

import (
	"context"
	"errors"
)

var (
	// ErrUserNotFound 表示跨模块读取的目标用户不存在。
	ErrUserNotFound = errors.New("user not found")
)

// UserSummary 是跨模块共享的稳定用户摘要 DTO。
//
// 该 DTO 只能承载其他模块明确可依赖的字段，避免把用户模块的内部模型直接泄漏出去。
type UserSummary struct {
	ID                    uint64
	Username              string
	Display               string
	ProtectedDefaultAdmin bool
}

// UserCandidate 是跨模块候选选择器所需的最小用户投影。
type UserCandidate struct {
	ID       uint64
	Username string
	Display  string
	Status   string
}

// UserCandidateQuery 描述跨模块候选选择器的服务端搜索和分页窗口。
type UserCandidateQuery struct {
	Search string
	Limit  int
	Offset int
}

// UserSecuritySummary 是安全态势读取方所需的窄化用户状态投影。
type UserSecuritySummary struct {
	ID     uint64
	Status string
}

// UserService 暴露其他模块可依赖的最小用户能力接口。
//
// 该接口的稳定性高于单个模块内部仓储；一旦签名或错误语义发生变化，需要同步评估所有依赖方。
type UserService interface {
	// GetUserByID 按 ID 返回稳定的用户摘要 DTO，而不是内部持久化模型。
	//
	// 未命中时实现应返回 ErrUserNotFound，方便调用方做统一分支处理。
	GetUserByID(ctx context.Context, id uint64) (UserSummary, error)
	// CountUsers 返回当前可管理用户总数，供跨模块摘要类只读能力使用。
	CountUsers(ctx context.Context) (int, error)
}

// UserCandidateReader 仅暴露受限候选选择所需的用户摘要查询，避免消费者依赖 user store。
type UserCandidateReader interface {
	ListUserCandidates(ctx context.Context, query UserCandidateQuery) ([]UserCandidate, int, error)
}

// UserSecurityReader 暴露安全态势聚合所需的窄化账户状态投影。
type UserSecurityReader interface {
	// ListSecuritySummaries 返回 afterID 之后按 ID 排序的有界数据页。
	ListSecuritySummaries(ctx context.Context, afterID uint64, limit int) ([]UserSecuritySummary, error)
}
