// Package menu 存放后端声明的导航元数据，供后续壳层装配使用。
package menu

// Item 表示一个由后端声明的菜单项。
type Item struct {
	// Code 是菜单项的稳定后端标识，用于后续增量对比、去重或权限联动。
	Code  string
	Title string
	Path  string
	Icon  string
	// Permission 记录访问该菜单所需的后端权限编码；留空表示暂不做权限门控。
	Permission string
	// Plugin 标记菜单归属的插件，便于启动诊断与后续按插件裁剪导航。
	Plugin string
}

// Registry 按注册顺序保存菜单声明，保证插件装配结果稳定可预期。
type Registry struct {
	items []Item
}

// NewRegistry 创建一个空的菜单注册表。
func NewRegistry() *Registry {
	return &Registry{items: make([]Item, 0)}
}

// Register 按调用顺序向注册表追加一个菜单项。
//
// 当前注册表保持“显式声明即生效”的最小语义，不在此处做去重或权限校验，
// 以便把冲突处理留给更接近装配阶段的调用方。
func (r *Registry) Register(item Item) {
	r.items = append(r.items, item)
}

// Items 返回当前已注册菜单集合的副本。
//
// 返回顺序与插件注册顺序一致，便于上层在生成导航时保持稳定输出。
func (r *Registry) Items() []Item {
	items := make([]Item, len(r.items))
	copy(items, r.items)
	return items
}
