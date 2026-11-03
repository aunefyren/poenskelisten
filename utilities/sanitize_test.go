package utilities

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestApplySanitizeRules(t *testing.T) {
	cases := []struct {
		name  string
		input string
		rules []string
		want  string
	}{
		{"trim", "  ada \t\n", []string{"trim"}, "ada"},
		{"lower", "Ada@Example.COM", []string{"lower"}, "ada@example.com"},
		{"trim then lower", "  Ada@Example.COM ", []string{"trim", "lower"}, "ada@example.com"},
		{"capfirst", "ada", []string{"capfirst"}, "Ada"},
		{"capfirst keeps the rest as typed", "van der Berg", []string{"capfirst"}, "Van der Berg"},
		{"capfirst leaves inner capitals", "mcDonald", []string{"capfirst"}, "McDonald"},
		{"capfirst after trim reaches the first letter", "  øystein", []string{"trim", "capfirst"}, "Øystein"},
		{"capfirst before trim sees the space", "  øystein", []string{"capfirst", "trim"}, "øystein"},
		{"text accepts plain text", "Hello, world!", []string{"text"}, "Hello, world!"},
		{"none leaves the value alone", "  Pass Word1 ", []string{"none"}, "  Pass Word1 "},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := applySanitizeRules(c.input, c.rules, "Field")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("applySanitizeRules(%q, %v) = %q, want %q", c.input, c.rules, got, c.want)
			}
		})
	}
}

func TestApplySanitizeRulesRejectsUnsafeText(t *testing.T) {
	for _, input := range []string{"<script>alert(1)</script>", `say "hi"`, "a > b", "`code`"} {
		_, err := applySanitizeRules(input, []string{"trim", "text"}, "Name")
		var textErr *TextValidationError
		if !errors.As(err, &textErr) {
			t.Errorf("applySanitizeRules(%q) error = %v, want a TextValidationError", input, err)
			continue
		}
		if textErr.Field != "Name" || textErr.Requirements == "" {
			t.Errorf("TextValidationError = %+v, want field Name and a requirements message", textErr)
		}
		if textErr.Error() != "field Name failed text validation" {
			t.Errorf("Error() = %q", textErr.Error())
		}
	}
}

func TestCapitalizeFirst(t *testing.T) {
	cases := []struct{ input, want string }{
		{"", ""},
		{"a", "A"},
		{"ada", "Ada"},
		{"Ada", "Ada"},
		{"åse", "Åse"},
		{"1st", "1st"},
		{"\xff", "\xff"}, // invalid UTF-8 is returned untouched
	}
	for _, c := range cases {
		if got := CapitalizeFirst(c.input); got != c.want {
			t.Errorf("CapitalizeFirst(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

type sanitizeInner struct {
	Label string `sanitize:"trim"`
}

type sanitizeEmbedded struct {
	Nickname string `sanitize:"trim,capfirst"`
}

type sanitizeSample struct {
	sanitizeEmbedded
	Email    string         `sanitize:"trim,lower"`
	Password string         `sanitize:"none"`
	Date     *string        `sanitize:"trim"`
	Missing  *string        `sanitize:"trim"`
	URIs     []string       `sanitize:"trim"`
	Groups   *[]string      `sanitize:"trim"`
	Inner    sanitizeInner  // nested struct, no tag of its own
	InnerPtr *sanitizeInner // nested through a pointer
	NilInner *sanitizeInner
	Count    int
	Flag     bool `sanitize:"trim"` // a rule on a non-string field does nothing
	private  string
}

func TestSanitizeRequest(t *testing.T) {
	date := " 2026-01-01 "
	groups := []string{" a ", "b "}
	request := sanitizeSample{
		sanitizeEmbedded: sanitizeEmbedded{Nickname: "  ada"},
		Email:            " Ada@Example.com ",
		Password:         " secret ",
		Date:             &date,
		URIs:             []string{" https://a.example ", "https://b.example\n"},
		Groups:           &groups,
		Inner:            sanitizeInner{Label: " x "},
		InnerPtr:         &sanitizeInner{Label: " y "},
		Count:            3,
		private:          " untouched ",
	}

	if err := SanitizeRequest(&request); err != nil {
		t.Fatalf("SanitizeRequest: %v", err)
	}

	checks := map[string][2]string{
		"Nickname": {request.Nickname, "Ada"},
		"Email":    {request.Email, "ada@example.com"},
		"Password": {request.Password, " secret "},
		"Date":     {*request.Date, "2026-01-01"},
		"Inner":    {request.Inner.Label, "x"},
		"InnerPtr": {request.InnerPtr.Label, "y"},
		"private":  {request.private, " untouched "},
	}
	for name, check := range checks {
		if check[0] != check[1] {
			t.Errorf("%s = %q, want %q", name, check[0], check[1])
		}
	}
	if !slices.Equal(request.URIs, []string{"https://a.example", "https://b.example"}) {
		t.Errorf("URIs = %q", request.URIs)
	}
	if !slices.Equal(*request.Groups, []string{"a", "b"}) {
		t.Errorf("Groups = %q", *request.Groups)
	}
	if request.Missing != nil || request.NilInner != nil {
		t.Error("nil pointers should stay nil")
	}
}

func TestSanitizeRequestErrors(t *testing.T) {
	t.Run("not a pointer", func(t *testing.T) {
		if err := SanitizeRequest(sanitizeSample{}); err == nil {
			t.Error("expected an error for a non-pointer")
		}
	})
	t.Run("pointer to non-struct", func(t *testing.T) {
		text := "x"
		if err := SanitizeRequest(&text); err == nil {
			t.Error("expected an error for a pointer to a string")
		}
	})
	t.Run("unknown rule", func(t *testing.T) {
		request := struct {
			Name string `sanitize:"trim,shout"`
		}{"x"}
		if err := SanitizeRequest(&request); err == nil {
			t.Error("expected an error for an unknown rule")
		}
	})
	t.Run("text failure in a nested struct", func(t *testing.T) {
		request := struct {
			Inner struct {
				Name string `sanitize:"trim,text"`
			}
		}{}
		request.Inner.Name = "<b>"
		var textErr *TextValidationError
		if err := SanitizeRequest(&request); !errors.As(err, &textErr) {
			t.Errorf("error = %v, want a TextValidationError", err)
		}
	})
}

func TestUntaggedStringFields(t *testing.T) {
	if problems := UntaggedStringFields(reflect.TypeOf(sanitizeSample{})); len(problems) != 0 {
		t.Errorf("fully tagged struct reported problems: %v", problems)
	}

	type inner struct {
		Bare string
	}
	type broken struct {
		Name    string
		Note    *string
		Tags    []string
		Typo    string `sanitize:"trim,lowr"`
		Mixed   string `sanitize:"none,trim"`
		Nested  inner
		Pointer *inner
		Fine    string `sanitize:"trim"`
		Number  int
	}
	got := UntaggedStringFields(reflect.TypeOf(&broken{}))
	want := []string{
		"broken.Name: no sanitize tag",
		"broken.Note: no sanitize tag",
		"broken.Tags: no sanitize tag",
		`broken.Typo: unknown sanitize rule "lowr"`,
		`broken.Mixed: sanitize rule "none" can't be combined with others`,
		"broken.Nested.Bare: no sanitize tag",
		"broken.Pointer.Bare: no sanitize tag",
	}
	if !slices.Equal(got, want) {
		t.Errorf("UntaggedStringFields =\n%q\nwant\n%q", got, want)
	}
}
