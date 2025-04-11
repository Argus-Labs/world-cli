package forge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/tidwall/gjson"
	"golang.org/x/term"

	"pkg.world.dev/world-cli/common/globalconfig"
)

const (
	jitterDivisor  time.Duration = 2 // Divisor used to calculate maximum jitter range
	RetryBaseDelay time.Duration = 100 * time.Millisecond
)

var (
	requestTimeout = 5 * time.Second
	httpClient     = &http.Client{
		Timeout: requestTimeout,
	}
)

// this is a variable so we can change it for testing login
var getCurrentConfigWithContext = GetCurrentConfigWithContext

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
	return makeRequestWithRetries(ctx, req)
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
	prefix := "Bearer "
	if argusid {
		prefix = "ArgusID "
	}
	req.Header.Add("Authorization", prefix+cred.Credential.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func makeRequestWithRetries(ctx context.Context, req *http.Request) ([]byte, error) {
	maxRetries := 5
	baseDelay := RetryBaseDelay
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			respBody, err := doRequest(req)
			if err == nil {
				return respBody, nil
			}

			// Don't retry if unauthorized
			if strings.Contains(err.Error(), "Unauthorized. Please login again using 'world login' command") {
				return nil, err
			}

			// Check if the error is retryable
			if !isRetryableError(err) {
				return nil, err
			}

			if i < maxRetries-1 { // Don't print retry message on last attempt
				fmt.Printf("Failed to make request [%s]: %s\n", req.URL, err.Error())
				fmt.Println("Retrying...")

				// Apply exponential backoff with jitter
				delay := exponentialBackoffWithJitter(baseDelay, i)

				// Use timer instead of Sleep to handle cancellation
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}
			}
			lastErr = err
		}
	}

	return nil, eris.Wrapf(lastErr, "Failed after %d retries", maxRetries)
}

// isRetryableError checks if the error is transient and should be retried.
func isRetryableError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	// Check HTTP status codes in error message (fallback)
	errorMsg := err.Error()
	return strings.Contains(errorMsg, "500") ||
		strings.Contains(errorMsg, "502") ||
		strings.Contains(errorMsg, "503") ||
		strings.Contains(errorMsg, "504") ||
		strings.Contains(errorMsg, "429")
}

// exponentialBackoffWithJitter calculates delay with exponential backoff and jitter.
func exponentialBackoffWithJitter(base time.Duration, attempt int) time.Duration {
	backoff := base * (1 << attempt)                                     // Exponential growth
	jitter := time.Duration(rand.Int63n(int64(backoff / jitterDivisor))) //nolint:gosec // it's safe to use rand here
	return backoff + jitter
}

func doRequest(req *http.Request) ([]byte, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, eris.New("Unauthorized. Please login again using 'world login' command")
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
	fmt.Println("\nℹ️  Use 'world login' to authenticate")
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
	cred, err := GetCurrentConfig()
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

func slugCheck(slug string, minLength int, maxLength int) error {
	if len(slug) < minLength || len(slug) > maxLength {
		return eris.Errorf("Slug must be between %d and %d characters", minLength, maxLength)
	}

	// Check if slug contains only allowed characters
	matched, err := regexp.MatchString("^[a-z0-9_]+$", slug)
	if err != nil {
		return eris.Wrap(err, "Error validating slug format")
	}
	if !matched {
		return eris.New("Slug can only contain lowercase letters, numbers, and underscores")
	}

	return nil
}

// NewTeaProgram will create a BubbleTea program that automatically sets the no input option
// if you are not on a TTY, so you can run the debugger. Call it just as you would call tea.NewProgram().
func NewTeaProgram(model tea.Model, opts ...tea.ProgramOption) *tea.Program {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		opts = append(opts, tea.WithInput(nil))
		// opts = append(opts, tea.WithoutRenderer())
	}
	return tea.NewProgram(model, opts...)
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func isValidURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

func replaceLast(x, y, z string) (x2 string) {
	i := strings.LastIndex(x, y)
	if i == -1 {
		return x
	}
	return x[:i] + z + x[i+len(y):]
}

func GetCurrentConfig() (globalconfig.GlobalConfig, error) {
	currConfig, err := globalconfig.GetGlobalConfig()
	// we deliberately ignore any error here and just return it at the end
	// so that we can fill out and much info as we do have
	currConfig.CurrRepoKnown = false
	currConfig.CurrRepoPath, currConfig.CurrRepoURL, _ = FindGitPathAndURL()
	if currConfig.CurrRepoURL != "" {
		for i := range currConfig.KnownProjects {
			knownProject := currConfig.KnownProjects[i]
			if knownProject.RepoURL == currConfig.CurrRepoURL && knownProject.RepoPath == currConfig.CurrRepoPath {
				currConfig.ProjectID = knownProject.ProjectID
				currConfig.OrganizationID = knownProject.OrganizationID
				currConfig.CurrRepoKnown = true
				break
			}
		}
	}
	return currConfig, err
}

func GetCurrentConfigWithContext(ctx context.Context) (*globalconfig.GlobalConfig, error) {
	currConfig, err := GetCurrentConfig()
	// we don't care if we got an error, we will just return it later
	if !currConfig.CurrRepoKnown && //nolint: nestif // not too complex
		currConfig.Credential.Token != "" &&
		currConfig.CurrRepoURL != "" {
		// needed a lookup, and have a token (so we should be logged in)
		// get the organization and project from the project's URL and path
		deployURL := fmt.Sprintf("%s/api/project/?url=%s&path=%s",
			baseURL, url.QueryEscape(currConfig.CurrRepoURL), url.QueryEscape(currConfig.CurrRepoPath))
		body, err := sendRequest(ctx, http.MethodGet, deployURL, nil)
		if err != nil {
			fmt.Println("⚠️ Warning: Failed to lookup World Forge project for Git Repo",
				currConfig.CurrRepoURL, "and path", currConfig.CurrRepoPath, ":", err)
			return &currConfig, err
		}

		// Parse response
		proj, err := parseResponse[project](body)
		if err != nil && err.Error() != "Missing data field in response" {
			// missing data field in response just means nothing was found
			fmt.Println("⚠️ Warning: Failed to parse project lookup response: ", err)
			return &currConfig, err
		}
		if proj != nil {
			// add to list of known projects
			currConfig.KnownProjects = append(currConfig.KnownProjects, globalconfig.KnownProject{
				ProjectID:      proj.ID,
				OrganizationID: proj.OrgID,
				RepoURL:        proj.RepoURL,
				RepoPath:       proj.RepoPath,
			})
			// save the config, but don't change the default ProjectID & OrgID
			err := globalconfig.SaveGlobalConfig(currConfig)
			if err != nil {
				fmt.Println("⚠️ Warning: Failed to save config: ", err)
				// continue on, this is not fatal
			}
			// now return a copy of it with the looked up ProjectID and OrganizationID set
			currConfig.ProjectID = proj.ID
			currConfig.OrganizationID = proj.OrgID
			currConfig.CurrRepoKnown = true
		}
	}
	return &currConfig, err
}

func FindGitPathAndURL() (string, string, error) {
	urlData, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return "", "", err
	}
	url := strings.TrimSpace(string(urlData))
	url = replaceLast(url, ".git", "")
	workingDir, err := os.Getwd()
	if err != nil {
		return "", url, err
	}
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", url, err
	}
	rootPath := strings.TrimSpace(string(root))
	path := strings.Replace(workingDir, rootPath, "", 1)
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return path, url, nil
}
