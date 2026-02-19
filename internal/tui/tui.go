// Package tui implements the Bubble Tea terminal user interface.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sprite-ai/agrev/internal/diff"
	"github.com/sprite-ai/agrev/internal/trace"
)

// Model is the top-level Bubble Tea model for agrev.
type Model struct {
	diffSet *diff.DiffSet
	trace   *trace.Trace // nil if no trace

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

	// Trace panel
	showTrace    bool
	traceScroll  int
	traceSteps   []trace.Step // steps relevant to current file (or all if no file filter)

	// Panels
	focusPanel int // 0=diff, 1=trace

	// Help
	showHelp bool
}

// New creates a new TUI model from a parsed diff set and optional trace.
func New(ds *diff.DiffSet, t *trace.Trace) Model {
	m := Model{
		diffSet:   ds,
		trace:     t,
		splitView: false,
	}
	m.updateLines()
	m.updateTraceSteps()
	return m
}

func (m *Model) updateLines() {
	if len(m.diffSet.Files) == 0 {
		m.lines = nil
		return
	}
	m.lines = renderFile(m.diffSet.Files[m.fileIndex])
}

func (m *Model) updateTraceSteps() {
	if m.trace == nil {
		m.traceSteps = nil
		return
	}

	if len(m.diffSet.Files) == 0 {
		m.traceSteps = m.trace.Steps
		return
	}

	// Get steps related to the current file
	f := m.diffSet.Files[m.fileIndex]
	name := f.Name()

	// Match by filename (trace may have absolute paths)
	var filtered []trace.Step
	for _, s := range m.trace.Steps {
		if s.FilePath != "" {
			base := filepath.Base(s.FilePath)
			if base == filepath.Base(name) || strings.HasSuffix(s.FilePath, name) {
				filtered = append(filtered, s)
			}
		}
	}

	if len(filtered) > 0 {
		m.traceSteps = filtered
	} else {
		// Show all steps if no file-specific matches
		m.traceSteps = m.trace.Steps
	}
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
		m.viewHeight = m.height - 4
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Down):
			if m.focusPanel == 0 {
				if m.scrollOffset < len(m.lines)-1 {
					m.scrollOffset++
				}
			} else {
				if m.traceScroll < len(m.traceSteps)-1 {
					m.traceScroll++
				}
			}

		case key.Matches(msg, keys.Up):
			if m.focusPanel == 0 {
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
			} else {
				if m.traceScroll > 0 {
					m.traceScroll--
				}
			}

		case key.Matches(msg, keys.NextFile):
			if m.fileIndex < len(m.diffSet.Files)-1 {
				m.fileIndex++
				m.scrollOffset = 0
				m.traceScroll = 0
				m.updateLines()
				m.updateTraceSteps()
			}

		case key.Matches(msg, keys.PrevFile):
			if m.fileIndex > 0 {
				m.fileIndex--
				m.scrollOffset = 0
				m.traceScroll = 0
				m.updateLines()
				m.updateTraceSteps()
			}

		case key.Matches(msg, keys.NextHunk):
			m.jumpToNextHunk()

		case key.Matches(msg, keys.PrevHunk):
			m.jumpToPrevHunk()

		case key.Matches(msg, keys.Toggle):
			m.splitView = !m.splitView

		case key.Matches(msg, keys.Trace):
			if m.trace != nil {
				m.showTrace = !m.showTrace
				if !m.showTrace {
					m.focusPanel = 0
				}
			}

		case key.Matches(msg, keys.FocusSwap):
			if m.showTrace {
				m.focusPanel = 1 - m.focusPanel
			}

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

	// Layout: file list on left, diff in center, trace on right (if shown)
	fileListWidth := m.fileListWidth()
	mainHeight := m.height - 2 // status bar

	// Calculate diff and trace widths
	remaining := m.width - fileListWidth - 1 // gap between file list and diff
	var diffWidth, traceWidth int
	if m.showTrace && m.trace != nil {
		traceWidth = remaining * 35 / 100 // 35% for trace
		if traceWidth < 30 {
			traceWidth = 30
		}
		diffWidth = remaining - traceWidth - 1 // -1 for gap
	} else {
		diffWidth = remaining
	}

	fileList := m.renderFileList(fileListWidth, mainHeight)
	diffView := m.renderDiffView(diffWidth, mainHeight)

	var main string
	if m.showTrace && m.trace != nil {
		traceView := m.renderTracePanel(traceWidth, mainHeight)
		main = lipgloss.JoinHorizontal(lipgloss.Top, fileList, " ", diffView, " ", traceView)
	} else {
		main = lipgloss.JoinHorizontal(lipgloss.Top, fileList, " ", diffView)
	}

	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}

