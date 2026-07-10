package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// User defines the user module's persistence model for users.
type User struct {
	ent.Schema
}

// Annotations returns the explicit users table mapping and comment settings.
func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "users"},
		schema.Comment("用户基础信息表（用户模块）"),
		entsql.WithComments(true),
	}
}

// Fields returns the user field definitions.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			Comment("用户名，用于登录和唯一标识").
			NotEmpty().
			Unique(),
		field.String("display").
			Comment("显示名称，用于后台展示").
			NotEmpty(),
		field.String("status").
			Comment("状态：enabled 启用，disabled 禁用").
			NotEmpty().
			Default("enabled"),
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
