package analysis

import (
	"fmt"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/sprite-ai/agrev/internal/diff"
	"github.com/sprite-ai/agrev/internal/model"
)

// Dependency/lockfile patterns.
var depFiles = map[string]string{
	"go.mod":             "go",
	"go.sum":             "go",
	"package.json":       "npm",
	"package-lock.json":  "npm",
	"yarn.lock":          "npm",
	"pnpm-lock.yaml":     "npm",
	"Cargo.toml":         "cargo",
	"Cargo.lock":         "cargo",
	"requirements.txt":   "pip",
	"Pipfile":            "pip",
	"Pipfile.lock":       "pip",
	"pyproject.toml":     "pip",
	"poetry.lock":        "pip",
	"Gemfile":            "gem",
	"Gemfile.lock":       "gem",
	"mix.exs":            "hex",
	"mix.lock":           "hex",
}

// NewDependencyPass detects new dependencies added in the diff.
func NewDependencyPass(ds *diff.DiffSet, repoDir string) []Finding {
	var findings []Finding

	for _, f := range ds.Files {
		name := f.Name()
		eco, isDep := depFiles[baseName(name)]
		if !isDep {
			continue
		}

		newDeps := extractNewDeps(f, eco)
		for _, dep := range newDeps {
			findings = append(findings, Finding{
				Pass:     "deps",
				File:     name,
				Line:     dep.line,
				Message:  fmt.Sprintf("New %s dependency: %s", eco, dep.name),
				Severity: model.SeverityWarning,
				Risk:     model.RiskMedium,
			})
		}
	}

	return findings
}

type depInfo struct {
	name string
	line int
}

func extractNewDeps(f *diff.File, ecosystem string) []depInfo {
	var deps []depInfo

	for _, frag := range f.Fragments {
		lineNum := int(frag.NewPosition)
		for _, line := range frag.Lines {
			if line.Op == gitdiff.OpAdd {
				text := strings.TrimSpace(line.Line)
				if dep := parseDepLine(text, ecosystem); dep != "" {
					deps = append(deps, depInfo{name: dep, line: lineNum})
				}
			}
			if line.Op == gitdiff.OpAdd || line.Op == gitdiff.OpContext {
				lineNum++
			}
		}
	}

	return deps
}

func parseDepLine(line, eco string) string {
	switch eco {
	case "go":
		// go.mod: require github.com/foo/bar v1.2.3
		// go.mod: \tgithub.com/foo/bar v1.2.3
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[1]
			}
		}
		// Inside require block
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.Contains(parts[0], "/") && !strings.HasPrefix(parts[0], "//") {
			return parts[0]
		}

	case "npm":
		// package.json: "dep-name": "^1.0.0"
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, ",")
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			name := strings.Trim(parts[0], `" `)
			if name != "" && !strings.HasPrefix(name, "@types/") &&
				name != "dependencies" && name != "devDependencies" &&
				name != "peerDependencies" && name != "name" && name != "version" {
				return name
			}
		}

	case "cargo":
		// Cargo.toml: dep-name = "1.0"  or  dep-name = { version = "1.0" }
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "[") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "=", 2)
			name := strings.TrimSpace(parts[0])
			if name != "" && name != "name" && name != "version" && name != "edition" &&
				name != "authors" && name != "description" && name != "license" &&
				!strings.Contains(name, ".") {
				return name
			}
		}

	case "pip":
		// requirements.txt: package==1.0.0 or package>=1.0
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			return ""
		}
		// Split on version specifiers
		for _, sep := range []string{"==", ">=", "<=", "!=", "~=", ">"} {
			if idx := strings.Index(line, sep); idx > 0 {
				return strings.TrimSpace(line[:idx])
			}
		}
		if !strings.Contains(line, " ") {
			return line
		}

	case "gem":
		// Gemfile: gem 'name', '~> 1.0'
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gem ") {
			parts := strings.SplitN(line, ",", 2)
			name := strings.TrimPrefix(parts[0], "gem ")
			name = strings.Trim(name, `'" `)
			return name
		}

	case "hex":
		// mix.exs: {:dep_name, "~> 1.0"}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "{:") {
			end := strings.Index(line, ",")
			if end > 2 {
				return strings.TrimPrefix(line[:end], "{:")
			}
		}
	}

	return ""
}

func baseName(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx >= 0 {
		return path[idx+1:]
	}
	return path
}
