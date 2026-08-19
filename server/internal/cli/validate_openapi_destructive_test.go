package cli

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestValidateOpenAPIDestructiveOperationsAcceptsSoftDelete(t *testing.T) {
	document := loadDestructiveValidationSpec(t, `
    "/api/users/{id}": {
      "delete": {
        "operationId": "deleteUser",
        "x-graft-destructive": {
          "kind": "resource_delete",
          "effect": "soft_delete",
          "execution": "synchronous",
          "retry": {"mode": "tombstone_idempotent"},
          "result": {"status": 204},
          "authorization": {"owner_check": false},
          "audit": {"required": true},
          "confirmation": {"required": true},
          "exposure": {"mcp": false}
        },
        "responses": {"204": {"description": "Deleted or already deleted."}}
      }
    }`)

	if err := validateOpenAPIDestructiveOperations(document); err != nil {
		t.Fatalf("validate soft delete metadata: %v", err)
	}
}

func TestValidateOpenAPIDestructiveOperationsAcceptsHardDeleteCommand(t *testing.T) {
	document := loadDestructiveValidationSpec(t, `
    "/api/audit/logs/{id}/deletions": {
      "post": {
        "operationId": "postAuditLogDeletion",
        "parameters": [{"name": "Idempotency-Key", "in": "header", "required": true, "schema": {"type": "string"}}],
        "x-graft-destructive": {
          "kind": "hard_delete",
          "effect": "hard_delete",
          "execution": "synchronous",
          "retry": {"mode": "idempotency_key"},
          "result": {"status": 200},
          "authorization": {"owner_check": false},
          "audit": {"required": true},
          "confirmation": {"required": true},
          "exposure": {"mcp": false}
        },
        "responses": {"200": {"description": "Deletion receipt."}}
      }
    }`)

	if err := validateOpenAPIDestructiveOperations(document); err != nil {
		t.Fatalf("validate hard delete metadata: %v", err)
	}
}

func TestValidateOpenAPIDestructiveOperationsRejectsOptionalHardDeleteIdempotencyKey(t *testing.T) {
	document := loadDestructiveValidationSpec(t, `
    "/api/audit/logs/{id}/deletions": {
      "post": {
        "operationId": "postAuditLogDeletion",
        "parameters": [{"name": "Idempotency-Key", "in": "header", "required": false, "schema": {"type": "string"}}],
        "x-graft-destructive": {
          "kind": "hard_delete",
          "effect": "hard_delete",
          "execution": "synchronous",
          "retry": {"mode": "idempotency_key"},
          "result": {"status": 200},
          "authorization": {"owner_check": false},
          "audit": {"required": true},
          "confirmation": {"required": true},
          "exposure": {"mcp": false}
        },
        "responses": {"200": {"description": "Deletion receipt."}}
      }
    }`)

	err := validateOpenAPIDestructiveOperations(document)
	if err == nil || !strings.Contains(err.Error(), "required Idempotency-Key header") {
		t.Fatalf("validate optional idempotency key error = %v", err)
	}
}

func TestValidateOpenAPIDestructiveOperationsAcceptsSharedPartialBatchResult(t *testing.T) {
	document := loadDestructiveValidationSpec(t, `
    "/api/users/deletions": {
      "post": {
        "operationId": "postUserDeletions",
        "x-graft-destructive": {
          "kind": "resource_delete",
          "effect": "soft_delete",
          "execution": "synchronous",
          "retry": {"mode": "tombstone_idempotent"},
          "result": {"status": 200},
          "authorization": {"owner_check": false},
          "audit": {"required": true},
          "confirmation": {"required": true},
          "exposure": {"mcp": false},
          "batch": {"mode": "partial", "max_items": 100}
        },
        "responses": {
          "200": {
            "description": "Batch result.",
            "content": {"application/json": {"schema": {"$ref": "#/components/schemas/EnvelopedDestructiveBatchResult"}}}
          }
        }
      }
    }`)

	if err := validateOpenAPIDestructiveOperations(document); err != nil {
		t.Fatalf("validate shared destructive batch result: %v", err)
	}
}

func TestValidateOpenAPIDestructiveOperationsRejectsModuleSpecificBatchResult(t *testing.T) {
	document := loadDestructiveValidationSpec(t, `
    "/api/users/deletions": {
      "post": {
        "operationId": "postUserDeletions",
        "x-graft-destructive": {
          "kind": "resource_delete",
          "effect": "soft_delete",
          "execution": "synchronous",
          "retry": {"mode": "tombstone_idempotent"},
          "result": {"status": 200},
          "authorization": {"owner_check": false},
          "audit": {"required": true},
          "confirmation": {"required": true},
          "exposure": {"mcp": false},
          "batch": {"mode": "partial", "max_items": 100}
        },
        "responses": {"200": {"description": "Module-specific result."}}
      }
    }`)

	err := validateOpenAPIDestructiveOperations(document)
	if err == nil || !strings.Contains(err.Error(), "EnvelopedDestructiveBatchResult") {
		t.Fatalf("validate module-specific batch result error = %v", err)
	}
}

func TestValidateOpenAPIDestructiveOperationsRejectsSynchronousExternalDestroy(t *testing.T) {
	document := loadDestructiveValidationSpec(t, `
    "/api/containers/{id}/remove": {
      "post": {
        "operationId": "postContainerRemove",
        "x-graft-destructive": {
          "kind": "external_destroy",
          "effect": "external_side_effect",
          "execution": "synchronous",
          "retry": {"mode": "task_receipt"},
          "result": {"status": 200},
          "authorization": {"owner_check": true},
          "audit": {"required": true},
          "confirmation": {"required": true},
          "exposure": {"mcp": false}
        },
        "responses": {"200": {"description": "Removed."}}
      }
    }`)

	err := validateOpenAPIDestructiveOperations(document)
	if err == nil || !strings.Contains(err.Error(), "execution=asynchronous") {
		t.Fatalf("validate synchronous external destroy error = %v", err)
	}
}

func TestValidateOpenAPIDestructiveOperationsRejectsMCPExposureDrift(t *testing.T) {
	document := loadDestructiveValidationSpec(t, `
    "/api/users/{id}": {
      "delete": {
        "operationId": "deleteUser",
        "x-graft-mcp": {},
        "x-graft-destructive": {
          "kind": "resource_delete",
          "effect": "soft_delete",
          "execution": "synchronous",
          "retry": {"mode": "tombstone_idempotent"},
          "result": {"status": 204},
          "authorization": {"owner_check": false},
          "audit": {"required": true},
          "confirmation": {"required": true},
          "exposure": {"mcp": false}
        },
        "responses": {"204": {"description": "Deleted."}}
      }
    }`)

	err := validateOpenAPIDestructiveOperations(document)
	if err == nil || !strings.Contains(err.Error(), "does not match x-graft-mcp") {
		t.Fatalf("validate MCP exposure drift error = %v", err)
	}
}

func loadDestructiveValidationSpec(t *testing.T, paths string) *openapi3.T {
	t.Helper()
	spec := `{
  "openapi": "3.1.0",
  "info": {"title": "test", "version": "1"},
  "x-graft-destructive-schema": {},
  "components": {"schemas": {"EnvelopedDestructiveBatchResult": {"type": "object"}}},
  "paths": {` + paths + `}
}`
	document, err := openapi3.NewLoader().LoadFromData([]byte(spec))
	if err != nil {
		t.Fatalf("load destructive validation spec: %v", err)
	}
	return document
}
