package forge

import (
	"context"
	"fmt"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
)

const (
	// For local development.
	worldForgeBaseURLLocal = "http://localhost:8001"

	// For Argus Dev.
	worldForgeBaseURLDev = "https://forge.argus.dev"

	// For Argus Production.
	worldForgeBaseURLProd = "https://forge.world.dev"

	// For local development.
	worldForgeRPCBaseURLLocal = "http://localhost:8002/rpc"

	// RPC Dev URL.
	worldForgeRPCBaseURLDev = "https://rpc.argus.dev"

	// RPC Prod URL.
	worldForgeRPCBaseURLProd = "https://rpc.world.dev"

	// For Argus ID Dev.
	argusIDBaseURLDev = "https://id.argus.dev"

	// For Argus ID Production.
	argusIDBaseURLProd = "https://id.argus.gg"
)

var (
	// baseUrl is the base URL for the Forge API.
	baseURL string
	rpcURL  string

	// login url stuff.
	argusIDBaseURL string
	argusIDAuthURL string

	// organization url stuff.
	organizationURL string

	// project url stuff.
	projectURLPattern = "%s/api/organization/%s/project"

	// user url stuff.
	userURL string

	// Env is the environment to use for the Forge API.
	Env = "PROD"
)

//nolint:lll // needed to put all the help text in the same line
var ForgeCmdPlugin struct {
	Login   *LoginCmd   `cmd:"" group:"Getting Started:" help:"Login to World Forge, creating a new account if necessary"`
	Deploy  *DeployCmd  `cmd:"" group:"Getting Started:" help:"Deploy your World Forge project to a TEST environment in the cloud"`
	Status  *StatusCmd  `cmd:"" group:"Getting Started:" help:"Check the status of your deployed World Forge project"`
	Promote *PromoteCmd `cmd:"" group:"Cloud Management Commands:" help:"Deploy your game project to a LIVE environment in the cloud"`
	Destroy *DestroyCmd `cmd:"" group:"Cloud Management Commands:" help:"Remove your game project's deployed infrastructure from the cloud"`
	Reset   *ResetCmd   `cmd:"" group:"Cloud Management Commands:" help:"Restart your game project with a clean state"`
	Logs    *LogsCmd    `cmd:"" group:"Cloud Management Commands:" help:"Tail logs for your game project"`
	Forge   *ForgeCmd   `cmd:""`
	User    *UserCmd    `cmd:""`
}

type ForgeCmd struct { //nolint:revive // this is the "forge" command within the "world" command
	Organization *OrganizationCmd `cmd:"" aliases:"org"  group:"Organization Commands:" help:"Manage your organizations"`
	Project      *ProjectCmd      `cmd:"" aliases:"proj" group:"Project Commands:"      help:"Manage your projects"`
}

// ------------------------------------------------------------------------------------------------
// Top level commands
// ------------------------------------------------------------------------------------------------

type LoginCmd struct {
}

func (c *LoginCmd) Run() error {
	err := login(context.Background())
	if err != nil {
		return eris.Wrap(err, "Login Failed: ")
	}
	return nil
}

type DeployCmd struct {
	Force bool `flag:"" help:"Force the deployment"`
}

func (c *DeployCmd) Run() error {
	deployType := "deploy"
	if c.Force {
		deployType = "forceDeploy"
	}
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingIDOnly, NeedExistingData)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return deployment(ctx, cmdState, deployType)
}

type StatusCmd struct {
}

func (c *StatusCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingIDOnly, NeedExistingData)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return status(ctx, cmdState)
}

type PromoteCmd struct {
}

func (c *PromoteCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingIDOnly, NeedExistingData)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return deployment(ctx, cmdState, "promote")
}

type DestroyCmd struct {
}

func (c *DestroyCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingIDOnly, NeedExistingData)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return deployment(ctx, cmdState, "destroy")
}

type ResetCmd struct {
}

func (c *ResetCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingIDOnly, NeedExistingData)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return deployment(ctx, cmdState, "reset")
}

//nolint:lll // needed to put all the help text in the same line
type LogsCmd struct {
	Region string `arg:"" enum:"ap-southeast-1,eu-central-1,us-east-1,us-west-2" default:"us-west-2" optional:"" help:"The region to tail logs for"`
	Env    string `arg:"" enum:"test,live"                                       default:"test"      optional:"" help:"The environment to tail logs for"`
}

func (c *LogsCmd) Run() error {
	ctx := context.Background()
	_, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingIDOnly, NeedExistingIDOnly)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return tailLogs(ctx, c.Region, c.Env)
}

// ------------------------------------------------------------------------------------------------
// Organization commands
// ------------------------------------------------------------------------------------------------

type OrganizationCmd struct {
	Create *CreateOrganizationCmd `cmd:"" group:"Organization Commands:" help:"Create a new organization"`
	Switch *SwitchOrganizationCmd `cmd:"" group:"Organization Commands:" help:"Switch to an organization"`
}

type CreateOrganizationCmd struct {
	Name      string `flag:"" help:"The name of the organization"`
	Slug      string `flag:"" help:"The slug of the organization"`
	AvatarURL string `flag:"" help:"The avatar URL of the organization" type:"url"`
}

func (c *CreateOrganizationCmd) Run() error {
	ctx := context.Background()
	_, err := SetupForgeCommandState(ctx, NeedLogin, Ignore, Ignore)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}

	_, err = createOrganization(ctx, c)
	return err
}

