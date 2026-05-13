// Package permission 存放插件声明的后端权限元数据，供后续鉴权装配使用。
package permission

// Item 表示一个由插件声明的权限点。
type Item struct {
	Code        string
	Name        string
	Description string
	Plugin      string
}

// Registry 按注册顺序保存权限声明，供后续鉴权与菜单装配复用。
type Registry struct {
	items []Item
}

// NewRegistry 创建一个空的权限注册表。
func NewRegistry() *Registry {
	return &Registry{items: make([]Item, 0)}
}

// Register 向注册表追加一个权限声明。
func (r *Registry) Register(item Item) {
	r.items = append(r.items, item)
}

// Items 返回当前已注册权限集合的副本，避免调用方直接修改内部切片。
func (r *Registry) Items() []Item {
	items := make([]Item, len(r.items))
	copy(items, r.items)
	return items
}
