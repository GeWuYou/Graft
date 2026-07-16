package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthCredential 定义一个用户资料对应的 auth 密码凭据。
type AuthCredential struct {
	ent.Schema
}

// Annotations 返回 auth_credentials 表映射及数据库注释配置。
func (AuthCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "auth_credentials"},
		schema.Comment("认证凭据表（认证模块）"),
		entsql.WithComments(true),
	}
}

// Fields 返回凭据字段；user_id 是由 user 模块拥有的稳定外部身份标识。
func (AuthCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("user_id").
			Comment("关联用户资料的稳定标识").
			Immutable(),
		field.String("password_hash").
			Comment("密码哈希值").
			Sensitive().
			Optional().
			Nillable(),
		field.Bool("must_change_password").
			Comment("是否必须在下次登录后修改密码").
			Default(false),
		field.Time("password_changed_at").
			Comment("最近一次修改密码时间").
			Optional().
			Nillable(),
		field.Time("created_at").
			Comment("凭据创建时间").
			Immutable().
			Default(time.Now),
		field.Time("updated_at").
			Comment("凭据最近更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes 为每个用户资料保留唯一凭据记录。
func (AuthCredential) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id").Unique()}
}