func (m Model) fileListWidth() int {
	maxLen := 20
	for _, f := range m.diffSet.Files {
		name := f.Name()
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}
	w := maxLen + 10
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

	innerHeight := height - 2
	content := b.String()
	return fileListStyle.Width(width).Height(innerHeight).Render(content)
}

func (m Model) renderDiffView(width, height int) string {
	if len(m.diffSet.Files) == 0 {
		return diffViewStyle.Width(width).Height(height - 2).Render("No changes")
	}

	f := m.diffSet.Files[m.fileIndex]
	innerWidth := width - 4
	innerHeight := height - 2

	header := fileHeaderStyle.Render(f.Name())

	visibleLines := innerHeight - 2
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

	borderStyle := diffViewStyle
	if m.focusPanel == 0 && m.showTrace {
		borderStyle = borderStyle.BorderForeground(colorBlue)
	}
	return borderStyle.Width(width).Height(innerHeight).Render(b.String())
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
	halfWidth := (width - 3) / 2

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

func (m Model) renderTracePanel(width, height int) string {
	innerWidth := width - 4
	innerHeight := height - 2

	var b strings.Builder

	// Header
	title := "Agent Trace"
	if m.trace != nil {
		title += fmt.Sprintf(" (%s)", m.trace.Source)
	}
	b.WriteString(traceHeaderStyle.Render(title))
	b.WriteByte('\n')

	if len(m.traceSteps) == 0 {
		b.WriteString(contextLineStyle.Render("No trace steps for this file"))
	} else {
		visibleLines := innerHeight - 2
		if visibleLines < 1 {
			visibleLines = 1
		}

		end := m.traceScroll + visibleLines
		if end > len(m.traceSteps) {
			end = len(m.traceSteps)
		}

		for i := m.traceScroll; i < end; i++ {
			step := m.traceSteps[i]
			b.WriteString(renderTraceStep(step, innerWidth, i == m.traceScroll))
			if i < end-1 {
				b.WriteByte('\n')
			}
		}
	}

	borderStyle := traceViewStyle
	if m.focusPanel == 1 {
		borderStyle = borderStyle.BorderForeground(colorBlue)
	}
	return borderStyle.Width(width).Height(innerHeight).Render(b.String())
}

func renderTraceStep(step trace.Step, width int, isCurrent bool) string {
	icon := stepIcon(step.Type)
	summary := step.Summary

	maxSummary := width - 4
	if len(summary) > maxSummary {
		summary = summary[:maxSummary-1] + "…"
	}

	line := fmt.Sprintf("%s %s", icon, summary)

	var style lipgloss.Style
	switch step.Type {
	case trace.StepFileWrite, trace.StepFileEdit:
		style = traceWriteStyle
	case trace.StepBash:
		style = traceBashStyle
	case trace.StepReasoning, trace.StepPlan:
		style = traceReasonStyle
	case trace.StepFileRead:
		style = traceReadStyle
	case trace.StepUserMessage:
		style = traceUserStyle
	default:
		style = contextLineStyle
	}

	return style.Width(width).Render(line)
}

func stepIcon(st trace.StepType) string {
	switch st {
	case trace.StepPlan:
		return "P"
	case trace.StepReasoning:
		return ">"
	case trace.StepFileRead:
		return "R"
	case trace.StepFileWrite:
		return "W"
	case trace.StepFileEdit:
		return "E"
	case trace.StepBash:
		return "$"
	case trace.StepUserMessage:
		return "U"
	default:
		return "."
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

	right := fmt.Sprintf("+%d -%d  %s", added, deleted, mode)

	if m.trace != nil {
		traceInfo := "t:trace"
		if m.showTrace {
			traceInfo = fmt.Sprintf("t:trace[%d]", len(m.traceSteps))
		}
		right += "  " + traceInfo
	}

	right += "  ? help "

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
		{"j/k", "Scroll up/down"},
		{"n/Tab", "Next file"},
		{"N/S-Tab", "Previous file"},
		{"]", "Next hunk"},
		{"[", "Previous hunk"},
		{"v", "Toggle unified/split view"},
		{"t", "Toggle trace panel"},
		{"Tab", "Switch focus (diff/trace)"},
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
func Run(ds *diff.DiffSet, t *trace.Trace) error {
	m := New(ds, t)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
