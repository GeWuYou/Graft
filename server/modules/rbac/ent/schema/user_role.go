// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"entgo.io/ent"
)

// UserRole defines the RBAC-owned user-to-role join model.
//
// The boundary intentionally stays at user_id / role_id identifiers so RBAC
// does not own or import the user module's Ent schema internals.
type UserRole struct {
	ent.Schema
}

// Mixin returns the shared join-table metadata and fields.
func (UserRole) Mixin() []ent.Mixin {
	return []ent.Mixin{
		associationRelationMixin{
			table: "user_roles",
			left:  "user_id",
			right: "role_id",
		},
	}
}

// Edges returns only the RBAC-owned role edge; user lookup stays outside Ent edges.
func (UserRole) Edges() []ent.Edge {
	return []ent.Edge{
		associationRelationEdge(
			associationEdgeSpec{name: "role", entityType: Role.Type, ref: "user_roles", field: "role_id"},
		),
	}
}
