package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anomalyco/gitwhy/pkg/config"
	"github.com/anomalyco/gitwhy/pkg/passthrough"
	"github.com/anomalyco/gitwhy/pkg/provenance"
	"github.com/anomalyco/gitwhy/pkg/storage"
)

var prViewFlags struct {
	why bool
}

var prViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a pull request with optional intent context",
	Long: `View a GitHub pull request, optionally with intent metadata.

Use --why to display provenance context alongside the standard PR view.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ghArgs := []string{"pr", "view"}
		ghArgs = append(ghArgs, args...)

		output, err := passthrough.ExecuteWithOutput(ghArgs)
		if err != nil {
			fmt.Fprint(os.Stderr, output)
			return err
		}

		fmt.Print(output)

		if prViewFlags.why && len(args) > 0 {
			prRef := args[0]

			repoPath, err := config.FindRepoRoot()
			if err != nil {
				return nil
			}

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

			record, err := backend.Get(prRef)
			if err != nil {
				return nil
			}

			printWhyPanel(record)
		}

		return nil
	},
}

func init() {
	prViewCmd.Flags().BoolVar(&prViewFlags.why, "why", false, "Show intent metadata alongside PR")
}

func printWhyPanel(record *provenance.Record) {
	fmt.Println(strings.Repeat("─", 50))
	if record.Intent.Summary != "" {
		fmt.Printf("  intent:  %s\n", record.Intent.Summary)
	}
	if record.Intent.Origin != "" {
		fmt.Printf("  origin:  %s", record.Intent.Origin)
		if record.Intent.Spec != "" {
			fmt.Printf(" (%s)", record.Intent.Spec)
		}
		fmt.Println()
	}
	fmt.Printf("  by:      %s\n", record.Attribution.By)
	if record.Intent.Spec != "" && record.Intent.SpecHash != "" {
		fmt.Printf("  spec:    %s @ %s\n", record.Intent.Spec, record.Intent.SpecHash)
	}
	if record.Context.Ticket != "" {
		fmt.Printf("  ticket:  %s\n", record.Context.Ticket)
	}
	fmt.Println(strings.Repeat("─", 50))
}


