package schema

import (
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Role defines the RBAC plugin's persistence model for roles.
type Role struct {
	ent.Schema
}

// Annotations returns the explicit roles table mapping.
func (Role) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "roles"},
	}
}

// Fields returns the role field definitions.
func (Role) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			Unique(),
		field.String("display").
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable(),
		field.Bool("builtin").
			Default(false),
		field.Time("created_at").
			Immutable().
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges returns the role relation definitions owned by the RBAC plugin.
func (Role) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user_roles", UserRole.Type),
		edge.To("role_permissions", RolePermission.Type),
	}
}
