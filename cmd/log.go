package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anomalyco/gitwhy/pkg/config"
	"github.com/anomalyco/gitwhy/pkg/provenance"
	"github.com/anomalyco/gitwhy/pkg/storage"
)

var logFlags struct {
	why bool
}

var logCmd = &cobra.Command{
	Use:   "log [path]",
	Short: "Show commit log with optional intent annotations",
	Long: `Display git log, optionally annotated with provenance metadata.

Use --why to annotate commits that have intent records with their
provenance context.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		gitArgs := []string{"-C", repoPath, "log", "--oneline"}
		gitArgs = append(gitArgs, args...)

		gitLog, err := exec.Command("git", gitArgs...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("git log: %w", err)
		}

		fmt.Print(string(gitLog))

		if logFlags.why && len(gitLog) > 0 {
			cfg, err := config.Load(repoPath)
			if err != nil {
				return nil
			}

			factory := storage.NewFactory()
			factory.Register("git-notes", storage.NewGitNotesBackend(repoPath))
			factory.Register("file", storage.NewFileBackend(config.ConfigDir(repoPath)))

			backend, err := factory.Get(string(cfg.Backend))
			if err != nil {
				return nil
			}

			lines := strings.Split(strings.TrimSpace(string(gitLog)), "\n")
			for _, line := range lines {
				parts := strings.Fields(line)
				if len(parts) == 0 {
					continue
				}
				commitHash := parts[0]
				record, err := backend.Get(commitHash)
				if err != nil {
					continue
				}
				printLogAnnotation(commitHash, record)
			}
		}

		return nil
	},
}

func printLogAnnotation(hash string, record *provenance.Record) {
	if record.Intent.Summary != "" {
		fmt.Printf("  ├─ intent: %s\n", record.Intent.Summary)
	}
	fmt.Printf("  ├─ by: %s", record.Attribution.By)
	if record.Intent.Spec != "" {
		fmt.Printf(" | spec: %s", record.Intent.Spec)
	}
	fmt.Println()
}

func init() {
	logCmd.Flags().BoolVar(&logFlags.why, "why", false, "Annotate log with intent metadata")
}
