package schema

import (
	"entgo.io/ent"
)

// RolePermission 定义角色与权限点的关联模型。
type RolePermission struct {
	ent.Schema
}

// Mixin 返回角色权限关联复用的表元数据与字段定义。
func (RolePermission) Mixin() []ent.Mixin {
	return []ent.Mixin{
		associationRelationMixin{
			table: "role_permissions",
			left:  "role_id",
			right: "permission_id",
		},
	}
}

// Edges 返回角色权限关联的关系定义。
func (RolePermission) Edges() []ent.Edge {
	return associationRelationEdges(
		associationEdgeSpec{name: "role", entityType: Role.Type, ref: "role_permissions", field: "role_id"},
		associationEdgeSpec{name: "permission", entityType: Permission.Type, ref: "role_permissions", field: "permission_id"},
	)
}
