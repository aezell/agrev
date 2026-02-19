package analysis

import (
	"strings"
	"testing"

	"github.com/aezell/agrev/internal/diff"
	"github.com/aezell/agrev/internal/model"
)

// --- Dependency detection tests ---

const depDiff = `diff --git a/go.mod b/go.mod
index abc1234..def5678 100644
--- a/go.mod
+++ b/go.mod
@@ -3,4 +3,6 @@ module example.com/myapp
 go 1.21

 require (
+	github.com/newdep/foo v1.2.3
+	github.com/anotherdep/bar v0.1.0
 	github.com/existing/dep v1.0.0
 )
`

func TestNewDependencyPass(t *testing.T) {
	ds, err := diff.Parse(depDiff)
	if err != nil {
		t.Fatal(err)
	}

	findings := NewDependencyPass(ds, "")

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d: %v", len(findings), findings)
	}

	for _, f := range findings {
		if f.Pass != "deps" {
			t.Errorf("expected pass 'deps', got %q", f.Pass)
		}
		if f.Risk != model.RiskMedium {
			t.Errorf("expected medium risk, got %s", f.Risk)
		}
	}
}

const npmDiff = `diff --git a/package.json b/package.json
index abc1234..def5678 100644
--- a/package.json
+++ b/package.json
@@ -5,3 +5,5 @@
   "dependencies": {
     "express": "^4.0.0",
+    "lodash": "^4.17.21",
+    "axios": "^1.6.0"
   }
`

func TestNpmDependencyDetection(t *testing.T) {
	ds, err := diff.Parse(npmDiff)
	if err != nil {
		t.Fatal(err)
	}

	findings := NewDependencyPass(ds, "")
	if len(findings) != 2 {
		t.Fatalf("expected 2 npm findings, got %d: %v", len(findings), findings)
	}
}

// --- Security surface tests ---

const secDiffAuth = `diff --git a/auth.go b/auth.go
new file mode 100644
--- /dev/null
+++ b/auth.go
@@ -0,0 +1,7 @@
+package main
+
+import "os"
+
+func getToken() string {
+	return os.Getenv("API_TOKEN")
+}
`

const secDiffDB = `diff --git a/db.go b/db.go
index abc1234..def5678 100644
--- a/db.go
+++ b/db.go
@@ -10,3 +10,5 @@ func query() {
 	x := 1
 	y := 2
 	z := 3
+	db.Exec("DELETE FROM users WHERE id = ?", id)
+	cmd := exec.Command("bash", "-c", userInput)
`

func TestSecuritySurfacePass(t *testing.T) {
	ds, err := diff.Parse(secDiffAuth + secDiffDB)
	if err != nil {
		t.Fatal(err)
	}

	findings := SecuritySurfacePass(ds, "")

	if len(findings) == 0 {
		t.Fatal("expected security findings")
	}

	// Should detect: os.Getenv (env/secrets), db.Exec (SQL), exec.Command (subprocess)
	categories := make(map[string]bool)
	for _, f := range findings {
		for _, cat := range []string{"environment/secrets", "SQL/database", "subprocess/exec"} {
			if containsCI(f.Message, cat) {
				categories[cat] = true
			}
		}
	}

	for _, want := range []string{"environment/secrets", "SQL/database", "subprocess/exec"} {
		if !categories[want] {
			t.Errorf("missing security category %q in findings", want)
			for _, f := range findings {
				t.Logf("  finding: %s", f)
			}
		}
	}
}

// --- Anti-pattern tests ---

const antiDiff = `diff --git a/handler.py b/handler.py
new file mode 100644
--- /dev/null
+++ b/handler.py
@@ -0,0 +1,14 @@
+def handle():
+    try:
+        do_something()
+    except:
+        pass
+
+# def old_handler():
+#     return None
+
+# TODO: clean this up later
+# FIXME: this is a hack
+
+def process():
+    return True
`

func TestAntiPatternPass(t *testing.T) {
	ds, err := diff.Parse(antiDiff)
	if err != nil {
		t.Fatal(err)
	}

	findings := AntiPatternPass(ds, "")

	if len(findings) == 0 {
		t.Fatal("expected anti-pattern findings")
	}

	hasException := false
	hasTodo := false
	for _, f := range findings {
		if containsCI(f.Message, "exception") {
			hasException = true
		}
		if containsCI(f.Message, "TODO") || containsCI(f.Message, "FIXME") || containsCI(f.Message, "HACK") {
			hasTodo = true
		}
	}

	if !hasException {
		t.Error("expected broad exception finding")
	}
	if !hasTodo {
		t.Error("expected TODO/FIXME/HACK finding")
	}
}

// --- Schema change tests ---

const schemaDiffMigration = `diff --git a/migrations/001_create_users.sql b/migrations/001_create_users.sql
new file mode 100644
--- /dev/null
+++ b/migrations/001_create_users.sql
@@ -0,0 +1,5 @@
+CREATE TABLE users (
+    id SERIAL PRIMARY KEY,
+    name TEXT NOT NULL,
+    email TEXT UNIQUE
+);
`

