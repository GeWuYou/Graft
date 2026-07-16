// Package menu 存放后端声明的导航元数据，供后续壳层装配使用。
package menu

import (
	"fmt"
	"sort"
	"strings"
)

// NodeKind 区分不可导航的领域分组和可导航的页面条目。
type NodeKind string

const (
	// NodeKindGroup 表示不可直接导航的领域分组。
	NodeKindGroup NodeKind = "group"
	// NodeKindEntry 表示可导航的页面条目。
	NodeKindEntry NodeKind = "entry"

	visitUnseen uint8 = iota
	visitActive
	visitDone

	domainOrderApplication    = 10
	domainOrderInfrastructure = 20
	domainOrderBuild          = 30
	domainOrderResources      = 40
	domainOrderObservability  = 50
	domainOrderSecurity       = 60
	domainOrderPlatform       = 70

	// RuntimeSectionKey 是仅影响侧边栏展示的运行时分区键。
	RuntimeSectionKey = "runtime"
	// AccessControlSectionKey 是仅影响侧边栏展示的访问控制分区键。
	AccessControlSectionKey = "access-control"
	// AccessControlSectionTitleKey 是访问控制分区标题的本地化键。
	AccessControlSectionTitleKey = "menu.section.access_control"
)

// Item 表示一个由后端声明的菜单项。
type Item struct {
	// Code 是菜单项的稳定后端标识，用于后续增量对比、去重或权限联动。
	Code       string
	ParentCode string
	Kind       NodeKind
	Title      string
	TitleKey   string
	// SectionKey 可选地把菜单项归入仅影响侧边栏展示的分区；它不是菜单节点、路由、权限或身份标识。
	SectionKey string
	// SectionTitleKey 是 SectionKey 对应的稳定本地化键。
	SectionTitleKey string
	Path            string
	Icon            string
	// Order 是后端声明的 canonical 导航排序值；数值越小越靠前。
	Order int
	// Permission 记录访问该菜单所需的后端权限编码；留空表示暂不做权限门控。
	Permission string
	// VisibleWhenConfigEnabled 是后端 bootstrap 菜单裁剪使用的内部 feature gate。
	//
	// 该字段不进入 bootstrap menu wire shape；System Config 只控制模块内部业务能力，
	// 不代表 backend module 加载/卸载 authority。
	VisibleWhenConfigEnabled string
	// Module 标记菜单归属的模块，便于启动诊断与后续按模块裁剪导航。
	Module string
}

// Registry 按注册顺序保存菜单声明，保证模块装配结果稳定可预期。
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
	if item.Kind == "" {
		item.Kind = NodeKindEntry
	}
	r.items = append(r.items, item)
}

// Items 返回当前已注册菜单集合的副本。
//
// 返回顺序与模块注册顺序一致，便于上层在生成导航时保持稳定输出。
func (r *Registry) Items() []Item {
	items := make([]Item, len(r.items))
	copy(items, r.items)
	return items
}

// Validate 在 bootstrap 输出菜单前拒绝结构不完整或存在环的导航图。
func (r *Registry) Validate() error {
	items := r.Items()
	byCode := make(map[string]Item, len(items))
	for _, item := range items {
		if err := validateItem(item, byCode); err != nil {
			return err
		}
		item.Code = strings.TrimSpace(item.Code)
		byCode[item.Code] = item
	}
	if err := validateParents(byCode); err != nil {
		return err
	}
	return validateCycles(byCode)
}

// validateItem 验证菜单项的代码唯一性、节点类型、路径及分区元数据配置。
func validateItem(item Item, existing map[string]Item) error {
	code := strings.TrimSpace(item.Code)
	if code == "" {
		return fmt.Errorf("menu item code is required")
	}
	if _, exists := existing[code]; exists {
		return fmt.Errorf("duplicate menu code %q", code)
	}
	kind := item.Kind
	if kind == "" {
		kind = NodeKindEntry
	}
	switch kind {
	case NodeKindGroup:
		return validateGroupItem(code, item)
	case NodeKindEntry:
		if strings.TrimSpace(item.Path) == "" {
			return fmt.Errorf("menu entry %q must declare a path", code)
		}
		return validateSectionMetadata(code, item)
	default:
		return fmt.Errorf("menu item %q has invalid kind %q", code, kind)
	}
}

