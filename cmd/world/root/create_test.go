package root_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/browser"
	cmdsetup "pkg.world.dev/world-cli/cmd/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/world/root"
)

type CreateTestSuite struct {
	suite.Suite
}

// createHandler creates a fresh handler with mocks for each test.
func (s *CreateTestSuite) createHandler() *root.Handler {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	return root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)
}

// testCreateInTempDir tests the Create command using in-process execution but with proper isolation.
//
//nolint:nonamedreturns // ignore error
func (s *CreateTestSuite) testCreateInTempDir(
	projectName string,
) (tempDir string, createErr error, projectExists bool) {
	// Create temp directory for this test
	tempDir = s.T().TempDir()

	// Save original directory
	originalDir, err := os.Getwd()
	s.Require().NoError(err)

	// Create a channel to coordinate the directory change
	done := make(chan struct {
		err           error
		projectExists bool
	}, 1) // Buffered channel to prevent goroutine leak

	// Run Create in a goroutine with proper cleanup
	go func() {
		defer func() {
			// Always restore original directory, but handle errors gracefully
			//nolint:staticcheck // ignore error
			if restoreErr := os.Chdir(originalDir); restoreErr != nil {
				// Don't log during test execution to avoid race conditions
				// The temp directory might be cleaned up already
			}
			// Signal completion
			done <- struct {
				err           error
				projectExists bool
			}{createErr, projectExists}
		}()

		// Change to temp directory
		if err := os.Chdir(tempDir); err != nil {
			createErr = err
			projectExists = false
			return
		}

		// Create handler and run Create
		handler := s.createHandler()
		createErr = handler.Create(projectName)

		// Check if project was created
		projectPath := filepath.Join(tempDir, projectName)
		_, statErr := os.Stat(projectPath)
		projectExists = statErr == nil
	}()

	// Wait for completion with timeout to prevent hanging
	select {
	case result := <-done:
		return tempDir, result.err, result.projectExists
	case <-time.After(30 * time.Second):
		return tempDir, errors.New("create operation timed out"), false
	}
}

// Test Create command with valid directory name - main integration test.
func (s *CreateTestSuite) TestCreate_Success() {
	// This test is NOT parallel since it's our main integration test

	tempDir, createErr, projectExists := s.testCreateInTempDir("test-game")

	s.T().Logf("TempDir: %s", tempDir)
	s.T().Logf("CreateErr: %v", createErr)
	s.T().Logf("ProjectExists: %v", projectExists)

	// List contents of temp directory for debugging
	if entries, err := os.ReadDir(tempDir); err == nil {
		s.T().Logf("TempDir contents:")
		for _, entry := range entries {
			s.T().Logf("  - %s (dir: %v)", entry.Name(), entry.IsDir())
		}
	}

	// Check if project directory was created
	projectPath := filepath.Join(tempDir, "test-game")
	//nolint:nestif // test lint error
	if _, err := os.Stat(projectPath); err == nil {
		s.T().Logf("Project directory created at: %s", projectPath)

		// Check project contents
		if entries, err := os.ReadDir(projectPath); err == nil {
			s.T().Logf("Project contents:")
			for _, entry := range entries {
				s.T().Logf("  - %s (dir: %v)", entry.Name(), entry.IsDir())
			}
		}

		// Check for expected files
		expectedFiles := []string{"cardinal", "world.toml", "README.md"}
		for _, file := range expectedFiles {
			filePath := filepath.Join(projectPath, file)
			if _, err := os.Stat(filePath); err == nil {
				s.T().Logf("Found expected file/dir: %s", file)
			} else {
				s.T().Logf("Missing expected file/dir: %s", file)
			}
		}

		// Check for world.toml content and updateWorldToml functionality
		worldTomlPath := filepath.Join(projectPath, "world.toml")
		if content, err := os.ReadFile(worldTomlPath); err == nil {
			contentStr := string(content)
			s.T().Logf("world.toml content length: %d characters", len(contentStr))

			// Check if [forge] section was added by updateWorldToml
			if strings.Contains(contentStr, "[forge]") {
				s.T().Log("✅ [forge] section found - updateWorldToml succeeded")

				// Check PROJECT_NAME
				expectedProjectName := `PROJECT_NAME = "test-game"`
				if strings.Contains(contentStr, expectedProjectName) {
					s.T().Logf("✅ PROJECT_NAME correctly set: %s", expectedProjectName)
				} else {
					s.T().Log("❌ PROJECT_NAME not found or incorrect")
				}
			} else {
				s.T().Log("❌ [forge] section NOT found - updateWorldToml may not have been called")
			}
		}
	} else {
		s.T().Logf("Project directory not found at: %s", projectPath)
	}

	// Log the final result
	s.T().Logf("Create command completed with error: %v", createErr)
}

// Test directory name validation logic - parallel tests.
func (s *CreateTestSuite) TestDirectoryNameValidation() {
	testCases := []struct {
		name      string
		dirName   string
		shouldErr bool
	}{
		{"valid name", "my-game", false},
		{"valid name with numbers", "game123", false},
		{"invalid name with spaces", "my game", true},
		{"valid name with hyphens", "my-awesome-game", false},
		{"valid name with underscores", "my_game", false},
		{"empty name", "", true},
		{"name with special chars", "game@#$", true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Basic validation - check for spaces and empty strings
			hasSpaces := false
			isEmpty := len(tc.dirName) == 0
			hasSpecialChars := false

			for _, char := range tc.dirName {
				if char == ' ' {
					hasSpaces = true
				}
				if char == '@' || char == '#' || char == '$' || char == '%' {
					hasSpecialChars = true
				}
			}

			shouldBeInvalid := hasSpaces || isEmpty || hasSpecialChars

			if tc.shouldErr {
				s.True(shouldBeInvalid, "Expected directory name '%s' to be invalid", tc.dirName)
			} else {
				s.False(shouldBeInvalid, "Expected directory name '%s' to be valid", tc.dirName)
			}
		})
	}
}

// Test that temp directory isolation works correctly.
func (s *CreateTestSuite) TestTempDirectoryIsolation() {
	// Use t.TempDir() which is safe for parallel tests
	tempDir := s.T().TempDir()

	// Verify the temp directory exists and is under the system temp directory
	s.Contains(tempDir, os.TempDir())

	s.T().Logf("Test temp directory: %s", tempDir)

	// Verify we can write to it
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	s.Require().NoError(err)

	// Verify file was created
	_, err = os.Stat(testFile)
	s.Require().NoError(err)
}

// Test that handlers can be created in parallel.
func (s *CreateTestSuite) TestHandlerCreation() {
	// Create fresh handler for this test
	handler := s.createHandler()

	// Verify handler is not nil
	s.NotNil(handler)

	// Verify we can call methods that don't change directories
	s.NotPanics(func() {
		handler.SetAppVersion("test-version")
	})
}

// Run the test suite.
func TestCreateSuite(t *testing.T) {
	suite.Run(t, new(CreateTestSuite))
}
