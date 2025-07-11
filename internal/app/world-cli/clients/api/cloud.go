package api

import (
	"context"
	"fmt"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

// ========================================
// Cloud Deployment Methods
// ========================================

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

// DeployProject deploy, resets, destroys, or promotes a project.
func (c *Client) DeployProject(
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
