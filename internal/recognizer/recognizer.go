// Package recognizer detects and scrubs PII from text using structured rules.
package recognizer

import (
	"regexp"
	"sort"
	"strings"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/pseudonym"
)

const (
	gitMentionTrailerDefaultPattern  = `(?im)(?:^(?:(?:Co|Signed|Reviewed|Acked|Tested|Helped|Reported|Mentored)[ -]?authored[ -]?by|(?:Co|Signed|Reviewed|Acked|Tested|Helped|Reported|Mentored)[ -]?by|With[ -]?help[ -]?from|Collaborated[ -]?with)\s*:\s*|(?:\s*(?:,|/|\+|&|;|\band\b)\s*))("?[^"<\n]+"?\s*<[^>\n]+>)`
	gitMentionUsernameDefaultPattern = `(?m)(?:^|[^A-Za-z0-9_])(@[A-Za-z0-9_-]+)`
	gitTicketDefaultPattern          = `(?im)^(?:Fixes|Closes|Resolves|Refs|References|See|Ticket|Issue)\s*:\s*(?:#\d+|#?[A-Z]+[-_ ]?\d+|[A-Z][A-Z0-9]+-\d+|(?:\d{2,5}[_-])?\d+)(?:\s*(?:,|/|\+|&|;|\band\b|\s+)\s*(?:#\d+|#?[A-Z]+[-_ ]?\d+|[A-Z][A-Z0-9]+-\d+|(?:\d{2,5}[_-])?\d+))*`
)

type (
	match struct {
		start, end int
		typ, value string
		priority   int
	}
	Engine struct {
		rules []rule
		gen   *pseudonym.Generator
	}
	rule struct {
		typ      string
		re       *regexp.Regexp
		priority int
	}
)

func classifyLabel(label string) string {
	return strings.ToUpper(strings.TrimSpace(label))
}

func New(cfg *config.Config, gen *pseudonym.Generator) *Engine {
	if cfg != nil {
		config.ApplyEnv(cfg)
	}
	var rules []rule
	if cfg == nil {
		return &Engine{gen: gen}
	}
	if cfg.Recognizers.Email {
		rules = append(
			rules,
			rule{typ: "EMAIL", re: regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`), priority: 80},
		)
	}
	if cfg.Recognizers.UUID {
		rules = append(
			rules,
			rule{
				typ: "UUID",
				re: regexp.MustCompile(
					`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`,
				),
				priority: 70,
			},
		)
	}
	if cfg.Recognizers.IPv4 {
		rules = append(
			rules,
			rule{
				typ:      "IP",
				re:       regexp.MustCompile(`\b(?:25[0-5]|2[0-4]\d|1?\d?\d)(?:\.(?:25[0-5]|2[0-4]\d|1?\d?\d)){3}\b`),
				priority: 90,
			},
		)
	}
	if cfg.Recognizers.Phone {
		rules = append(
			rules,
			rule{typ: "PHONE", re: regexp.MustCompile(`(?m)(?:\+?\d[\d .()\-/]{6,}\d)`), priority: 40},
		)
	}
	if _, override := cfg.Classifiers["git_mention"]; !override {
		rules = append(
			rules,
			rule{typ: "GIT_MENTION", re: regexp.MustCompile(gitMentionTrailerDefaultPattern), priority: 90},
			rule{typ: "GIT_MENTION", re: regexp.MustCompile(gitMentionUsernameDefaultPattern), priority: 90},
		)
	}
	if _, override := cfg.Classifiers["git_ticket"]; !override {
		rules = append(rules, rule{typ: "GIT_TICKET", re: regexp.MustCompile(gitTicketDefaultPattern), priority: 90})
	}
	for name, classifier := range cfg.Classifiers {
		pattern := classifier.Pattern
		if pattern == "" {
			if custom, ok := cfg.CustomPatterns[name]; ok {
				pattern = custom
			}
		}
		if pattern == "" {
			continue
		}
		label := classifier.Label
		if strings.TrimSpace(label) == "" {
			label = strings.ToUpper(name)
		}
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		rules = append(rules, rule{typ: classifyLabel(label), re: compiled, priority: 100})
	}
	return &Engine{rules: rules, gen: gen}
}

func redactTicketList(value string, gen *pseudonym.Generator) string {
	var out strings.Builder
	pos := 0
	for _, idx := range regexp.MustCompile(`(?:#\d+|#?[A-Z]+[-_ ]?\d+|[A-Z][A-Z0-9]+-\d+|(?:\d{2,5}[_-])?\d+)`).FindAllStringSubmatchIndex(value, -1) {
		start, end := idx[0], idx[1]
		out.WriteString(value[pos:start])
		out.WriteString(gen.Replacement("GIT_TICKET", value[start:end]))
		pos = end
	}
	out.WriteString(value[pos:])
	return out.String()
}

func (e *Engine) ScrubString(s string) string {
	var ms []match
	for _, r := range e.rules {
		for _, idx := range r.re.FindAllStringSubmatchIndex(s, -1) {
			start, end := idx[0], idx[1]
			if len(idx) >= 4 && idx[2] >= 0 && idx[3] > idx[2] {
				start, end = idx[2], idx[3]
			}
			ms = append(
				ms,
				match{start: start, end: end, typ: r.typ, value: s[start:end], priority: r.priority},
			)
		}
	}
	if len(ms) == 0 {
		return s
	}
	sort.Slice(ms, func(i, j int) bool {
		if ms[i].start == ms[j].start {
			if ms[i].end == ms[j].end {
				if ms[i].priority == ms[j].priority {
					return ms[i].typ > ms[j].typ
				}
				return ms[i].priority > ms[j].priority
			}
			return ms[i].end > ms[j].end
		}
		return ms[i].start < ms[j].start
	})
	filtered := make([]match, 0, len(ms))
	for _, m := range ms {
		if len(filtered) == 0 {
			filtered = append(filtered, m)
			continue
		}
		prev := &filtered[len(filtered)-1]
		if m.start >= prev.end {
			filtered = append(filtered, m)
			continue
		}
		if m.priority > prev.priority ||
			(m.priority == prev.priority && (m.end-m.start) >= (prev.end-prev.start)) ||
			(m.priority == prev.priority && (m.end-m.start) == (prev.end-prev.start) && m.end > prev.end) {
			*prev = m
		}
	}
	out := make([]byte, 0, len(s))
	pos := 0
	for _, m := range filtered {
		out = append(out, s[pos:m.start]...)
		if m.typ == "GIT_TICKET" && strings.Contains(m.value, ":") {
			out = append(out, redactTicketList(m.value, e.gen)...)
		} else {
			out = append(out, e.gen.Replacement(m.typ, m.value)...)
		}
		pos = m.end
	}
	out = append(out, s[pos:]...)
	return string(out)
}
