package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/migrationcontract"
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
var migrateSchemaContractRunner = runMigrationSchemaContract
var migrateConfigValidator = validateStartupConfiguration

// migrateUpOptions 封装一次显式迁移执行所需的输入。
type migrateUpOptions struct {
	migrationDir string
	workingDir   string
	allowDirty   bool
}

type migrateCheckSchemaOptions struct {
	format string
	mode   string
}

type atlasExecutorHandle struct {
	executor           atlasExecutor
	preflightRevisions func(context.Context) error
	close              func() error
}

type atlasExecutor interface {
	ExecuteN(context.Context, int) error
}

// newMigrateCommand 创建独立的 `graft migrate` 命令树，将 Atlas 迁移应用与校验
// 保持在显式 CLI 路径中，避免混入普通运行时启动流程。
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

	preflightOptions := migratePreflightOptions{}
	preflightCommand := &cobra.Command{
		Use:          "preflight",
		Short:        "Run read-only target-data checks declared by a migration preflight sidecar",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := migrateConfigValidator(cmd); err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			return migratePreflightRunner(cmd.Context(), cfg.Database.URL, preflightOptions.manifest)
		},
	}
	preflightCommand.Flags().StringVar(&preflightOptions.manifest, "manifest", "", "migration .preflight.yaml sidecar")
	_ = preflightCommand.MarkFlagRequired("manifest")
	command.AddCommand(preflightCommand)

	checkSchemaOptions := migrateCheckSchemaOptions{}
	checkSchemaCommand := &cobra.Command{
		Use:          "check-schema",
		Short:        "Check PostgreSQL schema contracts from the system catalog",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrateCheckSchema(cmd, checkSchemaOptions)
		},
	}
	checkSchemaCommand.Flags().StringVar(&checkSchemaOptions.format, "format", "text", "output format: text or json")
	checkSchemaCommand.Flags().StringVar(&checkSchemaOptions.mode, "mode", "enforce", "result mode: enforce or report")
	command.AddCommand(checkSchemaCommand)

	return command
}

// runMigrateUp 应用待处理的 Atlas 迁移，并在执行完成后关闭数据库资源。
// 迁移目录解析、执行或关闭过程中发生错误时返回错误；当没有待处理迁移时返回 nil。
func runMigrateUp(cmd *cobra.Command, opts migrateUpOptions) (err error) {
	if err := migrateConfigValidator(cmd); err != nil {
		return err
	}

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
	return executeAtlasMigrations(commandContext, handle)
}

func executeAtlasMigrations(ctx context.Context, handle *atlasExecutorHandle) error {
	if handle.preflightRevisions != nil {
		if err := handle.preflightRevisions(ctx); err != nil {
			return err
		}
	}
	if err := handle.executor.ExecuteN(ctx, 0); err != nil {
		if errors.Is(err, atlasmigrate.ErrNoPendingFiles) {
			return nil
		}
		return fmt.Errorf("apply atlas migrations: %w", err)
	}
	return nil
}

type migrateResolveOptions struct {
	migrationDir                    string
	workingDir                      string
	embeddedRegistryFreshnessRunner func(string) error
}

// runMigrateValidate 验证 Atlas 迁移目录是否有效。
func runMigrateValidate(opts migrateResolveOptions) error {
	if shouldValidateEmbeddedMigrationRegistryFreshness(opts.migrationDir) {
		registryFreshnessRunner := opts.embeddedRegistryFreshnessRunner
		if registryFreshnessRunner == nil {
			registryFreshnessRunner = validateEmbeddedMigrationRegistryFreshness
		}
		if err := registryFreshnessRunner(opts.workingDir); err != nil {
			return err
		}
	}
	dir, err := resolveAtlasMigrationDir(opts)
	if err != nil {
		return fmt.Errorf("resolve migration dir: %w", err)
	}
	if err := atlasmigrate.Validate(dir); err != nil {
		return fmt.Errorf("validate migration dir: %w", err)
	}
	return nil
}

