package forge

import (
	"context"
	"fmt"
	"strings"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/logger"
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
	Login        *LoginCmd        `cmd:"" group:"Getting Started:" help:"Login to World Forge, creating a new account if necessary"`
	Deploy       *DeployCmd       `cmd:"" group:"Getting Started:" help:"Deploy your World Forge project to a TEST environment in the cloud"`
	Status       *StatusCmd       `cmd:"" group:"Getting Started:" help:"Check the status of your deployed World Forge project"`
	Promote      *PromoteCmd      `cmd:"" group:"Cloud Management Commands:" help:"Deploy your game project to a LIVE environment in the cloud"`
	Destroy      *DestroyCmd      `cmd:"" group:"Cloud Management Commands:" help:"Remove your game project's deployed infrastructure from the cloud"`
	Reset        *ResetCmd        `cmd:"" group:"Cloud Management Commands:" help:"Restart your game project with a clean state"`
	Logs         *LogsCmd         `cmd:"" group:"Cloud Management Commands:" help:"Tail logs for your game project"`
	Organization *OrganizationCmd `cmd:"" aliases:"org"  group:"Organization Commands:" help:"Manage your organizations"`
	Project      *ProjectCmd      `cmd:"" aliases:"proj" group:"Project Commands:"      help:"Manage your projects"`
	User         *UserCmd         `cmd:""`
}

// ------------------------------------------------------------------------------------------------
// Top level commands
// ------------------------------------------------------------------------------------------------

type LoginCmd struct {
}

func (c *LoginCmd) Run() error {
	return WithForgeContextSetup(IgnoreLogin, Ignore, Ignore, login)
}

type DeployCmd struct {
	Force bool `flag:"" help:"Force the deployment"`
}

func (c *DeployCmd) Run() error {
	deployType := DeploymentTypeDeploy
	if c.Force {
		deployType = DeploymentTypeForceDeploy
	}
	return WithForgeContextSetup(NeedLogin, NeedExistingIDOnly, NeedExistingData, func(fCtx ForgeContext) error {
		return deployment(fCtx, deployType)
	})
}

type StatusCmd struct {
}

func (c *StatusCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingData, NeedExistingData, status)
}

type PromoteCmd struct {
}

func (c *PromoteCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingIDOnly, NeedExistingData, func(fCtx ForgeContext) error {
		return deployment(fCtx, DeploymentTypePromote)
	})
}

type DestroyCmd struct {
}

func (c *DestroyCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingIDOnly, NeedExistingData, func(fCtx ForgeContext) error {
		return deployment(fCtx, DeploymentTypeDestroy)
	})
}

type ResetCmd struct {
}

func (c *ResetCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingIDOnly, NeedExistingData, func(fCtx ForgeContext) error {
		return deployment(fCtx, DeploymentTypeReset)
	})
}

//nolint:lll // needed to put all the help text in the same line
type LogsCmd struct {
	Region string `arg:"" enum:"ap-southeast-1,eu-central-1,us-east-1,us-west-2" default:"us-west-2" optional:"" help:"The region to tail logs for"`
	Env    string `arg:"" enum:"test,live"                                       default:"test"      optional:"" help:"The environment to tail logs for"`
}

func (c *LogsCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingIDOnly, NeedExistingIDOnly, func(fCtx ForgeContext) error {
		return tailLogs(fCtx, c.Region, c.Env)
	})
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
	return WithForgeContextSetup(NeedLogin, NeedRepoLookup, NeedRepoLookup, func(fCtx ForgeContext) error {
		_, err := createOrganization(fCtx, c)
		return err
	})
}

type SwitchOrganizationCmd struct {
	Slug string `flag:"" help:"The slug of the organization to switch to"`
}

func (c *SwitchOrganizationCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedRepoLookup, NeedRepoLookup, func(fCtx ForgeContext) error {
		_, err := selectOrganization(fCtx, c)
		return err
	})
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
	return WithForgeContextSetup(NeedLogin, NeedExistingData, NeedRepoLookup, func(fCtx ForgeContext) error {
		_, err := createProject(fCtx, c)
		return err
	})
}

type SwitchProjectCmd struct {
	Slug string `flag:"" help:"The slug of the project to switch to"`
}

func (c *SwitchProjectCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingData, NeedRepoLookup, func(fCtx ForgeContext) error {
		_, err := selectProject(fCtx, c, false)
		return err
	})
}

type UpdateProjectCmd struct {
	Name      string `flag:"" help:"The new name of the project"`
	Slug      string `flag:"" help:"The new slug of the project"`
	AvatarURL string `flag:"" help:"The new avatar URL of the project" type:"url"`
}

func (c *UpdateProjectCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingData, NeedExistingData, func(fCtx ForgeContext) error {
		if fCtx.State.Project == nil {
			return eris.New("Forge setup failed, no project selected")
		}
		return fCtx.State.Project.updateProject(fCtx, c)
	})
}

type DeleteProjectCmd struct {
}

func (c *DeleteProjectCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingData, NeedExistingData, func(fCtx ForgeContext) error {
		if fCtx.State.Project == nil {
			return eris.New("Forge setup failed, no project selected")
		}
		return fCtx.State.Project.delete(fCtx)
	})
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
	Email string `flag:"" help:"The email of the user to invite"`
	Role  string `flag:"" help:"The role of the user to invite"`
}

func (c *InviteUserToOrganizationCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingData, Ignore, func(fCtx ForgeContext) error {
		if fCtx.State.Organization == nil {
			return eris.New("Forge setup failed, no organization selected")
		}
		return fCtx.State.Organization.inviteUser(fCtx, c)
	})
}

type ChangeUserRoleInOrganizationCmd struct {
	Email string `flag:"" help:"The email of the user to change the role of"`
	Role  string `flag:"" help:"The new role of the user"`
}

func (c *ChangeUserRoleInOrganizationCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, NeedExistingData, Ignore, func(fCtx ForgeContext) error {
		if fCtx.State.Organization == nil {
			return eris.New("Forge setup failed, no organization selected")
		}
		return fCtx.State.Organization.updateUserRole(fCtx, c)
	})
}

type UpdateUserCmd struct {
	Email     string `flag:"" help:"The email of the user to update"`
	Name      string `flag:"" help:"The new name of the user"`
	AvatarURL string `flag:"" help:"The new avatar URL of the user"  type:"url"`
}

func (c *UpdateUserCmd) Run() error {
	return WithForgeContextSetup(NeedLogin, Ignore, Ignore, func(fCtx ForgeContext) error {
		return updateUser(fCtx, c)
	})
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

func WithForgeContextSetup(
	needLogin LoginStepRequirement, needOrg, needProject StepRequirement,
	handler func(fCtx ForgeContext) error,
) error {
	ctx := context.Background()

	cfg, err := GetCurrentForgeConfig()
	if err != nil {
		printer.Notificationf("Warning: failed to load config: %s", err)
		logger.Error(eris.Wrap(err, "WithForgeSetup failed to get config"))
		return err
	}

	fCtx := ForgeContext{
		Context: ctx,
		Config:  &cfg,
	}

	err = fCtx.SetupForgeCommandState(needLogin, needOrg, needProject)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}

	// Call the handler and wait for it to finish
	if err := handler(fCtx); err != nil {
		if strings.Contains(err.Error(), ErrCannotSaveConfig.Error()) {
			printer.Errorln("Need to reset config, not implemented yet")
			printer.Errorln("Go to homeDir/.worldcli/config.json and delete the file")
			// TODO: reset the config
		}
		return err
	}

	return fCtx.Config.Save()
}
