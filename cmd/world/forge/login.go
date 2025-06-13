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
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"pkg.world.dev/world-cli/common/printer"
	teaspinner "pkg.world.dev/world-cli/tea/component/spinner"
)

var (
	maxLoginAttempts = 220              // 11 min x 60 sec ÷ 3 sec per attempt = 220 attempts
	tokenLeeway      = 60 * time.Second // expire early to account for clock skew and command execution time

	errPending = eris.New("token status pending")
)

type tokenStruct struct {
	Status string `json:"status"`
	JWT    string `json:"jwt"`
}

// login will open browser to login and save the token to the config file.
func login(fCtx ForgeContext) error {
	// Perform login based on authentication method
	if err := performLogin(fCtx); err != nil {
		return err
	}

	// Handle post-login configuration
	err := fCtx.SetupForgeCommandState(NeedLogin, NeedData, NeedData)
	if err != nil {
		if !loginErrorCheck(err) {
			// Even we have an error, if it's not a login error, we can display the login success message.
			displayLoginSuccess(*fCtx.Config)
		}
		return eris.Wrap(err, "forge command setup failed")
	}

	if fCtx.Config.CurrRepoKnown {
		printer.NewLine(1)
		printer.Headerln("   Known Project Details   ")
		printer.Infof("Organization: %s\n", fCtx.State.Organization.Name)
		printer.Infof("Org Slug:     %s\n", fCtx.State.Organization.Slug)
		printer.Infof("Project:      %s\n", fCtx.State.Project.Name)
		printer.Infof("Project Slug: %s\n", fCtx.State.Project.Slug)
		printer.Infof("Repository:   %s\n", fCtx.State.Project.RepoURL)
		printer.NewLine(1)
	}
	// Display login success message
	displayLoginSuccess(*fCtx.Config)

	return nil
}

func performLogin(fCtx ForgeContext) error {
	var err error
	fCtx.Config.Credential, err = loginWithArgusID(fCtx.Context)
	if err != nil {
		return eris.Wrap(err, "Failed to login")
	}

	// Save credential to config
	if err := fCtx.Config.Save(); err != nil {
		return eris.Wrap(err, "Failed to save credential")
	}

	return handleArgusIDPostLogin(fCtx)
}

func handleArgusIDPostLogin(fCtx ForgeContext) error {
	user, err := getUser(fCtx)
	if err != nil {
		errStr := eris.ToString(err, false)
		if strings.Contains(errStr, "503") {
			printer.Errorln("World Forge is currently experiencing issues, please try again later.")
			err = eris.Wrap(err, ErrHandledError.Error())
		}
		return err
	}

	fCtx.Config.Credential.ID = user.ID
	return fCtx.Config.Save()
}

func displayLoginSuccess(config Config) {
	printer.NewLine(1)
	printer.Headerln("   Login successful!  ")
	printer.Infof("Welcome, %s!\n", config.Credential.Name)
	printer.Infof("Your ID is: %s\n", config.Credential.ID)
	printer.NewLine(1)
	printer.Infoln("You're all set to start using World Forge!")
}

// GetToken will get the token from the config file.
//
//nolint:gocognit // This is a long function, but it's not too complex, better to keep it in one place.
func getToken(ctx context.Context, url string, result any) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Spinner Setup
	spinnerExited := atomic.Bool{}
	var wg sync.WaitGroup
	wg.Add(1)

	spin := teaspinner.Spinner{
		Spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		Cancel:  cancel,
	}
	spin.SetText("Logging in...")
	p := tea.NewProgram(&spin)

	// Run the spinner in a goroutine
	go func() {
		defer wg.Done()
		if _, err := p.Run(); err != nil {
			log.Error().Err(err).Msg("failed to run spinner")
			printer.Infoln("Logging in...") // If the spinner doesn't start correctly, fallback to a simple print.
		}
		spinnerExited.Store(true)
	}()

	// spinnnerCompleted will send a message to the spinner to stop and quit.
	spinnnerCompleted := func(didLogin bool) {
		if !spinnerExited.Load() {
			p.Send(teaspinner.LogMsg("spin: completed"))
			p.Send(tea.Quit())
			wg.Wait()
		}
		if didLogin {
			printer.Successln("Logged in!")
		} else {
			printer.Errorln("Login failed!")
		}
	}

	// Login Loop
	attempts := 1
	for attempts < maxLoginAttempts {
		select {
		case <-ctx.Done():
			spinnnerCompleted(false)
			return ctx.Err()
		case <-time.After(3 * time.Second):
			log.Debug().Int("attempt", attempts).Msg("login attempt")

			if !spinnerExited.Load() {
				p.Send(teaspinner.LogMsg("Logging in..."))
			}

			token, err := makeTokenRequest(ctx, url)
			if err != nil {
				attempts++
				continue
			}

			if err := handleTokenResponse(token, result); err != nil {
				if errors.Is(err, errPending) {
					attempts++
					continue
				}
				spinnnerCompleted(false)
				return err
			}

			spinnnerCompleted(true)
			return nil
		}
	}

	spinnnerCompleted(false)
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

func handleTokenResponse(response []byte, result interface{}) error {
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
		return nil
	default:
		return eris.New(fmt.Sprintf("Status: %s", tokenStruct.Status))
	}
}

func parseJWTToken(jwtToken string) (Credential, error) {
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
	if len(parts) != 3 {
		return Credential{}, eris.New("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Credential{}, eris.Wrap(err, "failed to decode token payload")
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return Credential{}, eris.Wrap(err, "failed to parse token claims")
	}

	return Credential{
		Token:          jwtToken,
		TokenExpiresAt: time.Unix(claims.Exp, 0).Add(-tokenLeeway), // expire shortly before real expiration
		Name:           claims.Name,
		ID:             claims.Sub,
	}, nil
}

func loginWithArgusID(ctx context.Context) (Credential, error) {
	// Get the link to login
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, argusIDAuthURL, nil)
	if err != nil {
		return Credential{}, eris.Wrap(err, "Failed to create request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Credential{}, eris.Wrap(err, "Failed to get login link")
	}
	defer resp.Body.Close()

	// Parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Credential{}, eris.Wrap(err, "Failed to read login link")
	}

	// Parse the response
	var loginLink struct {
		CallBackURL string `json:"callbackUrl"`
		ClientURL   string `json:"clientUrl"`
	}
	err = json.Unmarshal(body, &loginLink)
	if err != nil {
		return Credential{}, eris.Wrap(err, "Failed to parse login link")
	}

	// Open browser
	err = openBrowser(loginLink.ClientURL)
	if err != nil {
		return Credential{}, eris.Wrap(err, "Failed to open browser")
	}

	// Wait for user to login
	var token tokenStruct
	err = getToken(ctx, loginLink.CallBackURL, &token)
	if err != nil {
		return Credential{}, eris.Wrap(err, "Failed to get token")
	}

	// Parse jwt token to get name from metadata
	cred, err := parseJWTToken(token.JWT)
	if err != nil {
		return Credential{}, eris.Wrap(err, "Failed to get name from token")
	}

	return cred, nil
}
