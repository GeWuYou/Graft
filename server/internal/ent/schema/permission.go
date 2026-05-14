package schema

import (
	"regexp"
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Permission 定义 RBAC 权限点的持久化模型。
type Permission struct {
	ent.Schema
}

// Annotations 返回 permissions 表名映射。
func (Permission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "permissions"},
	}
}

// Fields 返回权限点字段定义。
func (Permission) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			NotEmpty().
			Match(regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)+$`)).
			Unique(),
		field.String("display").
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable(),
		field.Time("created_at").
			Immutable().
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges 返回权限点相关的关系定义。
func (Permission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("role_permissions", RolePermission.Type),
	}
}
