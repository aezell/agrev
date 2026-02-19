// Package tui implements the Bubble Tea terminal user interface.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sprite-ai/agrev/internal/diff"
)

// Model is the top-level Bubble Tea model for agrev.
type Model struct {
	diffSet *diff.DiffSet

	// UI state
	width  int
	height int

	// File list
	fileIndex int // currently selected file

	// Diff viewport
	scrollOffset int // scroll position within the current file's diff
	viewHeight   int // number of visible lines in the diff area

	// Rendered lines for the current file
	lines []renderedLine

	// View mode
	splitView bool

	// Help
	showHelp bool
}

// New creates a new TUI model from a parsed diff set.
func New(ds *diff.DiffSet) Model {
	m := Model{
		diffSet:   ds,
		splitView: false,
	}
	m.updateLines()
	return m
}

func (m *Model) updateLines() {
	if len(m.diffSet.Files) == 0 {
		m.lines = nil
		return
	}
	m.lines = renderFile(m.diffSet.Files[m.fileIndex])
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewHeight = m.height - 4 // status bar + help bar + borders
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Down):
			if m.scrollOffset < len(m.lines)-1 {
				m.scrollOffset++
			}

		case key.Matches(msg, keys.Up):
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}

		case key.Matches(msg, keys.NextFile):
			if m.fileIndex < len(m.diffSet.Files)-1 {
				m.fileIndex++
				m.scrollOffset = 0
				m.updateLines()
			}

		case key.Matches(msg, keys.PrevFile):
			if m.fileIndex > 0 {
				m.fileIndex--
				m.scrollOffset = 0
				m.updateLines()
			}

		case key.Matches(msg, keys.NextHunk):
			m.jumpToNextHunk()

		case key.Matches(msg, keys.PrevHunk):
			m.jumpToPrevHunk()

		case key.Matches(msg, keys.Toggle):
			m.splitView = !m.splitView

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
		}
	}

	return m, nil
}

func (m *Model) jumpToNextHunk() {
	for i := m.scrollOffset + 1; i < len(m.lines); i++ {
		if m.lines[i].IsHunk {
			m.scrollOffset = i
			return
		}
	}
}

func (m *Model) jumpToPrevHunk() {
	for i := m.scrollOffset - 1; i >= 0; i-- {
		if m.lines[i].IsHunk {
			m.scrollOffset = i
			return
		}
	}
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.showHelp {
		return m.renderHelp()
	}

	// Layout: file list on left, diff on right
	fileListWidth := m.fileListWidth()
	diffWidth := m.width - fileListWidth - 1 // -1 for gap

	fileList := m.renderFileList(fileListWidth, m.height-2)
	diffView := m.renderDiffView(diffWidth, m.height-2)

	main := lipgloss.JoinHorizontal(lipgloss.Top, fileList, " ", diffView)

	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}

func (m Model) fileListWidth() int {
	// Calculate based on longest filename, capped
	maxLen := 20
	for _, f := range m.diffSet.Files {
		name := f.Name()
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}
	w := maxLen + 10 // padding + stats
	if w > m.width/3 {
		w = m.width / 3
	}
	if w < 20 {
		w = 20
	}
	return w
}

func (m Model) renderFileList(width, height int) string {
	var b strings.Builder

	for i, f := range m.diffSet.Files {
		name := f.Name()

		// Truncate name if needed
		maxName := width - 8
		if maxName > 0 && len(name) > maxName {
			name = "…" + name[len(name)-maxName+1:]
		}

		stats := fmt.Sprintf("+%d -%d", f.AddedLines, f.DeletedLines)
		line := fmt.Sprintf("%-*s %s", maxName, name, stats)

		var style lipgloss.Style
		if i == m.fileIndex {
			style = fileItemSelectedStyle
		} else if f.IsNew {
			style = fileItemNewStyle
		} else if f.IsDeleted {
			style = fileItemDeletedStyle
		} else {
			style = fileItemStyle
		}

		b.WriteString(style.Width(width - 4).Render(line))
		if i < len(m.diffSet.Files)-1 {
			b.WriteByte('\n')
		}
	}

	innerHeight := height - 2 // borders
	content := b.String()
	return fileListStyle.Width(width).Height(innerHeight).Render(content)
}

func (m Model) renderDiffView(width, height int) string {
	if len(m.diffSet.Files) == 0 {
		return diffViewStyle.Width(width).Height(height - 2).Render("No changes")
	}

	f := m.diffSet.Files[m.fileIndex]
	innerWidth := width - 4  // borders + padding
	innerHeight := height - 2

	// File header
	header := fileHeaderStyle.Render(f.Name())

	// Calculate visible lines
	visibleLines := innerHeight - 2 // header takes some space
	if visibleLines < 1 {
		visibleLines = 1
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteByte('\n')

	if m.splitView {
		m.renderSplitDiff(&b, innerWidth, visibleLines)
	} else {
		m.renderUnifiedDiff(&b, innerWidth, visibleLines)
	}

	return diffViewStyle.Width(width).Height(innerHeight).Render(b.String())
}

func (m Model) renderUnifiedDiff(b *strings.Builder, width, visibleLines int) {
	end := m.scrollOffset + visibleLines
	if end > len(m.lines) {
		end = len(m.lines)
	}

	for i := m.scrollOffset; i < end; i++ {
		b.WriteString(styleLine(m.lines[i], width))
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
}

func (m Model) renderSplitDiff(b *strings.Builder, width, visibleLines int) {
	halfWidth := (width - 3) / 2 // -3 for separator

	end := m.scrollOffset + visibleLines
	if end > len(m.lines) {
		end = len(m.lines)
	}

	for i := m.scrollOffset; i < end; i++ {
		left, right := styleLineSplit(m.lines[i], halfWidth)
		b.WriteString(left)
		b.WriteString(" │ ")
		b.WriteString(right)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
}

func (m Model) renderStatusBar() string {
	nFiles, added, deleted := m.diffSet.Stats()

	left := fmt.Sprintf(" File %d/%d", m.fileIndex+1, nFiles)
	if len(m.lines) > 0 {
		left += fmt.Sprintf("  Line %d/%d", m.scrollOffset+1, len(m.lines))
	}

	mode := "unified"
	if m.splitView {
		mode = "split"
	}

	right := fmt.Sprintf("+%d -%d  %s  ? help ", added, deleted, mode)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	bar := statusBarStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
	return bar
}

func (m Model) renderHelp() string {
	var b strings.Builder

	b.WriteString(fileHeaderStyle.Render("agrev — Keyboard Shortcuts"))
	b.WriteString("\n\n")

	helpItems := []struct{ key, desc string }{
		{"↑/k", "Scroll up"},
		{"↓/j", "Scroll down"},
		{"n/Tab", "Next file"},
		{"N/S-Tab", "Previous file"},
		{"]", "Next hunk"},
		{"[", "Previous hunk"},
		{"v", "Toggle unified/split view"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
	}

	for _, item := range helpItems {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Width(12).Render(item.key),
			item.desc,
		))
	}

	b.WriteString("\n")
	b.WriteString(helpBarStyle.Render("Press ? to close help"))

	return b.String()
}

// Run starts the TUI application.
func Run(ds *diff.DiffSet) error {
	m := New(ds)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
