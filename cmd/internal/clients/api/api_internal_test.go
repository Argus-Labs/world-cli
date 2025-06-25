package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper function to create a response with given status and body.
func createResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)), // Add the full status
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// Test data structures for parsing tests.
type TestUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestNewClient(t *testing.T) {
	t.Parallel()
	baseURL := "https://api.example.com"
	client := NewClient(baseURL)

	assert.NotNil(t, client)

	// Type assertion to access internal fields
	apiClient := client.(*Client)
	assert.Equal(t, baseURL, apiClient.BaseURL)
	assert.NotNil(t, apiClient.HTTPClient)
	assert.Empty(t, apiClient.Token)
}

func TestSetAuthToken(t *testing.T) {
	t.Parallel()
	client := &Client{}
	token := "test-token-123"

	client.SetAuthToken(token)

	assert.Equal(t, token, client.Token)
}

func TestDefaultRequestConfig(t *testing.T) {
	t.Parallel()
	config := DefaultRequestConfig()

	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.BaseDelay)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "application/json", config.ContentType)
}

func TestParseResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		body        []byte
		expectError bool
		expected    TestUser
	}{
		{
			name:        "valid response",
			body:        []byte(`{"data": {"id": "123", "name": "John"}}`),
			expectError: false,
			expected:    TestUser{ID: "123", Name: "John"},
		},
		{
			name:        "missing data field",
			body:        []byte(`{"error": "something"}`),
			expectError: true,
		},
		{
			name:        "invalid json in data",
			body:        []byte(`{"data": "invalid"}`),
			expectError: true,
		},
		{
			name:        "empty response",
			body:        []byte(``),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseResponse[TestUser](tt.body)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseResponseSlice(t *testing.T) {
	t.Parallel()
	body := []byte(`{"data": [{"id": "1", "name": "John"}, {"id": "2", "name": "Jane"}]}`)

	result, err := parseResponse[[]TestUser](body)

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, "1", result[0].ID)
	require.Equal(t, "John", result[0].Name)
	require.Equal(t, "2", result[1].ID)
	require.Equal(t, "Jane", result[1].Name)
}

func TestPrepareRequest(t *testing.T) {
	t.Parallel()
	client := &Client{
		BaseURL: "https://api.example.com",
		Token:   "test-token",
	}

	tests := []struct {
		name       string
		method     string
		endpoint   string
		body       interface{}
		expectAuth bool
		expectBody bool
		expectJSON bool
	}{
		{
			name:       "GET request with auth",
			method:     "GET",
			endpoint:   "/api/user",
			body:       nil,
			expectAuth: true,
			expectBody: false,
			expectJSON: false,
		},
		{
			name:       "POST request with body",
			method:     "POST",
			endpoint:   "/api/user",
			body:       map[string]string{"name": "test"},
			expectAuth: true,
			expectBody: true,
			expectJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			req, err := client.prepareRequest(ctx, tt.method, tt.endpoint, tt.body)

			require.NoError(t, err)
			require.Equal(t, tt.method, req.Method)
			require.Equal(t, client.BaseURL+tt.endpoint, req.URL.String())

			if tt.expectAuth {
				require.Equal(t, "ArgusID "+client.Token, req.Header.Get("Authorization"))
			} else {
				require.Empty(t, req.Header.Get("Authorization"))
			}

			if tt.expectJSON {
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
			}

			if tt.expectBody {
				require.NotNil(t, req.Body)
			} else {
				require.Nil(t, req.Body)
			}
		})
	}
}

func TestPrepareRequestWithoutToken(t *testing.T) {
	t.Parallel()
	client := &Client{
		BaseURL: "https://api.example.com",
		// No token set
	}

	ctx := t.Context()
	req, err := client.prepareRequest(ctx, "GET", "/api/user", nil)

	require.NoError(t, err)
	require.Empty(t, req.Header.Get("Authorization"))
}

