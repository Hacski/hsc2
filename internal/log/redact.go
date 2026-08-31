package log

import (
	"regexp"
	"strings"
	"sync"
)

type Redactor struct {
	mu       sync.RWMutex
	rules    []Rule
	keepTail int
}

type Rule struct {
	Name string
	re   *regexp.Regexp
	repl string
}

func Compiled(name, pattern, repl string) Rule {
	return Rule{Name: name, re: regexp.MustCompile(pattern), repl: repl}
}

func New() *Redactor {
	return &Redactor{
		rules: []Rule{
			Compiled("uriUserPass", `(?i)(https?://)([^/\s:@]+):([^@/\s]+)@`, "$1<USER>:<PASS>@"),
			Compiled("connStringPassword", `(?i)(Password\s*=\s*)([^;]+)`, "$1<REDACTED>"),
			Compiled("password", `(?i)(password|passwd|pwd|passphrase)\s*[=:]\s*([^\s,;]+)`, "<$1:REDACTED>"),
			Compiled("bearer", `(?i)\bbearer\s+([A-Za-z0-9._~+/=-]+)`, "<BEARER:REDACTED>"),
			Compiled("token", `(?i)\b(token|apikey|api_key|access_key|secret_key|client_secret)\b\s*[=:]\s*([^\s,;]+)`, "<$1:REDACTED>"),
			Compiled("aws", `\b(AKIA[0-9A-Z]{16})\b`, "<AWS-ACCESS-KEY:REDACTED>"),
			Compiled("awsSecret", `\b(?i:aws_secret_access_key)\s*[=:]\s*(\S+)`, "<AWS-SECRET:REDACTED>"),
			Compiled("github", `\b(ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{20,})\b`, "<GITHUB-TOKEN:REDACTED>"),
			Compiled("privateKey", `(?s)-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----.*?-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`, "<PRIVATE-KEY:REDACTED>"),
			Compiled("email", `\b[\w.+-]+@[\w-]+\.[\w.]+\b`, "<EMAIL:REDACTED>"),
			Compiled("ssn", `\b\d{3}-\d{2}-\d{4}\b`, "<SSN:REDACTED>"),
			Compiled("creditCard", `\b(?:\d[ -]?){13,19}\b`, "<CARD:REDACTED>"),
		},
	}
}

func (r *Redactor) KeepTail(n int) *Redactor {
	r.keepTail = n
	return r
}

func (r *Redactor) Redact(s string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rule := range r.rules {
		s = rule.re.ReplaceAllString(s, rule.repl)
	}
	return s
}

func (r *Redactor) RedactBytes(b []byte) []byte {
	return []byte(r.Redact(string(b)))
}

func (r *Redactor) AddRule(rule Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
}

func (r *Redactor) Mask(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	n := r.keepTail
	if n > len(s)-4 {
		n = len(s) - 4
	}
	if n < 0 {
		n = 0
	}
	return strings.Repeat("*", len(s)-n) + s[len(s)-n:]
}
