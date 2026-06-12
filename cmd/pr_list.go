package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anomalyco/gitwhy/pkg/passthrough"
)

var prListFlags struct {
	attributed string
}

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests with optional attribution filter",
	Long:  `List GitHub pull requests, optionally filtered by attribution type.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ghArgs := []string{"pr", "list"}

		if prListFlags.attributed != "" {
			ghArgs = append(ghArgs, "--json", "number,title,author,body")
		}

		ghArgs = append(ghArgs, args...)

		output, err := passthrough.ExecuteWithOutput(ghArgs)
		if err != nil {
			fmt.Fprint(os.Stderr, output)
			return err
		}

		fmt.Print(output)
		return nil
	},
}

func init() {
	prListCmd.Flags().StringVar(&prListFlags.attributed, "attributed", "", "Filter by attribution (human, ai, agent:<name>)")
}
