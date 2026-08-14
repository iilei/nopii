package recognizer

import (
	"regexp"
	"sort"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/pseudonym"
)

type match struct {
	start, end int
	typ, value string
}
type Engine struct {
	rules []rule
	gen   *pseudonym.Generator
}
type rule struct {
	typ string
	re  *regexp.Regexp
}

func New(cfg config.Config, gen *pseudonym.Generator) *Engine {
	var rules []rule
	if cfg.Recognizers.Email {
		rules = append(rules, rule{"EMAIL", regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)})
	}
	if cfg.Recognizers.UUID {
		rules = append(rules, rule{"UUID", regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`)})
	}
	if cfg.Recognizers.IPv4 {
		rules = append(rules, rule{"IP", regexp.MustCompile(`\b(?:25[0-5]|2[0-4]\d|1?\d?\d)(?:\.(?:25[0-5]|2[0-4]\d|1?\d?\d)){3}\b`)})
	}
	if cfg.Recognizers.Phone {
		rules = append(rules, rule{"PHONE", regexp.MustCompile(`(?m)(?:\+?\d[\d .()\-/]{6,}\d)`)})
	}
	return &Engine{rules: rules, gen: gen}
}

func (e *Engine) ScrubString(s string) string {
	var ms []match
	for _, r := range e.rules {
		for _, idx := range r.re.FindAllStringIndex(s, -1) {
			ms = append(ms, match{idx[0], idx[1], r.typ, s[idx[0]:idx[1]]})
		}
	}
	if len(ms) == 0 {
		return s
	}
	sort.Slice(ms, func(i, j int) bool {
		if ms[i].start == ms[j].start {
			return ms[i].end > ms[j].end
		}
		return ms[i].start < ms[j].start
	})
	filtered := ms[:0]
	last := -1
	for _, m := range ms {
		if m.start >= last {
			filtered = append(filtered, m)
			last = m.end
		}
	}
	out := make([]byte, 0, len(s))
	pos := 0
	for _, m := range filtered {
		out = append(out, s[pos:m.start]...)
		out = append(out, e.gen.Replacement(m.typ, m.value)...)
		pos = m.end
	}
	out = append(out, s[pos:]...)
	return string(out)
}