func TestPrepareRequestMarshalError(t *testing.T) {
	t.Parallel()
	client := &Client{BaseURL: "https://api.example.com"}

	// Use a channel which can't be marshaled to JSON
	invalidBody := make(chan int)

	ctx := t.Context()
	_, err := client.prepareRequest(ctx, "POST", "/api/test", invalidBody)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to marshal request body")
}

func TestDoRequestSuccess(t *testing.T) {
	t.Parallel()
	mockClient := &MockHTTPClient{}
	client := &Client{
		BaseURL:    "https://api.example.com",
		HTTPClient: mockClient,
	}

	expectedBody := `{"data": {"id": "123"}}`
	response := createResponse(http.StatusOK, expectedBody)

	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(response, nil)

	ctx := t.Context()
	req, _ := client.prepareRequest(ctx, "GET", "/api/user", nil)

	body, err := client.doRequest(req)

	require.NoError(t, err)
	require.Equal(t, expectedBody, string(body))
	mockClient.AssertExpectations(t)
}

func TestDoRequestHTTPErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  "",
			expectedError: "401 Unauthorized.",
		},
		{
			name:          "forbidden",
			statusCode:    http.StatusForbidden,
			responseBody:  "",
			expectedError: "403 Forbidden.",
		},
		{
			name:          "server error with message",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"message": "Internal server error"}`,
			expectedError: "Internal server error",
		},
		{
			name:          "server error without message",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error": "bad request"}`,
			expectedError: "400 Bad Request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockClient := &MockHTTPClient{}
			client := &Client{
				BaseURL:    "https://api.example.com",
				HTTPClient: mockClient,
			}

			response := createResponse(tt.statusCode, tt.responseBody)
			mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(response, nil)

			ctx := t.Context()
			req, _ := client.prepareRequest(ctx, "GET", "/api/user", nil)

			_, err := client.doRequest(req)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestDoRequestNetworkError(t *testing.T) {
	t.Parallel()
	mockClient := &MockHTTPClient{}
	client := &Client{
		BaseURL:    "https://api.example.com",
		HTTPClient: mockClient,
	}

	networkErr := errors.New("network connection failed")
	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return((*http.Response)(nil), networkErr)

	ctx := t.Context()
	req, _ := client.prepareRequest(ctx, "GET", "/api/user", nil)

	_, err := client.doRequest(req)

	require.Error(t, err)
	require.Equal(t, networkErr, err)
	mockClient.AssertExpectations(t)
}

