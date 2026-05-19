package schema

import (
	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/mixin"
)

// UserRole 定义用户与角色的关联模型。
//
// 这里显式保留独立表，确保后续插件可以在不泄漏 ORM 细节的前提下演进附加元数据或审计字段。
type UserRole struct {
	ent.Schema
}

type associationRelationMixin struct {
	mixin.Schema
	table string
	left  string
	right string
}

func (m associationRelationMixin) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: m.table},
	}
}

func (m associationRelationMixin) Fields() []ent.Field {
	return associationRelationFields(m.left, m.right)
}

func (m associationRelationMixin) Indexes() []ent.Index {
	return associationIndexes(m.left, m.right)
}

type associationEdgeSpec struct {
	name       string
	entityType any
	ref        string
	field      string
}

func associationRelationEdges(left associationEdgeSpec, right associationEdgeSpec) []ent.Edge {
	return []ent.Edge{
		associationRelationEdge(left),
		associationRelationEdge(right),
	}
}

func associationRelationEdge(spec associationEdgeSpec) ent.Edge {
	return edge.From(spec.name, spec.entityType).
		Ref(spec.ref).
		Field(spec.field).
		Required().
		Unique()
}

// Mixin 返回用户角色关联复用的表元数据与字段定义。
func (UserRole) Mixin() []ent.Mixin {
	return []ent.Mixin{
		associationRelationMixin{
			table: "user_roles",
			left:  "user_id",
			right: "role_id",
		},
	}
}

// Edges 返回用户角色关联的关系定义。
func (UserRole) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").
			Required().
			Unique(),
		associationRelationEdge(
			associationEdgeSpec{name: "role", entityType: Role.Type, ref: "user_roles", field: "role_id"},
		),
	}
}
