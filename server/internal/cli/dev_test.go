package cli

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRunDevRunsMigrateBeforeServe 验证开发编排命令会先执行迁移，再启动服务。
func TestRunDevRunsMigrateBeforeServe(t *testing.T) {
	originalMigrateRunner := devMigrateRunner
	originalServeRunner := devServeRunner
	defer func() {
		devMigrateRunner = originalMigrateRunner
		devServeRunner = originalServeRunner
	}()

	var steps []string
	devMigrateRunner = func(_ *cobra.Command, migrationDir string) error {
		steps = append(steps, "migrate:"+migrationDir)
		return nil
	}
	devServeRunner = func(_ *cobra.Command, _ []string) error {
		steps = append(steps, "serve")
		return nil
	}

	err := runDev(&cobra.Command{}, nil, devOptions{migrationDir: defaultMigrationDir})
	if err != nil {
		t.Fatalf("run dev: %v", err)
	}

	expected := []string{"migrate:" + defaultMigrationDir, "serve"}
	if !reflect.DeepEqual(steps, expected) {
		t.Fatalf("expected %v, got %v", expected, steps)
	}
}

// TestRunDevStopsAfterMigrationFailure 验证迁移失败时不会继续启动服务。
func TestRunDevStopsAfterMigrationFailure(t *testing.T) {
	originalMigrateRunner := devMigrateRunner
	originalServeRunner := devServeRunner
	defer func() {
		devMigrateRunner = originalMigrateRunner
		devServeRunner = originalServeRunner
	}()

	devMigrateRunner = func(_ *cobra.Command, _ string) error {
		return errors.New("migrate failed")
	}
	devServeRunner = func(_ *cobra.Command, _ []string) error {
		t.Fatal("serve runner should not be called")
		return nil
	}

	err := runDev(&cobra.Command{}, nil, devOptions{migrationDir: defaultMigrationDir})
	if err == nil {
		t.Fatal("expected dev command error")
	}
	if !strings.Contains(err.Error(), "run development migrations") {
		t.Fatalf("expected migration context, got %v", err)
	}
}

// TestNewRootCommandRegistersDevCommand 验证根命令始终注册 `dev` 子命令。
func TestNewRootCommandRegistersDevCommand(t *testing.T) {
	command := NewRootCommand()

	found, _, err := command.Find([]string{"dev"})
	if err != nil {
		t.Fatalf("find dev command: %v", err)
	}
	if found == nil || found.Name() != "dev" {
		t.Fatalf("expected dev command, got %#v", found)
	}
}