func TestIsRetryableError(t *testing.T) {
	t.Parallel()
	client := &Client{}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "timeout error",
			err:      &timeoutError{},
			expected: true,
		},
		{
			name:     "500 server error",
			err:      errors.New("500 Internal Server Error"),
			expected: true,
		},
		{
			name:     "502 bad gateway",
			err:      errors.New("502 Bad Gateway"),
			expected: true,
		},
		{
			name:     "503 service unavailable",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "504 gateway timeout",
			err:      errors.New("504 Gateway Timeout"),
			expected: true,
		},
		{
			name:     "429 too many requests",
			err:      errors.New("429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "400 bad request",
			err:      errors.New("400 Bad Request"),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := client.isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper type that implements net.Error for testing timeout scenarios.
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestExponentialBackoffWithJitter(t *testing.T) {
	t.Parallel()
	client := &Client{}
	base := 100 * time.Millisecond

	// Test that backoff increases exponentially
	delay0 := client.exponentialBackoffWithJitter(base, 0)
	delay1 := client.exponentialBackoffWithJitter(base, 1)
	delay2 := client.exponentialBackoffWithJitter(base, 2)

	// Verify exponential growth (accounting for jitter)
	assert.Greater(t, delay1, delay0)
	assert.Greater(t, delay2, delay1)

	// Verify base delay is respected (with jitter, should be at least base)
	assert.GreaterOrEqual(t, delay0, base)

	// Verify jitter doesn't make delay too large
	maxDelay0 := base + (base / jitterDivisor)
	maxDelay1 := (base * 2) + ((base * 2) / jitterDivisor)

	assert.LessOrEqual(t, delay0, maxDelay0)
	assert.LessOrEqual(t, delay1, maxDelay1)
}

func TestSendRequestSuccess(t *testing.T) {
	t.Parallel()
	mockClient := &MockHTTPClient{}
	client := &Client{
		BaseURL:    "https://api.example.com",
		Token:      "test-token",
		HTTPClient: mockClient,
	}

	expectedBody := `{"data": {"id": "123"}}`
	response := createResponse(http.StatusOK, expectedBody)

	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(response, nil)

	ctx := t.Context()
	body, err := client.sendRequest(ctx, "GET", "/api/user", nil)

	require.NoError(t, err)
	require.Equal(t, expectedBody, string(body))
	mockClient.AssertExpectations(t)
}

func TestSendRequestWithRetries(t *testing.T) {
	t.Parallel()
	mockClient := &MockHTTPClient{}
	client := &Client{
		BaseURL:    "https://api.example.com",
		HTTPClient: mockClient,
	}

	// First call fails with retryable error, second succeeds
	expectedBody := `{"data": {"id": "123"}}`

	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
		createResponse(http.StatusInternalServerError, ""), nil).Once()
	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
		createResponse(http.StatusOK, expectedBody), nil).Once()

	ctx := t.Context()
	body, err := client.sendRequest(ctx, "GET", "/api/user", nil)

	require.NoError(t, err)
	require.Equal(t, expectedBody, string(body))
	mockClient.AssertExpectations(t)
}

func TestSendRequestNonRetryableError(t *testing.T) {
	t.Parallel()
	mockClient := &MockHTTPClient{}
	client := &Client{
		BaseURL:    "https://api.example.com",
		HTTPClient: mockClient,
	}

	// Unauthorized error should not be retried
	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
		createResponse(http.StatusUnauthorized, ""), nil).Once()

	ctx := t.Context()
	_, err := client.sendRequest(ctx, "GET", "/api/user", nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "401 Unauthorized")
	mockClient.AssertExpectations(t)
}

func TestSendRequestContextCancellation(t *testing.T) {
	t.Parallel()
	mockClient := &MockHTTPClient{}
	client := &Client{
		BaseURL:    "https://api.example.com",
		HTTPClient: mockClient,
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := client.sendRequest(ctx, "GET", "/api/user", nil)

	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestSendRequestMaxRetriesExceeded(t *testing.T) {
	t.Parallel()
	mockClient := &MockHTTPClient{}
	client := &Client{
		BaseURL:    "https://api.example.com",
		HTTPClient: mockClient,
	}

	// Always return retryable error (500 status code)
	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
		createResponse(http.StatusInternalServerError, "server error"), nil)

	ctx := t.Context()
	_, err := client.sendRequest(ctx, "GET", "/api/user", nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed after 5 retries")

	// Should have been called MaxRetries (5) times
	mockClient.AssertNumberOfCalls(t, "Do", 5)
}

func TestDebugServerError(t *testing.T) {
	t.Parallel()
	mockClient := &MockHTTPClient{}
	client := &Client{
		BaseURL:    "https://api.example.com",
		HTTPClient: mockClient,
	}

	// Return 500 error
	mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
		createResponse(http.StatusInternalServerError, "server error"), nil).Once()

	ctx := t.Context()
	req, _ := client.prepareRequest(ctx, "GET", "/api/user", nil)

	_, err := client.doRequest(req)

	require.Error(t, err)
	t.Logf("Error message: %s", err.Error())

	// Test if this error is considered retryable
	isRetryable := client.isRetryableError(err)
	t.Logf("Is retryable: %v", isRetryable)
	require.True(t, isRetryable, "500 error should be retryable")
}
