package app

import (
	"errors"
	"testing"

	"graft/server/internal/plugin"
)

type shutdownRecorderPlugin struct {
	name        string
	shutdownLog *[]string
	err         error
}

func (p shutdownRecorderPlugin) Name() string { return p.name }

func (p shutdownRecorderPlugin) Version() string { return "test" }

func (p shutdownRecorderPlugin) DependsOn() []string { return nil }

func (p shutdownRecorderPlugin) Register(ctx *plugin.Context) error { return nil }

func (p shutdownRecorderPlugin) Boot(ctx *plugin.Context) error { return nil }

func (p shutdownRecorderPlugin) Shutdown(ctx *plugin.Context) error {
	*p.shutdownLog = append(*p.shutdownLog, p.name)
	return p.err
}

// TestShutdownPluginsUsesReverseOrder 验证插件关闭顺序与启动顺序相反，
// 以便后启动的依赖先完成资源释放。
func TestShutdownPluginsUsesReverseOrder(t *testing.T) {
	log := make([]string, 0, 3)
	plugins := []plugin.Plugin{
		shutdownRecorderPlugin{name: "user", shutdownLog: &log},
		shutdownRecorderPlugin{name: "rbac", shutdownLog: &log},
		shutdownRecorderPlugin{name: "audit", shutdownLog: &log},
	}

	if err := shutdownPlugins(&plugin.Context{}, plugins); err != nil {
		t.Fatalf("shutdown plugins: %v", err)
	}

	expected := []string{"audit", "rbac", "user"}
	for index, name := range expected {
		if log[index] != name {
			t.Fatalf("expected shutdown order %v, got %v", expected, log)
		}
	}
}

// TestShutdownPluginsAggregatesErrors 验证多个插件关闭失败时会聚合错误，
// 避免后续失败被前一个失败覆盖。
func TestShutdownPluginsAggregatesErrors(t *testing.T) {
	plugins := []plugin.Plugin{
		shutdownRecorderPlugin{name: "user", shutdownLog: &[]string{}, err: errors.New("user failed")},
		shutdownRecorderPlugin{name: "rbac", shutdownLog: &[]string{}, err: errors.New("rbac failed")},
	}

	err := shutdownPlugins(&plugin.Context{}, plugins)
	if err == nil {
		t.Fatal("expected shutdown error")
	}
	if !errors.Is(err, plugins[0].(shutdownRecorderPlugin).err) {
		t.Fatal("expected joined error to include user failure")
	}
	if !errors.Is(err, plugins[1].(shutdownRecorderPlugin).err) {
		t.Fatal("expected joined error to include rbac failure")
	}
}
