package analysis

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/sprite-ai/agrev/internal/diff"
	"github.com/sprite-ai/agrev/internal/model"
)

// Anti-pattern regexes.
var (
	// Broad exception handling
	broadExceptPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)except\s*:`),                           // Python: bare except
		regexp.MustCompile(`(?i)except\s+Exception\s*:`),               // Python: catch-all
		regexp.MustCompile(`(?i)catch\s*\(\s*(Exception|Error|e)\s*\)`), // Java/C#
		regexp.MustCompile(`(?i)catch\s*\(\s*err(?:or)?\s*\)\s*\{`),    // Go-like (but Go doesn't have try/catch)
		regexp.MustCompile(`(?i)catch\s*\{`),                           // Scala/Kotlin bare catch
		regexp.MustCompile(`(?i)rescue\s*$`),                           // Ruby: bare rescue
		regexp.MustCompile(`(?i)rescue\s+StandardError`),               // Ruby: catch-all
		regexp.MustCompile(`\.catch\(\s*(?:_|err|\(\s*\))\s*=>`),       // JS: .catch((_) => or .catch(() =>
	}

	// Commented-out code patterns (lines that look like disabled code, not natural comments)
	commentedCodePatterns = []*regexp.Regexp{
		regexp.MustCompile(`^\s*(?://|#)\s*(?:func |def |class |if |for |while |return |import |from |const |let |var |pub fn )`),
		regexp.MustCompile(`^\s*(?://|#)\s*\w+\s*[({=]`),
		regexp.MustCompile(`^\s*{?/\*.*\b(?:func|def|class|return)\b.*\*/}?`),
	}

	// TODO/FIXME/HACK left behind by agent
	todoPattern = regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|XXX|TEMP|TEMPORARY)\b`)
)

// AntiPatternPass detects common agent anti-patterns.
func AntiPatternPass(ds *diff.DiffSet, repoDir string) []Finding {
	var findings []Finding

	for _, f := range ds.Files {
		name := f.Name()
		findings = append(findings, checkBroadExceptions(f, name)...)
		findings = append(findings, checkCommentedCode(f, name)...)
		findings = append(findings, checkTodos(f, name)...)
	}

	// Check for near-duplicate code blocks across files
	findings = append(findings, checkDuplication(ds)...)

	return findings
}

func checkBroadExceptions(f *diff.File, name string) []Finding {
	var findings []Finding

	for _, frag := range f.Fragments {
		lineNum := int(frag.NewPosition)
		for _, line := range frag.Lines {
			if line.Op == gitdiff.OpAdd {
				for _, pat := range broadExceptPatterns {
					if pat.MatchString(line.Line) {
						findings = append(findings, Finding{
							Pass:     "anti_patterns",
							File:     name,
							Line:     lineNum,
							Message:  fmt.Sprintf("Broad exception handling: %s", strings.TrimSpace(line.Line)),
							Severity: model.SeverityWarning,
							Risk:     model.RiskMedium,
						})
						break
					}
				}
			}
			if line.Op == gitdiff.OpAdd || line.Op == gitdiff.OpContext {
				lineNum++
			}
		}
	}

	return findings
}

func checkCommentedCode(f *diff.File, name string) []Finding {
	var findings []Finding

	for _, frag := range f.Fragments {
		lineNum := int(frag.NewPosition)
		for _, line := range frag.Lines {
			if line.Op == gitdiff.OpAdd {
				for _, pat := range commentedCodePatterns {
					if pat.MatchString(line.Line) {
						findings = append(findings, Finding{
							Pass:     "anti_patterns",
							File:     name,
							Line:     lineNum,
							Message:  fmt.Sprintf("Commented-out code: %s", strings.TrimSpace(line.Line)),
							Severity: model.SeverityWarning,
							Risk:     model.RiskLow,
						})
						break
					}
				}
			}
			if line.Op == gitdiff.OpAdd || line.Op == gitdiff.OpContext {
				lineNum++
			}
		}
	}

	return findings
}

func checkTodos(f *diff.File, name string) []Finding {
	var findings []Finding

	for _, frag := range f.Fragments {
		lineNum := int(frag.NewPosition)
		for _, line := range frag.Lines {
			if line.Op == gitdiff.OpAdd {
				if matches := todoPattern.FindString(line.Line); matches != "" {
					findings = append(findings, Finding{
						Pass:     "anti_patterns",
						File:     name,
						Line:     lineNum,
						Message:  fmt.Sprintf("Agent left %s marker: %s", matches, strings.TrimSpace(line.Line)),
						Severity: model.SeverityWarning,
						Risk:     model.RiskLow,
					})
				}
			}
			if line.Op == gitdiff.OpAdd || line.Op == gitdiff.OpContext {
				lineNum++
			}
		}
	}

	return findings
}

// checkDuplication looks for near-duplicate code blocks introduced by the diff.
// It uses a sliding window of N lines over added content and looks for repeated hashes.
func checkDuplication(ds *diff.DiffSet) []Finding {
	const windowSize = 4

	type blockLoc struct {
		file string
		line int
	}

	blocks := make(map[string][]blockLoc) // hash -> locations

	for _, f := range ds.Files {
		name := f.Name()

		// Collect all added lines with their line numbers
		type addedLine struct {
			text    string
			lineNum int
		}
		var added []addedLine

		for _, frag := range f.Fragments {
			lineNum := int(frag.NewPosition)
			for _, line := range frag.Lines {
				if line.Op == gitdiff.OpAdd {
					trimmed := strings.TrimSpace(line.Line)
					// Skip trivial lines
					if trimmed != "" && trimmed != "{" && trimmed != "}" && trimmed != ")" && trimmed != "(" {
						added = append(added, addedLine{text: trimmed, lineNum: lineNum})
					}
				}
				if line.Op == gitdiff.OpAdd || line.Op == gitdiff.OpContext {
					lineNum++
				}
			}
		}

		// Slide a window over the added lines
		for i := 0; i+windowSize <= len(added); i++ {
			var window []string
			for j := 0; j < windowSize; j++ {
				window = append(window, added[i+j].text)
			}
			h := hashBlock(window)
			blocks[h] = append(blocks[h], blockLoc{file: name, line: added[i].lineNum})
		}
	}

	var findings []Finding
	for _, locs := range blocks {
		if len(locs) < 2 {
			continue
		}
		// Report on the second (and subsequent) occurrences
		for _, loc := range locs[1:] {
			findings = append(findings, Finding{
				Pass:     "anti_patterns",
				File:     loc.file,
				Line:     loc.line,
				Message:  fmt.Sprintf("Near-duplicate code block (also at %s:%d)", locs[0].file, locs[0].line),
				Severity: model.SeverityWarning,
				Risk:     model.RiskMedium,
			})
		}
	}

	return findings
}

func hashBlock(lines []string) string {
	h := sha256.New()
	for _, l := range lines {
		h.Write([]byte(l))
		h.Write([]byte{'\n'})
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
