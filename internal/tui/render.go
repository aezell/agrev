package tui

import (
	"fmt"
	"math"
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

	// Finding annotation
	IsFinding  bool
	FindingRisk int // 0=low, 1=medium, 2=high (maps to model.RiskLevel)
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

// pulseColor interpolates between a dim and bright version of a color based on phase.
// Returns an animated lipgloss.Color that breathes between dim and full brightness.
func pulseColor(dimRGB, brightRGB [3]int, phase float64) lipgloss.Color {
	t := (math.Sin(phase) + 1) / 2 // 0.0 to 1.0
	r := dimRGB[0] + int(t*float64(brightRGB[0]-dimRGB[0]))
	g := dimRGB[1] + int(t*float64(brightRGB[1]-dimRGB[1]))
	b := dimRGB[2] + int(t*float64(brightRGB[2]-dimRGB[2]))
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// Finding color pairs: [dim, bright] for each risk level.
var (
	findingHighDim    = [3]int{0x8a, 0x5c, 0x3a} // muted orange
	findingHighBright = [3]int{0xff, 0xb8, 0x6c} // bright orange
	findingMedDim     = [3]int{0x8a, 0x8a, 0x4c} // muted yellow
	findingMedBright  = [3]int{0xf1, 0xfa, 0x8c} // bright yellow
	findingLowDim     = [3]int{0x8a, 0x8a, 0x8a} // muted white
	findingLowBright  = [3]int{0xf8, 0xf8, 0xf2} // bright white
)

// styleLine applies styling to a rendered line for unified view.
func styleLine(rl renderedLine, width int, phase float64) string {
	if rl.IsFinding {
		var dim, bright [3]int
		bold := false
		switch {
		case rl.FindingRisk >= 3:
			dim, bright = findingHighDim, findingHighBright
			bold = true
		case rl.FindingRisk >= 2:
			dim, bright = findingMedDim, findingMedBright
		default:
			dim, bright = findingLowDim, findingLowBright
		}
		color := pulseColor(dim, bright, phase)
		style := lipgloss.NewStyle().Foreground(color).Bold(bold)
		text := rl.Content
		if len(text) > width-2 {
			text = text[:width-3] + "…"
		}
		return style.Render(text)
	}

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
func styleLineSplit(rl renderedLine, halfWidth int, phase float64) (left, right string) {
	if rl.IsFinding {
		var dim, bright [3]int
		bold := false
		switch {
		case rl.FindingRisk >= 3:
			dim, bright = findingHighDim, findingHighBright
			bold = true
		case rl.FindingRisk >= 2:
			dim, bright = findingMedDim, findingMedBright
		default:
			dim, bright = findingLowDim, findingLowBright
		}
		color := pulseColor(dim, bright, phase)
		style := lipgloss.NewStyle().Foreground(color).Bold(bold)
		text := rl.Content
		if len(text) > halfWidth*2 {
			text = text[:halfWidth*2-1] + "…"
		}
		return style.Render(text), ""
	}

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
