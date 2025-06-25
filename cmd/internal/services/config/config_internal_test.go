package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	commonConfig "pkg.world.dev/world-cli/common/config"
)

type ConfigTestSuite struct {
	suite.Suite
	tempDir    string
	origGetDir func() (string, error)
}

func (s *ConfigTestSuite) SetupTest() {
	s.tempDir = s.T().TempDir()

	// Mock commonConfig.GetCLIConfigDir to return our temp directory
	s.origGetDir = commonConfig.GetCLIConfigDir
	//nolint:reassign // test code
	commonConfig.GetCLIConfigDir = func() (string, error) {
		return s.tempDir, nil
	}
}

func (s *ConfigTestSuite) TearDownTest() {
	// Restore original function
	//nolint:reassign // test code
	commonConfig.GetCLIConfigDir = s.origGetDir
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) TestNewService() {
	service, err := NewService(EnvDev)
	s.Require().NoError(err)
	s.NotNil(service)
	s.Implements((*ServiceInterface)(nil), service)

	realService := service.(*Service)
	s.Equal(EnvDev, realService.Env)
}

func (s *ConfigTestSuite) TestGetForgeConfig_NoFile() {
	// Test the REAL Client implementation
	service, err := NewService(EnvDev)
	s.Require().NoError(err)

	config := service.GetConfig()
	s.Empty(config.Credential.Token)
	s.Empty(config.KnownProjects)
	s.False(config.CurrRepoKnown)
}

func (s *ConfigTestSuite) TestGetForgeConfig_WithFile() {
	// Create a real config file in the mocked config directory
	testConfig := Config{
		OrganizationID: "org-123",
		ProjectID:      "proj-456",
		Credential: Credential{
			Token: "test-token",
			ID:    "user-123",
			Name:  "Test User",
			Email: "test@example.com",
		},
		KnownProjects: []KnownProject{
			{
				ProjectID:      "proj-789",
				ProjectName:    "test-project",
				OrganizationID: "org-123",
				RepoURL:        "https://github.com/test/repo",
				RepoPath:       "test/path",
			},
		},
	}

	// Save it using the real file naming logic
	configFile := filepath.Join(s.tempDir, "dev-forge-config.json")
	data, err := json.Marshal(testConfig)
	s.Require().NoError(err)
	err = os.WriteFile(configFile, data, 0600)
	s.Require().NoError(err)

	// Test the REAL Client.GetForgeConfig() method
	service, err := NewService(EnvDev)
	s.Require().NoError(err)
	config := service.GetConfig()

	// Verify it read the real file correctly
	s.Equal("test-token", config.Credential.Token)
	s.Equal("org-123", config.OrganizationID)
	s.Equal("proj-456", config.ProjectID)
	s.Len(config.KnownProjects, 1)
	s.Equal("test-project", config.KnownProjects[0].ProjectName)

	// Verify runtime fields are reset (line 54-57 in real implementation)
	s.False(config.CurrRepoKnown)
	s.Empty(config.CurrRepoURL)
	s.Empty(config.CurrRepoPath)
	s.Empty(config.CurrProjectName)
}

func (s *ConfigTestSuite) TestSave() {
	testConfig := Config{
		OrganizationID: "org-123",
		Credential: Credential{
			Token: "test-token",
		},
	}

	// Test the REAL Client.Save() method
	service, err := NewService(EnvDev)
	s.Require().NoError(err)

	service.(*Service).Config = testConfig
	err = service.Save()
	s.Require().NoError(err)

	// Verify the real implementation created the file with correct name
	configFile := filepath.Join(s.tempDir, "dev-forge-config.json")
	s.FileExists(configFile)

	// Verify content was marshaled correctly
	data, err := os.ReadFile(configFile)
	s.Require().NoError(err)

	var savedConfig Config
	err = json.Unmarshal(data, &savedConfig)
	s.Require().NoError(err)
	s.Equal("test-token", savedConfig.Credential.Token)
	s.Equal("org-123", savedConfig.OrganizationID)
}

func (s *ConfigTestSuite) TestGetConfigFileName() {
	tests := []struct {
		name         string
		env          string
		expectedFile string
	}{
		{
			name:         "production config",
			env:          EnvProd,
			expectedFile: "forge-config.json",
		},
		{
			name:         "development config",
			env:          EnvDev,
			expectedFile: "dev-forge-config.json",
		},
		{
			name:         "local config",
			env:          EnvLocal,
			expectedFile: "local-forge-config.json",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			service, err := NewService(tt.env)
			s.Require().NoError(err)
			filename, err := service.(*Service).getConfigFileName()
			s.Require().NoError(err)
			s.Contains(filename, tt.expectedFile)
			s.Equal(filepath.Join(s.tempDir, tt.expectedFile), filename)
		})
	}
}

func (s *ConfigTestSuite) TestMockImplementsInterface() {
	mockService := &MockService{}
	s.Implements((*ServiceInterface)(nil), mockService)
}
