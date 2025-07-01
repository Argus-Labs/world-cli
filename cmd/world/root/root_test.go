package root_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/browser"
	cmdsetup "pkg.world.dev/world-cli/cmd/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/world/root"
)

type RootTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *RootTestSuite) SetupTest() {
	// Only set up shared context - mocks will be created per test for parallel safety
	s.ctx = context.Background()
}

// Test Doctor Command.
func (s *RootTestSuite) TestDoctorCommand() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	// Execute doctor command
	err := handler.Doctor()

	// Assertions - Doctor should always succeed (it just checks dependencies)
	s.Require().NoError(err)
}

// Test Version Command.
func (s *RootTestSuite) TestVersionCommand() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	// Test version without check
	err := handler.Version(false)
	s.Require().NoError(err)

	// Test version with check - might fail due to network, but shouldn't panic
	s.NotPanics(func() {
		handler.Version(true)
	})
}

// Test SetAppVersion.
func (s *RootTestSuite) TestSetAppVersion() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	testVersion := "v2.0.0"

	// We can't directly test the internal state, but we can test that
	// the method doesn't panic
	s.NotPanics(func() {
		handler.SetAppVersion(testVersion)
	})
}

// Test Login Command - Success.
func (s *RootTestSuite) TestLogin_Success() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	// Setup mocks for successful login flow
	loginLink := api.LoginLinkResponse{
		ClientURL:   "https://auth.example.com/login",
		CallBackURL: "https://api.example.com/callback/123",
	}

	//nolint:lll // test lint error
	loginToken := models.LoginToken{
		Status: "success",
		JWT:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiSm9obiBEb2UiLCJpZCI6IjEyMzQ1IiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjk5OTk5OTk5OTksInN1YiI6IjEyMzQ1IiwiZW1haWwiOiJqb2huQGV4YW1wbGUuY29tIn0.invalid", // Mock JWT
	}

	user := models.User{
		ID:        "12345",
		Name:      "John Doe",
		Email:     "john.updated@example.com", // Different from JWT email to trigger UpdateUser
		AvatarURL: "https://example.com/avatar.jpg",
	}

	setupState := models.CommandState{
		Organization: &models.Organization{
			Name: "Test Org",
			Slug: "test-org",
		},
		Project: &models.Project{
			Name:    "Test Project",
			Slug:    "test-project",
			RepoURL: "https://github.com/test/repo",
		},
	}

	// Mock API calls
	mockAPIClient.On("GetLoginLink", mock.Anything).Return(loginLink, nil)
	mockBrowserClient.On("OpenURL", loginLink.ClientURL).Return(nil)
	mockAPIClient.On("GetLoginToken", mock.Anything, loginLink.CallBackURL).Return(loginToken, nil)
	mockAPIClient.On("SetAuthToken", mock.AnythingOfType("string")).Return()
	mockAPIClient.On("GetUser", mock.Anything).Return(user, nil)
	mockAPIClient.On(
		"UpdateUser", mock.Anything, "John Doe", "john.updated@example.com", "https://example.com/avatar.jpg").
		Return(nil)

	// Mock config operations
	configObj := &config.Config{
		Credential:    models.Credential{},
		CurrRepoKnown: true,
	}
	mockConfig.On("GetConfig").Return(configObj)
	mockConfig.On("Save").Return(nil)

	// Mock setup controller
	setupRequest := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedData,
		ProjectRequired:      models.NeedData,
	}
	mockSetupController.On("SetupCommandState", mock.Anything, setupRequest).Return(setupState, nil)

	// Execute login
	err := handler.Login(s.ctx)

	// Assertions
	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockSetupController.AssertExpectations(s.T())
	mockBrowserClient.AssertExpectations(s.T())
}

// Test Login Command - GetLoginLink Fails.
func (s *RootTestSuite) TestLogin_GetLoginLinkFails() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	// Mock API failure
	mockAPIClient.On("GetLoginLink", mock.Anything).Return(api.LoginLinkResponse{}, assert.AnError)

	// Mock config
	configObj := &config.Config{}
	mockConfig.On("GetConfig").Return(configObj)

	// Execute login
	err := handler.Login(s.ctx)

	// Assertions
	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to login")
	mockAPIClient.AssertExpectations(s.T())
}

// Test Login Command - Config Save Fails.
func (s *RootTestSuite) TestLogin_ConfigSaveFails() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	loginLink := api.LoginLinkResponse{
		ClientURL:   "https://auth.example.com/login",
		CallBackURL: "https://api.example.com/callback/123",
	}

	//nolint:lll // test lint error
	loginToken := models.LoginToken{
		Status: "success",
		JWT:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiSm9obiBEb2UiLCJpZCI6IjEyMzQ1IiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjk5OTk5OTk5OTksInN1YiI6IjEyMzQ1IiwiZW1haWwiOiJqb2huQGV4YW1wbGUuY29tIn0.invalid",
	}

	// Mock API calls
	mockAPIClient.On("GetLoginLink", mock.Anything).Return(loginLink, nil)
	mockBrowserClient.On("OpenURL", loginLink.ClientURL).Return(nil)
	mockAPIClient.On("GetLoginToken", mock.Anything, loginLink.CallBackURL).Return(loginToken, nil)
	mockAPIClient.On("SetAuthToken", mock.AnythingOfType("string")).Return()

	// Mock config operations - Save fails
	configObj := &config.Config{
		Credential: models.Credential{},
	}
	mockConfig.On("GetConfig").Return(configObj)
	mockConfig.On("Save").Return(assert.AnError)

	// Execute login
	err := handler.Login(s.ctx)

	// Assertions
	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to save credential")
}

