package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anomalyco/gitwhy/pkg/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gitwhy in a repository",
	Long: `Bootstraps gitwhy configuration in the current repository.

Creates .gitwhy/config.yaml with default settings and the git-notes storage backend.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}

		if config.IsInitialized(repoPath) {
			fmt.Fprintf(os.Stderr, "gitwhy already initialized in %s\n", config.ConfigDir(repoPath))
			return nil
		}

		cfg := config.DefaultConfig()
		cfg.RepoPath = repoPath
		cfg.Backend = config.BackendGitNotes
		cfg.GitNotes = &config.GitNotesConfig{Enabled: true}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("gitwhy initialized in %s\n", config.ConfigPath(repoPath))
		fmt.Printf("  backend: %s\n", cfg.Backend)
		return nil
	},
}