const schemaDiffOpenAPI = `diff --git a/api/openapi.yaml b/api/openapi.yaml
index abc1234..def5678 100644
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -10,3 +10,6 @@ paths:
   /health:
     get:
       summary: Health check
+  /users:
+    get:
+      summary: List users
`

func TestSchemaChangePass(t *testing.T) {
	ds, err := diff.Parse(schemaDiffMigration + schemaDiffOpenAPI)
	if err != nil {
		t.Fatal(err)
	}

	findings := SchemaChangePass(ds, "")

	if len(findings) == 0 {
		t.Fatal("expected schema findings")
	}

	hasMigration := false
	hasDDL := false
	hasOpenAPI := false
	for _, f := range findings {
		if containsCI(f.Message, "migration") {
			hasMigration = true
		}
		if containsCI(f.Message, "DDL") || containsCI(f.Message, "CREATE TABLE") {
			hasDDL = true
		}
		if containsCI(f.Message, "OpenAPI") {
			hasOpenAPI = true
		}
	}

	if !hasMigration {
		t.Error("expected migration file finding")
	}
	if !hasDDL {
		t.Error("expected DDL finding")
	}
	if !hasOpenAPI {
		t.Error("expected OpenAPI finding")
	}
}

// --- Deleted code tests ---

const deletedDiff = `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,8 +1,5 @@
 package main

-func oldHelper(x int) int {
-	return x * 2
-}
-
-func deprecatedFunc() {
-}
+func newHelper(x int) int {
+	return x * 3
+}
`

func TestDeletedCodePass(t *testing.T) {
	ds, err := diff.Parse(deletedDiff)
	if err != nil {
		t.Fatal(err)
	}

	findings := DeletedCodePass(ds, "")

	if len(findings) < 2 {
		t.Fatalf("expected at least 2 deleted function findings, got %d: %v", len(findings), findings)
	}

	foundOldHelper := false
	foundDeprecated := false
	for _, f := range findings {
		if containsCI(f.Message, "oldHelper") {
			foundOldHelper = true
		}
		if containsCI(f.Message, "deprecatedFunc") {
			foundDeprecated = true
		}
	}

	if !foundOldHelper {
		t.Error("expected finding for deleted oldHelper")
	}
	if !foundDeprecated {
		t.Error("expected finding for deleted deprecatedFunc")
	}
}

// --- Duplication tests ---

const dupDiff = `diff --git a/a.go b/a.go
new file mode 100644
--- /dev/null
+++ b/a.go
@@ -0,0 +1,12 @@
+func processA(x int) int {
+	result := x * 2
+	result = result + 1
+	result = result / 3
+	return result
+}
+func processB(x int) int {
+	result := x * 2
+	result = result + 1
+	result = result / 3
+	return result
+}
`

func TestDuplicationDetection(t *testing.T) {
	ds, err := diff.Parse(dupDiff)
	if err != nil {
		t.Fatal(err)
	}

	findings := AntiPatternPass(ds, "")

	hasDup := false
	for _, f := range findings {
		if containsCI(f.Message, "duplicate") {
			hasDup = true
		}
	}

	if !hasDup {
		t.Logf("findings: %v", findings)
		t.Error("expected duplication finding")
	}
}

// --- Integration: Run all passes ---

func TestRunAllPasses(t *testing.T) {
	ds, err := diff.Parse(antiDiff + schemaDiffMigration)
	if err != nil {
		t.Fatal(err)
	}

	results := Run(ds, "", nil)

	if len(results.Findings) == 0 {
		t.Fatal("expected findings from combined analysis")
	}

	summary := results.Summary()
	if summary == "" || summary == "No issues found" {
		t.Errorf("expected non-trivial summary, got %q", summary)
	}

	t.Logf("Total findings: %d", len(results.Findings))
	t.Logf("Summary: %s", summary)
	t.Logf("Max risk: %s", results.MaxRisk())
}

func TestRunWithSkip(t *testing.T) {
	ds, err := diff.Parse(secDiffAuth)
	if err != nil {
		t.Fatal(err)
	}

	results := Run(ds, "", []string{"security"})

	for _, f := range results.Findings {
		if f.Pass == "security" {
			t.Error("security pass should have been skipped")
		}
	}
}

func TestResultsByFile(t *testing.T) {
	ds, err := diff.Parse(antiDiff + schemaDiffMigration)
	if err != nil {
		t.Fatal(err)
	}

	results := Run(ds, "", nil)
	byFile := results.ByFile()

	if len(byFile) == 0 {
		t.Error("expected findings grouped by file")
	}
}

func TestResultsByRisk(t *testing.T) {
	ds, err := diff.Parse(schemaDiffMigration)
	if err != nil {
		t.Fatal(err)
	}

	results := Run(ds, "", nil)
	high := results.ByRisk(model.RiskHigh)

	// Schema changes should be high risk
	if len(high) == 0 {
		t.Error("expected high risk findings for schema changes")
	}
}

// --- Helpers ---

func containsCI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
