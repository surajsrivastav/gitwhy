package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/audit"
	"github.com/surajsrivastav/gitwhy/pkg/config"
	"github.com/surajsrivastav/gitwhy/pkg/provenance"
	"github.com/surajsrivastav/gitwhy/pkg/storage"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit provenance data",
	Long: `Export, search, and summarize provenance data.

gitwhy tracks developer provenance today — who/what wrote a change and why.
It is not a certified compliance product; see PRD.md's Compliance Roadmap
for what a compliance-ready tier would still need (signed records,
retention policy, access controls).`,
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export provenance data as JSON or CSV",
	Long: `Export all provenance records for a date range.

Supports JSON and CSV output formats for integration with dashboards and
reporting pipelines.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, backend, err := repoAndBackend()
		if err != nil {
			return err
		}
		_ = repoPath

		records, err := backend.List()
		if err != nil {
			return fmt.Errorf("list records: %w", err)
		}

		formatStr, _ := cmd.Flags().GetString("format")
		fromStr, _ := cmd.Flags().GetString("from")
		toStr, _ := cmd.Flags().GetString("to")

		opts := audit.ExportOptions{
			Format: audit.ExportFormat(formatStr),
			Filter: filterFromFlags(cmd),
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

var auditSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search provenance records with a table-formatted view",
	Long: `Filter provenance records and print them as a human-readable table.

Unlike "ghw audit export", this is meant for interactive use rather than
machine consumption.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, backend, err := repoAndBackend()
		if err != nil {
			return err
		}

		records, err := backend.List()
		if err != nil {
			return fmt.Errorf("list records: %w", err)
		}

		filtered := audit.ApplyFilter(records, filterFromFlags(cmd))
		fmt.Print(audit.TableFormat(filtered))
		return nil
	},
}

var auditGapsCmd = &cobra.Command{
	Use:   "gaps",
	Short: "List commits with no provenance record",
	Long: `Show every commit in range that has no provenance record.

Use --since to scope the search to a date (YYYY-MM-DD) or a git ref/branch
point (e.g. "main"). Use --fail-on-missing to exit 1 when gaps are found,
for wiring into CI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, backend, err := repoAndBackend()
		if err != nil {
			return err
		}

		since, _ := cmd.Flags().GetString("since")
		failOnMissing, _ := cmd.Flags().GetBool("fail-on-missing")

		commits, err := listCommits(repoPath, since)
		if err != nil {
			return fmt.Errorf("list commits: %w", err)
		}

		records, err := backend.List()
		if err != nil {
			return fmt.Errorf("list records: %w", err)
		}

		gaps := audit.FindGaps(commits, records)
		fmt.Print(audit.GapReport(gaps, len(commits)))

		if failOnMissing && len(gaps) > 0 {
			return &exitCodeError{code: 1}
		}
		return nil
	},
}

var auditVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify provenance records have not been tampered with",
	Long: `Batch-verify every provenance record's stored record_hash against its
own content, reporting any record whose hash no longer matches (indicating
the note was edited after being recorded).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, backend, err := repoAndBackend()
		if err != nil {
			return err
		}

		records, err := backend.List()
		if err != nil {
			return fmt.Errorf("list records: %w", err)
		}

		var tampered, unsigned int
		for _, r := range records {
			switch r.VerifyHash() {
			case provenance.HashValid:
				continue
			case provenance.HashMissing:
				unsigned++
			case provenance.HashTampered:
				tampered++
				fmt.Printf("  TAMPERED  %s  (record_hash mismatch)\n", r.Target.Ref)
			}
		}

		fmt.Printf("\n  %d records checked, %d tampered, %d without a hash\n", len(records), tampered, unsigned)
		if tampered > 0 {
			return &exitCodeError{code: 1}
		}
		return nil
	},
}

var auditSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show a summary of provenance data",
	Long: `Display a dashboard-style summary of provenance coverage.

Shows AI percentage, spec coverage, drift flags, and agent breakdown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, backend, err := repoAndBackend()
		if err != nil {
			return err
		}

		records, err := backend.List()
		if err != nil {
			return fmt.Errorf("list records: %w", err)
		}

		summary := audit.GenerateSummary(records)

		if commits, err := listCommits(repoPath, ""); err == nil {
			gaps := audit.FindGaps(commits, records)
			summary.SetCoverage(len(commits), len(commits)-len(gaps))
		}

		fmt.Print(summary.String())
		return nil
	},
}

// repoAndBackend resolves the repo root and configured storage backend,
// the pair almost every audit subcommand needs.
func repoAndBackend() (string, storage.Backend, error) {
	repoPath, err := config.FindRepoRoot()
	if err != nil {
		return "", nil, fmt.Errorf("find repo: %w", err)
	}

	cfg, err := config.Load(repoPath)
	if err != nil {
		return "", nil, fmt.Errorf("load config: %w", err)
	}

	factory := storage.NewFactory()
	factory.Register("git-notes", storage.NewGitNotesBackend(repoPath))
	factory.Register("file", storage.NewFileBackend(config.ConfigDir(repoPath)))

	backend, err := factory.Get(string(cfg.Backend))
	if err != nil {
		return "", nil, fmt.Errorf("get backend: %w", err)
	}

	return repoPath, backend, nil
}

// listCommits returns commit hash/date/subject triples for the repo.
// If since is a YYYY-MM-DD date, scopes to commits after that date.
// If since is a git ref, scopes to commits since that ref (ref..HEAD).
// If since is empty, returns the full history.
func listCommits(repoPath, since string) ([]audit.CommitInfo, error) {
	gitArgs := []string{"-C", repoPath, "log", "--format=%H|%aI|%s"}

	if since != "" {
		if _, err := time.Parse("2006-01-02", since); err == nil {
			gitArgs = append(gitArgs, "--since="+since)
		} else {
			gitArgs = append(gitArgs, since+"..HEAD")
		}
	}

	out, err := exec.Command("git", gitArgs...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log: %w\n%s", err, string(out))
	}

	var commits []audit.CommitInfo
	for _, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		commits = append(commits, audit.CommitInfo{Hash: parts[0], Date: parts[1], Subject: parts[2]})
	}
	return commits, nil
}

// registerFilterFlags adds the shared --by/--model/--ticket/--origin/--missing
// filter flags to a command.
func registerFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("by", "", "Filter by attribution (e.g. agent:claude-code, human)")
	cmd.Flags().String("model", "", "Filter by model")
	cmd.Flags().String("ticket", "", "Filter by ticket")
	cmd.Flags().String("origin", "", "Filter by origin")
	cmd.Flags().String("missing", "", "Only show records missing a field (model, ticket, spec, origin, prompt)")
}

func filterFromFlags(cmd *cobra.Command) audit.FilterOptions {
	by, _ := cmd.Flags().GetString("by")
	model, _ := cmd.Flags().GetString("model")
	ticket, _ := cmd.Flags().GetString("ticket")
	origin, _ := cmd.Flags().GetString("origin")
	missing, _ := cmd.Flags().GetString("missing")
	return audit.FilterOptions{By: by, Model: model, Ticket: ticket, Origin: origin, Missing: missing}
}

func init() {
	auditExportCmd.Flags().String("format", "json", "Output format (json, csv)")
	auditExportCmd.Flags().String("from", "", "Start date (YYYY-MM-DD)")
	auditExportCmd.Flags().String("to", "", "End date (YYYY-MM-DD)")
	registerFilterFlags(auditExportCmd)

	registerFilterFlags(auditSearchCmd)

	auditGapsCmd.Flags().String("since", "", "Scope to a date (YYYY-MM-DD) or git ref (e.g. main)")
	auditGapsCmd.Flags().Bool("fail-on-missing", false, "Exit 1 if any gaps are found (for CI)")

	auditCmd.AddCommand(auditExportCmd)
	auditCmd.AddCommand(auditSearchCmd)
	auditCmd.AddCommand(auditGapsCmd)
	auditCmd.AddCommand(auditVerifyCmd)
	auditCmd.AddCommand(auditSummaryCmd)
}
