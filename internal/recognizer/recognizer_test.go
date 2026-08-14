package recognizer

import (
	"strings"
	"testing"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/pseudonym"
)

func TestScrubEmailAndIP(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "scope", 12)
	e := New(cfg, g)
	out := e.ScrubString("alice@example.com from 10.20.30.40")
	if strings.Contains(out, "alice@example.com") || strings.Contains(out, "10.20.30.40") {
		t.Fatalf("PII remained: %q", out)
	}
	if !strings.Contains(out, "EMAIL_") || !strings.Contains(out, "IP_") {
		t.Fatalf("missing replacements: %q", out)
	}
}
