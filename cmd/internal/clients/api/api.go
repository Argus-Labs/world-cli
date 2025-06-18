package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rotisserie/eris"
	"github.com/tidwall/gjson"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

const (
	get  = http.MethodGet
	post = http.MethodPost
	put  = http.MethodPut
)

// API-specific methods that implement the business logic calls

// GetUser retrieves the current user information.
func (c *Client) GetUser(ctx context.Context) (models.User, error) {
	body, err := c.sendRequest(ctx, get, "/api/user", nil)
	if err != nil {
		return models.User{}, eris.Wrap(err, "Failed to get user")
	}
	return parseResponse[models.User](body)
}

// GetOrganizations retrieves the list of organizations the user belongs to.
func (c *Client) GetOrganizations(ctx context.Context) ([]models.Organization, error) {
	body, err := c.sendRequest(ctx, get, "/api/organization", nil)
	if err != nil {
		return []models.Organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	return parseResponse[[]models.Organization](body)
}

// GetOrganizationsInvitedTo retrieves organizations the user has been invited to.
func (c *Client) GetOrganizationsInvitedTo(ctx context.Context) ([]models.Organization, error) {
	body, err := c.sendRequest(ctx, get, "/api/organization/invited", nil)
	if err != nil {
		return []models.Organization{}, eris.Wrap(err, "Failed to get organization invitations")
	}

	return parseResponse[[]models.Organization](body)
}

// AcceptOrganizationInvitation accepts an invitation to join an organization.
func (c *Client) AcceptOrganizationInvitation(ctx context.Context, orgID string) error {
	endpoint := fmt.Sprintf("/api/organization/%s/accept-invitation", orgID)
	_, err := c.sendRequest(ctx, post, endpoint, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to accept organization invitation")
	}
	return nil
}

// GetProjects retrieves all projects for a given organization.
func (c *Client) GetProjects(ctx context.Context, orgID string) ([]models.Project, error) {
	endpoint := fmt.Sprintf("/api/organization/%s/project", orgID)
	body, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return []models.Project{}, eris.Wrap(err, "Failed to get projects")
	}

	return parseResponse[[]models.Project](body)
}

// LookupProjectFromRepo looks up a project based on repository URL and path.
func (c *Client) LookupProjectFromRepo(ctx context.Context, repoURL, repoPath string) (models.Project, error) {
	deployURL := fmt.Sprintf("/api/project/?url=%s&path=%s", url.QueryEscape(repoURL), url.QueryEscape(repoPath))
	body, err := c.sendRequest(ctx, get, deployURL, nil)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to lookup project from repo")
	}
	proj, err := parseResponse[models.Project](body)
	if err != nil && err.Error() != "Missing data field in response" {
		// missing data field in response just means nothing was found
		// but any other error is a problem
		return models.Project{}, eris.Wrap(err, "Failed to parse response")
	}
	return proj, nil
}

// GetOrganizationByID retrieves a specific organization by ID.
func (c *Client) GetOrganizationByID(ctx context.Context, id string) (models.Organization, error) {
	endpoint := fmt.Sprintf("/api/organization/%s", id)
	body, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return models.Organization{}, eris.Wrap(err, "Failed to get organization by ID")
	}
	return parseResponse[models.Organization](body)
}

// GetProjectByID retrieves a specific project by ID.
func (c *Client) GetProjectByID(ctx context.Context, id string) (models.Project, error) {
	endpoint := fmt.Sprintf("/api/project/%s", id)
	body, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to get project by ID")
	}
	return parseResponse[models.Project](body)
}

// parseResponse is a generic version that returns the parsed data.
func parseResponse[T any](body []byte) (T, error) {
	result := gjson.GetBytes(body, "data")
	if !result.Exists() {
		return *new(T), eris.New("Missing data field in response")
	}

	var data T
	if err := json.Unmarshal([]byte(result.Raw), &data); err != nil {
		return *new(T), eris.Wrap(err, "Failed to parse response")
	}

	return data, nil
}
