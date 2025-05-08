package teacmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/magefile/mage/sh"
	"gotest.tools/v3/assert"
)

const templateURLTest = "https://github.com/Argus-Labs/starter-game-template.git"

func TestGitCloneCmd(t *testing.T) {
	t.Parallel()
	type param struct {
		url       string
		targetDir string
		initMsg   string
	}

	test := []struct {
		name     string
		wantErr  bool
		expected int
		param    param
	}{
		{
			name:     "error clone wrong address",
			wantErr:  true,
			expected: 1,
			param: param{
				url:       "wrong address",
				targetDir: "targetDir",
				initMsg:   "initMsg",
			},
		},
		{
			name:    "success",
			wantErr: false,
			param: param{
				url:       templateURLTest,
				targetDir: filepath.Join(os.TempDir(), "worldclitest"),
				initMsg:   "initMsg",
			},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// clean up before test
			cleanUpDir(tt.param.targetDir)

			err := GitCloneCmd(tt.param.url, tt.param.targetDir, tt.param.initMsg)
			if tt.wantErr {
				assert.Equal(t, sh.ExitStatus(err), tt.expected)
			} else {
				assert.NilError(t, err)
			}

			// clean up after test
			cleanUpDir(tt.param.targetDir)
		})
	}
}

func cleanUpDir(targetDir string) {
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		err := os.RemoveAll(targetDir)
		if err != nil {
			fmt.Println(err) //nolint:forbidigo // test
		}
	}
}

func TestAppendToToml(t *testing.T) {
	t.Parallel()
	// Define test cases
	tests := []struct {
		name          string
		section       string
		fields        map[string]any
		expectedError error
	}{
		{
			name:    "append first section and fields",
			section: "example_section",
			fields: map[string]any{
				"field1": "example_value",
				"field2": 123,
			},
			expectedError: nil,
		},
		{
			name:    "append fields to existing section",
			section: "example_section",
			fields: map[string]any{
				"field1": "replaced_value",
				"field2": 321,
			},
			expectedError: nil,
		},
		{
			name:    "create new section and append fields",
			section: "new_section",
			fields: map[string]any{
				"field3": true,
				"field4": 3.14,
			},
			expectedError: nil,
		},
		// Add more test cases if needed
	}

	// Run test cases
	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a temporary file for this subtest
			tempFile, err := os.CreateTemp(t.TempDir(), "test.toml")
			if err != nil {
				t.Fatalf("failed to create temporary file: %v", err)
			}
			t.Cleanup(func() {
				os.Remove(tempFile.Name())
			})

			// For tests that need to append to existing content, create the initial content
			if tt.name == "append fields to existing section" {
				initialContent := `[example_section]
field1 = "old_value"
field2 = 0
`
				if err := os.WriteFile(tempFile.Name(), []byte(initialContent), 0644); err != nil {
					t.Fatalf("failed to write initial content: %v", err)
				}
			}

			err = appendToToml(tempFile.Name(), tt.section, tt.fields)
			if !errors.Is(err, tt.expectedError) {
				t.Errorf("unexpected error: got %v, want %v", err, tt.expectedError)
			}
		})
	}
}

func TestGenerateRandomHexString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "length 0",
			length: 0,
		},
		{
			name:   "length 8",
			length: 8,
		},
		{
			name:   "length 16",
			length: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := generateRandomHexString(tt.length)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(got) != tt.length*2 {
				t.Errorf("unexpected length of generated hex string: got %d, want %d", len(got), tt.length*2)
			}
		})
	}
}
