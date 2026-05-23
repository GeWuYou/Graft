package schema

import (
	"regexp"
	"time"

	"entgo.io/ent"
	entsql "entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Permission defines the RBAC plugin's persistence model for permissions.
type Permission struct {
	ent.Schema
}

// Annotations returns the explicit permissions table mapping.
func (Permission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "permissions"},
	}
}

// Fields returns the permission field definitions.
func (Permission) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			NotEmpty().
			Match(regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)+$`)).
			Unique(),
		field.String("display").
			NotEmpty(),
		field.String("description").
			Optional().
			Nillable(),
		field.String("category").
			NotEmpty().
			Default("api"),
		field.Time("created_at").
			Immutable().
			Default(time.Now),
		field.Uint64("created_by").
			Immutable().
			Default(0),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Uint64("updated_by").
			Default(0),
		field.Int64("deleted_at").
			Default(0),
		field.Uint64("deleted_by").
			Default(0),
	}
}

// Edges returns the permission relation definitions owned by the RBAC plugin.
func (Permission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("role_permissions", RolePermission.Type),
	}
}
