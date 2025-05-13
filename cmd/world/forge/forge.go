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

var ForgeCmdPlugin struct {
	Login   *LoginCmd   `cmd:"" group:"Getting Started:" help:"Login to World Forge, creating a new account if necessary"`
	Deploy  *DeployCmd  `cmd:"" group:"Getting Started:" help:"Deploy your World Forge project to a TEST environment in the cloud"`
	Status  *StatusCmd  `cmd:"" group:"Getting Started:" help:"Check the status of your deployed World Forge project"`
	Forge   *ForgeCmd   `cmd:""`
	User    *UserCmd    `cmd:""`
	Promote *PromoteCmd `cmd:"" group:"Management Commands:" help:"Deploy your game project to a LIVE environment in the cloud"`
	Destroy *DestroyCmd `cmd:"" group:"Management Commands:" help:"Remove your game project's deployed infrastructure from the cloud"`
	Reset   *ResetCmd   `cmd:"" group:"Management Commands:" help:"Restart your game project with a clean state"`
	Logs    *LogsCmd    `cmd:"" group:"Management Commands:" help:"Tail logs for your game project"`
}

type ForgeCmd struct {
	Organization *OrganizationCmd `cmd:"" aliases:"org" group:"Organization Commands:" help:"Manage your organizations"`
	Project      *ProjectCmd      `cmd:"" aliases:"proj" group:"Project Commands:" help:"Manage your projects"`
}

// ------------------------------------------------------------------------------------------------
// Top level commands
// ------------------------------------------------------------------------------------------------

type LoginCmd struct {
}

func (c *LoginCmd) Run() error {
	return login(context.Background())
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
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedIDOnly, NeedIDOnly)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return deployment(ctx, cmdState, deployType)
}

type StatusCmd struct {
}

func (c *StatusCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedIDOnly, NeedIDOnly)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return status(ctx, cmdState)
}

type PromoteCmd struct {
}

func (c *PromoteCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedIDOnly, NeedIDOnly)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return deployment(ctx, cmdState, "promote")
}

type DestroyCmd struct {
}

func (c *DestroyCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedIDOnly, NeedIDOnly)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return deployment(ctx, cmdState, "destroy")
}

type ResetCmd struct {
}

func (c *ResetCmd) Run() error {
	ctx := context.Background()
	cmdState, err := SetupForgeCommandState(ctx, NeedLogin, NeedIDOnly, NeedIDOnly)
	if err != nil {
		return eris.Wrap(err, "forge command setup failed")
	}
	return deployment(ctx, cmdState, "reset")
}

type LogsCmd struct {
	Region string `arg:"" enum:"ap-southeast-1,eu-central-1,us-east-1,us-west-2" default:"us-west-2" optional:"" help:"The region to tail logs for"`
	Env    string `arg:"" enum:"test,live" default:"test" optional:"" help:"The environment to tail logs for"`
}

func (c *LogsCmd) Run() error {
	return tailLogs(context.Background(), c.Region, c.Env)
}

// ------------------------------------------------------------------------------------------------
// Organization commands
// ------------------------------------------------------------------------------------------------

type OrganizationCmd struct {
	Create *CreateOrganizationCmd `cmd:"" group:"Organization Commands:" help:"Create a new organization"`
	Switch *SwitchOrganizationCmd `cmd:"" group:"Organization Commands:" help:"Switch to an organization"`
}

type CreateOrganizationCmd struct {
	Name      string `arg:"" optional:"" help:"The name of the organization"`
	Slug      string `arg:"" optional:"" help:"The slug of the organization"`
	AvatarURL string `arg:"" optional:"" type:"url" help:"The avatar URL of the organization"`
}

func (c *CreateOrganizationCmd) Run() error {
	// TODO: pass in name, slug, and avatarURL if provided
	org, err := createOrganization(context.Background())
	if err != nil {
		return eris.Wrap(err, "Failed to create organization")
	}
	printer.Successf("Created organization: %s\n", org.Name)
	return nil
}

type SwitchOrganizationCmd struct {
	Slug string `arg:"" optional:"" help:"The slug of the organization to switch to"`
}

func (c *SwitchOrganizationCmd) Run() error {
	// TODO: pass in slug if provided
	org, err := selectOrganization(context.Background())
	if err != nil {
		return eris.Wrap(err, "Failed to switch organization")
	}
	printer.Successf("Switched to organization: %s\n", org.Name)
	return nil
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
	Name      string `arg:"" optional:"" help:"The name of the project"`
	Slug      string `arg:"" optional:"" help:"The slug of the project"`
	AvatarURL string `arg:"" optional:"" type:"url" help:"The avatar URL of the project"`
}

func (c *CreateProjectCmd) Run() error {
	// TODO: pass in name, slug, and avatarURL if provided
	project, err := createProject(context.Background())
	if err != nil {
		return eris.Wrap(err, "Failed to create project")
	}
	printer.Successf("Created project: %s\n", project.Name)
	return nil
}

