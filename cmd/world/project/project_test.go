package project_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
	"pkg.world.dev/world-cli/cmd/world/project"
)

// ProjectTestSuite defines the test suite for project package.
type ProjectTestSuite struct {
	suite.Suite
}

// setupWorldProjectDir creates a temporary World project directory structure
// for tests that need to satisfy utils.IsInWorldCardinalRoot() checks.
// This function avoids changing the global working directory to prevent race conditions.
func (s *ProjectTestSuite) setupWorldProjectDir() string {
	// Create temporary directory with World project structure using testing utilities
	tmpDir := s.T().TempDir() // This automatically cleans up

	// Create World project structure
	err := os.MkdirAll(filepath.Join(tmpDir, "cardinal"), 0755)
	s.Require().NoError(err)

	// Create world.toml file
	worldTomlPath := filepath.Join(tmpDir, "world.toml")
	err = os.WriteFile(worldTomlPath, []byte(""), 0644)
	s.Require().NoError(err)

	// Initialize git repository to satisfy git checks (skip if git not available)
	if _, err := exec.LookPath("git"); err == nil {
		cmd := exec.Command("git", "init")
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			s.T().Logf("Warning: Could not initialize git repository: %v", err)
		}
	} else {
		s.T().Logf("Warning: git command not available, skipping git init")
	}

	// Return the temporary directory path instead of changing global working directory
	return tmpDir
}

// Helper method to create fresh mocks and handler for each test.
func (s *ProjectTestSuite) createTestHandler() (
	*project.Handler, *repo.MockClient, *config.MockService, *api.MockClient, *input.MockService) {
	mockRepoClient := &repo.MockClient{}
	mockConfigService := &config.MockService{}
	mockAPIClient := &api.MockClient{}
	mockInputService := &input.MockService{}

	// Create mock region selector that returns us-east-1
	mockRegionSelector := project.NewMockRegionSelector([]string{"us-east-1"}, nil)

	handler := project.NewHandlerWithRegionSelector(
		mockRepoClient,
		mockConfigService,
		mockAPIClient,
		mockInputService,
		mockRegionSelector,
	).(*project.Handler)

	return handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService
}

// Test fixtures.
func (s *ProjectTestSuite) createTestProject() models.Project {
	return models.Project{
		ID:        "proj-123",
		Name:      "Test Project",
		Slug:      "test_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "token123",
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"},
			Discord: models.ProjectConfigDiscord{
				Enabled: false,
			},
			Slack: models.ProjectConfigSlack{
				Enabled: false,
			},
		},
	}
}

func (s *ProjectTestSuite) createTestOrganization() models.Organization {
	return models.Organization{
		ID:   "org-123",
		Name: "Test Organization",
		Slug: "test_org",
	}
}

func (s *ProjectTestSuite) createTestConfig() *config.Config {
	return &config.Config{
		OrganizationID:  "org-123",
		ProjectID:       "proj-123",
		CurrRepoKnown:   false,
		CurrProjectName: "Test Project",
	}
}

// TestProjectSuite runs the test suite.
func TestProjectSuite(t *testing.T) {
	suite.Run(t, new(ProjectTestSuite))
}

