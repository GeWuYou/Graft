package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	destructiveExtensionName       = "x-graft-destructive"
	destructiveExtensionSchemaName = "x-graft-destructive-schema"
)

type destructiveOperationMetadata struct {
	Kind          string                           `json:"kind"`
	Effect        string                           `json:"effect"`
	Execution     string                           `json:"execution"`
	Retry         destructiveRetryMetadata         `json:"retry"`
	Result        destructiveResultMetadata        `json:"result"`
	Authorization destructiveAuthorizationMetadata `json:"authorization"`
	Audit         destructiveRequiredMetadata      `json:"audit"`
	Confirmation  destructiveRequiredMetadata      `json:"confirmation"`
	Exposure      destructiveExposureMetadata      `json:"exposure"`
	Batch         *destructiveBatchMetadata        `json:"batch,omitempty"`
}

type destructiveRetryMetadata struct {
	Mode string `json:"mode"`
}

type destructiveResultMetadata struct {
	Status int `json:"status"`
}

type destructiveAuthorizationMetadata struct {
	OwnerCheck *bool `json:"owner_check"`
}

type destructiveRequiredMetadata struct {
	Required *bool `json:"required"`
}

type destructiveExposureMetadata struct {
	MCP *bool `json:"mcp"`
}

type destructiveBatchMetadata struct {
	Mode     string `json:"mode"`
	MaxItems int    `json:"max_items"`
}

type destructiveShape struct {
	effect       string
	execution    string
	retryMode    string
	method       string
	resultStatus int
}

