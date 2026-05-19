package schema

import (
	"entgo.io/ent"

	rbacschema "graft/server/plugins/rbac/ent/schema"
)

// UserRole 保留 internal/ent 的兼容引用面，真正 schema 真值由 rbac 插件拥有。
type UserRole struct {
	ent.Schema
}

// Mixin 转发到 rbac 插件拥有的 schema 真值。
func (UserRole) Mixin() []ent.Mixin {
	return rbacschema.UserRole{}.Mixin()
}

// Edges 转发到 rbac 插件拥有的 schema 真值。
func (UserRole) Edges() []ent.Edge {
	return rbacschema.UserRole{}.Edges()
}
