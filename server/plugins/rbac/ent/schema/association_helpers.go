package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

func associationRelationFields(left string, right string) []ent.Field {
	return []ent.Field{
		field.Int(left),
		field.Int(right),
		associationCreatedAtField(),
	}
}

func associationCreatedAtField() ent.Field {
	return field.Time("created_at").
		Immutable().
		Default(time.Now)
}

func associationIndexes(left string, right string) []ent.Index {
	return []ent.Index{
		index.Fields(left, right).
			Unique(),
		index.Fields(right),
	}
}
