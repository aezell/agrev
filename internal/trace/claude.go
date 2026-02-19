package trace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Claude Code JSONL entry structure.
type claudeEntry struct {
	Type      string          `json:"type"`
	UUID      string          `json:"uuid"`
	Timestamp string          `json:"timestamp"`
	SessionID string          `json:"sessionId"`
	Message   json.RawMessage `json:"message"`
}

type claudeMessage struct {
	Role    string               `json:"role"`
	Content json.RawMessage      `json:"content"`
}

// Content can be a string or array of content blocks.
type claudeContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`       // tool name for tool_use
	Input     json.RawMessage `json:"input"`       // tool input for tool_use
	ToolUseID string          `json:"tool_use_id"` // for tool_result
	Content   json.RawMessage `json:"content"`     // for tool_result
}

// Tool input types
type writeInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type editInput struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

type readInput struct {
	FilePath string `json:"file_path"`
}

type bashInput struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// ParseClaudeCode parses a Claude Code JSONL trace file.
func ParseClaudeCode(path string) (*Trace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening trace: %w", err)
	}
	defer f.Close()

	return parseClaudeReader(f, path)
}

func parseClaudeReader(r io.Reader, source string) (*Trace, error) {
	trace := &Trace{
		Source: "claude-code",
	}

	filesSet := make(map[string]bool)
	var reasoningParts []string

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry claudeEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}

		if trace.SessionID == "" && entry.SessionID != "" {
			trace.SessionID = entry.SessionID
		}

		ts := parseTimestamp(entry.Timestamp)

		if trace.StartTime.IsZero() && !ts.IsZero() {
			trace.StartTime = ts
		}
		if !ts.IsZero() {
			trace.EndTime = ts
		}

		switch entry.Type {
		case "user":
			step := parseUserEntry(entry, ts)
			if step != nil {
				trace.Steps = append(trace.Steps, *step)
			}

		case "assistant":
			steps := parseAssistantEntry(entry, ts, filesSet, &reasoningParts)
			trace.Steps = append(trace.Steps, steps...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning trace: %w", err)
	}

	// Collect files
	for f := range filesSet {
		trace.FilesChanged = append(trace.FilesChanged, f)
	}

	// Generate summary
	trace.Summary = generateSummary(trace, reasoningParts)

	return trace, nil
}

func parseUserEntry(entry claudeEntry, ts time.Time) *Step {
	if len(entry.Message) == 0 {
		return nil
	}

	var msg claudeMessage
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return nil
	}

	// Content might be a string
	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		if text != "" {
			return &Step{
				Type:      StepUserMessage,
				Timestamp: ts,
				Summary:   truncateStr(text, 100),
				Detail:    text,
			}
		}
	}

	return nil
}

func parseAssistantEntry(entry claudeEntry, ts time.Time, filesSet map[string]bool, reasoning *[]string) []Step {
	if len(entry.Message) == 0 {
		return nil
	}

	var msg claudeMessage
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return nil
	}

	// Content might be a string
	var textContent string
	if err := json.Unmarshal(msg.Content, &textContent); err == nil {
		if textContent != "" {
			*reasoning = append(*reasoning, textContent)
			return []Step{{
				Type:      StepReasoning,
				Timestamp: ts,
				Summary:   truncateStr(textContent, 100),
				Detail:    textContent,
			}}
		}
		return nil
	}

	// Content is an array of blocks
	var blocks []claudeContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil
	}

	var steps []Step

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				*reasoning = append(*reasoning, block.Text)
				steps = append(steps, Step{
					Type:      StepReasoning,
					Timestamp: ts,
					Summary:   truncateStr(block.Text, 100),
					Detail:    block.Text,
				})
			}

		case "tool_use":
			step := parseToolUse(block, ts, filesSet)
			if step != nil {
				steps = append(steps, *step)
			}
		}
	}

	return steps
}

func parseToolUse(block claudeContentBlock, ts time.Time, filesSet map[string]bool) *Step {
	switch block.Name {
	case "Write":
		var inp writeInput
		if err := json.Unmarshal(block.Input, &inp); err == nil {
			filesSet[inp.FilePath] = true
			return &Step{
				Type:      StepFileWrite,
				Timestamp: ts,
				FilePath:  inp.FilePath,
				Summary:   fmt.Sprintf("Write %s", shortPath(inp.FilePath)),
				Detail:    truncateStr(inp.Content, 500),
			}
		}

	case "Edit":
		var inp editInput
		if err := json.Unmarshal(block.Input, &inp); err == nil {
			filesSet[inp.FilePath] = true
			return &Step{
				Type:      StepFileEdit,
				Timestamp: ts,
				FilePath:  inp.FilePath,
				Summary:   fmt.Sprintf("Edit %s", shortPath(inp.FilePath)),
				Detail:    fmt.Sprintf("-%s\n+%s", truncateStr(inp.OldString, 200), truncateStr(inp.NewString, 200)),
			}
		}

	case "Read":
		var inp readInput
		if err := json.Unmarshal(block.Input, &inp); err == nil {
			return &Step{
				Type:      StepFileRead,
				Timestamp: ts,
				FilePath:  inp.FilePath,
				Summary:   fmt.Sprintf("Read %s", shortPath(inp.FilePath)),
			}
		}

	case "Bash":
		var inp bashInput
		if err := json.Unmarshal(block.Input, &inp); err == nil {
			summary := inp.Description
			if summary == "" {
				summary = truncateStr(inp.Command, 80)
			}
			return &Step{
				Type:      StepBash,
				Timestamp: ts,
				Command:   inp.Command,
				Summary:   summary,
				Detail:    inp.Command,
			}
		}

	default:
		// Generic tool use
		return &Step{
			Type:      StepReasoning,
			Timestamp: ts,
			Summary:   fmt.Sprintf("Tool: %s", block.Name),
		}
	}

	return nil
}

func generateSummary(t *Trace, reasoningParts []string) string {
	var b strings.Builder

	// Count actions
	writes := len(t.StepsOfType(StepFileWrite))
	edits := len(t.StepsOfType(StepFileEdit))
	commands := len(t.StepsOfType(StepBash))

	b.WriteString("## Changes\n\n")

	if len(t.FilesChanged) > 0 {
		b.WriteString(fmt.Sprintf("Modified %d file(s)", len(t.FilesChanged)))
		if writes > 0 || edits > 0 {
			b.WriteString(fmt.Sprintf(" (%d writes, %d edits)", writes, edits))
		}
		if commands > 0 {
			b.WriteString(fmt.Sprintf(", ran %d command(s)", commands))
		}
		b.WriteString("\n\n")

		b.WriteString("### Files\n")
		for _, f := range t.FilesChanged {
			b.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
		b.WriteString("\n")
	}

	// Extract key reasoning - look for the first substantial reasoning block
	if len(reasoningParts) > 0 {
		b.WriteString("### Agent Reasoning\n")
		for _, part := range reasoningParts {
			if len(part) > 50 { // skip very short fragments
				b.WriteString(truncateStr(part, 500))
				b.WriteString("\n\n")
				break
			}
		}
	}

	return b.String()
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Try common formats
	for _, format := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	} {
		t, err := time.Parse(format, s)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func truncateStr(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func shortPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}
