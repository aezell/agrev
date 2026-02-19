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
