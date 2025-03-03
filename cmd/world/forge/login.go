package forge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/globalconfig"
)

var (
	maxAttempts = 12 // 12 * 5 = 1 minute
)

// login will open browser to login and save the token to the config file
func login(ctx context.Context) error {
	key := generateKey()
	url := fmt.Sprintf("%s?key=%s", loginURL, key)

	// Open browser
	err := openBrowser(url)
	if err != nil {
		return eris.Wrap(err, "Failed to open browser")
	}

	// Keep the selected org and project to be used after login
	config, err := globalconfig.GetGlobalConfig()
	if err != nil {
		// no config found, so we need to select the org and project
		config = globalconfig.GlobalConfig{}
	}

	orgID := config.OrganizationID
	projectID := config.ProjectID

	// Wait for user to login
	url = fmt.Sprintf("%s?key=%s", getTokenURL, key)
	token, err := getToken(ctx, url)
	if err != nil {
		return eris.Wrap(err, "Failed to get token")
	}

	// Parse jwt token to get name from metadata
	cred, err := parseCredential(token)
	if err != nil {
		return eris.Wrap(err, "Failed to get name from token")
	}

	// Save credential to config
	config.Credential = cred
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return eris.Wrap(err, "Failed to save credential")
	}

	// Handle organization selection
	orgID, err = handleOrganizationSelection(ctx, orgID)
	if err != nil {
		orgID = ""
	}

	// save orgID to config
	config.OrganizationID = orgID
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return eris.Wrap(err, "Failed to save organization ID")
	}

	// Handle project selection
	projectID, err = handleProjectSelection(ctx, projectID)
	if err != nil {
		projectID = ""
	}

	// save projectID to config
	config.ProjectID = projectID

	// Save config
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return eris.Wrap(err, "Failed to save credential")
	}

	// show the org list
	err = showOrganizationList(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to show organization list")
	}

	// show the project list
	err = showProjectList(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to show project list")
	}

	fmt.Println("\n✨ Login successful! ✨")
	fmt.Println("=====================")
	fmt.Printf("\n👋 Welcome, %s!\n", cred.Name)
	fmt.Printf("🆔 Your ID is: %s\n", cred.ID)
	fmt.Println("\n🚀 You're all set to start using World Forge!")

	return nil
}

// GetToken will get the token from the config file
func getToken(ctx context.Context, url string) (string, error) {
	// Create request every 3 seconds to check if the token is available
	attempts := 1

	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second): //nolint:gomnd
			fmt.Printf("\r🔄 Logging in... attempt %d", attempts)

			// Create request with context
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return "", eris.Wrap(err, "failed to create request")
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				return "", eris.Wrap(err, "failed to get token")
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Println("\n✨ Login token received successfully!")
				// Read the token from the response
				response, err := io.ReadAll(resp.Body)
				if err != nil {
					return "", eris.Wrap(err, "failed to read token")
				}
				token, err := parseResponse[string](response)
				if err != nil {
					return "", eris.Wrap(err, "failed to parse token")
				}
				return *token, nil
			}
			attempts++
		}
	}
	fmt.Println() // Add newline before error
	return "", eris.New("max attempts reached while waiting for token")
}

// parseCredential will parse the id and name from the token
func parseCredential(token string) (globalconfig.Credential, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 { //nolint:gomnd
		return globalconfig.Credential{}, eris.New("invalid token format")
	}

	// Get the payload (second part) of the JWT
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "failed to decode token payload")
	}

	// Parse the JSON payload
	var claims struct {
		UserMetadata struct {
			Name string `json:"name"`
			ID   string `json:"sub"`
		} `json:"user_metadata"`
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "failed to parse token claims")
	}

	if claims.UserMetadata.Name == "" {
		return globalconfig.Credential{}, eris.New("name not found in token")
	}

	return globalconfig.Credential{
		Token: token,
		Name:  claims.UserMetadata.Name,
		ID:    claims.UserMetadata.ID,
	}, nil
}
