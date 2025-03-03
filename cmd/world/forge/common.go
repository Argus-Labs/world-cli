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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/tidwall/gjson"

	"pkg.world.dev/world-cli/common/globalconfig"
)

var (
	requestTimeout = 5 * time.Second
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

// sendRequest sends an HTTP request with auth token and returns the response body
func sendRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	// Prepare request body and headers
	req, err := prepareRequest(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// Make request with retries
	return makeRequestWithRetries(req)
}

func prepareRequest(ctx context.Context, method, url string, body interface{}) (*http.Request, error) {
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

	// Add headers
	req.Header.Add("Authorization", "Bearer "+cred.Credential.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func makeRequestWithRetries(req *http.Request) ([]byte, error) {
	maxRetries := 5
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if lastErr != nil {
			fmt.Printf("Failed to make request [%s]: %s\n", req.URL, lastErr.Error())
			fmt.Println("Retrying...")
			time.Sleep(1 * time.Second)
		}

		respBody, err := doRequest(req)
		if err == nil {
			return respBody, nil
		}
		lastErr = err
	}

	return nil, eris.Wrapf(lastErr, "Failed after %d retries", maxRetries)
}

func doRequest(req *http.Request) ([]byte, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, eris.New("Unauthorized. Please login again using 'world forge login' command")
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		message := gjson.GetBytes(body, "message").String()
		return nil, eris.New(message)
	}

	return io.ReadAll(resp.Body)
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

func printNoOrganizations() {
	fmt.Println("\n🏢 No Organizations Found")
	fmt.Println("=========================")
	fmt.Println("\n❌ You don't have any organizations yet.")
	fmt.Println("\nℹ️  Use 'world forge organization create' to create one.")
}

func printNoSelectedOrganization() {
	fmt.Println("\n🏢 No Organization Selected")
	fmt.Println("==========================")
	fmt.Println("\n❌ You don't have any organization selected.")
	fmt.Println("\nℹ️  Use 'world forge organization switch' to select one")
}

func printNoSelectedProject() {
	fmt.Println("\n📁 No Project Selected")
	fmt.Println("=====================")
	fmt.Println("\n❌ You don't have any project selected.")
	fmt.Println("\nℹ️  Use 'world forge project switch' to select one")
}

func printNoProjectsInOrganization() {
	fmt.Println("\n📦 No Projects Found")
	fmt.Println("====================")
	fmt.Println("\n❌ You don't have any projects in this organization yet.")
	fmt.Println("\nℹ️  Use 'world forge project create' to create your first project!")
}

func printAuthenticationRequired() {
	fmt.Println("\n🔒 Authentication Required")
	fmt.Println("========================")
	fmt.Println("\n❌ You are not currently logged in")
	fmt.Println("\nℹ️  Use 'world forge login' to authenticate")
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
		printAuthenticationRequired()
		return false
	}

	if cred.Credential.Token == "" {
		printAuthenticationRequired()
		return false
	}

	return true
}
