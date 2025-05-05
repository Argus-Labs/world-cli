package forge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

// initFlow represents the initialization flow for the forge system.
type initFlow struct {
	config               ForgeConfig
	State                ForgeCommandState
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

// / OrganizationRequirement is the requirement for the organization step.
type StepRequirement int

const (
	Ignore             StepRequirement = iota
	NeedIDOnly                         // we only need the id (not sure if we will use this or not)
	NeedExistingIDOnly                 // need id but can't create new one (not sure if we will use this or not)
	NeedData                           // we need all the data, can create new one
	NeedExistingData                   // we must all the data but we can't create a new one
	MustNotExist                       // we must not have an this
)

var (
	flow *initFlow
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
// NOTE: we ALWAYS return the state, even if there is an error, so you can use it in your error handling
func SetupForgeCommandState(cmd *cobra.Command,
	loginReq LoginStepRequirement,
	orgReq StepRequirement,
	projectReq StepRequirement) (*ForgeCommandState, error) {
	config, err := GetCurrentForgeConfig()
	if err != nil {
		return nil, err
	}
	// if the repo wan't recognized from the config, and we need project or org, then we need to to do a backend lookup
	needRepoLookup := !config.CurrRepoKnown && projectReq != Ignore && orgReq != Ignore && config.CurrRepoURL != ""

	flow = &initFlow{
		config:               config,
		requiredLogin:        loginReq,
		requiredOrganization: orgReq,
		requiredProject:      projectReq,
		loginStepDone:        false,
		organizationStepDone: false,
		projectStepDone:      false,
		State: ForgeCommandState{
			Command:      cmd,
			User:         nil,
			Organization: nil,
			Project:      nil,
		},
	}
	loggedIn := config.Credential.Token == ""
	// TODO: check for credential expiration

	if flow.requiredLogin == NeedLogin && !loggedIn {
		err := performLogin(cmd.Context(), &config)
		if err != nil {
			return &flow.State, err
		}
		loggedIn = true
	}
	if flow.requiredLogin == NeedLogin {
		user, err := getUser(cmd.Context())
		if err != nil {
			return &flow.State, err
		}
		flow.State.User = &user
		flow.loginStepDone = true
	}

	// if we need to lookup the project based on the git repo, do that now
	if needRepoLookup {
		if !loggedIn {
			return &flow.State, fmt.Errorf("not logged in, can't lookup project from git repo")
		}
		ctx := cmd.Context()
		err := doRepoLookup(ctx, &config)
		if err != nil {
			return &flow.State, err
		}
	}
	needOrgIdOnly := flow.requiredOrganization == NeedIDOnly || flow.requiredOrganization == NeedExistingIDOnly
	needProjectIdOnly := flow.requiredProject == NeedIDOnly || flow.requiredProject == NeedExistingIDOnly
	haveOrgId := flow.config.OrganizationID != ""
	haveProjectId := flow.config.ProjectID != ""

	switch {

	// check for conditions where we can exit early without asking the user for anything

	case flow.requiredOrganization == MustNotExist && haveOrgId:
		// ERROR: we have an org id, but we need to not belong to any org
		return &flow.State, fmt.Errorf("organization already exists")

	case flow.requiredProject == MustNotExist && haveProjectId:
		// ERROR: we have a project id, but we need to not belong to any project
		return &flow.State, fmt.Errorf("project already exists")

	case flow.requiredOrganization == MustNotExist && flow.requiredProject == MustNotExist:
		flow.organizationStepDone = true
		flow.projectStepDone = true
		return &flow.State, nil // everything is as it should be

		// check for only needing IDs and we have them
	case needOrgIdOnly && haveOrgId && needProjectIdOnly && haveProjectId:
		flow.State.Organization = &organization{
			ID: flow.config.OrganizationID,
		}
		flow.State.Project = &project{
			ID:   flow.config.ProjectID,
			Name: flow.config.CurrProjectName,
		}
		flow.organizationStepDone = true
		flow.projectStepDone = true
		return &flow.State, nil // we have the ids we need
	}

	// FIXME: handle the errors coming back from the handleX() functions
	// now make sure we get the org info
	if haveOrgId && needOrgIdOnly {
		flow.State.Organization = &organization{
			ID: flow.config.OrganizationID,
		}
		flow.organizationStepDone = true
	} else {
		switch flow.requiredOrganization {
		case NeedData, NeedIDOnly:
			flow.handleNeedOrgData()
		case NeedExistingData, NeedExistingIDOnly:
			flow.handleNeedExistingOrgData()
		}
	}

	// now get the project info
	if haveProjectId && needProjectIdOnly {
		flow.State.Project = &project{
			ID:   flow.config.ProjectID,
			Name: flow.config.CurrProjectName,
		}
	} else {
		switch flow.requiredProject {
		case NeedData, NeedIDOnly:
			flow.handleNeedProjectData()
		case NeedExistingData, NeedExistingIDOnly:
			flow.handleNeedExistingProjectData()
		}
	}

	// TODO: if the user selected a project, we need to add it to KnownProjects and save the config

	return &flow.State, nil
}

// so you can get the state from anywhere.
func GetForgeCommandState() *ForgeCommandState {
	if flow == nil {
		// this is a logic error so we want to have it fail fast and loudly
		panic("SetupForgeCommandState must be called before GetForgeCommandState")
	}
	return &flow.State
}

// doRepoLookup looks up the project from the git repo and updates the config
// it returns an error if the project is not found or if there is an error
// it returns nil if the project is found and the config is updated
// if the lookup worked but there is no matching project, it will return nil
// and the config will not be changed
func doRepoLookup(ctx context.Context, config *ForgeConfig) error {
	// needed a repo lookup, and we are logged in, so try to lookup the project
	deployURL := fmt.Sprintf("%s/api/project/?url=%s&path=%s",
		baseURL, url.QueryEscape(config.CurrRepoURL), url.QueryEscape(config.CurrRepoPath))
	body, err := sendRequest(ctx, http.MethodGet, deployURL, nil)
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
		AddKnownProject(config, proj)
		// save the config, but don't change the default ProjectID & OrgID
		err := SaveForgeConfig(*config)
		if err != nil {
			fmt.Println("⚠️ Warning: Failed to save config: ", err)
			// continue on, this is not fatal
		}
		// now return a copy of it with the looked up ProjectID and OrganizationID set
		config.ProjectID = proj.ID
		config.OrganizationID = proj.OrgID
		config.CurrProjectName = proj.Name
		config.CurrRepoKnown = true
	}
	return nil
}

func AddKnownProject(config *ForgeConfig, proj *project) {
	config.KnownProjects = append(config.KnownProjects, KnownProject{
		ProjectID:      proj.ID,
		OrganizationID: proj.OrgID,
		RepoURL:        proj.RepoURL,
		RepoPath:       proj.RepoPath,
		ProjectName:    proj.Name,
	})
}

func (flow *initFlow) handleNeedOrgData() error {
	// TODO: get the org data from the user
	// some cases you will need to handle:
	// - the user has no orgs, create a new one
	// - the user has one org, use it or create a new one
	// - the user has multiple orgs, and they need to select one or create a new one
	return nil
}

func (flow *initFlow) handleNeedExistingOrgData() error {
	// TODO: get the org data from the user
	// some cases you will need to handle:
	// - the user has no orgs, fail
	// - the user has one org, use it
	// - the user has multiple orgs, and they need to select one
	return nil
}

func (flow *initFlow) handleNeedProjectData() error {
	// TODO: get the project data
	// some cases you will need to handle:
	// - the user has no projects, create a new one
	// - the user has one project, use it or create a new one
	// - the user has multiple projects, and they need to select one or create a new one
	return nil
}

func (flow *initFlow) handleNeedExistingProjectData() error {
	// TODO: get the project data
	// some cases you will need to handle:
	// - the user has no projects, fail
	// - the user has one project, use it
	// - the user has multiple projects, and they need to select one
	return nil
}