// validateGroupItem 校验菜单分组不声明路径，且分区元数据保持一致。
func validateGroupItem(code string, item Item) error {
	if strings.TrimSpace(item.Path) != "" {
		return fmt.Errorf("menu group %q must not declare a path", code)
	}
	return validateSectionMetadata(code, item)
}

// validateSectionMetadata 校验菜单项的侧边栏分区键和分区标题本地化键是否同时声明。
// 两者仅声明其一时返回错误，否则返回 nil。
func validateSectionMetadata(code string, item Item) error {
	sectionKey := strings.TrimSpace(item.SectionKey)
	titleKey := strings.TrimSpace(item.SectionTitleKey)
	if (sectionKey == "") != (titleKey == "") {
		return fmt.Errorf("menu item %q must declare both section key and section title key", code)
	}
	return nil
}

// validateParents 校验每个菜单项的 parent 存在且确实是分组节点。
func validateParents(items map[string]Item) error {
	for code, item := range items {
		parent := strings.TrimSpace(item.ParentCode)
		if parent == "" {
			continue
		}
		parentItem, exists := items[parent]
		if !exists {
			return fmt.Errorf("menu item %q references unknown parent %q", code, parent)
		}
		if parentItem.Kind != NodeKindGroup {
			return fmt.Errorf("menu item %q parent %q is not a group", code, parent)
		}
	}
	return nil
}

// validateCycles 检查菜单层级是否存在环，并返回首个检测到的环节点。
func validateCycles(items map[string]Item) error {
	state := make(map[string]uint8, len(items))
	var visit func(string) error
	visit = func(code string) error {
		if state[code] == visitActive {
			return fmt.Errorf("menu graph contains a cycle at %q", code)
		}
		if state[code] == visitDone {
			return nil
		}
		state[code] = visitActive
		if parent := strings.TrimSpace(items[code].ParentCode); parent != "" {
			if err := visit(parent); err != nil {
				return err
			}
		}
		state[code] = visitDone
		return nil
	}
	codes := make([]string, 0, len(items))
	for code := range items {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if err := visit(code); err != nil {
			return err
		}
	}
	return nil
}

// RegisterDomainGroups 注册预定义的顶层领域菜单分组；当注册表为 nil 时忽略操作。
func RegisterDomainGroups(r *Registry) {
	if r == nil {
		return
	}
	for _, item := range []Item{
		{Code: "domain.application", Kind: NodeKindGroup, TitleKey: "menu.domain.application.title", Icon: "application", Order: domainOrderApplication, Module: "core.navigation"},
		{Code: "domain.infrastructure", Kind: NodeKindGroup, TitleKey: "menu.domain.infrastructure.title", Icon: "infrastructure", Order: domainOrderInfrastructure, Module: "core.navigation"},
		{Code: "domain.build", Kind: NodeKindGroup, TitleKey: "menu.domain.build.title", Icon: "build", Order: domainOrderBuild, Module: "core.navigation"},
		{Code: "domain.resources", Kind: NodeKindGroup, TitleKey: "menu.domain.resources.title", Icon: "resources", Order: domainOrderResources, Module: "core.navigation"},
		{Code: "domain.observability", Kind: NodeKindGroup, TitleKey: "menu.domain.observability.title", Icon: "observability", Order: domainOrderObservability, Module: "core.navigation"},
		{Code: "domain.security", Kind: NodeKindGroup, TitleKey: "menu.domain.security.title", Icon: "security", Order: domainOrderSecurity, Module: "core.navigation"},
		{Code: "domain.platform", Kind: NodeKindGroup, TitleKey: "menu.domain.platform.title", Icon: "platform", Order: domainOrderPlatform, Module: "core.navigation"},
	} {
		r.Register(item)
	}
}
