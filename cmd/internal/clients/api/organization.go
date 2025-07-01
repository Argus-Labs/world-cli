package api

import (
	"context"
	"fmt"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

// ErrOrganizationSlugAlreadyExists is passed from forge to world-cli, Must always match.
var ErrOrganizationSlugAlreadyExists = eris.New("organization slug already exists")

// ========================================
// Organization Management Methods
// ========================================

// GetOrganizations retrieves the list of organizations the user belongs to.
func (c *Client) GetOrganizations(ctx context.Context) ([]models.Organization, error) {
	body, err := c.sendRequest(ctx, get, "/api/organization", nil)
	if err != nil {
		return []models.Organization{}, eris.Wrap(err, "Failed to get organizations")
	}

	return parseResponse[[]models.Organization](body)
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
