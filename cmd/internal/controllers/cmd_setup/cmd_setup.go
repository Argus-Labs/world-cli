package cmdsetup

import (
	"context"
	"errors"
	"time"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/cmd/internal/clients/api"
	"pkg.world.dev/world-cli/cmd/internal/clients/repo"
	"pkg.world.dev/world-cli/cmd/internal/interfaces"
	"pkg.world.dev/world-cli/cmd/internal/models"
	"pkg.world.dev/world-cli/cmd/internal/services/config"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/printer"
)

func NewController(
	configService config.ServiceInterface,
	repoClient repo.ClientInterface,
	organizationHandler interfaces.OrganizationHandler,
	projectHandler interfaces.ProjectHandler,
	apiClient api.ClientInterface,
) interfaces.CommandSetupController {
	return &Controller{
		configService:       configService,
		repoClient:          repoClient,
		organizationHandler: organizationHandler,
		projectHandler:      projectHandler,
		apiClient:           apiClient,
	}
}

// SetupCommandState performs the setup flow and returns the established state.
func (c *Controller) SetupCommandState(ctx context.Context, req models.SetupRequest) (*models.CommandState, error) {
	result := &models.CommandState{}

	cfg := c.handleConfig()

	// Handle login step
	if err := c.handleLogin(ctx, req.LoginRequired, result, cfg); err != nil {
		return result, err
	}

	// Handle organization invitations if logged in
	if result.LoggedIn && req.LoginRequired != models.IgnoreLogin {
		if err := c.handleOrganizationInvitations(ctx); err != nil {
			return result, err
		}
	}

	// Handle repository lookup if needed
	if err := c.handleRepoLookup(ctx, req, result, cfg); err != nil {
		return result, err
	}

	// Handle state setup
	if err := c.handleStateSetup(ctx, req, result, cfg); err != nil {
		return result, err
	}

	// Save config changes
	if err := c.configService.Save(); err != nil {
		return result, eris.Wrap(err, "failed to save config after setup")
	}

	return result, nil
}

func (c *Controller) handleConfig() *config.Config {
	cfg := c.configService.GetConfig()
	if cfg == nil {
		logger.Error("config is nil")
		cfg = &config.Config{}
	}

	// we deliberately ignore any error here,
	// so that we can fill out and much info as we do have
	cfg.CurrRepoKnown = false
	cfg.CurrRepoPath, cfg.CurrRepoURL, _ = c.repoClient.FindGitPathAndURL()
	if cfg.CurrRepoURL != "" {
		for i := range cfg.KnownProjects {
			knownProject := cfg.KnownProjects[i]
			if knownProject.RepoURL == cfg.CurrRepoURL && knownProject.RepoPath == cfg.CurrRepoPath {
				cfg.ProjectID = knownProject.ProjectID
				cfg.OrganizationID = knownProject.OrganizationID
				cfg.CurrProjectName = knownProject.ProjectName
				cfg.CurrRepoKnown = true
				break
			}
		}
	}
	return cfg
}

// HandleOrganizationInvitations processes any pending organization invitations.
func (c *Controller) handleOrganizationInvitations(ctx context.Context) error {
	orgs, err := c.apiClient.GetOrganizationsInvitedTo(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organizations invitation list")
	}

	if len(orgs) > 0 {
		printer.NewLine(1)
		printer.Headerln("  Organization Invitations  ")
	}

	for _, org := range orgs {
		retry := true
		for retry {
			printer.Infof("You are invited to join the organization: %s [%s]\n", org.Name, org.Slug)
			input := getInput("Would you like to join? [Y/n]", "Y")
			switch input {
			case "Y":
				if err := c.apiClient.AcceptOrganizationInvitation(ctx, org.ID); err != nil {
					return eris.Wrap(err, "failed to accept organization invitation")
				}
				retry = false
			case "n", "":
				retry = false
			default:
				printer.Errorln("Invalid input, must be capital 'Y' or 'n'")
				printer.NewLine(1)
			}
		}
	}
	return nil
}

// handleLogin manages the login verification step.
func (c *Controller) handleLogin(
	ctx context.Context,
	loginReq models.LoginRequirement,
	result *models.CommandState,
	cfg *config.Config,
) error {
	// Check if we have an unexpired token
	loggedIn := cfg.Credential.Token != ""
	tokenExpiresAt := cfg.Credential.TokenExpiresAt
	if tokenExpiresAt.IsZero() || tokenExpiresAt.Before(time.Now()) {
		loggedIn = false
	}

	// If we need to login and we are not logged in, return an error
	if loginReq == models.NeedLogin && !loggedIn {
		printer.Errorln("Login required, please run `world login`")
		return ErrLogin
	}

	// If we need the login step, always get the user info
	if loginReq == models.NeedLogin && loggedIn {
		user, err := c.apiClient.GetUser(ctx)
		if err != nil {
			return err
		}
		result.User = &user
	}

	result.LoggedIn = loggedIn
	return nil
}

