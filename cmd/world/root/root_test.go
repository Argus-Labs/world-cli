package root

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	"pkg.world.dev/world-cli/common/login"
)

// outputFromCmd runs the rootCmd with the given cmd arguments and returns the output of the command along with
// any errors.
func outputFromCmd(cobra *cobra.Command, cmd string) ([]string, error) {
	stdOut := &bytes.Buffer{}
	stdErr := &bytes.Buffer{}
	cobra.SetOut(stdOut)
	defer func() {
		cobra.SetOut(nil)
	}()
	cobra.SetErr(stdErr)
	defer func() {
		cobra.SetErr(nil)
	}()
	cobra.SetArgs(strings.Split(cmd, " "))
	defer func() {
		cobra.SetArgs(nil)
	}()

	if err := cobra.Execute(); err != nil {
		return nil, fmt.Errorf("root command failed with: %w", err)
	}
	lines := strings.Split(stdOut.String(), "\n")
	errorStr := stdErr.String()
	if len(errorStr) > 0 {
		return lines, errors.New(errorStr)
	}

	return lines, nil
}

func TestSubcommandsHaveHelpText(t *testing.T) {
	lines, err := outputFromCmd(rootCmd, "help")
	assert.NilError(t, err)
	seenSubcommands := map[string]int{
		"cardinal":   0,
		"completion": 0,
		"doctor":     0,
		"help":       0,
		"version":    0,
	}

	for _, line := range lines {
		for subcommand := range seenSubcommands {
			if strings.HasPrefix(line, "  "+subcommand) {
				seenSubcommands[subcommand]++
			}
		}
	}

	for subcommand, count := range seenSubcommands {
		assert.Check(t, count > 0, "subcommand %q is not listed in the help command", subcommand)
	}
}

func TestExecuteDoctorCommand(t *testing.T) {
	teaOut := &bytes.Buffer{}
	_, err := outputFromCmd(getDoctorCmd(teaOut), "")
	assert.NilError(t, err)

	seenDependencies := map[string]int{
		"Git":                      0,
		"Go":                       0,
		"Docker":                   0,
		"Docker Compose":           0,
		"Docker daemon is running": 0,
	}

	lines := strings.Split(teaOut.String(), "\r\n")
	for _, line := range lines {
		// Remove the first three characters for the example(✓  Git)
		resultString := ""
		if len(line) > 5 {
			resultString = line[5:]
		}
		for dep := range seenDependencies {
			if resultString != "" && resultString == dep {
				seenDependencies[dep]++
			}
		}
	}

	for dep, count := range seenDependencies {
		assert.Check(t, count > 0, "dependencies %q is not listed in dependencies checking", dep)
	}
}

