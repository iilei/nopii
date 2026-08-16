package stream

import "testing"

func TestParseMailIdentityValidVariants(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantEmail string
	}{
		{
			name:      "quoted name",
			input:     "\"Alice Example\" <alice@example.com>",
			wantName:  "Alice Example",
			wantEmail: "alice@example.com",
		},
		{
			name:      "unquoted name",
			input:     "Alice Example <alice@example.com>",
			wantName:  "Alice Example",
			wantEmail: "alice@example.com",
		},
		{
			name:      "plus addressing",
			input:     "\"Jane Doe\" <jane+tag@example.co.uk>",
			wantName:  "Jane Doe",
			wantEmail: "jane+tag@example.co.uk",
		},
		{
			name:      "punctuation in name",
			input:     "\"O'Neil, Alice\" <alice@example.com>",
			wantName:  "O'Neil, Alice",
			wantEmail: "alice@example.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotName, gotEmail, err := parseMailIdentity(tc.input)
			if err != nil {
				t.Fatalf("parseMailIdentity(%q) returned error: %v", tc.input, err)
			}
			if gotName != tc.wantName {
				t.Fatalf("parseMailIdentity(%q) name = %q, want %q", tc.input, gotName, tc.wantName)
			}
			if gotEmail != tc.wantEmail {
				t.Fatalf("parseMailIdentity(%q) email = %q, want %q", tc.input, gotEmail, tc.wantEmail)
			}
		})
	}
}

func TestParseMailIdentityInvalidVariants(t *testing.T) {
	tests := []string{
		"Alice Example",
		"Alice Example <>",
		"<alice@example.com>",
		"\"\" <alice@example.com>",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, _, err := parseMailIdentity(input); err == nil {
				t.Fatalf("parseMailIdentity(%q) unexpectedly accepted invalid identity", input)
			}
		})
	}
}
