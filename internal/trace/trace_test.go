package trace

import (
	"strings"
	"testing"
)

func TestParseClaudeCode(t *testing.T) {
	jsonl := `{"type":"user","sessionId":"abc-123","timestamp":"2026-01-15T10:00:00Z","message":{"role":"user","content":"Add a login page"}}
{"type":"assistant","sessionId":"abc-123","timestamp":"2026-01-15T10:00:05Z","message":{"role":"assistant","content":[{"type":"text","text":"I'll create a login page for you."}]}}
{"type":"assistant","sessionId":"abc-123","timestamp":"2026-01-15T10:00:10Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Write","input":{"file_path":"/app/login.go","content":"package main\n\nfunc login() {}\n"}}]}}
{"type":"assistant","sessionId":"abc-123","timestamp":"2026-01-15T10:00:15Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"/app/main.go","old_string":"// routes","new_string":"router.Handle(\"/login\", loginHandler)"}}]}}
{"type":"assistant","sessionId":"abc-123","timestamp":"2026-01-15T10:00:20Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./...","description":"Run tests"}}]}}
{"type":"assistant","sessionId":"abc-123","timestamp":"2026-01-15T10:00:25Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/app/config.go"}}]}}
`

	trace, err := parseClaudeReader(strings.NewReader(jsonl), "test")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if trace.Source != "claude-code" {
		t.Errorf("expected source 'claude-code', got %q", trace.Source)
	}
	if trace.SessionID != "abc-123" {
		t.Errorf("expected session 'abc-123', got %q", trace.SessionID)
	}

	// Should have: user message, reasoning, write, edit, bash, read
	if len(trace.Steps) != 6 {
		t.Fatalf("expected 6 steps, got %d", len(trace.Steps))
	}

	// Check step types in order
	expected := []StepType{StepUserMessage, StepReasoning, StepFileWrite, StepFileEdit, StepBash, StepFileRead}
	for i, want := range expected {
		if trace.Steps[i].Type != want {
			t.Errorf("step[%d]: expected type %s, got %s", i, want, trace.Steps[i].Type)
		}
	}

	// Check file write
	writeStep := trace.Steps[2]
	if writeStep.FilePath != "/app/login.go" {
		t.Errorf("write step: expected path '/app/login.go', got %q", writeStep.FilePath)
	}

	// Check edit
	editStep := trace.Steps[3]
	if editStep.FilePath != "/app/main.go" {
		t.Errorf("edit step: expected path '/app/main.go', got %q", editStep.FilePath)
	}

	// Check bash
	bashStep := trace.Steps[4]
	if bashStep.Command != "go test ./..." {
		t.Errorf("bash step: expected command 'go test ./...', got %q", bashStep.Command)
	}
	if bashStep.Summary != "Run tests" {
		t.Errorf("bash step: expected summary 'Run tests', got %q", bashStep.Summary)
	}

	// Files changed should include written/edited files
	if len(trace.FilesChanged) != 2 {
		t.Errorf("expected 2 files changed, got %d: %v", len(trace.FilesChanged), trace.FilesChanged)
	}

	// FileSteps should work
	loginSteps := trace.FileSteps("/app/login.go")
	if len(loginSteps) != 1 {
		t.Errorf("expected 1 step for login.go, got %d", len(loginSteps))
	}

	// Summary should be generated
	if trace.Summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestParseGenericJSONL(t *testing.T) {
	jsonl := `{"type":"plan","content":"I'll add rate limiting using a token bucket"}
{"type":"file_read","path":"api/middleware.go"}
{"type":"file_edit","path":"api/middleware.go","description":"Add RateLimiter struct"}
{"type":"bash","command":"go test ./...","exit_code":0}
{"type":"reasoning","content":"Tests pass. Now I need to wire this into the router."}
{"type":"file_write","path":"api/ratelimit.go","description":"Create rate limit config"}
`

	trace, err := parseGenericReader(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if trace.Source != "generic" {
		t.Errorf("expected source 'generic', got %q", trace.Source)
	}

	if len(trace.Steps) != 6 {
		t.Fatalf("expected 6 steps, got %d", len(trace.Steps))
	}

	expected := []StepType{StepPlan, StepFileRead, StepFileEdit, StepBash, StepReasoning, StepFileWrite}
	for i, want := range expected {
		if trace.Steps[i].Type != want {
			t.Errorf("step[%d]: expected %s, got %s", i, want, trace.Steps[i].Type)
		}
	}

	if len(trace.FilesChanged) != 2 {
		t.Errorf("expected 2 files changed, got %d", len(trace.FilesChanged))
	}
}

func TestParseAider(t *testing.T) {
	md := `# aider chat started at 2026-01-15 10:00:00

#### make the function async

I'll modify the function to be async. Here's the change:

` + "```python" + `
async def foo():
    await bar()
` + "```" + `

#### /ask what does this code do?

The code implements a simple rate limiter using the token bucket algorithm.
`

	tmpFile := t.TempDir() + "/history.md"
	if err := writeTestFile(tmpFile, md); err != nil {
		t.Fatal(err)
	}

	trace, err := ParseAider(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if trace.Source != "aider" {
		t.Errorf("expected source 'aider', got %q", trace.Source)
	}

	// Should have user messages and reasoning blocks
	if len(trace.Steps) < 2 {
		t.Fatalf("expected at least 2 steps, got %d", len(trace.Steps))
	}

	// First step should be user message
	if trace.Steps[0].Type != StepUserMessage {
		t.Errorf("expected first step to be user message, got %s", trace.Steps[0].Type)
	}
}

func TestStepTypeString(t *testing.T) {
	tests := []struct {
		st   StepType
		want string
	}{
		{StepPlan, "plan"},
		{StepReasoning, "reasoning"},
		{StepFileRead, "read"},
		{StepFileWrite, "write"},
		{StepFileEdit, "edit"},
		{StepBash, "bash"},
		{StepType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.st.String(); got != tt.want {
			t.Errorf("StepType(%d).String() = %q, want %q", tt.st, got, tt.want)
		}
	}
}

func TestDetectNoRepo(t *testing.T) {
	path, format := Detect("/nonexistent/path")
	if path != "" {
		t.Errorf("expected empty path, got %q (format: %s)", path, format)
	}
}

func writeTestFile(path, content string) error {
	return writeFile(path, content)
}
