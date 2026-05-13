// Package menu 存放后端声明的导航元数据，供后续壳层装配使用。
package menu

// Item 表示一个由后端声明的菜单项。
type Item struct {
	Code       string
	Title      string
	Path       string
	Icon       string
	Permission string
	Plugin     string
}

// Registry 按注册顺序保存菜单声明，保证插件装配结果稳定可预期。
type Registry struct {
	items []Item
}

// NewRegistry 创建一个空的菜单注册表。
func NewRegistry() *Registry {
	return &Registry{items: make([]Item, 0)}
}

// Register 向注册表追加一个菜单项。
func (r *Registry) Register(item Item) {
	r.items = append(r.items, item)
}

// Items 返回当前已注册菜单集合的副本，避免外部修改内部状态。
func (r *Registry) Items() []Item {
	items := make([]Item, len(r.items))
	copy(items, r.items)
	return items
}
