package diff

import (
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// HighlightedLine represents a line with syntax-highlighted tokens.
type HighlightedLine struct {
	Tokens []Token
}

// Token is a syntax-highlighted chunk of text.
type Token struct {
	Text  string
	Color string // ANSI color string, empty for default
}

// Plain returns the concatenated plain text of all tokens.
func (hl HighlightedLine) Plain() string {
	var b strings.Builder
	for _, t := range hl.Tokens {
		b.WriteString(t.Text)
	}
	return b.String()
}

// HighlightLines applies syntax highlighting to source lines for a given filename.
// Returns one HighlightedLine per input line.
func HighlightLines(filename string, lines []string) []HighlightedLine {
	lexer := lexerForFile(filename)
	if lexer == nil {
		return plainLines(lines)
	}

	source := strings.Join(lines, "\n")
	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return plainLines(lines)
	}

	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}

	result := make([]HighlightedLine, 0, len(lines))
	current := HighlightedLine{}

	for _, token := range iterator.Tokens() {
		// Split tokens that span multiple lines
		parts := strings.Split(token.Value, "\n")
		for i, part := range parts {
			if i > 0 {
				result = append(result, current)
				current = HighlightedLine{}
			}
			if part != "" {
				current.Tokens = append(current.Tokens, Token{
					Text:  part,
					Color: tokenColor(style, token.Type),
				})
			}
		}
	}
	result = append(result, current)

	// Pad result if we have fewer lines than input
	for len(result) < len(lines) {
		result = append(result, HighlightedLine{Tokens: []Token{{Text: ""}}})
	}

	return result
}

func plainLines(lines []string) []HighlightedLine {
	result := make([]HighlightedLine, len(lines))
	for i, line := range lines {
		result[i] = HighlightedLine{Tokens: []Token{{Text: line}}}
	}
	return result
}

func lexerForFile(filename string) chroma.Lexer {
	lexer := lexers.Match(filename)
	if lexer == nil {
		ext := filepath.Ext(filename)
		if ext != "" {
			lexer = lexers.Match("file" + ext)
		}
	}
	if lexer != nil {
		lexer = chroma.Coalesce(lexer)
	}
	return lexer
}

func tokenColor(style *chroma.Style, tt chroma.TokenType) string {
	entry := style.Get(tt)
	if entry.Colour.IsSet() {
		return entry.Colour.String()
	}
	return ""
}
