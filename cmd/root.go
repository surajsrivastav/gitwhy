package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/surajsrivastav/gitwhy/pkg/passthrough"
)

var (
	commitVersion string
	buildDate     string
)

var rootCmd = &cobra.Command{
	Use:   "ghw",
	Short: "Intent-aware Git CLI — wraps gh with provenance tracking",
	Long: `ghw is a CLI tool that wraps and extends the GitHub CLI (gh)
with a provenance and intent layer built for the age of AI-generated code.

Every commit, PR, and code review carries hidden context: why was this
change made, what prompted it, was it human-authored or AI-generated,
and does it still match the original intent?

ghw makes that context first-class.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       commitVersion,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func SetVersion(commit, date string) {
	commitVersion = commit
	buildDate = date
	if commitVersion == "" {
		commitVersion = "dev"
	}
	rootCmd.Version = commitVersion
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	RunE: func(cmd *cobra.Command, args []string) error {
		version := commitVersion
		if version == "" {
			version = "dev"
		}
		fmt.Printf("ghw %s", version)
		if buildDate != "" {
			fmt.Printf(" (%s)", buildDate)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(whyCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() {
	args := os.Args[1:]
	if len(args) > 0 && isPassthrough(args) {
		if !passthrough.IsGHAvailable() {
			fmt.Fprintf(os.Stderr, "ghw: unknown command %q — gh not found in PATH\n", args[0])
			os.Exit(1)
		}
		if err := passthrough.Execute(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func isPassthrough(args []string) bool {
	if len(args) == 0 {
		return false
	}

	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" ||
		args[0] == "--version" || args[0] == "-v" {
		return false
	}

	knownCommands := map[string]bool{
		"commit": true, "init": true, "config": true,
		"why": true, "log": true, "diff": true, "audit": true,
		"version": true,
	}

	if knownCommands[args[0]] {
		return false
	}

	if args[0] == "pr" && len(args) > 1 {
		knownPRSubcommands := map[string]bool{
			"create": true, "view": true, "list": true,
		}
		if knownPRSubcommands[args[1]] {
			return false
		}
	}

	return true
}
