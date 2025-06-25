package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rotisserie/eris"
	"github.com/tidwall/gjson"
	"pkg.world.dev/world-cli/common/printer"
)

const (
	jitterDivisor = 2 // Divisor used to calculate maximum jitter range
)

var _ ClientInterface = (*Client)(nil)

// NewClient creates a new API client with the given base URL.
func NewClient(baseURL, rpcURL string) ClientInterface {
	return &Client{
		BaseURL: baseURL,
		RPCURL:  rpcURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetAuthToken updates the client's authentication credentials.
func (c *Client) SetAuthToken(token string) {
	c.Token = token
}

// TODO: Remove this once we have a proper RPC client
func (c *Client) GetRPCBaseURL() string {
	return c.RPCURL
}

// sendRequest sends an HTTP request with auth token and returns the response body.
//

func (c *Client) sendRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	// Prepare request body and headers
	req, err := c.prepareRequest(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	// Make request with retries
	return c.makeRequestWithRetries(ctx, req)
}

// prepareRequest creates an HTTP request with proper headers and authentication.
func (c *Client) prepareRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, eris.Wrap(err, "Failed to marshal request body")
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Construct full URL
	url := c.BaseURL + endpoint

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to create request")
	}

	// Add authentication if available
	if c.Token != "" {
		req.Header.Add("Authorization", "ArgusID "+c.Token)
	}

	// Set content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// DefaultRequestConfig returns sensible defaults.
func DefaultRequestConfig() RequestConfig {
	return RequestConfig{
		MaxRetries:  5,
		BaseDelay:   100 * time.Millisecond,
		Timeout:     30 * time.Second,
		ContentType: "application/json",
	}
}

// makeRequestWithRetries executes the HTTP request with exponential backoff retry logic.
//
//nolint:gocognit // breaking this up will increase complexity of the package.
func (c *Client) makeRequestWithRetries(ctx context.Context, req *http.Request) ([]byte, error) {
	config := DefaultRequestConfig()
	var lastErr error

	for i := range config.MaxRetries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if i > 0 {
				printer.MoveCursorUp(1)
				printer.ClearToEndOfLine()
				printer.Infoln("Retrying...                                                                  ")
			}

			respBody, err := c.doRequest(req)
			if err == nil {
				return respBody, nil
			}

			// Don't retry if unauthorized
			if strings.Contains(err.Error(), "Unauthorized.") || strings.Contains(err.Error(), "Forbidden.") {
				return nil, err
			}

			// Check if the error is retryable
			if !c.isRetryableError(err) {
				return nil, err
			}

			if i < config.MaxRetries-1 { // Don't print retry message on last attempt
				// Apply exponential backoff with jitter
				delay := c.exponentialBackoffWithJitter(config.BaseDelay, i)
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

	return nil, eris.Wrapf(lastErr, "Failed after %d retries", config.MaxRetries)
}

// doRequest executes a single HTTP request.
func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	resp, err := c.HTTPClient.Do(req)
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

// isRetryableError checks if the error is transient and should be retried.
func (c *Client) isRetryableError(err error) bool {
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
func (c *Client) exponentialBackoffWithJitter(base time.Duration, attempt int) time.Duration {
	backoff := base * (1 << attempt)                                     // Exponential growth
	jitter := time.Duration(rand.Int63n(int64(backoff / jitterDivisor))) //nolint:gosec // it's safe to use rand here
	return backoff + jitter
}
