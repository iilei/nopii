package recognizer_test

import (
	"strings"
	"testing"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/pseudonym"
	recognizerpkg "github.com/iilei/nopii/internal/recognizer"
)

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
		"Co-authored-by: Jane Doe <jane@example.com>, Jon Doe <jon@example.com>, O'Neil, Alice <alice@example.com>\n",
	)
	for _, plain := range []string{"Jane Doe", "jon@example.com", "O'Neil, Alice", "alice@example.com", "jane@example.com"} {
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
