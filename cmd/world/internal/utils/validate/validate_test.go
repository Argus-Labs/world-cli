package validate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/cmd/world/internal/utils/validate"
)

type ValidationTestSuite struct {
	suite.Suite
}

func TestValidationSuite(t *testing.T) {
	suite.Run(t, new(ValidationTestSuite))
}

func (s *ValidationTestSuite) TestValidateEmail() {
	tests := []struct {
		name        string
		email       string
		expectError bool
	}{
		// Valid ASCII emails (using real domains)
		{
			name:        "valid simple email",
			email:       "test@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with subdomain",
			email:       "user@mail.google.com",
			expectError: false,
		},
		{
			name:        "valid email with plus",
			email:       "user+tag@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with dots",
			email:       "user.name@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with underscore",
			email:       "user_name@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with percent",
			email:       "user%tag@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with dash",
			email:       "user-tag@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with numbers",
			email:       "user123@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with uppercase",
			email:       "User@Gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with long domain",
			email:       "test@example.co.uk",
			expectError: false,
		},
		{
			name:        "valid email with short domain",
			email:       "test@example.io",
			expectError: false,
		},
		{
			name:        "valid email with special characters",
			email:       "test!user#tag$%&'*+-/=?^_`{|}~@gmail.com",
			expectError: false,
		},

		// Valid international emails (Unicode support) - using real domains
		{
			name:        "valid email with German umlaut",
			email:       "müller@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with French accent",
			email:       "françois@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with Spanish accent",
			email:       "josé@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with Chinese characters",
			email:       "用户@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with Cyrillic characters",
			email:       "тест@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with Japanese characters",
			email:       "テスト@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with Arabic characters",
			email:       "اختبار@gmail.com",
			expectError: false,
		},
		{
			name:        "valid email with international domain",
			email:       "test@müller.de",
			expectError: false,
		},
		{
			name:        "valid email with French domain",
			email:       "test@café.fr",
			expectError: false,
		},
		{
			name:        "valid email with Chinese domain",
			email:       "test@例子.公司",
			expectError: false,
		},

		// Invalid emails
		{
			name:        "empty email",
			email:       "",
			expectError: true,
		},
		{
			name:        "missing @ symbol",
			email:       "testexample.com",
			expectError: true,
		},
		{
			name:        "missing domain",
			email:       "test@",
			expectError: true,
		},
		{
			name:        "missing local part",
			email:       "@example.com",
			expectError: true,
		},
		{
			name:        "consecutive dots in local part",
			email:       "test..user@gmail.com",
			expectError: true,
		},
		{
			name:        "consecutive dots in domain",
			email:       "test@example..com",
			expectError: true,
		},
		{
			name:        "starts with dot",
			email:       ".test@gmail.com",
			expectError: true,
		},
		{
			name:        "ends with dot",
			email:       "test.@gmail.com",
			expectError: true,
		},
		{
			name:        "domain starts with dot",
			email:       "test@.example.com",
			expectError: true,
		},
		{
			name:        "domain ends with dot",
			email:       "test@example.com.",
			expectError: true,
		},
		{
			name:        "invalid characters in local part",
			email:       "test<user@gmail.com",
			expectError: true,
		},
		{
			name:        "invalid characters in domain",
			email:       "test@example<.com",
			expectError: true,
		},
		{
			name:        "spaces in email",
			email:       "test user@gmail.com",
			expectError: true,
		},
		{
			name:        "multiple @ symbols",
			email:       "test@user@gmail.com",
			expectError: true,
		},
		{
			name:        "local part too long",
			email:       strings.Repeat("a", 65) + "@gmail.com",
			expectError: true,
		},
		{
			name:        "domain too long",
			email:       "test@" + strings.Repeat("a", 250) + ".com",
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.T().Parallel()
			err := validate.Email(tt.email)
			if tt.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ValidationTestSuite) TestValidateName() {
	tests := []struct {
		name        string
		inputName   string
		maxLength   int
		expectError bool
	}{
		// Valid names
		{
			name:        "valid simple name",
			inputName:   "John Doe",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with numbers",
			inputName:   "User123",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with underscore",
			inputName:   "user_name",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with dash",
			inputName:   "user-name",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with dot",
			inputName:   "user.name",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with space",
			inputName:   "John Smith",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with apostrophe",
			inputName:   "O'Connor",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with accented characters",
			inputName:   "José García",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with Chinese characters",
			inputName:   "张三",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name with Cyrillic characters",
			inputName:   "Иван Петров",
			maxLength:   50,
			expectError: false,
		},
		{
			name:        "valid name at max length",
			inputName:   strings.Repeat("a", 50),
			maxLength:   50,
			expectError: false,
		},

		// Invalid names
		{
			name:        "empty name",
			inputName:   "",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name too long",
			inputName:   strings.Repeat("a", 51),
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with less than symbol",
			inputName:   "user<name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with greater than symbol",
			inputName:   "user>name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with colon",
			inputName:   "user:name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with quote",
			inputName:   "user\"name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with forward slash",
			inputName:   "user/name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with backslash",
			inputName:   "user\\name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with pipe",
			inputName:   "user|name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with question mark",
			inputName:   "user?name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with asterisk",
			inputName:   "user*name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with null character",
			inputName:   "user\x00name",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with tab character",
			inputName:   "user\tname",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with newline character",
			inputName:   "user\nname",
			maxLength:   50,
			expectError: true,
		},
		{
			name:        "name with carriage return",
			inputName:   "user\rname",
			maxLength:   50,
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.T().Parallel()
			err := validate.Name(tt.inputName, tt.maxLength)
			if tt.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ValidationTestSuite) TestIsValidURL() {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		// Valid URLs (using real domains that should resolve)
		{
			name:        "valid http URL",
			url:         "http://google.com",
			expectError: false,
		},
		{
			name:        "valid https URL",
			url:         "https://google.com",
			expectError: false,
		},
		{
			name:        "valid URL with subdomain",
			url:         "https://www.google.com",
			expectError: false,
		},
		{
			name:        "valid URL with path",
			url:         "https://google.com/search",
			expectError: false,
		},
		{
			name:        "valid URL with query parameters",
			url:         "https://google.com?q=test",
			expectError: false,
		},
		{
			name:        "valid URL with fragment",
			url:         "https://google.com#section",
			expectError: false,
		},
		{
			name:        "valid URL with port",
			url:         "https://google.com:443",
			expectError: false,
		},
		{
			name:        "valid URL with multiple subdomains",
			url:         "https://maps.google.com",
			expectError: false,
		},
		{
			name:        "valid URL with short TLD",
			url:         "https://github.io",
			expectError: false,
		},
		{
			name:        "valid URL with long TLD",
			url:         "https://example.co.uk",
			expectError: false,
		},

		// Invalid URLs
		{
			name:        "missing protocol",
			url:         "google.com",
			expectError: true,
		},
		{
			name:        "invalid protocol",
			url:         "ftp://google.com",
			expectError: true,
		},
		{
			name:        "missing hostname",
			url:         "https://",
			expectError: true,
		},
		{
			name:        "missing TLD",
			url:         "https://example",
			expectError: true,
		},
		{
			name:        "localhost not allowed",
			url:         "https://localhost",
			expectError: true,
		},
		{
			name:        "localhost with port not allowed",
			url:         "https://localhost:8080",
			expectError: true,
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
		},
		{
			name:        "malformed URL",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "URL with spaces",
			url:         "https://google.com/path with spaces",
			expectError: true,
		},
		{
			name:        "URL with invalid characters",
			url:         "https://google.com/path<with>invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.T().Parallel()
			err := validate.IsURL(tt.url)
			if tt.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *ValidationTestSuite) TestIsInWorldCardinalRoot() {
	// Save current working directory
	originalCwd, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		// Restore original working directory
		err := os.Chdir(originalCwd)
		s.Require().NoError(err)
	}()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "world-cardinal-test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	// Test case 1: Not in World Cardinal root (empty directory)
	err = os.Chdir(tempDir)
	s.Require().NoError(err)

	isRoot, err := validate.IsInWorldCardinalRoot()
	s.Require().NoError(err)
	s.Require().False(isRoot, "Should not be in World Cardinal root when directory is empty")

	// Test case 2: Not in World Cardinal root (only world.toml exists)
	worldTomlPath := filepath.Join(tempDir, "world.toml")
	err = os.WriteFile(worldTomlPath, []byte("test content"), 0644)
	s.Require().NoError(err)

	isRoot, err = validate.IsInWorldCardinalRoot()
	s.Require().NoError(err)
	s.Require().False(isRoot, "Should not be in World Cardinal root when only world.toml exists")

	// Test case 3: Not in World Cardinal root (only cardinal directory exists)
	err = os.Remove(worldTomlPath)
	s.Require().NoError(err)
	cardinalDirPath := filepath.Join(tempDir, "cardinal")
	err = os.Mkdir(cardinalDirPath, 0755)
	s.Require().NoError(err)

	isRoot, err = validate.IsInWorldCardinalRoot()
	s.Require().NoError(err)
	s.Require().False(isRoot, "Should not be in World Cardinal root when only cardinal directory exists")

	// Test case 4: In World Cardinal root (both world.toml and cardinal directory exist)
	err = os.WriteFile(worldTomlPath, []byte("test content"), 0644)
	s.Require().NoError(err)

	isRoot, err = validate.IsInWorldCardinalRoot()
	s.Require().NoError(err)
	s.Require().True(isRoot, "Should be in World Cardinal root when both world.toml and cardinal directory exist")

	// Test case 5: Not in World Cardinal root (world.toml is a directory)
	err = os.Remove(worldTomlPath)
	s.Require().NoError(err)
	err = os.Mkdir(worldTomlPath, 0755)
	s.Require().NoError(err)

	isRoot, err = validate.IsInWorldCardinalRoot()
	s.Require().NoError(err)
	s.Require().False(isRoot, "Should not be in World Cardinal root when world.toml is a directory")

	// Test case 6: Not in World Cardinal root (cardinal is a file)
	err = os.RemoveAll(cardinalDirPath)
	s.Require().NoError(err)
	err = os.WriteFile(cardinalDirPath, []byte("test content"), 0644)
	s.Require().NoError(err)

	isRoot, err = validate.IsInWorldCardinalRoot()
	s.Require().NoError(err)
	s.Require().False(isRoot, "Should not be in World Cardinal root when cardinal is a file")
}
