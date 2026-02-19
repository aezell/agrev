// Package trace handles ingestion and parsing of agent conversation traces.
package trace

import "time"

// StepType categorizes a step in the agent's workflow.
type StepType int

const (
	StepPlan StepType = iota
	StepReasoning
	StepFileRead
	StepFileWrite
	StepFileEdit
	StepBash
	StepToolResult
	StepUserMessage
)

func (s StepType) String() string {
	switch s {
	case StepPlan:
		return "plan"
	case StepReasoning:
		return "reasoning"
	case StepFileRead:
		return "read"
	case StepFileWrite:
		return "write"
	case StepFileEdit:
		return "edit"
	case StepBash:
		return "bash"
	case StepToolResult:
		return "result"
	case StepUserMessage:
		return "user"
	default:
		return "unknown"
	}
}

// Step is a single action in the agent's timeline.
type Step struct {
	Type      StepType
	Timestamp time.Time
	Summary   string // short description of this step
	Detail    string // full content (may be long)

	// File-related fields (for read/write/edit steps)
	FilePath string

	// Bash-related fields
	Command  string
	ExitCode int

	// For correlation with diff hunks
	LineStart int // 0 if unknown
	LineEnd   int // 0 if unknown
}

// Trace is the parsed representation of an agent conversation.
type Trace struct {
	Source    string    // "claude-code", "aider", "generic"
	SessionID string
	StartTime time.Time
	EndTime   time.Time
	Steps     []Step

	// Derived data
	Summary      string   // generated PR-style summary
	FilesChanged []string // files touched by the agent
}

// FileSteps returns all steps that touch the given file path.
func (t *Trace) FileSteps(path string) []Step {
	var result []Step
	for _, s := range t.Steps {
		if s.FilePath == path {
			result = append(result, s)
		}
	}
	return result
}

// StepsOfType returns all steps of the given type.
func (t *Trace) StepsOfType(st StepType) []Step {
	var result []Step
	for _, s := range t.Steps {
		if s.Type == st {
			result = append(result, s)
		}
	}
	return result
}
