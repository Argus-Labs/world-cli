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
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/tidwall/gjson"
	"golang.org/x/term"
	"pkg.world.dev/world-cli/common/printer"
)

const (
	jitterDivisor  time.Duration = 2 // Divisor used to calculate maximum jitter range
	RetryBaseDelay time.Duration = 100 * time.Millisecond
	requestTimeout time.Duration = 5 * time.Second
)

var (
	httpClient = &http.Client{
		Timeout: requestTimeout,
	}
	// Pre-compiled regex for merging multiple underscores.
	underscoreRegex = regexp.MustCompile(`_+`)
)

var generateKey = func() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

// Change from function to variable.
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
		printer.Infof("Could not automatically open browser. Please visit this URL:\n%s\n", url)
	}
	if err != nil {
		printer.Infof("Failed to open browser automatically. Please visit this URL:\n%s\n", url)
	}
	return nil
}

var getInput = func(prompt, defaultStr string) string {
	if prompt != "" {
		printer.Info(prompt)
	}
	if defaultStr != "" {
		printer.Infof(" [%s]: ", defaultStr)
	} else {
		printer.Info(": ")
	}
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n') // only returns error if input doesn't end in delimiter
	input = strings.TrimSpace(input)
	if input == "" && defaultStr != "" {
		// display the default value as if they typed it in
		printer.MoveCursorUp(1)
		printer.MoveCursorRight(len(defaultStr) + 4 + len(prompt))
		printer.Infoln(defaultStr)
		return defaultStr
	}
	return input
}

// sendRequest sends an HTTP request with auth token and returns the response body.
func sendRequest(ctx context.Context, method, url string, body any) ([]byte, error) {
	// Prepare request body and headers
	req, err := prepareRequest(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// Make request with retries
	return makeRequestWithRetries(ctx, req)
}

func prepareRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, eris.Wrap(err, "Failed to marshal request body")
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Get credential from config
	config, err := GetForgeConfig()
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get credential")
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to create request")
	}

	// Add headers
	prefix := "ArgusID "
	req.Header.Add("Authorization", prefix+config.Credential.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func makeRequestWithRetries(ctx context.Context, req *http.Request) ([]byte, error) { //nolint:gocognit
	maxRetries := 5
	baseDelay := RetryBaseDelay
	var lastErr error

	for i := range maxRetries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if i > 0 {
				printer.MoveCursorUp(1)
				printer.ClearToEndOfLine()
				printer.Infoln("Retrying...                                                                  ")
			}
			respBody, err := doRequest(req)
			if err == nil {
				return respBody, nil
			}

			// Don't retry if unauthorized
			if strings.Contains(err.Error(), "Unauthorized.") || strings.Contains(err.Error(), "Forbidden.") {
				return nil, err
			}

			// Check if the error is retryable
			if !isRetryableError(err) {
				return nil, err
			}

			if i < maxRetries-1 { // Don't print retry message on last attempt
				// Apply exponential backoff with jitter
				delay := exponentialBackoffWithJitter(baseDelay, i)
				prompt := fmt.Sprintf("Failed to make request [%s]: %s. Will retry...", req.URL, err.Error())
				printer.MoveCursorUp(1) // move cursor up to overwrite the previous "Retrying" line
				printer.Errorln(prompt)

				// Use timer instead of Sleep to handle cancellation
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}
			} else {
				printer.MoveCursorUp(1)
				printer.ClearToEndOfLine()
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
			return nil, eris.New("401 Unauthorized.")
		}
		if resp.StatusCode == http.StatusForbidden {
			return nil, eris.New("403 Forbidden.")
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		message := gjson.GetBytes(body, "message").String()
		if message == "" {
			return nil, eris.New(resp.Status)
		}
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
	printer.NewLine(1)
	printer.Headerln("   No Organizations Found   ")
	printer.Info("1. Use ")
	printer.Notification("'world forge organization create'")
	printer.Infoln(" to create an organization.")
	printer.Info("2. Have a member send invite using ")
	printer.Notification("'world forge organization invite'")
	printer.Infoln(".")
}

func printNoSelectedOrganization() {
	printer.NewLine(1)
	printer.Headerln("   No Organization Selected   ")
	printer.Infoln("You don't have any organization selected.")
	printer.Info("Use ")
	printer.Notification("'world forge organization switch'")
	printer.Infoln(" to select one!")
}

func printNoSelectedProject() {
	printer.NewLine(1)
	printer.Headerln("   No Project Selected   ")
	printer.Infoln("You don't have any project selected.")
	printer.Info("Use ")
	printer.Notification("'world forge project switch'")
	printer.Infoln(" to select one!")
}

func printNoProjectsInOrganization() {
	printer.NewLine(1)
	printer.Headerln("   No Projects Found   ")
	printer.Infoln("You don't have any projects in this organization yet.")
	printer.Info("Use ")
	printer.Notification("'world forge project create'")
	printer.Infoln(" to start your first project!")
}

// slugToSaneCheck checks that slug is valid, and returns a sanitized version.
func slugToSaneCheck(slug string, minLength int, maxLength int) (string, error) {
	if len(slug) < minLength {
		return slug, eris.Errorf("Slug must be at least %d characters", minLength)
	}

	// Check if slug contains only allowed characters.
	matched, err := regexp.MatchString("^[a-z0-9_]+$", slug)
	if err != nil {
		return slug, eris.Wrap(err, "Error validating slug format")
	}
	if !matched {
		return slug, eris.New("Slug can only contain lowercase letters, numbers, and underscores")
	}

	// Process the slug, and ensure it's in sane format.
	returnSlug := strings.ToLower(strings.TrimSpace(slug))
	returnSlug = strings.ReplaceAll(returnSlug, " ", "_")
	returnSlug = underscoreRegex.ReplaceAllString(returnSlug, "_")
	returnSlug = strings.Trim(returnSlug, "_")

	if len(returnSlug) > maxLength {
		return returnSlug[:maxLength], nil
	}

	return returnSlug, nil
}

func CreateSlugFromName(name string, minLength int, maxLength int) string {
	shorten := len(name) > maxLength

	var slug string
	wroteUnderscore := false
	hadCapital := false
	for i, r := range name {
		switch {
		case unicode.IsLower(r) || unicode.IsNumber(r):
			// copy lowercase letters and numbers
			slug += string(r)
			wroteUnderscore = false
			hadCapital = unicode.IsNumber(r) // treat numbers as capital letters
		case unicode.IsUpper(r):
			// convert capital letter to lower, with _ if dealing with CamelCase ( -> camel_case )
			if !shorten && i != 0 && !wroteUnderscore && !hadCapital {
				slug += "_"
			}
			slug += string(unicode.ToLower(r))
			wroteUnderscore = false
			hadCapital = true
		case (r == '_' || !shorten) && !wroteUnderscore:
			// underscore is preserved (but many fused into one)
			// unless the input was too long, other characters are converted to underscores (but many fused into one)
			slug += "_"
			wroteUnderscore = true
			hadCapital = false
		}
	}
	slug = strings.Trim(slug, "_")
	if len(slug) < minLength {
		slug += "_" + uuid.NewString()[:8] // add the first 8 characters of the UUID
		slug = strings.TrimLeft(slug, "_")
	}
	if len(slug) > maxLength {
		slug = slug[:maxLength]
		slug = strings.TrimRight(slug, "_")
	}
	return slug
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

func replaceLast(x, y, z string) string {
	i := strings.LastIndex(x, y)
	if i == -1 {
		return x
	}
	return x[:i] + z + x[i+len(y):]
}
