# RBAC generated bindings

Run `go generate ./internal/contract/openapi/rbac` from `server` to rebuild the
models and Gin strict wrappers selected by `oapi-codegen.yaml`. The selected
operations include role and permission management, user-role bindings, and
the two saved-view surfaces.

The RBAC module owns the mount paths, operation-specific permissions and
business validation. Its responses use the shared `httpx` envelope.
