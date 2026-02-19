package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sprite-ai/agrev/internal/analysis"
	"github.com/sprite-ai/agrev/internal/diff"
	"github.com/sprite-ai/agrev/internal/model"
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
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().StringP("trace", "t", "", "path to agent trace file")
	checkCmd.Flags().StringP("format", "f", "text", "output format: text, json, markdown")
	checkCmd.Flags().StringSlice("skip", nil, "analysis passes to skip")
}

func runCheck(cmd *cobra.Command, args []string) error {
	contextLines := 3

	raw, err := getDiff(args, contextLines)
	if err != nil {
		return err
	}

	if strings.TrimSpace(raw) == "" {
		fmt.Println("No changes to check.")
		return nil
	}

	ds, err := diff.Parse(raw)
	if err != nil {
		return fmt.Errorf("parsing diff: %w", err)
	}

	if len(ds.Files) == 0 {
		fmt.Println("No changes to check.")
		return nil
	}

	skip, _ := cmd.Flags().GetStringSlice("skip")

	repoDir, _ := gitRepoRoot()
	results := analysis.Run(ds, repoDir, skip)

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		return outputJSON(results)
	case "markdown":
		return outputMarkdown(ds, results)
	default:
		return outputText(ds, results)
	}
}

func outputText(ds *diff.DiffSet, results *analysis.Results) error {
	nFiles, added, deleted := ds.Stats()
	fmt.Printf("%d file(s) changed, +%d -%d\n", nFiles, added, deleted)
	fmt.Printf("Analysis: %s\n\n", results.Summary())

	if len(results.Findings) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	byFile := results.ByFile()
	for file, findings := range byFile {
		fmt.Printf("  %s\n", file)
		for _, f := range findings {
			icon := riskIcon(f.Risk)
			loc := ""
			if f.Line > 0 {
				loc = fmt.Sprintf(":%d", f.Line)
			}
			fmt.Printf("    %s [%s] %s%s: %s\n", icon, f.Pass, file, loc, f.Message)
		}
		fmt.Println()
	}

	// Set exit code
	maxRisk := results.MaxRisk()
	if maxRisk >= model.RiskHigh {
		os.Exit(2)
	} else if maxRisk >= model.RiskLow {
		os.Exit(1)
	}

	return nil
}

func outputJSON(results *analysis.Results) error {
	type jsonFinding struct {
		Pass     string `json:"pass"`
		File     string `json:"file"`
		Line     int    `json:"line,omitempty"`
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Risk     string `json:"risk"`
	}

	type jsonOutput struct {
		Summary  string        `json:"summary"`
		MaxRisk  string        `json:"max_risk"`
		Total    int           `json:"total"`
		Findings []jsonFinding `json:"findings"`
	}

	out := jsonOutput{
		Summary: results.Summary(),
		MaxRisk: results.MaxRisk().String(),
		Total:   len(results.Findings),
	}

	for _, f := range results.Findings {
		out.Findings = append(out.Findings, jsonFinding{
			Pass:     f.Pass,
			File:     f.File,
			Line:     f.Line,
			Message:  f.Message,
			Severity: severityStr(f.Severity),
			Risk:     f.Risk.String(),
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputMarkdown(ds *diff.DiffSet, results *analysis.Results) error {
	nFiles, added, deleted := ds.Stats()
	fmt.Printf("## Analysis Report\n\n")
	fmt.Printf("**%d file(s)** changed, **+%d** insertions, **-%d** deletions\n\n", nFiles, added, deleted)
	fmt.Printf("**Risk:** %s | **Findings:** %d\n\n", results.MaxRisk(), len(results.Findings))

	if len(results.Findings) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	fmt.Println("| Risk | Pass | File | Message |")
	fmt.Println("|------|------|------|---------|")
	for _, f := range results.Findings {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Printf("| %s | %s | `%s` | %s |\n", f.Risk, f.Pass, loc, f.Message)
	}

	return nil
}

func riskIcon(r model.RiskLevel) string {
	switch r {
	case model.RiskCritical:
		return "!!"
	case model.RiskHigh:
		return "! "
	case model.RiskMedium:
		return "* "
	case model.RiskLow:
		return "- "
	default:
		return "  "
	}
}

func severityStr(s model.Severity) string {
	switch s {
	case model.SeverityError:
		return "error"
	case model.SeverityWarning:
		return "warning"
	default:
		return "info"
	}
}
