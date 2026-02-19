package trace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Generic JSONL trace format:
//   {"type": "plan", "content": "I'll add rate limiting..."}
//   {"type": "file_read", "path": "api/middleware.go"}
//   {"type": "file_edit", "path": "api/middleware.go", "description": "Add RateLimiter struct"}
//   {"type": "file_write", "path": "api/middleware.go", "description": "Create new file"}
//   {"type": "bash", "command": "go test ./...", "exit_code": 0}
//   {"type": "reasoning", "content": "Tests pass. Now I need to..."}

type genericEntry struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Command     string `json:"command"`
	ExitCode    int    `json:"exit_code"`
	Timestamp   string `json:"timestamp"`
}

// ParseGenericJSONL parses a generic JSONL trace file.
func ParseGenericJSONL(path string) (*Trace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening trace: %w", err)
	}
	defer f.Close()

	return parseGenericReader(f)
}

func parseGenericReader(r io.Reader) (*Trace, error) {
	trace := &Trace{
		Source: "generic",
	}

	filesSet := make(map[string]bool)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry genericEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		ts := parseTimestamp(entry.Timestamp)

		if trace.StartTime.IsZero() && !ts.IsZero() {
			trace.StartTime = ts
		}
		if !ts.IsZero() {
			trace.EndTime = ts
		}

		switch entry.Type {
		case "plan":
			trace.Steps = append(trace.Steps, Step{
				Type:      StepPlan,
				Timestamp: ts,
				Summary:   truncateStr(entry.Content, 100),
				Detail:    entry.Content,
			})

		case "reasoning":
			trace.Steps = append(trace.Steps, Step{
				Type:      StepReasoning,
				Timestamp: ts,
				Summary:   truncateStr(entry.Content, 100),
				Detail:    entry.Content,
			})

		case "file_read":
			trace.Steps = append(trace.Steps, Step{
				Type:     StepFileRead,
				Timestamp: ts,
				FilePath: entry.Path,
				Summary:  fmt.Sprintf("Read %s", shortPath(entry.Path)),
			})

		case "file_write":
			filesSet[entry.Path] = true
			summary := entry.Description
			if summary == "" {
				summary = fmt.Sprintf("Write %s", shortPath(entry.Path))
			}
			trace.Steps = append(trace.Steps, Step{
				Type:      StepFileWrite,
				Timestamp: ts,
				FilePath:  entry.Path,
				Summary:   summary,
				Detail:    entry.Content,
			})

		case "file_edit":
			filesSet[entry.Path] = true
			summary := entry.Description
			if summary == "" {
				summary = fmt.Sprintf("Edit %s", shortPath(entry.Path))
			}
			trace.Steps = append(trace.Steps, Step{
				Type:      StepFileEdit,
				Timestamp: ts,
				FilePath:  entry.Path,
				Summary:   summary,
				Detail:    entry.Content,
			})

		case "bash":
			trace.Steps = append(trace.Steps, Step{
				Type:      StepBash,
				Timestamp: ts,
				Command:   entry.Command,
				ExitCode:  entry.ExitCode,
				Summary:   truncateStr(entry.Command, 80),
				Detail:    entry.Command,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning trace: %w", err)
	}

	for f := range filesSet {
		trace.FilesChanged = append(trace.FilesChanged, f)
	}

	return trace, nil
}
