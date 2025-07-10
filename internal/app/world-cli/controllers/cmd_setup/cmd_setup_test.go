package cmdsetup_test

import (
	"context"
	"testing"
	"time"

	"github.com/rotisserie/eris"
	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/internal/app/world-cli/clients/api"
	"pkg.world.dev/world-cli/internal/app/world-cli/clients/repo"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/organization"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/project"
	cmdsetup "pkg.world.dev/world-cli/internal/app/world-cli/controllers/cmd_setup"
	"pkg.world.dev/world-cli/internal/app/world-cli/interfaces"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
	"pkg.world.dev/world-cli/internal/app/world-cli/services/config"
	"pkg.world.dev/world-cli/internal/app/world-cli/services/input"
)

// SetupCommandSuite is a test suite for the SetupCommand controller.
type SetupCommandSuite struct {
	suite.Suite
}

// Helper method to create fresh mocks and controller for each test.
func (s *SetupCommandSuite) createTestController() (
	interfaces.CommandSetupController,
	*api.MockClient,
	*config.MockService,
	*input.MockService,
	*repo.MockClient,
	*organization.MockHandler,
	*project.MockHandler,
) {
	mockAPI := &api.MockClient{}
	mockConfig := &config.MockService{}
	mockInput := &input.MockService{}
	mockRepo := &repo.MockClient{}
	mockOrgHandler := &organization.MockHandler{}
	mockProjectHandler := &project.MockHandler{}

	controller := cmdsetup.NewController(
		mockConfig,
		mockRepo,
		mockOrgHandler,
		mockProjectHandler,
		mockAPI,
		mockInput,
	)

	return controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler
}

// TestSetupCommandSuite runs the test suite.
func TestSetupCommandSuite(t *testing.T) {
	suite.Run(t, new(SetupCommandSuite))
}

// TestLoginScenarios tests various login scenarios.
func (s *SetupCommandSuite) TestLoginScenarios() {
	// s.T().Parallel()

	testCases := []struct {
		name           string
		loginRequired  models.LoginRequirement
		hasToken       bool
		tokenExpired   bool
		expectError    bool
		errorContains  string
		expectLoggedIn bool
	}{
		{
			name:           "Ignore login - should succeed",
			loginRequired:  models.IgnoreLogin,
			hasToken:       false,
			tokenExpired:   false,
			expectError:    false,
			expectLoggedIn: false,
		},
		{
			name:          "Need login - no token - should fail",
			loginRequired: models.NeedLogin,
			hasToken:      false,
			tokenExpired:  false,
			expectError:   true,
			errorContains: "not logged in",
		},
		{
			name:           "Need login - valid token - should succeed",
			loginRequired:  models.NeedLogin,
			hasToken:       true,
			tokenExpired:   false,
			expectError:    false,
			expectLoggedIn: true,
		},
		{
			name:          "Need login - expired token - should fail",
			loginRequired: models.NeedLogin,
			hasToken:      true,
			tokenExpired:  true,
			expectError:   true,
			errorContains: "not logged in",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
			ctx := context.Background()

			cfg := &config.Config{}
			if tc.hasToken {
				if tc.tokenExpired {
					cfg.Credential.TokenExpiresAt = time.Now().Add(-time.Hour)
				} else {
					cfg.Credential.TokenExpiresAt = time.Now().Add(time.Hour)
				}
				cfg.Credential.Token = "valid-token"
			}

			mockConfig.On("GetConfig").Return(cfg)
			mockRepo.On("FindGitPathAndURL").Return("", "", nil)

			if tc.hasToken && !tc.tokenExpired {
				user := models.User{ID: "user-123"}
				mockAPI.On("GetUser", ctx).Return(user, nil)
				mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
			}

			if !tc.expectError {
				mockConfig.On("Save").Return(nil)
			}

			req := models.SetupRequest{
				LoginRequired: tc.loginRequired,
			}

			result, err := controller.SetupCommandState(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				if tc.errorContains != "" {
					s.Require().Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(result)
				s.Require().Equal(tc.expectLoggedIn, result.LoggedIn)
			}

			mockAPI.AssertExpectations(s.T())
			mockConfig.AssertExpectations(s.T())
			mockInput.AssertExpectations(s.T())
			mockRepo.AssertExpectations(s.T())
			mockOrgHandler.AssertExpectations(s.T())
			mockProjectHandler.AssertExpectations(s.T())
		})
	}
}

