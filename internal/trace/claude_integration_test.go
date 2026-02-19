package trace

import (
	"os"
	"testing"
)

func TestParseRealClaudeCodeTrace(t *testing.T) {
	path := "/home/sprite/.claude/projects/-home-sprite/7227b516-1fb5-4caa-bebc-32999f20ed86.jsonl"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("no real Claude Code trace available")
	}

	trace, err := ParseClaudeCode(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if trace.Source != "claude-code" {
		t.Errorf("expected source 'claude-code', got %q", trace.Source)
	}

	if len(trace.Steps) == 0 {
		t.Error("expected steps from real trace")
	}

	// Log stats for inspection
	types := make(map[string]int)
	for _, s := range trace.Steps {
		types[s.Type.String()]++
	}
	t.Logf("Session: %s", trace.SessionID)
	t.Logf("Steps: %d", len(trace.Steps))
	t.Logf("Files: %d", len(trace.FilesChanged))
	for k, v := range types {
		t.Logf("  %s: %d", k, v)
	}
	t.Logf("Files changed:")
	for _, f := range trace.FilesChanged {
		t.Logf("  %s", f)
	}
	for i, s := range trace.Steps[:min(15, len(trace.Steps))] {
		t.Logf("  [%d] %s: %s", i, s.Type, s.Summary)
	}
}
