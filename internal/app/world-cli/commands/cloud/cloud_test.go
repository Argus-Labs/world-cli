package cloud_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/internal/app/world-cli/clients/api"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/cloud"
	"pkg.world.dev/world-cli/internal/app/world-cli/commands/project"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
	"pkg.world.dev/world-cli/internal/app/world-cli/services/config"
	"pkg.world.dev/world-cli/internal/app/world-cli/services/input"
)

type CloudTestSuite struct {
	suite.Suite
}

func TestCloudSuite(t *testing.T) {
	suite.Run(t, new(CloudTestSuite))
}

// Helper function to create test handler with mocks.
func (s *CloudTestSuite) createTestHandler() (
	*cloud.Handler,
	*api.MockClient,
	*config.MockService,
	*input.MockService,
	*project.MockHandler,
) {
	mockAPI := &api.MockClient{}
	mockConfig := &config.MockService{}
	mockInput := &input.MockService{}
	mockProject := &project.MockHandler{}

	handler := cloud.NewHandler(mockAPI, mockConfig, mockProject, mockInput)

	return handler, mockAPI, mockConfig, mockInput, mockProject
}

func (s *CloudTestSuite) createTestProject() models.Project {
	return models.Project{
		ID:      "test-project-id",
		Name:    "Test Project",
		Slug:    "test-project",
		OrgID:   "test-org-id",
		RepoURL: "https://github.com/argus-labs/starter-game-template",
	}
}

func (s *CloudTestSuite) createTestOrganization() models.Organization {
	return models.Organization{
		ID:   "test-org-id",
		Name: "Test Org",
		Slug: "test-org",
	}
}

