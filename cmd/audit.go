package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/audit"
	"github.com/surajsrivastav/gitwhy/pkg/config"
	"github.com/surajsrivastav/gitwhy/pkg/storage"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit provenance data",
	Long:  "Export and summarize provenance data for analysis and compliance.",
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export provenance data as JSON or CSV",
	Long: `Export all provenance records for a date range.

Supports JSON and CSV output formats for integration with dashboards,
audit pipelines, and compliance tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		records, err := backend.List()
		if err != nil {
			return fmt.Errorf("list records: %w", err)
		}

		formatStr, _ := cmd.Flags().GetString("format")
		fromStr, _ := cmd.Flags().GetString("from")
		toStr, _ := cmd.Flags().GetString("to")

		opts := audit.ExportOptions{
			Format: audit.ExportFormat(formatStr),
		}

		if fromStr != "" {
			opts.From, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				return fmt.Errorf("parse --from date: %w", err)
			}
		}
		if toStr != "" {
			opts.To, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				return fmt.Errorf("parse --to date: %w", err)
			}
		}

		output, err := audit.GenerateExport(records, opts)
		if err != nil {
			return fmt.Errorf("generate export: %w", err)
		}

		fmt.Print(output)
		return nil
	},
}

var auditSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show a summary of provenance data",
	Long: `Display a dashboard-style summary of provenance coverage.

Shows AI percentage, spec coverage, drift flags, and agent breakdown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		records, err := backend.List()
		if err != nil {
			return fmt.Errorf("list records: %w", err)
		}

		summary := audit.GenerateSummary(records)
		fmt.Print(summary.String())
		return nil
	},
}

func init() {
	auditExportCmd.Flags().String("format", "json", "Output format (json, csv)")
	auditExportCmd.Flags().String("from", "", "Start date (YYYY-MM-DD)")
	auditExportCmd.Flags().String("to", "", "End date (YYYY-MM-DD)")

	auditCmd.AddCommand(auditExportCmd)
	auditCmd.AddCommand(auditSummaryCmd)
}
