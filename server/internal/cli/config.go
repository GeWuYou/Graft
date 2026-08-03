package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
)

// configValidateOptions 保存配置校验命令的显式来源选择。
type configValidateOptions struct {
	envFile     string
	composeFile string
	set         []string
	format      string
}

// newConfigCommand 创建部署配置治理命令树。
func newConfigCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate configuration contracts",
	}
	command.AddCommand(newConfigValidateCommand())
	return command
}

// newConfigValidateCommand 创建不连接外部资源的配置契约校验命令。
func newConfigValidateCommand() *cobra.Command {
	opts := configValidateOptions{format: "text"}
	command := &cobra.Command{
		Use:          "validate",
		Short:        "Validate effective environment configuration",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigValidate(cmd, opts)
		},
	}
	command.Flags().StringVar(&opts.envFile, "env-file", "", "read configuration values from this .env file")
	command.Flags().StringVar(&opts.composeFile, "compose-file", "", "validate this Docker Compose file before checking environment values")
	command.Flags().StringArrayVar(&opts.set, "set", nil, "override a configuration value as KEY=VALUE; repeatable")
	command.Flags().StringVar(&opts.format, "format", "text", "report format: text, json, or patch")
	return command
}

func runConfigValidate(cmd *cobra.Command, opts configValidateOptions) error {
	if opts.format != "text" && opts.format != "json" && opts.format != "patch" {
		return fmt.Errorf("unsupported configuration report format %q", opts.format)
	}
	if err := config.ValidateComposeFile(opts.composeFile); err != nil {
		report := config.Report{Findings: []config.Finding{{
			Code:        "compose",
			Severity:    config.SeverityError,
			Key:         "compose",
			Description: err.Error(),
		}}}
		if writeErr := config.WriteReport(cmd.OutOrStdout(), report, opts.format); writeErr != nil {
			return writeErr
		}
		return err
	}
	report, err := config.ResolveAndValidate(config.ResolveOptions{EnvFile: opts.envFile, Set: opts.set, DiscoverEnvFile: true})
	if writeErr := config.WriteReport(cmd.OutOrStdout(), report, opts.format); writeErr != nil {
		return writeErr
	}
	return err
}
