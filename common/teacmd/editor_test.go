package teacmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

const (
	testDir       = ".test-worldcli"
	testTargetDir = ".test-worldcli/.editor"
)

func TestSetupCardinalEditor(t *testing.T) {
	t.Run("setup cardinal editor", func(t *testing.T) {
		cleanUpDir(testDir)

		editorDir, err := downloadReleaseIfNotCached(latestReleaseURL, testDir)
		assert.NilError(t, err)

		// check if editor directory exists
		_, err = os.Stat(editorDir)
		exists := os.IsNotExist(err)
		assert.Equal(t, exists, false)

		// check if it's not empty
		dir, err := os.ReadDir(editorDir)
		assert.NilError(t, err)
		assert.Assert(t, len(dir) != 0)

		// check if folder is renamed
		err = copyDir(editorDir, testTargetDir)
		assert.NilError(t, err)

		_, err = os.Stat(testTargetDir)
		exists = os.IsNotExist(err)
		assert.Equal(t, exists, false)

		// check if project id is replaced
		projectID := "__THIS_IS_FOR_TESTING_ONLY__"
		err = replaceProjectIDs(testTargetDir, projectID)
		assert.NilError(t, err)

		containsNewID, err := containsCardinalProjectIDPlaceholder(testTargetDir, projectID)
		assert.NilError(t, err)
		assert.Equal(t, containsNewID, true)

		// TODO: check if cardinal editor works

		cleanUpDir(testDir)
	})
}

func containsCardinalProjectIDPlaceholder(dir string, originalID string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, file := range files {
		// recurse over child directories
		if file.IsDir() {
			contains, err := containsCardinalProjectIDPlaceholder(filepath.Join(dir, file.Name()), originalID)
			if contains || err != nil {
				return contains, err
			}
			continue
		}

		if strings.HasSuffix(file.Name(), ".js") {
			filePath := filepath.Join(dir, file.Name())

			content, err := os.ReadFile(filePath)
			if err != nil {
				return false, err
			}

			if strings.Contains(string(content), originalID) {
				return true, nil
			}
		}
	}

	return false, nil
}
func TestCopyDir(t *testing.T) {
	t.Run("Test copy directory", func(t *testing.T) {
		err := os.MkdirAll("tmp", 0755)
		assert.NilError(t, err)

		err = os.MkdirAll(filepath.Join("tmp", "subdir"), 0755)
		assert.NilError(t, err)

		_, err = os.Create(filepath.Join("tmp", "file1"))
		assert.NilError(t, err)

		_, err = os.Create(filepath.Join("tmp", "subdir", "file2"))
		assert.NilError(t, err)

		err = copyDir("tmp", "tmp2")
		assert.NilError(t, err)

		_, err = os.Stat("tmp")
		assert.NilError(t, err)

		_, err = os.Stat("tmp2")
		assert.NilError(t, err)

		_, err = os.Stat(filepath.Join("tmp2", "subdir"))
		assert.NilError(t, err)

		_, err = os.Stat(filepath.Join("tmp2", "file1"))
		assert.NilError(t, err)

		_, err = os.Stat(filepath.Join("tmp2", "subdir", "file2"))
		assert.NilError(t, err)

		cleanUpDir("tmp")
		cleanUpDir("tmp2")
	})
}

func TestStrippedGUID(t *testing.T) {
	t.Run("Test guid doesn't contain -", func(t *testing.T) {
		s := strippedGUID()
		assert.Check(t, !strings.Contains(s, "-"))
	})
}

func TestAddFileVersion(t *testing.T) {
	testCases := []struct {
		version    string
		shouldFail bool
	}{
		{"v0.1.0", false},
		{"/v1.0.1", true},
	}

	for _, tc := range testCases {
		err := addFileVersion(tc.version)

		if tc.shouldFail && err == nil {
			t.Errorf("Expected addFileVersion to fail for version '%s', but it didn't", tc.version)
		} else if !tc.shouldFail {
			if err != nil {
				t.Errorf("addFileVersion failed for version '%s': %s", tc.version, err)
			}

			// Check if file exists
			_, err = os.Stat(tc.version)
			if os.IsNotExist(err) {
				t.Errorf("file %s was not created", tc.version)
			}

			// Cleanup
			err = os.Remove(tc.version)
			if err != nil {
				t.Logf("Failed to delete test file '%s': %s", tc.version, err)
			}
		}
	}
}

func TestGetModuleVersion(t *testing.T) {
	// Setup temporary go.mod file for testing
	gomodContent := `
module example.com/mymodule

go 1.21.1

require (
	pkg.world.dev/world-engine/cardinal v1.2.3-beta
	github.com/moduleexample/v3 v3.2.1
)
`
	gomodPath := "./test-go.mod"
	err := os.WriteFile(gomodPath, []byte(gomodContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(gomodPath) // clean up

	tests := []struct {
		name        string
		gomodPath   string
		modulePath  string
		wantVersion string
		expectError bool
	}{
		{
			name:        "Module exists",
			gomodPath:   gomodPath,
			modulePath:  "pkg.world.dev/world-engine/cardinal",
			wantVersion: "v1.2.3-beta",
			expectError: false,
		},
		{
			name:        "Module does not exist",
			gomodPath:   gomodPath,
			modulePath:  "nonexistent/module",
			wantVersion: "",
			expectError: true,
		},
		{
			name:        "go.mod file does not exist",
			gomodPath:   "./nonexistent-go.mod",
			modulePath:  "github.com/moduleexample",
			wantVersion: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := getModuleVersion(tt.gomodPath, tt.modulePath)
			if (err != nil) != tt.expectError {
				t.Errorf("getModuleVersion() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if version != tt.wantVersion {
				t.Errorf("getModuleVersion() = %v, want %v", version, tt.wantVersion)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file and defer its removal
	tempFile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatalf("Unable to create temporary file: %s", err)
	}
	defer os.Remove(tempFile.Name())

	// Test case where the file does exist
	if exists := fileExists(tempFile.Name()); !exists {
		t.Errorf("fileExists(%s) = %v, want %v", tempFile.Name(), exists, true)
	}

	// Remove the file to simulate it not existing
	tempFile.Close()
	os.Remove(tempFile.Name())

	// Test case where the file does not exist
	if exists := fileExists(tempFile.Name()); exists {
		t.Errorf("fileExists(%s) = %v, want %v", tempFile.Name(), exists, false)
	}

	// Create a temporary directory and defer its removal
	tempDir, err := os.MkdirTemp("", "example")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %s", err)
	}
	defer os.RemoveAll(tempDir)

	// Test case where the path is a directory
	if exists := fileExists(tempDir); exists {
		t.Errorf("fileExists(%s) = %v, want %v", tempDir, exists, false)
	}
}
