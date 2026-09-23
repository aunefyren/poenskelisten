package utilities

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

var DBTrue = true

func PrintASCII() {
	fmt.Println(``)
	fmt.Println(`P Ø N S K E L I S T E N`)
	fmt.Println(``)
}

func ValidatePasswordFormat(password string) (bool, string, error) {
	requirements := "Password must have a minimum of eight characters, at least one uppercase letter, one lowercase letter and one number."

	if len(password) < 8 {
		return false, requirements, nil
	}

	match, err := regexp.Match(`[A-ZÆØÅ]{1,20}`, []byte(password))
	if err != nil {
		return false, requirements, err
	} else if !match {
		return false, requirements, nil
	}

	match, err = regexp.Match(`[a-zæøå]{1,20}`, []byte(password))
	if err != nil {
		return false, requirements, err
	} else if !match {
		return false, requirements, nil
	}

	match, err = regexp.Match(`[0-9]{1,20}`, []byte(password))
	if err != nil {
		return false, requirements, err
	} else if !match {
		return false, requirements, nil
	}

	return true, requirements, nil
}

func ValidateTextCharacters(string string) (bool, string, error) {
	requirements := `Text must not contain <, >, or ".`

	if string == "" {
		return true, requirements, nil
	}

	match, err := regexp.Match(`^[^<>"\x60]+$`, []byte(string))
	if err != nil {
		return false, requirements, err
	} else if !match {
		return false, requirements, nil
	}

	return true, requirements, nil

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
