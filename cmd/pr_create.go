package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/config"
	"github.com/surajsrivastav/gitwhy/pkg/passthrough"
	"github.com/surajsrivastav/gitwhy/pkg/provenance"
	"github.com/surajsrivastav/gitwhy/pkg/storage"
)

var prCreateFlags struct {
	intent  string
	origin  string
	spec    string
	agent   string
	title   string
	body    string
}

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request with intent capture",
	Long: `Create a GitHub pull request with optional intent and provenance metadata.

Works like gh pr create but adds structured intent capture that is stored
alongside the PR for future reference.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !passthrough.IsGHAvailable() {
			return fmt.Errorf("gh not found in PATH — gh pr create requires the GitHub CLI")
		}
		ghArgs := []string{"pr", "create"}
		ghArgs = append(ghArgs, args...)

		if prCreateFlags.title != "" {
			ghArgs = append(ghArgs, "--title", prCreateFlags.title)
		}
		if prCreateFlags.body != "" {
			ghArgs = append(ghArgs, "--body", prCreateFlags.body)
		}

		output, err := passthrough.ExecuteWithOutput(ghArgs)
		if err != nil {
			fmt.Fprint(os.Stderr, output)
			return err
		}

		fmt.Print(output)

		prURL := strings.TrimSpace(output)
		prNumber := extractPRNumber(prURL)

		if prNumber != "" && hasPRProvenanceFlags() {
			repoPath, err := config.FindRepoRoot()
			if err != nil {
				return fmt.Errorf("find repo: %w", err)
			}

			agent := prCreateFlags.agent
			if agent == "" {
				agent = resolveAgent()
			}
			attribution := provenance.AttributionUnknown
			if agent != "" {
				attribution = provenance.AgentAttribution(agent)
			}

			originStr := prCreateFlags.origin
			if originStr == "" {
				if agent != "" {
					originStr = "spec"
				} else {
					originStr = string(provenance.OriginUnknown)
				}
			}

			intent := prCreateFlags.intent
			if intent == "" {
				intent = "unknown"
			}

			record := provenance.NewRecord(provenance.TargetPR, prNumber)
			record.SetAttribution(attribution)
			record.SetIntent(intent, provenance.OriginType(originStr), prCreateFlags.spec, "")

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

			if err := backend.Store(record); err != nil {
				return fmt.Errorf("store provenance record: %w", err)
			}

			fmt.Printf("  gitwhy: provenance recorded (%s backend)\n", backend.Name())
		}

		return nil
	},
}

func init() {
	prCreateCmd.Flags().StringVar(&prCreateFlags.intent, "intent", "", "Description of why this change was made")
	prCreateCmd.Flags().StringVar(&prCreateFlags.origin, "origin", "", "Origin type (human, spec, prompt, template, upstream)")
	prCreateCmd.Flags().StringVar(&prCreateFlags.spec, "spec", "", "Reference to the spec or ticket that drove this change")
	prCreateCmd.Flags().StringVar(&prCreateFlags.agent, "agent", "", "Agent name if AI-generated")
	prCreateCmd.Flags().StringVarP(&prCreateFlags.title, "title", "t", "", "PR title")
	prCreateCmd.Flags().StringVarP(&prCreateFlags.body, "body", "b", "", "PR body")
}

func extractPRNumber(url string) string {
	parts := strings.Split(strings.TrimSpace(url), "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	for _, c := range last {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return last
}

func hasPRProvenanceFlags() bool {
	return prCreateFlags.intent != "" || prCreateFlags.origin != "" ||
		prCreateFlags.spec != "" || prCreateFlags.agent != ""
}
