package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/config"
	"github.com/surajsrivastav/gitwhy/pkg/provenance"
	"github.com/surajsrivastav/gitwhy/pkg/storage"
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
			if by == "" {
				by = resolveAgent()
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
				switch {
				case by == "human":
					originStr = "human"
				case by != "":
					originStr = "spec"
				default:
					originStr = string(provenance.OriginUnknown)
				}
			}

			intentSet := commitFlag.intent != ""
			record.SetIntent(
				commitFlag.intent,
				provenance.OriginType(originStr),
				commitFlag.spec,
				commitFlag.specHash,
			)

			session, _ := config.LoadSession(repoPath)

			model := commitFlag.model
			if model == "" && session != nil {
				model = session.Model
			}
			if model == "" {
				model = resolveModel()
			}
			if model == "" {
				model = cfg.DefaultModel
			}

			prompt := commitFlag.prompt
			if prompt == "" && session != nil {
				prompt = session.Prompt
			}
			if prompt == "" {
				prompt = resolvePrompt()
			}
			if commitFlag.prompt == "" {
				commitFlag.prompt = prompt
			}

			if commitFlag.ticket == "" && session != nil {
				commitFlag.ticket = session.Ticket
			}
			if commitFlag.origin == "" && session != nil {
				originStr = session.Origin
			}

			ticketSet := commitFlag.ticket != ""
			record.SetContext(commitFlag.ticket, commitFlag.prompt, model)

			autoFillContext(record, repoPath, intentSet, ticketSet, cfg.Summary)

			if record.Intent.Summary == "" {
				record.Intent.Summary = "unknown"
			}
			if record.Intent.Origin == "" {
				record.Intent.Origin = provenance.OriginUnknown
			}
			if record.Context.Model == "" {
				record.Context.Model = "unknown"
			}
			if record.Context.Ticket == "" {
				record.Context.Ticket = "unknown"
			}
			if record.Context.Prompt == "" {
				record.Context.Prompt = "unknown"
			}
			if record.Context.Branch == "" {
				record.Context.Branch = "unknown"
			}

			factory := storage.NewFactory()
			factory.Register("git-notes", storage.NewGitNotesBackend(repoPath))
			factory.Register("file", storage.NewFileBackend(config.ConfigDir(repoPath)))

			backend, err := factory.Get(string(cfg.Backend))
			if err != nil {
				return fmt.Errorf("get backend: %w", err)
			}

			if err := backend.Store(record); err != nil {
				if isCapture {
					_ = config.AppendCaptureError(repoPath, commitHash, err.Error())
				}
				return fmt.Errorf("store provenance record: %w", err)
			}

			fmt.Printf("  gitwhy: provenance recorded (%s backend)\n", backend.Name())
			_ = config.WriteLastCapture(repoPath, commitHash, backend.Name())
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
	"CLAUDE_CODE_MODEL",
	"COPILOT_AGENT_MODEL",
	"COPILOT_MODEL",
	"ANTHROPIC_MODEL",
	"OPENAI_MODEL",
	"CLAUDE_MODEL",
	"GITHUB_MODEL",
	"AI_MODEL",
	"GITWHY_MODEL",
	"GHW_MODEL",
}

var envPromptVars = []string{
	"COPILOT_AGENT_PROMPT",
	"GITWHY_PROMPT",
	"GHW_PROMPT",
}

// resolveAgent tries to detect the AI agent from the environment.
// It parses AI_AGENT (e.g. "claude-code/2.1.126/agent") and returns the
// tool name (e.g. "claude-code"), which becomes "agent:claude-code" in the record.
// Falls back to Copilot detection via COPILOT_AGENT_MODEL or COPILOT_MODEL.
func resolveAgent() string {
	if v := os.Getenv("AI_AGENT"); v != "" {
		parts := strings.SplitN(v, "/", 2)
		return parts[0]
	}
	if os.Getenv("COPILOT_AGENT_MODEL") != "" || os.Getenv("COPILOT_MODEL") != "" {
		return "copilot"
	}
	return ""
}

// resolveModel tries to auto-detect the model name from the environment.
// Returns the first match in envModelVars priority order, or empty string.
func resolveModel() string {
	for _, env := range envModelVars {
		if v := os.Getenv(env); v != "" {
			fmt.Fprintf(os.Stderr, "  gitwhy: auto-detected model %q from %s\n", v, env)
			return v
		}
	}
	return ""
}

// resolvePrompt tries to auto-detect the prompt from the environment.
// Returns the prompt string or empty if no prompt is available.
func resolvePrompt() string {
	for _, env := range envPromptVars {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}

