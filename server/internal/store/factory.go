package store

// Factory 暴露插件可见的最小仓储集合。
//
// MVP 阶段刻意保持接口收敛，只在确有插件需求时再增加新的仓储访问入口。
type Factory interface {
	// Users 返回用户能力插件使用的用户仓储。
	Users() UserRepository
}
