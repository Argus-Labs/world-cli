package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"github.com/rotisserie/eris"
	"github.com/tidwall/gjson"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

const (
	get  = http.MethodGet
	post = http.MethodPost
	put  = http.MethodPut
	del  = http.MethodDelete
)

// ErrOrganizationSlugAlreadyExists is passed from forge to world-cli, Must always match.
var ErrOrganizationSlugAlreadyExists = eris.New("organization slug already exists")

// ErrProjectSlugAlreadyExists is passed from forge to world-cli, Must always match.
var ErrProjectSlugAlreadyExists = eris.New("project slug already exists")

var (
	ErrNoOrganizationID = eris.New("organization ID is required")
	ErrNoProjectID      = eris.New("project ID is required")
	ErrNoProjectSlug    = eris.New("project slug is required")
)

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
	if id == "" {
		return models.Organization{}, ErrNoOrganizationID
	}

	endpoint := fmt.Sprintf("/api/organization/%s", id)
	body, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return models.Organization{}, eris.Wrap(err, "Failed to get organization by ID")
	}
	return parseResponse[models.Organization](body)
}

// GetProjectByID retrieves a specific project by ID.
func (c *Client) GetProjectByID(ctx context.Context, orgID, projID string) (models.Project, error) {
	if projID == "" {
		return models.Project{}, ErrNoProjectID
	}
	if orgID == "" {
		return models.Project{}, ErrNoOrganizationID
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s", orgID, projID)
	body, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to get project by ID")
	}
	return parseResponse[models.Project](body)
}

// CreateOrganization creates a new organization.
func (c *Client) CreateOrganization(ctx context.Context, name, slug, avatarURL string) (models.Organization, error) {
	payload := map[string]string{
		"name":       name,
		"slug":       slug,
		"avatar_url": avatarURL,
	}

	body, err := c.sendRequest(ctx, post, "/api/organization", payload)
	if err != nil {
		return models.Organization{}, eris.Wrap(err, "Failed to create organization")
	}

	return parseResponse[models.Organization](body)
}

// GetListRegions retrieves available regions for a project.
func (c *Client) GetListRegions(ctx context.Context, orgID, projID string) ([]string, error) {
	if orgID == "" {
		return nil, ErrNoOrganizationID
	} else if projID == "" {
		return nil, ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s/regions", orgID, projID)
	body, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get regions")
	}

	regionMap, err := parseResponse[map[string]bool](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse regions")
	}

	regions := make([]string, 0, len(regionMap))
	for region := range regionMap {
		regions = append(regions, region)
	}

	// Sort regions for consistent output
	sort.Strings(regions)
	return regions, nil
}

// CheckProjectSlugIsTaken checks if a project slug is already taken.
func (c *Client) CheckProjectSlugIsTaken(ctx context.Context, orgID, projID, slug string) error {
	if orgID == "" {
		return ErrNoOrganizationID
	}
	if slug == "" {
		return ErrNoProjectSlug
	}
	if projID == "" {
		return ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s/%s/check_slug", orgID, projID, slug)
	_, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to check project slug")
	}

	return nil
}

// CreateProject creates a new project.
func (c *Client) CreateProject(ctx context.Context, orgID string, project models.Project) (models.Project, error) {
	if orgID == "" {
		return models.Project{}, ErrNoOrganizationID
	}

	payload := map[string]interface{}{
		"name":       project.Name,
		"slug":       project.Slug,
		"repo_url":   project.RepoURL,
		"repo_token": project.RepoToken,
		"repo_path":  project.RepoPath,
		"org_id":     orgID,
		"config":     project.Config,
		"avatar_url": project.AvatarURL,
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project", orgID)
	body, err := c.sendRequest(ctx, post, endpoint, payload)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to create project")
	}

	return parseResponse[models.Project](body)
}

// UpdateProject updates a project.
func (c *Client) UpdateProject(
	ctx context.Context,
	orgID, projID string,
	project models.Project,
) (models.Project, error) {
	if orgID == "" {
		return models.Project{}, ErrNoOrganizationID
	}
	if projID == "" {
		return models.Project{}, ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s", orgID, projID)
	body, err := c.sendRequest(ctx, put, endpoint, map[string]interface{}{
		"name":       project.Name,
		"slug":       project.Slug,
		"repo_url":   project.RepoURL,
		"repo_token": project.RepoToken,
		"repo_path":  project.RepoPath,
		"config":     project.Config,
		"avatar_url": project.AvatarURL,
	})
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to update project")
	}

	return parseResponse[models.Project](body)
}

// DeleteProject deletes a project.
func (c *Client) DeleteProject(ctx context.Context, orgID, projID string) error {
	if orgID == "" {
		return ErrNoOrganizationID
	}
	if projID == "" {
		return ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s", orgID, projID)
	_, err := c.sendRequest(ctx, del, endpoint, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to delete project")
	}

	return nil
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
