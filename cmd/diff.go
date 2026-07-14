package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/drift"
)

var diffFlags struct {
	drift    bool
	spec     string
	specHash string
}

var diffCmd = &cobra.Command{
	Use:   "diff [file]",
	Short: "Show diff with optional drift detection",
	Long: `Show git diff, optionally with drift detection against a spec.

Use --drift to compare the current state against the originating spec
and detect if the implementation has diverged. --spec accepts a local file
path, a GitHub issue URL, or any http(s) URL — the spec is re-fetched and
re-hashed at compare time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if diffFlags.drift && diffFlags.spec != "" && len(args) > 0 {
			filePath := args[0]

			report, err := drift.DetectDrift(filePath, diffFlags.spec, diffFlags.spec, diffFlags.specHash, "")
			if err != nil {
				return fmt.Errorf("drift detection: %w", err)
			}

			fmt.Print(report.String())
			return nil
		}

		gitArgs := []string{"diff"}
		gitArgs = append(gitArgs, args...)

		cmdDiff := exec.Command("git", gitArgs...)
		cmdDiff.Stdin = os.Stdin
		cmdDiff.Stdout = os.Stdout
		cmdDiff.Stderr = os.Stderr

		if err := cmdDiff.Run(); err != nil {
			return fmt.Errorf("git diff: %w", err)
		}

		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVar(&diffFlags.drift, "drift", false, "Detect spec drift")
	diffCmd.Flags().StringVar(&diffFlags.spec, "spec", "", "Spec to compare against (file path, GitHub issue URL, or http(s) URL)")
	diffCmd.Flags().StringVar(&diffFlags.specHash, "spec-hash", "", "Spec hash recorded at generation time (compare against the current fetch)")
}


