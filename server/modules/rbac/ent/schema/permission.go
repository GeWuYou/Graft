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

// Permission defines the RBAC plugin's persistence model for permissions.
type Permission struct {
	ent.Schema
}

// Annotations returns the explicit permissions table mapping.
func (Permission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "permissions"},
		schema.Comment("权限点信息表（RBAC 插件）"),
		entsql.WithComments(true),
	}
}

// Fields returns the permission field definitions.
func (Permission) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("权限点编码，采用点分层级格式").
			NotEmpty().
			Match(regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)+$`)).
			Unique(),
		field.String("display").
			Comment("权限点显示名称").
			NotEmpty(),
		field.String("description").
			Comment("权限点描述").
			Optional().
			Nillable(),
		field.String("category").
			Comment("权限类别：api 表示接口权限").
			NotEmpty().
			Default("api"),
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
		field.Int64("deleted_at").
			Comment("软删除时间戳，0 表示未删除").
			Default(0),
		field.Uint64("deleted_by").
			Comment("删除人用户 ID，0 表示未删除").
			Default(0),
	}
}

// Edges returns the permission relation definitions owned by the RBAC plugin.
func (Permission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("role_permissions", RolePermission.Type),
	}
}
