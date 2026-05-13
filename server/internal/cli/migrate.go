package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
)

const defaultMigrationDir = "internal/ent/migrate/migrations"

func newMigrateCommand() *cobra.Command {
	var migrationDir string

	command := &cobra.Command{
		Use:   "migrate",
		Short: "Run explicit database migration commands",
	}
	command.PersistentFlags().StringVar(&migrationDir, "dir", defaultMigrationDir, "migration directory")

	command.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Apply pending Atlas versioned migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolve working directory: %w", err)
			}

			absDir, err := resolveMigrationDir(workingDir, migrationDir)
			if err != nil {
				return fmt.Errorf("resolve migration dir: %w", err)
			}

			atlasPath, err := exec.LookPath("atlas")
			if err != nil {
				return fmt.Errorf("find atlas CLI: %w", err)
			}

			command := exec.CommandContext(
				cmd.Context(),
				atlasPath,
				"migrate",
				"apply",
				"--dir", "file://"+filepath.ToSlash(absDir),
				"--url", cfg.Database.URL,
			)
			command.Stdout = cmd.OutOrStdout()
			command.Stderr = cmd.ErrOrStderr()
			command.Stdin = os.Stdin

			if err := command.Run(); err != nil {
				return fmt.Errorf("apply atlas migrations: %w", err)
			}

			return nil
		},
	})

	return command
}

func resolveMigrationDir(baseDir string, migrationDir string) (string, error) {
	if strings.TrimSpace(migrationDir) == "" {
		return "", fmt.Errorf("migration dir is required")
	}

	searchDirs := []string{migrationDir}
	if migrationDir == defaultMigrationDir {
		searchDirs = append(searchDirs, filepath.Join("server", migrationDir))
	}

	current := baseDir
	for {
		for _, relativeDir := range searchDirs {
			candidate := filepath.Join(current, relativeDir)
			info, err := os.Stat(candidate)
			if err == nil {
				if !info.IsDir() {
					return "", fmt.Errorf("migration dir %s is not a directory", candidate)
				}

				absDir, err := filepath.Abs(candidate)
				if err != nil {
					return "", fmt.Errorf("resolve migration dir %s: %w", candidate, err)
				}

				return absDir, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("cannot find migration dir %q from %s", migrationDir, baseDir)
}
