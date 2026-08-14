package stream

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/pseudonym"
	"github.com/iilei/nopii/internal/recognizer"
)

func TestGitV1(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "repo", 12)
	r := recognizer.New(cfg, g)
	p := New(g, r)
	input := strings.Join(
		[]string{
			GitMagic,
			"abc123",
			"parent",
			"Alice Example",
			"alice@example.com",
			"Bob Example",
			"bob@example.com",
			"100",
			"101",
			"Fix requested by carol@example.com",
		},
		"\x1f",
	) + "\x00"
	var out bytes.Buffer
	if err := p.Process(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, plain := range []string{"Alice Example", "alice@example.com", "Bob Example", "bob@example.com", "carol@example.com"} {
		if strings.Contains(s, plain) {
			t.Fatalf("plain PII %q remained in %q", plain, s)
		}
	}
	if !strings.Contains(s, "commit abc123") || !strings.Contains(s, "PERSON_") {
		t.Fatalf("unexpected output: %q", s)
	}
}
