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
