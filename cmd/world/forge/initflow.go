package forge

import (
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
	MustNotExist                       // we must not have an organization
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
func SetupForgeCommandState(cmd *cobra.Command,
	loginReq LoginStepRequirement,
	orgReq StepRequirement,
	projectReq StepRequirement) (*ForgeCommandState, error) {
	config, err := GetCurrentForgeConfig()
	if err != nil {
		return nil, err
	}
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
	if flow.requiredLogin == NeedLogin && config.Credential.Token == "" {
		// TODO:attempt to login
		// if err != nil {
		// 	return nil, err
		// }
	}
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
