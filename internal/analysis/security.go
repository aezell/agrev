package analysis

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/sprite-ai/agrev/internal/diff"
	"github.com/sprite-ai/agrev/internal/model"
)

// Security-sensitive patterns grouped by category.
var securityPatterns = []struct {
	category string
	patterns []*regexp.Regexp
	risk     model.RiskLevel
}{
	{
		category: "authentication",
		patterns: compilePatterns(
			`(?i)(auth|login|logout|signin|signup|password|credential|token|jwt|oauth|session|cookie)`,
		),
		risk: model.RiskHigh,
	},
	{
		category: "authorization",
		patterns: compilePatterns(
			`(?i)(permission|role|access.?control|rbac|acl|authorize|forbidden|is.?admin|can.?access)`,
		),
		risk: model.RiskHigh,
	},
	{
		category: "SQL/database",
		patterns: compilePatterns(
			`(?i)(db\.exec|db\.query|\.prepare\(|raw.?sql|sql\.)`,
			`(?i)(\bSELECT\b|\bINSERT\b|\bUPDATE\b|\bDELETE\b|\bDROP\b|\bALTER\b)\s`,
			`(?i)(connection\.execute|cursor\.execute)`,
		),
		risk: model.RiskHigh,
	},
	{
		category: "cryptography",
		patterns: compilePatterns(
			`(?i)(encrypt|decrypt|hash|hmac|cipher|aes|rsa|sha256|sha512|bcrypt|argon|scrypt|pbkdf)`,
			`(?i)(private.?key|public.?key|secret.?key|signing.?key|crypto\.)`,
		),
		risk: model.RiskHigh,
	},
	{
		category: "file system",
		patterns: compilePatterns(
			`(?i)(os\.Remove|os\.Rename|os\.Chmod|os\.Chown|os\.MkdirAll|os\.WriteFile|ioutil\.WriteFile)`,
			`(?i)(unlink|rmdir|chmod|chown|write_file|open.*[\"']w)`,
			`(?i)(path\.join|filepath\.join).*\.\.|\.\.\/`,
		),
		risk: model.RiskMedium,
	},
	{
		category: "environment/secrets",
		patterns: compilePatterns(
			`(?i)(os\.Getenv|os\.environ|process\.env|ENV\[|getenv)`,
			`(?i)(api.?key|secret|password|token)\s*[:=]`,
			`(?i)(PRIVATE|SECRET|PASSWORD|TOKEN|KEY)\s*=\s*["']`,
		),
		risk: model.RiskMedium,
	},
	{
		category: "network/HTTP",
		patterns: compilePatterns(
			`(?i)(http\.ListenAndServe|\.listen\(|cors|origin|allow.?origin)`,
			`(?i)(tls\.Config|InsecureSkipVerify|disable.?ssl|verify.?ssl.*false)`,
		),
		risk: model.RiskMedium,
	},
	{
		category: "subprocess/exec",
		patterns: compilePatterns(
			`(?i)(exec\.Command|os\.system|subprocess|child_process|shell_exec|system\()`,
			`(?i)(eval\(|exec\(|compile\()`,
		),
		risk: model.RiskHigh,
	},
}

func compilePatterns(patterns ...string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return compiled
}

// SecuritySurfacePass flags changes to security-sensitive code.
func SecuritySurfacePass(ds *diff.DiffSet, repoDir string) []Finding {
	var findings []Finding

	for _, f := range ds.Files {
		name := f.Name()

		for _, frag := range f.Fragments {
			lineNum := int(frag.NewPosition)
			for _, line := range frag.Lines {
				if line.Op == gitdiff.OpAdd {
					text := line.Line
					// Skip comment-only lines
					trimmed := strings.TrimSpace(text)
					if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") {
						if line.Op == gitdiff.OpAdd || line.Op == gitdiff.OpContext {
							lineNum++
						}
						continue
					}
					for _, sp := range securityPatterns {
						for _, re := range sp.patterns {
							if re.MatchString(text) {
								findings = append(findings, Finding{
									Pass:     "security",
									File:     name,
									Line:     lineNum,
									Message:  fmt.Sprintf("Security-sensitive change (%s): %s", sp.category, strings.TrimSpace(text)),
									Severity: model.SeverityWarning,
									Risk:     sp.risk,
								})
								break // one finding per pattern group per line
							}
						}
					}
				}
				if line.Op == gitdiff.OpAdd || line.Op == gitdiff.OpContext {
					lineNum++
				}
			}
		}
	}

	return deduplicateFindings(findings)
}

// deduplicateFindings removes findings with the same file+line+message.
func deduplicateFindings(findings []Finding) []Finding {
	seen := make(map[string]bool)
	var result []Finding
	for _, f := range findings {
		key := fmt.Sprintf("%s:%d:%s", f.File, f.Line, f.Message)
		if !seen[key] {
			seen[key] = true
			result = append(result, f)
		}
	}
	return result
}
