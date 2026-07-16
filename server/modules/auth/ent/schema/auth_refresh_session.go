package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthRefreshSession 定义 auth 所拥有的 refresh token 会话生命周期记录。
type AuthRefreshSession struct {
	ent.Schema
}

// Annotations 返回 auth_refresh_sessions 表映射及数据库注释配置。
func (AuthRefreshSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "auth_refresh_sessions"},
		schema.Comment("认证刷新令牌会话表（认证模块）"),
		entsql.WithComments(true),
	}
}

// Fields 返回 refresh session 生命周期字段。
func (AuthRefreshSession) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("user_id").
			Comment("关联用户资料的稳定标识").
			Immutable(),
		field.String("token_id").
			Comment("刷新令牌唯一标识").
			NotEmpty().
			Unique().
			Immutable(),
		field.Time("expires_at").
			Comment("刷新令牌失效时间").
			Immutable(),
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

// Indexes 返回用户会话生命周期查询所需的索引。
func (AuthRefreshSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("expires_at"),
	}
}