type SwitchOrganizationCmd struct {
	Slug string `flag:"" help:"The slug of the organization to switch to"`
}

func (c *SwitchOrganizationCmd) Run() error {
	ctx := context.Background()
	_, err := SetupForgeCommandState(ctx, NeedLogin, Ignore, Ignore)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}

	_, err = selectOrganization(ctx, c)
	return err
}

// ------------------------------------------------------------------------------------------------
// Project commands
// ------------------------------------------------------------------------------------------------

type ProjectCmd struct {
	Create *CreateProjectCmd `cmd:"" group:"Project Commands:" help:"Create a new project"`
	Switch *SwitchProjectCmd `cmd:"" group:"Project Commands:" help:"Switch to a different project"`
	Update *UpdateProjectCmd `cmd:"" group:"Project Commands:" help:"Update your project"`
	Delete *DeleteProjectCmd `cmd:"" group:"Project Commands:" help:"Delete your project"`
}

type CreateProjectCmd struct {
	Name      string `flag:"" help:"The name of the project"`
	Slug      string `flag:"" help:"The slug of the project"`
	AvatarURL string `flag:"" help:"The avatar URL of the project" type:"url"`
}

func (c *CreateProjectCmd) Run() error {
	ctx := context.Background()
	_, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingData, Ignore)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}

	_, err = createProject(ctx, c)
	return err
}

type SwitchProjectCmd struct {
	Slug string `flag:"" help:"The slug of the project to switch to"`
}

func (c *SwitchProjectCmd) Run() error {
	ctx := context.Background()
	_, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingData, Ignore)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}

	_, err = selectProject(ctx, c)
	return err
}

type UpdateProjectCmd struct {
	Name      string `flag:"" help:"The new name of the project"`
	Slug      string `flag:"" help:"The new slug of the project"`
	AvatarURL string `flag:"" help:"The new avatar URL of the project" type:"url"`
}

func (c *UpdateProjectCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingData, NeedExistingData)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	if cmdState.Project == nil {
		return eris.New("Forge setup failed, no project selected")
	}
	return cmdState.Project.updateProject(ctx, c)
}

type DeleteProjectCmd struct {
}

func (c *DeleteProjectCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingData, NeedExistingData)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	if cmdState.Project == nil {
		return eris.New("Forge setup failed, no project selected")
	}
	return cmdState.Project.delete(ctx)
}

// ------------------------------------------------------------------------------------------------
// User commands
// ------------------------------------------------------------------------------------------------

//nolint:lll // needed to put all the help text in the same line
type UserCmd struct {
	Invite *InviteUserToOrganizationCmd     `cmd:"" group:"User Commands:" optional:"" help:"Invite a user to an organization"`
	Role   *ChangeUserRoleInOrganizationCmd `cmd:"" group:"User Commands:" optional:"" help:"Change a user's role in an organization"`
	Update *UpdateUserCmd                   `cmd:"" group:"User Commands:" optional:"" help:"Update a user"`
}

type InviteUserToOrganizationCmd struct {
	ID   string `flag:"" help:"The ID of the user to invite"`
	Role string `flag:"" help:"The role of the user to invite"`
}

func (c *InviteUserToOrganizationCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingData, Ignore)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	if cmdState.Organization.ID == "" {
		return eris.New("Forge setup failed, no organization selected")
	}
	return cmdState.Organization.inviteUser(ctx, c)
}

type ChangeUserRoleInOrganizationCmd struct {
	ID   string `flag:"" help:"The ID of the user to change the role of"`
	Role string `flag:"" help:"The new role of the user"`
}

func (c *ChangeUserRoleInOrganizationCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedExistingData, Ignore)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	if cmdState.Organization.ID == "" {
		return eris.New("Forge setup failed, no organization selected")
	}
	return cmdState.Organization.updateUserRole(ctx, c)
}

type UpdateUserCmd struct {
	Email     string `flag:"" help:"The email of the user to update"`
	Name      string `flag:"" help:"The new name of the user"`
	AvatarURL string `flag:"" help:"The new avatar URL of the user"  type:"url"`
}

func (c *UpdateUserCmd) Run() error {
	ctx := context.Background()
	_, err := SetupForgeCommandState(ctx, NeedLogin, Ignore, Ignore)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return updateUser(ctx, c)
}

func InitForgeBase(env string) {
	// Set urls based on env
	switch env {
	case EnvLocal:
		baseURL = worldForgeBaseURLLocal
		rpcURL = worldForgeRPCBaseURLLocal
		argusIDBaseURL = argusIDBaseURLDev
		Env = EnvLocal
		printer.Notificationln("Forge Env: LOCAL")
	case EnvDev:
		baseURL = worldForgeBaseURLDev
		rpcURL = worldForgeRPCBaseURLDev
		argusIDBaseURL = argusIDBaseURLDev
		Env = EnvDev
		printer.Notificationln("Forge Env: DEV")
	default:
		rpcURL = worldForgeRPCBaseURLProd
		baseURL = worldForgeBaseURLProd
		argusIDBaseURL = argusIDBaseURLProd
		Env = EnvProd
	}

	// Set login URL
	argusIDAuthURL = fmt.Sprintf("%s/api/auth/service-auth-session", argusIDBaseURL)

	// Set organization URL
	organizationURL = fmt.Sprintf("%s/api/organization", baseURL)

	// Set user URL
	userURL = fmt.Sprintf("%s/api/user", baseURL)
}
