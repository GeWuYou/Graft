// Package cronx 存放插件声明的定时任务元数据，供后续调度器装配使用。
package cronx

// Job 描述一个待注册的定时任务。
type Job struct {
	Name     string
	Schedule string
	Plugin   string
}

// Registry 按注册顺序保存任务声明，供后续调度器接线阶段消费。
type Registry struct {
	items []Job
}

// NewRegistry 创建一个空的定时任务注册表。
func NewRegistry() *Registry {
	return &Registry{items: make([]Job, 0)}
}

// Register 向注册表追加一个定时任务声明。
func (r *Registry) Register(item Job) {
	r.items = append(r.items, item)
}

// Items 返回当前已注册任务集合的副本，避免外部篡改内部切片。
func (r *Registry) Items() []Job {
	items := make([]Job, len(r.items))
	copy(items, r.items)
	return items
}
