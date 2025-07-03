package api

import (
	"context"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

var (
	ErrNoUserEmail = eris.New("user email is required")
	ErrNoUserName  = eris.New("user name is required")
)

// ========================================
// User Management Methods
// ========================================

// GetUser retrieves the current user information.
func (c *Client) GetUser(ctx context.Context) (models.User, error) {
	body, err := c.sendRequest(ctx, get, "/api/user", nil)
	if err != nil {
		return models.User{}, eris.Wrap(err, "Failed to get user")
	}
	return parseResponse[models.User](body)
}

// UpdateUser updates the current user information.
func (c *Client) UpdateUser(ctx context.Context, name, email string) error {
	if email == "" {
		return ErrNoUserEmail
	}
	if name == "" {
		return ErrNoUserName
	}

	payload := models.User{
		Name:  name,
		Email: email,
	}

	_, err := c.sendRequest(ctx, put, "/api/user", payload)
	if err != nil {
		return eris.Wrap(err, "Failed to update user")
	}

	return nil
}
