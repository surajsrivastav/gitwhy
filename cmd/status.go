package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show gitwhy installation and capture status",
	Long: `Display the current status of gitwhy in the repository.

Shows whether the post-commit hook is installed, which backend is
configured, when the last provenance capture occurred, and whether
any capture errors have been logged.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}

		cfg, err := config.Load(repoPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		hookPath := filepath.Join(repoPath, ".git", "hooks", "post-commit")
		hookState := detectHook(hookPath)

		receipt, err := config.ReadLastCapture(repoPath)
		if err != nil {
			// Non-fatal: treat as no capture.
			receipt = nil
		}

		errCount, err := config.CountCaptureErrors(repoPath)
		if err != nil {
			// Non-fatal: treat as zero.
			errCount = 0
		}

		fmt.Println("  gitwhy status")
		fmt.Println("  ────────────────────────────────────")

		// Hook line.
		switch hookState {
		case hookGitwhy:
			fmt.Println("  hook:          installed ✓")
		case hookForeign:
			fmt.Println("  hook:          foreign hook present  →  run: ghw init")
		default:
			fmt.Println("  hook:          not installed  →  run: ghw init")
		}

		// Backend line.
		backend := string(cfg.Backend)
		if backend == "" {
			backend = config.DefaultBackend
		}
		fmt.Printf("  backend:       %s\n", backend)

		// Last capture line.
		if receipt == nil {
			fmt.Println("  last capture:  never")
		} else {
			shortHash := receipt.Commit
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			t, err := time.Parse(time.RFC3339, receipt.Timestamp)
			if err != nil {
				fmt.Printf("  last capture:  %s (%s) ✓\n", receipt.Timestamp, shortHash)
			} else {
				ago := time.Since(t).Round(time.Minute)
				fmt.Printf("  last capture:  %s ago (%s) ✓\n", ago, shortHash)
			}
		}

		// Errors line.
		if errCount == 0 {
			fmt.Println("  errors:        none ✓")
		} else {
			fmt.Printf("  errors:        %d  →  check .gitwhy/%s\n", errCount, config.CaptureErrorsFile)
		}

		return nil
	},
}
