// Package permission 存放模块声明的后端权限元数据，供后续鉴权装配使用。
package permission

// Item 表示一个由模块声明的权限点。
type Item struct {
	// Code 是权限点的稳定编码，路由、菜单与鉴权策略都应围绕它对齐。
	Code string
	// Name 是权限展示 fallback 文案；key-aware 消费端应优先使用 DisplayKey。
	Name string
	// DisplayKey 是权限展示名称的稳定本地化 key。
	DisplayKey string
	// Description 是权限说明 fallback 文案；key-aware 消费端应优先使用 DescriptionKey。
	Description string
	// DescriptionKey 是权限说明的稳定本地化 key。
	DescriptionKey string
	// Module 标记权限声明来源，便于定位冲突与后续按模块聚合能力。
	Module string
	// Resource 是权限所属领域，用于权限目录分组和内置策略声明。
	Resource string
	// Action 是资源上的稳定动作，例如 view、deploy、delete。
	Action string
	// RiskLevel 标识权限变更的严重级别：low、medium、high 或 critical。
	RiskLevel string
	// RiskCategory 标识权限操作类别：read、write、destructive 或 security。
	RiskCategory string
}

// Risk levels 表示权限严重级别，Risk categories 表示权限操作类别。
const (
	RiskLevelLow            = "low"
	RiskLevelMedium         = "medium"
	RiskLevelHigh           = "high"
	RiskLevelCritical       = "critical"
	RiskCategoryRead        = "read"
	RiskCategoryWrite       = "write"
	RiskCategoryDestructive = "destructive"
	RiskCategorySecurity    = "security"
)

// Valid 仅接受可分组且可风险审查的权限声明。
func (i Item) Valid() bool {
	if i.Code == "" || i.Module == "" || i.Resource == "" || i.Action == "" || i.RiskCategory == "" {
		return false
	}
	switch i.RiskLevel {
	case RiskLevelLow, RiskLevelMedium, RiskLevelHigh, RiskLevelCritical:
	default:
		return false
	}
	switch i.RiskCategory {
	case RiskCategoryRead, RiskCategoryWrite, RiskCategoryDestructive, RiskCategorySecurity:
		return true
	default:
		return false
	}
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
