package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/config"
)

var (
	initNoHook    bool
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

		// Load existing config if present, otherwise start from defaults.
		cfg, err := config.Load(repoPath)
		if err != nil {
			return fmt.Errorf("load existing config: %w", err)
		}
		cfg.RepoPath = repoPath

		// Ensure sensible defaults without overwriting existing user settings.
		if cfg.Backend == "" {
			cfg.Backend = config.BackendGitNotes
		}
		if cfg.GitNotes == nil {
			cfg.GitNotes = &config.GitNotesConfig{Enabled: true}
		}

		if !initNoHook {
			if cfg.AutoCapture == nil {
				cfg.AutoCapture = &config.AutoCaptureConfig{
					Enabled:   true,
					DefaultBy: initDefaultBy,
				}
			} else {
				// Respect existing AutoCapture.Enabled unless user explicitly requests no-hook.
				cfg.AutoCapture.Enabled = true
				if initDefaultBy != "" {
					cfg.AutoCapture.DefaultBy = initDefaultBy
				}
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

const hookSignature = "# gitwhy auto-capture hook"

// hookStatus describes the state of an existing post-commit hook.
type hookStatus int

const (
	hookAbsent   hookStatus = iota // no hook file
	hookGitwhy                     // hook is gitwhy-signed (idempotent)
	hookForeign                    // hook exists but not gitwhy
)

func detectHook(hookPath string) hookStatus {
	data, err := os.ReadFile(hookPath)
	if err != nil {
		return hookAbsent
	}
	if strings.Contains(string(data), hookSignature) {
		return hookGitwhy
	}
	return hookForeign
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

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}

	switch detectHook(hookPath) {
	case hookGitwhy:
		fmt.Printf("  hook:    post-commit hook already installed (skipping)\n")
		return nil
	case hookForeign:
		bakPath := filepath.Join(hooksDir, ".post-commit.bak")
		if err := os.Rename(hookPath, bakPath); err != nil {
			return fmt.Errorf("backup existing hook: %w", err)
		}
		fmt.Fprintf(os.Stderr, "  warning: backed up existing hook to %s\n", bakPath)
	}

	hookContent := fmt.Sprintf(`#!/bin/sh
# gitwhy auto-capture hook — installed by ghw init
# Records provenance metadata after every git commit.
# Tries PATH first, falls back to the path at install time.
GHW_CAPTURE=1 ghw commit 2>/dev/null || GHW_CAPTURE=1 %s commit 2>/dev/null || true
`, ghwPath)

	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return fmt.Errorf("write hook: %w", err)
	}

	return nil
}
