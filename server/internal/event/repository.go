package event

import "context"

// Repository 是未来 Outbox 持久化实现的最小边界。
//
// 本地 dispatcher 不使用 Repository；Durable 发布必须由业务事务中的实现原子写入
// 业务事实和 Outbox，不能在事务提交后补写一条孤立事件。
type Repository interface {
	Append(context.Context, Event) (Receipt, error)
}
