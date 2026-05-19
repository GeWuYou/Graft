package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"

	rbacschema "graft/server/plugins/rbac/ent/schema"
)

// Role 保留 internal/ent 的兼容引用面，真正 schema 真值由 rbac 插件拥有。
type Role struct {
	ent.Schema
}

// Annotations 转发到 rbac 插件拥有的 schema 真值。
func (Role) Annotations() []schema.Annotation {
	return rbacschema.Role{}.Annotations()
}

// Fields 转发到 rbac 插件拥有的 schema 真值。
func (Role) Fields() []ent.Field {
	return rbacschema.Role{}.Fields()
}

// Edges 转发到 rbac 插件拥有的 schema 真值。
func (Role) Edges() []ent.Edge {
	return rbacschema.Role{}.Edges()
}
