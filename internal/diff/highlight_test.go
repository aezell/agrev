package diff

import (
	"testing"
)

func TestHighlightLines(t *testing.T) {
	lines := []string{
		"package main",
		"",
		"func main() {",
		`	fmt.Println("hello")`,
		"}",
	}

	highlighted := HighlightLines("main.go", lines)

	if len(highlighted) != len(lines) {
		t.Fatalf("expected %d highlighted lines, got %d", len(lines), len(highlighted))
	}

	// First line should have tokens
	if len(highlighted[0].Tokens) == 0 {
		t.Error("expected tokens in first line")
	}

	// Plain text should match original
	if highlighted[0].Plain() != "package main" {
		t.Errorf("plain text mismatch: %q", highlighted[0].Plain())
	}
}

func TestHighlightLinesUnknownLanguage(t *testing.T) {
	lines := []string{"some content", "more content"}
	highlighted := HighlightLines("unknown.xyz123", lines)

	if len(highlighted) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(highlighted))
	}
	if highlighted[0].Plain() != "some content" {
		t.Errorf("expected plain passthrough, got %q", highlighted[0].Plain())
	}
}
