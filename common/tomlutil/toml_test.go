package tomlutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TOMLTestSuite struct {
	suite.Suite
	tempDir string
}

func (s *TOMLTestSuite) SetupTest() {
	s.tempDir = s.T().TempDir()
}

// createTestFile is a helper function that creates a temporary TOML file with given content
func (s *TOMLTestSuite) createTestFile(content string) string {
	tmpFile := filepath.Join(s.tempDir, "test.toml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(s.T(), err, "Failed to create test file")
	return tmpFile
}

func (s *TOMLTestSuite) TestReadTOML() {
	// Create a temporary test file
	content := `
[test]
key = "value"
number = 42

[nested]
string = "nested value"
`
	tmpFile := s.createTestFile(content)

	// Test successful read
	var config map[string]any
	err := ReadTOML(tmpFile, &config)
	s.Require().NoError(err)
	s.Equal("value", config["test"].(map[string]any)["key"])
	s.Equal(int64(42), config["test"].(map[string]any)["number"])

	// Test reading non-existent file
	err = ReadTOML("nonexistent.toml", &config)
	s.Error(err)
}

func (s *TOMLTestSuite) TestWriteTOML() {
	tmpFile := filepath.Join(s.tempDir, "write_test.toml")

	// Test writing new file
	config := map[string]any{
		"section": map[string]any{
			"key":    "value",
			"number": int64(42),
		},
	}

	err := WriteTOML(tmpFile, config)
	s.Require().NoError(err)

	// Verify written content
	var readConfig map[string]any
	err = ReadTOML(tmpFile, &readConfig)
	s.Require().NoError(err)
	s.Equal(config, readConfig)

	// Test writing to invalid path
	err = WriteTOML("/invalid/path/test.toml", config)
	s.Error(err)
}

func (s *TOMLTestSuite) TestUpdateTOMLSection() {
	// Create initial test file
	initialContent := `
[existing]
key = "value"

[update]
keep = "kept"
change = "old"
`
	tmpFile := s.createTestFile(initialContent)

	// Test updating existing section
	updates := map[string]any{
		"change": "new",
		"add":    "added",
	}
	err := UpdateTOMLSection(tmpFile, "update", updates)
	s.Require().NoError(err)

	// Verify updates
	var config map[string]any
	err = ReadTOML(tmpFile, &config)
	s.Require().NoError(err)

	updateSection := config["update"].(map[string]any)
	s.Equal("kept", updateSection["keep"])
	s.Equal("new", updateSection["change"])
	s.Equal("added", updateSection["add"])

	// Test creating new section
	newUpdates := map[string]any{
		"new": "value",
	}
	err = UpdateTOMLSection(tmpFile, "newsection", newUpdates)
	s.Require().NoError(err)

	err = ReadTOML(tmpFile, &config)
	s.Require().NoError(err)
	s.Equal("value", config["newsection"].(map[string]any)["new"])
}

func (s *TOMLTestSuite) TestGetTOMLSection() {
	// Create test file
	content := `
[section]
key = "value"
number = 42

[empty]
`
	tmpFile := s.createTestFile(content)

	// Test getting existing section
	section, err := GetTOMLSection(tmpFile, "section")
	s.Require().NoError(err)
	s.Equal("value", section["key"])
	s.Equal(int64(42), section["number"])

	// Test getting non-existent section
	_, err = GetTOMLSection(tmpFile, "nonexistent")
	s.Error(err)

	// Test getting empty section
	emptySection, err := GetTOMLSection(tmpFile, "empty")
	s.Require().NoError(err)
	s.Empty(emptySection)
}

func (s *TOMLTestSuite) TestCreateTOMLFile() {
	tmpFile := filepath.Join(s.tempDir, "nested", "create_test.toml")

	// Test creating new file with sections
	sections := map[string]map[string]any{
		"section1": {
			"key": "value",
		},
		"section2": {
			"number": int64(42),
		},
	}

	err := CreateTOMLFile(tmpFile, sections)
	s.Require().NoError(err)

	// Verify file was created with correct content
	var config map[string]any
	err = ReadTOML(tmpFile, &config)
	s.Require().NoError(err)
	s.Equal("value", config["section1"].(map[string]any)["key"])
	s.Equal(int64(42), config["section2"].(map[string]any)["number"])

	// Test creating file that already exists (should do nothing)
	newSections := map[string]map[string]any{
		"different": {"key": "value"},
	}
	err = CreateTOMLFile(tmpFile, newSections)
	s.Require().NoError(err)

	// Verify original content wasn't changed
	err = ReadTOML(tmpFile, &config)
	s.Require().NoError(err)
	s.Equal("value", config["section1"].(map[string]any)["key"])
}

// TestTOMLSuite runs the test suite
func TestTOMLSuite(t *testing.T) {
	suite.Run(t, new(TOMLTestSuite))
}
