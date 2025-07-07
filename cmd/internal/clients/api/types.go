package api

import (
	"context"
	"net/http"
	"time"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

const (
	get  = http.MethodGet
	post = http.MethodPost
	put  = http.MethodPut
	del  = http.MethodDelete
)

var (
	ErrNoOrganizationID = eris.New("organization ID is required")
	ErrNoProjectID      = eris.New("project ID is required")
	ErrNoProjectSlug    = eris.New("project slug is required")
)

// Interface implementation check.
var _ ClientInterface = &Client{}

// Client implements HTTP API client with retry logic and authentication.
type Client struct {
	BaseURL        string
	ArgusIDBaseURL string
	// TODO: Remove this once we have a proper RPC client
	RPCURL     string
	Token      string
	HTTPClient HTTPClientInterface
}

// ClientInterface defines the contract for making API calls.
// This interface focuses on business operations rather than low-level HTTP details.
type ClientInterface interface {
	// ========================================
	// Authentication Methods
	// ========================================

	// GetLoginLink initiates the login flow by getting the login URLs from ArgusID
	GetLoginLink(ctx context.Context) (LoginLinkResponse, error)
	// GetLoginToken polls the callback URL to get the login token status
	GetLoginToken(ctx context.Context, callbackURL string) (models.LoginToken, error)
	// SetAuthToken updates the client's authentication token for API requests
	SetAuthToken(token string)

	// ========================================
	// User Management Methods
	// ========================================

	// GetUser retrieves the current authenticated user's information
	GetUser(ctx context.Context) (models.User, error)
	// UpdateUser updates the current user's profile information
	UpdateUser(ctx context.Context, name, email string) error
	// InviteUserToOrganization invites a user to join an organization with a specific role
	InviteUserToOrganization(ctx context.Context, orgID, userEmail, role string) error
	// UpdateUserRoleInOrganization updates a user's role within an organization
	UpdateUserRoleInOrganization(ctx context.Context, orgID, userEmail, role string) error
	// GetOrganizationsInvitedTo retrieves organizations the user has been invited to
	GetOrganizationsInvitedTo(ctx context.Context) ([]models.Organization, error)
	// AcceptOrganizationInvitation accepts an invitation to join an organization
	AcceptOrganizationInvitation(ctx context.Context, orgID string) error

	// ========================================
	// Organization Management Methods
	// ========================================

	// GetOrganizations retrieves all organizations the user belongs to
	GetOrganizations(ctx context.Context) ([]models.Organization, error)
	// GetOrganizationByID retrieves a specific organization by its ID
	GetOrganizationByID(ctx context.Context, id string) (models.Organization, error)
	// CreateOrganization creates a new organization
	CreateOrganization(ctx context.Context, name, slug string) (models.Organization, error)

	// ========================================
	// Project Management Methods
	// ========================================

	// GetProjects retrieves all projects within an organization
	GetProjects(ctx context.Context, orgID string) ([]models.Project, error)
	// GetProjectByID retrieves a specific project by its ID
	GetProjectByID(ctx context.Context, projID, orgID string) (models.Project, error)
	// LookupProjectFromRepo looks up a project based on repository URL and path
	LookupProjectFromRepo(ctx context.Context, repoURL, repoPath string) (models.Project, error)
	// CreateProject creates a new project within an organization
	CreateProject(ctx context.Context, orgID string, project models.Project) (models.Project, error)
	// UpdateProject updates an existing project's configuration
	UpdateProject(ctx context.Context, orgID, projID string, project models.Project) (models.Project, error)
	// DeleteProject removes a project from an organization
	DeleteProject(ctx context.Context, orgID, projID string) error
	// CheckProjectSlugIsTaken verifies if a project slug is available
	CheckProjectSlugIsTaken(ctx context.Context, orgID, projID, slug string) error

	// ========================================
	// Cloud Deployment Methods
	// ========================================

	// GetListRegions retrieves available deployment regions for a project
	GetListRegions(ctx context.Context, orgID, projID string) ([]string, error)
	// PreviewDeployment shows what would happen during a deployment without executing it
	PreviewDeployment(ctx context.Context, orgID, projID, deployType string) (models.DeploymentPreview, error)
	// DeployProject deploy, resets, destroys, or promotes a project
	DeployProject(ctx context.Context, orgID, projID, deployType string) error
	// GetTemporaryCredential retrieves temporary credentials for a project
	GetTemporaryCredential(ctx context.Context, orgID, projID string) (models.TemporaryCredential, error)
	// GetDeploymentStatus retrieves the current deployment status for a project
	GetDeploymentStatus(ctx context.Context, projID string) ([]byte, error)
	// GetHealthStatus retrieves the health status of deployed services
	GetHealthStatus(ctx context.Context, projID string) ([]byte, error)
	// GetDeploymentHealthStatus retrieves detailed health check results for deployments
	GetDeploymentHealthStatus(ctx context.Context, projID string) (map[string]models.DeploymentHealthCheckResult, error)

	// ========================================
	// Utility Methods
	// ========================================

	// GetRPCBaseURL returns the RPC service base URL
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

// Login link response structure.
type LoginLinkResponse struct {
	CallBackURL string `json:"callbackUrl"`
	ClientURL   string `json:"clientUrl"`
}
