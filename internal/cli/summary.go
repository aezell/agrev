package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:   "summary [commit-range]",
	Short: "Generate a PR description from agent trace",
	Long: `Parse the agent conversation trace and generate a summary suitable
for use as a pull request description.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Phase 2 — generate summary from trace
		fmt.Println("agrev summary: not yet implemented")
		return nil
	},
}

func init() {
	summaryCmd.Flags().StringP("trace", "t", "", "path to agent trace file")
	summaryCmd.Flags().StringP("format", "f", "markdown", "output format: markdown, text, json")
}
