package cli

import (
	"testing"
)

func TestRootCommandHasSubcommands(t *testing.T) {
	cmds := rootCmd.Commands()
	names := make(map[string]bool)
	for _, c := range cmds {
		names[c.Name()] = true
	}

	for _, want := range []string{"review", "summary", "check", "version"} {
		if !names[want] {
			t.Errorf("root command missing subcommand %q", want)
		}
	}
}

func TestVersionOutput(t *testing.T) {
	// version vars are set via ldflags; in tests they have their defaults
	if version != "dev" {
		t.Errorf("expected default version %q, got %q", "dev", version)
	}
}

func TestCheckCommandAcceptsFormats(t *testing.T) {
	f := checkCmd.Flags().Lookup("format")
	if f == nil {
		t.Fatal("check command missing --format flag")
	}
	if f.DefValue != "text" {
		t.Errorf("expected default format 'text', got %q", f.DefValue)
	}
}

func TestReviewCommandHasOutputPatch(t *testing.T) {
	f := reviewCmd.Flags().Lookup("output-patch")
	if f == nil {
		t.Fatal("review command missing --output-patch flag")
	}
}

func TestHTMLEscape(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"a & b", "a &amp; b"},
		{`<a href="x">`, `&lt;a href=&quot;x&quot;&gt;`},
	}

	for _, tt := range tests {
		got := htmlEscape(tt.input)
		if got != tt.want {
			t.Errorf("htmlEscape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
