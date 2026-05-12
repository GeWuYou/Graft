package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

			absDir, err := filepath.Abs(migrationDir)
			if err != nil {
				return fmt.Errorf("resolve migration dir: %w", err)
			}

			if _, err := os.Stat(absDir); err != nil {
				return fmt.Errorf("stat migration dir %s: %w", absDir, err)
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
