package stream_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iilei/nopii/internal/config"
	"github.com/iilei/nopii/internal/pseudonym"
	"github.com/iilei/nopii/internal/recognizer"
	streampkg "github.com/iilei/nopii/internal/stream"
)

const (
	aliceExample  = "Alice Example"
	aliceEmail    = "alice@example.com"
	bobExample    = "Bob Example"
	bobEmail      = "bob@example.com"
	carolEmail    = "carol@example.com"
	gitCommitHash = "abc123"
)

func TestGitV1(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "repo", 12)
	r := recognizer.New(&cfg, g)
	p := streampkg.New(g, r, config.DateClampConfig{})
	input := strings.Join(
		[]string{
			streampkg.GitMagic,
			gitCommitHash,
			"parent",
			aliceExample,
			aliceEmail,
			bobExample,
			bobEmail,
			"100",
			"101",
			"Fix requested by " + carolEmail,
		},
		"\x1f",
	) + "\x00"
	var out bytes.Buffer
	if err := p.Process(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, plain := range []string{aliceExample, aliceEmail, bobExample, bobEmail, carolEmail} {
		if strings.Contains(s, plain) {
			t.Fatalf("plain PII %q remained in %q", plain, s)
		}
	}
	if !strings.Contains(s, "commit "+gitCommitHash) || !strings.Contains(s, "PERSON_") {
		t.Fatalf("unexpected output: %q", s)
	}
}
