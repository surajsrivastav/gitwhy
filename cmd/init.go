package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/config"
)

var (
	initNoHook   bool
	initDefaultBy string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gitwhy in a repository",
	Long: `Bootstraps gitwhy configuration in the current repository.

Creates .gitwhy/config.yaml with default settings and installs a git
post-commit hook for automatic provenance capture after every commit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}

		cfg := config.DefaultConfig()
		cfg.RepoPath = repoPath
		cfg.Backend = config.BackendGitNotes
		cfg.GitNotes = &config.GitNotesConfig{Enabled: true}

		if !initNoHook {
			cfg.AutoCapture = &config.AutoCaptureConfig{
				Enabled:   true,
				DefaultBy: initDefaultBy,
			}
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("gitwhy initialized in %s\n", config.ConfigPath(repoPath))
		fmt.Printf("  backend: %s\n", cfg.Backend)

		if !initNoHook {
			if err := installHook(repoPath); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: failed to install hook: %v\n", err)
			} else {
				fmt.Printf("  hook:    post-commit hook installed\n")
				if initDefaultBy != "" {
					fmt.Printf("  default attribution: %s\n", initDefaultBy)
				}
			}
		}

		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initNoHook, "no-hook", false, "Skip installing git post-commit hook")
	initCmd.Flags().StringVar(&initDefaultBy, "default-by", "", "Default attribution for auto-capture (e.g. agent:opencode)")
}

func installHook(repoPath string) error {
	ghwPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find ghw binary: %w", err)
	}

	ghwPath, err = filepath.Abs(ghwPath)
	if err != nil {
		return fmt.Errorf("resolve ghw path: %w", err)
	}

	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "post-commit")

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		fmt.Fprintf(os.Stderr, "  warning: post-commit hook already exists, overwriting\n")
	}

	hookContent := fmt.Sprintf(`#!/bin/sh
# gitwhy auto-capture hook — installed by ghw init
# Records provenance metadata after every git commit.
# Tries PATH first, falls back to the path at install time.
GHW_CAPTURE=1 ghw commit 2>/dev/null || GHW_CAPTURE=1 %s commit 2>/dev/null || true
`, ghwPath)

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}

	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return fmt.Errorf("write hook: %w", err)
	}

	return nil
}
