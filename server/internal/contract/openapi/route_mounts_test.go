package openapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

type operationMount struct {
	method string
	path   string
}

// TestBusinessOperationsMountStrictHandlers 校验每个业务契约操作恰好有一个 Gin 挂载点调用生成 wrapper。
// 非业务路由由 core 持有，容器 Shell 使用独立的 WebSocket 升级链路。
func TestBusinessOperationsMountStrictHandlers(t *testing.T) {
	t.Parallel()

	root, err := findRepositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	spec, err := loader.LoadFromFile(filepath.Join(root, "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	expected := businessOperationMounts(t, spec)
	files, err := filepath.Glob(filepath.Join(root, "server", "modules", "*", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	registered := make(map[string]token.Position)
	for _, occurrence := range strictRouteMounts(t, files) {
		mount, exists := expected[occurrence.operation]
		switch {
		case !exists:
			t.Errorf("%s mounts non-business or unknown operation %s", occurrence.position, occurrence.operation)
		case mount.method != occurrence.method:
			t.Errorf("%s mounts %s as %s, want %s %s", occurrence.position, occurrence.operation, occurrence.method, mount.method, mount.path)
		}
		if previous, found := registered[occurrence.operation]; found {
			t.Errorf("%s and %s both mount %s", previous, occurrence.position, occurrence.operation)
		}
		registered[occurrence.operation] = occurrence.position
	}
	for operation, mount := range expected {
		if _, ok := registered[operation]; !ok {
			t.Errorf("%s %s (%s) has no module-owned strict handler mount", mount.method, mount.path, operation)
		}
	}
	t.Logf("checked %d strict mounts for %d business operations", len(registered), len(expected))
}

type strictRouteMount struct {
	operation string
	method    string
	position  token.Position
}

func strictRouteMounts(t *testing.T, files []string) []strictRouteMount {
	t.Helper()
	var mounts []strictRouteMount
	positions := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(positions, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				operation, method := strictMountedOperation(call)
				if operation != "" {
					mounts = append(mounts, strictRouteMount{operation, method, positions.Position(call.Pos())})
				}
			}
			return true
		})
	}
	return mounts
}

func businessOperationMounts(t *testing.T, spec *openapi3.T) map[string]operationMount {
	t.Helper()
	// 访问日志、应用日志、工作台、健康与实时能力均由 core 的对应 HTTP 入口注册。
	// Shell 路由由容器模块手动升级 WebSocket，并在握手中消费一次性会话凭证。
	exempt := map[string]bool{
		"/healthz": true, "/mcp": true, "/ws": true, "/sse": true,
		"/api/platform/capabilities": true, "/api/realtime/subscriptions": true,
		"/api/modules/runtime": true, "/api/modules/runtime/{module_key}": true,
		"/api/dashboard/summary": true, "/api/dashboard/widgets/{widget_id}": true,
		"/api/access-log": true, "/api/access-log/{id}": true,
		"/api/access-log/saved-views": true, "/api/access-log/saved-views/{viewId}": true,
		"/api/app-log": true, "/api/app-log/{id}": true, "/api/app-log/deletions": true,
		"/api/app-log/saved-views": true, "/api/app-log/saved-views/{viewId}": true,
		"/api/ops/containers/{id}/shell/ws": true,
	}
	expected := make(map[string]operationMount)
	for path, item := range spec.Paths.Map() {
		if exempt[path] {
			delete(exempt, path)
			continue
		}
		for method, operation := range item.Operations() {
			name := operation.OperationID
			if name == "" {
				t.Fatalf("missing operation ID for %s %s", method, path)
			}
			identifier := strings.ToUpper(name[:1]) + name[1:]
			if prior, exists := expected[identifier]; exists {
				t.Fatalf("operation %s identifies both %s and %s", identifier, prior.path, path)
			}
			expected[identifier] = operationMount{method, path}
		}
	}
	for stale := range exempt {
		t.Errorf("strict route exception must point to an existing canonical path: %s", stale)
	}
	return expected
}

func strictMountedOperation(mount *ast.CallExpr) (operation string, method string) {
	route, ok := mount.Fun.(*ast.SelectorExpr)
	if !ok || len(mount.Args) < 2 {
		return "", ""
	}
	switch route.Sel.Name {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
	default:
		return "", ""
	}
	handler := mount.Args[len(mount.Args)-1]
	if wrapper, ok := handler.(*ast.CallExpr); ok {
		selector, qualified := wrapper.Fun.(*ast.SelectorExpr)
		if !qualified || selector.Sel.Name != "Route" || len(wrapper.Args) != 2 {
			return "", ""
		}
		pkg, imported := selector.X.(*ast.Ident)
		if !imported || pkg.Name != "strictsupport" {
			return "", ""
		}
		handler = wrapper.Args[0]
	}
	selector, ok := handler.(*ast.SelectorExpr)
	if !ok {
		return "", ""
	}
	switch receiver := selector.X.(type) {
	case *ast.Ident:
		if receiver.Name == "wrapper" {
			return selector.Sel.Name, route.Sel.Name
		}
	case *ast.SelectorExpr:
		if receiver.Sel.Name == "wrapper" {
			return selector.Sel.Name, route.Sel.Name
		}
	}
	return "", ""
}