// Test Login Command - GetUser Fails.
func (s *RootTestSuite) TestLogin_GetUserFails() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	loginLink := api.LoginLinkResponse{
		ClientURL:   "https://auth.example.com/login",
		CallBackURL: "https://api.example.com/callback/123",
	}

	//nolint:lll // test lint error
	loginToken := models.LoginToken{
		Status: "success",
		JWT:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiSm9obiBEb2UiLCJpZCI6IjEyMzQ1IiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjk5OTk5OTk5OTksInN1YiI6IjEyMzQ1IiwiZW1haWwiOiJqb2huQGV4YW1wbGUuY29tIn0.invalid",
	}

	// Mock API calls
	mockAPIClient.On("GetLoginLink", mock.Anything).Return(loginLink, nil)
	mockBrowserClient.On("OpenURL", loginLink.ClientURL).Return(nil)
	mockAPIClient.On("GetLoginToken", mock.Anything, loginLink.CallBackURL).Return(loginToken, nil)
	mockAPIClient.On("SetAuthToken", mock.AnythingOfType("string")).Return()
	mockAPIClient.On("GetUser", mock.Anything).Return(models.User{}, assert.AnError)

	// Mock config operations
	configObj := &config.Config{
		Credential: models.Credential{},
	}
	mockConfig.On("GetConfig").Return(configObj)
	mockConfig.On("Save").Return(nil)

	// Execute login
	err := handler.Login(s.ctx)

	// Assertions
	s.Require().Error(err)
}

// Test Login Command - SetupCommandState Fails.
func (s *RootTestSuite) TestLogin_SetupCommandStateFails() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	loginLink := api.LoginLinkResponse{
		ClientURL:   "https://auth.example.com/login",
		CallBackURL: "https://api.example.com/callback/123",
	}

	//nolint:lll // test lint error
	loginToken := models.LoginToken{
		Status: "success",
		JWT:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiSm9obiBEb2UiLCJpZCI6IjEyMzQ1IiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjk5OTk5OTk5OTksInN1YiI6IjEyMzQ1IiwiZW1haWwiOiJqb2huQGV4YW1wbGUuY29tIn0.invalid",
	}

	user := models.User{
		ID:        "12345",
		Name:      "John Doe",
		Email:     "john@example.com",
		AvatarURL: "https://example.com/avatar.jpg",
	}

	// Mock API calls
	mockAPIClient.On("GetLoginLink", mock.Anything).Return(loginLink, nil)
	mockBrowserClient.On("OpenURL", loginLink.ClientURL).Return(nil)
	mockAPIClient.On("GetLoginToken", mock.Anything, loginLink.CallBackURL).Return(loginToken, nil)
	mockAPIClient.On("SetAuthToken", mock.AnythingOfType("string")).Return()
	mockAPIClient.On("GetUser", mock.Anything).Return(user, nil)
	mockAPIClient.On("UpdateUser", mock.Anything, user.Name, user.Email, user.AvatarURL).Return(nil)

	// Mock config operations
	configObj := &config.Config{
		Credential: models.Credential{},
	}
	mockConfig.On("GetConfig").Return(configObj)
	mockConfig.On("Save").Return(nil)

	// Mock setup controller failure
	setupRequest := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedData,
		ProjectRequired:      models.NeedData,
	}
	mockSetupController.On("SetupCommandState", mock.Anything, setupRequest).
		Return(models.CommandState{}, assert.AnError)

	// Execute login
	err := handler.Login(s.ctx)

	// Assertions
	s.Require().Error(err)
	s.Contains(err.Error(), "forge command setup failed")
}

// Test Login Command - Pending Token Status.
func (s *RootTestSuite) TestLogin_PendingTokenStatus() {
	// Create fresh mocks for this test
	mockConfig := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockSetupController := &cmdsetup.MockController{}
	mockBrowserClient := &browser.MockClient{}
	handler := root.NewHandler("v1.0.0", mockConfig, mockAPIClient, mockSetupController, mockBrowserClient)

	loginLink := api.LoginLinkResponse{
		ClientURL:   "https://auth.example.com/login",
		CallBackURL: "https://api.example.com/callback/123",
	}

	pendingToken := models.LoginToken{
		Status: "pending",
	}

	// Mock API calls - return pending status multiple times, then timeout
	mockAPIClient.On("GetLoginLink", mock.Anything).Return(loginLink, nil)
	mockBrowserClient.On("OpenURL", loginLink.ClientURL).Return(nil)
	mockAPIClient.On("GetLoginToken", mock.Anything, loginLink.CallBackURL).Return(pendingToken, nil)

	// Mock config
	configObj := &config.Config{}
	mockConfig.On("GetConfig").Return(configObj)

	// Create a context with short timeout to avoid long test execution
	ctx, cancel := context.WithTimeout(s.ctx, 100*time.Millisecond)
	defer cancel()

	// Execute login
	err := handler.Login(ctx)

	// Assertions - should fail due to timeout/pending status
	s.Error(err)
}

// Run the test suite.
func TestRootSuite(t *testing.T) {
	suite.Run(t, new(RootTestSuite))
}
