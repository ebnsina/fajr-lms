package api

import (
	"crypto/rand"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/ebnsina/fajr-lms/internal/httpx"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,126}$`)

func invalid(field, message string) *httpx.Error {
	return &httpx.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_" + field, Message: message, Field: field}
}

// requireText trims and length-checks a required field.
func requireText(field, value string, max int) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", invalid(field, "This field is required.")
	}
	if len([]rune(v)) > max {
		return "", invalid(field, "This field is too long.")
	}
	return v, nil
}

// Slugify keeps ASCII words only, so an Arabic or Bengali title falls back to a
// generated slug rather than an empty one.
func Slugify(title string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(title) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > 100 {
		slug = strings.Trim(slug[:100], "-")
	}
	if !slugRe.MatchString(slug) {
		return "c-" + strings.ToLower(rand.Text()[:10])
	}
	return slug
}
