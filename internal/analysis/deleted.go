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

// Function/method definition patterns for various languages.
var funcDefPatterns = []*regexp.Regexp{
	// Go: func Name(
	regexp.MustCompile(`^\s*func\s+(\w+)\s*\(`),
	// Go method: func (r *Type) Name(
	regexp.MustCompile(`^\s*func\s+\([^)]+\)\s+(\w+)\s*\(`),
	// Python: def name(
	regexp.MustCompile(`^\s*def\s+(\w+)\s*\(`),
	// JS/TS: function name(  or  const name = (  or  name(
	regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(`),
	regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(`),
	// Ruby: def name
	regexp.MustCompile(`^\s*def\s+(\w+)`),
	// Rust: fn name(  or  pub fn name(
	regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+(\w+)\s*[(<]`),
	// Java/C#: visibility type name(
	regexp.MustCompile(`^\s*(?:public|private|protected|static|final|abstract|override|async)\s+.*?(\w+)\s*\(`),
	// Elixir: def name(  or  defp name(
	regexp.MustCompile(`^\s*defp?\s+(\w+)\s*[(\n]`),
}

// DeletedCodePass checks for deleted functions and warns if they have test references.
func DeletedCodePass(ds *diff.DiffSet, repoDir string) []Finding {
	var findings []Finding

	for _, f := range ds.Files {
		name := f.Name()
		deletedFuncs := extractDeletedFunctions(f)

		for _, fn := range deletedFuncs {
			// Search for test references
			testRefs := findTestReferences(repoDir, name, fn.name)
			if len(testRefs) > 0 {
				findings = append(findings, Finding{
					Pass:     "deleted",
					File:     name,
					Line:     fn.line,
					Message:  fmt.Sprintf("Deleted function %q is referenced in tests: %s", fn.name, strings.Join(testRefs, ", ")),
					Severity: model.SeverityError,
					Risk:     model.RiskHigh,
				})
			} else {
				findings = append(findings, Finding{
					Pass:     "deleted",
					File:     name,
					Line:     fn.line,
					Message:  fmt.Sprintf("Deleted function: %s", fn.name),
					Severity: model.SeverityInfo,
					Risk:     model.RiskLow,
				})
			}
		}
	}

	return findings
}

type funcInfo struct {
	name string
	line int
}

func extractDeletedFunctions(f *diff.File) []funcInfo {
	var funcs []funcInfo

	for _, frag := range f.Fragments {
		lineNum := int(frag.OldPosition)
		for _, line := range frag.Lines {
			if line.Op == gitdiff.OpDelete {
				text := line.Line
				for _, pat := range funcDefPatterns {
					if matches := pat.FindStringSubmatch(text); len(matches) > 1 {
						funcs = append(funcs, funcInfo{name: matches[1], line: lineNum})
						break
					}
				}
			}
			if line.Op == gitdiff.OpDelete || line.Op == gitdiff.OpContext {
				lineNum++
			}
		}
	}

	return funcs
}

func findTestReferences(repoDir, filePath, funcName string) []string {
	if repoDir == "" {
		return nil
	}

	var refs []string
	testPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(funcName) + `\b`)

	// Determine test file patterns based on language
	dir := filepath.Dir(filepath.Join(repoDir, filePath))
	testGlobs := []string{
		filepath.Join(dir, "*_test.*"),
		filepath.Join(dir, "test_*"),
		filepath.Join(dir, "*_spec.*"),
		filepath.Join(dir, "**", "*_test.*"),
	}

	for _, pattern := range testGlobs {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			if testPattern.Match(content) {
				rel, _ := filepath.Rel(repoDir, match)
				if rel == "" {
					rel = match
				}
				refs = append(refs, rel)
			}
		}
	}

	return refs
}
