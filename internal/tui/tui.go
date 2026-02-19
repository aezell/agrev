// Package tui implements the Bubble Tea terminal user interface.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/aezell/agrev/internal/analysis"
	"github.com/aezell/agrev/internal/diff"
	"github.com/aezell/agrev/internal/model"
	"github.com/aezell/agrev/internal/trace"
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

	// Analysis
	analysisResults *analysis.Results
	fileFindings    []analysis.Finding // findings for current file

	// Review decisions
	decisions map[int]model.ReviewDecision // fileIndex -> decision

	// Summary view
	showSummary   bool
	summaryScroll int

	// Help
	showHelp bool
}

// New creates a new TUI model from a parsed diff set and optional trace.
func New(ds *diff.DiffSet, t *trace.Trace, ar *analysis.Results) Model {
	m := Model{
		diffSet:         ds,
		trace:           t,
		splitView:       false,
		analysisResults: ar,
		decisions:       make(map[int]model.ReviewDecision),
	}
	m.updateLines()
	m.updateTraceSteps()
	m.updateFileFindings()
	return m
}

func (m *Model) updateFileFindings() {
	if m.analysisResults == nil || len(m.diffSet.Files) == 0 {
		m.fileFindings = nil
		return
	}

	byFile := m.analysisResults.ByFile()
	name := m.diffSet.Files[m.fileIndex].Name()
	m.fileFindings = byFile[name]
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
		// In summary view, handle differently
		if m.showSummary {
			return m.updateSummary(msg)
		}

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
				m.updateFileFindings()
			}

		case key.Matches(msg, keys.PrevFile):
			if m.fileIndex > 0 {
				m.fileIndex--
				m.scrollOffset = 0
				m.traceScroll = 0
				m.updateLines()
				m.updateTraceSteps()
				m.updateFileFindings()
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

		case key.Matches(msg, keys.Approve):
			if len(m.diffSet.Files) > 0 {
				m.decisions[m.fileIndex] = model.DecisionApproved
				m.advanceAfterDecision()
			}

		case key.Matches(msg, keys.Reject):
			if len(m.diffSet.Files) > 0 {
				m.decisions[m.fileIndex] = model.DecisionRejected
				m.advanceAfterDecision()
			}

		case key.Matches(msg, keys.Undo):
			if len(m.diffSet.Files) > 0 {
				delete(m.decisions, m.fileIndex)
			}

		case key.Matches(msg, keys.Finish):
			m.showSummary = true
			m.summaryScroll = 0
		}
	}

	return m, nil
}

func (m *Model) advanceAfterDecision() {
	// Auto-advance to the next undecided file
	for i := m.fileIndex + 1; i < len(m.diffSet.Files); i++ {
		if _, decided := m.decisions[i]; !decided {
			m.fileIndex = i
			m.scrollOffset = 0
			m.traceScroll = 0
			m.updateLines()
			m.updateTraceSteps()
			m.updateFileFindings()
			return
		}
	}
	// If all remaining are decided, stay on current file
}

func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Down):
		m.summaryScroll++
	case key.Matches(msg, keys.Up):
		if m.summaryScroll > 0 {
			m.summaryScroll--
		}
	case key.Matches(msg, keys.Finish):
		// Pressing Enter on summary exits
		return m, tea.Quit
	case msg.String() == "esc":
		// Go back to review
		m.showSummary = false
	}
	return m, nil
}

// ReviewDecisions returns the current per-file decisions.
func (m Model) ReviewDecisions() map[int]model.ReviewDecision {
	return m.decisions
}

