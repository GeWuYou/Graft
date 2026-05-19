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

// RefreshSession defines the user plugin's refresh-session persistence model.
type RefreshSession struct {
	ent.Schema
}

// Annotations returns the explicit refresh_sessions table mapping.
func (RefreshSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "refresh_sessions"},
	}
}

// Fields returns the refresh-session field definitions.
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

// Indexes returns the secondary indexes required by the user plugin's auth flows.
func (RefreshSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("expires_at"),
	}
}