// TestHandleOrganizationInvitationsAcceptInvitation tests accepting organization invitations.
func (s *SetupCommandSuite) TestHandleOrganizationInvitationsAcceptInvitation() {
	// s.T().Parallel()
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
	}
	user := models.User{ID: "user-123"}
	orgs := []models.Organization{
		{ID: "org-123", Name: "Test Org", Slug: "test-org"},
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return(orgs, nil)
	mockInput.On("Confirm", ctx, "Would you like to join? [Y/n]", "Y").Return(true, nil)
	mockAPI.On("AcceptOrganizationInvitation", ctx, "org-123").Return(nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired: models.NeedLogin,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestHandleOrganizationInvitationsDeclineInvitation tests declining organization invitations.
func (s *SetupCommandSuite) TestHandleOrganizationInvitationsDeclineInvitation() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
	}
	user := models.User{ID: "user-123"}
	orgs := []models.Organization{
		{ID: "org-123", Name: "Test Org", Slug: "test-org"},
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return(orgs, nil)
	mockInput.On("Confirm", ctx, "Would you like to join? [Y/n]", "Y").Return(false, nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired: models.NeedLogin,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestRepoLookupSuccess tests successful repository lookup.
func (s *SetupCommandSuite) TestRepoLookupSuccess() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
		CurrRepoURL:  "https://github.com/test/repo",
		CurrRepoPath: "path",
		KnownProjects: []config.KnownProject{
			{
				ProjectID:      "proj-789",
				ProjectName:    "Found Project",
				OrganizationID: "org-789",
				RepoURL:        "https://github.com/test/repo",
				RepoPath:       "path",
			},
		},
	}
	user := models.User{ID: "user-123"}
	foundProject := models.Project{
		ID:    "proj-789",
		Name:  "Found Project",
		OrgID: "org-789",
	}
	foundOrg := models.Organization{
		ID:   "org-789",
		Name: "Found Org",
		Slug: "found-org",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("path", "https://github.com/test/repo", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
	mockAPI.On("GetOrganizationByID", ctx, "org-789").Return(foundOrg, nil)
	mockAPI.On("GetProjectByID", ctx, "org-789", "proj-789").Return(foundProject, nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired:   models.NeedLogin,
		ProjectRequired: models.NeedRepoLookup,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().True(result.LoggedIn)
	s.Require().NotNil(result.Organization)
	s.Require().NotNil(result.Project)
	s.Require().Equal("org-789", result.Organization.ID)
	s.Require().Equal("proj-789", result.Project.ID)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestRepoLookupNotLoggedIn tests repository lookup when user is not logged in.
func (s *SetupCommandSuite) TestRepoLookupNotLoggedIn() {
	// s.T().Parallel()
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	cfg := &config.Config{
		CurrRepoURL:  "https://github.com/test/repo",
		CurrRepoPath: "path",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("path", "https://github.com/test/repo", nil)

	req := models.SetupRequest{
		LoginRequired:   models.IgnoreLogin,
		ProjectRequired: models.NeedRepoLookup,
	}

	_, err := controller.SetupCommandState(ctx, req)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not logged in, can't lookup project from git repo")

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestNeedOrgDataNoOrgsCreateNew tests creating a new organization when none exist.
func (s *SetupCommandSuite) TestNeedOrgDataNoOrgsCreateNew() {
	// s.T().Parallel()
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
	}
	user := models.User{ID: "user-123"}
	newOrg := models.Organization{
		ID:   "new-org-123",
		Name: "New Org",
		Slug: "new-org",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
	mockAPI.On("GetOrganizations", ctx).Return([]models.Organization{}, nil)
	mockInput.On("Confirm", ctx, "Would you like to create one? (Y/n)", "Y").Return(true, nil)
	mockOrgHandler.On("Create", ctx, models.CreateOrganizationFlags{}).
		Return(newOrg, nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedData,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.Organization)
	s.Require().Equal("new-org-123", result.Organization.ID)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestNeedOrgDataOneOrgUseExisting tests using an existing organization when only one exists.
func (s *SetupCommandSuite) TestNeedOrgDataOneOrgUseExisting() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
	}
	user := models.User{ID: "user-123"}
	existingOrg := models.Organization{
		ID:   "existing-org-123",
		Name: "Existing Org",
		Slug: "existing-org",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
	mockAPI.On("GetOrganizations", ctx).Return([]models.Organization{existingOrg}, nil)
	mockInput.On("Prompt", ctx, "Use this organization? (Y/n/c to create new)", "Y").Return("Y", nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedData,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.Organization)
	s.Require().Equal("existing-org-123", result.Organization.ID)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestNeedOrgDataMultipleOrgs tests handling multiple organizations.
func (s *SetupCommandSuite) TestNeedOrgDataMultipleOrgs() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
	}
	user := models.User{ID: "user-123"}
	orgs := []models.Organization{
		{ID: "org-1", Name: "Org 1", Slug: "org-1"},
		{ID: "org-2", Name: "Org 2", Slug: "org-2"},
	}
	selectedOrg := orgs[0]

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
	mockAPI.On("GetOrganizations", ctx).Return(orgs, nil)
	mockOrgHandler.On("PromptForSwitch", ctx, orgs, true).
		Return(selectedOrg, nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedData,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.Organization)
	s.Require().Equal("org-1", result.Organization.ID)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestNeedProjectDataNoProjectsCreateNew tests creating a new project when none exist.
func (s *SetupCommandSuite) TestNeedProjectDataNoProjectsCreateNew() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
		OrganizationID: "org-123",
	}
	user := models.User{ID: "user-123"}
	newProject := models.Project{
		ID:    "new-proj-123",
		Name:  "New Project",
		OrgID: "org-123",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
	mockAPI.On("GetProjects", ctx, "org-123").Return([]models.Project{}, nil)
	mockProjectHandler.On("PreCreateUpdateValidation", true).Return("", "", nil)
	mockInput.On("Prompt", ctx, "Would you like to create a new project? (Y/n)", "Y").Return("Y", nil)
	mockProjectHandler.On("Create", ctx, models.Organization{ID: "org-123"}, models.CreateProjectFlags{}).
		Return(newProject, nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedIDOnly,
		ProjectRequired:      models.NeedData,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.Project)
	s.Require().Equal("new-proj-123", result.Project.ID)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestNeedProjectDataOneProjectUseExisting tests using an existing project when only one exists.
func (s *SetupCommandSuite) TestNeedProjectDataOneProjectUseExisting() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
		OrganizationID: "org-123",
	}
	user := models.User{ID: "user-123"}
	existingProject := models.Project{
		ID:    "existing-proj-123",
		Name:  "Existing Project",
		Slug:  "existing-project",
		OrgID: "org-123",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
	mockAPI.On("GetProjects", ctx, "org-123").Return([]models.Project{existingProject}, nil)
	mockProjectHandler.On("PreCreateUpdateValidation", false).Return("", "", repo.ErrNotInGitRepository)
	mockInput.On("Prompt", ctx, "Select project: Existing Project [existing-project]? (Y/n)", "Y").Return("Y", nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedIDOnly,
		ProjectRequired:      models.NeedData,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.Project)
	s.Require().Equal("existing-proj-123", result.Project.ID)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestMustNotExistOrganizationExists tests the scenario where organization must not exist but one does.
func (s *SetupCommandSuite) TestMustNotExistOrganizationExists() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	cfg := &config.Config{
		OrganizationID: "org-123", // Has organization
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)

	req := models.SetupRequest{
		OrganizationRequired: models.MustNotExist,
	}

	_, err := controller.SetupCommandState(ctx, req)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "organization already exists")

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestMustNotExistProjectExists tests the scenario where project must not exist but one does.
func (s *SetupCommandSuite) TestMustNotExistProjectExists() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	cfg := &config.Config{
		ProjectID: "proj-123", // Has project
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)

	req := models.SetupRequest{
		ProjectRequired: models.MustNotExist,
	}

	_, err := controller.SetupCommandState(ctx, req)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "project already exists")

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestConfigSaveError tests handling of configuration save errors.
func (s *SetupCommandSuite) TestConfigSaveError() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	cfg := &config.Config{}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockConfig.On("Save").Return(config.ErrCannotSaveConfig)

	req := models.SetupRequest{
		LoginRequired: models.IgnoreLogin,
	}

	_, err := controller.SetupCommandState(ctx, req)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to save config after setup")

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestIDOnlySuccess tests the scenario where only IDs are needed and are available.
func (s *SetupCommandSuite) TestIDOnlySuccess() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	cfg := &config.Config{
		OrganizationID:  "org-123",
		ProjectID:       "proj-456",
		CurrProjectName: "Test Project",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired:        models.IgnoreLogin,
		OrganizationRequired: models.NeedIDOnly,
		ProjectRequired:      models.NeedIDOnly,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().NotNil(result.Organization)
	s.Require().NotNil(result.Project)
	s.Require().Equal("org-123", result.Organization.ID)
	s.Require().Equal("proj-456", result.Project.ID)
	s.Require().Equal("Test Project", result.Project.Name)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestRepoLookupNewProjectDiscovered tests discovering a new project from repository lookup.
func (s *SetupCommandSuite) TestRepoLookupNewProjectDiscovered() {
	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
	ctx := context.Background()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
		CurrRepoURL:  "https://github.com/test/repo",
		CurrRepoPath: "path",
		// No KnownProjects initially - this is a new discovery
	}
	user := models.User{ID: "user-123"}
	foundProject := models.Project{
		ID:    "proj-789",
		Name:  "Found Project",
		OrgID: "org-789",
	}
	foundOrg := models.Organization{
		ID:   "org-789",
		Name: "Found Org",
		Slug: "found-org",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("path", "https://github.com/test/repo", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
	mockAPI.On("LookupProjectFromRepo", ctx, "https://github.com/test/repo", "path").Return(foundProject, nil)
	mockConfig.On("AddKnownProject", "proj-789", "Found Project", "org-789", "https://github.com/test/repo", "path")
	// After discovery, the service will call inKnownRepo which needs these API calls
	mockAPI.On("GetOrganizationByID", ctx, "org-789").Return(foundOrg, nil)
	mockAPI.On("GetProjectByID", ctx, "org-789", "proj-789").Return(foundProject, nil)
	mockConfig.On("Save").Return(nil).Twice() // Once for AddKnownProject, once at the end

	req := models.SetupRequest{
		LoginRequired:   models.NeedLogin,
		ProjectRequired: models.NeedRepoLookup,
	}

	result, err := controller.SetupCommandState(ctx, req)

	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().True(result.LoggedIn)
	s.Require().NotNil(result.Organization)
	s.Require().NotNil(result.Project)
	s.Require().Equal("org-789", result.Organization.ID)
	s.Require().Equal("proj-789", result.Project.ID)

	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockRepo.AssertExpectations(s.T())
	mockOrgHandler.AssertExpectations(s.T())
	mockProjectHandler.AssertExpectations(s.T())
}

// TestOrganizationInvitationScenarios tests organization invitation handling.
func (s *SetupCommandSuite) TestOrganizationInvitationScenarios() {
	// s.T().Parallel()

	testCases := []struct {
		name         string
		invitations  []models.Organization
		userChoice   string
		expectAccept bool
		expectError  bool
	}{
		{
			name:        "No invitations - should continue",
			invitations: []models.Organization{},
		},
		{
			name:         "Accept invitation",
			invitations:  []models.Organization{{ID: "org-123", Name: "Test Org", Slug: "test-org"}},
			userChoice:   "Y",
			expectAccept: true,
		},
		{
			name:        "Decline invitation",
			invitations: []models.Organization{{ID: "org-123", Name: "Test Org", Slug: "test-org"}},
			userChoice:  "n",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
			ctx := context.Background()

			futureTime := time.Now().Add(time.Hour)
			cfg := &config.Config{
				Credential: models.Credential{
					Token:          "valid-token",
					TokenExpiresAt: futureTime,
				},
			}
			user := models.User{ID: "user-123"}

			mockConfig.On("GetConfig").Return(cfg)
			mockRepo.On("FindGitPathAndURL").Return("", "", nil)
			mockAPI.On("GetUser", ctx).Return(user, nil)
			mockAPI.On("GetOrganizationsInvitedTo", ctx).Return(tc.invitations, nil)

			if len(tc.invitations) > 0 {
				mockInput.On("Confirm", ctx, "Would you like to join? [Y/n]", "Y").Return(tc.userChoice == "Y", nil)
				if tc.expectAccept {
					mockAPI.On("AcceptOrganizationInvitation", ctx, "org-123").Return(nil)
				}
			}

			mockConfig.On("Save").Return(nil)

			req := models.SetupRequest{
				LoginRequired: models.NeedLogin,
			}

			result, err := controller.SetupCommandState(ctx, req)

			s.Require().NoError(err)
			s.Require().NotNil(result)

			mockAPI.AssertExpectations(s.T())
			mockConfig.AssertExpectations(s.T())
			mockInput.AssertExpectations(s.T())
			mockRepo.AssertExpectations(s.T())
			mockOrgHandler.AssertExpectations(s.T())
			mockProjectHandler.AssertExpectations(s.T())
		})
	}
}

// TestOrganizationDataScenarios tests organization data handling scenarios.
func (s *SetupCommandSuite) TestOrganizationDataScenarios() {
	// s.T().Parallel()

	testCases := []struct {
		name          string
		orgRequired   models.SetupRequirement
		existingOrgs  []models.Organization
		userChoice    string
		expectCreate  bool
		expectError   bool
		errorContains string
		expectedOrgID string
	}{
		{
			name:          "No orgs required - should succeed",
			orgRequired:   models.Ignore,
			expectedOrgID: "",
		},
		{
			name:          "Need data - no orgs - create new",
			orgRequired:   models.NeedData,
			existingOrgs:  []models.Organization{},
			userChoice:    "Y",
			expectCreate:  true,
			expectedOrgID: "new-org-123",
		},
		{
			name:          "Need data - one org - use existing",
			orgRequired:   models.NeedData,
			existingOrgs:  []models.Organization{{ID: "existing-org-123", Name: "Existing Org", Slug: "existing-org"}},
			userChoice:    "Y",
			expectedOrgID: "existing-org-123",
		},
		{
			name:          "Need data - one org - user cancels",
			orgRequired:   models.NeedData,
			existingOrgs:  []models.Organization{{ID: "existing-org-123", Name: "Existing Org", Slug: "existing-org"}},
			userChoice:    "n",
			expectError:   true,
			errorContains: "Organization selection canceled",
		},
		{
			name:          "Need data - one org - user creates new",
			orgRequired:   models.NeedData,
			existingOrgs:  []models.Organization{{ID: "existing-org-123", Name: "Existing Org", Slug: "existing-org"}},
			userChoice:    "c",
			expectCreate:  true,
			expectedOrgID: "new-org-456",
		},
		{
			name:        "Need data - multiple orgs - use switch",
			orgRequired: models.NeedData,
			existingOrgs: []models.Organization{
				{ID: "org-1", Name: "Org 1", Slug: "org-1"},
				{ID: "org-2", Name: "Org 2", Slug: "org-2"},
			},
			expectedOrgID: "org-1",
		},
		{
			name:          "Must not exist - org exists - should fail",
			orgRequired:   models.MustNotExist,
			existingOrgs:  []models.Organization{},
			expectError:   true,
			errorContains: "organization already exists",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
			ctx := context.Background()

			futureTime := time.Now().Add(time.Hour)
			cfg := &config.Config{
				Credential: models.Credential{
					Token:          "valid-token",
					TokenExpiresAt: futureTime,
				},
			}

			// Set organization ID for MustNotExist test
			if tc.orgRequired == models.MustNotExist {
				cfg.OrganizationID = "org-123"
			}

			user := models.User{ID: "user-123"}

			mockConfig.On("GetConfig").Return(cfg)
			mockRepo.On("FindGitPathAndURL").Return("", "", nil)
			mockAPI.On("GetUser", ctx).Return(user, nil)
			mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)

			//nolint:nestif // this is a test
			if tc.orgRequired == models.NeedData {
				mockAPI.On("GetOrganizations", ctx).Return(tc.existingOrgs, nil)

				//nolint:gocritic // this is a test
				if len(tc.existingOrgs) == 0 {
					mockInput.On("Confirm", ctx, "Would you like to create one? (Y/n)", "Y").
						Return(tc.userChoice == "Y", nil)
					if tc.expectCreate {
						newOrg := models.Organization{ID: "new-org-123", Name: "New Org", Slug: "new-org"}
						mockOrgHandler.On("Create", ctx, models.CreateOrganizationFlags{}).Return(newOrg, nil)
					}
				} else if len(tc.existingOrgs) == 1 {
					mockInput.On("Prompt", ctx, "Use this organization? (Y/n/c to create new)", "Y").Return(tc.userChoice, nil)
					if tc.userChoice == "c" {
						newOrg := models.Organization{ID: "new-org-456", Name: "New Org", Slug: "new-org"}
						mockOrgHandler.On("Create", ctx, models.CreateOrganizationFlags{}).Return(newOrg, nil)
					}
				} else {
					selectedOrg := tc.existingOrgs[0]
					mockOrgHandler.On("PromptForSwitch", ctx, tc.existingOrgs, true).Return(selectedOrg, nil)
				}
			}

			if !tc.expectError {
				mockConfig.On("Save").Return(nil)
			}

			req := models.SetupRequest{
				LoginRequired:        models.NeedLogin,
				OrganizationRequired: tc.orgRequired,
			}

			result, err := controller.SetupCommandState(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				if tc.errorContains != "" {
					s.Require().Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(result)
				if tc.expectedOrgID != "" {
					s.Require().NotNil(result.Organization)
					s.Require().Equal(tc.expectedOrgID, result.Organization.ID)
				}
			}

			mockAPI.AssertExpectations(s.T())
			mockConfig.AssertExpectations(s.T())
			mockInput.AssertExpectations(s.T())
			mockRepo.AssertExpectations(s.T())
			mockOrgHandler.AssertExpectations(s.T())
			mockProjectHandler.AssertExpectations(s.T())
		})
	}
}

// TestProjectDataScenarios tests project data handling scenarios.
func (s *SetupCommandSuite) TestProjectDataScenarios() {
	// s.T().Parallel()

	testCases := []struct {
		name              string
		projectRequired   models.SetupRequirement
		existingProjects  []models.Project
		userChoice        string
		inRepoRoot        bool
		expectCreate      bool
		expectError       bool
		errorContains     string
		expectedProjectID string
	}{
		{
			name:              "No projects required - should succeed",
			projectRequired:   models.Ignore,
			expectedProjectID: "",
		},
		{
			name:              "Need data - no projects - create new",
			projectRequired:   models.NeedData,
			existingProjects:  []models.Project{},
			userChoice:        "Y",
			inRepoRoot:        true,
			expectCreate:      true,
			expectedProjectID: "new-proj-123",
		},
		{
			name:             "Need data - no projects - user cancels",
			projectRequired:  models.NeedData,
			existingProjects: []models.Project{},
			userChoice:       "n",
			inRepoRoot:       true,
			expectError:      true,
			errorContains:    "Project creation canceled",
		},
		{
			name:            "Need data - one project - use existing",
			projectRequired: models.NeedData,
			existingProjects: []models.Project{
				{ID: "existing-proj-123", Name: "Existing Project", Slug: "existing-project", OrgID: "org-123"},
			},
			userChoice:        "Y",
			inRepoRoot:        false,
			expectedProjectID: "existing-proj-123",
		},
		{
			name:            "Need data - one project - user cancels",
			projectRequired: models.NeedData,
			existingProjects: []models.Project{
				{ID: "existing-proj-123", Name: "Existing Project", Slug: "existing-project", OrgID: "org-123"},
			},
			userChoice:    "n",
			inRepoRoot:    false,
			expectError:   true,
			errorContains: "Project selection canceled",
		},
		{
			name:            "Need data - one project - user creates new (in repo root)",
			projectRequired: models.NeedData,
			existingProjects: []models.Project{
				{ID: "existing-proj-123", Name: "Existing Project", Slug: "existing-project", OrgID: "org-123"},
			},
			userChoice:        "c",
			inRepoRoot:        true,
			expectCreate:      true,
			expectedProjectID: "new-proj-456",
		},
		{
			name:            "Need data - multiple projects - use switch",
			projectRequired: models.NeedData,
			existingProjects: []models.Project{
				{ID: "proj-1", Name: "Project 1", Slug: "project-1", OrgID: "org-123"},
				{ID: "proj-2", Name: "Project 2", Slug: "project-2", OrgID: "org-123"},
			},
			expectedProjectID: "proj-1",
		},
		{
			name:             "Must not exist - project exists - should fail",
			projectRequired:  models.MustNotExist,
			existingProjects: []models.Project{},
			expectError:      true,
			errorContains:    "project already exists",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
			ctx := context.Background()

			futureTime := time.Now().Add(time.Hour)
			cfg := &config.Config{
				Credential: models.Credential{
					Token:          "valid-token",
					TokenExpiresAt: futureTime,
				},
				OrganizationID: "org-123",
			}

			// Set project ID for MustNotExist test
			if tc.projectRequired == models.MustNotExist {
				cfg.ProjectID = "proj-123"
			}

			user := models.User{ID: "user-123"}

			mockConfig.On("GetConfig").Return(cfg)
			mockRepo.On("FindGitPathAndURL").Return("", "", nil)
			mockAPI.On("GetUser", ctx).Return(user, nil)
			mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)

			//nolint:nestif // this is a test
			if tc.projectRequired == models.NeedData {
				mockAPI.On("GetProjects", ctx, "org-123").Return(tc.existingProjects, nil)

				//nolint:gocritic // this is a test
				if len(tc.existingProjects) == 0 {
					mockProjectHandler.On("PreCreateUpdateValidation", true).Return("", "", nil)
					mockInput.On("Prompt", ctx, "Would you like to create a new project? (Y/n)", "Y").
						Return(tc.userChoice, nil)
					if tc.expectCreate {
						newProject := models.Project{ID: "new-proj-123", Name: "New Project", OrgID: "org-123"}
						mockProjectHandler.On("Create", ctx, models.Organization{ID: "org-123"}, models.CreateProjectFlags{}).
							Return(newProject, nil)
					}
				} else if len(tc.existingProjects) == 1 {
					if tc.inRepoRoot {
						mockProjectHandler.On("PreCreateUpdateValidation", false).Return("", "", nil)
						prompt := "Select project: Existing Project [existing-project]? (Y/n/c to create new)"
						mockInput.On("Prompt", ctx, prompt, "Y").Return(tc.userChoice, nil)
						if tc.userChoice == "c" {
							newProject := models.Project{ID: "new-proj-456", Name: "New Project", OrgID: "org-123"}
							mockProjectHandler.On(
								"Create",
								ctx,
								models.Organization{ID: "org-123"},
								models.CreateProjectFlags{},
							).Return(newProject, nil)
						}
					} else {
						mockProjectHandler.On("PreCreateUpdateValidation", false).Return("", "", repo.ErrNotInGitRepository)
						prompt := "Select project: Existing Project [existing-project]? (Y/n)"
						mockInput.On("Prompt", ctx, prompt, "Y").Return(tc.userChoice, nil)
					}
				} else {
					selectedProject := tc.existingProjects[0]
					mockProjectHandler.On(
						"Switch",
						ctx,
						models.SwitchProjectFlags{},
						models.Organization{ID: "org-123"},
						true,
					).Return(selectedProject, nil)
				}
			}

			if !tc.expectError {
				mockConfig.On("Save").Return(nil)
			}

			req := models.SetupRequest{
				LoginRequired:        models.NeedLogin,
				OrganizationRequired: models.NeedIDOnly,
				ProjectRequired:      tc.projectRequired,
			}

			result, err := controller.SetupCommandState(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				if tc.errorContains != "" {
					s.Require().Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(result)
				if tc.expectedProjectID != "" {
					s.Require().NotNil(result.Project)
					s.Require().Equal(tc.expectedProjectID, result.Project.ID)
				}
			}

			mockAPI.AssertExpectations(s.T())
			mockConfig.AssertExpectations(s.T())
			mockInput.AssertExpectations(s.T())
			mockRepo.AssertExpectations(s.T())
			mockOrgHandler.AssertExpectations(s.T())
			mockProjectHandler.AssertExpectations(s.T())
		})
	}
}

// TestExistingDataScenarios tests existing data handling scenarios.
func (s *SetupCommandSuite) TestExistingDataScenarios() {
	// s.T().Parallel()

	testCases := []struct {
		name              string
		orgRequired       models.SetupRequirement
		projectRequired   models.SetupRequirement
		existingOrgs      []models.Organization
		existingProjects  []models.Project
		existingOrgID     string
		existingProjectID string
		expectError       bool
		errorContains     string
		expectedOrgID     string
		expectedProjectID string
	}{
		{
			name:          "Need existing org data - no orgs",
			orgRequired:   models.NeedExistingData,
			existingOrgs:  []models.Organization{},
			expectError:   true,
			errorContains: "Organization selection canceled",
		},
		{
			name:          "Need existing org data - one org",
			orgRequired:   models.NeedExistingData,
			existingOrgs:  []models.Organization{{ID: "existing-org-123", Name: "Existing Org", Slug: "existing-org"}},
			expectedOrgID: "existing-org-123",
		},
		{
			name:        "Need existing org data - multiple orgs with existing ID",
			orgRequired: models.NeedExistingData,
			existingOrgs: []models.Organization{
				{ID: "org-1", Name: "Org 1", Slug: "org-1"},
				{ID: "org-2", Name: "Org 2", Slug: "org-2"},
			},
			existingOrgID: "org-1",
			expectedOrgID: "org-1",
		},
		{
			name:        "Need existing org data - multiple orgs without existing ID",
			orgRequired: models.NeedExistingData,
			existingOrgs: []models.Organization{
				{ID: "org-1", Name: "Org 1", Slug: "org-1"},
				{ID: "org-2", Name: "Org 2", Slug: "org-2"},
			},
			expectedOrgID: "org-1",
		},
		{
			name:             "Need existing project data - no projects",
			projectRequired:  models.NeedExistingData,
			existingProjects: []models.Project{},
			expectError:      true,
			errorContains:    "Project selection canceled",
		},
		{
			name:            "Need existing project data - one project",
			projectRequired: models.NeedExistingData,
			existingProjects: []models.Project{
				{ID: "existing-proj-123", Name: "Existing Project", Slug: "existing-project", OrgID: "org-123"},
			},
			expectedProjectID: "existing-proj-123",
		},
		{
			name:            "Need existing project data - multiple projects with existing ID",
			orgRequired:     models.NeedExistingData,
			projectRequired: models.NeedExistingData,
			existingOrgs:    []models.Organization{{ID: "org-123", Name: "Test Org", Slug: "test-org"}},
			existingProjects: []models.Project{
				{ID: "proj-1", Name: "Project 1", Slug: "project-1", OrgID: "org-123"},
				{ID: "proj-2", Name: "Project 2", Slug: "project-2", OrgID: "org-123"},
			},
			existingProjectID: "proj-1",
			expectedProjectID: "proj-1",
		},
		{
			name:            "Need existing project data - multiple projects without existing ID",
			orgRequired:     models.NeedExistingData,
			projectRequired: models.NeedExistingData,
			existingOrgs:    []models.Organization{{ID: "org-123", Name: "Test Org", Slug: "test-org"}},
			existingProjects: []models.Project{
				{ID: "proj-1", Name: "Project 1", Slug: "project-1", OrgID: "org-123"},
				{ID: "proj-2", Name: "Project 2", Slug: "project-2", OrgID: "org-123"},
			},
			expectedProjectID: "proj-1",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := s.createTestController()
			ctx := context.Background()

			futureTime := time.Now().Add(time.Hour)
			cfg := &config.Config{
				Credential: models.Credential{
					Token:          "valid-token",
					TokenExpiresAt: futureTime,
				},
				OrganizationID: tc.existingOrgID,
				ProjectID:      tc.existingProjectID,
			}

			// Ensure OrganizationID is set for project lookups
			if len(tc.existingProjects) > 0 || tc.projectRequired == models.NeedExistingData {
				if cfg.OrganizationID == "" {
					cfg.OrganizationID = "org-123"
				}
			}

			user := models.User{ID: "user-123"}

			mockConfig.On("GetConfig").Return(cfg)
			mockRepo.On("FindGitPathAndURL").Return("", "", nil)
			mockAPI.On("GetUser", ctx).Return(user, nil)
			mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
			//nolint:nestif // this is a test
			if tc.orgRequired == models.NeedExistingData {
				mockAPI.On("GetOrganizations", ctx).Return(tc.existingOrgs, nil)

				if len(tc.existingOrgs) == 0 {
					mockOrgHandler.On("PrintNoOrganizations").Return()
				} else if len(tc.existingOrgs) > 1 {
					if tc.existingOrgID != "" {
						selectedOrg := tc.existingOrgs[0]
						mockAPI.On("GetOrganizationByID", ctx, tc.existingOrgID).Return(selectedOrg, nil)
					} else {
						selectedOrg := tc.existingOrgs[0]
						mockAPI.On("GetOrganizationByID", ctx, "").Return(models.Organization{}, eris.New("not found"))
						mockOrgHandler.On("PromptForSwitch", ctx, tc.existingOrgs, false).Return(selectedOrg, nil)
					}
				}
			}
			//nolint:nestif // this is a test
			if tc.projectRequired == models.NeedExistingData {
				mockAPI.On("GetProjects", ctx, cfg.OrganizationID).Return(tc.existingProjects, nil)

				if len(tc.existingProjects) == 0 {
					mockProjectHandler.On("PrintNoProjectsInOrganization").Return()
				} else if len(tc.existingProjects) > 1 {
					if tc.existingProjectID != "" {
						selectedProject := tc.existingProjects[0]
						mockAPI.On("GetProjectByID", ctx, cfg.OrganizationID, tc.existingProjectID).Return(selectedProject, nil)
					} else {
						selectedProject := tc.existingProjects[0]
						mockAPI.On("GetProjectByID", ctx, cfg.OrganizationID, "").Return(models.Project{}, eris.New("not found"))
						mockProjectHandler.On(
							"Switch",
							ctx,
							models.SwitchProjectFlags{},
							models.Organization{ID: cfg.OrganizationID, Name: "Test Org", Slug: "test-org"},
							false,
						).Return(selectedProject, nil)
					}
				}
			}

			if !tc.expectError {
				mockConfig.On("Save").Return(nil)
			}

			req := models.SetupRequest{
				LoginRequired:        models.NeedLogin,
				OrganizationRequired: tc.orgRequired,
				ProjectRequired:      tc.projectRequired,
			}

			result, err := controller.SetupCommandState(ctx, req)

			if tc.expectError {
				s.Require().Error(err)
				if tc.errorContains != "" {
					s.Require().Contains(err.Error(), tc.errorContains)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(result)
				if tc.expectedOrgID != "" {
					s.Require().NotNil(result.Organization)
					s.Require().Equal(tc.expectedOrgID, result.Organization.ID)
				}
				if tc.expectedProjectID != "" {
					s.Require().NotNil(result.Project)
					s.Require().Equal(tc.expectedProjectID, result.Project.ID)
				}
			}

			mockAPI.AssertExpectations(s.T())
			mockConfig.AssertExpectations(s.T())
			mockInput.AssertExpectations(s.T())
			mockRepo.AssertExpectations(s.T())
			mockOrgHandler.AssertExpectations(s.T())
			mockProjectHandler.AssertExpectations(s.T())
		})
	}
}
