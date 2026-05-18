package cli

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"graft/server/internal/plugin"
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

type serveRecorderPlugin struct {
	name string
}

func (p serveRecorderPlugin) Name() string { return p.name }

func (p serveRecorderPlugin) Version() string { return "0.1.0" }

func (p serveRecorderPlugin) DependsOn() []string { return nil }

func (p serveRecorderPlugin) Register(_ *plugin.Context) error { return nil }

func (p serveRecorderPlugin) Boot(_ *plugin.Context) error { return nil }

func (p serveRecorderPlugin) Shutdown(_ *plugin.Context) error { return nil }

// TestRunServeBuildsPluginsFromRegistry 验证 serve 通过 registry 入口拿到插件集合，
// 而不是在 CLI 内部继续手写中心化插件列表。
func TestRunServeBuildsPluginsFromRegistry(t *testing.T) {
	originalBuildPlugins := serveBuildPlugins
	originalNewRuntime := serveNewRuntime
	originalNotifyContext := serveNotifyContext
	defer func() {
		serveBuildPlugins = originalBuildPlugins
		serveNewRuntime = originalNewRuntime
		serveNotifyContext = originalNotifyContext
	}()

	expectedPlugins := []plugin.Plugin{
		serveRecorderPlugin{name: "audit"},
		serveRecorderPlugin{name: "user"},
	}
	var buildCalls int
	var gotPlugins []string
	runtime := &serveRecorderRuntime{}

	serveBuildPlugins = func() ([]plugin.Plugin, error) {
		buildCalls++
		return expectedPlugins, nil
	}
	serveNewRuntime = func(plugins ...plugin.Plugin) (runtimeRunner, error) {
		for _, current := range plugins {
			gotPlugins = append(gotPlugins, current.Name())
		}
		return runtime, nil
	}
	serveNotifyContext = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return parent, func() {}
	}

	if err := runServe(&cobra.Command{}, nil); err != nil {
		t.Fatalf("run serve: %v", err)
	}

	if buildCalls != 1 {
		t.Fatalf("expected registry build call once, got %d", buildCalls)
	}

	expectedNames := []string{"audit", "user"}
	if !reflect.DeepEqual(gotPlugins, expectedNames) {
		t.Fatalf("expected runtime plugins %v, got %v", expectedNames, gotPlugins)
	}
	if runtime.runCtx == nil {
		t.Fatal("expected runtime to receive a context")
	}
}

// TestRunServeReportsRegistryBuildFailure 验证 registry 构造失败会直接阻断 serve。
func TestRunServeReportsRegistryBuildFailure(t *testing.T) {
	originalBuildPlugins := serveBuildPlugins
	defer func() {
		serveBuildPlugins = originalBuildPlugins
	}()

	serveBuildPlugins = func() ([]plugin.Plugin, error) {
		return nil, errors.New("registry failed")
	}

	err := runServe(&cobra.Command{}, nil)
	if err == nil {
		t.Fatal("expected serve error")
	}
	if !strings.Contains(err.Error(), "build runtime plugins") {
		t.Fatalf("expected registry context, got %v", err)
	}
}

// TestRunServeUsesCommandContextWhenPresent 验证 serve 会把命令上下文传给运行时。
func TestRunServeUsesCommandContextWhenPresent(t *testing.T) {
	originalBuildPlugins := serveBuildPlugins
	originalNewRuntime := serveNewRuntime
	originalNotifyContext := serveNotifyContext
	defer func() {
		serveBuildPlugins = originalBuildPlugins
		serveNewRuntime = originalNewRuntime
		serveNotifyContext = originalNotifyContext
	}()

	expectedCtx := context.WithValue(context.Background(), serveTestContextKey{}, "serve")
	runtime := &serveRecorderRuntime{}

	serveBuildPlugins = func() ([]plugin.Plugin, error) {
		return []plugin.Plugin{serveRecorderPlugin{name: "user"}}, nil
	}
	serveNewRuntime = func(_ ...plugin.Plugin) (runtimeRunner, error) {
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
