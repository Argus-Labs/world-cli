package api

import (
	"context"
	"net/http"
	"time"

	"pkg.world.dev/world-cli/cmd/internal/models"
)

// Client implements HTTP API client with retry logic and authentication.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient HTTPClientInterface
}

// ClientInterface defines the contract for making API calls.
// This interface focuses on business operations rather than low-level HTTP details.
type ClientInterface interface {
	// API-specific methods that return parsed models
	GetUser(ctx context.Context) (models.User, error)
	GetOrganizations(ctx context.Context) ([]models.Organization, error)
	GetOrganizationsInvitedTo(ctx context.Context) ([]models.Organization, error)
	AcceptOrganizationInvitation(ctx context.Context, orgID string) error
	GetProjects(ctx context.Context, orgID string) ([]models.Project, error)
	LookupProjectFromRepo(ctx context.Context, repoURL, repoPath string) (models.Project, error)
	GetOrganizationByID(ctx context.Context, id string) (models.Organization, error)
	GetProjectByID(ctx context.Context, id string) (models.Project, error)

	// Authentication
	SetAuthToken(token string)
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
