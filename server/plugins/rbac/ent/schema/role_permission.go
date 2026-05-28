package schema

import (
	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/mixin"
)

// RolePermission defines the RBAC plugin's role-to-permission join model.
type RolePermission struct {
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
		entsqlAnnotation(m.table),
		entsql.WithComments(true),
		schema.Comment(associationTableComment(m.table)),
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

func entsqlAnnotation(table string) schema.Annotation {
	return entsql.Annotation{Table: table}
}

// Mixin returns the shared join-table metadata and fields.
func (RolePermission) Mixin() []ent.Mixin {
	return []ent.Mixin{
		associationRelationMixin{
			table: "role_permissions",
			left:  "role_id",
			right: "permission_id",
		},
	}
}

// Edges returns the role-permission join edges.
func (RolePermission) Edges() []ent.Edge {
	return associationRelationEdges(
		associationEdgeSpec{name: "role", entityType: Role.Type, ref: "role_permissions", field: "role_id"},
		associationEdgeSpec{name: "permission", entityType: Permission.Type, ref: "role_permissions", field: "permission_id"},
	)
}
