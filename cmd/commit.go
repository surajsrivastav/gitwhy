package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anomalyco/gitwhy/pkg/config"
	"github.com/anomalyco/gitwhy/pkg/provenance"
	"github.com/anomalyco/gitwhy/pkg/storage"
)

type commitFlags struct {
	by       string
	intent   string
	spec     string
	specHash string
	origin   string
	ticket   string
	prompt   string
	model    string
	message  string
}

var commitFlag commitFlags

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit with provenance flags",
	Long: `Record a git commit with optional provenance metadata.

Captures who (or what) authored the change, why it was made, and what
spec or prompt drove it. Stored as structured metadata using the
configured storage backend.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		cfg, err := config.Load(repoPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		isCapture := os.Getenv("GHW_CAPTURE") != ""

		if !isCapture {
			gitArgs := []string{"-C", repoPath, "commit"}

			if commitFlag.message != "" {
				gitArgs = append(gitArgs, "-m", commitFlag.message)
			}

			gitArgs = append(gitArgs, args...)

			gitCmd := exec.Command("git", gitArgs...)
			gitCmd.Stdin = os.Stdin
			gitCmd.Stdout = os.Stdout
			gitCmd.Stderr = os.Stderr

			if err := gitCmd.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				return fmt.Errorf("git commit: %w", err)
			}
		}

		if isCapture || hasProvenanceFlags() {
			hashOutput, err := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD").CombinedOutput()
			if err != nil {
				return fmt.Errorf("get commit hash: %w", err)
			}
			commitHash := strings.TrimSpace(string(hashOutput))

			record := provenance.NewRecord(provenance.TargetCommit, commitHash)

			by := commitFlag.by
			if by == "" && isCapture && cfg.AutoCapture != nil && cfg.AutoCapture.DefaultBy != "" {
				by = cfg.AutoCapture.DefaultBy
			}

			if by != "" {
				switch strings.ToLower(by) {
				case "human":
					record.SetAttribution(provenance.AttributionHuman)
				case "copilot":
					record.SetAttribution(provenance.AttributionCopilot)
				default:
					if strings.HasPrefix(strings.ToLower(by), "agent:") {
						record.SetAttribution(provenance.AttributionType(by))
					} else {
						record.SetAttribution(provenance.AgentAttribution(by))
					}
				}
			}

			originStr := commitFlag.origin
			if originStr == "" {
				if by != "" && by != "human" {
					originStr = "spec"
				} else {
					originStr = "human"
				}
			}

			intentSet := commitFlag.intent != ""
			record.SetIntent(
				commitFlag.intent,
				provenance.OriginType(originStr),
				commitFlag.spec,
				commitFlag.specHash,
			)

			model := commitFlag.model
			if model == "" {
				model = resolveModel()
			}

			ticketSet := commitFlag.ticket != ""
			record.SetContext(commitFlag.ticket, commitFlag.prompt, model)

			autoFillContext(record, repoPath, intentSet, ticketSet, cfg.Summary)

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
	commitCmd.Flags().StringVarP(&commitFlag.message, "message", "m", "", "Commit message")
	commitCmd.Flags().StringVar(&commitFlag.by, "by", "", "Attribution (human, copilot, agent:<name>)")
	commitCmd.Flags().StringVar(&commitFlag.intent, "intent", "", "Description of why this change was made")
	commitCmd.Flags().StringVar(&commitFlag.spec, "spec", "", "Reference to the spec or ticket that drove this change")
	commitCmd.Flags().StringVar(&commitFlag.specHash, "spec-hash", "", "Hash of spec content at generation time")
	commitCmd.Flags().StringVar(&commitFlag.origin, "origin", "", "Origin type (human, spec, prompt, template, upstream)")
	commitCmd.Flags().StringVar(&commitFlag.ticket, "ticket", "", "Ticket or issue reference")
	commitCmd.Flags().StringVar(&commitFlag.prompt, "prompt", "", "Prompt used (if AI-generated)")
	commitCmd.Flags().StringVar(&commitFlag.model, "model", "", "Model name (auto-detected from environment if omitted; captures the model active at commit time)")
}

func hasProvenanceFlags() bool {
	return commitFlag.by != "" || commitFlag.intent != "" || commitFlag.spec != "" ||
		commitFlag.specHash != "" || commitFlag.origin != "" || commitFlag.ticket != "" ||
		commitFlag.prompt != "" || commitFlag.model != ""
}

// envModelVars lists environment variables checked for model auto-detection,
// ordered by specificity (most specific first).
var envModelVars = []string{
	"ANTHROPIC_MODEL",
	"OPENAI_MODEL",
	"CLAUDE_MODEL",
	"GITHUB_MODEL",
	"AI_MODEL",
}

// resolveModel tries to auto-detect the model name from the environment.
// If detection is ambiguous or nothing is found, it prompts the user interactively.
// Returns the empty string if detection fails and prompting is not possible.
func resolveModel() string {
	sources := make(map[string]string) // value → env var name

	for _, env := range envModelVars {
		if v := os.Getenv(env); v != "" {
			sources[v] = env
		}
	}

	switch len(sources) {
	case 0:
		return promptModel("")
	case 1:
		for v := range sources {
			fmt.Fprintf(os.Stderr, "  gitwhy: auto-detected model %q from %s\n", v, sources[v])
			return v
		}
	default:
		fmt.Fprintf(os.Stderr, "  gitwhy: multiple model candidates found\n")
		for v, env := range sources {
			fmt.Fprintf(os.Stderr, "    - %s (from %s)\n", v, env)
		}
		return promptModel("")
	}

	return ""
}

// promptModel asks the user to enter a model name interactively.
// If no terminal is available (piped input), returns empty string.
func promptModel(suggestion string) string {
	if !isTerminal() {
		return ""
	}

	if suggestion != "" {
		fmt.Fprintf(os.Stderr, "  gitwhy: enter model name [%s]: ", suggestion)
	} else {
		fmt.Fprint(os.Stderr, "  gitwhy: enter model name (or leave empty): ")
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return suggestion
	}
	return input
}

// isTerminal returns true if stdin is a terminal (interactive session).
func isTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
