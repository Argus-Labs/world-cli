package forge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/globalconfig"
)

// sendRequest sends an HTTP request with auth token and returns the response body
func sendRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, eris.Wrap(err, "Failed to marshal request body")
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Get credential from config
	cred, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get credential")
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to create request")
	}

	// Add authorization header
	req.Header.Add("Authorization", "Bearer "+cred.Credential.Token)

	// Add content-type header for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to make request")
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, eris.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to read response body")
	}

	return respBody, nil
}

func parseResponse[T any](body []byte) (*T, error) {
	// Parse wrapper response
	var response struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	return &response.Data, nil
}

func printNoSelectedOrganization() {
	fmt.Println("You don't have any organization selected.")
	fmt.Println("Use 'world forge organization switch' to select one.")
	fmt.Println()
}

func printNoSelectedProject() {
	fmt.Println("You don't have any project selected.")
	fmt.Println("Use 'world forge project switch' to select one.")
	fmt.Println()
}

func printNoProjectsInOrganization() {
	fmt.Println("You don't have any projects in this organization yet.")
	fmt.Println("Use 'world forge project create' to create one.")
	fmt.Println()
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}
