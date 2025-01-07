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

	// Wait for user to login
	fmt.Println("Waiting for user to login...")
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

	fmt.Println("Login successful")
	fmt.Println("Welcome, ", cred.Name)
	fmt.Println("Your ID is: ", cred.ID)

	// Save token and name to config file
	config := globalconfig.GlobalConfig{
		Credential: cred,
	}
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return eris.Wrap(err, "Failed to save credential")
	}

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
			fmt.Println("Logging in... attempt", attempts)

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
