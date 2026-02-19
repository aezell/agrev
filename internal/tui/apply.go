package tui

import (
	"fmt"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/sprite-ai/agrev/internal/diff"
	"github.com/sprite-ai/agrev/internal/model"
)

// ReviewResult holds the outcome of an interactive review session.
type ReviewResult struct {
	Decisions map[int]model.ReviewDecision
	Files     []*diff.File
}

// ApprovedFiles returns only the files that were approved.
func (r *ReviewResult) ApprovedFiles() []*diff.File {
	var approved []*diff.File
	for i, f := range r.Files {
		if r.Decisions[i] == model.DecisionApproved {
			approved = append(approved, f)
		}
	}
	return approved
}

// RejectedFiles returns only the files that were rejected.
func (r *ReviewResult) RejectedFiles() []*diff.File {
	var rejected []*diff.File
	for i, f := range r.Files {
		if r.Decisions[i] == model.DecisionRejected {
			rejected = append(rejected, f)
		}
	}
	return rejected
}

// PendingFiles returns files with no decision.
func (r *ReviewResult) PendingFiles() []*diff.File {
	var pending []*diff.File
	for i, f := range r.Files {
		if _, ok := r.Decisions[i]; !ok {
			pending = append(pending, f)
		}
	}
	return pending
}

// GeneratePatch creates a unified diff string containing only the approved files.
func (r *ReviewResult) GeneratePatch() string {
	approved := r.ApprovedFiles()
	if len(approved) == 0 {
		return ""
	}

	var b strings.Builder
	for _, f := range approved {
		b.WriteString(formatFilePatch(f))
	}
	return b.String()
}

// GenerateCommitMessage creates a suggested commit message from approved changes.
func (r *ReviewResult) GenerateCommitMessage() string {
	approved := r.ApprovedFiles()
	if len(approved) == 0 {
		return ""
	}

	var b strings.Builder
	if len(approved) == 1 {
		f := approved[0]
		if f.IsNew {
			b.WriteString(fmt.Sprintf("Add %s", f.Name()))
		} else if f.IsDeleted {
			b.WriteString(fmt.Sprintf("Remove %s", f.Name()))
		} else {
			b.WriteString(fmt.Sprintf("Update %s", f.Name()))
		}
	} else {
		added, modified, deleted := 0, 0, 0
		for _, f := range approved {
			if f.IsNew {
				added++
			} else if f.IsDeleted {
				deleted++
			} else {
				modified++
			}
		}

		var parts []string
		if modified > 0 {
			parts = append(parts, fmt.Sprintf("update %d file(s)", modified))
		}
		if added > 0 {
			parts = append(parts, fmt.Sprintf("add %d file(s)", added))
		}
		if deleted > 0 {
			parts = append(parts, fmt.Sprintf("remove %d file(s)", deleted))
		}

		b.WriteString(strings.Join(parts, ", "))
		// Capitalize first letter
		msg := b.String()
		if len(msg) > 0 {
			b.Reset()
			b.WriteString(strings.ToUpper(msg[:1]) + msg[1:])
		}
	}

	b.WriteString("\n\nApproved files:\n")
	for _, f := range approved {
		b.WriteString(fmt.Sprintf("  - %s\n", f.Name()))
	}

	rejected := r.RejectedFiles()
	if len(rejected) > 0 {
		b.WriteString("\nRejected files:\n")
		for _, f := range rejected {
			b.WriteString(fmt.Sprintf("  - %s\n", f.Name()))
		}
	}

	return b.String()
}

// formatFilePatch reconstructs a unified diff for a single file.
func formatFilePatch(f *diff.File) string {
	var b strings.Builder

	oldName := f.OldName
	newName := f.NewName
	if oldName == "" {
		oldName = "/dev/null"
	}
	if newName == "" {
		newName = "/dev/null"
	}

	b.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", f.Name(), f.Name()))
	if f.IsNew {
		b.WriteString("new file mode 100644\n")
	} else if f.IsDeleted {
		b.WriteString("deleted file mode 100644\n")
	}
	b.WriteString(fmt.Sprintf("--- a/%s\n", oldName))
	b.WriteString(fmt.Sprintf("+++ b/%s\n", newName))

	for _, frag := range f.Fragments {
		b.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@",
			frag.OldPosition, frag.OldLines,
			frag.NewPosition, frag.NewLines))
		if frag.Comment != "" {
			b.WriteString(" " + frag.Comment)
		}
		b.WriteString("\n")

		for _, line := range frag.Lines {
			switch line.Op {
			case gitdiff.OpContext:
				b.WriteString(" " + line.Line)
			case gitdiff.OpDelete:
				b.WriteString("-" + line.Line)
			case gitdiff.OpAdd:
				b.WriteString("+" + line.Line)
			}
			if !strings.HasSuffix(line.Line, "\n") {
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}
