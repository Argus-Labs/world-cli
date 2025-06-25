package api

import (
	"context"
	"net/http"

	"github.com/stretchr/testify/mock"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

// Ensure MockClient implements the interface.
var _ ClientInterface = (*MockClient)(nil)

// MockClient is a mock implementation of ClientInterface.
type MockClient struct {
	mock.Mock
}

// API-specific methods

// GetUser mocks getting user information.
func (m *MockClient) GetUser(ctx context.Context) (models.User, error) {
	args := m.Called(ctx)
	return args.Get(0).(models.User), args.Error(1)
}

// GetOrganizations mocks getting organizations.
func (m *MockClient) GetOrganizations(ctx context.Context) ([]models.Organization, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Organization), args.Error(1)
}

// GetOrganizationsInvitedTo mocks getting organization invitations.
func (m *MockClient) GetOrganizationsInvitedTo(ctx context.Context) ([]models.Organization, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Organization), args.Error(1)
}

// AcceptOrganizationInvitation mocks accepting an organization invitation.
func (m *MockClient) AcceptOrganizationInvitation(ctx context.Context, orgID string) error {
	args := m.Called(ctx, orgID)
	return args.Error(0)
}

// GetProjects mocks getting projects for an organization.
func (m *MockClient) GetProjects(ctx context.Context, orgID string) ([]models.Project, error) {
	args := m.Called(ctx, orgID)
	return args.Get(0).([]models.Project), args.Error(1)
}

// LookupProjectFromRepo mocks looking up a project from repository information.
func (m *MockClient) LookupProjectFromRepo(ctx context.Context, repoURL, repoPath string) (models.Project, error) {
	args := m.Called(ctx, repoURL, repoPath)
	return args.Get(0).(models.Project), args.Error(1)
}

// GetOrganizationByID mocks getting an organization by ID.
func (m *MockClient) GetOrganizationByID(ctx context.Context, id string) (models.Organization, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.Organization), args.Error(1)
}

// GetProjectByID mocks getting a project by ID.
func (m *MockClient) GetProjectByID(ctx context.Context, id string) (models.Project, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(models.Project), args.Error(1)
}

// Authentication

// SetAuthToken mocks setting auth token.
func (m *MockClient) SetAuthToken(token string) {
	m.Called(token)
}

// MockHTTPClient is a mock implementation of HTTPClientInterface for testing.
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}
