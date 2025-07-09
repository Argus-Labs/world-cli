package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"pkg.world.dev/world-cli/cmd/world/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/cmd/world/internal/services/input"
	"pkg.world.dev/world-cli/cmd/world/user"
)

// UserTestSuite defines the test suite for user package.
type UserTestSuite struct {
	suite.Suite
}

// Helper method to create fresh mocks and handler for each test.
func (s *UserTestSuite) createTestHandler() (*user.Handler, *api.MockClient, *input.MockService) {
	mockAPIClient := &api.MockClient{}
	mockInputService := &input.MockService{}

	handler := user.NewHandler(
		mockAPIClient,
		mockInputService,
	).(*user.Handler)

	return handler, mockAPIClient, mockInputService
}

// Test fixtures.
func (s *UserTestSuite) createTestOrganization() models.Organization {
	return models.Organization{
		ID:   "org-123",
		Name: "Test Organization",
		Slug: "test_org",
	}
}

func (s *UserTestSuite) createTestUser() models.User {
	return models.User{
		ID:    "user-123",
		Name:  "Test User",
		Email: "test@example.com",
	}
}

// TestUserSuite runs the test suite.
func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}

func (s *UserTestSuite) TestHandler_InviteToOrganization_Success() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.InviteUserToOrganizationFlags{
		Email: "invite@example.com",
		Role:  "member",
	}

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter user email to invite", "invite@example.com").
		Return("invite@example.com", nil)
	mockInputService.On(
		"Select", ctx, "Available Roles", "Select a role by number", []string{"member", "admin", "owner", "none"}, 0).
		Return(0, nil)

	// Mock API call
	mockAPIClient.On("InviteUserToOrganization", ctx, "org-123", "invite@example.com", "member").
		Return(nil)

	err := handler.InviteToOrganization(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_InviteToOrganization_EmptyEmail() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.InviteUserToOrganizationFlags{
		Email: "",
		Role:  "member",
	}

	// Mock input interactions - user enters empty email
	mockInputService.On("Prompt", ctx, "Enter user email to invite", "").
		Return("", nil)

	err := handler.InviteToOrganization(ctx, org, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "email cannot be empty")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_InviteToOrganization_InputError() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.InviteUserToOrganizationFlags{}

	// Mock input error
	inputErr := errors.New("input error")
	mockInputService.On("Prompt", ctx, "Enter user email to invite", "").
		Return("", inputErr)

	err := handler.InviteToOrganization(ctx, org, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get user email")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_InviteToOrganization_APIError() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.InviteUserToOrganizationFlags{
		Email: "invite@example.com",
		Role:  "admin",
	}

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter user email to invite", "invite@example.com").
		Return("invite@example.com", nil)
	mockInputService.On(
		"Select", ctx, "Available Roles", "Select a role by number", []string{"member", "admin", "owner", "none"}, 1).
		Return(1, nil)

	// Mock API error
	apiErr := errors.New("API error")
	mockAPIClient.On("InviteUserToOrganization", ctx, "org-123", "invite@example.com", "admin").
		Return(apiErr)

	err := handler.InviteToOrganization(ctx, org, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to invite user to organization")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_InviteToOrganization_EmailFailed() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.InviteUserToOrganizationFlags{
		Email: "invite@example.com",
		Role:  "member",
	}

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter user email to invite", "invite@example.com").
		Return("invite@example.com", nil)
	mockInputService.On(
		"Select", ctx, "Available Roles", "Select a role by number", []string{"member", "admin", "owner", "none"}, 0).
		Return(0, nil)

	// Mock API error with email failed message
	emailErr := errors.New("Organization email invite failed, but invite is still created in CLI.")
	mockAPIClient.On("InviteUserToOrganization", ctx, "org-123", "invite@example.com", "member").
		Return(emailErr)

	err := handler.InviteToOrganization(ctx, org, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to invite user to organization")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_ChangeRoleInOrganization_Success() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.ChangeUserRoleInOrganizationFlags{
		Email: "user@example.com",
		Role:  "admin",
	}

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter user email to update", "user@example.com").
		Return("user@example.com", nil)
	mockInputService.On(
		"Select", ctx, "Available Roles", "Select a role by number", []string{"member", "admin", "owner", "none"}, 1).
		Return(1, nil)

	// Mock API call
	mockAPIClient.On("UpdateUserRoleInOrganization", ctx, "org-123", "user@example.com", "admin").
		Return(nil)

	err := handler.ChangeRoleInOrganization(ctx, org, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_ChangeRoleInOrganization_EmptyEmail() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.ChangeUserRoleInOrganizationFlags{
		Email: "",
		Role:  "admin",
	}

	// Mock input interactions - user enters empty email
	mockInputService.On("Prompt", ctx, "Enter user email to update", "").
		Return("", nil)

	err := handler.ChangeRoleInOrganization(ctx, org, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "email cannot be empty")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_ChangeRoleInOrganization_InputError() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.ChangeUserRoleInOrganizationFlags{}

	// Mock input error
	inputErr := errors.New("input error")
	mockInputService.On("Prompt", ctx, "Enter user email to update", "").
		Return("", inputErr)

	err := handler.ChangeRoleInOrganization(ctx, org, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get user email")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_ChangeRoleInOrganization_APIError() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	org := s.createTestOrganization()
	flags := models.ChangeUserRoleInOrganizationFlags{
		Email: "user@example.com",
		Role:  "member",
	}

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter user email to update", "user@example.com").
		Return("user@example.com", nil)
	mockInputService.On(
		"Select", ctx, "Available Roles", "Select a role by number", []string{"member", "admin", "owner", "none"}, 0).
		Return(0, nil)

	// Mock API error
	apiErr := errors.New("API error")
	mockAPIClient.On("UpdateUserRoleInOrganization", ctx, "org-123", "user@example.com", "member").
		Return(apiErr)

	err := handler.ChangeRoleInOrganization(ctx, org, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to set user role in organization")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_Update_Success() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	currentUser := s.createTestUser()
	flags := models.UpdateUserFlags{
		Name: "Updated User",
	}

	// Mock getting current user
	mockAPIClient.On("GetUser", ctx).
		Return(currentUser, nil)

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter name", "Updated User").
		Return("Updated User", nil)

	// Mock API call
	mockAPIClient.On("UpdateUser", ctx, "Updated User", "test@example.com").
		Return(nil)

	err := handler.Update(ctx, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_Update_EmptyName() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	currentUser := s.createTestUser()
	flags := models.UpdateUserFlags{
		Name: "",
	}

	// Mock getting current user
	mockAPIClient.On("GetUser", ctx).
		Return(currentUser, nil)

	// Mock input interactions - user enters empty name first, then valid name
	mockInputService.On("Prompt", ctx, "Enter name", "Test User").
		Return("", nil).Once()
	mockInputService.On("Prompt", ctx, "Enter name", "Test User").
		Return("Valid User", nil).Once()

	// Mock API call
	mockAPIClient.On("UpdateUser", ctx, "Valid User", "test@example.com").
		Return(nil)

	err := handler.Update(ctx, flags)

	s.Require().NoError(err)
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_Update_GetUserError() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	flags := models.UpdateUserFlags{
		Name: "Updated User",
	}

	// Mock getting current user error
	getUserErr := errors.New("get user error")
	mockAPIClient.On("GetUser", ctx).
		Return(models.User{}, getUserErr)

	err := handler.Update(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to get current user")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_Update_InputError() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	currentUser := s.createTestUser()
	flags := models.UpdateUserFlags{}

	// Mock getting current user
	mockAPIClient.On("GetUser", ctx).
		Return(currentUser, nil)

	// Mock input error
	inputErr := errors.New("input error")
	mockInputService.On("Prompt", ctx, "Enter name", "Test User").
		Return("", inputErr)

	err := handler.Update(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to input user name")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_Update_APIError() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	ctx := context.Background()
	currentUser := s.createTestUser()
	flags := models.UpdateUserFlags{
		Name: "Updated User",
	}

	// Mock getting current user
	mockAPIClient.On("GetUser", ctx).
		Return(currentUser, nil)

	// Mock input interactions
	mockInputService.On("Prompt", ctx, "Enter name", "Updated User").
		Return("Updated User", nil)

	// Mock API error
	apiErr := errors.New("update user error")
	mockAPIClient.On("UpdateUser", ctx, "Updated User", "test@example.com").
		Return(apiErr)

	err := handler.Update(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "Failed to update user")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}

func (s *UserTestSuite) TestHandler_Update_ContextCanceled() {
	s.T().Parallel()

	handler, mockAPIClient, mockInputService := s.createTestHandler()
	currentUser := s.createTestUser()
	flags := models.UpdateUserFlags{}

	// Cancel context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Mock getting current user
	mockAPIClient.On("GetUser", ctx).
		Return(currentUser, nil)

	// Mock input interaction that will fail due to canceled context
	mockInputService.On("Prompt", ctx, "Enter name", "Test User").
		Return("", context.Canceled)

	err := handler.Update(ctx, flags)

	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
	mockAPIClient.AssertExpectations(s.T())
	mockInputService.AssertExpectations(s.T())
}
