package organization_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/cmd/internal/services/input"
	"pkg.world.dev/world-cli/cmd/world/organization"
	"pkg.world.dev/world-cli/cmd/world/project"
)

// OrganizationTestSuite defines the test suite for organization package.
type OrganizationTestSuite struct {
	suite.Suite
}

// Helper method to create fresh mocks and handler for each test.
func (s *OrganizationTestSuite) createTestHandler() (
	*organization.Handler, *project.MockHandler, *input.MockService, *api.MockClient, *config.MockService) {
	mockProjectHandler := &project.MockHandler{}
	mockInputService := &input.MockService{}
	mockAPIClient := &api.MockClient{}
	mockConfigService := &config.MockService{}

	handler := organization.NewHandler(
		mockProjectHandler,
		mockInputService,
		mockAPIClient,
		mockConfigService,
	).(*organization.Handler)

	return handler, mockProjectHandler, mockInputService, mockAPIClient, mockConfigService
}

// Test fixtures.
func (s *OrganizationTestSuite) createTestOrganization() models.Organization {
	return models.Organization{
		ID:   "org-123",
		Name: "Test Organization",
		Slug: "test_org",
	}
}

func (s *OrganizationTestSuite) createTestConfig() *config.Config {
	return &config.Config{
		OrganizationID:  "org-123",
		ProjectID:       "proj-456",
		CurrRepoKnown:   false,
		CurrProjectName: "Test Project",
	}
}

// TestOrganizationSuite runs the test suite.
func TestOrganizationSuite(t *testing.T) {
	suite.Run(t, new(OrganizationTestSuite))
}

