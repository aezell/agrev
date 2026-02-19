package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sprite-ai/agrev/internal/diff"
)

const testDiff = `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

 func main() {
-	println("hello")
+	println("hello world")
+	println("goodbye")
 }
diff --git a/util.go b/util.go
new file mode 100644
--- /dev/null
+++ b/util.go
@@ -0,0 +1,5 @@
+package main
+
+func add(a, b int) int {
+	return a + b
+}
`

func setupModel(t *testing.T) Model {
	t.Helper()
	ds, err := diff.Parse(testDiff)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	m := New(ds)
	// Simulate window size
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return newM.(Model)
}

func TestModelInit(t *testing.T) {
	m := setupModel(t)

	if m.fileIndex != 0 {
		t.Errorf("expected fileIndex 0, got %d", m.fileIndex)
	}
	if len(m.lines) == 0 {
		t.Error("expected lines to be rendered")
	}
	if m.diffSet == nil {
		t.Error("expected diffSet to be set")
	}
}

func TestNavigation(t *testing.T) {
	m := setupModel(t)

	// Move to next file
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newM.(Model)
	if m.fileIndex != 1 {
		t.Errorf("expected fileIndex 1 after next, got %d", m.fileIndex)
	}

	// Move past end — should stay
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newM.(Model)
	if m.fileIndex != 1 {
		t.Errorf("expected fileIndex 1 at end, got %d", m.fileIndex)
	}

	// Move back
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	m = newM.(Model)
	if m.fileIndex != 0 {
		t.Errorf("expected fileIndex 0 after prev, got %d", m.fileIndex)
	}
}

func TestScrolling(t *testing.T) {
	m := setupModel(t)

	// Scroll down
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newM.(Model)
	if m.scrollOffset != 1 {
		t.Errorf("expected scrollOffset 1, got %d", m.scrollOffset)
	}

	// Scroll up
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newM.(Model)
	if m.scrollOffset != 0 {
		t.Errorf("expected scrollOffset 0, got %d", m.scrollOffset)
	}

	// Can't scroll above 0
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newM.(Model)
	if m.scrollOffset != 0 {
		t.Errorf("expected scrollOffset 0 at top, got %d", m.scrollOffset)
	}
}

func TestToggleView(t *testing.T) {
	m := setupModel(t)

	if m.splitView {
		t.Error("expected unified view by default")
	}

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = newM.(Model)
	if !m.splitView {
		t.Error("expected split view after toggle")
	}

	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = newM.(Model)
	if m.splitView {
		t.Error("expected unified view after second toggle")
	}
}

func TestViewRenders(t *testing.T) {
	m := setupModel(t)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// Should contain the filename
	if !strings.Contains(view, "main.go") {
		t.Error("expected view to contain 'main.go'")
	}

	// Should contain diff content
	if !strings.Contains(view, "hello") {
		t.Error("expected view to contain 'hello'")
	}
}

func TestHelpToggle(t *testing.T) {
	m := setupModel(t)

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newM.(Model)
	if !m.showHelp {
		t.Error("expected help to be shown")
	}

	view := m.View()
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("expected help view to contain shortcuts")
	}
}
