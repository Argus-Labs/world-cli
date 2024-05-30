package root

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/globalconfig"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/login"
	"pkg.world.dev/world-cli/tea/component/program"
	"pkg.world.dev/world-cli/tea/style"
)

var (
	// token is the credential used to authenticate with the World Forge Service
	token string

	// world forge base URL
	worldForgeBaseURL = "http://localhost:3000"

	defaultRetryAfterSeconds = 3
)

// loginCmd logs into the World Forge Service
func getLoginCmd() *cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate using an access token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger.SetDebugMode(cmd)

			err := loginOnBrowser(cmd.Context())
			if err != nil {
				return eris.Wrap(err, "failed to login")
			}

			return nil
		},
	}

	return loginCmd
}

func loginOnBrowser(ctx context.Context) error {
	encryption, err := login.NewEncryption()
	if err != nil {
		logger.Error("Failed to create login encryption", err)
		return err
	}

	encodedPubKey := encryption.EncodedPublicKey()
	sessionID := uuid.NewString()
	tokenName := generateTokenNameWithFallback()

	loginURL := fmt.Sprintf("%s/cli/login?session_id=%s&token=%s&pub_key=%s",
		worldForgeBaseURL, sessionID, tokenName, encodedPubKey)

	loginMessage := "In case the browser didn't open, please open the following link in your browser"
	fmt.Print(style.CLIHeader("World Forge", style.DoubleRightIcon.Render(loginMessage)), "\n")
	fmt.Printf("%s\n\n", loginURL)
	if err := login.RunOpenCmd(ctx, loginURL); err != nil {
		logger.Error("Failed to open browser", err)
		return err
	}

	// Wait for the token to be generated
	if err := program.RunProgram(ctx, func(p program.Program, ctx context.Context) error {
		p.Send(program.StatusMsg("Waiting response from world forge service..."))

		pollURL := fmt.Sprintf("%s/auth/cli/login/%s", worldForgeBaseURL, sessionID)
		accessToken, err := pollForAccessToken(ctx, pollURL)

		if err != nil {
			return err
		}

		token, err = encryption.DecryptAccessToken(accessToken.AccessToken, accessToken.PublicKey, accessToken.Nonce)
		if err != nil {
			return err
		}

		if err := globalconfig.SetWorldForgeToken(tokenName, token); err != nil {
			logger.Error("Failed to set access token", err)
			return err
		}

		return nil
	}); err != nil {
		logger.Error("Failed to get access token", err)
		return err
	}

	fmt.Println(style.TickIcon.Render("Successfully logged in :"))
	// Print the token
	credential, err := globalconfig.GetWorldForgeCredential()
	if err != nil {
		logger.Warn("Failed to get the access token when print", err)
	}
	stringCredential, err := json.MarshalIndent(credential, "", "  ")
	if err != nil {
		logger.Warn("Failed to marshal the access token when print", err)
	}
	fmt.Println(style.BoldText.Render(string(stringCredential)))
	return nil
}

func generateTokenName() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", eris.Wrap(err, "cannot retrieve current user")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "", eris.Wrap(err, "cannot retrieve hostname")
	}

	return fmt.Sprintf("cli_%s@%s_%d", user.Username, hostname, time.Now().Unix()), nil
}

func generateTokenNameWithFallback() string {
	name, err := generateTokenName()
	if err != nil {
		name = fmt.Sprintf("cli_%d", time.Now().Unix())
	}
	return name
}

func pollForAccessToken(ctx context.Context, url string) (login.AccessTokenResponse, error) {
	var accessTokenResponse login.AccessTokenResponse

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return accessTokenResponse, eris.Wrap(err, "cannot fetch access token")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return accessTokenResponse, eris.Wrap(err, "cannot fetch access token")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		retryAfterSeconds, err := strconv.Atoi(resp.Header.Get("Retry-After"))
		if err != nil {
			retryAfterSeconds = defaultRetryAfterSeconds
		}
		t := time.NewTimer(time.Duration(retryAfterSeconds) * time.Second)
		select {
		case <-ctx.Done():
			t.Stop()
		case <-t.C:
		}
		return pollForAccessToken(ctx, url)
	}

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)

		if err != nil {
			return accessTokenResponse, eris.Wrap(err, "cannot read access token response body")
		}

		if err := json.Unmarshal(body, &accessTokenResponse); err != nil {
			return accessTokenResponse, eris.Wrap(err, "cannot unmarshal access token response")
		}

		return accessTokenResponse, nil
	}

	return accessTokenResponse, errors.Errorf("HTTP %s: cannot retrieve access token", resp.Status)
}
