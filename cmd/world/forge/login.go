package forge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/globalconfig"
)

const (
	// ArgusID Service URL
	argusIDServiceURL = "https://id.argus-dev.com/api/auth/service-auth-session"
)

var (
	maxLoginAttempts = 12 // 12 * 5 = 1 minute

	errPending = eris.New("token status pending")
)

type tokenStruct struct {
	Status string `json:"status"`
	JWT    string `json:"jwt"`
}

// login will open browser to login and save the token to the config file
func login(ctx context.Context) error {
	// Keep the selected org and project to be used after login
	config, _ := getCurrentConfigWithContext(ctx)

	orgID := config.OrganizationID
	projectID := config.ProjectID

	var err error
	if argusid {
		config.Credential, err = loginWithArgusID(ctx)
	} else {
		config.Credential, err = loginWithWorldForge(ctx)
	}
	if err != nil {
		return eris.Wrap(err, "Failed to login")
	}

	// Save credential to config
	err = globalconfig.SaveGlobalConfig(*config)
	if err != nil {
		return eris.Wrap(err, "Failed to save credential")
	}

	if config.CurrRepoKnown { //nolint: nestif // this is not unreasonably complex
		proj, err := getSelectedProject(ctx)
		if err != nil {
			fmt.Println("⚠️ Warning: Failed to get project", projectID, ":", err)
		}
		org, err := getSelectedOrganization(ctx)
		if err != nil {
			fmt.Println("⚠️ Warning: Failed to get organization", orgID, ":", err)
		}
		if proj.Name != "" && org.Name != "" {
			fmt.Printf("📁 Auto-selected project %s (%s) in organization %s (%s)\n",
				proj.Name, proj.Slug,
				org.Name, org.Slug)
		}
	} else {
		// we weren't able to identify the current project automatically, so
		// we have to handle organization selection
		orgID, err = handleOrganizationSelection(ctx, orgID)
		if err != nil {
			orgID = ""
		}
		// save orgID to config
		config.OrganizationID = orgID
		err = globalconfig.SaveGlobalConfig(*config)
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
		err = globalconfig.SaveGlobalConfig(*config)
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
	}
	fmt.Println("\n✨ Login successful! ✨")
	fmt.Println("=====================")
	fmt.Printf("\n👋 Welcome, %s!\n", config.Credential.Name)
	fmt.Printf("🆔 Your ID is: %s\n", config.Credential.ID)
	fmt.Println("\n🚀 You're all set to start using World Forge!")

	return nil
}

// GetToken will get the token from the config file
func getToken(ctx context.Context, url string, argusid bool, result interface{}) error {
	attempts := 1

	for attempts < maxLoginAttempts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second): //nolint:gomnd
			fmt.Printf("\r🔄 Logging in... attempt %d", attempts)

			token, err := makeTokenRequest(ctx, url)
			if err != nil {
				attempts++
				continue
			}

			if err := handleTokenResponse(token, argusid, result); err != nil {
				if errors.Is(err, errPending) {
					attempts++
					continue
				}
				return err
			}

			return nil
		}
	}

	fmt.Println() // Add newline before error
	return eris.New("max attempts reached while waiting for token")
}

func makeTokenRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get token")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, eris.New("non-200 status code")
	}

	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, eris.Wrap(err, "failed to read token")
	}

	return response, nil
}

func handleTokenResponse(response []byte, argusid bool, result interface{}) error {
	if argusid {
		return handleArgusIDToken(response, result)
	}
	return handleWorldForgeToken(response, result)
}

func handleArgusIDToken(response []byte, result interface{}) error {
	err := json.Unmarshal(response, &result)
	if err != nil {
		return eris.Wrap(err, "failed to parse response")
	}

	tokenStruct, ok := result.(*tokenStruct)
	if !ok {
		return eris.New("failed to parse response")
	}

	switch tokenStruct.Status {
	case "pending":
		return errPending
	case "success":
		fmt.Println("\n✨ Login token received successfully!")
		return nil
	default:
		return eris.New(fmt.Sprintf("Status: %s", tokenStruct.Status))
	}
}

func handleWorldForgeToken(response []byte, result interface{}) error {
	token, err := parseResponse[string](response)
	if err != nil {
		return eris.Wrap(err, "failed to parse response")
	}

	if token == nil {
		return eris.New("token is nil")
	}

	strPtr, ok := result.(*string)
	if !ok {
		return eris.New("invalid result type")
	}

	*strPtr = *token
	return nil
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

func parseArgusIDToken(jwtToken string) (globalconfig.Credential, error) {
	var claims struct {
		Name          string    `json:"name"`
		ID            string    `json:"id"`
		Sub           string    `json:"sub"`
		Email         string    `json:"email"`
		EmailVerified bool      `json:"emailVerified"`
		Image         *string   `json:"image"`
		CreatedAt     time.Time `json:"createdAt"`
		UpdatedAt     time.Time `json:"updatedAt"`
		Aud           string    `json:"aud"`
		Iss           string    `json:"iss"`
		Exp           int64     `json:"exp"`
		Iat           int64     `json:"iat"`
	}

	parts := strings.Split(jwtToken, ".")
	if len(parts) != 3 { //nolint:gomnd
		return globalconfig.Credential{}, eris.New("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "failed to decode token payload")
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "failed to parse token claims")
	}

	return globalconfig.Credential{
		Token: jwtToken,
		Name:  claims.Name,
		ID:    claims.Sub,
	}, nil
}

func loginWithWorldForge(ctx context.Context) (globalconfig.Credential, error) {
	key := generateKey()
	url := fmt.Sprintf("%s?key=%s", loginURL, key)

	// Open browser
	err := openBrowser(url)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to open browser")
	}

	// Wait for user to login
	url = fmt.Sprintf("%s?key=%s", getTokenURL, key)
	var token string
	err = getToken(ctx, url, false, &token)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to get token")
	}

	// Parse jwt token to get name from metadata
	cred, err := parseCredential(token)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to get name from token")
	}

	return cred, nil
}

func loginWithArgusID(ctx context.Context) (globalconfig.Credential, error) {
	// Get the link to login
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, argusIDServiceURL, nil)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to create request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to get login link")
	}
	defer resp.Body.Close()

	// Parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to read login link")
	}

	// Parse the response
	var loginLink struct {
		CallBackURL string `json:"callbackUrl"`
		ClientURL   string `json:"clientUrl"`
	}
	err = json.Unmarshal(body, &loginLink)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to parse login link")
	}

	// Open browser
	err = openBrowser(loginLink.ClientURL)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to open browser")
	}

	// Wait for user to login
	var tokenStruct tokenStruct
	err = getToken(ctx, loginLink.CallBackURL, true, &tokenStruct)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to get token")
	}

	// Parse jwt token to get name from metadata
	cred, err := parseArgusIDToken(tokenStruct.JWT)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to get name from token")
	}

	// Need to get User ID from World Forge if using Argus ID
	// This is because the Argus ID is not used as a World Forge user ID
	user, err := getUser(ctx)
	if err != nil {
		return globalconfig.Credential{}, eris.Wrap(err, "Failed to get world forge user")
	}
	cred.ID = user.ID

	return cred, nil
}
