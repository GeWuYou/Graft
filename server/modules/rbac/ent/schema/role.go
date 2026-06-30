package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Role defines the RBAC module's persistence model for roles.
type Role struct {
	ent.Schema
}

// Annotations returns the explicit roles table mapping.
func (Role) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "roles"},
		schema.Comment("角色信息表（RBAC 模块）"),
		entsql.WithComments(true),
	}
}

// Fields returns the role field definitions.
func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("角色标识名称，用于唯一识别角色").
			NotEmpty().
			Unique(),
		field.String("display").
			Comment("角色显示名称").
			NotEmpty(),
		field.String("description").
			Comment("角色描述").
			Optional().
			Nillable(),
		field.Bool("builtin").
			Comment("是否为系统内置角色").
			Default(false),
		field.Time("created_at").
			Comment("创建时间").
			Immutable().
			Default(time.Now),
		field.Uint64("created_by").
			Comment("创建人用户 ID，0 表示系统").
			Immutable().
			Default(0),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Uint64("updated_by").
			Comment("最后更新人用户 ID，0 表示系统").
			Default(0),
		field.Int64("disabled_at").
			Comment("禁用时间戳，0 表示启用").
			Default(0),
		field.Int64("deleted_at").
			Comment("软删除时间戳，0 表示未删除").
			Default(0),
		field.Uint64("deleted_by").
			Comment("删除人用户 ID，0 表示未删除").
			Default(0),
	}
}

// Edges returns the role relation definitions owned by the RBAC module.
func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user_roles", UserRole.Type),
		edge.To("role_permissions", RolePermission.Type),
	}
}
