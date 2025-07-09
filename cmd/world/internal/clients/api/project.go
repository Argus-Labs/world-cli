package api

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
)

// ErrProjectSlugAlreadyExists is passed from forge to world-cli, Must always match.
var ErrProjectSlugAlreadyExists = eris.New("project slug already exists")

// ========================================
// Project Management Methods
// ========================================

// GetProjects retrieves all projects for a given organization.
func (c *Client) GetProjects(ctx context.Context, orgID string) ([]models.Project, error) {
	endpoint := fmt.Sprintf("/api/organization/%s/project", orgID)
	body, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return []models.Project{}, eris.Wrap(err, "Failed to get projects")
	}

	return parseResponse[[]models.Project](body)
}

// GetProjectByID retrieves a specific project by ID.
func (c *Client) GetProjectByID(ctx context.Context, orgID, projID string) (models.Project, error) {
	if orgID == "" {
		return models.Project{}, ErrNoOrganizationID
	}
	if projID == "" {
		return models.Project{}, ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s", orgID, projID)
	body, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return models.Project{}, eris.Wrap(err, "Failed to get project by ID")
	}
	return parseResponse[models.Project](body)
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

// CheckProjectSlugIsTaken checks if a project slug is already taken.
func (c *Client) CheckProjectSlugIsTaken(ctx context.Context, orgID, projID, slug string) error {
	if orgID == "" {
		return ErrNoOrganizationID
	}
	if projID == "" {
		return ErrNoProjectID
	}
	if slug == "" {
		return ErrNoProjectSlug
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s/%s/check_slug", orgID, projID, slug)
	_, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to check project slug")
	}

	return nil
}

// GetListRegions retrieves available regions for a project.
func (c *Client) GetListRegions(ctx context.Context, orgID, projID string) ([]string, error) {
	if orgID == "" {
		return nil, ErrNoOrganizationID
	}
	if projID == "" {
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
