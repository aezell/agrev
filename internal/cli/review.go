package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sprite-ai/agrev/internal/analysis"
	"github.com/sprite-ai/agrev/internal/diff"
	"github.com/sprite-ai/agrev/internal/trace"
	"github.com/sprite-ai/agrev/internal/tui"
)

var reviewCmd = &cobra.Command{
	Use:   "review [commit-range]",
	Short: "Open an interactive review session",
	Long: `Open an interactive TUI for reviewing changes. By default, reviews
HEAD against its parent commit. Optionally specify a commit range.

You can also pipe a diff into agrev:
  git diff main...HEAD | agrev review -`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReview,
}

func init() {
	reviewCmd.Flags().StringP("trace", "t", "", "path to agent trace file")
	reviewCmd.Flags().Bool("no-trace", false, "skip trace auto-detection")
	reviewCmd.Flags().IntP("context", "C", 3, "lines of context around changes")
	reviewCmd.Flags().Bool("stat", false, "print diff stats and exit (non-interactive)")
}

func runReview(cmd *cobra.Command, args []string) error {
	contextLines, _ := cmd.Flags().GetInt("context")

	raw, err := getDiff(args, contextLines)
	if err != nil {
		return err
	}

	if strings.TrimSpace(raw) == "" {
		fmt.Println("No changes to review.")
		return nil
	}

	ds, err := diff.Parse(raw)
	if err != nil {
		return fmt.Errorf("parsing diff: %w", err)
	}

	if len(ds.Files) == 0 {
		fmt.Println("No changes to review.")
		return nil
	}

	stat, _ := cmd.Flags().GetBool("stat")
	if stat {
		return printStat(ds)
	}

	// Load trace
	t, traceSource := loadTrace(cmd)
	if t != nil {
		fmt.Fprintf(os.Stderr, "Loaded %s trace: %d steps, %d files\n",
			traceSource, len(t.Steps), len(t.FilesChanged))
	}

	// Run analysis
	repoDir, _ := gitRepoRoot()
	ar := analysis.Run(ds, repoDir, nil)
	if len(ar.Findings) > 0 {
		fmt.Fprintf(os.Stderr, "Analysis: %s\n", ar.Summary())
	}

	return tui.Run(ds, t, ar)
}

func loadTrace(cmd *cobra.Command) (*trace.Trace, string) {
	noTrace, _ := cmd.Flags().GetBool("no-trace")
	if noTrace {
		return nil, ""
	}

	tracePath, _ := cmd.Flags().GetString("trace")
	if tracePath != "" {
		t, err := trace.Load(tracePath, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load trace %s: %v\n", tracePath, err)
			return nil, ""
		}
		return t, t.Source
	}

	// Auto-detect
	repoDir, err := gitRepoRoot()
	if err != nil {
		return nil, ""
	}

	t, err := trace.DetectAndLoad(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: trace detection failed: %v\n", err)
		return nil, ""
	}

	if t != nil {
		return t, t.Source
	}

	return nil, ""
}

func getDiff(args []string, contextLines int) (string, error) {
	// Read from stdin if "-" is passed
	if len(args) == 1 && args[0] == "-" {
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	// Find repo root
	repoDir, err := gitRepoRoot()
	if err != nil {
		return "", fmt.Errorf("not in a git repository (or git not installed): %w", err)
	}

	if len(args) == 1 {
		// Explicit commit range
		return diff.GitDiffRange(repoDir, args[0], contextLines)
	}

	// Default: HEAD vs parent
	return diff.GitDiffHead(repoDir, contextLines)
}

func printStat(ds *diff.DiffSet) error {
	files, added, deleted := ds.Stats()
	fmt.Printf("%d file(s) changed, %d insertions(+), %d deletions(-)\n\n", files, added, deleted)
	for _, f := range ds.Files {
		status := "M"
		if f.IsNew {
			status = "A"
		} else if f.IsDeleted {
			status = "D"
		} else if f.IsRenamed {
			status = "R"
		}
		fmt.Printf("  %s %-50s +%-4d -%d\n", status, f.Name(), f.AddedLines, f.DeletedLines)
	}
	return nil
}

func gitRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