// runMigrateCheckSchema 输出 default migration chain 在 PostgreSQL 中形成的 catalog contract 结果。
func runMigrateCheckSchema(cmd *cobra.Command, opts migrateCheckSchemaOptions) error { //nolint:cyclop // schema gate validates independent format, mode, config, output, and finding paths.
	format := strings.TrimSpace(opts.format)
	if format == "" {
		format = "text"
	}
	if format != "text" && format != "json" {
		return fmt.Errorf("unsupported schema check format %q: expected text or json", format)
	}
	mode := strings.TrimSpace(opts.mode)
	if mode == "" {
		mode = "enforce"
	}
	if mode != "enforce" && mode != "report" {
		return fmt.Errorf("unsupported schema check mode %q: expected enforce or report", mode)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	result, err := migrateSchemaContractRunner(cmd.Context(), cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("check migration schema contract: %w", err)
	}
	if err := writeMigrationSchemaContractResult(cmd, format, result); err != nil {
		return err
	}
	if mode == "enforce" && result.HasEnforceFindings() {
		return errors.New("migration schema contract has enforce findings")
	}
	return nil
}

func runMigrationSchemaContract(ctx context.Context, databaseURL string) (migrationcontract.Result, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return migrationcontract.Result{}, fmt.Errorf("open postgres database pool: %w", err)
	}
	defer func() { _ = db.Close() }()

	checker, err := migrationcontract.NewChecker(db)
	if err != nil {
		return migrationcontract.Result{}, err
	}
	return checker.Check(ctx)
}

func writeMigrationSchemaContractResult(cmd *cobra.Command, format string, result migrationcontract.Result) error {
	if format == "json" {
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encode migration schema contract result: %w", err)
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
		return err
	}
	if len(result.Findings) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "migration schema contract: ok")
		return err
	}
	for _, finding := range result.Findings {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s: %s\n", finding.Severity, finding.ID, finding.Object, finding.Message); err != nil {
			return err
		}
	}
	return nil
}

func shouldValidateEmbeddedMigrationRegistryFreshness(migrationDir string) bool {
	return !strings.HasPrefix(strings.TrimSpace(migrationDir), externalMigrationDirPrefix)
}

func validateEmbeddedMigrationRegistryFreshness(workingDir string) error {
	moduleRoot := workingDir
	if strings.TrimSpace(moduleRoot) == "" {
		var err error
		moduleRoot, err = migrateGetwd()
		if err != nil {
			return fmt.Errorf("resolve working directory for embedded migration registry validation: %w", err)
		}
	}

	resolvedModuleRoot, matched, err := matchBackendModuleRoot(moduleRoot)
	if err != nil {
		return fmt.Errorf("resolve backend module root for embedded migration registry validation: %w", err)
	}
	if !matched {
		return fmt.Errorf("cannot locate server module root for embedded migration registry validation")
	}

	if err := moduleregistry.ValidateEmbeddedMigrationRegistryFreshness(resolvedModuleRoot); err != nil {
		return fmt.Errorf("validate embedded migration registry freshness: %w", err)
	}
	return nil
}

// resolveAtlasMigrationDir 解析 Atlas 迁移目录；未提供工作目录时使用当前进程目录。
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

	revisions := newAtlasRevisionStore(sqlDB)
	executor, err := atlasmigrate.NewExecutor(
		driver,
		dir,
		revisions,
		atlasmigrate.WithAllowDirty(allowDirty),
		atlasmigrate.WithLogger(logger),
		atlasmigrate.WithOperatorVersion("graft"),
	)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create atlas migration executor: %w", err)
	}

	return &atlasExecutorHandle{
		executor:           executor,
		preflightRevisions: func(ctx context.Context) error { return validateAtlasRevisionMetadata(ctx, revisions) },
		close:              sqlDB.Close,
	}, nil
}

// validateAtlasRevisionMetadata 在 Atlas 恢复部分执行的迁移前校验修订元数据，
// 防止 Atlas v1.2.3 对不安全的进度或 partial hash 数据建立索引。
func validateAtlasRevisionMetadata(ctx context.Context, store atlasmigrate.RevisionReadWriter) error {
	revisions, err := store.ReadRevisions(ctx)
	if err != nil {
		return fmt.Errorf("read atlas revisions for preflight: %w", err)
	}
	for _, revision := range revisions {
		if revision.Applied < 0 || revision.Total < 0 {
			return fmt.Errorf("unsafe atlas revision metadata for version %q: applied=%d, total=%d; repair atlas_schema_revisions before retrying", revision.Version, revision.Applied, revision.Total)
		}
		if revision.Applied > revision.Total {
			return fmt.Errorf("unsafe atlas revision metadata for version %q: applied=%d exceeds total=%d; repair atlas_schema_revisions before retrying", revision.Version, revision.Applied, revision.Total)
		}
		if revision.Applied > 0 && revision.Applied < revision.Total && len(revision.PartialHashes) < revision.Applied {
			return fmt.Errorf("unsafe atlas revision metadata for version %q: applied=%d requires at least %d partial hashes, found %d; repair atlas_schema_revisions before retrying", revision.Version, revision.Applied, revision.Applied, len(revision.PartialHashes))
		}
	}
	return nil
}
