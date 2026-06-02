// Package permission 存放模块声明的后端权限元数据，供后续鉴权装配使用。
package permission

// Item 表示一个由模块声明的权限点。
type Item struct {
	// Code 是权限点的稳定编码，路由、菜单与鉴权策略都应围绕它对齐。
	Code        string
	Name        string
	Description string
	// Category 是权限点的稳定分类元数据，由权限声明侧提供 canonical 真值。
	Category string
	// Module 标记权限声明来源，便于定位冲突与后续按模块聚合能力。
	Module string
}

// Registry 按注册顺序保存权限声明，供后续鉴权与菜单装配复用。
type Registry struct {
	items []Item
}

// NewRegistry 创建一个空的权限注册表。
func NewRegistry() *Registry {
	return &Registry{items: make([]Item, 0)}
}

// Register 按调用顺序向注册表追加一个权限声明。
//
// 该方法不隐式合并同名权限，目的是让重复声明在装配或测试阶段显式暴露。
func (r *Registry) Register(item Item) {
	r.items = append(r.items, item)
}

// Items 返回当前已注册权限集合的副本。
//
// 调用方只能读取快照，不能借由返回值回写注册表内部状态。
func (r *Registry) Items() []Item {
	items := make([]Item, len(r.items))
	copy(items, r.items)
	return items
}
