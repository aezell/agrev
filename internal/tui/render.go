package tui

import (
	"fmt"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/charmbracelet/lipgloss"
	"github.com/aezell/agrev/internal/diff"
)

// renderedLine is a single line of diff output ready for display.
type renderedLine struct {
	OldNum  int    // 0 means not applicable (add-only)
	NewNum  int    // 0 means not applicable (delete-only)
	Op      gitdiff.LineOp
	Content string // raw text content (no trailing newline)
	IsHunk  bool   // true if this is a hunk header

	// Syntax highlighting tokens (nil = no highlighting)
	Tokens []diff.Token
}

// renderFile produces renderedLines for a file's diff fragments.
func renderFile(f *diff.File) []renderedLine {
	var lines []renderedLine

	// Collect all content lines for syntax highlighting
	var contentLines []string
	for _, frag := range f.Fragments {
		for _, line := range frag.Lines {
			contentLines = append(contentLines, strings.TrimRight(line.Line, "\n\r"))
		}
	}

	// Highlight all content lines at once
	highlighted := diff.HighlightLines(f.Name(), contentLines)
	hlIdx := 0

	for i, frag := range f.Fragments {
		// Hunk header
		header := formatHunkHeader(frag)
		lines = append(lines, renderedLine{
			IsHunk:  true,
			Content: header,
		})

		oldLine := int(frag.OldPosition)
		newLine := int(frag.NewPosition)

		for _, line := range frag.Lines {
			rl := renderedLine{
				Op:      line.Op,
				Content: strings.TrimRight(line.Line, "\n\r"),
			}

			if hlIdx < len(highlighted) {
				rl.Tokens = highlighted[hlIdx].Tokens
				hlIdx++
			}

			switch line.Op {
			case gitdiff.OpContext:
				rl.OldNum = oldLine
				rl.NewNum = newLine
				oldLine++
				newLine++
			case gitdiff.OpDelete:
				rl.OldNum = oldLine
				oldLine++
			case gitdiff.OpAdd:
				rl.NewNum = newLine
				newLine++
			}

			lines = append(lines, rl)
		}

		// Add a blank separator between hunks (but not after the last)
		if i < len(f.Fragments)-1 {
			lines = append(lines, renderedLine{Content: ""})
		}
	}

	return lines
}

func formatHunkHeader(frag *gitdiff.TextFragment) string {
	old := fmt.Sprintf("-%d", frag.OldPosition)
	if frag.OldLines != 1 {
		old += fmt.Sprintf(",%d", frag.OldLines)
	}
	new := fmt.Sprintf("+%d", frag.NewPosition)
	if frag.NewLines != 1 {
		new += fmt.Sprintf(",%d", frag.NewLines)
	}

	header := fmt.Sprintf("@@ %s %s @@", old, new)
	if frag.Comment != "" {
		header += " " + frag.Comment
	}
	return header
}

// renderHighlightedContent renders line content with syntax tokens and diff coloring.
func renderHighlightedContent(rl renderedLine, prefix string) string {
	if len(rl.Tokens) == 0 {
		return prefix + rl.Content
	}

	var b strings.Builder
	b.WriteString(prefix)

	for _, tok := range rl.Tokens {
		if tok.Color != "" && rl.Op == gitdiff.OpContext {
			// Apply syntax color only for context lines
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(tok.Color)).Render(tok.Text))
		} else {
			b.WriteString(tok.Text)
		}
	}

	return b.String()
}

// styleLine applies styling to a rendered line for unified view.
func styleLine(rl renderedLine, width int) string {
	if rl.IsHunk {
		return hunkHeaderStyle.Width(width).Render(rl.Content)
	}

	var oldNum, newNum string
	if rl.OldNum > 0 {
		oldNum = fmt.Sprintf("%4d", rl.OldNum)
	} else {
		oldNum = "    "
	}
	if rl.NewNum > 0 {
		newNum = fmt.Sprintf("%4d", rl.NewNum)
	} else {
		newNum = "    "
	}

	lineNums := lineNumberStyle.Render(oldNum) + " " + lineNumberStyle.Render(newNum)

	var prefix string
	var style func(string) string

	switch rl.Op {
	case gitdiff.OpAdd:
		prefix = "+"
		style = func(s string) string { return addedLineStyle.Render(s) }
	case gitdiff.OpDelete:
		prefix = "-"
		style = func(s string) string { return deletedLineStyle.Render(s) }
	default:
		prefix = " "
		style = nil // context lines get syntax highlighting instead
	}

	var content string
	if style == nil {
		// Context line: use syntax highlighting
		content = renderHighlightedContent(rl, prefix)
	} else {
		content = style(prefix + rl.Content)
	}

	// Truncate long lines
	maxContent := width - 12
	if maxContent > 0 && lipgloss.Width(content) > maxContent {
		// Simple truncation for styled strings
		content = truncate(prefix+rl.Content, maxContent)
		if style != nil {
			content = style(content)
		}
	}

	return lineNums + " " + content
}

// styleLineSplit renders a line for split (side-by-side) view.
func styleLineSplit(rl renderedLine, halfWidth int) (left, right string) {
	if rl.IsHunk {
		half := hunkHeaderStyle.Width(halfWidth).Render(rl.Content)
		return half, ""
	}

	maxContent := halfWidth - 7

	switch rl.Op {
	case gitdiff.OpDelete:
		num := fmt.Sprintf("%4d", rl.OldNum)
		content := truncate(rl.Content, maxContent)
		left = lineNumberStyle.Render(num) + " " + deletedLineStyle.Render("-"+content)
		right = strings.Repeat(" ", halfWidth)
	case gitdiff.OpAdd:
		left = strings.Repeat(" ", halfWidth)
		num := fmt.Sprintf("%4d", rl.NewNum)
		content := truncate(rl.Content, maxContent)
		right = lineNumberStyle.Render(num) + " " + addedLineStyle.Render("+"+content)
	default:
		oldNum := "    "
		newNum := "    "
		if rl.OldNum > 0 {
			oldNum = fmt.Sprintf("%4d", rl.OldNum)
		}
		if rl.NewNum > 0 {
			newNum = fmt.Sprintf("%4d", rl.NewNum)
		}
		content := truncate(rl.Content, maxContent)
		left = lineNumberStyle.Render(oldNum) + " " + contextLineStyle.Render(" "+content)
		right = lineNumberStyle.Render(newNum) + " " + contextLineStyle.Render(" "+content)
	}

	return left, right
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}