// DecisionCounts returns counts of approved, rejected, and pending files.
func (m Model) DecisionCounts() (approved, rejected, pending int) {
	for i := range m.diffSet.Files {
		switch m.decisions[i] {
		case model.DecisionApproved:
			approved++
		case model.DecisionRejected:
			rejected++
		default:
			pending++
		}
	}
	return
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

	if m.showSummary {
		return m.renderSummary()
	}

	if m.showHelp {
		return m.renderHelp()
	}

	// Layout: file list on left, diff in center, trace on right (if shown)
	// Each bordered panel adds 4 chars (2 border + 2 padding) beyond its Width().
	const panelChrome = 4 // border (2) + padding (2) per panel
	const gap = 1         // space between panels

	fileListWidth := m.fileListWidth()
	mainHeight := m.height - 2 // status bar

	// Calculate diff and trace widths
	// Total budget: m.width = fileList(width+chrome) + gap + diff(width+chrome) [+ gap + trace(width+chrome)]
	var diffWidth, traceWidth int
	if m.showTrace && m.trace != nil {
		available := m.width - (fileListWidth + panelChrome) - gap - gap - panelChrome - panelChrome
		traceWidth = available * 35 / 100
		if traceWidth < 26 {
			traceWidth = 26
		}
		diffWidth = available - traceWidth
	} else {
		diffWidth = m.width - (fileListWidth + panelChrome) - gap - panelChrome
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

		// Decision indicator
		var indicator string
		switch m.decisions[i] {
		case model.DecisionApproved:
			indicator = fileApprovedStyle.Render("V ")
		case model.DecisionRejected:
			indicator = fileRejectedStyle.Render("X ")
		default:
			indicator = filePendingStyle.Render("- ")
		}

		maxName := width - 12
		if maxName > 0 && len(name) > maxName {
			name = "…" + name[len(name)-maxName+1:]
		}

		stats := fmt.Sprintf("+%d -%d", f.AddedLines, f.DeletedLines)
		line := fmt.Sprintf("%-*s %s", maxName, name, stats)

		var style lipgloss.Style
		if i == m.fileIndex {
			style = fileItemSelectedStyle
		} else if m.decisions[i] == model.DecisionApproved {
			style = lipgloss.NewStyle().Foreground(colorGreen)
		} else if m.decisions[i] == model.DecisionRejected {
			style = lipgloss.NewStyle().Foreground(colorRed)
		} else if f.IsNew {
			style = fileItemNewStyle
		} else if f.IsDeleted {
			style = fileItemDeletedStyle
		} else {
			style = fileItemStyle
		}

		b.WriteString(indicator + style.Width(width - 8).Render(line))
		if i < len(m.diffSet.Files)-1 {
			b.WriteByte('\n')
		}
	}

	innerHeight := height - 2
	content := b.String()
	// Clip to prevent overflow
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
		content = strings.Join(contentLines, "\n")
	}
	return fileListStyle.Width(width).Height(innerHeight).Render(content)
}

func (m Model) renderDiffView(width, height int) string {
	if len(m.diffSet.Files) == 0 {
		return diffViewStyle.Width(width).Height(height - 2).Render("No changes")
	}

	f := m.diffSet.Files[m.fileIndex]
	innerWidth := width // content width inside the border+padding
	innerHeight := height - 2

	headerText := f.Name()
	if len(m.fileFindings) > 0 {
		headerText += fmt.Sprintf("  [%d findings]", len(m.fileFindings))
	}
	header := fileHeaderStyle.Render(headerText)

	visibleLines := innerHeight - 2
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Build line->findings map for inline annotations
	findingsByLine := make(map[int][]analysis.Finding)
	var fileLevelFindings []analysis.Finding
	for _, fin := range m.fileFindings {
		if fin.Line == 0 {
			fileLevelFindings = append(fileLevelFindings, fin)
		} else {
			findingsByLine[fin.Line] = append(findingsByLine[fin.Line], fin)
		}
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteByte('\n')

	// Header with its bottom padding takes 2 lines
	usedLines := 2

	// Show file-level findings under the header
	for _, fin := range fileLevelFindings {
		if usedLines >= visibleLines {
			break
		}
		b.WriteString(renderFinding(fin, innerWidth))
		b.WriteByte('\n')
		usedLines++
	}

	diffLines := visibleLines - usedLines
	if diffLines < 1 {
		diffLines = 1
	}

	if m.splitView {
		m.renderSplitDiff(&b, innerWidth, diffLines, findingsByLine)
	} else {
		m.renderUnifiedDiff(&b, innerWidth, diffLines, findingsByLine)
	}

	// Clip content to innerHeight lines to prevent overflow
	content := b.String()
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
		content = strings.Join(contentLines, "\n")
	}

	borderStyle := diffViewStyle
	if m.focusPanel == 0 && m.showTrace {
		borderStyle = borderStyle.BorderForeground(colorBlue)
	}
	return borderStyle.Width(width).Height(innerHeight).Render(content)
}

func (m Model) renderUnifiedDiff(b *strings.Builder, width, visibleLines int, findingsByLine map[int][]analysis.Finding) {
	end := m.scrollOffset + visibleLines
	if end > len(m.lines) {
		end = len(m.lines)
	}

	linesWritten := 0
	for i := m.scrollOffset; i < end && linesWritten < visibleLines; i++ {
		rl := m.lines[i]
		b.WriteString(styleLine(rl, width))
		linesWritten++

		// Show inline findings for this line's new line number
		if rl.NewNum > 0 {
			if findings, ok := findingsByLine[rl.NewNum]; ok {
				for _, fin := range findings {
					if linesWritten >= visibleLines {
						break
					}
					b.WriteByte('\n')
					b.WriteString(renderFinding(fin, width))
					linesWritten++
				}
			}
		}

		if linesWritten < visibleLines {
			b.WriteByte('\n')
		}
	}
}

