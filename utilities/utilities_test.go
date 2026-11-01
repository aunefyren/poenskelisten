package utilities

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPrintASCII(t *testing.T) {
	// Just confirm it doesn't panic; it only writes to stdout.
	PrintASCII()
}

func TestValidatePasswordFormat(t *testing.T) {
	cases := []struct {
		name     string
		password string
		want     bool
	}{
		{"valid", "Password1", true},
		{"too short", "Pass1", false},
		{"no uppercase", "password1", false},
		{"no lowercase", "PASSWORD1", false},
		{"no digit", "Password", false},
		{"empty", "", false},
		{"norwegian letters count as valid case classes", "Æøåpassord1", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ok, requirements, err := ValidatePasswordFormat(c.password)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != c.want {
				t.Errorf("ValidatePasswordFormat(%q) = %v, want %v", c.password, ok, c.want)
			}
			if requirements == "" {
				t.Error("expected a non-empty requirements message")
			}
		})
	}
}

func TestValidateTextCharacters(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string is allowed", "", true},
		{"plain text", "Hello, world!", true},
		{"less than", "a < b", false},
		{"greater than", "a > b", false},
		{"double quote", `say "hi"`, false},
		{"backtick", "`code`", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ok, requirements, err := ValidateTextCharacters(c.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != c.want {
				t.Errorf("ValidateTextCharacters(%q) = %v, want %v", c.input, ok, c.want)
			}
			if requirements == "" {
				t.Error("expected a non-empty requirements message")
			}
		})
	}
}

func TestCleanConsoleEmail(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"plain", "user@example.com", "user@example.com", false},
		{"surrounding whitespace", "  user@example.com\t\n", "user@example.com", false},
		{"double quotes", `"user@example.com"`, "user@example.com", false},
		{"single quotes and spaces", " 'user@example.com' ", "user@example.com", false},
		{"case preserved", "User@Example.com", "User@Example.com", false},
		{"empty", "   ", "", true},
		{"empty quotes", `""`, "", true},
		{"embedded newline", "user@example.com\nforged log line", "", true},
		{"embedded space", "user @example.com", "", true},
		{"control character", "user@exa\x00mple.com", "", true},
		{"no at sign", "user.example.com", "", true},
		{"display name form", "User <user@example.com>", "", true},
		{"too long", strings.Repeat("a", 250) + "@example.com", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CleanConsoleEmail(c.input)
			if (err != nil) != c.wantErr {
				t.Fatalf("CleanConsoleEmail(%q) err = %v, wantErr %v", c.input, err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("CleanConsoleEmail(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	id := uuid.New()
	cases := []struct {
		name    string
		input   string
		want    uuid.UUID
		wantErr error
	}{
		{"valid", id.String(), id, nil},
		{"nil", "00000000-0000-0000-0000-000000000000", uuid.Nil, ErrNilID},
		{"nil without dashes", "00000000000000000000000000000000", uuid.Nil, ErrNilID},
		{"nil in braces", "{00000000-0000-0000-0000-000000000000}", uuid.Nil, ErrNilID},
		{"nil as URN", "urn:uuid:00000000-0000-0000-0000-000000000000", uuid.Nil, ErrNilID},
		{"empty", "", uuid.Nil, nil},
		{"garbage", "not-a-uuid", uuid.Nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseID(c.input)
			if got != c.want {
				t.Errorf("ParseID(%q) = %v, want %v", c.input, got, c.want)
			}
			if c.want != uuid.Nil {
				if err != nil {
					t.Errorf("ParseID(%q) error = %v", c.input, err)
				}
			} else if err == nil || (c.wantErr != nil && !errors.Is(err, c.wantErr)) {
				t.Errorf("ParseID(%q) error = %v, want %v", c.input, err, c.wantErr)
			}
		})
	}
}

func TestHasNilID(t *testing.T) {
	if HasNilID() || HasNilID(uuid.New(), uuid.New()) {
		t.Error("HasNilID reported a nil UUID among real IDs")
	}
	if !HasNilID(uuid.New(), uuid.Nil) {
		t.Error("HasNilID missed the nil UUID")
	}
}
