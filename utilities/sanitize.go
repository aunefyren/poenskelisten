package utilities

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SanitizeTag is the struct tag that declares how a request field is cleaned
// after JSON binding. Every string field of a bound request struct must carry
// one, so a field can't be added without deciding how it's handled; the
// controllers tests enforce that with UntaggedStringFields.
//
// Rules are comma-separated and applied left to right:
//
//	trim      strip surrounding whitespace
//	lower     lowercase (e-mail addresses)
//	capfirst  uppercase the first letter, leaving the rest as typed (person
//	          names: "van der Berg" and "McDonald" survive, unlike title case)
//	text      reject <, >, " and backtick via ValidateTextCharacters, for
//	          anything rendered back to other users
//	none      leave the value exactly as sent (passwords, tokens, image data)
const SanitizeTag = "sanitize"

// TextValidationError reports a field that failed the "text" rule. It's a
// caller mistake, not an internal failure, so handlers answer it with a 400
// carrying Requirements.
type TextValidationError struct {
	Field        string
	Requirements string
}

func (e *TextValidationError) Error() string {
	return "field " + e.Field + " failed text validation"
}

// SanitizeRequest applies the sanitize tags of the struct request points to,
// in place. It handles string, *string, []string and *[]string fields, and
// recurses into nested and embedded structs. A string field without a tag is
// left alone here; UntaggedStringFields is what catches it, at test time.
func SanitizeRequest(request any) error {
	value := reflect.ValueOf(request)
	if value.Kind() != reflect.Pointer || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("SanitizeRequest needs a pointer to a struct, got %T", request)
	}
	return sanitizeStruct(value.Elem())
}

func sanitizeStruct(value reflect.Value) error {
	structType := value.Type()
	for i := range structType.NumField() {
		field := structType.Field(i)
		// An embedded struct is walked even when its type is unexported: its
		// exported fields are promoted, and JSON binds them.
		if !field.IsExported() && !field.Anonymous {
			continue
		}
		fieldValue := value.Field(i)

		if nested := derefStruct(fieldValue); nested.IsValid() {
			if err := sanitizeStruct(nested); err != nil {
				return err
			}
			continue
		}

		tag, tagged := field.Tag.Lookup(SanitizeTag)
		if !tagged {
			continue
		}
		rules, err := parseSanitizeRules(tag)
		if err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}

		for _, target := range stringTargets(fieldValue) {
			cleaned, err := applySanitizeRules(target.String(), rules, field.Name)
			if err != nil {
				return err
			}
			target.SetString(cleaned)
		}
	}
	return nil
}

// derefStruct returns the struct a field holds (directly or through a non-nil
// pointer), or an invalid Value if it isn't one.
func derefStruct(fieldValue reflect.Value) reflect.Value {
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			return reflect.Value{}
		}
		fieldValue = fieldValue.Elem()
	}
	if fieldValue.Kind() == reflect.Struct {
		return fieldValue
	}
	return reflect.Value{}
}

// stringTargets returns the settable string values a field holds: itself, the
// string behind a pointer, or every element of a string slice.
func stringTargets(fieldValue reflect.Value) []reflect.Value {
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			return nil
		}
		fieldValue = fieldValue.Elem()
	}
	switch {
	case fieldValue.Kind() == reflect.String:
		return []reflect.Value{fieldValue}
	case fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.String:
		targets := make([]reflect.Value, fieldValue.Len())
		for i := range targets {
			targets[i] = fieldValue.Index(i)
		}
		return targets
	}
	return nil
}

var knownSanitizeRules = map[string]bool{"trim": true, "lower": true, "capfirst": true, "text": true, "none": true}

func parseSanitizeRules(tag string) ([]string, error) {
	rules := strings.Split(tag, ",")
	for _, rule := range rules {
		if !knownSanitizeRules[rule] {
			return nil, fmt.Errorf("unknown sanitize rule %q", rule)
		}
		if rule == "none" && len(rules) > 1 {
			return nil, fmt.Errorf("sanitize rule \"none\" can't be combined with others")
		}
	}
	return rules, nil
}

func applySanitizeRules(text string, rules []string, fieldName string) (string, error) {
	for _, rule := range rules {
		switch rule {
		case "trim":
			text = strings.TrimSpace(text)
		case "lower":
			text = strings.ToLower(text)
		case "capfirst":
			text = CapitalizeFirst(text)
		case "text":
			if valid, requirements := ValidateTextCharacters(text); !valid {
				return "", &TextValidationError{Field: fieldName, Requirements: requirements}
			}
		}
	}
	return text, nil
}

// CapitalizeFirst uppercases the first letter of text and leaves the rest
// untouched.
func CapitalizeFirst(text string) string {
	first, size := utf8.DecodeRuneInString(text)
	if first == utf8.RuneError {
		return text
	}
	return string(unicode.ToUpper(first)) + text[size:]
}

// UntaggedStringFields lists the string-holding fields of a struct type
// (string, *string, []string, *[]string, including those inside nested and
// embedded structs) that lack a valid sanitize tag. A malformed tag is listed
// too, with the reason.
func UntaggedStringFields(structType reflect.Type) []string {
	for structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	var problems []string
	collectUntagged(structType, structType.Name(), &problems)
	return problems
}

func collectUntagged(structType reflect.Type, prefix string, problems *[]string) {
	for i := range structType.NumField() {
		field := structType.Field(i)
		// An embedded struct is walked even when its type is unexported: its
		// exported fields are promoted, and JSON binds them.
		if !field.IsExported() && !field.Anonymous {
			continue
		}
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		name := prefix + "." + field.Name

		if fieldType.Kind() == reflect.Struct {
			collectUntagged(fieldType, name, problems)
			continue
		}
		holdsString := fieldType.Kind() == reflect.String ||
			(fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.String)
		if !holdsString {
			continue
		}

		tag, tagged := field.Tag.Lookup(SanitizeTag)
		if !tagged {
			*problems = append(*problems, name+": no sanitize tag")
		} else if _, err := parseSanitizeRules(tag); err != nil {
			*problems = append(*problems, name+": "+err.Error())
		}
	}
}
