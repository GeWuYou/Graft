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
	runCtx context.Context
	runErr error
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
	defer func() {
		serveNewRuntime = originalNewRuntime
		serveNotifyContext = originalNotifyContext
	}()

	expectedCtx := context.WithValue(context.Background(), serveTestContextKey{}, "serve")
	runtime := &serveRecorderRuntime{}

	serveNewRuntime = func() (runtimeRunner, error) {
		return runtime, nil
	}
	serveNotifyContext = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return parent, func() {}
	}

	cmd := &cobra.Command{}
	cmd.SetContext(expectedCtx)

	if err := runServe(cmd, nil); err != nil {
		t.Fatalf("run serve: %v", err)
	}

	if runtime.runCtx != expectedCtx {
		t.Fatalf("expected serve to use command context")
	}
}

// TestRunServeReportsRuntimeConstructionFailure 验证 runtime 构造失败会直接阻断 serve。
func TestRunServeReportsRuntimeConstructionFailure(t *testing.T) {
	originalNewRuntime := serveNewRuntime
	defer func() {
		serveNewRuntime = originalNewRuntime
	}()

	serveNewRuntime = func() (runtimeRunner, error) {
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
