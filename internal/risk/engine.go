package risk

import (
	"regexp"
	"strings"
)

type Finding struct{ Code, Level, Reason string }
type Result struct {
	Input, Redacted, Level string
	Findings               []Finding
}

func (r Result) Has(code string) bool {
	for _, finding := range r.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

var rules = []struct {
	code, level, reason string
	pattern             *regexp.Regexp
}{
	{"prompt_injection", "high", "instruction override or system prompt extraction", regexp.MustCompile(`(?i)(ignore|disregard)\s+(all\s+)?previous|system\s+prompt|reveal\s+instructions`)},
	{"api_key", "critical", "API key-like secret", regexp.MustCompile(`(?i)\b(sk|ag_live|AKIA)[-_]?[A-Za-z0-9]{10,}`)},
	{"email", "low", "email address", regexp.MustCompile(`\b[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}\b`)},
	{"phone", "low", "phone number", regexp.MustCompile(`(?:^|[^0-9])1[3-9][0-9]{9}(?:$|[^0-9])`)},
}

func Analyze(input string) Result {
	result := Result{Input: input, Redacted: input, Level: "none"}
	for _, rule := range rules {
		if rule.pattern.MatchString(input) {
			result.Findings = append(result.Findings, Finding{Code: rule.code, Level: rule.level, Reason: rule.reason})
			result.Redacted = rule.pattern.ReplaceAllString(result.Redacted, "[REDACTED]")
		}
	}
	for _, finding := range result.Findings {
		if severity(finding.Level) > severity(result.Level) {
			result.Level = finding.Level
		}
	}
	return result
}

func severity(level string) int {
	return map[string]int{"none": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}[strings.ToLower(level)]
}
