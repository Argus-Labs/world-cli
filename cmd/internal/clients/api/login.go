package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

// ========================================
// Authentication Methods
// ========================================

// GetLoginLink gets the login link from ArgusID service.
func (c *Client) GetLoginLink(ctx context.Context) (LoginLinkResponse, error) {
	authURL := fmt.Sprintf("%s/api/auth/service-auth-session", c.ArgusIDBaseURL)

	// Make direct HTTP request since this goes to ArgusID, not the main API
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, nil)
	if err != nil {
		return LoginLinkResponse{}, eris.Wrap(err, "Failed to create request")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return LoginLinkResponse{}, eris.Wrap(err, "Failed to get login link")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LoginLinkResponse{}, eris.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return LoginLinkResponse{}, eris.Wrap(err, "Failed to read login link")
	}

	var loginLink LoginLinkResponse
	err = json.Unmarshal(body, &loginLink)
	if err != nil {
		return LoginLinkResponse{}, eris.Wrap(err, "Failed to parse login link")
	}

	return loginLink, nil
}

// GetLoginToken polls the callback URL for login token.
func (c *Client) GetLoginToken(ctx context.Context, callbackURL string) (models.LoginToken, error) {
	// Make direct HTTP request since this goes to ArgusID, not the main API
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, callbackURL, nil)
	if err != nil {
		return models.LoginToken{}, eris.Wrap(err, "Failed to create request")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return models.LoginToken{}, eris.Wrap(err, "Failed to get login token")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.LoginToken{}, eris.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.LoginToken{}, eris.Wrap(err, "Failed to read login token")
	}

	var token models.LoginToken
	err = json.Unmarshal(body, &token)
	if err != nil {
		return models.LoginToken{}, eris.Wrap(err, "Failed to parse login token")
	}

	return token, nil
}
