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

func processIdentity(t *testing.T, author string) error {
	t.Helper()
	cfg := config.Defaults()
	gen := pseudonym.New([]byte("secret"), "scope", 12)
	rec := recognizer.New(&cfg, gen)
	processor := streampkg.New(gen, rec, config.DateClampConfig{})
	input := strings.Join([]string{
		streampkg.GitMagic,
		"hash",
		"",
		author,
		`"Committer" <committer@example.com>`,
		"100",
		"101",
		"message\n",
	}, "\x1f") + "\x00"
	var out bytes.Buffer
	return processor.Process(strings.NewReader(input), &out)
}

func TestParseMailIdentityValidVariants(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "quoted name",
			input: `"` + aliceExample + `" <` + aliceEmail + `>`,
		},
		{
			name:  "unquoted name",
			input: aliceExample + " <" + aliceEmail + ">",
		},
		{
			name:  "plus addressing",
			input: `"Jane Doe" <jane+tag@example.co.uk>`,
		},
		{
			name:  "punctuation in name",
			input: `"O'Neil, Alice" <alice@example.com>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := processIdentity(t, tc.input); err != nil {
				t.Fatalf("valid identity %q was rejected: %v", tc.input, err)
			}
		})
	}
}

func TestParseMailIdentityInvalidVariants(t *testing.T) {
	tests := []string{
		aliceExample,
		aliceExample + " <>",
		"<alice@example.com>",
		"\"\" <alice@example.com>",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if err := processIdentity(t, input); err == nil {
				t.Fatalf("invalid identity %q was unexpectedly accepted", input)
			}
		})
	}
}
