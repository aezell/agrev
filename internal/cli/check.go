package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/aezell/agrev/internal/analysis"
	"github.com/aezell/agrev/internal/diff"
	"github.com/aezell/agrev/internal/model"
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
	checkCmd.Flags().StringP("format", "f", "text", "output format: text, json, markdown, html")
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
	case "html":
		return outputHTML(ds, results)
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

func outputHTML(ds *diff.DiffSet, results *analysis.Results) error {
	nFiles, added, deleted := ds.Stats()

	fmt.Print(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>agrev Analysis Report</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; background: #282a36; color: #f8f8f2; }
  h1 { color: #bd93f9; }
  .summary { background: #343746; padding: 16px; border-radius: 8px; margin-bottom: 24px; }
  .summary span { margin-right: 24px; }
  .risk-high { color: #ff5555; font-weight: bold; }
  .risk-medium { color: #f1fa8c; }
  .risk-low { color: #8be9fd; }
  .risk-info { color: #6272a4; }
  table { width: 100%; border-collapse: collapse; }
  th { text-align: left; padding: 8px 12px; background: #44475a; color: #f8f8f2; }
  td { padding: 8px 12px; border-bottom: 1px solid #44475a; }
  tr:hover { background: #343746; }
  .pass { color: #bd93f9; }
  .file { color: #8be9fd; }
  code { background: #343746; padding: 2px 6px; border-radius: 4px; font-size: 0.9em; }
  .clean { color: #50fa7b; font-size: 1.2em; }
  footer { margin-top: 32px; color: #6272a4; font-size: 0.85em; }
</style>
</head>
<body>
<h1>agrev Analysis Report</h1>
`)

	fmt.Printf(`<div class="summary">
  <span><strong>%d</strong> file(s) changed</span>
  <span style="color:#50fa7b">+%d</span>
  <span style="color:#ff5555">-%d</span>
  <span>Risk: <span class="risk-%s">%s</span></span>
  <span>Findings: <strong>%d</strong></span>
</div>
`, nFiles, added, deleted, results.MaxRisk().String(), results.MaxRisk(), len(results.Findings))

	if len(results.Findings) == 0 {
		fmt.Println(`<p class="clean">No issues found.</p>`)
	} else {
		fmt.Println(`<table>
<thead><tr><th>Risk</th><th>Pass</th><th>File</th><th>Message</th></tr></thead>
<tbody>`)
		for _, f := range results.Findings {
			loc := f.File
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			riskClass := "risk-" + f.Risk.String()
			fmt.Printf(`<tr><td class="%s">%s</td><td class="pass">%s</td><td class="file"><code>%s</code></td><td>%s</td></tr>
`, riskClass, f.Risk, f.Pass, loc, htmlEscape(f.Message))
		}
		fmt.Println(`</tbody></table>`)
	}

	fmt.Println(`<footer>Generated by <strong>agrev</strong></footer>
</body>
</html>`)

	// Set exit code
	maxRisk := results.MaxRisk()
	if maxRisk >= model.RiskHigh {
		os.Exit(2)
	} else if maxRisk >= model.RiskLow {
		os.Exit(1)
	}

	return nil
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
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