func (s *OrganizationTestSuite) TestHandler_Create_Success() {
	s.T().Parallel()

	handler, _, mockInputService, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateOrganizationFlags{
		Name: "Test Organization",
		Slug: "test-org",
	}

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter organization name", "Test Organization").
		Return("Test Organization", nil)
	mockInputService.On("Prompt", ctx, "Enter organization slug", "test_org").
		Return("test_org", nil)
	mockInputService.On("Confirm", ctx, "Create organization with these details? (Y/n)", "n").Return(true, nil)

	// Mock API call
	mockAPIClient.On("CreateOrganization", ctx, "Test Organization", "test_org").
		Return(testOrg, nil)

	// Mock config operations
	mockConfigService.On("GetConfig").Return(s.createTestConfig())
	mockConfigService.On("Save").Return(nil)

	result, err := handler.Create(ctx, flags)

	s.Require().NoError(err)
	s.Equal(testOrg, result)
	mockInputService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Create_InvalidName() {
	s.T().Parallel()

	handler, _, mockInputService, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	flags := models.CreateOrganizationFlags{
		Name: "", // Empty name to trigger validation
	}

	// Mock input interactions - user enters empty name first, then valid name
	mockInputService.On("Prompt", ctx, "Enter organization name", "").Return("", nil).Once()
	mockInputService.On("Prompt", ctx, "Enter organization name", "").Return("Good Organization", nil).Once()
	mockInputService.On("Prompt", ctx, "Enter organization slug", "goodorganizatio").
		Return("goodorganizatio", nil)
	mockInputService.On("Confirm", ctx, "Create organization with these details? (Y/n)", "n").Return(true, nil)

	testOrg := models.Organization{
		ID:   "org-123",
		Name: "Good Organization",
		Slug: "goodorganizatio",
	}

	mockAPIClient.On("CreateOrganization", ctx, "Good Organization", "goodorganizatio").Return(testOrg, nil)
	mockConfigService.On("GetConfig").Return(s.createTestConfig())
	mockConfigService.On("Save").Return(nil)

	result, err := handler.Create(ctx, flags)

	s.Require().NoError(err)
	s.Equal(testOrg, result)
	mockInputService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Create_UserDeclinesThenAccepts() {
	s.T().Parallel()

	handler, _, mockInputService, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	flags := models.CreateOrganizationFlags{
		Name: "Test Organization",
	}

	// First iteration: user declines
	mockInputService.On("Prompt", ctx, "Enter organization name", "Test Organization").
		Return("Test Organization", nil).
		Once()
	mockInputService.On("Prompt", ctx, "Enter organization slug", "testorganizatio").
		Return("testorganizatio", nil).
		Once()
	mockInputService.On("Confirm", ctx, "Create organization with these details? (Y/n)", "n").
		Return(false, nil).
		Once()

	// Second iteration: user accepts (with different name to show change)
	mockInputService.On("Prompt", ctx, "Enter organization name", "Test Organization").
		Return("Final Organization", nil).
		Once()
	mockInputService.On("Prompt", ctx, "Enter organization slug", "finalorganizati").
		Return("finalorganizati", nil).
		Once()
	mockInputService.On("Confirm", ctx, "Create organization with these details? (Y/n)", "n").
		Return(true, nil).
		Once()

	finalOrg := models.Organization{
		ID:   "org-456",
		Name: "Final Organization",
		Slug: "finalorganizati",
	}
	mockAPIClient.On("CreateOrganization", ctx, "Final Organization", "finalorganizati").Return(finalOrg, nil)
	mockConfigService.On("GetConfig").Return(s.createTestConfig())
	mockConfigService.On("Save").Return(nil)

	result, err := handler.Create(ctx, flags)

	s.Require().NoError(err)
	s.Equal(finalOrg, result)
	mockInputService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Create_SlugAlreadyExists() {
	s.T().Parallel()

	handler, _, mockInputService, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	flags := models.CreateOrganizationFlags{
		Name: "Test Organization",
		Slug: "test-org",
	}

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter organization name", "Test Organization").
		Return("Test Organization", nil)
	mockInputService.On("Prompt", ctx, "Enter organization slug", "test_org").
		Return("test_org", nil)
	mockInputService.On("Confirm", ctx, "Create organization with these details? (Y/n)", "n").Return(true, nil)

	// API call fails with slug already exists error - function returns error immediately
	slugExistsErr := errors.New("organization slug already exists")
	mockAPIClient.On("CreateOrganization", ctx, "Test Organization", "test_org").
		Return(models.Organization{}, slugExistsErr)

	// No config operations expected since the method returns error immediately
	result, err := handler.Create(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to create organization")
	s.Equal(models.Organization{}, result)
	mockInputService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Create_InputError() {
	s.T().Parallel()

	handler, _, mockInputService, _, _ := s.createTestHandler()
	ctx := context.Background()
	flags := models.CreateOrganizationFlags{}

	// Mock input error
	inputErr := errors.New("input error")
	mockInputService.On("Prompt", ctx, "Enter organization name", "").Return("", inputErr)

	result, err := handler.Create(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get organization name")
	s.Equal(models.Organization{}, result)
	mockInputService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Switch_WithSlug_Success() {
	s.T().Parallel()

	handler, mockProjectHandler, _, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.SwitchOrganizationFlags{
		Slug: "test_org",
	}

	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false

	orgs := []models.Organization{testOrg}

	// Mock config service
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls
	mockAPIClient.On("GetOrganizations", ctx).Return(orgs, nil)

	// Mock project handler
	mockProjectHandler.On("HandleSwitch", ctx, testOrg).Return(nil)

	result, err := handler.Switch(ctx, flags)

	s.Require().NoError(err)
	s.Equal(testOrg, result)
	mockProjectHandler.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Switch_WithSlug_NotFound() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchOrganizationFlags{
		Slug: "nonexistent-org",
	}

	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false

	orgs := []models.Organization{s.createTestOrganization()}

	// Mock config service
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock API calls
	mockAPIClient.On("GetOrganizations", ctx).Return(orgs, nil)

	result, err := handler.Switch(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Organization not found with slug:")
	s.Equal(models.Organization{}, result)
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Switch_CurrentRepoKnown() {
	s.T().Parallel()

	handler, _, _, _, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchOrganizationFlags{}

	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = true
	cfg.CurrProjectName = "Test Project"

	// Mock config service
	mockConfigService.On("GetConfig").Return(cfg)

	result, err := handler.Switch(ctx, flags)

	s.Require().Error(err)
	s.Equal(organization.ErrCannotSwitchOrganization, err)
	s.Equal(models.Organization{}, result)
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Switch_NoOrganizations() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchOrganizationFlags{}

	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false

	// Mock config service
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock API calls - return empty organizations list
	mockAPIClient.On("GetOrganizations", ctx).Return([]models.Organization{}, nil)

	result, err := handler.Switch(ctx, flags)

	s.Require().NoError(err)
	s.Equal(models.Organization{}, result)
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Switch_MultipleOrganizations_Success() {
	s.T().Parallel()

	handler, mockProjectHandler, mockInputService, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchOrganizationFlags{}

	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false

	testOrg := s.createTestOrganization()
	orgs := []models.Organization{testOrg}

	// Mock config service
	mockConfigService.On("GetConfig").Return(cfg)
	mockConfigService.On("Save").Return(nil)

	// Mock API calls
	mockAPIClient.On("GetOrganizations", ctx).Return(orgs, nil)

	// Mock input service for PromptForSwitch
	mockInputService.On("Prompt", ctx, "Enter organization number ('q' to quit)", "").Return("1", nil)

	// Mock project handler
	mockProjectHandler.On("HandleSwitch", ctx, testOrg).Return(nil)

	result, err := handler.Switch(ctx, flags)

	s.Require().NoError(err)
	s.Equal(testOrg, result)
	mockProjectHandler.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_PromptForSwitch_ValidSelection() {
	s.T().Parallel()

	handler, _, mockInputService, _, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	orgs := []models.Organization{testOrg}

	// Mock input service
	mockInputService.On("Prompt", ctx, "Enter organization number ('q' to quit)", "").Return("1", nil)

	// Mock config service - GetConfig is called by saveOrganization
	mockConfigService.On("GetConfig").Return(s.createTestConfig())
	mockConfigService.On("Save").Return(nil)

	result, err := handler.PromptForSwitch(ctx, orgs, false)

	s.Require().NoError(err)
	s.Equal(testOrg, result)
	mockInputService.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_PromptForSwitch_UserQuits() {
	s.T().Parallel()

	handler, _, mockInputService, _, _ := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	orgs := []models.Organization{testOrg}

	// Mock input service - user quits
	mockInputService.On("Prompt", ctx, "Enter organization number ('q' to quit)", "").Return("q", nil)

	result, err := handler.PromptForSwitch(ctx, orgs, false)

	s.Require().Error(err)
	s.Equal(organization.ErrOrganizationSelectionCanceled, err)
	s.Equal(models.Organization{}, result)
	mockInputService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_PromptForSwitch_CreateNew() {
	s.T().Parallel()

	handler, _, mockInputService, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	orgs := []models.Organization{testOrg}

	// Mock input service - user chooses to create new
	mockInputService.On("Prompt", ctx, "Enter organization number ('c' to create new or 'q' to quit)", "").
		Return("c", nil)

	// Mock inputs for create flow
	mockInputService.On("Prompt", ctx, "Enter organization name", "").Return("New Organization", nil)
	mockInputService.On("Prompt", ctx, "Enter organization slug", "neworganization").Return("neworganization", nil)
	mockInputService.On("Confirm", ctx, "Create organization with these details? (Y/n)", "n").Return(true, nil)

	newOrg := models.Organization{
		ID:   "org-new",
		Name: "New Organization",
		Slug: "neworganization",
	}

	// Mock API calls
	mockAPIClient.On("CreateOrganization", ctx, "New Organization", "neworganization").Return(newOrg, nil)

	// Mock config service
	mockConfigService.On("GetConfig").Return(s.createTestConfig())
	mockConfigService.On("Save").Return(nil)

	result, err := handler.PromptForSwitch(ctx, orgs, true)

	s.Require().NoError(err)
	s.Equal(newOrg, result)
	mockInputService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_PromptForSwitch_InvalidSelection() {
	s.T().Parallel()

	handler, _, mockInputService, _, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	orgs := []models.Organization{testOrg}

	// Mock input service - user enters invalid selection first, then valid
	mockInputService.On("Prompt", ctx, "Enter organization number ('q' to quit)", "").Return("999", nil).Once()
	mockInputService.On("Prompt", ctx, "Enter organization number ('q' to quit)", "").Return("1", nil).Once()

	// Mock config service - GetConfig is called by saveOrganization when user selects valid org
	mockConfigService.On("GetConfig").Return(s.createTestConfig())
	mockConfigService.On("Save").Return(nil)

	result, err := handler.PromptForSwitch(ctx, orgs, false)

	s.Require().NoError(err)
	s.Equal(testOrg, result)
	mockInputService.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_PromptForSwitch_ContextCanceled() {
	s.T().Parallel()

	handler, _, mockInputService, _, _ := s.createTestHandler()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context immediately

	testOrg := s.createTestOrganization()
	orgs := []models.Organization{testOrg}

	// Mock input service to return context canceled error
	mockInputService.On("Prompt", ctx, "Enter organization number ('q' to quit)", "").Return("", context.Canceled)

	result, err := handler.PromptForSwitch(ctx, orgs, false)

	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
	s.Equal(models.Organization{}, result)
	mockInputService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Switch_APIError() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	flags := models.SwitchOrganizationFlags{}

	cfg := s.createTestConfig()
	cfg.CurrRepoKnown = false

	// Mock config service
	mockConfigService.On("GetConfig").Return(cfg)

	// Mock API error
	apiErr := errors.New("API error")
	mockAPIClient.On("GetOrganizations", ctx).Return([]models.Organization{}, apiErr)

	result, err := handler.Switch(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get organizations")
	s.Equal(models.Organization{}, result)
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_Create_SaveError() {
	s.T().Parallel()

	handler, _, mockInputService, mockAPIClient, mockConfigService := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	flags := models.CreateOrganizationFlags{
		Name: "Test Organization",
	}

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter organization name", "Test Organization").
		Return("Test Organization", nil)
	mockInputService.On("Prompt", ctx, "Enter organization slug", "testorganizatio").Return("testorganizatio", nil)
	mockInputService.On("Confirm", ctx, "Create organization with these details? (Y/n)", "n").Return(true, nil)

	// Mock API call
	mockAPIClient.On("CreateOrganization", ctx, "Test Organization", "testorganizatio").Return(testOrg, nil)

	// Mock config operations - save fails
	mockConfigService.On("GetConfig").Return(s.createTestConfig())
	saveErr := errors.New("save error")
	mockConfigService.On("Save").Return(saveErr)

	result, err := handler.Create(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to save organization")
	s.Equal(models.Organization{}, result)
	mockInputService.AssertExpectations(s.T())
	mockAPIClient.AssertExpectations(s.T())
	mockConfigService.AssertExpectations(s.T())
}

func (s *OrganizationTestSuite) TestHandler_PromptForSwitch_InputError() {
	s.T().Parallel()

	handler, _, mockInputService, _, _ := s.createTestHandler()
	ctx := context.Background()
	testOrg := s.createTestOrganization()
	orgs := []models.Organization{testOrg}

	// Mock input error
	inputErr := errors.New("input error")
	mockInputService.On("Prompt", ctx, "Enter organization number ('q' to quit)", "").Return("", inputErr)

	result, err := handler.PromptForSwitch(ctx, orgs, false)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get organization number")
	s.Equal(models.Organization{}, result)
	mockInputService.AssertExpectations(s.T())
}

// TestHandler_MembersList_Success tests successful members list retrieval.
func (s *OrganizationTestSuite) TestHandler_MembersList_Success() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.MembersListFlags{
		IncludeRemoved: false,
	}

	// Create test members with different roles
	testMembers := []models.OrganizationMember{
		{
			Role: models.RoleOwner,
			User: models.User{Name: "Owner User", Email: "owner@example.com"},
		},
		{
			Role: models.RoleAdmin,
			User: models.User{Name: "Admin User", Email: "admin@example.com"},
		},
		{
			Role: models.RoleMember,
			User: models.User{Name: "Member User", Email: "member@example.com"},
		},
	}

	mockAPIClient.On("GetOrganizationMembers", ctx, org.ID).Return(testMembers, nil)

	err := handler.MembersList(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
}

// TestHandler_MembersList_WithRemovedList tests members list with removed list flag.
func (s *OrganizationTestSuite) TestHandler_MembersList_WithRemovedList() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.MembersListFlags{
		IncludeRemoved: true,
	}

	// Create test members including RoleNone (removed members)
	testMembers := []models.OrganizationMember{
		{
			Role: models.RoleOwner,
			User: models.User{Name: "Owner User", Email: "owner@example.com"},
		},
		{
			Role: models.RoleNone,
			User: models.User{Name: "Removed User", Email: "removed@example.com"},
		},
	}

	mockAPIClient.On("GetOrganizationMembers", ctx, org.ID).Return(testMembers, nil)

	err := handler.MembersList(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
}

// TestHandler_MembersList_WithoutRemovedList tests members list without removed list flag.
func (s *OrganizationTestSuite) TestHandler_MembersList_WithoutRemovedList() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.MembersListFlags{
		IncludeRemoved: false,
	}

	// Create test members including RoleNone (removed members)
	testMembers := []models.OrganizationMember{
		{
			Role: models.RoleOwner,
			User: models.User{Name: "Owner User", Email: "owner@example.com"},
		},
		{
			Role: models.RoleNone,
			User: models.User{Name: "Removed User", Email: "removed@example.com"},
		},
	}

	mockAPIClient.On("GetOrganizationMembers", ctx, org.ID).Return(testMembers, nil)

	err := handler.MembersList(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
}

// TestHandler_MembersList_EmptyMembers tests members list with no members.
func (s *OrganizationTestSuite) TestHandler_MembersList_EmptyMembers() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.MembersListFlags{
		IncludeRemoved: false,
	}

	// Empty members list
	testMembers := []models.OrganizationMember{}

	mockAPIClient.On("GetOrganizationMembers", ctx, org.ID).Return(testMembers, nil)

	err := handler.MembersList(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
}

// TestHandler_MembersList_AllRoles tests members list with all possible roles.
func (s *OrganizationTestSuite) TestHandler_MembersList_AllRoles() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.MembersListFlags{
		IncludeRemoved: true,
	}

	// Create test members with all possible roles
	testMembers := []models.OrganizationMember{
		{
			Role: models.RoleOwner,
			User: models.User{Name: "Owner User", Email: "owner@example.com"},
		},
		{
			Role: models.RoleAdmin,
			User: models.User{Name: "Admin User 1", Email: "admin1@example.com"},
		},
		{
			Role: models.RoleAdmin,
			User: models.User{Name: "Admin User 2", Email: "admin2@example.com"},
		},
		{
			Role: models.RoleMember,
			User: models.User{Name: "Member User 1", Email: "member1@example.com"},
		},
		{
			Role: models.RoleMember,
			User: models.User{Name: "Member User 2", Email: "member2@example.com"},
		},
		{
			Role: models.RoleMember,
			User: models.User{Name: "Member User 3", Email: "member3@example.com"},
		},
		{
			Role: models.RoleNone,
			User: models.User{Name: "Removed User", Email: "removed@example.com"},
		},
	}

	mockAPIClient.On("GetOrganizationMembers", ctx, org.ID).Return(testMembers, nil)

	err := handler.MembersList(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
}

// TestHandler_MembersList_InvalidRole tests members list with invalid role handling.
func (s *OrganizationTestSuite) TestHandler_MembersList_InvalidRole() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.MembersListFlags{
		IncludeRemoved: false,
	}

	// Create test members with invalid role
	testMembers := []models.OrganizationMember{
		{
			Role: models.Role("invalid_role"),
			User: models.User{Name: "Invalid User", Email: "invalid@example.com"},
		},
		{
			Role: models.RoleOwner,
			User: models.User{Name: "Owner User", Email: "owner@example.com"},
		},
	}

	mockAPIClient.On("GetOrganizationMembers", ctx, org.ID).Return(testMembers, nil)

	err := handler.MembersList(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
}

// TestHandler_MembersList_APIError tests members list when API call fails.
func (s *OrganizationTestSuite) TestHandler_MembersList_APIError() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.MembersListFlags{
		IncludeRemoved: false,
	}

	// Mock API call to return error
	mockAPIClient.On("GetOrganizationMembers", ctx, org.ID).Return(nil, errors.New("API error"))

	err := handler.MembersList(ctx, org, flags)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "API error")
	s.Require().Contains(err.Error(), "MembersList: Failed to get organization members")
	mockAPIClient.AssertExpectations(s.T())
}

// TestHandler_MembersList_EmptyUserData tests members list with empty user data.
func (s *OrganizationTestSuite) TestHandler_MembersList_EmptyUserData() {
	s.T().Parallel()

	handler, _, _, mockAPIClient, _ := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.MembersListFlags{
		IncludeRemoved: false,
	}

	// Create test members with empty user data
	testMembers := []models.OrganizationMember{
		{
			Role: models.RoleOwner,
			User: models.User{Name: "", Email: ""},
		},
		{
			Role: models.RoleAdmin,
			User: models.User{Name: "Admin User", Email: ""},
		},
		{
			Role: models.RoleMember,
			User: models.User{Name: "", Email: "member@example.com"},
		},
	}

	mockAPIClient.On("GetOrganizationMembers", ctx, org.ID).Return(testMembers, nil)

	err := handler.MembersList(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
}
