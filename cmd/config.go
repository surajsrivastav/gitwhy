package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/surajsrivastav/gitwhy/pkg/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gitwhy configuration",
	Long: `View and modify gitwhy configuration settings.

Configuration is stored in .gitwhy/config.yaml in the repository root.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		cfg, err := config.Load(repoPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if len(args) == 0 {
			data, _ := yaml.Marshal(cfg)
			fmt.Print(string(data))
			return nil
		}

		switch args[0] {
		case "backend":
			fmt.Println(cfg.Backend)
		default:
			return fmt.Errorf("unknown config key: %s", args[0])
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		cfg, err := config.Load(repoPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		key, value := args[0], args[1]
		switch key {
		case "backend":
			switch value {
			case "git-notes", "file", "metadata":
				cfg.Backend = config.StorageBackend(value)
			default:
				return fmt.Errorf("invalid backend: %s (valid: git-notes, file, metadata)", value)
			}
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("  %s = %s\n", key, value)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}
