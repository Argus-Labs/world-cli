package repo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RepoTestSuite struct {
	suite.Suite
	client ClientInterface
	ctx    context.Context
}

func (s *RepoTestSuite) SetupTest() {
	s.client = NewClient()
	s.ctx = context.Background()
}

func TestRepoSuite(t *testing.T) {
	suite.Run(t, new(RepoTestSuite))
}

func (s *RepoTestSuite) TestNewClient() {
	client := NewClient()
	s.NotNil(client)
	s.Implements((*ClientInterface)(nil), client)
}

func (s *RepoTestSuite) TestFindGitPathAndURL() {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		s.T().Skip("git not available")
	}

	tests := []struct {
		name          string
		setupRepo     func(t *testing.T) string // returns repo dir
		expectedPath  string
		expectedURL   string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid git repo with origin",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()

				// Initialize git repo
				cmd := exec.Command("git", "init")
				cmd.Dir = dir
				require.NoError(t, cmd.Run())

				// Add origin remote
				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
				cmd.Dir = dir
				require.NoError(t, cmd.Run())

				// Create subdirectory
				subDir := filepath.Join(dir, "subdir")
				require.NoError(t, os.MkdirAll(subDir, 0755))

				return subDir
			},
			expectedPath: "subdir",
			expectedURL:  "https://github.com/test/repo",
			expectError:  false,
		},
		{
			name: "git repo at root",
			setupRepo: func(t *testing.T) string {
				dir := t.TempDir()

				// Initialize git repo
				cmd := exec.Command("git", "init")
				cmd.Dir = dir
				require.NoError(t, cmd.Run())

				// Add origin remote
				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
				cmd.Dir = dir
				require.NoError(t, cmd.Run())

				return dir
			},
			expectedPath: "",
			expectedURL:  "https://github.com/test/repo",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		// capture range variable
		s.Run(tt.name, func() {
			// Setup
			originalDir, err := os.Getwd()
			s.Require().NoError(err)
			defer func() {
				s.Require().NoError(os.Chdir(originalDir))
			}()

			repoDir := tt.setupRepo(s.T())
			s.Require().NoError(os.Chdir(repoDir))

			// Test
			path, url, err := s.client.FindGitPathAndURL()

			if tt.expectError {
				s.Require().Error(err)
				if tt.errorContains != "" {
					s.Contains(err.Error(), tt.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Equal(tt.expectedPath, path)
				s.Equal(tt.expectedURL, url)
			}
		})
	}
}

func (s *RepoTestSuite) TestFindGitPathAndURL_NotInGitRepo() {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		s.T().Skip("git not available")
	}

	// Setup - change to temp dir that's not a git repo
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		s.Require().NoError(os.Chdir(originalDir))
	}()

	tempDir := s.T().TempDir()
	s.Require().NoError(os.Chdir(tempDir))

	// Test
	_, _, err = s.client.FindGitPathAndURL()

	s.Require().Error(err)
	s.Contains(err.Error(), "git repo")
}

