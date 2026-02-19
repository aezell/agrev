package diff

import (
	"testing"
)

const sampleDiff = `diff --git a/hello.go b/hello.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/hello.go
@@ -0,0 +1,11 @@
+package main
+
+import "fmt"
+
+func main() {
+	fmt.Println("hello")
+}
+
+func add(a, b int) int {
+	return a + b
+}
diff --git a/readme.md b/readme.md
index abc1234..def5678 100644
--- a/readme.md
+++ b/readme.md
@@ -1,3 +1,4 @@
 # Project

-Old description
+New description
+Added line
`

func TestParse(t *testing.T) {
	ds, err := Parse(sampleDiff)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(ds.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(ds.Files))
	}

	// First file: new file
	f0 := ds.Files[0]
	if !f0.IsNew {
		t.Error("expected hello.go to be new")
	}
	if f0.Name() != "hello.go" {
		t.Errorf("expected name 'hello.go', got %q", f0.Name())
	}
	if f0.AddedLines != 11 {
		t.Errorf("expected 11 added lines, got %d", f0.AddedLines)
	}

	// Second file: modified
	f1 := ds.Files[1]
	if f1.Name() != "readme.md" {
		t.Errorf("expected name 'readme.md', got %q", f1.Name())
	}
	if f1.AddedLines != 2 {
		t.Errorf("expected 2 added lines, got %d", f1.AddedLines)
	}
	if f1.DeletedLines != 1 {
		t.Errorf("expected 1 deleted line, got %d", f1.DeletedLines)
	}

	// Stats
	files, added, deleted := ds.Stats()
	if files != 2 {
		t.Errorf("stats: expected 2 files, got %d", files)
	}
	if added != 13 {
		t.Errorf("stats: expected 13 added, got %d", added)
	}
	if deleted != 1 {
		t.Errorf("stats: expected 1 deleted, got %d", deleted)
	}
}

func TestParseEmpty(t *testing.T) {
	ds, err := Parse("")
	if err != nil {
		t.Fatalf("Parse empty failed: %v", err)
	}
	if len(ds.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(ds.Files))
	}
}
