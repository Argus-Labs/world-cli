package forge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/tidwall/gjson"

	"pkg.world.dev/world-cli/common/globalconfig"
)

var (
	requestTimeout = 2 * time.Second
	httpClient     = &http.Client{
		Timeout: requestTimeout,
	}
)

var generateKey = func() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// Change from function to variable
var openBrowser = func(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		fmt.Printf("Could not automatically open browser. Please visit this URL:\n%s\n", url)
	}
	if err != nil {
		fmt.Printf("Failed to open browser automatically. Please visit this URL:\n%s\n", url)
	}
	return nil
}

var getInput = func() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", eris.Wrap(err, "Failed to read input")
	}
	return strings.TrimSpace(input), nil
}

var getInputInt = func() (int, error) {
	input, err := getInput()
	if err != nil {
		return 0, eris.Wrap(err, "Failed to read input")
	}
	return strconv.Atoi(input)
}

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

	// Make request with timeout
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to make request")
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		// if 401 show message to login again
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, eris.New("Unauthorized. Please login again using 'world forge login' command")
		}

		// parse response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, eris.Wrap(err, "Failed to read response body")
		}
		// get message from response body
		message := gjson.GetBytes(body, "message").String()
		return nil, eris.New(message)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to read response body")
	}

	return respBody, nil
}

func parseResponse[T any](body []byte) (*T, error) {
	result := gjson.GetBytes(body, "data")
	if !result.Exists() {
		return nil, eris.New("Missing data field in response")
	}

	var data T
	if err := json.Unmarshal([]byte(result.Raw), &data); err != nil {
		return nil, eris.Wrap(err, "Failed to parse response")
	}

	return &data, nil
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

func checkLogin() bool {
	cred, err := globalconfig.GetGlobalConfig()
	if err != nil {
		fmt.Println("You are not logged in. Please login first")
		fmt.Println("Use 'world forge login' to login")
		return false
	}

	if cred.Credential.Token == "" {
		fmt.Println("You are not logged in. Please login first")
		fmt.Println("Use 'world forge login' to login")
		return false
	}

	return true
}
