package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User defines the user plugin's persistence model for users.
type User struct {
	ent.Schema
}

// Fields returns the user field definitions.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			NotEmpty().
			Unique(),
		field.String("display").
			NotEmpty(),
		field.String("status").
			NotEmpty().
			Default("enabled"),
		field.String("password_hash").
			Sensitive().
			Optional().
			Nillable(),
		field.Bool("must_change_password").
			Default(false),
		field.Time("password_changed_at").
			Optional().
			Nillable(),
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

// Edges returns the user-to-refresh-session relation definitions.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("refresh_sessions", RefreshSession.Type),
	}
}
