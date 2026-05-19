package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User 定义 MVP 用户能力的持久化模型。
type User struct {
	ent.Schema
}

// Fields 返回用户字段定义。
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			NotEmpty().
			Unique(),
		field.String("display").
			NotEmpty(),
		field.String("password_hash").
			Sensitive().
			Optional().
			Nillable(),
		// must_change_password 标识该用户下次登录后是否必须先完成改密。
		// 默认管理员初始化和后续密码重置会把它设为 true；用户成功改密后再清回 false。
		field.Bool("must_change_password").
			Default(false),
		field.Time("password_changed_at").
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

// Edges 返回用户相关的关系定义。
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("refresh_sessions", RefreshSession.Type),
	}
}
