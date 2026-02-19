package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DetectAndLoad finds trace files automatically and loads the most recent one.
// It searches in order of priority: explicit path, Claude Code traces, Aider history.
func DetectAndLoad(repoDir string) (*Trace, error) {
	path, format := Detect(repoDir)
	if path == "" {
		return nil, nil // no trace found is not an error
	}
	return Load(path, format)
}

// Load parses a trace file with the given format hint.
func Load(path string, format string) (*Trace, error) {
	switch format {
	case "claude-code":
		return ParseClaudeCode(path)
	case "aider":
		return ParseAider(path)
	case "generic":
		return ParseGenericJSONL(path)
	default:
		// Try to detect from content
		return autoLoad(path)
	}
}

// Detect searches for trace files and returns the path and format of the best match.
func Detect(repoDir string) (path, format string) {
	// 1. Claude Code traces in ~/.claude/projects/
	if p := detectClaudeCode(repoDir); p != "" {
		return p, "claude-code"
	}

	// 2. Aider history in the repo
	if p := detectAider(repoDir); p != "" {
		return p, "aider"
	}

	// 3. Generic .agrev-trace.jsonl in the repo
	generic := filepath.Join(repoDir, ".agrev-trace.jsonl")
	if _, err := os.Stat(generic); err == nil {
		return generic, "generic"
	}

	return "", ""
}

func detectClaudeCode(repoDir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Claude Code stores traces in ~/.claude/projects/<encoded-path>/
	// The path encoding replaces / with -
	claudeProjectsDir := filepath.Join(home, ".claude", "projects")

	entries, err := os.ReadDir(claudeProjectsDir)
	if err != nil {
		return ""
	}

	// Find project dirs that might match our repo
	absRepo, err := filepath.Abs(repoDir)
	if err != nil {
		return ""
	}

	// Claude Code encodes the path by replacing / with -
	encodedPath := strings.ReplaceAll(absRepo, "/", "-")

	var matchingDir string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Check if the directory name matches or contains our repo path
		if e.Name() == encodedPath || strings.HasPrefix(encodedPath, e.Name()) || strings.HasPrefix(e.Name(), encodedPath) {
			matchingDir = filepath.Join(claudeProjectsDir, e.Name())
			break
		}
	}

	if matchingDir == "" {
		// Also try parent directories (repo might be in a subdirectory)
		for dir := absRepo; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			enc := strings.ReplaceAll(dir, "/", "-")
			for _, e := range entries {
				if e.IsDir() && e.Name() == enc {
					matchingDir = filepath.Join(claudeProjectsDir, e.Name())
					break
				}
			}
			if matchingDir != "" {
				break
			}
		}
	}

	if matchingDir == "" {
		return ""
	}

	// Find the most recent JSONL file
	return mostRecentJSONL(matchingDir)
}

func mostRecentJSONL(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	type fileInfo struct {
		path    string
		modTime int64
	}

	var jsonlFiles []fileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		jsonlFiles = append(jsonlFiles, fileInfo{
			path:    filepath.Join(dir, e.Name()),
			modTime: info.ModTime().Unix(),
		})
	}

	if len(jsonlFiles) == 0 {
		return ""
	}

	// Sort by modification time, most recent first
	sort.Slice(jsonlFiles, func(i, j int) bool {
		return jsonlFiles[i].modTime > jsonlFiles[j].modTime
	})

	return jsonlFiles[0].path
}

func detectAider(repoDir string) string {
	historyFile := filepath.Join(repoDir, ".aider.chat.history.md")
	if _, err := os.Stat(historyFile); err == nil {
		return historyFile
	}
	return ""
}

func autoLoad(path string) (*Trace, error) {
	// Try Claude Code format first (JSONL with "type" and "message" fields)
	if strings.HasSuffix(path, ".jsonl") {
		t, err := ParseClaudeCode(path)
		if err == nil && len(t.Steps) > 0 {
			return t, nil
		}

		// Fall back to generic JSONL
		t, err = ParseGenericJSONL(path)
		if err == nil && len(t.Steps) > 0 {
			return t, nil
		}
	}

	// Try Aider
	if strings.HasSuffix(path, ".md") {
		return ParseAider(path)
	}

	return nil, fmt.Errorf("unable to determine trace format for %s", path)
}
