package editor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSetupCardinalEditor(t *testing.T) {
	t.Parallel()

	// Create test-specific directories
	testDir := filepath.Join(t.TempDir(), "test-worldcli")
	testTargetDir := filepath.Join(testDir, ".editor")

	t.Run("setup cardinal editor", func(t *testing.T) {
		t.Parallel()
		assert.NilError(t, cleanUpDir(testDir))

		latestVersion, err := getLatestReleaseVersion()
		assert.NilError(t, err)
		downloadURL := fmt.Sprintf(downloadURLPattern, latestVersion, latestVersion)
		editorDir, err := downloadReleaseIfNotCached(latestVersion, downloadURL, testDir)
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
		assert.NilError(t, cleanUpDir(testDir))
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
	t.Parallel()

	t.Run("Test copy directory", func(t *testing.T) {
		t.Parallel()
		// Use test-specific directories
		srcDir := filepath.Join(t.TempDir(), "tmp")
		dstDir := filepath.Join(t.TempDir(), "tmp2")

		err := os.MkdirAll(srcDir, 0755)
		assert.NilError(t, err)

		err = os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
		assert.NilError(t, err)

		_, err = os.Create(filepath.Join(srcDir, "file1"))
		assert.NilError(t, err)

		_, err = os.Create(filepath.Join(srcDir, "subdir", "file2"))
		assert.NilError(t, err)

		err = copyDir(srcDir, dstDir)
		assert.NilError(t, err)

		_, err = os.Stat(srcDir)
		assert.NilError(t, err)

		_, err = os.Stat(dstDir)
		assert.NilError(t, err)

		_, err = os.Stat(filepath.Join(dstDir, "subdir"))
		assert.NilError(t, err)

		_, err = os.Stat(filepath.Join(dstDir, "file1"))
		assert.NilError(t, err)

		_, err = os.Stat(filepath.Join(dstDir, "subdir", "file2"))
		assert.NilError(t, err)

		assert.NilError(t, cleanUpDir(srcDir))
		assert.NilError(t, cleanUpDir(dstDir))
	})
}

func TestStrippedGUID(t *testing.T) {
	t.Parallel()

	t.Run("Test guid doesn't contain -", func(t *testing.T) {
		t.Parallel()
		s := strippedGUID()
		assert.Check(t, !strings.Contains(s, "-"))
	})
}

func TestAddFileVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		version    string
		shouldFail bool
	}{
		{
			name:       "valid version",
			version:    "v0.1.0",
			shouldFail: false,
		},
		{
			name:       "invalid version with leading slash",
			version:    "/v1.0.1",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		// capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// For invalid paths, use the original path to test the error case
			versionPath := tc.version
			if !tc.shouldFail {
				versionPath = filepath.Join(t.TempDir(), tc.version)
			}

			err := addFileVersion(versionPath)

			if tc.shouldFail {
				if err == nil {
					t.Errorf("Expected addFileVersion to fail for version '%s', but it didn't", tc.version)
				}
			} else {
				if err != nil {
					t.Errorf("addFileVersion failed for version '%s': %s", tc.version, err)
				}

				// Check if file exists
				_, err = os.Stat(versionPath)
				if os.IsNotExist(err) {
					t.Errorf("file %s was not created", versionPath)
				}
			}
		})
	}
}

func TestGetModuleVersion(t *testing.T) {
	t.Parallel()

	// Setup temporary go.mod file for testing
	gomodContent := `
module example.com/mymodule
go 1.21.1
require (
	pkg.world.dev/world-engine/cardinal v1.2.3-beta
	github.com/moduleexample/v3 v3.2.1
)
`
	gomodPath := filepath.Join(t.TempDir(), "test-go.mod")
	err := os.WriteFile(gomodPath, []byte(gomodContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

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
			gomodPath:   filepath.Join(t.TempDir(), "nonexistent-go.mod"),
			modulePath:  "github.com/moduleexample",
			wantVersion: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

	t.Run("Test file exists", func(t *testing.T) {
		t.Parallel()
		// Create a temporary file
		tempFile, err := os.CreateTemp(t.TempDir(), "example")
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

		// Test case where the path is a directory
		if exists := fileExists(t.TempDir()); exists {
			t.Errorf("fileExists(%s) = %v, want %v", t.TempDir(), exists, false)
		}
	})
}

func TestGetVersionMap(t *testing.T) {
	t.Parallel()

	// Define test cases
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		want           map[string]string
		wantErr        bool
	}{
		{
			name: "successful response with specific version map",
			serverResponse: `{
				"v1.2.2-beta": "v0.1.0",
				"v1.2.3-beta": "v0.1.0",
				"v1.2.4-beta": "v0.3.0"
			}`,
			serverStatus: http.StatusOK,
			want: map[string]string{
				"v1.2.2-beta": "v0.1.0",
				"v1.2.3-beta": "v0.1.0",
				"v1.2.4-beta": "v0.3.0",
			},
			wantErr: false,
		},
		{
			name:           "HTTP error",
			serverResponse: `HTTP error occurred`,
			serverStatus:   http.StatusInternalServerError,
			want:           nil,
			wantErr:        true,
		},
		{
			name:           "invalid JSON",
			serverResponse: `{"invalid": 1, "format": true}`, // invalid JSON for map[string]string
			serverStatus:   http.StatusOK,
			want:           nil,
			wantErr:        true,
		},
	}

	for _, tc := range tests {
		// capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.serverStatus)
				fmt.Fprintln(w, tc.serverResponse)
			}))
			defer server.Close()
			// Call the function with the test server URL
			got, err := getVersionMap(server.URL)
			if (err != nil) != tc.wantErr {
				t.Errorf("getVersionMap() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if err == nil && !compareMaps(got, tc.want) {
				t.Errorf("getVersionMap() got = %v, want %v", got, tc.want)
			}
		})
	}
}

// compareMaps helps in comparing two maps for equality.
func compareMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
func cleanUpDir(targetDir string) error {
	return os.RemoveAll(targetDir)
}
