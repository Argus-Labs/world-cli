package cmdsetup_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	cmdsetup "pkg.world.dev/world-cli/cmd/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
	"pkg.world.dev/world-cli/cmd/world/organization"
	"pkg.world.dev/world-cli/cmd/world/project"
)

// Helper function to create a controller with fresh mocks for each test.
func createTestController() (
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

func TestSetupCommandState_IgnoreLogin_Success(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

	cfg := &config.Config{
		OrganizationID: "org-123",
		ProjectID:      "proj-456",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired:        models.IgnoreLogin,
		OrganizationRequired: models.Ignore,
		ProjectRequired:      models.Ignore,
	}

	result, err := controller.SetupCommandState(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.LoggedIn)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_NeedLogin_NotLoggedIn(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

	cfg := &config.Config{
		Credential: models.Credential{
			Token: "", // No token
		},
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)

	req := models.SetupRequest{
		LoginRequired: models.NeedLogin,
	}

	_, err := controller.SetupCommandState(ctx, req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "not logged in")

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_NeedLogin_Success(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

	futureTime := time.Now().Add(time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "valid-token",
			TokenExpiresAt: futureTime,
		},
	}
	user := models.User{
		ID:    "user-123",
		Name:  "Test User",
		Email: "test@example.com",
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockAPI.On("GetUser", ctx).Return(user, nil)
	mockAPI.On("GetOrganizationsInvitedTo", ctx).Return([]models.Organization{}, nil)
	mockConfig.On("Save").Return(nil)

	req := models.SetupRequest{
		LoginRequired: models.NeedLogin,
	}

	result, err := controller.SetupCommandState(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.LoggedIn)
	require.Equal(t, "user-123", result.User.ID)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_ExpiredToken(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

	pastTime := time.Now().Add(-time.Hour)
	cfg := &config.Config{
		Credential: models.Credential{
			Token:          "expired-token",
			TokenExpiresAt: pastTime,
		},
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)

	req := models.SetupRequest{
		LoginRequired: models.NeedLogin,
	}

	_, err := controller.SetupCommandState(ctx, req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "not logged in")

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestHandleOrganizationInvitations_AcceptInvitation(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestHandleOrganizationInvitations_DeclineInvitation(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_RepoLookup_Success(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.LoggedIn)
	require.NotNil(t, result.Organization)
	require.NotNil(t, result.Project)
	require.Equal(t, "org-789", result.Organization.ID)
	require.Equal(t, "proj-789", result.Project.ID)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_RepoLookup_NotLoggedIn(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.Error(t, err)
	require.Contains(t, err.Error(), "not logged in, can't lookup project from git repo")

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_NeedOrgData_NoOrgs_CreateNew(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Organization)
	require.Equal(t, "new-org-123", result.Organization.ID)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_NeedOrgData_OneOrg_UseExisting(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Organization)
	require.Equal(t, "existing-org-123", result.Organization.ID)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_NeedOrgData_MultipleOrgs(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Organization)
	require.Equal(t, "org-1", result.Organization.ID)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_NeedProjectData_NoProjects_CreateNew(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Project)
	require.Equal(t, "new-proj-123", result.Project.ID)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_NeedProjectData_OneProject_UseExisting(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Project)
	require.Equal(t, "existing-proj-123", result.Project.ID)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_MustNotExist_OrganizationExists(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

	cfg := &config.Config{
		OrganizationID: "org-123", // Has organization
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)

	req := models.SetupRequest{
		OrganizationRequired: models.MustNotExist,
	}

	_, err := controller.SetupCommandState(ctx, req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "organization already exists")

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_MustNotExist_ProjectExists(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

	cfg := &config.Config{
		ProjectID: "proj-123", // Has project
	}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)

	req := models.SetupRequest{
		ProjectRequired: models.MustNotExist,
	}

	_, err := controller.SetupCommandState(ctx, req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "project already exists")

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_ConfigSaveError(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

	cfg := &config.Config{}

	mockConfig.On("GetConfig").Return(cfg)
	mockRepo.On("FindGitPathAndURL").Return("", "", nil)
	mockConfig.On("Save").Return(config.ErrCannotSaveConfig)

	req := models.SetupRequest{
		LoginRequired: models.IgnoreLogin,
	}

	_, err := controller.SetupCommandState(ctx, req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to save config after setup")

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_IDOnly_Success(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Organization)
	require.NotNil(t, result.Project)
	require.Equal(t, "org-123", result.Organization.ID)
	require.Equal(t, "proj-456", result.Project.ID)
	require.Equal(t, "Test Project", result.Project.Name)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}

func TestSetupCommandState_RepoLookup_NewProjectDiscovered(t *testing.T) {
	t.Parallel()

	controller, mockAPI, mockConfig, mockInput, mockRepo, mockOrgHandler, mockProjectHandler := createTestController()
	ctx := t.Context()

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

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.LoggedIn)
	require.NotNil(t, result.Organization)
	require.NotNil(t, result.Project)
	require.Equal(t, "org-789", result.Organization.ID)
	require.Equal(t, "proj-789", result.Project.ID)

	mockAPI.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockInput.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOrgHandler.AssertExpectations(t)
	mockProjectHandler.AssertExpectations(t)
}
