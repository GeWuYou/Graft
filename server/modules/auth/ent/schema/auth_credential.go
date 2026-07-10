package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthCredential defines the auth-owned password credential for one user profile.
type AuthCredential struct {
	ent.Schema
}

// Annotations returns the explicit auth_credentials table mapping and comment settings.
func (AuthCredential) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "auth_credentials"},
		schema.Comment("认证凭据表（认证模块）"),
		entsql.WithComments(true),
	}
}

// Fields returns the credential fields. user_id is a stable external identity owned by user.
func (AuthCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("user_id").
			Comment("关联用户资料的稳定标识"),
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

// Indexes keeps one credential record for each user profile.
func (AuthCredential) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id").Unique()}
}
