package utilities

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

var DBTrue = true

func PrintASCII() {
	fmt.Println(``)
	fmt.Println(`P Ø N S K E L I S T E N`)
	fmt.Println(``)
}

// The validator patterns are compiled once at startup; MustCompile panics on a
// bad pattern, so the validators themselves have no error path to report.
var (
	passwordUpperPattern  = regexp.MustCompile(`[A-ZÆØÅ]`)
	passwordLowerPattern  = regexp.MustCompile(`[a-zæøå]`)
	passwordDigitPattern  = regexp.MustCompile(`[0-9]`)
	textCharactersPattern = regexp.MustCompile(`^[^<>"\x60]+$`)
)

func ValidatePasswordFormat(password string) (bool, string) {
	requirements := "Password must have a minimum of eight characters, at least one uppercase letter, one lowercase letter and one number."

	valid := len(password) >= 8 &&
		passwordUpperPattern.MatchString(password) &&
		passwordLowerPattern.MatchString(password) &&
		passwordDigitPattern.MatchString(password)

	return valid, requirements
}

func ValidateTextCharacters(text string) (bool, string) {
	requirements := `Text must not contain <, >, or ".`

	return text == "" || textCharactersPattern.MatchString(text), requirements
}

// CleanConsoleEmail normalises an e-mail address given as a startup flag or
// environment variable before it is used for a lookup or written to the log.
// It trims surrounding whitespace and quotes, which are easy to paste in by
// accident, and rejects control characters (which could forge log lines) and
// anything that isn't a bare address.
func CleanConsoleEmail(input string) (string, error) {
	email := strings.TrimSpace(strings.Trim(strings.TrimSpace(input), `"'`))
	if email == "" {
		return "", errors.New("e-mail address is empty")
	}
	if len(email) > 254 {
		return "", errors.New("e-mail address is too long")
	}
	for _, r := range email {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", errors.New("e-mail address contains whitespace or control characters")
		}
	}
	// ParseAddress also accepts forms like "Name <a@b>"; requiring the parsed
	// address to equal the input keeps it to a bare address.
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", errors.New("not a valid e-mail address")
	}
	return email, nil
}

// ErrNilID refuses the nil UUID (all zeros) where a request names a record:
// no record has it, and GORM takes it for an unset field in a struct
// condition, which drops the condition.
var ErrNilID = errors.New("the nil UUID is not a valid ID")

// ParseID parses an ID taken from a request, refusing the nil UUID.
func ParseID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, err
	}
	if id == uuid.Nil {
		return uuid.Nil, ErrNilID
	}
	return id, nil
}

// HasNilID reports whether ids holds the nil UUID, which decoding JSON also
// leaves in place of a missing ID.
func HasNilID(ids ...uuid.UUID) bool {
	return slices.Contains(ids, uuid.Nil)
}
