package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check [commit-range]",
	Short: "Run analysis and output a report (non-interactive)",
	Long: `Run all analysis passes on the diff and output a structured report.
Useful for CI, pre-commit hooks, and piping into other tools.

Exit codes:
  0 — clean, no issues found
  1 — warnings found
  2 — high risk items found`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Phase 5 — run analysis passes and output report
		fmt.Println("agrev check: not yet implemented")
		return nil
	},
}

func init() {
	checkCmd.Flags().StringP("trace", "t", "", "path to agent trace file")
	checkCmd.Flags().StringP("format", "f", "text", "output format: text, json, markdown, html")
	checkCmd.Flags().StringSlice("skip", nil, "analysis passes to skip")
}