func (s *ProjectTestSuite) TestHandler_Create_Success() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()
	testOrg := s.createTestOrganization()
	flags := models.CreateProjectFlags{
		Name: "Test Project",
		Slug: "test_project",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock PreCreateUpdateValidation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock API calls
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{"us-east-1", "us-west-2"}, nil)
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(testOrg, nil)

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter project name", "Test Project").
		Return("Test Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "test_project").
		Return("test_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "", "cardinal").Return(nil)

	// Create expected project for API call (matches what gets built during creation)
	expectedProjectForAPI := models.Project{
		Name:      "Test Project",
		Slug:      "test_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "", // Empty because repo validation succeeded without token
		Update:    false,
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"}, // Set by mock region selector
			Discord: models.ProjectConfigDiscord{
				Enabled: false,
			},
			Slack: models.ProjectConfigSlack{
				Enabled: false,
			},
		},
	}

	// Mock API slug check and create
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "00000000-0000-0000-0000-000000000000", "test_project").
		Return(nil)
	mockAPIClient.On("CreateProject", ctx, "org-123", expectedProjectForAPI).Return(testProject, nil)

	result, err := handler.Create(ctx, testOrg, flags)

	s.Require().NoError(err)
	s.Equal(testProject, result)
	mockRepoClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Create_CurrentRepoKnown() {
	s.T().Parallel()

	handler, _, mockConfigService, _, _ := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateProjectFlags{
		Name: "Test Project",
	}

	// Mock config service - current repo is known
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = true
	cfg.CurrProjectName = "Existing Project"
	mockConfigService.On("GetConfig").Return(cfg)

	result, err := handler.Create(ctx, testOrg, flags)

	s.Require().Error(err)
	s.Equal(project.ErrCannotCreateSwitchProject, err)
	s.Equal(models.Project{}, result)
	mockConfigService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Create_ValidationError() {
	s.T().Parallel()

	handler, mockRepoClient, mockConfigService, _, _ := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateProjectFlags{
		Name: "Test Project",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock validation failure
	validationErr := errors.New("not in git repository")
	mockRepoClient.On("FindGitPathAndURL").Return("", "", validationErr)

	result, err := handler.Create(ctx, testOrg, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to validate project creation")
	s.Equal(models.Project{}, result)
	mockRepoClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Create_GetRegionsError() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateProjectFlags{
		Name: "Test Project",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock successful validation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock API error for regions
	regionsErr := errors.New("failed to get regions")
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(s.createTestOrganization(), nil)
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{}, regionsErr)

	result, err := handler.Create(ctx, testOrg, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get available regions")
	s.Equal(models.Project{}, result)
	mockRepoClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Delete_Success() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()

	// Mock input confirmation
	mockInputService.On("Prompt", ctx, "Type 'Yes' to confirm deletion of 'Test Project (test_project)'", "no").
		Return("Yes", nil)

	// Mock API call
	mockAPIClient.On("DeleteProject", ctx, "org-123", "proj-123").Return(nil)

	// Mock config service
	mockConfigService.On("RemoveKnownProject", "proj-123", "org-123").Return(nil)

	err := handler.Delete(ctx, testProject)

	s.Require().NoError(err)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Delete_UserDeclines() {
	s.T().Parallel()

	handler, _, _, _, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()

	// Mock input confirmation - user declines
	mockInputService.On("Prompt", ctx, "Type 'Yes' to confirm deletion of 'Test Project (test_project)'", "no").
		Return("no", nil)

	err := handler.Delete(ctx, testProject)

	s.Require().NoError(err)
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Delete_IncorrectConfirmation() {
	s.T().Parallel()

	handler, _, _, _, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()

	// Mock input confirmation - user types "yes" instead of "Yes"
	mockInputService.On("Prompt", ctx, "Type 'Yes' to confirm deletion of 'Test Project (test_project)'", "no").
		Return("yes", nil)

	err := handler.Delete(ctx, testProject)

	s.Require().NoError(err)
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Delete_InputError() {
	s.T().Parallel()

	handler, _, _, _, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()

	// Mock input error
	inputErr := errors.New("input error")
	mockInputService.On("Prompt", ctx, "Type 'Yes' to confirm deletion of 'Test Project (test_project)'", "no").
		Return("", inputErr)

	err := handler.Delete(ctx, testProject)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to prompt for confirmation")
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Delete_APIError() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()

	// Mock input confirmation
	mockInputService.On("Prompt", ctx, "Type 'Yes' to confirm deletion of 'Test Project (test_project)'", "no").
		Return("Yes", nil)

	// Mock API error
	apiErr := errors.New("API error")
	mockAPIClient.On("DeleteProject", ctx, "org-123", "proj-123").Return(apiErr)

	err := handler.Delete(ctx, testProject)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to delete project")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_WithSlug_Success() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()
	flags := models.SwitchProjectFlags{
		Slug: "test_project",
	}
	testOrg := models.Organization{
		ID: "org-123",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls
	projects := []models.Project{testProject}
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)

	result, err := handler.Switch(ctx, flags, testOrg, false)

	s.Require().NoError(err)
	s.Equal(testProject, result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_WithSlug_NotFound() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{
		Slug: "nonexistent-project",
	}
	testOrg := models.Organization{
		ID: "org-123",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock API calls
	projects := []models.Project{s.createTestProject()}
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)

	result, err := handler.Switch(ctx, flags, testOrg, false)

	s.Require().Error(err)
	s.Equal(models.Project{}, result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_CurrentRepoKnown() {
	s.T().Parallel()

	handler, _, mockConfigService, _, _ := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{}
	testOrg := models.Organization{
		ID: "org-123",
	}
	// Mock config service - current repo is known
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = true
	cfg.CurrProjectName = "Existing Project"
	mockConfigService.On("GetConfig").Return(cfg)

	result, err := handler.Switch(ctx, flags, testOrg, false)

	s.Require().Error(err)
	s.Equal(project.ErrCannotCreateSwitchProject, err)
	s.Equal(models.Project{}, result)
	mockConfigService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_NoProjects() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{}
	testOrg := models.Organization{
		ID: "org-123",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock API calls - return empty projects list
	mockAPIClient.On("GetProjects", ctx, "org-123").Return([]models.Project{}, nil)

	result, err := handler.Switch(ctx, flags, testOrg, false)

	s.Require().NoError(err)
	s.Equal(models.Project{}, result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_APIError() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{}
	testOrg := models.Organization{
		ID: "org-123",
	}
	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock API error
	apiErr := errors.New("API error")
	mockAPIClient.On("GetProjects", ctx, "org-123").Return([]models.Project{}, apiErr)

	result, err := handler.Switch(ctx, flags, testOrg, false)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get projects")
	s.Equal(models.Project{}, result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Update_EmptyProject() {
	s.T().Parallel()

	handler, mockRepoClient, _, _, _ := s.createTestHandler()
	ctx := context.Background()
	emptyProject := models.Project{} // Empty project
	testOrg := s.createTestOrganization()
	flags := models.UpdateProjectFlags{
		Name: "Updated Project",
	}

	// Mock validation failure - not in repository
	validationErr := errors.New("not in git repository")
	mockRepoClient.On("FindGitPathAndURL").Return("", "", validationErr)

	err := handler.Update(ctx, emptyProject, testOrg, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to validate project update")
	mockRepoClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Update_ValidationError() {
	s.T().Parallel()

	handler, mockRepoClient, _, _, _ := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()
	testOrg := s.createTestOrganization()
	flags := models.UpdateProjectFlags{
		Name: "Updated Project",
	}

	// Mock validation failure
	validationErr := errors.New("not in git repository")
	mockRepoClient.On("FindGitPathAndURL").Return("", "", validationErr)

	err := handler.Update(ctx, testProject, testOrg, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to validate project update")
	mockRepoClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_HandleSwitch_SingleProject() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()
	testOrg := models.Organization{
		ID: "org-123",
	}
	// Mock config service
	cfg := s.createTestConfig()
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls
	projects := []models.Project{testProject}
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)

	err := handler.HandleSwitch(ctx, testOrg)

	s.Require().NoError(err)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_HandleSwitch_NoProjects() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, _, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testOrg := models.Organization{
		ID: "org-123",
	}
	// Mock API calls - no projects
	mockAPIClient.On("GetProjects", ctx, "org-123").Return([]models.Project{}, nil)

	// Mock PreCreateUpdateValidation - success (can create project)
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock user declines project creation
	mockInputService.On("Confirm", ctx, "Do you want to create a new project now? (y/n)", "Y").
		Return(false, nil)

	err = handler.HandleSwitch(ctx, testOrg)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
	mockRepoClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_PreCreateUpdateValidation_Success() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, _, _, _ := s.createTestHandler()

	// Mock successful validation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	repoPath, repoURL, err := handler.PreCreateUpdateValidation(false)

	// Should succeed because we're in a proper World project directory
	s.Equal("cardinal", repoPath)
	s.Equal("https://github.com/test/repo", repoURL)
	s.Require().NoError(err) // Should succeed now
	mockRepoClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_PreCreateUpdateValidation_NotInRepo() {
	s.T().Parallel()

	handler, mockRepoClient, _, _, _ := s.createTestHandler()

	// Mock validation failure - not in repository
	validationErr := errors.New("not in git repository")
	mockRepoClient.On("FindGitPathAndURL").Return("", "", validationErr)

	repoPath, repoURL, err := handler.PreCreateUpdateValidation(false)

	s.Require().Error(err)
	s.Empty(repoPath)
	s.Empty(repoURL)
	mockRepoClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Create_SlugAlreadyExists() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateProjectFlags{
		Name: "Test Project",
		Slug: "test_project",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock PreCreateUpdateValidation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock API calls
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{"us-east-1"}, nil)
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(testOrg, nil)

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter project name", "Test Project").
		Return("Test Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "test_project").
		Return("test_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "", "cardinal").Return(nil)

	// Mock API slug check and create - slug already exists error
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "00000000-0000-0000-0000-000000000000", "test_project").
		Return(nil)

	// Expected project with region from mock selector
	expectedProject := models.Project{
		Name:      "Test Project",
		Slug:      "test_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
		Update:    false,
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"},
			Discord: models.ProjectConfigDiscord{
				Enabled: false,
			},
			Slack: models.ProjectConfigSlack{
				Enabled: false,
			},
		},
	}

	mockAPIClient.On("CreateProject", ctx, "org-123", expectedProject).
		Return(models.Project{}, api.ErrProjectSlugAlreadyExists)

	result, err := handler.Create(ctx, testOrg, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to create project")
	s.Equal(models.Project{}, result)
	mockRepoClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Update_Success() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()
	testOrg := s.createTestOrganization()
	flags := models.UpdateProjectFlags{
		Name: "Updated Project",
		Slug: "updated_project",
	}

	// Mock PreCreateUpdateValidation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock API calls
	mockAPIClient.On("GetListRegions", ctx, "org-123", "proj-123").
		Return([]string{"us-east-1", "us-west-2"}, nil)

	// Mock input interactions for update
	mockInputService.On("Prompt", ctx, "Enter project name", "Updated Project").
		Return("Updated Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "updated_project").
		Return("updated_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation - first tries public (empty token), then prompts for token
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").
		Return(errors.New("repo is private"))
	mockInputService.On("Prompt", ctx, "\nEnter Token", "token123").Return("token123", nil)
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "token123").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "token123", "cardinal").Return(nil)

	// Mock API slug check and update
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "proj-123", "updated_project").
		Return(nil)

	updatedProject := testProject
	updatedProject.Name = "Updated Project"
	updatedProject.Slug = "updated_project"
	updatedProject.Update = true
	updatedProject.Config.Region = []string{"us-east-1"}

	mockAPIClient.On("UpdateProject", ctx, "org-123", "proj-123", updatedProject).
		Return(updatedProject, nil)

	// Mock config service
	mockConfigService.On("RemoveKnownProject", "proj-123", "org-123").Return(nil)

	err = handler.Update(ctx, testProject, testOrg, flags)

	s.Require().NoError(err)
	mockRepoClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Update_SlugAlreadyExists() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()
	testOrg := s.createTestOrganization()
	flags := models.UpdateProjectFlags{
		Name: "Updated Project",
		Slug: "existing_slug",
	}

	// Mock PreCreateUpdateValidation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock API calls
	mockAPIClient.On("GetListRegions", ctx, "org-123", "proj-123").
		Return([]string{"us-east-1"}, nil)

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter project name", "Updated Project").
		Return("Updated Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "existing_slug").
		Return("existing_slug", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation - first tries public (empty token), then prompts for token
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").
		Return(errors.New("repo is private"))
	mockInputService.On("Prompt", ctx, "\nEnter Token", "token123").Return("token123", nil)
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "token123").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "token123", "cardinal").Return(nil)

	// Mock API slug check and update - slug already exists
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "proj-123", "existing_slug").
		Return(nil)

	updatedProject := testProject
	updatedProject.Name = "Updated Project"
	updatedProject.Slug = "existing_slug"
	updatedProject.Update = true
	updatedProject.Config.Region = []string{"us-east-1"}

	mockAPIClient.On("UpdateProject", ctx, "org-123", "proj-123", updatedProject).
		Return(models.Project{}, api.ErrProjectSlugAlreadyExists)

	// Mock config service (won't be called due to error)
	mockConfigService.On("RemoveKnownProject", "proj-123", "org-123").Return(nil)

	err = handler.Update(ctx, testProject, testOrg, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to update project")
	mockRepoClient.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_EnableCreation_NoProjects() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{}
	org := s.createTestOrganization()
	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls - no projects, but enableCreation is true
	mockAPIClient.On("GetProjects", ctx, "org-123").Return([]models.Project{}, nil)
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(org, nil)
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{"us-east-1"}, nil)

	// Mock PreCreateUpdateValidation for creation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock create project flow
	mockInputService.On("Prompt", ctx, "Enter project name", "").
		Return("New Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "new_project").
		Return("new_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "", "cardinal").Return(nil)

	// Mock API calls for creation
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "00000000-0000-0000-0000-000000000000", "new_project").
		Return(nil)

	newProject := models.Project{
		ID:        "new-proj-123",
		Name:      "New Project",
		Slug:      "new_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
	}

	expectedProject := models.Project{
		Name:      "New Project",
		Slug:      "new_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
		Update:    false,
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"},
			Discord: models.ProjectConfigDiscord{
				Enabled: false,
			},
			Slack: models.ProjectConfigSlack{
				Enabled: false,
			},
		},
	}

	mockAPIClient.On("CreateProject", ctx, "org-123", expectedProject).
		Return(newProject, nil)

	result, err := handler.Switch(ctx, flags, org, true) // enableCreation = true

	s.Require().NoError(err)
	s.Equal(newProject, result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockRepoClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_MultipleProjects_UserSelection() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{} // No slug provided
	org := s.createTestOrganization()
	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Create multiple projects
	project1 := s.createTestProject()
	project2 := models.Project{
		ID:    "proj-456",
		Name:  "Second Project",
		Slug:  "second_project",
		OrgID: "org-123",
	}
	projects := []models.Project{project1, project2}

	// Mock API calls
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)

	// Mock user selecting project 2
	mockInputService.On("Prompt", ctx, "Enter project number ('q' to quit)", "").
		Return("2", nil)

	result, err := handler.Switch(ctx, flags, org, false)

	s.Require().NoError(err)
	s.Equal(project2, result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_UserQuits() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{}
	org := s.createTestOrganization()
	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock API calls with multiple projects
	projects := []models.Project{s.createTestProject()}
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)

	// Mock user quitting
	mockInputService.On("Prompt", ctx, "Enter project number ('q' to quit)", "").
		Return("q", nil)

	result, err := handler.Switch(ctx, flags, org, false)

	s.Require().NoError(err)
	s.Equal(models.Project{}, result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_InvalidSelection() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{}
	org := s.createTestOrganization()
	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls
	projects := []models.Project{s.createTestProject()}
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)

	// Mock user providing invalid input, then valid input
	mockInputService.On("Prompt", ctx, "Enter project number ('q' to quit)", "").
		Return("invalid", nil).Once()
	mockInputService.On("Prompt", ctx, "Enter project number ('q' to quit)", "").
		Return("1", nil).Once()

	result, err := handler.Switch(ctx, flags, org, false)

	s.Require().NoError(err)
	s.Equal(s.createTestProject(), result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_HandleSwitch_MultipleProjects_CurrentSelected() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()
	org := s.createTestOrganization()
	// Mock config service with current project ID matching one of the projects
	cfg := s.createTestConfig()
	cfg.ProjectID = "proj-123" // Matches testProject.ID
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock API calls with multiple projects including current one
	projects := []models.Project{
		testProject,
		{ID: "proj-456", Name: "Other Project", OrgID: "org-123"},
	}
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)

	err := handler.HandleSwitch(ctx, org)

	s.Require().NoError(err)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_HandleSwitch_MultipleProjects_NoneSelected() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()
	org := s.createTestOrganization()
	// Mock config service with different project ID
	cfg := s.createTestConfig()
	cfg.ProjectID = "different-proj-id"
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls with multiple projects
	projects := []models.Project{testProject}
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)

	err := handler.HandleSwitch(ctx, org)

	s.Require().NoError(err)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Delete_ConfigError() {
	s.T().Parallel()

	handler, _, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testProject := s.createTestProject()

	// Mock input confirmation
	mockInputService.On("Prompt", ctx, "Type 'Yes' to confirm deletion of 'Test Project (test_project)'", "no").
		Return("Yes", nil)

	// Mock successful API call
	mockAPIClient.On("DeleteProject", ctx, "org-123", "proj-123").Return(nil)

	// Mock config service error
	configErr := errors.New("config save error")
	mockConfigService.On("RemoveKnownProject", "proj-123", "org-123").Return(configErr)

	err := handler.Delete(ctx, testProject)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to remove project from config")
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_PreCreateUpdateValidation_NotInWorldRoot() {
	s.T().Parallel()

	handler, mockRepoClient, _, _, _ := s.createTestHandler()

	// Mock successful git validation but not in World root
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	repoPath, repoURL, err := handler.PreCreateUpdateValidation(false)

	// Should fail because not in World Cardinal root (no world.toml/cardinal dir setup)
	s.Require().Error(err)
	s.Contains(err.Error(), "Not in a World Cardinal root")
	s.Equal("cardinal", repoPath)
	s.Equal("https://github.com/test/repo", repoURL)
	mockRepoClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Switch_CreateOption_InRepoRoot() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchProjectFlags{}
	org := s.createTestOrganization()
	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls - multiple projects exist
	projects := []models.Project{s.createTestProject()}
	mockAPIClient.On("GetProjects", ctx, "org-123").Return(projects, nil)
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(s.createTestOrganization(), nil)
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{"us-east-1"}, nil)

	// Mock PreCreateUpdateValidation for enabling create option (success = no error)
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock user selecting 'c' to create new project
	mockInputService.On("Prompt", ctx, "Enter project number ('c' to create new, 'q' to quit)", "").
		Return("c", nil)

	// Mock create project flow
	mockInputService.On("Prompt", ctx, "Enter project name", "").
		Return("New Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "new_project").
		Return("new_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "", "cardinal").Return(nil)

	// Mock API calls for creation
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "00000000-0000-0000-0000-000000000000", "new_project").
		Return(nil)

	newProject := models.Project{
		ID:        "new-proj-123",
		Name:      "New Project",
		Slug:      "new_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
	}

	expectedProject := models.Project{
		Name:      "New Project",
		Slug:      "new_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
		Update:    false,
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"},
			Discord: models.ProjectConfigDiscord{
				Enabled: false,
			},
			Slack: models.ProjectConfigSlack{
				Enabled: false,
			},
		},
	}

	mockAPIClient.On("CreateProject", ctx, "org-123", expectedProject).
		Return(newProject, nil)

	result, err := handler.Switch(ctx, flags, org, true) // enableCreation = true

	s.Require().NoError(err)
	s.Equal(newProject, result)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockRepoClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_HandleSwitch_NoProjects_CreateConfirmed() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	// Mock config service
	cfg := s.createTestConfig()
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls - no projects
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(s.createTestOrganization(), nil)
	mockAPIClient.On("GetProjects", ctx, "org-123").Return([]models.Project{}, nil)
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{"us-east-1"}, nil)

	// Mock PreCreateUpdateValidation - success (can create project)
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock user confirms project creation
	mockInputService.On("Confirm", ctx, "Do you want to create a new project now? (y/n)", "Y").
		Return(true, nil)

	// Mock create project flow
	mockInputService.On("Prompt", ctx, "Enter project name", "").
		Return("New Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "new_project").
		Return("new_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "", "cardinal").Return(nil)

	// Mock API calls for creation
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "00000000-0000-0000-0000-000000000000", "new_project").
		Return(nil)

	newProject := models.Project{
		ID:        "new-proj-123",
		Name:      "New Project",
		Slug:      "new_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
	}

	expectedProject := models.Project{
		Name:      "New Project",
		Slug:      "new_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
		Update:    false,
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"},
			Discord: models.ProjectConfigDiscord{
				Enabled: false,
			},
			Slack: models.ProjectConfigSlack{
				Enabled: false,
			},
		},
	}

	mockAPIClient.On("CreateProject", ctx, "org-123", expectedProject).
		Return(newProject, nil)

	err = handler.HandleSwitch(ctx, org)

	s.Require().NoError(err)
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockRepoClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Create_InvalidURL() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateProjectFlags{
		Name: "Test Project",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock PreCreateUpdateValidation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock API calls
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{"us-east-1"}, nil)
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(s.createTestOrganization(), nil)

	// Mock input interactions - invalid URL first, then valid
	mockInputService.On("Prompt", ctx, "Enter project name", "Test Project").
		Return("Test Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "test_project").
		Return("test_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("invalid-url", nil).Once() // Invalid URL first
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil).Once() // Valid URL second

	// Continue with rest of flow
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation - first for invalid URL (with https:// prepended), then for valid URL
	mockRepoClient.On("ValidateRepoToken", ctx, "https://invalid-url", "").
		Return(errors.New("invalid repository")).
		Once()
	// Mock token prompt for invalid URL (user will be prompted for token after validation fails)
	mockInputService.On("Prompt", ctx, "\nEnter Token", "").Return("", nil).Once()
	// Mock second validation attempt for invalid URL with empty token (still fails)
	mockRepoClient.On("ValidateRepoToken", ctx, "https://invalid-url", "").
		Return(errors.New("invalid repository")).
		Once()
	// After first URL fails, user enters valid URL
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "", "cardinal").Return(nil)

	// Mock API calls
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "00000000-0000-0000-0000-000000000000", "test_project").
		Return(nil)

	expectedProject := models.Project{
		Name:      "Test Project",
		Slug:      "test_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
		Update:    false,
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"},
			Discord: models.ProjectConfigDiscord{
				Enabled: false,
			},
			Slack: models.ProjectConfigSlack{
				Enabled: false,
			},
		},
	}

	testProject := s.createTestProject()
	mockAPIClient.On("CreateProject", ctx, "org-123", expectedProject).Return(testProject, nil)

	result, err := handler.Create(ctx, testOrg, flags)

	s.Require().NoError(err)
	s.Equal(testProject, result)
	mockRepoClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Create_WithNotifications() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateProjectFlags{
		Name: "Test Project",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock PreCreateUpdateValidation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock API calls
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{"us-east-1"}, nil)
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(s.createTestOrganization(), nil)

	// Mock input interactions with notifications enabled
	mockInputService.On("Prompt", ctx, "Enter project name", "Test Project").
		Return("Test Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "test_project").
		Return("test_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)

	// Discord notifications enabled
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(true, nil)
	mockInputService.On("Prompt", ctx, "Enter Discord bot token", "").
		Return("discord-bot-token-123", nil)
	mockInputService.On("Prompt", ctx, "Enter Discord channel ID", "").
		Return("discord-channel-123", nil)

	// Slack notifications enabled
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(true, nil)
	mockInputService.On("Prompt", ctx, "Enter Slack token", "").
		Return("slack-token-123", nil)
	mockInputService.On("Prompt", ctx, "Enter Slack channel ID", "").
		Return("slack-channel-123", nil)

	// Mock repo validation
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").Return(nil)
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "", "cardinal").Return(nil)

	// Mock API calls
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "00000000-0000-0000-0000-000000000000", "test_project").
		Return(nil)

	expectedProject := models.Project{
		Name:      "Test Project",
		Slug:      "test_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
		Update:    false,
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"},
			Discord: models.ProjectConfigDiscord{
				Enabled: true,
				Token:   "discord-bot-token-123",
				Channel: "discord-channel-123",
			},
			Slack: models.ProjectConfigSlack{
				Enabled: true,
				Token:   "slack-token-123",
				Channel: "slack-channel-123",
			},
		},
	}

	testProject := s.createTestProject()
	testProject.Config.Discord.Enabled = true
	testProject.Config.Discord.Token = "discord-bot-token-123"
	testProject.Config.Discord.Channel = "discord-channel-123"
	testProject.Config.Slack.Enabled = true
	testProject.Config.Slack.Token = "slack-token-123"
	testProject.Config.Slack.Channel = "slack-channel-123"

	mockAPIClient.On("CreateProject", ctx, "org-123", expectedProject).Return(testProject, nil)

	result, err := handler.Create(ctx, testOrg, flags)

	s.Require().NoError(err)
	s.Equal(testProject, result)
	mockRepoClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Create_PublicTokenSelection() {
	// Remove s.T().Parallel() to avoid race conditions with directory changes

	// Setup World project directory structure and change to it
	tmpDir := s.setupWorldProjectDir()
	originalDir, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		os.Chdir(originalDir)
	})
	err = os.Chdir(tmpDir)
	s.Require().NoError(err)

	handler, mockRepoClient, mockConfigService, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateProjectFlags{
		Name: "Test Project",
	}

	// Mock config service
	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock PreCreateUpdateValidation
	mockRepoClient.On("FindGitPathAndURL").Return("cardinal", "https://github.com/test/repo", nil)

	// Mock API calls
	mockAPIClient.On("GetListRegions", ctx, "org-123", "00000000-0000-0000-0000-000000000000").
		Return([]string{"us-east-1"}, nil)
	mockAPIClient.On("GetOrganizationByID", ctx, "org-123").Return(testOrg, nil)

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter project name", "Test Project").
		Return("Test Project", nil)
	mockInputService.On("Prompt", ctx, "Slug", "test_project").
		Return("test_project", nil)
	mockInputService.On("Prompt", ctx, "Enter Repository URL", "https://github.com/test/repo").
		Return("https://github.com/test/repo", nil)
	mockInputService.On("Prompt", ctx, "Enter path to Cardinal within Repo (Empty Valid)", "cardinal").
		Return("cardinal", nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Discord notifications? (y/n)", "n").
		Return(false, nil)
	mockInputService.On("Confirm", ctx, "Do you want to set up Slack notifications? (y/n)", "n").
		Return(false, nil)

	// Mock repo validation - first fails (private), user selects "public", second validation succeeds
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").
		Return(errors.New("repo is private")).
		Once()
	mockInputService.On("Prompt", ctx, "\nEnter Token", "").Return("public", nil)
	// After user selects "public", processRepoToken converts it to empty string, and validation should succeed
	mockRepoClient.On("ValidateRepoToken", ctx, "https://github.com/test/repo", "").Return(nil).Once()
	mockRepoClient.On("ValidateRepoPath", ctx, "https://github.com/test/repo", "", "cardinal").Return(nil)

	// Mock API calls
	mockAPIClient.On("CheckProjectSlugIsTaken", ctx, "org-123", "00000000-0000-0000-0000-000000000000", "test_project").
		Return(nil)

	expectedProject := models.Project{
		Name:      "Test Project",
		Slug:      "test_project",
		OrgID:     "org-123",
		RepoURL:   "https://github.com/test/repo",
		RepoPath:  "cardinal",
		RepoToken: "",
		Update:    false,
		Config: models.ProjectConfig{
			Region: []string{"us-east-1"},
			Discord: models.ProjectConfigDiscord{
				Enabled: false,
			},
			Slack: models.ProjectConfigSlack{
				Enabled: false,
			},
		},
	}

	testProject := s.createTestProject()
	testProject.RepoToken = ""
	mockAPIClient.On("CreateProject", ctx, "org-123", expectedProject).Return(testProject, nil)

	result, err := handler.Create(ctx, testOrg, flags)

	s.Require().NoError(err)
	s.Equal(testProject, result)
	mockRepoClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Utils_GetSelectedProject_NoProjectID() {
	s.T().Parallel()

	handler, mockRepoClient, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	// Mock API calls - no projects (HandleSwitch calls GetProjects first)
	mockAPIClient.On("GetProjects", ctx, "org-123").Return([]models.Project{}, nil)

	// Mock PreCreateUpdateValidation (needed by handleNoProjects when no projects exist)
	mockRepoClient.On("FindGitPathAndURL").Return("", "", errors.New("not in git repository"))

	// Call the utility function through HandleSwitch which uses it
	err := handler.HandleSwitch(ctx, testOrg)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
	mockRepoClient.AssertExpectations(s.T())
}

func (s *ProjectTestSuite) TestHandler_Utils_GetSelectedProject_NoOrganization() {
	s.T().Parallel()

	handler, mockRepoClient, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	// Mock API calls - no projects (HandleSwitch calls GetProjects first)
	mockAPIClient.On("GetProjects", ctx, "org-123").Return([]models.Project{}, nil)

	// Mock PreCreateUpdateValidation (needed by handleNoProjects when no projects exist)
	mockRepoClient.On("FindGitPathAndURL").Return("", "", errors.New("not in git repository"))

	// Call through HandleSwitch
	err := handler.HandleSwitch(ctx, testOrg)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
	mockRepoClient.AssertExpectations(s.T())
}
