package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthRefreshSession defines the auth-owned refresh-token session lifecycle record.
type AuthRefreshSession struct {
	ent.Schema
}

// Annotations returns the explicit auth_refresh_sessions table mapping and comment settings.
func (AuthRefreshSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "auth_refresh_sessions"},
		schema.Comment("认证刷新令牌会话表（认证模块）"),
		entsql.WithComments(true),
	}
}

// Fields returns the refresh-session lifecycle fields.
func (AuthRefreshSession) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("user_id").
			Comment("关联用户资料的稳定标识"),
		field.String("token_id").
			Comment("刷新令牌唯一标识").
			NotEmpty().
			Unique(),
		field.Time("expires_at").
			Comment("刷新令牌失效时间"),
		field.Time("revoked_at").
			Comment("会话撤销时间，为空表示仍有效").
			Optional().
			Nillable(),
		field.String("replaced_by_token_id").
			Comment("令牌轮换后替代当前会话的新令牌标识").
			Optional().
			Nillable(),
		field.Time("created_at").
			Comment("会话创建时间").
			Immutable().
			Default(time.Now),
		field.Time("updated_at").
			Comment("会话最近更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes returns indexes for user-session lifecycle queries.
func (AuthRefreshSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("expires_at"),
	}
}