// handleRepoLookup manages repository lookup logic.
func (c *Controller) handleRepoLookup(
	ctx context.Context,
	req models.SetupRequest,
	result *models.CommandState,
	cfg *config.Config,
) error {
	// Check if we have a valid repo to look up
	hasValidRepo := !cfg.CurrRepoKnown && cfg.CurrRepoURL != ""

	// Check if either requirement explicitly needs repo lookup
	needsExplicitRepoLookup := req.ProjectRequired == models.NeedRepoLookup ||
		req.OrganizationRequired == models.NeedRepoLookup

	// Check if project needs data (not Ignore) but isn't explicitly NeedRepoLookup
	// We only care about project requirements since repo lookup is project-focused
	needsDataLookup := req.ProjectRequired != models.Ignore && req.ProjectRequired != models.NeedRepoLookup

	needRepoLookup := hasValidRepo && (needsExplicitRepoLookup || needsDataLookup)

	// if we need to lookup the project based on the git repo, do that now
	//nolint:nestif // this is a simple if/else block
	if needRepoLookup {
		if !result.LoggedIn {
			return errors.New("not logged in, can't lookup project from git repo")
		}

		project, err := c.apiClient.LookupProjectFromRepo(ctx, cfg.CurrRepoURL, cfg.CurrRepoPath)
		if err != nil {
			return eris.Wrap(err, "failed to lookup project from git repo")
		}

		if project.ID != "" {
			c.configService.AddKnownProject(
				project.ID,
				project.Name,
				project.OrgID,
				cfg.CurrRepoURL,
				cfg.CurrRepoPath,
			)
			if err := c.configService.Save(); err != nil {
				printer.Notificationf("Warning: Failed to save config: %s", err)
				logger.Error(eris.Wrap(err, "AddKnownProject failed to save config"))
				// continue on, this is not fatal
			}

			// Update config with found project
			cfg.ProjectID = project.ID
			cfg.OrganizationID = project.OrgID
			cfg.CurrProjectName = project.Name
			cfg.CurrRepoKnown = true
		}
	}

	return nil
}

// handleStateSetup manages the final state setup based on requirements
//
//nolint:gocognit // Complex logic, but it's simple to follow
func (c *Controller) handleStateSetup(
	ctx context.Context,
	req models.SetupRequest,
	result *models.CommandState,
	cfg *config.Config,
) error {
	needOrgIDOnly := req.OrganizationRequired == models.NeedIDOnly ||
		req.OrganizationRequired == models.NeedExistingIDOnly
	needProjectIDOnly := req.ProjectRequired == models.NeedIDOnly || req.ProjectRequired == models.NeedExistingIDOnly
	haveOrgID := cfg.OrganizationID != ""
	haveProjectID := cfg.ProjectID != ""

	switch {
	// check for conditions where we can exit early without asking the user for anything
	case req.OrganizationRequired == models.MustNotExist && haveOrgID:
		// ERROR: we have an org id, but we need to not belong to any org
		return errors.New("organization already exists")

	case req.ProjectRequired == models.MustNotExist && haveProjectID:
		// ERROR: we have a project id, but we need to not belong to any project
		return errors.New("project already exists")

	case req.OrganizationRequired == models.MustNotExist && req.ProjectRequired == models.MustNotExist:
		return nil // everything is as it should be

		// check for only needing IDs and we have them
	case needOrgIDOnly && haveOrgID && needProjectIDOnly && haveProjectID:
		result.Organization = &models.Organization{
			ID: cfg.OrganizationID,
		}
		result.Project = &models.Project{
			ID:   cfg.ProjectID,
			Name: cfg.CurrProjectName,
		}
		return nil // we have the ids we need
	}

	if c.inKnownRepo(ctx, cfg, result) {
		return nil // we have the data we need
	}

	// FIXME: handle the errors coming back from the handleX() functions
	// now make sure we get the org info
	if haveOrgID && needOrgIDOnly {
		result.Organization = &models.Organization{
			ID: cfg.OrganizationID,
		}
	} else {
		switch req.OrganizationRequired { //nolint:exhaustive // don't need to handle all cases
		case models.NeedData, models.NeedIDOnly:
			if err := c.handleNeedOrgData(ctx, result, cfg); err != nil {
				return err
			}
		case models.NeedExistingData, models.NeedExistingIDOnly:
			if err := c.handleNeedExistingOrgData(ctx, result, cfg); err != nil {
				return err
			}
		}
	}

	// now get the project info
	if haveProjectID && needProjectIDOnly {
		result.Project = &models.Project{
			ID:   cfg.ProjectID,
			Name: cfg.CurrProjectName,
		}
	} else {
		switch req.ProjectRequired { //nolint:exhaustive // don't need to handle all cases
		case models.NeedData, models.NeedIDOnly:
			if err := c.handleNeedProjectData(ctx, result, cfg); err != nil {
				return err
			}
		case models.NeedExistingData, models.NeedExistingIDOnly:
			if err := c.handleNeedExistingProjectData(ctx, result, cfg); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Controller) inKnownRepo(ctx context.Context, cfg *config.Config, result *models.CommandState) bool {
	if cfg.CurrRepoKnown {
		org, err := c.apiClient.GetOrganizationByID(ctx, cfg.OrganizationID)
		if err != nil {
			return false
		}
		proj, err := c.apiClient.GetProjectByID(ctx, cfg.ProjectID)
		if err != nil {
			return false
		}
		c.updateProject(cfg, &proj, result)
		c.updateOrganization(cfg, &org, result)
		cfg.CurrRepoKnown = true
		return true
	}
	return false
}

// getInput is a simple input helper (this should be injected for better testing).
var getInput = func(_, defaultStr string) string {
	// This is a simplified version - the real implementation would be more robust
	return defaultStr
}
