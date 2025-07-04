package slug_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/cmd/internal/utils/slug"
)

type SlugTestSuite struct {
	suite.Suite
}

func TestSlugTestSuite(t *testing.T) {
	suite.Run(t, new(SlugTestSuite))
}

func (suite *SlugTestSuite) TestToSaneCheck() {
	tests := []struct {
		name      string
		input     string
		minLength int
		maxLength int
		expected  string
		hasError  bool
	}{
		{
			name:      "valid slug",
			input:     "hello_world",
			minLength: 3,
			maxLength: 20,
			expected:  "hello_world",
			hasError:  false,
		},
		{
			name:      "valid slug with numbers",
			input:     "hello123_world",
			minLength: 3,
			maxLength: 20,
			expected:  "hello123_world",
			hasError:  false,
		},
		{
			name:      "slug too short",
			input:     "hi",
			minLength: 3,
			maxLength: 20,
			expected:  "hi",
			hasError:  true,
		},
		{
			name:      "slug with uppercase",
			input:     "HelloWorld",
			minLength: 3,
			maxLength: 20,
			expected:  "HelloWorld",
			hasError:  true,
		},
		{
			name:      "slug with spaces",
			input:     "hello world",
			minLength: 3,
			maxLength: 20,
			expected:  "hello world",
			hasError:  true,
		},
		{
			name:      "slug with multiple underscores",
			input:     "hello___world",
			minLength: 3,
			maxLength: 20,
			expected:  "hello_world",
			hasError:  false,
		},
		{
			name:      "slug with leading/trailing underscores",
			input:     "_hello_world_",
			minLength: 3,
			maxLength: 20,
			expected:  "hello_world",
			hasError:  false,
		},
		{
			name:      "slug with leading/trailing spaces",
			input:     "  hello world  ",
			minLength: 3,
			maxLength: 20,
			expected:  "  hello world  ",
			hasError:  true,
		},
		{
			name:      "slug too long",
			input:     "very_long_slug_that_exceeds_maximum_length",
			minLength: 3,
			maxLength: 20,
			expected:  "very_long_slug_that_",
			hasError:  false,
		},
		{
			name:      "slug with invalid characters",
			input:     "hello-world!",
			minLength: 3,
			maxLength: 20,
			expected:  "hello-world!",
			hasError:  true,
		},
		{
			name:      "empty slug",
			input:     "",
			minLength: 3,
			maxLength: 20,
			expected:  "",
			hasError:  true,
		},
		{
			name:      "slug with mixed case and spaces",
			input:     "Hello World 123",
			minLength: 3,
			maxLength: 20,
			expected:  "Hello World 123",
			hasError:  true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		suite.Run(tt.name, func() {
			suite.T().Parallel()
			result, err := slug.ToSaneCheck(tt.input, tt.minLength, tt.maxLength)

			if tt.hasError {
				suite.Require().Error(err)
				suite.Equal(tt.expected, result)
			} else {
				suite.Require().NoError(err)
				suite.Equal(tt.expected, result)
			}
		})
	}
}

func (suite *SlugTestSuite) TestToSaneCheck_ErrorTypes() {
	suite.Run("too short error", func() {
		suite.T().Parallel()
		_, err := slug.ToSaneCheck("hi", 5, 20)
		suite.Require().Error(err)
		suite.Require().Contains(err.Error(), "Slug must be at least 5 characters")
	})

	suite.Run("invalid characters error", func() {
		suite.T().Parallel()
		_, err := slug.ToSaneCheck("hello-world", 3, 20)
		suite.Require().Error(err)
		suite.Require().Contains(err.Error(), "Slug can only contain lowercase letters, numbers, and underscores")
	})
}

func (suite *SlugTestSuite) TestCreateFromName() {
	tests := []struct {
		name      string
		input     string
		minLength int
		maxLength int
		expected  string
	}{
		{
			name:      "simple name",
			input:     "Hello World",
			minLength: 3,
			maxLength: 20,
			expected:  "hello_world",
		},
		{
			name:      "camelCase",
			input:     "camelCase",
			minLength: 3,
			maxLength: 20,
			expected:  "camel_case",
		},
		{
			name:      "PascalCase",
			input:     "PascalCase",
			minLength: 3,
			maxLength: 20,
			expected:  "pascal_case",
		},
		{
			name:      "with numbers",
			input:     "Hello123World",
			minLength: 3,
			maxLength: 20,
			expected:  "hello123world",
		},
		{
			name:      "with underscores",
			input:     "hello_world_test",
			minLength: 3,
			maxLength: 20,
			expected:  "hello_world_test",
		},
		{
			name:      "with special characters",
			input:     "hello@world#test",
			minLength: 3,
			maxLength: 20,
			expected:  "hello_world_test",
		},
		{
			name:      "too short name",
			input:     "Hi",
			minLength: 5,
			maxLength: 20,
			expected:  "hi_", // will have UUID suffix
		},
		{
			name:      "too long name",
			input:     "very_long_name_that_exceeds_maximum_length",
			minLength: 3,
			maxLength: 15,
			expected:  "very_long_name",
		},
		{
			name:      "empty name",
			input:     "",
			minLength: 3,
			maxLength: 20,
			expected:  "", // will be replaced by UUID
		},
		{
			name:      "single character",
			input:     "A",
			minLength: 3,
			maxLength: 20,
			expected:  "a_", // will have UUID suffix
		},
		{
			name:      "multiple consecutive capitals",
			input:     "HTMLParser",
			minLength: 3,
			maxLength: 20,
			expected:  "htmlparser",
		},
		{
			name:      "name with numbers at start",
			input:     "123Hello",
			minLength: 3,
			maxLength: 20,
			expected:  "123hello",
		},
	}

	for _, tt := range tests {
		// capture range variable
		suite.Run(tt.name, func() {
			suite.T().Parallel()
			result := slug.CreateFromName(tt.input, tt.minLength, tt.maxLength)

			// For cases where the result should have a UUID suffix, we can't predict the exact result
			// but we can check the structure
			//nolint:gocritic // This is a test
			if strings.HasSuffix(tt.expected, "_") {
				suite.GreaterOrEqual(len(result), tt.minLength)
				suite.LessOrEqual(len(result), tt.maxLength)
				suite.True(strings.HasPrefix(result, strings.TrimSuffix(tt.expected, "_")))
				suite.Contains(result, "_")
			} else if tt.expected == "" {
				// Empty input results in a UUID
				suite.GreaterOrEqual(len(result), tt.minLength)
				suite.LessOrEqual(len(result), tt.maxLength)
				suite.Len(result, 8) // UUID should be 8 characters
				uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}$`)
				suite.True(uuidRegex.MatchString(result))
			} else {
				suite.Equal(tt.expected, result)
			}
		})
	}
}

func (suite *SlugTestSuite) TestCreateFromName_EdgeCases() {
	suite.Run("minimum length enforcement", func() {
		suite.T().Parallel()
		result := slug.CreateFromName("Hi", 10, 20)
		suite.GreaterOrEqual(len(result), 10)
		suite.LessOrEqual(len(result), 20)
		suite.True(strings.HasPrefix(result, "hi_"))
	})

	suite.Run("maximum length enforcement", func() {
		suite.T().Parallel()
		result := slug.CreateFromName("very_long_name_that_exceeds_maximum_length", 3, 10)
		suite.LessOrEqual(len(result), 10)
		suite.False(strings.HasSuffix(result, "_"))
	})

	suite.Run("UUID suffix format", func() {
		suite.T().Parallel()
		result := slug.CreateFromName("Hi", 10, 20)
		parts := strings.Split(result, "_")
		suite.Len(parts, 2)
		suite.Equal("hi", parts[0])
		suite.Len(parts[1], 8) // UUID should be 8 characters

		// Verify it's a valid UUID format (hexadecimal)
		uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}$`)
		suite.True(uuidRegex.MatchString(parts[1]))
	})
}

func (suite *SlugTestSuite) TestCreateFromName_Consistency() {
	suite.T().Parallel()
	// Test that the same input produces consistent results
	input := "Hello World"
	result1 := slug.CreateFromName(input, 3, 20)
	result2 := slug.CreateFromName(input, 3, 20)
	suite.Equal(result1, result2)
}

func (suite *SlugTestSuite) TestCreateFromName_LengthConstraints() {
	suite.Run("exact minimum length", func() {
		suite.T().Parallel()
		result := slug.CreateFromName("Hi", 5, 20)
		suite.GreaterOrEqual(len(result), 5)
	})

	suite.Run("exact maximum length", func() {
		suite.T().Parallel()
		result := slug.CreateFromName("very_long_name_that_exceeds_maximum_length", 3, 15)
		suite.LessOrEqual(len(result), 15)
	})

	suite.Run("minimum equals maximum", func() {
		suite.T().Parallel()
		result := slug.CreateFromName("Hello World", 10, 10)
		suite.Len(result, 10)
	})
}

func (suite *SlugTestSuite) TestCreateFromName_SpecialCharacters() {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"hyphens", "hello-world", "hello_world"},
		{"dots", "hello.world", "hello_world"},
		{"commas", "hello,world", "hello_world"},
		{"exclamation", "hello!world", "hello_world"},
		{"question", "hello?world", "hello_world"},
		{"at symbol", "hello@world", "hello_world"},
		{"hash", "hello#world", "hello_world"},
		{"dollar", "hello$world", "hello_world"},
		{"percent", "hello%world", "hello_world"},
		{"ampersand", "hello&world", "hello_world"},
		{"asterisk", "hello*world", "hello_world"},
		{"parentheses", "hello(world)", "hello_world"},
		{"brackets", "hello[world]", "hello_world"},
		{"braces", "hello{world}", "hello_world"},
		{"pipe", "hello|world", "hello_world"},
		{"backslash", "hello\\world", "hello_world"},
		{"forward slash", "hello/world", "hello_world"},
		{"plus", "hello+world", "hello_world"},
		{"equals", "hello=world", "hello_world"},
		{"semicolon", "hello;world", "hello_world"},
		{"colon", "hello:world", "hello_world"},
		{"quote", "hello\"world", "hello_world"},
		{"apostrophe", "hello'world", "hello_world"},
		{"backtick", "hello`world", "hello_world"},
		{"tilde", "hello~world", "hello_world"},
		{"caret", "hello^world", "hello_world"},
	}

	for _, tt := range tests {
		// capture range variable
		suite.Run(tt.name, func() {
			suite.T().Parallel()
			result := slug.CreateFromName(tt.input, 3, 20)
			suite.Equal(tt.want, result)
		})
	}
}

func (suite *SlugTestSuite) TestCreateFromName_Unicode() {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"accented characters", "café", "café"},
		{"cyrillic", "привет", "привет"},
		{"chinese", "你好", ""}, // will have UUID suffix due to minLength
		{"emoji", "hello😀world", "hello_world"},
		{"mixed unicode", "café_привет_hello", "café_привет_h"},
	}

	for _, tt := range tests {
		// capture range variable
		suite.Run(tt.name, func() {
			suite.T().Parallel()
			result := slug.CreateFromName(tt.input, 3, 20)

			// For Chinese characters, the result will be a UUID (8 characters)
			if tt.name == "chinese" {
				suite.GreaterOrEqual(len(result), 3)
				suite.LessOrEqual(len(result), 20)
				// Should be exactly 8 characters (UUID)
				suite.Len(result, 8)
				// Should be hexadecimal
				uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}$`)
				suite.True(uuidRegex.MatchString(result))
			} else {
				suite.Equal(tt.want, result)
			}
		})
	}
}
