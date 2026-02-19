package analysis

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/sprite-ai/agrev/internal/diff"
	"github.com/sprite-ai/agrev/internal/model"
)

// Schema/migration file patterns.
var schemaPatterns = []struct {
	pattern     *regexp.Regexp
	description string
}{
	{regexp.MustCompile(`(?i)migrat`), "database migration"},
	{regexp.MustCompile(`(?i)schema`), "schema definition"},
	{regexp.MustCompile(`\.proto$`), "protobuf definition"},
	{regexp.MustCompile(`(?i)(openapi|swagger)\.(ya?ml|json)$`), "OpenAPI spec"},
	{regexp.MustCompile(`(?i)graphql$`), "GraphQL schema"},
	{regexp.MustCompile(`\.prisma$`), "Prisma schema"},
	{regexp.MustCompile(`(?i)alembic.*\.py$`), "Alembic migration"},
	{regexp.MustCompile(`(?i)flyway`), "Flyway migration"},
	{regexp.MustCompile(`(?i)knex.*migrat`), "Knex migration"},
	{regexp.MustCompile(`(?i)sequel.*migrat`), "Sequel migration"},
	{regexp.MustCompile(`(?i)active_record.*migrat`), "ActiveRecord migration"},
	{regexp.MustCompile(`(?i)ecto.*migrat`), "Ecto migration"},
}

// SQL DDL keywords in added lines.
var ddlPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(CREATE|ALTER|DROP)\s+(TABLE|INDEX|VIEW|SCHEMA|DATABASE|TYPE|SEQUENCE)\b`),
	regexp.MustCompile(`(?i)\bADD\s+COLUMN\b`),
	regexp.MustCompile(`(?i)\bDROP\s+COLUMN\b`),
	regexp.MustCompile(`(?i)\bRENAME\s+(TABLE|COLUMN)\b`),
	regexp.MustCompile(`(?i)\bMODIFY\s+COLUMN\b`),
}

// SchemaChangePass detects changes to database schemas, migrations, and API specs.
func SchemaChangePass(ds *diff.DiffSet, repoDir string) []Finding {
	var findings []Finding

	for _, f := range ds.Files {
		name := f.Name()

		// Check if the file itself is a schema/migration file
		for _, sp := range schemaPatterns {
			if sp.pattern.MatchString(name) {
				risk := model.RiskHigh
				findings = append(findings, Finding{
					Pass:     "schema",
					File:     name,
					Message:  fmt.Sprintf("Changes to %s file", sp.description),
					Severity: model.SeverityWarning,
					Risk:     risk,
				})
				break
			}
		}

		// Check for DDL statements in added lines
		findings = append(findings, checkDDL(f)...)
	}

	return findings
}

func checkDDL(f *diff.File) []Finding {
	var findings []Finding
	name := f.Name()

	for _, frag := range f.Fragments {
		lineNum := int(frag.NewPosition)
		for _, line := range frag.Lines {
			if line.Op == gitdiff.OpAdd {
				text := line.Line
				for _, pat := range ddlPatterns {
					if pat.MatchString(text) {
						findings = append(findings, Finding{
							Pass:     "schema",
							File:     name,
							Line:     lineNum,
							Message:  fmt.Sprintf("DDL statement: %s", strings.TrimSpace(text)),
							Severity: model.SeverityWarning,
							Risk:     model.RiskHigh,
						})
						break
					}
				}
			}
			if line.Op == gitdiff.OpAdd || line.Op == gitdiff.OpContext {
				lineNum++
			}
		}
	}

	return findings
}
