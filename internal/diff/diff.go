// Package diff handles parsing git diffs into structured representations.
package diff

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// File represents a single file in a diff with its parsed fragments.
type File struct {
	OldName    string
	NewName    string
	IsNew      bool
	IsDeleted  bool
	IsRenamed  bool
	IsBinary   bool
	Fragments  []*gitdiff.TextFragment
	AddedLines int
	DeletedLines int
}

// Name returns the display name for the file.
func (f *File) Name() string {
	if f.IsRenamed {
		return fmt.Sprintf("%s → %s", f.OldName, f.NewName)
	}
	if f.IsNew {
		return f.NewName
	}
	if f.IsDeleted {
		return f.OldName
	}
	if f.NewName != "" {
		return f.NewName
	}
	return f.OldName
}

// DiffSet holds the parsed diff for all files.
type DiffSet struct {
	Files []*File
	Raw   string // the raw unified diff text
}

// Stats returns aggregate statistics.
func (ds *DiffSet) Stats() (files, added, deleted int) {
	files = len(ds.Files)
	for _, f := range ds.Files {
		added += f.AddedLines
		deleted += f.DeletedLines
	}
	return
}

// Parse reads a unified diff string and returns a DiffSet.
func Parse(raw string) (*DiffSet, error) {
	parsed, _, err := gitdiff.Parse(strings.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parsing diff: %w", err)
	}

	ds := &DiffSet{Raw: raw}
	for _, f := range parsed {
		df := &File{
			IsNew:     f.IsNew,
			IsDeleted: f.IsDelete,
			IsRenamed: f.IsRename,
			IsBinary:  f.IsBinary,
		}

		if f.OldName != "" {
			df.OldName = f.OldName
		}
		if f.NewName != "" {
			df.NewName = f.NewName
		}

		for _, frag := range f.TextFragments {
			df.Fragments = append(df.Fragments, frag)
			for _, line := range frag.Lines {
				switch line.Op {
				case gitdiff.OpAdd:
					df.AddedLines++
				case gitdiff.OpDelete:
					df.DeletedLines++
				}
			}
		}

		ds.Files = append(ds.Files, df)
	}

	return ds, nil
}

// GitDiff runs `git diff` with the given arguments and returns the raw output.
func GitDiff(repoDir string, args ...string) (string, error) {
	cmdArgs := append([]string{"diff"}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Dir = repoDir
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}

	return string(out), nil
}

// GitDiffHead returns the diff of HEAD against its parent.
func GitDiffHead(repoDir string, contextLines int) (string, error) {
	return GitDiff(repoDir, fmt.Sprintf("-U%d", contextLines), "HEAD~1", "HEAD")
}

// GitDiffRange returns the diff for a commit range like "main...HEAD".
func GitDiffRange(repoDir string, commitRange string, contextLines int) (string, error) {
	return GitDiff(repoDir, fmt.Sprintf("-U%d", contextLines), commitRange)
}