func (s *RepoTestSuite) TestValidateRepoPath() {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid path",
			path:        "valid/path",
			expectError: false,
		},
		{
			name:        "empty path",
			path:        "",
			expectError: false,
		},
		{
			name:        "path with spaces",
			path:        "invalid path",
			expectError: true,
		},
		{
			name:        "complex valid path",
			path:        "src/main/go",
			expectError: false,
		},
	}

	for _, tt := range tests {
		// capture range variable
		s.Run(tt.name, func() {
			err := s.client.ValidateRepoPath(s.ctx, "https://github.com/test/repo", "token", tt.path)

			if tt.expectError {
				s.Require().Error(err)
				s.Contains(err.Error(), "invalid path")
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *RepoTestSuite) TestValidateRepoToken_WithMockServer() {
	tests := []struct {
		name          string
		provider      string
		repoURL       string
		token         string
		mockResponse  int
		mockBody      string
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid github repo with token",
			provider:     "github",
			repoURL:      "https://github.com/test/repo",
			token:        "valid-token",
			mockResponse: http.StatusOK,
			mockBody:     `{"id": 1, "name": "repo"}`,
			expectError:  false,
		},
		{
			name:         "valid github repo without token",
			provider:     "github",
			repoURL:      "https://github.com/test/repo",
			token:        "",
			mockResponse: http.StatusOK,
			mockBody:     `{"id": 1, "name": "repo"}`,
			expectError:  false,
		},
		{
			name:          "invalid github repo",
			provider:      "github",
			repoURL:       "https://github.com/test/repo",
			token:         "invalid-token",
			mockResponse:  http.StatusNotFound,
			mockBody:      `{"message": "Not Found"}`,
			expectError:   true,
			errorContains: "GitHub validation failed",
		},
		{
			name:         "valid gitlab repo",
			provider:     "gitlab",
			repoURL:      "https://gitlab.com/test/repo",
			token:        "valid-token",
			mockResponse: http.StatusOK,
			mockBody:     `{"id": 1, "name": "repo"}`,
			expectError:  false,
		},
		{
			name:         "valid bitbucket repo",
			provider:     "bitbucket",
			repoURL:      "https://bitbucket.org/test/repo",
			token:        "valid-token",
			mockResponse: http.StatusOK,
			mockBody:     `{"name": "repo"}`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		// capture range variable
		s.Run(tt.name, func() {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the correct headers are set
				switch tt.provider {
				case "github":
					if tt.token != "" {
						s.Equal("token "+tt.token, r.Header.Get("Authorization"))
					}
				case "gitlab":
					if tt.token != "" {
						s.Equal(tt.token, r.Header.Get("Private-Token"))
					}
				case "bitbucket":
					if tt.token != "" {
						s.Equal("Bearer "+tt.token, r.Header.Get("Authorization"))
					}
				}

				w.WriteHeader(tt.mockResponse)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			// Test the individual provider validation functions directly
			var err error
			switch tt.provider {
			case "github":
				err = validateGitHub(s.ctx, tt.repoURL, tt.token, server.URL)
			case "gitlab":
				err = validateGitLab(s.ctx, tt.repoURL, tt.token, server.URL)
			case "bitbucket":
				err = validateBitbucket(s.ctx, tt.repoURL, tt.token, server.URL)
			}

			if tt.expectError {
				s.Require().Error(err)
				if tt.errorContains != "" {
					s.Contains(err.Error(), tt.errorContains)
				}
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *RepoTestSuite) TestValidateRepoToken_ProviderRouting() {
	// Test that ValidateRepoToken correctly identifies providers and routes to the right validation function
	// We can't easily test the HTTP calls without complex mocking, but we can test the routing logic

	tests := []struct {
		name          string
		repoURL       string
		expectError   bool
		errorContains string
	}{
		{
			name:          "github URL routes correctly",
			repoURL:       "https://github.com/test/repo",
			expectError:   true, // Will fail HTTP call but should identify GitHub
			errorContains: "GitHub validation failed",
		},
		{
			name:          "gitlab URL routes correctly",
			repoURL:       "https://gitlab.com/test/repo",
			expectError:   true, // Will fail HTTP call but should identify GitLab
			errorContains: "GitLab validation failed",
		},
		{
			name:          "bitbucket URL routes correctly",
			repoURL:       "https://bitbucket.org/test/repo",
			expectError:   true, // Will fail HTTP call but should identify Bitbucket
			errorContains: "bitbucket validation failed",
		},
	}

	for _, tt := range tests {
		// capture range variable
		s.Run(tt.name, func() {
			err := s.client.ValidateRepoToken(s.ctx, tt.repoURL, "fake-token")

			if tt.expectError {
				s.Require().Error(err)
				s.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *RepoTestSuite) TestIdentifyProvider() {
	tests := []struct {
		name             string
		repoURL          string
		expectedProvider string
		expectedAPIURL   string
		expectError      bool
	}{
		{
			name:             "github.com",
			repoURL:          "https://github.com/user/repo",
			expectedProvider: "GitHub",
			expectedAPIURL:   "https://api.github.com",
			expectError:      false,
		},
		{
			name:             "gitlab.com",
			repoURL:          "https://gitlab.com/user/repo",
			expectedProvider: "GitLab",
			expectedAPIURL:   "https://gitlab.com/api/v4",
			expectError:      false,
		},
		{
			name:             "bitbucket.org",
			repoURL:          "https://bitbucket.org/user/repo",
			expectedProvider: "Bitbucket",
			expectedAPIURL:   "https://api.bitbucket.org/2.0",
			expectError:      false,
		},
		{
			name:        "unknown provider",
			repoURL:     "https://example.com/user/repo",
			expectError: true,
		},
		{
			name:        "invalid URL",
			repoURL:     "not-a-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		s.Run(tt.name, func() {
			provider, apiURL, err := identifyProvider(tt.repoURL)

			if tt.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(tt.expectedProvider, provider)
				s.Equal(tt.expectedAPIURL, apiURL)
			}
		})
	}
}

func (s *RepoTestSuite) TestReplaceLast() {
	tests := []struct {
		name     string
		input    string
		old      string
		new      string
		expected string
	}{
		{
			name:     "replace .git suffix",
			input:    "https://github.com/user/repo.git",
			old:      ".git",
			new:      "",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "no match",
			input:    "https://github.com/user/repo",
			old:      ".git",
			new:      "",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "multiple matches - replace last",
			input:    "test.git.git",
			old:      ".git",
			new:      "",
			expected: "test.git",
		},
	}

	for _, tt := range tests {
		// capture range variable
		s.Run(tt.name, func() {
			result := replaceLast(tt.input, tt.old, tt.new)
			s.Equal(tt.expected, result)
		})
	}
}

// Test that mock implements interface (for other tests that use the mock).
func (s *RepoTestSuite) TestMockImplementsInterface() {
	mockClient := &MockClient{}
	s.Implements((*ClientInterface)(nil), mockClient)
}
