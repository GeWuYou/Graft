package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"

	rbacschema "graft/server/plugins/rbac/ent/schema"
)

// Permission 保留 internal/ent 的兼容引用面，真正 schema 真值由 rbac 插件拥有。
type Permission struct {
	ent.Schema
}

// Annotations 转发到 rbac 插件拥有的 schema 真值。
func (Permission) Annotations() []schema.Annotation {
	return rbacschema.Permission{}.Annotations()
}

// Fields 转发到 rbac 插件拥有的 schema 真值。
func (Permission) Fields() []ent.Field {
	return rbacschema.Permission{}.Fields()
}

// Edges 转发到 rbac 插件拥有的 schema 真值。
func (Permission) Edges() []ent.Edge {
	return rbacschema.Permission{}.Edges()
}
