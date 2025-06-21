package cmdsetup

import (
	"context"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/pkg/models"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
// Setup Service Mock
///////////////////////////////////////////////////////////////////////////////////////////////////

// Ensure MockService implements the interface.
var _ models.SetupServiceInterface = (*MockService)(nil)

// MockService is a mock implementation of SetupServiceInterface.
type MockService struct {
	mock.Mock
}

// SetupCommandState mocks the setup command.
func (m *MockService) SetupCommandState(
	ctx context.Context,
	req models.SetupRequest,
) (*models.CommandState, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CommandState), args.Error(1)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Organization Handler Mock
///////////////////////////////////////////////////////////////////////////////////////////////////

// Ensure MockOrganizationHandler implements the interface.
var _ OrganizationHandler = (*MockOrganizationHandler)(nil)

// MockOrganizationHandler is a mock implementation of OrganizationHandler.
type MockOrganizationHandler struct {
	mock.Mock
}

// PromptForOrganization mocks selecting an organization.
func (m *MockOrganizationHandler) PromptForOrganization(
	ctx context.Context,
	orgs []models.Organization,
	createNew bool,
) (models.Organization, error) {
	args := m.Called(ctx, orgs, createNew)
	return args.Get(0).(models.Organization), args.Error(1)
}

// CreateOrganization mocks creating an organization.
func (m *MockOrganizationHandler) CreateOrganization(
	ctx context.Context, flags models.CreateOrganizationFlags) (models.Organization, error) {
	args := m.Called(ctx, flags)
	return args.Get(0).(models.Organization), args.Error(1)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Project Handler Mock
///////////////////////////////////////////////////////////////////////////////////////////////////

var _ ProjectHandler = (*MockProjectHandler)(nil)

// MockProjectHandler is a mock implementation of ProjectHandler.
type MockProjectHandler struct {
	mock.Mock
}

// PromptForProject mocks selecting a project.
func (m *MockProjectHandler) PromptForProject(
	ctx context.Context,
	projects []models.Project,
	createNew bool,
) (models.Project, error) {
	args := m.Called(ctx, projects, createNew)
	return args.Get(0).(models.Project), args.Error(1)
}

// CreateProject mocks creating a project.
func (m *MockProjectHandler) Create(
	ctx context.Context,
	state *models.CommandState,
	flags models.CreateProjectFlags,
) (models.Project, error) {
	args := m.Called(ctx, state, flags)
	return args.Get(0).(models.Project), args.Error(1)
}

// ProjectPreCreateUpdateValidation mocks validating a project before creation.
func (m *MockProjectHandler) ProjectPreCreateUpdateValidation() (string, string, error) {
	args := m.Called()
	return args.String(0), args.String(1), args.Error(2)
}
