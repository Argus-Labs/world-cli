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

// UpdateUser mocks updating user information.
func (m *MockClient) UpdateUser(ctx context.Context, name, email string) error {
	args := m.Called(ctx, name, email)
	return args.Error(0)
}

// UpdateUserRoleInOrganization mocks updating user role in organization.
func (m *MockClient) UpdateUserRoleInOrganization(ctx context.Context, orgID, userEmail, role string) error {
	args := m.Called(ctx, orgID, userEmail, role)
	return args.Error(0)
}

// InviteUserToOrganization mocks inviting a user to an organization.
func (m *MockClient) InviteUserToOrganization(ctx context.Context, orgID, userEmail, role string) error {
	args := m.Called(ctx, orgID, userEmail, role)
	return args.Error(0)
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
func (m *MockClient) GetProjectByID(ctx context.Context, orgID, projID string) (models.Project, error) {
	args := m.Called(ctx, orgID, projID)
	return args.Get(0).(models.Project), args.Error(1)
}

// CreateOrganization mocks creating an organization.
func (m *MockClient) CreateOrganization(
	ctx context.Context,
	name, slug string,
) (models.Organization, error) {
	args := m.Called(ctx, name, slug)
	return args.Get(0).(models.Organization), args.Error(1)
}

// GetListRegions mocks getting list of regions.
func (m *MockClient) GetListRegions(ctx context.Context, orgID, projID string) ([]string, error) {
	args := m.Called(ctx, orgID, projID)
	return args.Get(0).([]string), args.Error(1)
}

// CheckProjectSlugIsTaken mocks checking if a project slug is taken.
func (m *MockClient) CheckProjectSlugIsTaken(ctx context.Context, orgID, projID, slug string) error {
	args := m.Called(ctx, orgID, projID, slug)
	return args.Error(0)
}

// CreateProject mocks creating a project.
func (m *MockClient) CreateProject(ctx context.Context, orgID string, project models.Project) (models.Project, error) {
	args := m.Called(ctx, orgID, project)
	return args.Get(0).(models.Project), args.Error(1)
}

// UpdateProject mocks updating a project.
func (m *MockClient) UpdateProject(
	ctx context.Context,
	orgID, projID string,
	project models.Project,
) (models.Project, error) {
	args := m.Called(ctx, orgID, projID, project)
	return args.Get(0).(models.Project), args.Error(1)
}

// DeleteProject mocks deleting a project.
func (m *MockClient) DeleteProject(ctx context.Context, orgID, projID string) error {
	args := m.Called(ctx, orgID, projID)
	return args.Error(0)
}

// PreviewDeployment mocks previewing a deployment.
func (m *MockClient) PreviewDeployment(
	ctx context.Context,
	orgID, projID, deployType string,
) (models.DeploymentPreview, error) {
	args := m.Called(ctx, orgID, projID, deployType)
	return args.Get(0).(models.DeploymentPreview), args.Error(1)
}

// DeployProject mocks deploying a project.
func (m *MockClient) DeployProject(
	ctx context.Context,
	orgID, projID, deployType string,
) error {
	args := m.Called(ctx, orgID, projID, deployType)
	return args.Error(0)
}

// GetTemporaryCredential mocks getting temporary credential.

func (m *MockClient) GetTemporaryCredential(
	ctx context.Context,
	orgID, projID string,
) (models.TemporaryCredential, error) {
	args := m.Called(ctx, orgID, projID)
	return args.Get(0).(models.TemporaryCredential), args.Error(1)
}

// GetDeploymentStatus mocks getting deployment status.
func (m *MockClient) GetDeploymentStatus(ctx context.Context, projID string) ([]byte, error) {
	args := m.Called(ctx, projID)
	return args.Get(0).([]byte), args.Error(1)
}

// GetHealthStatus mocks getting health status.
func (m *MockClient) GetHealthStatus(ctx context.Context, projID string) ([]byte, error) {
	args := m.Called(ctx, projID)
	return args.Get(0).([]byte), args.Error(1)
}

// GetDeploymentHealthStatus mocks getting deployment health status.
func (m *MockClient) GetDeploymentHealthStatus(
	ctx context.Context,
	projID string,
) (map[string]models.DeploymentHealthCheckResult, error) {
	args := m.Called(ctx, projID)
	return args.Get(0).(map[string]models.DeploymentHealthCheckResult), args.Error(1)
}

// GetOrganizationMembers mocks getting organization members.
func (m *MockClient) GetOrganizationMembers(ctx context.Context, orgID string) ([]models.OrganizationMember, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.OrganizationMember), args.Error(1)
}

// Utility methods

// SetAuthToken mocks setting auth token.
func (m *MockClient) SetAuthToken(token string) {
	m.Called(token)
}

// TODO: Remove this once we have a proper RPC client
// GetRPCBaseURL mocks getting RPC base URL.
func (m *MockClient) GetRPCBaseURL() string {
	return m.Called().String(0)
}

// Authentication methods

// GetLoginLink mocks getting the login link.
func (m *MockClient) GetLoginLink(ctx context.Context) (LoginLinkResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(LoginLinkResponse), args.Error(1)
}

// GetLoginToken mocks getting the login token.
func (m *MockClient) GetLoginToken(ctx context.Context, callbackURL string) (models.LoginToken, error) {
	args := m.Called(ctx, callbackURL)
	return args.Get(0).(models.LoginToken), args.Error(1)
}

// MockHTTPClient is a mock implementation of HTTPClientInterface for testing.
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}