// validateOpenAPIDestructiveOperations 校验 canonical OpenAPI 中已声明的消除性操作。
// 存量 operation 只有在运行时语义完成迁移后才声明 metadata；本校验拒绝 metadata 与 method、status、Task receipt 或 MCP 暴露发生漂移。
func validateOpenAPIDestructiveOperations(document *openapi3.T) error {
	if document == nil {
		return fmt.Errorf("destructive operation validation requires an OpenAPI document")
	}
	if _, ok := document.Extensions[destructiveExtensionSchemaName]; !ok {
		return fmt.Errorf("OpenAPI root is missing %s", destructiveExtensionSchemaName)
	}

	paths := slices.Collect(maps.Keys(document.Paths.Map()))
	sort.Strings(paths)
	for _, path := range paths {
		pathItem := document.Paths.Find(path)
		if pathItem == nil {
			return fmt.Errorf("OpenAPI path %q is unavailable during destructive operation validation", path)
		}
		methods := slices.Collect(maps.Keys(pathItem.Operations()))
		sort.Strings(methods)
		for _, method := range methods {
			operation := pathItem.GetOperation(method)
			if operation == nil || operation.Extensions == nil {
				continue
			}
			rawMetadata, declared := operation.Extensions[destructiveExtensionName]
			if !declared {
				continue
			}
			if err := validateOpenAPIDestructiveOperation(pathItem, operation, path, method, rawMetadata); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateOpenAPIDestructiveOperation(
	pathItem *openapi3.PathItem,
	operation *openapi3.Operation,
	path string,
	method string,
	rawMetadata any,
) error {
	metadata, err := decodeDestructiveOperationMetadata(rawMetadata)
	if err != nil {
		return destructiveOperationError(operation, method, path, "%v", err)
	}
	if err := validateDestructiveMetadataFields(metadata); err != nil {
		return destructiveOperationError(operation, method, path, "%v", err)
	}

	method = strings.ToUpper(method)
	if err := validateDestructiveKind(pathItem, operation, path, method, metadata); err != nil {
		return destructiveOperationError(operation, method, path, "%v", err)
	}
	if err := validateDestructiveResult(operation, metadata); err != nil {
		return destructiveOperationError(operation, method, path, "%v", err)
	}

	_, hasMCPMetadata := operation.Extensions["x-graft-mcp"]
	if *metadata.Exposure.MCP != hasMCPMetadata {
		return destructiveOperationError(
			operation,
			method,
			path,
			"exposure.mcp=%t does not match x-graft-mcp presence=%t",
			*metadata.Exposure.MCP,
			hasMCPMetadata,
		)
	}
	return nil
}

func decodeDestructiveOperationMetadata(rawMetadata any) (destructiveOperationMetadata, error) {
	payload, err := json.Marshal(rawMetadata)
	if err != nil {
		return destructiveOperationMetadata{}, fmt.Errorf("marshal %s metadata: %w", destructiveExtensionName, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var metadata destructiveOperationMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return destructiveOperationMetadata{}, fmt.Errorf("decode %s metadata: %w", destructiveExtensionName, err)
	}
	return metadata, nil
}

func validateDestructiveMetadataFields(metadata destructiveOperationMetadata) error {
	if err := validateDestructiveMetadataEnums(metadata); err != nil {
		return err
	}
	if err := validateDestructiveMetadataRequirements(metadata); err != nil {
		return err
	}
	return validateDestructiveBatchMetadata(metadata.Batch)
}

func validateDestructiveMetadataEnums(metadata destructiveOperationMetadata) error {
	if !isOneOf(metadata.Kind, "resource_delete", "relationship_remove", "credential_revoke", "lifecycle_termination", "hard_delete", "external_destroy") {
		return fmt.Errorf("kind %q is invalid", metadata.Kind)
	}
	if !isOneOf(metadata.Effect, "soft_delete", "hard_delete", "relationship_removal", "revocation", "external_side_effect") {
		return fmt.Errorf("effect %q is invalid", metadata.Effect)
	}
	if !isOneOf(metadata.Execution, "synchronous", "asynchronous") {
		return fmt.Errorf("execution %q is invalid", metadata.Execution)
	}
	if !isOneOf(metadata.Retry.Mode, "tombstone_idempotent", "idempotency_key", "task_receipt") {
		return fmt.Errorf("retry.mode %q is invalid", metadata.Retry.Mode)
	}
	if !isOneOf(metadata.Result.Status, http.StatusOK, http.StatusAccepted, http.StatusNoContent) {
		return fmt.Errorf("result.status %d is invalid", metadata.Result.Status)
	}
	return nil
}

func validateDestructiveMetadataRequirements(metadata destructiveOperationMetadata) error {
	if metadata.Authorization.OwnerCheck == nil {
		return fmt.Errorf("authorization.owner_check is required")
	}
	if metadata.Audit.Required == nil || !*metadata.Audit.Required {
		return fmt.Errorf("audit.required must be true")
	}
	if metadata.Confirmation.Required == nil {
		return fmt.Errorf("confirmation.required is required")
	}
	if metadata.Exposure.MCP == nil {
		return fmt.Errorf("exposure.mcp is required")
	}
	return nil
}

func validateDestructiveBatchMetadata(batch *destructiveBatchMetadata) error {
	if batch == nil {
		return nil
	}
	if !isOneOf(batch.Mode, "partial", "atomic") {
		return fmt.Errorf("batch.mode %q is invalid", batch.Mode)
	}
	if batch.MaxItems < 1 {
		return fmt.Errorf("batch.max_items must be positive")
	}
	return nil
}

func validateDestructiveKind(
	pathItem *openapi3.PathItem,
	operation *openapi3.Operation,
	path string,
	method string,
	metadata destructiveOperationMetadata,
) error {
	switch metadata.Kind {
	case "resource_delete":
		if metadata.Batch != nil {
			return requireDestructiveBatchShape(metadata, "soft_delete")
		}
		return requireDestructiveShape(metadata, method, destructiveShape{
			effect: "soft_delete", execution: "synchronous", retryMode: "tombstone_idempotent",
			method: http.MethodDelete, resultStatus: http.StatusNoContent,
		})
	case "relationship_remove":
		if metadata.Batch != nil {
			return requireDestructiveBatchShape(metadata, "relationship_removal")
		}
		return requireDestructiveShape(metadata, method, destructiveShape{
			effect: "relationship_removal", execution: "synchronous", retryMode: "tombstone_idempotent",
			method: http.MethodDelete, resultStatus: http.StatusNoContent,
		})
	case "hard_delete":
		return validateHardDeleteShape(pathItem, operation, path, method, metadata)
	case "external_destroy":
		return requireDestructiveShape(metadata, method, destructiveShape{
			effect: "external_side_effect", execution: "asynchronous", retryMode: "task_receipt",
			method: http.MethodPost, resultStatus: http.StatusAccepted,
		})
	}
	return nil
}

func validateHardDeleteShape(
	pathItem *openapi3.PathItem,
	operation *openapi3.Operation,
	path string,
	method string,
	metadata destructiveOperationMetadata,
) error {
	if metadata.Effect != "hard_delete" || method != http.MethodPost || metadata.Retry.Mode != "idempotency_key" {
		return fmt.Errorf("hard_delete requires effect=hard_delete, POST, and retry.mode=idempotency_key")
	}
	if !strings.HasSuffix(strings.TrimRight(path, "/"), "/deletions") {
		return fmt.Errorf("hard_delete path must end with /deletions")
	}
	if !hasRequiredHeaderParameter(pathItem, operation, "Idempotency-Key") {
		return fmt.Errorf("hard_delete requires a required Idempotency-Key header parameter")
	}
	return nil
}

func requireDestructiveBatchShape(metadata destructiveOperationMetadata, effect string) error {
	if metadata.Effect != effect || metadata.Execution != "synchronous" || metadata.Result.Status != http.StatusOK {
		return fmt.Errorf("batch requires effect=%s, execution=synchronous, and result.status=200", effect)
	}
	return nil
}

func requireDestructiveShape(
	metadata destructiveOperationMetadata,
	method string,
	required destructiveShape,
) error {
	if metadata.Effect != required.effect || metadata.Execution != required.execution || metadata.Retry.Mode != required.retryMode || method != required.method || metadata.Result.Status != required.resultStatus {
		return fmt.Errorf(
			"requires effect=%s, execution=%s, retry.mode=%s, method=%s, and result.status=%d",
			required.effect,
			required.execution,
			required.retryMode,
			required.method,
			required.resultStatus,
		)
	}
	return nil
}

func validateDestructiveResult(operation *openapi3.Operation, metadata destructiveOperationMetadata) error {
	response := operation.Responses.Status(metadata.Result.Status)
	if response == nil || response.Value == nil {
		return fmt.Errorf("declared result.status %d is missing from responses", metadata.Result.Status)
	}
	if metadata.Result.Status == http.StatusNoContent && len(response.Value.Content) != 0 {
		return fmt.Errorf("204 destructive response must not declare content")
	}
	if err := validateAsynchronousDestructiveResult(response.Value, metadata); err != nil {
		return err
	}
	return validateSynchronousDestructiveBatchResult(response.Value, metadata)
}

func validateAsynchronousDestructiveResult(response *openapi3.Response, metadata destructiveOperationMetadata) error {
	if metadata.Execution != "asynchronous" {
		return nil
	}
	if !responseUsesSchema(response, "enveloped-task-receipt") {
		return fmt.Errorf("asynchronous destructive response must use enveloped-task-receipt")
	}
	return nil
}

func validateSynchronousDestructiveBatchResult(response *openapi3.Response, metadata destructiveOperationMetadata) error {
	if metadata.Batch == nil || metadata.Execution != "synchronous" {
		return nil
	}
	if !responseUsesSchema(response, "EnvelopedDestructiveBatchResult") {
		return fmt.Errorf("synchronous destructive batch response must use EnvelopedDestructiveBatchResult")
	}
	return nil
}

func responseUsesSchema(response *openapi3.Response, schemaName string) bool {
	mediaType := response.Content.Get("application/json")
	if mediaType == nil || mediaType.Schema == nil {
		return false
	}
	refName := mediaType.Schema.Ref
	if separator := strings.LastIndex(refName, "/"); separator >= 0 {
		refName = refName[separator+1:]
	}
	refName = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(refName, ".yaml"), ".yml"), ".json")
	normalize := func(value string) string {
		return strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.ToLower(value))
	}
	return normalize(refName) == normalize(schemaName)
}

func hasRequiredHeaderParameter(pathItem *openapi3.PathItem, operation *openapi3.Operation, name string) bool {
	parameters := make(openapi3.Parameters, 0, len(pathItem.Parameters)+len(operation.Parameters))
	parameters = append(parameters, pathItem.Parameters...)
	parameters = append(parameters, operation.Parameters...)
	for _, parameter := range parameters {
		if parameter != nil && parameter.Value != nil &&
			strings.EqualFold(parameter.Value.Name, name) &&
			parameter.Value.In == openapi3.ParameterInHeader &&
			parameter.Value.Required {
			return true
		}
	}
	return false
}

func destructiveOperationError(operation *openapi3.Operation, method string, path string, format string, args ...any) error {
	operationID := strings.TrimSpace(operation.OperationID)
	if operationID == "" {
		operationID = "<missing-operation-id>"
	}
	return fmt.Errorf("OpenAPI destructive operation %s %s (%s): %s", strings.ToUpper(method), path, operationID, fmt.Sprintf(format, args...))
}

func isOneOf[T comparable](value T, allowed ...T) bool {
	return slices.Contains(allowed, value)
}
