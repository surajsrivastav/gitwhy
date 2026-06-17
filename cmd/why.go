package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/config"
	"github.com/surajsrivastav/gitwhy/pkg/provenance"
	"github.com/surajsrivastav/gitwhy/pkg/storage"
)

var whyCmd = &cobra.Command{
	Use:   "why <commit>",
	Short: "Show full provenance record for a commit",
	Long: `Display the complete provenance record for a specific commit or PR.

Shows who authored the change, why it was made, what spec drove it,
and all associated metadata.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]

		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		cfg, err := config.Load(repoPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		factory := storage.NewFactory()
		factory.Register("git-notes", storage.NewGitNotesBackend(repoPath))
		factory.Register("file", storage.NewFileBackend(config.ConfigDir(repoPath)))

		backend, err := factory.Get(string(cfg.Backend))
		if err != nil {
			return fmt.Errorf("get backend: %w", err)
		}

		record, err := backend.Get(ref)
		if err != nil {
			return fmt.Errorf("no provenance record for %s", ref)
		}

		printFullProvenance(record)
		return nil
	},
}

func printFullProvenance(record *provenance.Record) {
	fmt.Println("  gitwhy provenance record")
	fmt.Println("  ─────────────────────────")
	fmt.Printf("  schema:    %s\n", record.Schema)
	fmt.Printf("  target:    %s %s\n", record.Target.Type, record.Target.Ref)
	fmt.Printf("  by:        %s\n", record.Attribution.By)
	fmt.Printf("  when:      %s\n", record.Attribution.Timestamp)

	if t, err := time.Parse(time.RFC3339, record.Attribution.Timestamp); err == nil {
		fmt.Printf("            (%s ago)\n", time.Since(t).Round(time.Minute))
	}

	fmt.Println()
	if record.Intent.Summary != "" {
		fmt.Printf("  intent:    %s\n", record.Intent.Summary)
	}
	if record.Intent.Spec != "" {
		fmt.Printf("  spec:      %s", record.Intent.Spec)
		if record.Intent.SpecHash != "" {
			fmt.Printf(" @ %s", record.Intent.SpecHash)
		}
		fmt.Println()
	}
	if record.Intent.Origin != "" {
		fmt.Printf("  origin:    %s\n", record.Intent.Origin)
	}
	fmt.Println()

	if record.Context.Ticket != "" || record.Context.Prompt != "" || record.Context.Model != "" ||
		record.Context.Branch != "" {
		fmt.Println("  context:")
		if record.Context.Ticket != "" {
			fmt.Printf("    ticket:   %s\n", record.Context.Ticket)
		}
		if record.Context.Prompt != "" {
			fmt.Printf("    prompt:   %s\n", record.Context.Prompt)
		}
		if record.Context.Model != "" {
			fmt.Printf("    model:    %s\n", record.Context.Model)
		}
		if record.Context.Branch != "" {
			fmt.Printf("    branch:   %s\n", record.Context.Branch)
		}
	}
}


