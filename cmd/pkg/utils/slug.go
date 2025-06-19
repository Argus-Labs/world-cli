package utils

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

// Pre-compiled regex for merging multiple underscores.
var underscoreRegex = regexp.MustCompile(`_+`)

// SlugToSaneCheck checks that slug is valid, and returns a sanitized version.
func SlugToSaneCheck(slug string, minLength int, maxLength int) (string, error) {
	if len(slug) < minLength {
		return slug, eris.Errorf("Slug must be at least %d characters", minLength)
	}

	// Check if slug contains only allowed characters.
	matched, err := regexp.MatchString("^[a-z0-9_]+$", slug)
	if err != nil {
		return slug, eris.Wrap(err, "Error validating slug format")
	}
	if !matched {
		return slug, eris.New("Slug can only contain lowercase letters, numbers, and underscores")
	}

	// Process the slug, and ensure it's in sane format.
	returnSlug := strings.ToLower(strings.TrimSpace(slug))
	returnSlug = strings.ReplaceAll(returnSlug, " ", "_")
	returnSlug = underscoreRegex.ReplaceAllString(returnSlug, "_")
	returnSlug = strings.Trim(returnSlug, "_")

	if len(returnSlug) > maxLength {
		return returnSlug[:maxLength], nil
	}

	return returnSlug, nil
}

func CreateSlugFromName(name string, minLength int, maxLength int) string {
	shorten := len(name) > maxLength

	var slug string
	wroteUnderscore := false
	hadCapital := false
	for i, r := range name {
		switch {
		case unicode.IsLower(r) || unicode.IsNumber(r):
			// copy lowercase letters and numbers
			slug += string(r)
			wroteUnderscore = false
			hadCapital = unicode.IsNumber(r) // treat numbers as capital letters
		case unicode.IsUpper(r):
			// convert capital letter to lower, with _ if dealing with CamelCase ( -> camel_case )
			if !shorten && i != 0 && !wroteUnderscore && !hadCapital {
				slug += "_"
			}
			slug += string(unicode.ToLower(r))
			wroteUnderscore = false
			hadCapital = true
		case (r == '_' || !shorten) && !wroteUnderscore:
			// underscore is preserved (but many fused into one)
			// unless the input was too long, other characters are converted to underscores (but many fused into one)
			slug += "_"
			wroteUnderscore = true
			hadCapital = false
		}
	}
	slug = strings.Trim(slug, "_")
	if len(slug) < minLength {
		slug += "_" + uuid.NewString()[:8] // add the first 8 characters of the UUID
		slug = strings.TrimLeft(slug, "_")
	}
	if len(slug) > maxLength {
		slug = slug[:maxLength]
		slug = strings.TrimRight(slug, "_")
	}
	return slug
}
