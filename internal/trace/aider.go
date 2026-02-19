package trace

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseAider parses an Aider chat history markdown file.
// Aider writes to .aider.chat.history.md with a format like:
//
//	# aider chat started at 2024-01-15 10:30:00
//
//	#### /ask what does this code do?
//
//	The code implements...
//
//	#### make the function async
//
//	I'll modify the function to be async...
//	```python
//	async def foo():
//	    ...
//	```
func ParseAider(path string) (*Trace, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening aider trace: %w", err)
	}
	defer f.Close()

	trace := &Trace{
		Source: "aider",
	}

	filesSet := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var currentBlock strings.Builder
	var currentType StepType
	inBlock := false

	flushBlock := func() {
		if !inBlock || currentBlock.Len() == 0 {
			return
		}
		text := strings.TrimSpace(currentBlock.String())
		if text == "" {
			return
		}

		step := Step{
			Type:    currentType,
			Summary: truncateStr(text, 100),
			Detail:  text,
		}

		// Look for file paths in edit blocks
		if currentType == StepReasoning {
			// Check if this block mentions file edits
			for _, line := range strings.Split(text, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "```") && len(line) > 3 {
					// Code block start - skip
					continue
				}
				// Look for file path patterns in Aider output
				if isFilePath(line) {
					filesSet[line] = true
					step.FilePath = line
					step.Type = StepFileEdit
					step.Summary = fmt.Sprintf("Edit %s", shortPath(line))
				}
			}
		}

		trace.Steps = append(trace.Steps, step)
		currentBlock.Reset()
		inBlock = false
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Chat session header
		if strings.HasPrefix(line, "# aider chat started") {
			continue
		}

		// User command (starts with ####)
		if strings.HasPrefix(line, "#### ") {
			flushBlock()
			cmd := strings.TrimPrefix(line, "#### ")
			trace.Steps = append(trace.Steps, Step{
				Type:    StepUserMessage,
				Summary: truncateStr(cmd, 100),
				Detail:  cmd,
			})
			currentType = StepReasoning
			inBlock = true
			continue
		}

		if inBlock {
			currentBlock.WriteString(line)
			currentBlock.WriteByte('\n')
		}
	}

	flushBlock()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning aider trace: %w", err)
	}

	for f := range filesSet {
		trace.FilesChanged = append(trace.FilesChanged, f)
	}

	return trace, nil
}

// isFilePath is a simple heuristic to detect file paths.
func isFilePath(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \t{}()[]") {
		return false
	}
	// Must contain a dot (extension) or slash
	return strings.Contains(s, ".") && !strings.HasPrefix(s, "http")
}
