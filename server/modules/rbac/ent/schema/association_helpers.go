package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

func associationRelationFields(left string, right string) []ent.Field {
	return []ent.Field{
		field.Int(left).
			Comment(associationFieldComment(left)),
		field.Int(right).
			Comment(associationFieldComment(right)),
		associationCreatedAtField(),
	}
}

func associationCreatedAtField() ent.Field {
	return field.Time("created_at").
		Comment("创建时间").
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

func associationTableComment(table string) string {
	switch table {
	case "user_roles":
		return "用户与角色关联表（RBAC 模块）"
	case "role_permissions":
		return "角色与权限关联表（RBAC 模块）"
	default:
		return "关联表"
	}
}

func associationFieldComment(name string) string {
	switch name {
	case "user_id":
		return "用户 ID"
	case "role_id":
		return "角色 ID"
	case "permission_id":
		return "权限 ID"
	default:
		return "关联字段"
	}
}