type SwitchProjectCmd struct {
	Slug string `arg:"" optional:"" help:"The slug of the project to switch to"`
}

func (c *SwitchProjectCmd) Run() error {
	// TODO: pass in slug if provided
	project, err := selectProject(context.Background())
	if err != nil {
		return eris.Wrap(err, "Failed to select project")
	}
	if project == nil {
		printer.Infoln("No project selected.")
		return nil
	}
	printer.Successf("Switched to project: %s\n", project.Name)
	return nil
}

type UpdateProjectCmd struct {
	Name      string `arg:"" optional:"" help:"The new name of the project"`
	Slug      string `arg:"" optional:"" help:"The new slug of the project"`
	AvatarURL string `arg:"" optional:"" type:"url" help:"The new avatar URL of the project"`
}

func (c *UpdateProjectCmd) Run() error {
	// TODO: pass in name, slug, and avatarURL if provided
	return updateProject(context.Background())
}

type DeleteProjectCmd struct {
}

func (c *DeleteProjectCmd) Run() error {
	return deleteProject(context.Background())
}

// ------------------------------------------------------------------------------------------------
// User commands
// ------------------------------------------------------------------------------------------------

type UserCmd struct {
	Invite *InviteUserToOrganizationCmd     `cmd:"" group:"User Commands:" optional:"" help:"Invite a user to an organization"`
	Role   *ChangeUserRoleInOrganizationCmd `cmd:"" group:"User Commands:" optional:"" help:"Change a user's role in an organization"`
	Update *UpdateUserCmd                   `cmd:"" group:"User Commands:" optional:"" help:"Update a user"`
}

type InviteUserToOrganizationCmd struct {
	Email string `arg:"" help:"The email of the user to invite"`
	Role  string `arg:"" help:"The role of the user to invite"`
}

func (c *InviteUserToOrganizationCmd) Run() error {
	// TODO: pass in email, role if provided
	return inviteUserToOrganization(context.Background())
}

type ChangeUserRoleInOrganizationCmd struct {
	Email string `arg:"" help:"The email of the user to change the role of"`
	Role  string `arg:"" help:"The new role of the user"`
}

func (c *ChangeUserRoleInOrganizationCmd) Run() error {
	// TODO: pass in email, role if provided
	return updateUserRoleInOrganization(context.Background())
}

type UpdateUserCmd struct {
	Email string `arg:"" help:"The email of the user to update"`
	Role  string `arg:"" help:"The new role of the user"`
}

func (c *UpdateUserCmd) Run() error {
	// TODO: pass in email, role if provided
	return updateUser(context.Background())
}

/*

	logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "Tail logs for a project",
		Long: `Stream logs from your deployed project in real-time.

This command connects to your project's deployment and displays logs as they are generated,
allowing you to monitor application behavior and troubleshoot issues in real-time.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			region, err := cmd.Flags().GetString("region")
			if err != nil {
				region = ""
			}
			env, err := cmd.Flags().GetString("env")
			if err != nil {
				env = ""
			}
			return tailLogs(cmd.Context(), region, env)
		},
	}
)
*/

func InitForgeBase(env string) {
	// Set urls based on env
	switch env {
	case EnvLocal:
		baseURL = worldForgeBaseURLLocal
		rpcURL = worldForgeRPCBaseURLLocal
		argusIDBaseURL = argusIDBaseURLDev
		Env = EnvLocal
	case EnvDev:
		baseURL = worldForgeBaseURLDev
		rpcURL = worldForgeRPCBaseURLDev
		argusIDBaseURL = argusIDBaseURLDev
		Env = EnvDev
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

func InitForgeCmds() {
	// Add organization commands
	/*	organizationCmd.AddCommand(createOrganizationCmd)
		organizationCmd.AddCommand(switchOrganizationCmd)
		ForgeCmd.AddCommand(organizationCmd)

		// Add user commands
		userCmd.AddCommand(inviteUserToOrganizationCmd)
		userCmd.AddCommand(changeUserRoleInOrganizationCmd)
		userCmd.AddCommand(updateUserCmd)

		// Add project commands
		projectCmd.AddCommand(createProjectCmd)
		projectCmd.AddCommand(switchProjectCmd)
		projectCmd.AddCommand(deleteProjectCmd)
		projectCmd.AddCommand(updateProjectCmd)
		ForgeCmd.AddCommand(projectCmd)

		// Add deployment commands
		deployCmd.Flags().Bool("force", false,
			"Start the deploy even if one is currently running. Cancels current running deploy.")

		logsCmd.Flags().String("region", "", "The region to tail logs for.")
		logsCmd.Flags().String("env", "", "The environment to tail logs for.") */
}

//func AddCommands(rootCmd *cobra.Command) {
// Add login command  `world login`
/*	rootCmd.AddCommand(loginCmd)

	// deployment and status commands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(promoteCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(logsCmd)
	// user commands
	rootCmd.AddCommand(userCmd)

	// add all the other 'forge' commands
	rootCmd.AddCommand(ForgeCmd) */
//}
