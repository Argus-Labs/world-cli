package tomlutil

import (
	"os"
	"path/filepath"
	"testing"
)

// createTestFile is a helper function that creates a temporary TOML file with given content.
func createTestFile(t *testing.T, content string) string {
	tempDir := t.TempDir()
	tmpFile := filepath.Join(tempDir, "test.toml")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return tmpFile
}

func TestReadTOML(t *testing.T) {
	t.Parallel()
	// Create a temporary test file
	content := `
[test]
key = "value"
number = 42
[nested]
string = "nested value"
`
	tmpFile := createTestFile(t, content)

	// Test successful read
	var config map[string]any
	err := ReadTOML(tmpFile, &config)
	if err != nil {
		t.Errorf("ReadTOML failed: %v", err)
	}
	testSection, ok := config["test"].(map[string]any)
	if !ok {
		t.Fatal("test section should exist and be a map")
	}
	if testSection["key"] != "value" {
		t.Error("Expected key to be 'value'")
	}
	if testSection["number"] != int64(42) {
		t.Error("Expected number to be 42")
	}

	// Test reading non-existent file
	err = ReadTOML("nonexistent.toml", &config)
	if err == nil {
		t.Error("Expected error when reading non-existent file")
	}
}

func TestWriteTOML(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	tmpFile := filepath.Join(tempDir, "write_test.toml")

	// Test writing new file
	config := map[string]any{
		"section": map[string]any{
			"key":    "value",
			"number": int64(42),
		},
	}

	err := WriteTOML(tmpFile, config)
	if err != nil {
		t.Errorf("WriteTOML failed: %v", err)
	}

	// Verify written content
	var readConfig map[string]any
	err = ReadTOML(tmpFile, &readConfig)
	if err != nil {
		t.Errorf("ReadTOML failed: %v", err)
	}
	section, ok := readConfig["section"].(map[string]any)
	if !ok {
		t.Fatal("section should exist and be a map")
	}
	if section["key"] != "value" {
		t.Error("Expected key to be 'value'")
	}
	if section["number"] != int64(42) {
		t.Error("Expected number to be 42")
	}

	// Test writing to invalid path
	err = WriteTOML("/invalid/path/test.toml", config)
	if err == nil {
		t.Error("Expected error when writing to invalid path")
	}
}

func TestUpdateTOMLSection(t *testing.T) {
	t.Parallel()
	// Create initial test file
	initialContent := `
[existing]
key = "value"
[update]
keep = "kept"
change = "old"
`
	tmpFile := createTestFile(t, initialContent)

	// Test updating existing section
	updates := map[string]any{
		"change": "new",
		"add":    "added",
	}
	err := UpdateTOMLSection(tmpFile, "update", updates)
	if err != nil {
		t.Errorf("UpdateTOMLSection failed: %v", err)
	}

	// Verify updates
	var config map[string]any
	err = ReadTOML(tmpFile, &config)
	if err != nil {
		t.Errorf("ReadTOML failed: %v", err)
	}

	updateSection, ok := config["update"].(map[string]any)
	if !ok {
		t.Fatal("update section should exist and be a map")
	}
	if updateSection["keep"] != "kept" {
		t.Error("Expected keep to be 'kept'")
	}
	if updateSection["change"] != "new" {
		t.Error("Expected change to be 'new'")
	}
	if updateSection["add"] != "added" {
		t.Error("Expected add to be 'added'")
	}

	// Test creating new section
	newUpdates := map[string]any{
		"new": "value",
	}
	err = UpdateTOMLSection(tmpFile, "newsection", newUpdates)
	if err != nil {
		t.Errorf("UpdateTOMLSection failed: %v", err)
	}

	err = ReadTOML(tmpFile, &config)
	if err != nil {
		t.Errorf("ReadTOML failed: %v", err)
	}
	newSection, ok := config["newsection"].(map[string]any)
	if !ok {
		t.Fatal("newsection should exist and be a map")
	}
	if newSection["new"] != "value" {
		t.Error("Expected new to be 'value'")
	}
}

func TestGetTOMLSection(t *testing.T) {
	t.Parallel()
	// Create test file
	content := `
[section]
key = "value"
number = 42
[empty]
`
	tmpFile := createTestFile(t, content)

	// Test getting existing section
	section, err := GetTOMLSection(tmpFile, "section")
	if err != nil {
		t.Errorf("GetTOMLSection failed: %v", err)
	}
	if section["key"] != "value" {
		t.Error("Expected key to be 'value'")
	}
	if section["number"] != int64(42) {
		t.Error("Expected number to be 42")
	}

	// Test getting non-existent section
	_, err = GetTOMLSection(tmpFile, "nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent section")
	}

	// Test getting empty section
	emptySection, err := GetTOMLSection(tmpFile, "empty")
	if err != nil {
		t.Errorf("GetTOMLSection failed: %v", err)
	}
	if len(emptySection) != 0 {
		t.Error("Expected empty section to be empty")
	}
}

func TestCreateTOMLFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	tmpFile := filepath.Join(tempDir, "nested", "create_test.toml")

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
	if err != nil {
		t.Errorf("CreateTOMLFile failed: %v", err)
	}

	// Verify file was created with correct content
	var config map[string]any
	err = ReadTOML(tmpFile, &config)
	if err != nil {
		t.Errorf("ReadTOML failed: %v", err)
	}
	if config["section1"].(map[string]any)["key"] != "value" {
		t.Error("Expected key to be 'value'")
	}
	if config["section2"].(map[string]any)["number"] != int64(42) {
		t.Error("Expected number to be 42")
	}

	// Test creating file that already exists (should do nothing)
	newSections := map[string]map[string]any{
		"different": {"key": "value"},
	}
	err = CreateTOMLFile(tmpFile, newSections)
	if err != nil {
		t.Errorf("CreateTOMLFile failed: %v", err)
	}

	// Verify original content wasn't changed
	err = ReadTOML(tmpFile, &config)
	if err != nil {
		t.Errorf("ReadTOML failed: %v", err)
	}
	if config["section1"].(map[string]any)["key"] != "value" {
		t.Error("Expected key to be 'value'")
	}
}
