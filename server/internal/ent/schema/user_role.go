package schema

import (
	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
)

// UserRole 定义用户与角色的关联模型。
//
// 这里显式保留独立表，确保后续插件可以在不泄漏 ORM 细节的前提下演进附加元数据或审计字段。
type UserRole struct {
	ent.Schema
}

// Annotations 返回 user_roles 表名映射。
func (UserRole) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_roles"},
	}
}

// Fields 返回用户角色关联字段定义。
func (UserRole) Fields() []ent.Field {
	return associationRelationFields("user_id", "role_id")
}

// Edges 返回用户角色关联的关系定义。
func (UserRole) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("user_roles").
			Field("user_id").
			Required().
			Unique(),
		edge.From("role", Role.Type).
			Ref("user_roles").
			Field("role_id").
			Required().
			Unique(),
	}
}

// Indexes 返回用户角色关联的唯一约束与辅助索引。
func (UserRole) Indexes() []ent.Index {
	return associationIndexes("user_id", "role_id")
}
