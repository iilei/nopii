package recognizer_test

import (
	"strings"
	"testing"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/pseudonym"
	recognizerpkg "github.com/iilei/nopii/internal/recognizer"
)

const (
	janeDoe    = "Jane Doe"
	jonEmail   = "jon@example.com"
	aliceName  = "O'Neil, Alice"
	aliceEmail = "alice@example.com"
	janeEmail  = "jane@example.com"
)

func assertScrubsList(
	t *testing.T,
	e *recognizerpkg.Engine,
	input string,
	plain []string,
	label string,
	count int,
	header string,
) {
	t.Helper()
	out := e.ScrubString(input)
	for _, value := range plain {
		if strings.Contains(out, value) {
			t.Fatalf("%s remained: %q", label, out)
		}
	}
	if got := strings.Count(out, label+"_"); got != count {
		t.Fatalf("expected %d %s replacements, got %d in %q", count, label, got, out)
	}
	if !strings.Contains(out, header) {
		t.Fatalf("missing %q in %q", header, out)
	}
}

func newTestEngine() *recognizerpkg.Engine {
	cfg := config.Defaults()
	gen := pseudonym.New([]byte("secret"), "scope", 12)
	return recognizerpkg.New(&cfg, gen)
}

func TestScrubEmailAndIP(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "scope", 12)
	e := recognizerpkg.New(&cfg, g)
	out := e.ScrubString("alice@example.com from 10.20.30.40")
	if strings.Contains(out, "alice@example.com") || strings.Contains(out, "10.20.30.40") {
		t.Fatalf("PII remained: %q", out)
	}
	if !strings.Contains(out, "EMAIL_") || !strings.Contains(out, "IP_") {
		t.Fatalf("missing replacements: %q", out)
	}
}

func TestScrubCustomClassifier(t *testing.T) {
	cfg := config.Defaults()
	cfg.Classifiers = map[string]config.ClassifierConfig{
		"username": {Label: "USER", Pattern: `(?m)(?:^|[[:space:]])@([A-Za-z0-9_-]+)`},
	}
	g := pseudonym.New([]byte("secret"), "scope", 12)
	e := recognizerpkg.New(&cfg, g)
	out := e.ScrubString("owner @alice-simpson was here")
	if strings.Contains(out, "alice-simpson") {
		t.Fatalf("custom username remained: %q", out)
	}
	if !strings.Contains(out, "USER_") {
		t.Fatalf("missing custom replacement: %q", out)
	}
}

func TestScrubGitMentionDefaults(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "scope", 12)
	e := recognizerpkg.New(&cfg, g)
	out := e.ScrubString("Co-authored-by: Alice Example <alice@example.com>\n@bob wrote this\n")
	if strings.Contains(out, "Alice Example") || strings.Contains(out, "alice@example.com") ||
		strings.Contains(out, "@bob") {
		t.Fatalf("git mention remained: %q", out)
	}
	if !strings.Contains(out, "GIT_MENTION_") {
		t.Fatalf("missing git mention replacement: %q", out)
	}
}

func TestScrubGitMentionMultipleRecipients(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "scope", 12)
	e := recognizerpkg.New(&cfg, g)
	out := e.ScrubString(
		"Co-authored-by: " + janeDoe + " <jane@example.com>, Jon Doe <" + jonEmail + ">, " + aliceName + " <" + aliceEmail + ">\n",
	)
	for _, plain := range []string{janeDoe, jonEmail, aliceName, aliceEmail, janeEmail} {
		if strings.Contains(out, plain) {
			t.Fatalf("git mention remained: %q", out)
		}
	}
	if !strings.Contains(out, "Co-authored-by:") || !strings.Contains(out, "GIT_MENTION_") {
		t.Fatalf("missing git mention replacements: %q", out)
	}
	if count := strings.Count(out, "GIT_MENTION_"); count != 3 {
		t.Fatalf("expected 3 git mention replacements, got %d in %q", count, out)
	}
	if !strings.Contains(out, ", ") {
		t.Fatalf("expected comma-separated recipients to be preserved: %q", out)
	}
}

func TestScrubGitMentionAlternateSeparators(t *testing.T) {
	e := newTestEngine()
	assertScrubsList(
		t,
		e,
		"Co-authored-by: Jane Doe <jane@example.com> and Jon Doe <jon@example.com> / Alice Smith <alice@example.com> + Bob Doe <bob@example.com> & Carol Jones <carol@example.com>; Dan Roe <dan@example.com>\n",
		[]string{"Jane Doe", "jon@example.com", "Alice Smith", "bob@example.com", "Carol Jones", "dan@example.com"},
		"GIT_MENTION",
		6,
		"Co-authored-by:",
	)
}

func TestScrubGitTicketDefaults(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "scope", 12)
	e := recognizerpkg.New(&cfg, g)
	out := e.ScrubString("Fixes: #123\nCloses: GH-456\n")
	if strings.Contains(out, "#123") || strings.Contains(out, "GH-456") {
		t.Fatalf("git ticket remained: %q", out)
	}
	if !strings.Contains(out, "GIT_TICKET_") {
		t.Fatalf("missing git ticket replacement: %q", out)
	}
}

func TestScrubGitTicketMultipleValues(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "scope", 12)
	e := recognizerpkg.New(&cfg, g)
	out := e.ScrubString("Fixes: GH-123, JIRA-0815\n")
	if strings.Contains(out, "GH-123") || strings.Contains(out, "JIRA-0815") {
		t.Fatalf("git ticket list remained: %q", out)
	}
	if count := strings.Count(out, "GIT_TICKET_"); count != 2 {
		t.Fatalf("expected 2 git ticket replacements, got %d in %q", count, out)
	}
	if !strings.Contains(out, "Fixes: ") || !strings.Contains(out, ", ") {
		t.Fatalf("expected ticket list formatting to remain: %q", out)
	}
}

func TestScrubGitTicketAlternateSeparators(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "scope", 12)
	e := recognizerpkg.New(&cfg, g)
	assertScrubsList(t, e, "Refs: #123 and GH-456 / JIRA-0815 + 98765 & SYS-42; 1000\n",
		[]string{"#123", "GH-456", "JIRA-0815", "98765", "SYS-42", "1000"},
		"GIT_TICKET", 6, "Refs: ")
}
