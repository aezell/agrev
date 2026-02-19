package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/aezell/agrev/internal/diff"
	"github.com/aezell/agrev/internal/model"
)

// BlastRadiusPass estimates how many callers reference changed functions.
func BlastRadiusPass(ds *diff.DiffSet, repoDir string) []Finding {
	if repoDir == "" {
		return nil
	}

	var findings []Finding

	for _, f := range ds.Files {
		name := f.Name()

		// Extract function names that appear in changed lines (both added and deleted)
		changedFuncs := extractChangedFunctions(f)

		for _, fn := range changedFuncs {
			count := countReferences(repoDir, name, fn)
			if count > 15 {
				findings = append(findings, Finding{
					Pass:     "blast_radius",
					File:     name,
					Line:     0,
					Message:  fmt.Sprintf("Function %q has %d references (high blast radius)", fn, count),
					Severity: model.SeverityWarning,
					Risk:     model.RiskHigh,
				})
			} else if count > 5 {
				findings = append(findings, Finding{
					Pass:     "blast_radius",
					File:     name,
					Line:     0,
					Message:  fmt.Sprintf("Function %q has %d references across the codebase", fn, count),
					Severity: model.SeverityWarning,
					Risk:     model.RiskMedium,
				})
			}
		}
	}

	return findings
}

func extractChangedFunctions(f *diff.File) []string {
	seen := make(map[string]bool)
	var funcs []string

	for _, frag := range f.Fragments {
		for _, line := range frag.Lines {
			if line.Op == gitdiff.OpContext {
				continue
			}
			for _, pat := range funcDefPatterns {
				if matches := pat.FindStringSubmatch(line.Line); len(matches) > 1 {
					name := matches[1]
					if !seen[name] && len(name) > 2 { // skip very short names
						seen[name] = true
						funcs = append(funcs, name)
					}
				}
			}
		}
	}

	return funcs
}

func countReferences(repoDir, sourceFile, funcName string) int {
	if len(funcName) < 3 {
		return 0
	}

	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(funcName) + `\b`)
	count := 0

	// Walk the repo directory looking for source files
	_ = filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Skip hidden dirs, vendor, node_modules, etc.
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules" || base == "dist" || base == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only check source files
		if !isSourceFile(path) {
			return nil
		}

		// Skip the source file itself
		rel, _ := filepath.Rel(repoDir, path)
		if rel == sourceFile {
			return nil
		}

		// Read and search
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		matches := pattern.FindAll(content, -1)
		count += len(matches)

		// Early exit if we have enough
		if count > 20 {
			return filepath.SkipAll
		}

		return nil
	})

	return count
}

func isSourceFile(path string) bool {
	ext := filepath.Ext(path)
	switch ext {
	case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".rb", ".rs",
		".java", ".kt", ".scala", ".c", ".cpp", ".h", ".hpp",
		".cs", ".ex", ".exs", ".erl", ".hs", ".ml", ".swift":
		return true
	}
	return false
}
