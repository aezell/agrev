// Package analysis implements static analysis passes over diffs.
package analysis

import (
	"fmt"
	"strings"

	"github.com/aezell/agrev/internal/diff"
	"github.com/aezell/agrev/internal/model"
)

// Finding represents a single analysis finding attached to a file and line range.
type Finding struct {
	Pass     string // which analysis pass produced this
	File     string
	Line     int    // primary line number (in new file), 0 if file-level
	Message  string
	Severity model.Severity
	Risk     model.RiskLevel
}

func (f Finding) String() string {
	loc := f.File
	if f.Line > 0 {
		loc = fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return fmt.Sprintf("[%s] %s: %s", f.Pass, loc, f.Message)
}

// Results holds all findings from running analysis passes.
type Results struct {
	Findings []Finding
}

// ByFile returns findings grouped by file path.
func (r *Results) ByFile() map[string][]Finding {
	m := make(map[string][]Finding)
	for _, f := range r.Findings {
		m[f.File] = append(m[f.File], f)
	}
	return m
}

// ByRisk returns findings at or above the given risk level.
func (r *Results) ByRisk(minRisk model.RiskLevel) []Finding {
	var result []Finding
	for _, f := range r.Findings {
		if f.Risk >= minRisk {
			result = append(result, f)
		}
	}
	return result
}

// MaxRisk returns the highest risk level among all findings.
func (r *Results) MaxRisk() model.RiskLevel {
	max := model.RiskInfo
	for _, f := range r.Findings {
		if f.Risk > max {
			max = f.Risk
		}
	}
	return max
}

// Summary returns a one-line summary of findings.
func (r *Results) Summary() string {
	if len(r.Findings) == 0 {
		return "No issues found"
	}

	counts := make(map[model.RiskLevel]int)
	for _, f := range r.Findings {
		counts[f.Risk]++
	}

	var parts []string
	for _, level := range []model.RiskLevel{model.RiskCritical, model.RiskHigh, model.RiskMedium, model.RiskLow, model.RiskInfo} {
		if c := counts[level]; c > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", c, level))
		}
	}
	return strings.Join(parts, ", ")
}

// Pass is a function that analyzes a diff and returns findings.
type Pass func(ds *diff.DiffSet, repoDir string) []Finding

// AllPasses returns the ordered list of all analysis passes.
func AllPasses() []Pass {
	return []Pass{
		NewDependencyPass,
		SecuritySurfacePass,
		DeletedCodePass,
		SchemaChangePass,
		AntiPatternPass,
		BlastRadiusPass,
	}
}

// PassNames maps pass functions to their names (for --skip flag).
var PassNames = map[string]Pass{
	"deps":          NewDependencyPass,
	"security":      SecuritySurfacePass,
	"deleted":       DeletedCodePass,
	"schema":        SchemaChangePass,
	"anti_patterns": AntiPatternPass,
	"blast_radius":  BlastRadiusPass,
}

// Run executes all passes (or a subset) and returns the aggregated results.
func Run(ds *diff.DiffSet, repoDir string, skip []string) *Results {
	skipSet := make(map[string]bool)
	for _, s := range skip {
		skipSet[s] = true
	}

	results := &Results{}

	for name, pass := range PassNames {
		if skipSet[name] {
			continue
		}
		findings := pass(ds, repoDir)
		results.Findings = append(results.Findings, findings...)
	}

	return results
}
