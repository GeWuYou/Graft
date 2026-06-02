package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// RefreshSession defines the user module's refresh-session persistence model.
type RefreshSession struct {
	ent.Schema
}

// Annotations returns the explicit refresh_sessions table mapping.
func (RefreshSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "refresh_sessions"},
		schema.Comment("刷新令牌会话表（用户插件）"),
		entsql.WithComments(true),
	}
}

// Fields returns the refresh-session field definitions.
func (RefreshSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("所属用户 ID"),
		field.String("token_id").
			Comment("刷新令牌唯一标识").
			NotEmpty().
			Unique(),
		field.Time("expires_at").
			Comment("过期时间"),
		field.Time("revoked_at").
			Comment("撤销时间，为空表示未撤销").
			Optional().
			Nillable(),
		field.String("replaced_by_token_id").
			Comment("轮换后替换当前令牌的新令牌 ID").
			Optional().
			Nillable(),
		field.Time("created_at").
			Comment("创建时间").
			Immutable().
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges returns the refresh-session-to-user relation definitions.
func (RefreshSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("refresh_sessions").
			Field("user_id").
			Required().
			Unique(),
	}
}

// Indexes returns the secondary indexes required by the user module's auth flows.
func (RefreshSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("expires_at"),
	}
}
