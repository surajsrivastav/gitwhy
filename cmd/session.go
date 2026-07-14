package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/config"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage the active provenance session",
	Long: `Track the model, ticket, and prompt for the coding session in progress.

The post-commit hook reads .gitwhy/session.yaml (when present) before
falling back to environment-variable detection or config defaults, so the
model that actually wrote the code gets recorded — not whatever happens
to be active at commit time.`,
}

var sessionStartFlags struct {
	model  string
	ticket string
	prompt string
	origin string
}

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a session, recording model/ticket/prompt for later commits",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		if sessionStartFlags.model == "" && sessionStartFlags.ticket == "" &&
			sessionStartFlags.prompt == "" && sessionStartFlags.origin == "" {
			return promptSessionInteractive(repoPath)
		}

		session := &config.SessionState{
			Model:  sessionStartFlags.model,
			Ticket: sessionStartFlags.ticket,
			Prompt: sessionStartFlags.prompt,
			Origin: sessionStartFlags.origin,
		}

		if err := config.SaveSession(repoPath, session); err != nil {
			return fmt.Errorf("save session: %w", err)
		}

		fmt.Println("  gitwhy: session started")
		printSession(session, "")
		return nil
	},
}

var sessionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the active session",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		session, err := config.LoadSession(repoPath)
		if err != nil {
			return fmt.Errorf("load session: %w", err)
		}

		if session == nil {
			fmt.Println("  no active session  →  run: ghw session start")
			return nil
		}

		age := ""
		if info, statErr := sessionFileAge(repoPath); statErr == nil {
			age = info
		}

		fmt.Println("  gitwhy session")
		printSession(session, age)
		return nil
	},
}

var sessionSetCmd = &cobra.Command{
	Use:   "set <field> <value>",
	Short: "Update a single field of the active session",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		session, err := config.LoadSession(repoPath)
		if err != nil {
			return fmt.Errorf("load session: %w", err)
		}
		if session == nil {
			session = &config.SessionState{}
		}

		field, value := args[0], args[1]
		switch field {
		case "model":
			session.Model = value
		case "ticket":
			session.Ticket = value
		case "prompt":
			session.Prompt = value
		case "origin":
			session.Origin = value
		default:
			return fmt.Errorf("unknown session field: %s (valid: model, ticket, prompt, origin)", field)
		}

		if err := config.SaveSession(repoPath, session); err != nil {
			return fmt.Errorf("save session: %w", err)
		}

		fmt.Printf("  session.%s = %s\n", field, value)
		return nil
	},
}

var sessionEndCmd = &cobra.Command{
	Use:   "end",
	Short: "Clear the active session",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := config.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}

		if err := config.ClearSession(repoPath); err != nil {
			return fmt.Errorf("clear session: %w", err)
		}

		fmt.Println("  gitwhy: session ended")
		return nil
	},
}

func printSession(session *config.SessionState, age string) {
	if session.Model != "" {
		fmt.Printf("  model:   %s\n", session.Model)
	}
	if session.Ticket != "" {
		fmt.Printf("  ticket:  %s\n", session.Ticket)
	}
	if session.Prompt != "" {
		fmt.Printf("  prompt:  %s\n", session.Prompt)
	}
	if session.Origin != "" {
		fmt.Printf("  origin:  %s\n", session.Origin)
	}
	if age != "" {
		fmt.Printf("  started: %s ago\n", age)
	}
}

func sessionFileAge(repoPath string) (string, error) {
	fi, err := os.Stat(config.SessionPath(repoPath))
	if err != nil {
		return "", err
	}
	return time.Since(fi.ModTime()).Round(time.Minute).String(), nil
}

func promptLine(label string) string {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptSessionInteractive(repoPath string) error {
	fmt.Println("  gitwhy: starting an interactive session (leave blank to skip a field)")

	model := promptLine("  model:  ")
	ticket := promptLine("  ticket: ")
	prompt := promptLine("  prompt: ")
	origin := promptLine("  origin: ")

	session := &config.SessionState{
		Model:  model,
		Ticket: ticket,
		Prompt: prompt,
		Origin: origin,
	}

	if err := config.SaveSession(repoPath, session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	fmt.Println("  gitwhy: session started")
	printSession(session, "")
	return nil
}

func init() {
	sessionStartCmd.Flags().StringVar(&sessionStartFlags.model, "model", "", "Model active for this session (e.g. claude-sonnet-4-6)")
	sessionStartCmd.Flags().StringVar(&sessionStartFlags.ticket, "ticket", "", "Ticket or issue reference")
	sessionStartCmd.Flags().StringVar(&sessionStartFlags.prompt, "prompt", "", "Prompt driving this session")
	sessionStartCmd.Flags().StringVar(&sessionStartFlags.origin, "origin", "", "Origin type (human, spec, prompt, template, upstream)")

	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionStatusCmd)
	sessionCmd.AddCommand(sessionSetCmd)
	sessionCmd.AddCommand(sessionEndCmd)
}