func (m Model) renderSplitDiff(b *strings.Builder, width, visibleLines int, findingsByLine map[int][]analysis.Finding) {
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
	innerWidth := width // content width inside the border+padding
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

	// Clip to prevent overflow
	content := b.String()
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > innerHeight {
		contentLines = contentLines[:innerHeight]
		content = strings.Join(contentLines, "\n")
	}

	borderStyle := traceViewStyle
	if m.focusPanel == 1 {
		borderStyle = borderStyle.BorderForeground(colorBlue)
	}
	return borderStyle.Width(width).Height(innerHeight).Render(content)
}

func renderFinding(fin analysis.Finding, width int) string {
	var style lipgloss.Style
	switch {
	case fin.Risk >= model.RiskHigh:
		style = findingHighStyle
	case fin.Risk >= model.RiskMedium:
		style = findingMediumStyle
	default:
		style = findingLowStyle
	}

	loc := ""
	if fin.Line > 0 {
		loc = fmt.Sprintf(":%d", fin.Line)
	}

	text := fmt.Sprintf("  >> [%s%s] %s", fin.Pass, loc, fin.Message)
	maxLen := width - 2
	if maxLen > 0 && len(text) > maxLen {
		text = text[:maxLen-1] + "…"
	}

	return style.Render(text)
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

	if m.analysisResults != nil && len(m.analysisResults.Findings) > 0 {
		right += fmt.Sprintf("  risk:%s", m.analysisResults.MaxRisk())
	}

	if m.trace != nil {
		traceInfo := "t:trace"
		if m.showTrace {
			traceInfo = fmt.Sprintf("t:trace[%d]", len(m.traceSteps))
		}
		right += "  " + traceInfo
	}

	approved, rejected, pending := m.DecisionCounts()
	if approved > 0 || rejected > 0 {
		right += fmt.Sprintf("  %dV %dX %d?", approved, rejected, pending)
	}

	right += "  ? help"

	barGap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if barGap < 0 {
		barGap = 0
	}

	content := left + strings.Repeat(" ", barGap) + right
	bar := lipgloss.NewStyle().
		Foreground(colorFg).
		Background(colorBgLight).
		Width(m.width).
		Render(content)
	return bar
}

func (m Model) renderSummary() string {
	var b strings.Builder

	b.WriteString(summaryHeaderStyle.Render("Review Summary"))
	b.WriteString("\n\n")

	approved, rejected, pending := m.DecisionCounts()
	total := len(m.diffSet.Files)

	b.WriteString(fmt.Sprintf("  %d file(s) reviewed out of %d\n\n", total-pending, total))

	if approved > 0 {
		b.WriteString(summaryApprovedStyle.Render(fmt.Sprintf("  V Approved: %d", approved)))
		b.WriteString("\n")
	}
	if rejected > 0 {
		b.WriteString(summaryRejectedStyle.Render(fmt.Sprintf("  X Rejected: %d", rejected)))
		b.WriteString("\n")
	}
	if pending > 0 {
		b.WriteString(summaryPendingStyle.Render(fmt.Sprintf("  ? Pending:  %d", pending)))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// List files by decision
	for i, f := range m.diffSet.Files {
		name := f.Name()
		switch m.decisions[i] {
		case model.DecisionApproved:
			b.WriteString(summaryApprovedStyle.Render(fmt.Sprintf("  V %s", name)))
		case model.DecisionRejected:
			b.WriteString(summaryRejectedStyle.Render(fmt.Sprintf("  X %s", name)))
		default:
			b.WriteString(summaryPendingStyle.Render(fmt.Sprintf("  ? %s", name)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpBarStyle.Render("  Press Enter to exit  |  Esc to go back"))

	return b.String()
}

func (m Model) renderHelp() string {
	var b strings.Builder

	b.WriteString(fileHeaderStyle.Render("agrev — Keyboard Shortcuts"))
	b.WriteString("\n\n")

	helpItems := []struct{ key, desc string }{
		{"j/k", "Scroll up/down"},
		{"n", "Next file"},
		{"N", "Previous file"},
		{"]", "Next hunk"},
		{"[", "Previous hunk"},
		{"a", "Approve current file"},
		{"x", "Reject current file"},
		{"u", "Undo decision"},
		{"Enter", "Finish review (summary)"},
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

// Run starts the TUI application and returns the review result.
func Run(ds *diff.DiffSet, t *trace.Trace, ar *analysis.Results) (*ReviewResult, error) {
	m := New(ds, t, ar)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	fm := finalModel.(Model)
	result := &ReviewResult{
		Decisions: fm.decisions,
		Files:     ds.Files,
	}
	return result, nil
}
