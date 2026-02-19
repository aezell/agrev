package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review [commit-range]",
	Short: "Open an interactive review session",
	Long: `Open an interactive TUI for reviewing changes. By default, reviews
HEAD against its parent commit. Optionally specify a commit range.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Phase 1 — launch TUI with diff viewer
		fmt.Println("agrev review: not yet implemented")
		return nil
	},
}

func init() {
	reviewCmd.Flags().StringP("trace", "t", "", "path to agent trace file")
	reviewCmd.Flags().Bool("no-trace", false, "skip trace auto-detection")
	reviewCmd.Flags().Bool("side-by-side", true, "use side-by-side diff view")
	reviewCmd.Flags().IntP("context", "C", 3, "lines of context around changes")
}
