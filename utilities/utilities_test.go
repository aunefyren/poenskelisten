package utilities

import "testing"

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
