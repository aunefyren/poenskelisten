package utilities

import (
	"fmt"
	"regexp"
)

var DBTrue = true

func PrintASCII() {
	fmt.Println(``)
	fmt.Println(`P Ø N S K E L I S T E N`)
	fmt.Println(``)
	return
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
