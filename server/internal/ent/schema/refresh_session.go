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

// RefreshSession 定义刷新会话的持久化模型。
//
// 该模型只负责承载 refresh token 生命周期所需的最小状态，不在 schema 层引入
// 登录策略、设备风控或权限缓存等更高层业务语义。
type RefreshSession struct {
	ent.Schema
}

// Annotations 返回 refresh_sessions 表名映射。
func (RefreshSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "refresh_sessions"},
	}
}

// Fields 返回刷新会话的字段定义。
func (RefreshSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id"),
		field.String("token_id").
			NotEmpty().
			Unique(),
		field.Time("expires_at"),
		field.Time("revoked_at").
			Optional().
			Nillable(),
		field.String("replaced_by_token_id").
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

// Edges 返回刷新会话与用户之间的关系定义。
func (RefreshSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("refresh_sessions").
			Field("user_id").
			Required().
			Unique(),
	}
}

// Indexes 返回刷新会话需要的辅助索引。
func (RefreshSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("expires_at"),
	}
}
