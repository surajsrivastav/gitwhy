package cmd

import (
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests with provenance",
	Long:  `Work with GitHub pull requests, extended with intent and provenance metadata.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	prCmd.AddCommand(prCreateCmd)
	prCmd.AddCommand(prViewCmd)
	prCmd.AddCommand(prListCmd)
}
