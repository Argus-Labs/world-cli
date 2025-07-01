package root

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	cmdsetup "pkg.world.dev/world-cli/cmd/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/common/printer"
	teaspinner "pkg.world.dev/world-cli/tea/component/spinner"
)

var (
	maxLoginAttempts = 220              // 11 min x 60 sec ÷ 3 sec per attempt = 220 attempts
	tokenLeeway      = 60 * time.Second // expire early to account for clock skew and command execution time

	errPending = eris.New("token status pending")
)

func (h *Handler) Login(ctx context.Context) error {
	err := h.performLogin(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to login")
	}

	setupRequest := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedData,
		ProjectRequired:      models.NeedData,
	}

	state, err := h.setupController.SetupCommandState(ctx, setupRequest)
	if err != nil {
		if !cmdsetup.LoginErrorCheck(err) {
			// Even we have an error, if it's not a login error, we can display the login success message.
			displayLoginSuccess(*h.configService.GetConfig())
		}
		return eris.Wrap(err, "forge command setup failed")
	}

	config := h.configService.GetConfig()
	if config.CurrRepoKnown {
		printer.NewLine(1)
		printer.Headerln("   Known Project Details   ")
		printer.Infof("Organization: %s\n", state.Organization.Name)
		printer.Infof("Org Slug:     %s\n", state.Organization.Slug)
		printer.Infof("Project:      %s\n", state.Project.Name)
		printer.Infof("Project Slug: %s\n", state.Project.Slug)
		printer.Infof("Repository:   %s\n", state.Project.RepoURL)
		printer.NewLine(1)
	}
	// Display login success message
	displayLoginSuccess(*config)

	return nil
}

func (h *Handler) performLogin(ctx context.Context) error {
	var err error
	h.configService.GetConfig().Credential, err = h.loginWithArgusID(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to login")
	}

	h.apiClient.SetAuthToken(h.configService.GetConfig().Credential.Token)

	// Save credential to config
	if err := h.configService.Save(); err != nil {
		return eris.Wrap(err, "Failed to save credential")
	}

	return h.handleArgusIDPostLogin(ctx)
}

func (h *Handler) loginWithArgusID(ctx context.Context) (models.Credential, error) {
	// Get the login link using API client
	loginLink, err := h.apiClient.GetLoginLink(ctx)
	if err != nil {
		return models.Credential{}, eris.Wrap(err, "Failed to get login link")
	}

	// Open browser
	err = h.browserClient.OpenURL(loginLink.ClientURL)
	if err != nil {
		return models.Credential{}, eris.Wrap(err, "Failed to open browser")
	}

	// Wait for user to login
	var token models.LoginToken
	err = h.getToken(ctx, loginLink.CallBackURL, &token)
	if err != nil {
		return models.Credential{}, eris.Wrap(err, "Failed to get token")
	}

	// Parse jwt token to get name from metadata
	cred, err := parseJWTToken(token.JWT)
	if err != nil {
		return models.Credential{}, eris.Wrap(err, "Failed to get name from token")
	}

	return cred, nil
}

func (h *Handler) handleArgusIDPostLogin(ctx context.Context) error {
	user, err := h.apiClient.GetUser(ctx)
	if err != nil {
		errStr := eris.ToString(err, false)
		if strings.Contains(errStr, "503") {
			printer.Errorln("World Forge is currently experiencing issues, please try again later.")
			err = eris.Wrap(err, "handled error")
		}
		return err
	}

	h.configService.GetConfig().Credential.ID = user.ID

	// Update user email if it's different from the JWT token
	if user.Email != h.configService.GetConfig().Credential.Email {
		err = h.apiClient.UpdateUser(ctx, user.Name, user.Email, user.AvatarURL)
		if err != nil {
			return eris.Wrap(err, "Failed to update user email")
		}
	}

	return h.configService.Save()
}

// getToken polls the callback URL until the user completes login.
//
//nolint:gocognit // this is a complex function but it does what it needs to do
func (h *Handler) getToken(ctx context.Context, url string, result *models.LoginToken) error {
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

	// spinnerCompleted will send a message to the spinner to stop and quit.
	spinnerCompleted := func(didLogin bool) {
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
			spinnerCompleted(false)
			return ctx.Err()
		case <-time.After(3 * time.Second):
			log.Debug().Int("attempt", attempts).Msg("login attempt")

			if !spinnerExited.Load() {
				p.Send(teaspinner.LogMsg("Logging in..."))
			}

			token, err := h.apiClient.GetLoginToken(ctx, url)
			if err != nil {
				attempts++
				continue
			}

			if err := handleTokenResponse(token, result); err != nil {
				if errors.Is(err, errPending) {
					attempts++
					continue
				}
				spinnerCompleted(false)
				return err
			}

			spinnerCompleted(true)
			return nil
		}
	}

	spinnerCompleted(false)
	return eris.New("max attempts reached while waiting for token")
}

// handleTokenResponse processes the token response.
func handleTokenResponse(token models.LoginToken, result *models.LoginToken) error {
	switch token.Status {
	case "pending":
		return errPending
	case "success":
		*result = token
		return nil
	default:
		return eris.New(fmt.Sprintf("Status: %s", token.Status))
	}
}

// parseJWTToken parses the JWT token and extracts credential information.
func parseJWTToken(jwtToken string) (models.Credential, error) {
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
		return models.Credential{}, eris.New("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return models.Credential{}, eris.Wrap(err, "failed to decode token payload")
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return models.Credential{}, eris.Wrap(err, "failed to parse token claims")
	}

	return models.Credential{
		Token:          jwtToken,
		TokenExpiresAt: time.Unix(claims.Exp, 0).Add(-tokenLeeway), // expire shortly before real expiration
		Name:           claims.Name,
		ID:             claims.Sub,
		Email:          claims.Email,
	}, nil
}

func displayLoginSuccess(config config.Config) {
	printer.NewLine(1)
	printer.Headerln("   Login successful!  ")
	printer.Infof("Welcome, %s!\n", config.Credential.Name)
	printer.Infof("ID: %s\n", config.Credential.ID)
	printer.Infof("Email: %s\n", config.Credential.Email)
	printer.NewLine(1)
	printer.Infoln("You're all set to start using World Forge!")
}
