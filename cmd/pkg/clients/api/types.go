package api

import (
	"context"
	"net/http"
	"time"

	"pkg.world.dev/world-cli/cmd/pkg/models"
)

// Client implements HTTP API client with retry logic and authentication.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient HTTPClientInterface
}

// ClientInterface defines the contract for making authenticated HTTP requests.
type ClientInterface interface {
	// Core HTTP methods
	Get(ctx context.Context, endpoint string) ([]byte, error)
	Post(ctx context.Context, endpoint string, body interface{}) ([]byte, error)
	Put(ctx context.Context, endpoint string, body interface{}) ([]byte, error)
	Delete(ctx context.Context, endpoint string) ([]byte, error)

	// API-specific methods that return parsed models
	GetUser(ctx context.Context) (models.User, error)
	GetOrganizations(ctx context.Context) ([]models.Organization, error)
	GetOrganizationsInvitedTo(ctx context.Context) ([]models.Organization, error)
	AcceptOrganizationInvitation(ctx context.Context, orgID string) error
	GetProjects(ctx context.Context, orgID string) ([]models.Project, error)
	LookupProjectFromRepo(ctx context.Context, repoURL, repoPath string) (models.Project, error)
	GetOrganizationByID(ctx context.Context, id string) (models.Organization, error)
	GetProjectByID(ctx context.Context, id string) (models.Project, error)

	// Utility methods
	SetAuthToken(token string)
	ParseResponse(body []byte, result interface{}) error
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
