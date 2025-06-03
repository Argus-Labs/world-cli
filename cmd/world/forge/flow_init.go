package forge

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
)

var (
	ErrLogin = eris.New("not logged in")
)

// initFlow represents the initialization flow for the forge system.
type initFlow struct {
	requiredLogin        LoginStepRequirement
	requiredOrganization StepRequirement
	requiredProject      StepRequirement
	loginStepDone        bool
	organizationStepDone bool
	projectStepDone      bool
}

// / LoginStepRequirement is the requirement for the login step.
type LoginStepRequirement int

const (
	IgnoreLogin LoginStepRequirement = iota // don't care if we are logged in or not
	NeedLogin
)

// StepRequirement is the requirement used for the organization, and project steps.
type StepRequirement int

const (
	Ignore             StepRequirement = iota
	NeedRepoLookup                     // we need to lookup the project from the git repo
	NeedIDOnly                         // we only need the id (not sure if we will use this or not)
	NeedExistingIDOnly                 // need id but can't create new one (not sure if we will use this or not)
	NeedData                           // we need all the data, can create new one
	NeedExistingData                   // we must all the data but we can't create a new one
	MustNotExist                       // we must not have this
)

// SetupForgeCommandState initializes the forge system and returns the completed state
// it should be called at the beginning of the command before doing the work of the command itself
// the requirements flags are used to tell the system what state we much be in before we can run the command
// the setup will go through each step and return an error if the state is not met
// some steps, such are reading the org or project data require a login; but you can control actual login behavior
// separately with the loginReq flag.
// For example:
//
//	NeedLogin, NeedData will try to login if needed, and only fail if it can't login.
//	IgnoreLogin, NeedData will fail if we are not already logged in, and will not attempt to login.
//	IgnoreLogin, NeedExistingIDOnly will not try to login, and will try different ways to figure out the OrgID
//	               including sending requests to the server if we are logged in. But if we aren't logged in and
//	               don't have an existing org id already known via config or other means, it will fail.
//
// NOTE: we ALWAYS return the state, even if there is an error, so you can use it in your error handling.
func (fCtx *ForgeContext) SetupForgeCommandState(
	loginReq LoginStepRequirement,
	orgReq StepRequirement,
	projectReq StepRequirement,
) error {
	if fCtx == nil || fCtx.Config == nil {
		return errors.New("ForgeContext is nil or Config is nil")
	}

	flow := &initFlow{
		requiredLogin:        loginReq,
		requiredOrganization: orgReq,
		requiredProject:      projectReq,
		loginStepDone:        false,
		organizationStepDone: false,
		projectStepDone:      false,
	}

	if err := flow.handleLogin(fCtx); err != nil {
		return err
	}

	if err := flow.handleRepoLookup(fCtx); err != nil {
		return err
	}

	if err := flow.handleStateSetup(fCtx); err != nil {
		return err
	}

	return nil
}

func (flow *initFlow) handleLogin(fCtx *ForgeContext) error {
	// if we have an unexpired token, we are logged in
	loggedIn := fCtx.Config.Credential.Token != ""
	tokenExpiresAt := fCtx.Config.Credential.TokenExpiresAt
	if tokenExpiresAt.IsZero() || tokenExpiresAt.Before(time.Now()) {
		loggedIn = false
	}

	// if we need to login and we are not logged in, return an error
	if flow.requiredLogin == NeedLogin && !loggedIn {
		printer.Errorln("Login required, please run `world login`")
		return ErrLogin
	}

	// if we need the login step, always get the user info
	if flow.requiredLogin == NeedLogin {
		user, err := getUser(*fCtx)
		if err != nil {
			return err
		}
		fCtx.State.User = &user
		flow.loginStepDone = true
	}
	fCtx.State.LoggedIn = loggedIn
	return nil
}

func (flow *initFlow) handleRepoLookup(fCtx *ForgeContext) error {
	// Check if we have a valid repo to look up
	hasValidRepo := !fCtx.Config.CurrRepoKnown && fCtx.Config.CurrRepoURL != ""

	// Check if either requirement explicitly needs repo lookup
	needsExplicitRepoLookup := flow.requiredProject == NeedRepoLookup ||
		flow.requiredOrganization == NeedRepoLookup

	// Check if project needs data (not Ignore) but isn't explicitly NeedRepoLookup
	// We only care about project requirements since repo lookup is project-focused
	needsDataLookup := flow.requiredProject != Ignore && flow.requiredProject != NeedRepoLookup

	needRepoLookup := hasValidRepo && (needsExplicitRepoLookup || needsDataLookup)

	// if we need to lookup the project based on the git repo, do that now
	if needRepoLookup {
		if !fCtx.State.LoggedIn {
			return errors.New("not logged in, can't lookup project from git repo")
		}
		err := flow.doRepoLookup(fCtx)
		if err != nil {
			return err
		}
	}
	return nil
}

