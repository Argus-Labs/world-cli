package tomlutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ReadTOML reads a TOML file and unmarshals it into the provided interface.
func ReadTOML(path string, v interface{}) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read TOML file at '%s': %w", path, err)
	}

	if err := toml.Unmarshal(content, v); err != nil {
		return fmt.Errorf("failed to parse TOML file at '%s': %w", path, err)
	}

	return nil
}

// WriteTOML marshals the provided interface and writes it to a TOML file.
func WriteTOML(path string, v interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create TOML file at '%s': %w", path, err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("failed to write TOML file: %w", err)
	}

	return nil
}

// UpdateTOMLSection updates a specific section in a TOML file.
// If the section doesn't exist, it will be created.
func UpdateTOMLSection(path string, sectionName string, updates map[string]interface{}) error {
	// Read the existing config
	var config map[string]interface{}
	if err := ReadTOML(path, &config); err != nil {
		return err
	}

	// Get or create the section
	section, ok := config[sectionName].(map[string]interface{})
	if !ok {
		section = make(map[string]interface{})
		config[sectionName] = section
	}

	// Apply updates
	for key, value := range updates {
		section[key] = value
	}

	// Write back to file
	return WriteTOML(path, config)
}

// GetTOMLSection reads a TOML file and returns a specific section.
func GetTOMLSection(path string, sectionName string) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := ReadTOML(path, &config); err != nil {
		return nil, err
	}

	section, ok := config[sectionName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("section '%s' not found in TOML file", sectionName)
	}

	return section, nil
}

// CreateTOMLFile creates a new TOML file with the given sections if it doesn't exist.
// If the file already exists, it does nothing.
func CreateTOMLFile(path string, sections map[string]map[string]interface{}) error {
	// Check if file exists
	if _, err := os.Stat(path); err == nil {
		return nil // File exists, nothing to do
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for TOML file: %w", err)
	}

	// Create the file with the given sections
	return WriteTOML(path, sections)
}
