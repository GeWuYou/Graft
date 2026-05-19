package schema

import (
	"entgo.io/ent"

	rbacschema "graft/server/plugins/rbac/ent/schema"
)

// RolePermission 保留 internal/ent 的兼容引用面，真正 schema 真值由 rbac 插件拥有。
type RolePermission struct {
	ent.Schema
}

// Mixin 转发到 rbac 插件拥有的 schema 真值。
func (RolePermission) Mixin() []ent.Mixin {
	return rbacschema.RolePermission{}.Mixin()
}

// Edges 转发到 rbac 插件拥有的 schema 真值。
func (RolePermission) Edges() []ent.Edge {
	return rbacschema.RolePermission{}.Edges()
}
