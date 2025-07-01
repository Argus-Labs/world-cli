package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	ErrNoUserEmail      = eris.New("user email is required")
	ErrNoUserName       = eris.New("user name is required")
	ErrNoUserAvatarURL  = eris.New("user avatar URL is required")
)

// GetUser retrieves the current user information.
func (c *Client) GetUser(ctx context.Context) (models.User, error) {
	body, err := c.sendRequest(ctx, get, "/api/user", nil)
	if err != nil {
		return models.User{}, eris.Wrap(err, "Failed to get user")
	}
	return parseResponse[models.User](body)
}

// UpdateUser updates the current user information.
func (c *Client) UpdateUser(ctx context.Context, name, email, avatarURL string) error {
	if email == "" {
		return ErrNoUserEmail
	}
	if name == "" {
		return ErrNoUserName
	}
	if avatarURL == "" {
		return ErrNoUserAvatarURL
	}

	payload := models.User{
		Name:      name,
		Email:     email,
		AvatarURL: avatarURL,
	}

	_, err := c.sendRequest(ctx, put, "/api/user", payload)
	if err != nil {
		return eris.Wrap(err, "Failed to update user")
	}

	return nil
}

// UpdateUserRoleInOrganization updates the role of a user in an organization.
func (c *Client) UpdateUserRoleInOrganization(ctx context.Context, orgID, userEmail, role string) error {
	_, err := c.sendRequest(ctx, post, fmt.Sprintf("/api/organization/%s/update-role", orgID), map[string]string{
		"target_user_email": userEmail,
		"role":              role,
	})
	if err != nil {
		return eris.Wrap(err, "Failed to update user role in organization")
	}

	return nil
}

// InviteUserToOrganization invites a user to an organization.
func (c *Client) InviteUserToOrganization(ctx context.Context, orgID, userEmail, role string) error {
	_, err := c.sendRequest(ctx, post, fmt.Sprintf("/api/organization/%s/invite", orgID), map[string]string{
		"invited_user_email": userEmail,
		"role":               role,
	})
	if err != nil {
		return eris.Wrap(err, "Failed to invite user to organization")
	}

	return nil
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

// PreviewDeployment previews a deployment.
func (c *Client) PreviewDeployment(
	ctx context.Context,
	orgID, projID, deployType string,
) (models.DeploymentPreview, error) {
	endpoint := fmt.Sprintf("/api/organization/%s/project/%s/%s?preview=true", orgID, projID, deployType)
	resultBytes, err := c.sendRequest(ctx, post, endpoint, nil)
	if err != nil {
		return models.DeploymentPreview{}, eris.Wrap(err, fmt.Sprintf("Failed to %s project", deployType))
	}

	return parseResponse[models.DeploymentPreview](resultBytes)
}

// DeployProject deploys a project with multipart upload.
func (c *Client) DeployProject(
	ctx context.Context,
	orgID, projID, deployType, commitHash string,
	imageReader io.Reader, successPush bool,
) error {
	if orgID == "" {
		return ErrNoOrganizationID
	}
	if projID == "" {
		return ErrNoProjectID
	}
	/* create multipart request */
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// add commit_hash to the request
	err := writer.WriteField("commit_hash", commitHash)
	if err != nil {
		return eris.Wrap(err, "Failed to write commit hash")
	}

	// if the image was not pushed to the registry in the local machine, add the image to the request
	// World Forge will push the image to the registry
	if !successPush {
		// add the image to the request
		part, err := writer.CreateFormFile("file", "image.tar")
		if err != nil {
			return eris.Wrap(err, "Failed to create form file")
		}
		_, err = io.Copy(part, imageReader)
		if err != nil {
			return eris.Wrap(err, "Failed to copy image to request")
		}
	} else {
		deployType = "deploy?nofile=true"
	}

	writer.Close()
	/* end of multipart request */

	if deployType == "forceDeploy" {
		deployType = "deploy?force=true"
		if successPush {
			deployType = "deploy?force=true&nofile=true"
		}
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s/%s", orgID, projID, deployType)
	// Create request with proper Content-Type for multipart
	req, err := c.prepareRequest(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to create request")
	}
	// Override body and content type for multipart
	req.Body = io.NopCloser(body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Use the existing retry logic
	_, err = c.makeRequestWithRetries(ctx, req)
	if err != nil {
		return eris.Wrap(err, fmt.Sprintf("Failed to %s project", deployType))
	}
	return nil
}

// DeployProject deploys a project with multipart upload.
func (c *Client) ResetDestroyPromoteProject(
	ctx context.Context,
	orgID, projID, deployType string,
) error {
	if orgID == "" {
		return ErrNoOrganizationID
	}
	if projID == "" {
		return ErrNoProjectID
	}
	endpoint := fmt.Sprintf("/api/organization/%s/project/%s/%s", orgID, projID, deployType)

	_, err := c.sendRequest(ctx, post, endpoint, nil)
	if err != nil {
		return eris.Wrap(err, fmt.Sprintf("Failed to %s project", deployType))
	}

	return nil
}

func (c *Client) GetTemporaryCredential(ctx context.Context, orgID, projID string) (models.TemporaryCredential, error) {
	if orgID == "" {
		return models.TemporaryCredential{}, ErrNoOrganizationID
	}
	if projID == "" {
		return models.TemporaryCredential{}, ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/organization/%s/project/%s/temporary-credential", orgID, projID)
	result, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return models.TemporaryCredential{}, eris.Wrap(err, "Failed to get temporary credential")
	}

	return parseResponse[models.TemporaryCredential](result)
}

// GetDeploymentStatus retrieves deployment status.
func (c *Client) GetDeploymentStatus(ctx context.Context, projID string) ([]byte, error) {
	if projID == "" {
		return nil, ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/deployment/%s", projID)
	return c.sendRequest(ctx, get, endpoint, nil)
}

// GetHealthStatus retrieves health status.
func (c *Client) GetHealthStatus(ctx context.Context, projID string) ([]byte, error) {
	if projID == "" {
		return nil, ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/health/%s", projID)
	return c.sendRequest(ctx, get, endpoint, nil)
}

// GetDeploymentHealthStatus retrieves deployment health status.
func (c *Client) GetDeploymentHealthStatus(
	ctx context.Context,
	projID string,
) (map[string]models.DeploymentHealthCheckResult, error) {
	if projID == "" {
		return nil, ErrNoProjectID
	}

	endpoint := fmt.Sprintf("/api/health/%s", projID)
	result, err := c.sendRequest(ctx, get, endpoint, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get deployment status")
	}

	return parseResponse[map[string]models.DeploymentHealthCheckResult](result)
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
