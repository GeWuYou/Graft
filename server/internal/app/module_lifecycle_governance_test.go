package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestModuleLifecycleMethodsAvoidDetachedContexts enforces that module lifecycle
// code does not detach from runtime-owned shutdown semantics by introducing
// background contexts or local time.After shutdown windows.
func TestModuleLifecycleMethodsAvoidDetachedContexts(t *testing.T) {
	t.Parallel()

	currentFile, ok := currentTestFile()
	if !ok {
		t.Fatal("resolve current file")
	}

	modulesRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "modules"))
	fileSet := token.NewFileSet()
	var violations []string

	walkErr := filepath.WalkDir(modulesRoot, func(path string, entry os.DirEntry, err error) error {
		return collectModuleLifecycleViolations(fileSet, path, entry, err, &violations)
	})
	if walkErr != nil {
		t.Fatalf("scan module lifecycle governance: %v", walkErr)
	}
	if len(violations) > 0 {
		for _, violation := range violations {
			t.Error(violation)
		}
		t.Fatal("module lifecycle governance violations found")
	}
}

func currentTestFile() (string, bool) {
	_, currentFile, _, ok := runtime.Caller(0)
	return currentFile, ok
}

func collectModuleLifecycleViolations(fileSet *token.FileSet, path string, entry os.DirEntry, walkErr error, violations *[]string) error {
	if walkErr != nil {
		return walkErr
	}
	if shouldSkipLifecycleGovernanceFile(path, entry) {
		return nil
	}

	node, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return err
	}

	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !isModuleLifecycleMethod(funcDecl) {
			continue
		}
		collectLifecycleCallViolations(fileSet, node, funcDecl, violations)
	}

	return nil
}

func shouldSkipLifecycleGovernanceFile(path string, entry os.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	if filepath.Ext(path) != ".go" || filepath.Base(path) == "" {
		return true
	}
	matched, _ := filepath.Match("*_test.go", filepath.Base(path))
	return matched
}

func isModuleLifecycleMethod(funcDecl *ast.FuncDecl) bool {
	if funcDecl == nil || funcDecl.Recv == nil || funcDecl.Body == nil || len(funcDecl.Recv.List) == 0 {
		return false
	}
	return isModuleReceiver(funcDecl.Recv.List[0].Type)
}

func collectLifecycleCallViolations(fileSet *token.FileSet, file *ast.File, funcDecl *ast.FuncDecl, violations *[]string) {
	prohibitedImports := prohibitedLifecycleImports(file)
	ast.Inspect(funcDecl.Body, func(child ast.Node) bool {
		message := lifecycleViolationMessage(fileSet, child, prohibitedImports)
		if message != "" {
			*violations = append(*violations, message)
		}
		return true
	})
}

func prohibitedLifecycleImports(file *ast.File) map[string]string {
	prohibited := make(map[string]string, 2)
	if file == nil {
		return prohibited
	}

	for _, spec := range file.Imports {
		pathValue, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if pathValue != "context" && pathValue != "time" {
			continue
		}

		importName := ""
		if spec.Name != nil {
			importName = spec.Name.Name
		} else {
			importName = path.Base(pathValue)
		}
		if importName == "." || importName == "_" || importName == "" {
			continue
		}
		prohibited[importName] = pathValue
	}

	return prohibited
}

func lifecycleViolationMessage(fileSet *token.FileSet, node ast.Node, prohibitedImports map[string]string) string {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return ""
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	pkg, ok := selector.X.(*ast.Ident)
	if !ok {
		return ""
	}

	position := fileSet.Position(call.Pos())
	switch {
	case prohibitedImports[pkg.Name] == "context" && selector.Sel.Name == "Background":
		return position.String() + ": Module receiver method must not call context.Background()"
	case prohibitedImports[pkg.Name] == "time" && selector.Sel.Name == "After":
		return position.String() + ": Module receiver method must not create a local time.After shutdown window"
	default:
		return ""
	}
}

func isModuleReceiver(expr ast.Expr) bool {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name == "Module"
	case *ast.StarExpr:
		ident, ok := typed.X.(*ast.Ident)
		return ok && ident.Name == "Module"
	default:
		return false
	}
}

func TestLifecycleViolationMessageRejectsAliasedBackgroundAndAfter(t *testing.T) {
	t.Parallel()

	source := strings.Join([]string{
		"package sample",
		"",
		"import (",
		`	ctxpkg "context"`,
		`	timer "time"`,
		")",
		"",
		"type Module struct{}",
		"",
		"func (m *Module) Shutdown() {",
		"\t_ = ctxpkg.Background()",
		"\t<-timer.After(5)",
		"}",
	}, "\n")

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "sample.go", source, 0)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}

	funcDecl := firstLifecycleFuncDecl(t, file)
	prohibitedImports := prohibitedLifecycleImports(file)
	var violations []string
	collectLifecycleCallViolations(fileSet, file, funcDecl, &violations)

	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d: %v", len(violations), violations)
	}
	if !containsViolation(violations, "must not call context.Background()") {
		t.Fatalf("expected context.Background violation, got %v", violations)
	}
	if !containsViolation(violations, "must not create a local time.After shutdown window") {
		t.Fatalf("expected time.After violation, got %v", violations)
	}
	if prohibitedImports["ctxpkg"] != "context" || prohibitedImports["timer"] != "time" {
		t.Fatalf("expected aliased imports to map to context/time, got %v", prohibitedImports)
	}
}

func firstLifecycleFuncDecl(t *testing.T, file *ast.File) *ast.FuncDecl {
	t.Helper()

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok && isModuleLifecycleMethod(funcDecl) {
			return funcDecl
		}
	}

	t.Fatal("expected module lifecycle function declaration")
	return nil
}

func containsViolation(violations []string, suffix string) bool {
	for _, violation := range violations {
		if strings.HasSuffix(violation, suffix) {
			return true
		}
	}
	return false
}
