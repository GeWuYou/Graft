package cli

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type serveRecorderRuntime struct {
	buildCtx context.Context
	runCtx   context.Context
	runErr   error
}

type serveTestContextKey struct{}

func (r *serveRecorderRuntime) Run(ctx context.Context) error {
	r.runCtx = ctx
	return r.runErr
}

// TestRunServeUsesCommandContextWhenPresent 验证 serve 会把命令上下文传给运行时。
func TestRunServeUsesCommandContextWhenPresent(t *testing.T) {
	originalNewRuntime := serveNewRuntime
	originalNotifyContext := serveNotifyContext
	originalConfigValidator := serveConfigValidator
	defer func() {
		serveNewRuntime = originalNewRuntime
		serveNotifyContext = originalNotifyContext
		serveConfigValidator = originalConfigValidator
	}()

	expectedCtx := context.WithValue(context.Background(), serveTestContextKey{}, "serve")
	runtime := &serveRecorderRuntime{}

	serveNewRuntime = func(ctx context.Context) (runtimeRunner, error) {
		runtime.buildCtx = ctx
		return runtime, nil
	}
	serveNotifyContext = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return parent, func() {}
	}
	serveConfigValidator = func(*cobra.Command) error { return nil }

	cmd := &cobra.Command{}
	cmd.SetContext(expectedCtx)

	if err := runServe(cmd, nil); err != nil {
		t.Fatalf("run serve: %v", err)
	}

	if runtime.buildCtx != expectedCtx {
		t.Fatalf("expected serve to use command context when building runtime")
	}
	if runtime.runCtx != expectedCtx {
		t.Fatalf("expected serve to use command context")
	}
}

// TestRunServeReportsRuntimeConstructionFailure 验证 runtime 构造失败会直接阻断 serve。
func TestRunServeReportsRuntimeConstructionFailure(t *testing.T) {
	originalNewRuntime := serveNewRuntime
	originalConfigValidator := serveConfigValidator
	defer func() {
		serveNewRuntime = originalNewRuntime
		serveConfigValidator = originalConfigValidator
	}()
	serveConfigValidator = func(*cobra.Command) error { return nil }

	serveNewRuntime = func(context.Context) (runtimeRunner, error) {
		return nil, errors.New("runtime build failed")
	}

	err := runServe(&cobra.Command{}, nil)
	if err == nil {
		t.Fatal("expected serve error")
	}
	if !strings.Contains(err.Error(), "create runtime") {
		t.Fatalf("expected runtime construction context, got %v", err)
	}
}

func TestRunServeStopsBeforeRuntimeWhenConfigurationIsInvalid(t *testing.T) {
	originalNewRuntime := serveNewRuntime
	originalConfigValidator := serveConfigValidator
	defer func() {
		serveNewRuntime = originalNewRuntime
		serveConfigValidator = originalConfigValidator
	}()

	called := false
	serveNewRuntime = func(context.Context) (runtimeRunner, error) {
		called = true
		return &serveRecorderRuntime{}, nil
	}
	serveConfigValidator = func(*cobra.Command) error { return errors.New("configuration validation failed") }

	if err := runServe(&cobra.Command{}, nil); err == nil {
		t.Fatal("expected configuration validation failure")
	}
	if called {
		t.Fatal("runtime must not be constructed after configuration validation failure")
	}
}
