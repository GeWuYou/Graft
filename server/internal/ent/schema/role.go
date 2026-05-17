package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Role 定义 RBAC 角色的持久化模型。
type Role struct {
	ent.Schema
}

// Annotations 返回 roles 表名映射。
func (Role) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "roles"},
	}
}

// Fields 返回角色字段定义。
func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Unique(),
		field.String("display").
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable(),
		field.Bool("builtin").
			Default(false),
		field.Time("created_at").
			Immutable().
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges 返回角色相关的关系定义。
func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user_roles", UserRole.Type),
		edge.To("role_permissions", RolePermission.Type),
	}
}