func TestCreateStartStopRestartPurge(t *testing.T) {
	// Create Cardinal
	gameDir, err := os.MkdirTemp("", "game-template-start")
	assert.NilError(t, err)

	// Remove dir
	defer func() {
		err = os.RemoveAll(gameDir)
		assert.NilError(t, err)
	}()

	// Change dir
	err = os.Chdir(gameDir)
	assert.NilError(t, err)

	// set tea ouput to variable
	teaOut := &bytes.Buffer{}
	createCmd := getCreateCmd(teaOut)
	createCmd.SetArgs([]string{gameDir})

	err = createCmd.Execute()
	assert.NilError(t, err)

	// Start cardinal
	rootCmd.SetArgs([]string{"cardinal", "start", "--build", "--detach", "--editor=false"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	defer func() {
		// Purge cardinal
		rootCmd.SetArgs([]string{"cardinal", "purge"})
		err = rootCmd.Execute()
		assert.NilError(t, err)
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Restart cardinal
	rootCmd.SetArgs([]string{"cardinal", "restart", "--detach"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Stop cardinal
	rootCmd.SetArgs([]string{"cardinal", "stop"})
	err = rootCmd.Execute()
	assert.NilError(t, err)

	// Check and wait until cardinal shutdowns
	assert.Assert(t, cardinalIsDown(t), "Cardinal is not successfully shutdown")
}

func TestDev(t *testing.T) {
	// Create Cardinal
	gameDir, err := os.MkdirTemp("", "game-template-dev")
	assert.NilError(t, err)

	// Remove dir
	defer func() {
		err = os.RemoveAll(gameDir)
		assert.NilError(t, err)
	}()

	// Change dir
	err = os.Chdir(gameDir)
	assert.NilError(t, err)

	// set tea ouput to variable
	teaOut := &bytes.Buffer{}
	createCmd := getCreateCmd(teaOut)
	createCmd.SetArgs([]string{gameDir})

	err = createCmd.Execute()
	assert.NilError(t, err)

	// Start cardinal dev
	ctx, cancel := context.WithCancel(context.Background())
	rootCmd.SetArgs([]string{"cardinal", "dev", "--editor=false"})
	go func() {
		err := rootCmd.ExecuteContext(ctx)
		assert.NilError(t, err)
	}()

	// Check and wait until cardinal is healthy
	assert.Assert(t, cardinalIsUp(t), "Cardinal is not running")

	// Shutdown the program
	cancel()

	// Check and wait until cardinal shutdowns
	assert.Assert(t, cardinalIsDown(t), "Cardinal is not successfully shutdown")
}

func TestCheckLatestVersion(t *testing.T) {
	t.Run("success scenario", func(t *testing.T) {
		AppVersion = "v1.0.0"
		err := checkLatestVersion()
		assert.NilError(t, err)
	})

	t.Run("error version format", func(t *testing.T) {
		AppVersion = "wrong format"
		err := checkLatestVersion()
		assert.Error(t, err, "error parsing current version: Malformed version: wrong format")
	})
}

func cardinalIsUp(t *testing.T) bool {
	up := false
	for i := 0; i < 10; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:4040", time.Second)
		if err != nil {
			time.Sleep(time.Second)
			t.Log("Failed to connect to Cardinal, retrying...")
			continue
		}
		_ = conn.Close()
		up = true
		break
	}
	return up
}

func cardinalIsDown(t *testing.T) bool {
	down := false
	for i := 0; i < 10; i++ {
		conn, err := net.DialTimeout("tcp", "localhost:4040", time.Second)
		if err != nil {
			down = true
			break
		}
		_ = conn.Close()
		time.Sleep(time.Second)
		t.Log("Cardinal is still running, retrying...")
		continue
	}
	return down
}

func TestGenerateTokenNameWithFallback(t *testing.T) {
	// Attempt to generate a token name
	name := generateTokenNameWithFallback()

	// Ensure the name follows the expected pattern
	tassert.Contains(t, name, "cli_")

	// Additional checks if user and hostname can be retrieved in the environment
	currentUser, userErr := user.Current()
	hostname, hostErr := os.Hostname()
	if userErr == nil && hostErr == nil {
		expectedPrefix := fmt.Sprintf("cli_%s@%s_", currentUser.Username, hostname)
		tassert.Contains(t, name, expectedPrefix)
	}
}

func TestPollForAccessToken(t *testing.T) {
	tests := []struct {
		name             string
		statusCode       int
		retryAfterHeader string
		responseBody     string
		expectError      bool
		expectedResponse login.AccessTokenResponse
	}{
		{
			name:         "Successful token retrieval",
			statusCode:   http.StatusOK,
			responseBody: `{"access_token": "test_token", "pub_key": "test_pub_key", "nonce": "test_nonce"}`,
			expectedResponse: login.AccessTokenResponse{
				AccessToken: "test_token",
				PublicKey:   "test_pub_key",
				Nonce:       "test_nonce",
			},
			expectError: false,
		},
		{
			name:             "Retry on 404 with Retry-After header",
			statusCode:       http.StatusNotFound,
			retryAfterHeader: "1",
			expectError:      true,
		},
		{
			name:             "Retry on 404 without Retry-After header",
			statusCode:       http.StatusNotFound,
			retryAfterHeader: "",
			expectError:      true,
		},
		{
			name:         "Error on invalid JSON response",
			statusCode:   http.StatusOK,
			responseBody: `invalid_json`,
			expectError:  true,
		},
		{
			name:        "Error on non-200/404 status",
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.retryAfterHeader != "" {
					w.Header().Set("Retry-After", test.retryAfterHeader)
				}
				w.WriteHeader(test.statusCode)
				w.Write([]byte(test.responseBody)) //nolint:errcheck // Ignore error for test
			})

			server := httptest.NewServer(handler)
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			response, err := pollForAccessToken(ctx, server.URL)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tassert.Equal(t, test.expectedResponse, response)
			}
		})
	}
}
