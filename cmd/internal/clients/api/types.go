package api

import (
	"context"
	"io"
	"net/http"
	"time"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

// Interface implementation check.
var _ ClientInterface = &Client{}

// Client implements HTTP API client with retry logic and authentication.
type Client struct {
	BaseURL string
	// TODO: Remove this once we have a proper RPC client
	RPCURL     string
	Token      string
	HTTPClient HTTPClientInterface
}

// ClientInterface defines the contract for making API calls.
// This interface focuses on business operations rather than low-level HTTP details.
type ClientInterface interface {
	// API-specific methods that return parsed models
	GetUser(ctx context.Context) (models.User, error)
	UpdateUser(ctx context.Context, name, email, avatarURL string) error
	UpdateUserRoleInOrganization(ctx context.Context, orgID, userEmail, role string) error
	InviteUserToOrganization(ctx context.Context, orgID, userEmail, role string) error
	GetOrganizations(ctx context.Context) ([]models.Organization, error)
	GetOrganizationsInvitedTo(ctx context.Context) ([]models.Organization, error)
	AcceptOrganizationInvitation(ctx context.Context, orgID string) error
	GetProjects(ctx context.Context, orgID string) ([]models.Project, error)
	LookupProjectFromRepo(ctx context.Context, repoURL, repoPath string) (models.Project, error)
	GetOrganizationByID(ctx context.Context, id string) (models.Organization, error)
	GetProjectByID(ctx context.Context, projID, orgID string) (models.Project, error)
	CreateOrganization(ctx context.Context, name, slug, avatarURL string) (models.Organization, error)
	GetListRegions(ctx context.Context, orgID, projID string) ([]string, error)
	CheckProjectSlugIsTaken(ctx context.Context, orgID, projID, slug string) error
	CreateProject(ctx context.Context, orgID string, project models.Project) (models.Project, error)
	UpdateProject(ctx context.Context, orgID, projID string, project models.Project) (models.Project, error)
	DeleteProject(ctx context.Context, orgID, projID string) error
	PreviewDeployment(ctx context.Context, orgID, projID, deployType string) (models.DeploymentPreview, error)

	// Cloud deployment methods
	DeployProject(
		ctx context.Context,
		orgID, projID, deployType, commitHash string,
		imageReader io.Reader,
		successPush bool,
	) error
	ResetDestroyPromoteProject(ctx context.Context, orgID, projID, deployType string) error
	GetTemporaryCredential(ctx context.Context, orgID, projID string) (models.TemporaryCredential, error)
	GetDeploymentStatus(ctx context.Context, projID string) ([]byte, error)
	GetHealthStatus(ctx context.Context, projID string) ([]byte, error)
	GetDeploymentHealthStatus(ctx context.Context, projID string) (map[string]models.DeploymentHealthCheckResult, error)
	// Utility methods
	SetAuthToken(token string)
	// TODO: Remove this once we have a proper RPC client
	GetRPCBaseURL() string
}

// HTTPClientInterface allows for mocking the underlying HTTP client.
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

// RequestConfig holds configuration for individual requests.
type RequestConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	Timeout       time.Duration
	ContentType   string
	CustomHeaders map[string]string
}