func (s *CloudTestSuite) TestHandler_DeploymentDeploy_Success() {
	handler, mockAPI, mockConfig, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock config for project deployment
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock preview deployment with proper return value
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeDeploy,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeDeploy).
		Return(previewResponse, nil)

	// Mock confirmation
	mockInput.On("Confirm", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(true, nil)

	// Mock deployment API calls
	mockAPI.On("DeployProject", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeDeploy).
		Return(nil)

	// Mock deployment status polling
	statusResponse := []byte(`{
		"data": {
			"dev": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "created"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil)

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeDeploy)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentDeploy_UserCancels() {
	handler, mockAPI, _, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock preview deployment
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeDeploy,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", ctx, "test-org-id", "test-project-id", models.DeploymentTypeDeploy).
		Return(previewResponse, nil)

	// Mock user declining
	mockInput.On("Confirm", ctx, "Do you want to proceed with the Deploying? (Y/n)", "n").
		Return(false, nil)

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeDeploy)

	s.Require().NoError(err) // Should not error when user cancels
	mockAPI.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentDestroy_Success() {
	handler, mockAPI, mockConfig, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock config for project deployment
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock preview deployment with proper return value
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeDestroy,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeDestroy).
		Return(previewResponse, nil)

	// Mock confirmation
	mockInput.On("Confirm", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(true, nil)

	// Mock deployment API calls
	mockAPI.On("DeployProject", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeDestroy).
		Return(nil)

	// Mock deployment status polling with proper context handling
	statusResponse := []byte(`{
		"data": {
			"dev": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "removed"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil)

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeDestroy)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentReset_Success() {
	handler, mockAPI, mockConfig, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock config for project deployment
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock preview deployment with proper return value
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeReset,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeReset).
		Return(previewResponse, nil)

	// Mock confirmation
	mockInput.On("Confirm", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(true, nil)

	// Mock deployment API calls
	mockAPI.On("DeployProject", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeReset).
		Return(nil)

	// Mock deployment status polling
	statusResponse := []byte(`{
		"data": {
			"dev": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "created"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil)

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeReset)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentPromote_Success() {
	handler, mockAPI, mockConfig, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock config for project deployment
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock preview deployment with proper return value
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypePromote,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypePromote).
		Return(previewResponse, nil)

	// Mock confirmation
	mockInput.On("Confirm", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(true, nil)

	// Mock deployment API calls
	mockAPI.On("DeployProject", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypePromote).
		Return(nil)

	// Mock deployment status polling
	statusResponse := []byte(`{
		"data": {
			"live": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "created"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil)

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypePromote)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentForceDeploy_Success() {
	handler, mockAPI, mockConfig, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock config for project deployment
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock preview deployment with proper return value
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeForceDeploy,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeForceDeploy).
		Return(previewResponse, nil)

	// Mock confirmation
	mockInput.On("Confirm", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(true, nil)

	// Mock deployment API calls
	mockAPI.On("DeployProject", mock.Anything, "test-org-id", "test-project-id", "deploy?force=true").
		Return(nil)

	// Mock deployment status polling for destroy first
	statusResponse := []byte(`{
		"data": {
			"dev": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "removed"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil).Once()

	// Then for deploy
	statusResponse2 := []byte(`{
		"data": {
			"dev": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "created"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse2, nil)

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeForceDeploy)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentNoOrganization() {
	handler, _, _, _, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Test with empty organization ID
	err := handler.Deployment(ctx, "", project, models.DeploymentTypeDeploy)

	s.Require().NoError(err) // Should not error, just return early
}

func (s *CloudTestSuite) TestHandler_DeploymentCreateProject() {
	handler, mockAPI, mockConfig, mockInput, mockProject := s.createTestHandler()
	ctx := context.Background()
	project := models.Project{
		ID:      "", // Empty project ID to trigger creation
		RepoURL: "https://github.com/argus-labs/starter-game-template",
	}

	// Mock config for project deployment
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock getting organization
	org := s.createTestOrganization()
	mockAPI.On("GetOrganizationByID", mock.Anything, "test-org-id").Return(org, nil)

	// Mock project creation
	createdProject := models.Project{
		ID:      "new-project-id",
		Name:    "Test Project",
		Slug:    "test-project",
		OrgID:   "test-org-id",
		RepoURL: "https://github.com/argus-labs/starter-game-template",
	}
	mockProject.On("Create", mock.Anything, org, models.CreateProjectFlags{}).Return(createdProject, nil)

	// Mock preview deployment with new project ID
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeDeploy,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", mock.Anything, "test-org-id", "new-project-id", models.DeploymentTypeDeploy).
		Return(previewResponse, nil)

	// Mock user confirmation
	mockInput.On("Confirm", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(true, nil)

	// Mock deploy project call with proper signature
	mockAPI.On("DeployProject", mock.Anything, "test-org-id", "new-project-id", models.DeploymentTypeDeploy).
		Return(nil)

	// Mock deployment status check
	statusResponse := []byte(`{
		"data": {
			"dev": {
				"project_id": "new-project-id",
				"created_by": "test-user", 
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "created"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "new-project-id").Return(statusResponse, nil)

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeDeploy)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
	mockProject.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentPreviewError() {
	handler, mockAPI, _, _, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock preview deployment error
	mockAPI.On("PreviewDeployment", ctx, "test-org-id", "test-project-id", models.DeploymentTypeDeploy).
		Return(models.DeploymentPreview{}, errors.New("preview failed"))

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeDeploy)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to preview deployment")
	mockAPI.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentInputError() {
	handler, mockAPI, _, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock preview deployment
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeDeploy,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", ctx, "test-org-id", "test-project-id", models.DeploymentTypeDeploy).
		Return(previewResponse, nil)

	// Mock input error
	mockInput.On("Confirm", ctx, "Do you want to proceed with the Deploying? (Y/n)", "n").
		Return(false, errors.New("input failed"))

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeDeploy)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to prompt user")
	mockAPI.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_DeploymentAPIError() {
	handler, mockAPI, _, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock preview deployment
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeDestroy,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", ctx, "test-org-id", "test-project-id", models.DeploymentTypeDestroy).
		Return(previewResponse, nil)

	// Mock user confirmation
	mockInput.On("Confirm", ctx, "Do you want to proceed with the Destroying? (Y/n)", "n").
		Return(true, nil)

	// Mock API error
	mockAPI.On("DeployProject", ctx, "test-org-id", "test-project-id", models.DeploymentTypeDestroy).
		Return(errors.New("API error"))

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeDestroy)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to deploy project")
	mockAPI.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_Status_Success() {
	handler, mockAPI, _, _, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	project := s.createTestProject()

	// Mock deployment status
	statusResponse := []byte(`{
		"data": {
			"dev": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "created"
			},
			"live": {
				"project_id": "test-project-id", 
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "removed"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil)

	// Note: No health status call expected since none of the deployments should trigger health checks
	// dev is "created" with type "deploy" but we don't have the DeployType field in the response

	err := handler.Status(ctx, org, project)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_Status_DeploymentStatusError() {
	handler, mockAPI, _, _, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	project := s.createTestProject()

	// Mock deployment status error
	mockAPI.On("GetDeploymentStatus", ctx, "test-project-id").
		Return([]byte{}, errors.New("status error"))

	err := handler.Status(ctx, org, project)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get deployment status")
	mockAPI.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_Status_InvalidJSON() {
	handler, mockAPI, _, _, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	project := s.createTestProject()

	// Mock invalid JSON response
	statusResponse := []byte(`invalid json`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil)

	err := handler.Status(ctx, org, project)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to unmarshal deployment response")
	mockAPI.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_TailLogs_Success() {
	handler, mockAPI, mockConfig, mockInput, _ := s.createTestHandler()
	ctx := context.Background()

	// Mock config
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock API calls
	mockAPI.On("GetOrganizationByID", mock.Anything, "test-org-id").
		Return(s.createTestOrganization(), nil)
	mockAPI.On("GetProjectByID", mock.Anything, "test-org-id", "test-project-id").
		Return(s.createTestProject(), nil)

	// Mock health status for environment list
	healthResponse := map[string]models.DeploymentHealthCheckResult{
		"dev": {OK: true, Offline: false},
	}
	mockAPI.On("GetDeploymentHealthStatus", mock.Anything, "test-project-id").
		Return(healthResponse, nil).Maybe()

	// Mock input prompts
	mockInput.On("Prompt", mock.Anything, "Choose an environment", "1").Return("1", nil)
	mockInput.On("Prompt", mock.Anything, "", "").Return("", nil)

	// Mock RPC base URL for logs client
	mockAPI.On("GetRPCBaseURL").Return("https://api.example.com")

	// Note: We'll skip testing the actual log streaming since it requires real network connections
	// Just test that the parameters are gathered correctly

	// Create a context with cancellation to avoid hanging on the real network call
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	err := handler.TailLogs(cancelCtx, "us-west-2", "test")

	// Expect an error due to context cancellation, not network issues
	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
}

// Test handler methods directly instead of command structs.
func (s *CloudTestSuite) TestHandler_Deploy() {
	handler, mockAPI, mockConfig, mockInput, _ := s.createTestHandler()
	ctx := context.Background()
	project := s.createTestProject()

	// Mock config
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock preview deployment
	previewResponse := models.DeploymentPreview{
		OrgName:        "Test Org",
		OrgSlug:        "test-org",
		ProjectName:    "Test Project",
		ProjectSlug:    "test-project",
		ExecutorName:   "test-executor",
		DeploymentType: models.DeploymentTypeDeploy,
		TickRate:       20,
		Regions:        []string{"us-west-2"},
	}
	mockAPI.On("PreviewDeployment", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeDeploy).
		Return(previewResponse, nil)

	// Mock confirmation
	mockInput.On("Confirm", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(true, nil)

	// Mock deployment API calls
	mockAPI.On("DeployProject", mock.Anything, "test-org-id", "test-project-id", models.DeploymentTypeDeploy).
		Return(nil)

	// Mock deployment status polling
	statusResponse := []byte(`{
		"data": {
			"dev": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "created"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil)

	err := handler.Deployment(ctx, "test-org-id", project, models.DeploymentTypeDeploy)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
	mockInput.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_Status() {
	handler, mockAPI, _, _, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	project := s.createTestProject()

	// Mock deployment status
	statusResponse := []byte(`{
		"data": {
			"dev": {
				"project_id": "test-project-id",
				"created_by": "test-user",
				"created_at": "2023-01-01T00:00:00Z",
				"deployment_status": "created"
			}
		}
	}`)
	mockAPI.On("GetDeploymentStatus", mock.Anything, "test-project-id").Return(statusResponse, nil)

	err := handler.Status(ctx, org, project)

	s.Require().NoError(err)
	mockAPI.AssertExpectations(s.T())
}

func (s *CloudTestSuite) TestHandler_TailLogs() {
	handler, mockAPI, mockConfig, mockInput, _ := s.createTestHandler()
	ctx := context.Background()

	// Mock config
	testConfig := config.Config{
		OrganizationID: "test-org-id",
		ProjectID:      "test-project-id",
	}
	mockConfig.On("GetConfig").Return(&testConfig).Maybe()

	// Mock API calls
	mockAPI.On("GetOrganizationByID", mock.Anything, "test-org-id").
		Return(s.createTestOrganization(), nil)
	mockAPI.On("GetProjectByID", mock.Anything, "test-org-id", "test-project-id").
		Return(s.createTestProject(), nil)

	// Mock input prompts
	mockInput.On("Prompt", mock.Anything, "Choose an environment", "1").Return("1", nil)
	mockInput.On("Prompt", mock.Anything, "", "").Return("", nil)

	// Mock RPC base URL for logs client
	mockAPI.On("GetRPCBaseURL").Return("https://api.example.com")

	// Create a context with cancellation to avoid hanging on the real network call
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	err := handler.TailLogs(cancelCtx, "us-west-2", "test")

	// Expect an error due to context cancellation, not network issues
	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
	mockAPI.AssertExpectations(s.T())
	mockConfig.AssertExpectations(s.T())
}