//nolint:gocognit // Complex logic, but it's simple to follow
func (flow *initFlow) handleStateSetup(fCtx *ForgeContext) error {
	needOrgIDOnly := flow.requiredOrganization == NeedIDOnly || flow.requiredOrganization == NeedExistingIDOnly
	needProjectIDOnly := flow.requiredProject == NeedIDOnly || flow.requiredProject == NeedExistingIDOnly
	haveOrgID := fCtx.Config.OrganizationID != ""
	haveProjectID := fCtx.Config.ProjectID != ""

	switch {
	// check for conditions where we can exit early without asking the user for anything
	case flow.requiredOrganization == MustNotExist && haveOrgID:
		// ERROR: we have an org id, but we need to not belong to any org
		return errors.New("organization already exists")

	case flow.requiredProject == MustNotExist && haveProjectID:
		// ERROR: we have a project id, but we need to not belong to any project
		return errors.New("project already exists")

	case flow.requiredOrganization == MustNotExist && flow.requiredProject == MustNotExist:
		flow.organizationStepDone = true
		flow.projectStepDone = true
		return nil // everything is as it should be

		// check for only needing IDs and we have them
	case needOrgIDOnly && haveOrgID && needProjectIDOnly && haveProjectID:
		fCtx.State.Organization = &organization{
			ID: fCtx.Config.OrganizationID,
		}
		fCtx.State.Project = &project{
			ID:   fCtx.Config.ProjectID,
			Name: fCtx.Config.CurrProjectName,
		}
		flow.organizationStepDone = true
		flow.projectStepDone = true
		return nil // we have the ids we need
	}

	if flow.inKnownRepo(fCtx) {
		return nil // we have the data we need
	}

	// FIXME: handle the errors coming back from the handleX() functions
	// now make sure we get the org info
	if haveOrgID && needOrgIDOnly {
		fCtx.State.Organization = &organization{
			ID: fCtx.Config.OrganizationID,
		}
		flow.organizationStepDone = true
	} else {
		switch flow.requiredOrganization { //nolint:exhaustive // don't need to handle all cases
		case NeedData, NeedIDOnly:
			if err := flow.handleNeedOrgData(fCtx); err != nil {
				return err
			}
		case NeedExistingData, NeedExistingIDOnly:
			if err := flow.handleNeedExistingOrgData(fCtx); err != nil {
				return err
			}
		}
	}

	// now get the project info
	if haveProjectID && needProjectIDOnly {
		fCtx.State.Project = &project{
			ID:   fCtx.Config.ProjectID,
			Name: fCtx.Config.CurrProjectName,
		}
	} else {
		switch flow.requiredProject { //nolint:exhaustive // don't need to handle all cases
		case NeedData, NeedIDOnly:
			if err := flow.handleNeedProjectData(fCtx); err != nil {
				return err
			}
		case NeedExistingData, NeedExistingIDOnly:
			if err := flow.handleNeedExistingProjectData(fCtx); err != nil {
				return err
			}
		}
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// Helper functions
///////////////////////////////////////////////////////////////////////////////

// doRepoLookup looks up the project from the git repo and updates the config
// it returns an error if the project is not found or if there is an error
// it returns nil if the project is found and the config is updated
// if the lookup worked but there is no matching project, it will return nil
// and the config will not be changed.
func (flow *initFlow) doRepoLookup(fCtx *ForgeContext) error {
	// needed a repo lookup, and we are logged in, so try to lookup the project
	deployURL := fmt.Sprintf("%s/api/project/?url=%s&path=%s",
		baseURL, url.QueryEscape(fCtx.Config.CurrRepoURL),
		url.QueryEscape(fCtx.Config.CurrRepoPath))
	body, err := sendRequest(*fCtx, http.MethodGet, deployURL, nil)
	if err != nil {
		// we need this, so fail if we can't get it
		return fmt.Errorf("failed to lookup project from git repo: %w", err)
	}

	// Parse response
	proj, err := parseResponse[project](body)
	if err != nil && err.Error() != "Missing data field in response" {
		// missing data field in response just means nothing was found
		// but any other error is a problem
		return err
	}
	if proj != nil {
		// add to list of known projects
		proj.AddKnownProject(fCtx.Config)
		// now return a copy of it with the looked up ProjectID and OrganizationID set
		fCtx.Config.ProjectID = proj.ID
		fCtx.Config.OrganizationID = proj.OrgID
		fCtx.Config.CurrProjectName = proj.Name
		fCtx.Config.CurrRepoKnown = true
	}
	return nil
}

func (flow *initFlow) inKnownRepo(fCtx *ForgeContext) bool {
	if fCtx.Config.CurrRepoKnown {
		org, err := getOrganizationDataByID(*fCtx, fCtx.Config.OrganizationID)
		if err != nil {
			return false
		}
		proj, err := getProjectDataByID(*fCtx, fCtx.Config.ProjectID)
		if err != nil {
			return false
		}
		flow.updateProject(fCtx, &proj)
		flow.updateOrganization(fCtx, &org)
		fCtx.Config.CurrRepoKnown = true
		return true
	}
	return false
}

// loginErrorCheck is used to check if the error is a login error.
// Used to prevent reprinting the error which was already printed in a user friendly manner.
func loginErrorCheck(err error) bool {
	if err == nil {
		return false
	}
	return eris.Is(err, ErrLogin)
}
