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
	gitParent     = "parent"
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
			gitParent,
			"\"" + aliceExample + "\" <" + aliceEmail + ">",
			"\"" + bobExample + "\" <" + bobEmail + ">",
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
	if !strings.Contains(s, "Author: \"PERSON_") || !strings.Contains(s, "Committer: \"PERSON_") {
		t.Fatalf("expected quoted display-name format in output: %q", s)
	}
}

func TestGitV1ScrubsTrailerBodyOnly(t *testing.T) {
	cfg := config.Defaults()
	g := pseudonym.New([]byte("secret"), "repo", 12)
	r := recognizer.New(&cfg, g)
	p := streampkg.New(g, r, config.DateClampConfig{})
	body := "Fix summary\n\nCo-authored-by: Alice Example <alice@example.com>\nReviewed-by: Bob Example <bob@example.com>\n"
	input := strings.Join(
		[]string{
			streampkg.GitMagic,
			gitCommitHash,
			gitParent,
			"\"" + aliceExample + "\" <" + aliceEmail + ">",
			"\"" + bobExample + "\" <" + bobEmail + ">",
			"100",
			"101",
			body,
		},
		"\x1f",
	) + "\x00"
	var out bytes.Buffer
	if err := p.Process(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, plain := range []string{aliceExample, aliceEmail, bobExample, bobEmail} {
		if strings.Contains(s, plain) {
			t.Fatalf("plain trailer PII %q remained in %q", plain, s)
		}
	}
	if !strings.Contains(s, "GIT_MENTION_") {
		t.Fatalf("expected git mention replacement in output: %q", s)
	}
}
