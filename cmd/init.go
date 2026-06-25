package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/config"
	"github.com/surajsrivastav/gitwhy/pkg/storage"
)

var (
	initNoHook      bool
	initDefaultBy   string
	initRemote      string
	initNoNotesSync bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gitwhy in a repository",
	Long: `Bootstraps gitwhy configuration in the current repository.

Creates .gitwhy/config.yaml with default settings and installs a git
post-commit hook for automatic provenance capture after every commit.

Also configures the git remote so that provenance notes (refs/notes/gitwhy)
are pushed and fetched alongside normal git push/pull, making them
automatically shared with the whole team.`,
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

		if err := writeGitignore(repoPath); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: failed to write .gitwhy/.gitignore: %v\n", err)
		}

		if !initNoNotesSync {
			if err := configureNotesSharing(repoPath, initRemote); err != nil {
				fmt.Fprintf(os.Stderr, "  warning: notes sharing not configured: %v\n", err)
			}
		}

		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initNoHook, "no-hook", false, "Skip installing git post-commit hook")
	initCmd.Flags().StringVar(&initDefaultBy, "default-by", "", "Default attribution for auto-capture (e.g. agent:opencode)")
	initCmd.Flags().StringVar(&initRemote, "remote", "origin", "Git remote to configure for notes sharing")
	initCmd.Flags().BoolVar(&initNoNotesSync, "no-notes-sync", false, "Skip configuring git remote for notes push/fetch")
}

// configureNotesSharing adds push and fetch refspecs for refs/notes/gitwhy to
// the given remote so that `git push` and `git fetch` automatically sync
// provenance notes across the team. The operation is idempotent.
func configureNotesSharing(repoPath, remote string) error {
	// Verify the remote exists before touching its config.
	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", remote).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remote %q not found — run with --no-notes-sync to skip, or set --remote to a valid remote name", remote)
	}
	remoteURL := strings.TrimSpace(string(out))

	notesRef := storage.NotesRef // refs/notes/gitwhy
	pushSpec := notesRef + ":" + notesRef
	fetchSpec := "+" + notesRef + ":" + notesRef

	// Read all existing push/fetch refspecs for this remote.
	existingPush := remoteRefspecs(repoPath, remote, "push")
	existingFetch := remoteRefspecs(repoPath, remote, "fetch")

	addedPush, addedFetch := false, false

	if !containsSpec(existingPush, pushSpec) {
		if out, err := exec.Command("git", "-C", repoPath, "config", "--add",
			"remote."+remote+".push", pushSpec).CombinedOutput(); err != nil {
			return fmt.Errorf("set push refspec: %w\n%s", err, out)
		}
		addedPush = true
	}

	if !containsSpec(existingFetch, fetchSpec) {
		if out, err := exec.Command("git", "-C", repoPath, "config", "--add",
			"remote."+remote+".fetch", fetchSpec).CombinedOutput(); err != nil {
			return fmt.Errorf("set fetch refspec: %w\n%s", err, out)
		}
		addedFetch = true
	}

	if !addedPush && !addedFetch {
		fmt.Printf("  notes:   sharing already configured for remote %q (%s)\n", remote, remoteURL)
		return nil
	}

	fmt.Printf("  notes:   sharing configured for remote %q (%s)\n", remote, remoteURL)
	fmt.Printf("           git push / git fetch will now sync provenance notes\n")
	fmt.Printf("           run `git push %s %s` to publish existing notes\n", remote, notesRef)
	return nil
}

// remoteRefspecs returns the list of push or fetch refspecs for a remote.
func remoteRefspecs(repoPath, remote, direction string) []string {
	out, err := exec.Command("git", "-C", repoPath, "config", "--get-all",
		"remote."+remote+"."+direction).CombinedOutput()
	if err != nil {
		return nil
	}
	var specs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			specs = append(specs, line)
		}
	}
	return specs
}

func containsSpec(specs []string, target string) bool {
	for _, s := range specs {
		if s == target {
			return true
		}
	}
	return false
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
		// Always overwrite so re-running ghw init updates the hook to the latest template.
	case hookForeign:
		bakPath := filepath.Join(hooksDir, ".post-commit.bak")
		if err := os.Rename(hookPath, bakPath); err != nil {
			return fmt.Errorf("backup existing hook: %w", err)
		}
		fmt.Fprintf(os.Stderr, "  warning: backed up existing hook to %s\n", bakPath)
	}

	version := commitVersion
	if version == "" {
		version = "dev"
	}
	date := buildDate
	if date == "" {
		date = "unknown"
	}

	hookContent := fmt.Sprintf(`#!/bin/sh
# gitwhy auto-capture hook — installed by ghw@%s on %s
# Records provenance metadata after every git commit.
# Tries PATH first, falls back to the path at install time.
GHW_CAPTURE=1 ghw commit || GHW_CAPTURE=1 %s commit || true
`, version, date, ghwPath)

	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return fmt.Errorf("write hook: %w", err)
	}

	fmt.Printf("  hook:    post-commit hook installed\n")
	return nil
}

// writeGitignore writes (unconditionally) a .gitwhy/.gitignore that prevents
// gitwhy local-state files from being accidentally committed.
func writeGitignore(repoPath string) error {
	dir := config.ConfigDir(repoPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	content := "# gitwhy local state — do not commit\nlast-capture\ncapture-errors.log\nsession.yaml\n"
	gitignorePath := filepath.Join(dir, config.GitignoreFile)
	if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}
