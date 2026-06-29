package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/moduleregistry"
)

// defaultMigrationDir 定义 `server` 模块默认迁移链使用的 registry 选择器。
const defaultMigrationDir = moduleregistry.DefaultMigrationDir

const migrationVersionMatchCount = 2
const externalMigrationDirPrefix = "file:"

// 这些变量保留为可替换的命令边界，便于测试覆盖 cwd、compile-time registry、
// 嵌入式迁移资源解析以及 Atlas 执行装配。
var migrateGetwd = os.Getwd
var migrateRegistryMigrationDirs = moduleregistry.MigrationDirs
var migrateEmbeddedMigrationDirByPath = moduleregistry.EmbeddedMigrationDirByPath
var migrateReadDir = os.ReadDir
var migrateOpenExecutor = openAtlasExecutor

// migrateUpOptions 封装一次显式迁移执行所需的输入。
type migrateUpOptions struct {
	migrationDir string
	workingDir   string
	allowDirty   bool
}

type atlasExecutorHandle struct {
	executor atlasExecutor
	close    func() error
}

type atlasExecutor interface {
	ExecuteN(context.Context, int) error
}

// newMigrateCommand 创建显式数据库迁移命令树。
//
// 迁移能力保持在独立的 `graft migrate` 子树下，避免普通运行时启动路径
// NewMigrateCommand creates the migrate command with subcommands for applying and validating Atlas migrations.
func newMigrateCommand() *cobra.Command {
	var migrationDir string

	command := &cobra.Command{
		Use:   "migrate",
		Short: "Run explicit database migration commands",
	}
	command.PersistentFlags().StringVar(&migrationDir, "dir", defaultMigrationDir, "migration directory or owner-aligned default chain")

	upOptions := migrateUpOptions{}
	upCommand := &cobra.Command{
		Use:   "up",
		Short: "Apply pending Atlas versioned migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			upOptions.migrationDir = migrationDir
			return runMigrateUp(cmd, upOptions)
		},
	}
	upCommand.Flags().BoolVar(&upOptions.allowDirty, "allow-dirty", false, "allow the first migration run against a disposable database that is not Atlas-clean")
	command.AddCommand(upCommand)
	command.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate migration assets without connecting to the database",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runMigrateValidate(migrateResolveOptions{migrationDir: migrationDir})
		},
	})

	return command
}

// runMigrateUp 执行一次 Atlas 版本化迁移应用。
//
// 参数：
//   - cmd: 当前 Cobra 命令，用于继承上下文和标准输入输出。
//   - opts: 迁移目录与工作目录等显式执行选项。
//
// 返回值：
// runMigrateUp 应用待处理的迁移到数据库。
// runMigrateUp 执行一次 Atlas 迁移并关闭相关资源。
// 迁移目录解析、执行或关闭过程中发生错误时返回错误；当没有待处理迁移时返回 nil。
func runMigrateUp(cmd *cobra.Command, opts migrateUpOptions) (err error) {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dir, err := resolveAtlasMigrationDir(migrateResolveOptions{
		migrationDir: opts.migrationDir,
		workingDir:   opts.workingDir,
	})
	if err != nil {
		return fmt.Errorf("resolve migration dir: %w", err)
	}

	commandContext := cmd.Context()
	if commandContext == nil {
		commandContext = context.Background()
	}

	handle, err := migrateOpenExecutor(cfg.Database.URL, dir, newAtlasCommandLogger(cmd), opts.allowDirty)
	if err != nil {
		return err
	}
	defer func() {
		if handle.close == nil {
			return
		}
		closeErr := handle.close()
		if closeErr == nil {
			return
		}
		if err == nil {
			err = fmt.Errorf("close atlas executor: %w", closeErr)
			return
		}
		err = errors.Join(err, fmt.Errorf("close atlas executor: %w", closeErr))
	}()

	if err := handle.executor.ExecuteN(commandContext, 0); err != nil {
		if errors.Is(err, atlasmigrate.ErrNoPendingFiles) {
			return nil
		}
		return fmt.Errorf("apply atlas migrations: %w", err)
	}

	return nil
}

type migrateResolveOptions struct {
	migrationDir string
	workingDir   string
}

// runMigrateValidate 验证 Atlas 迁移目录是否有效。
func runMigrateValidate(opts migrateResolveOptions) error {
	dir, err := resolveAtlasMigrationDir(opts)
	if err != nil {
		return fmt.Errorf("resolve migration dir: %w", err)
	}
	if err := atlasmigrate.Validate(dir); err != nil {
		return fmt.Errorf("validate migration dir: %w", err)
	}
	return nil
}

// ResolveAtlasMigrationDir resolves an Atlas migration directory, using the current working directory if none is provided.
func resolveAtlasMigrationDir(opts migrateResolveOptions) (atlasmigrate.Dir, error) {
	workingDir := opts.workingDir
	if strings.TrimSpace(workingDir) == "" {
		var err error
		workingDir, err = migrateGetwd()
		if err != nil {
			return nil, fmt.Errorf("resolve working directory: %w", err)
		}
	}

	return buildAtlasMigrationDir(workingDir, opts.migrationDir)
}

// openAtlasExecutor 为指定的数据库和迁移目录创建一个 Atlas 迁移执行器。
// 返回的句柄包含执行器以及用于关闭数据库连接的函数。
func openAtlasExecutor(databaseURL string, dir atlasmigrate.Dir, logger atlasmigrate.Logger, allowDirty bool) (*atlasExecutorHandle, error) {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres database pool: %w", err)
	}

	driver, err := atlaspostgres.Open(sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("open atlas postgres driver: %w", err)
	}

	executor, err := atlasmigrate.NewExecutor(
		driver,
		dir,
		newAtlasRevisionStore(sqlDB),
		atlasmigrate.WithAllowDirty(allowDirty),
		atlasmigrate.WithLogger(logger),
		atlasmigrate.WithOperatorVersion("graft"),
	)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create atlas migration executor: %w", err)
	}

	return &atlasExecutorHandle{
		executor: executor,
		close:    sqlDB.Close,
	}, nil
}
