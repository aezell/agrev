package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/aezell/agrev/internal/trace"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generate a PR description from agent trace",
	Long: `Parse the agent conversation trace and generate a summary suitable
for use as a pull request description.`,
	RunE: runSummary,
}

func init() {
	summaryCmd.Flags().StringP("trace", "t", "", "path to agent trace file")
	summaryCmd.Flags().StringP("format", "f", "markdown", "output format: markdown, text")
}

func runSummary(cmd *cobra.Command, args []string) error {
	tracePath, _ := cmd.Flags().GetString("trace")

	var t *trace.Trace
	var err error

	if tracePath != "" {
		t, err = trace.Load(tracePath, "")
		if err != nil {
			return fmt.Errorf("loading trace: %w", err)
		}
	} else {
		// Auto-detect
		repoDir, repoErr := gitRepoRoot()
		if repoErr != nil {
			return fmt.Errorf("not in a git repository; use --trace to specify trace file: %w", repoErr)
		}
		t, err = trace.DetectAndLoad(repoDir)
		if err != nil {
			return fmt.Errorf("detecting trace: %w", err)
		}
	}

	if t == nil {
		fmt.Fprintln(os.Stderr, "No agent trace found. Use --trace to specify a trace file.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Source: %s (%d steps, %d files)\n\n", t.Source, len(t.Steps), len(t.FilesChanged))
	fmt.Print(t.Summary)

	return nil
}
