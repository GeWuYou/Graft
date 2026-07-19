package projection

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRenderTypeScriptIsDeterministicAndFiltersInternalValues(t *testing.T) {
	entries := Registry()
	first, err := RenderTypeScript(entries)
	if err != nil {
		t.Fatalf("render first projection: %v", err)
	}
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}
	second, err := RenderTypeScript(entries)
	if err != nil {
		t.Fatalf("render reordered projection: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("expected projection rendering to be deterministic")
	}
	output := string(first)
	if !strings.Contains(output, "export const ERROR_CODE") || !strings.Contains(output, "export type ErrorCode") {
		t.Fatalf("expected literal object and union type, got %s", output)
	}
	if strings.Contains(output, "X-Trace-Id") {
		t.Fatalf("internal descriptor leaked into web output: %s", output)
	}
	lastIndex := -1
	for _, marker := range []string{"AUTH_SCHEME", "ERROR_CODE", "HTTP_HEADER", "MESSAGE_KEY"} {
		index := strings.Index(output, marker)
		if index <= lastIndex {
			t.Fatalf("expected canonical generation order for %q, got %s", marker, output)
		}
		lastIndex = index
	}
}

func TestValidateRejectsDuplicateValuesAndInvalidMetadata(t *testing.T) {
	entries := Registry()
	entries = append(entries, Entry{
		ID:         "platform.error-code.duplicate",
		Name:       "DUPLICATE",
		Kind:       KindErrorCode,
		Owner:      "server/internal/contract/errorcode",
		Lifecycle:  LifecycleActive,
		Visibility: VisibilityWeb,
		Value:      entries[1].Value,
	})
	if err := Validate(entries); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate value validation error, got %v", err)
	}

	entries = Registry()
	entries[0].Owner = ""
	if err := Validate(entries); err == nil || !strings.Contains(err.Error(), "requires id, name, and owner") {
		t.Fatalf("expected metadata validation error, got %v", err)
	}

	entries = Registry()
	entries[0].Lifecycle = LifecycleDeprecated
	if err := Validate(entries); err == nil || !strings.Contains(err.Error(), "requires replacement") {
		t.Fatalf("expected lifecycle validation error, got %v", err)
	}

	entries = Registry()
	entries[0].Lifecycle = LifecycleDeprecated
	entries[0].Replacement = "missing.replacement"
	if err := Validate(entries); err == nil || !strings.Contains(err.Error(), "active replacement") {
		t.Fatalf("expected deprecated replacement validation error, got %v", err)
	}
}

func TestRegistryValuesReferenceExistingConstants(t *testing.T) {
	_, filePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve registry source path")
	}
	sourcePath := filepath.Join(filepath.Dir(filePath), "registry.go")
	// sourcePath 由当前测试文件目录和固定 registry.go 文件名组成，不接受外部输入。
	source, err := os.ReadFile(sourcePath) // #nosec G304
	if err != nil {
		t.Fatalf("read registry source: %v", err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), sourcePath, source, 0)
	if err != nil {
		t.Fatalf("parse registry source: %v", err)
	}

	valueFields := 0
	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, element := range literal.Elts {
			field, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := field.Key.(*ast.Ident)
			if !ok || key.Name != "Value" {
				continue
			}
			valueFields++
			if _, ok := field.Value.(*ast.SelectorExpr); !ok {
				t.Errorf("Value field must reference an existing typed constant, got %T", field.Value)
			}
		}
		return true
	})
	if valueFields == 0 {
		t.Fatal("expected registry descriptors with typed value references")
	}
}

func TestTargetsKeepModuleContractsOutOfPlatformArtifact(t *testing.T) {
	targets := Targets()

	outputs := make(map[string]string, len(targets))
	for _, target := range targets {
		rendered, err := RenderTarget(target)
		if err != nil {
			t.Fatalf("render target %q: %v", target.Path, err)
		}
		outputs[target.Path] = string(rendered)
	}
	if strings.Contains(outputs["platform.ts"], "CONTAINER_PERMISSION_CODE") {
		t.Fatalf("module contract leaked into platform artifact: %s", outputs["platform.ts"])
	}
	for _, marker := range []string{"CONTAINER_PERMISSION_CODE", "CONTAINER_REALTIME_TOPIC", "DOCKER_IMAGE_REMOVE_ERROR_CODES"} {
		if !strings.Contains(outputs["modules/container.ts"], marker) {
			t.Fatalf("container artifact is missing %q: %s", marker, outputs["modules/container.ts"])
		}
	}
	for path, marker := range map[string]string{
		"modules/access-log.ts":     "ACCESS_LOG_PERMISSION_CODE",
		"modules/announcement.ts":   "ANNOUNCEMENT_PERMISSION_CODE",
		"modules/app-log.ts":        "APP_LOG_PERMISSION_CODE",
		"modules/audit.ts":          "AUDIT_PERMISSION_CODE",
		"modules/monitor.ts":        "MONITOR_PERMISSION_CODE",
		"modules/notification.ts":   "NOTIFICATION_PERMISSION_CODE",
		"modules/project.ts":        "PROJECT_REALTIME_TOPIC",
		"modules/rbac.ts":           "RBAC_PERMISSION_CODE",
		"modules/runtime-target.ts": "RUNTIME_TARGET_REALTIME_TOPIC",
		"modules/scheduled-task.ts": "SCHEDULED_TASK_PERMISSION_CODE",
		"modules/security.ts":       "SECURITY_PERMISSION_CODE",
		"modules/system-config.ts":  "SYSTEM_CONFIG_PERMISSION_CODE",
		"modules/task.ts":           "TASK_REALTIME_EVENT",
		"modules/user.ts":           "USER_PERMISSION_CODE",
	} {
		if !strings.Contains(outputs[path], marker) {
			t.Fatalf("module artifact %q is missing %q: %s", path, marker, outputs[path])
		}
	}
}

func TestRenderTargetUsesTargetGroupMetadata(t *testing.T) {
	target := Target{
		Path:    "modules/example.ts",
		Groups:  []Group{{Kind: KindPermissionCode, Constant: "EXAMPLE_PERMISSION_CODE", TypeName: "ExamplePermissionCode"}},
		Entries: []Entry{{ID: "example.permission.view", Name: "VIEW", Kind: KindPermissionCode, Owner: "example", Lifecycle: LifecycleActive, Visibility: VisibilityWeb, Value: "example.view"}},
	}
	rendered, err := RenderTarget(target)
	if err != nil {
		t.Fatalf("render target: %v", err)
	}
	output := string(rendered)
	if !strings.Contains(output, "export const EXAMPLE_PERMISSION_CODE") || !strings.Contains(output, "export type ExamplePermissionCode") {
		t.Fatalf("target metadata was not applied: %s", output)
	}
	if strings.Contains(output, "CONTAINER_PERMISSION_CODE") {
		t.Fatalf("target inherited container-specific export name: %s", output)
	}
}
