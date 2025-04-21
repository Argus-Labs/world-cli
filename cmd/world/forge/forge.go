package forge

import (
	"fmt"
	"os"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/globalconfig"
)

const (
	// For local development.
	worldForgeBaseURLLocal = "http://localhost:8001"

	// For production.
	worldForgeBaseURLProd = "https://forge.world.dev"
)

var (
	// baseUrl is the base URL for the Forge API.
	baseURL string

	// login url stuff.
	loginURL    string
	getTokenURL string

	// organization url stuff.
	organizationURL string

	// project url stuff.
	projectURLPattern = "%s/api/organization/%s/project"

	// user url stuff.
	userURL string

	// Set this to true if you want to use ArgusID for default login.
	argusid = false
)

var ForgeCmd = &cobra.Command{
	Use:   "forge",
	Short: "Manage and deploy your World Forge projects with ease",
	Long: `Access the World Forge platform to create, manage, and deploy your game projects.

World Forge provides a complete project management solution for your game development,
allowing you to organize teams, manage deployments, and monitor your game services.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !checkLogin() {
			return nil
		}

		// Get user info
		globalConfig, err := GetCurrentConfig()
		if err != nil {
			return eris.Wrap(err, "Failed to get user")
		}

		fmt.Println("   World Forge Status")
		fmt.Println("========================")
		fmt.Println("\n    User Information")
		fmt.Println("------------------------")
		fmt.Printf("ID:   %s\n", globalConfig.Credential.ID)
		fmt.Printf("Name: %s\n", globalConfig.Credential.Name)

		// Try to show org list and project list
		// Show organization list
		err = showOrganizationList(cmd.Context())

		if err == nil {
			// Show project list, if we have an org
			_ = showProjectList(cmd.Context())
		}

		// add separator
		fmt.Println("\n================================================")

		return cmd.Help()
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Connect your account to World Forge",
	Long: `Securely authenticate with World Forge to access your projects and teams.

This command opens your browser for a secure login process and saves your credentials
locally for future CLI commands. You'll need to complete this step before using most
World Forge features.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return login(cmd.Context())
	},
}

// Organization commands.
var (
	organizationCmd = &cobra.Command{
		Use:   "organization",
		Short: "Create and manage your development teams",
		Long: `Organize your development teams and control project access.
		
This command helps you create, switch between, and manage organizations
that serve as containers for your World Forge projects and team members.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := showOrganizationList(cmd.Context())
			if err == nil {
				// add separator
				fmt.Println("\n================================================")
			}
			return cmd.Help()
		},
	}

	createOrganizationCmd = &cobra.Command{
		Use:   "create",
		Short: "Set up a new development team",
		Long: `Create a new organization to manage your team and projects.
		
This command walks you through setting up a new organization with a unique name,
slug, and avatar URL. Organizations serve as containers for your projects and team members.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			_, err := createOrganization(cmd.Context())
			if err != nil {
				return eris.Wrap(err, "Failed to create organization")
			}
			return nil
		},
	}

	switchOrganizationCmd = &cobra.Command{
		Use:   "switch",
		Short: "Change your active development team",
		Long: `Select a different organization as your active working context.
		
This command displays a list of all organizations you belong to and allows you
to select one as your active context for subsequent commands. Projects and resources
are organized within organizations.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			org, err := selectOrganization(cmd.Context())
			if err != nil {
				return eris.Wrap(err, "Failed to select organization")
			}
			fmt.Println("Switched to organization: ", org.Name)
			return nil
		},
	}
)

// User Commands.
var (
	userCmd = &cobra.Command{
		Use:   "user",
		Short: "Manage users",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			err := showOrganizationList(cmd.Context())
			if err == nil {
				// add separator
				fmt.Println("\n================================================")
			}
			return cmd.Help()
		},
	}

	inviteUserToOrganizationCmd = &cobra.Command{
		Use:   "invite",
		Short: "Invite a user to selected organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return inviteUserToOrganization(cmd.Context())
		},
	}

	changeUserRoleInOrganizationCmd = &cobra.Command{
		Use:   "role",
		Short: "Change a user's role in selected organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return updateUserRoleInOrganization(cmd.Context())
		},
	}

	updateUserCmd = &cobra.Command{
		Use:   "update",
		Short: "Update user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return updateUser(cmd.Context())
		},
	}
)

// Project commands.
var (
	projectCmd = &cobra.Command{
		Use:   "project",
		Short: "Create and manage your game projects",
		Long: `Build and organize your World Engine game projects.
		
This command helps you create, switch between, and manage your game projects,
providing a centralized way to handle your game's development lifecycle.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			err := showProjectList(cmd.Context())
			if err == nil {
				// add separator
				fmt.Println("\n================================================")
			}
			return cmd.Help()
		},
	}

	switchProjectCmd = &cobra.Command{
		Use:   "switch",
		Short: "Change your active game project",
		Long: `Select a different project as your active working context.
		
This command displays a list of all projects in your current organization and allows you
to select one as your active context for subsequent commands. All deployment and
management operations will target this selected project.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			prj, err := selectProject(cmd.Context())
			if err != nil {
				return eris.Wrap(err, "Failed to select project")
			}
			if prj == nil {
				fmt.Println("No project selected.")
				return nil
			}
			fmt.Println("Switched to project: ", prj.Name)
			return nil
		},
	}

	createProjectCmd = &cobra.Command{
		Use:   "create",
		Short: "Set up a new game project",
		Long: `Create a new World Engine game project with customized settings.
		
This command guides you through creating a new project with your desired configuration,
including repository settings, deployment regions, notification integrations, and more.
All settings can be updated later using the 'update' command.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			_, err := createProject(cmd.Context())
			if err != nil {
				return eris.Wrap(err, "Failed to create project")
			}
			return nil
		},
	}

	deleteProjectCmd = &cobra.Command{
		Use:   "delete",
		Short: "Remove a game project from your organization",
		Long: `Permanently delete a project from your organization.
		
This command allows you to remove a project that is no longer needed. You will be
prompted to confirm the deletion to prevent accidental removal of important projects.
This action cannot be undone.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return deleteProject(cmd.Context())
		},
	}

	updateProjectCmd = &cobra.Command{
		Use:   "update",
		Short: "Modify your existing game project settings",
		Long: `Update configuration settings for your current game project.
		
This command allows you to modify various aspects of your project including name, 
repository settings, deployment regions, notification integrations, and more. You'll
be guided through each setting with the option to keep existing values.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return updateProject(cmd.Context())
		},
	}
)

// Deployment commands.
var (
	deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Launch your game project to the cloud",
		Long: `Deploy your World Engine game project to production servers.
		
This command builds and deploys your game to the selected regions, making it
available for players. Use the --force flag to restart a deployment if one
is already in progress.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			force, _ := cmd.Flags().GetBool("force")
			deployType := "deploy"
			if force {
				deployType = "forceDeploy"
			}
			return deployment(cmd.Context(), deployType)
		},
	}

	destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "Shut down your deployed game services",
		Long: `Remove your game project's deployed infrastructure from the cloud.
		
This command terminates all running instances of your game in the cloud, freeing up
resources. Your project configuration remains intact, allowing you to redeploy later
if needed.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return deployment(cmd.Context(), "destroy")
		},
	}

	resetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Restart your game project with a clean state",
		Long: `Reset your deployed game project to its initial state.
		
This command clears all game state data while keeping your deployment running,
allowing you to start fresh without redeploying the entire infrastructure.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return deployment(cmd.Context(), "reset")
		},
	}

	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check your game project's deployment status",
		Long: `View the current state of your deployed game project.
		
This command shows detailed information about your project's deployment status,
including running instances, regions, and any ongoing deployment operations.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return status(cmd.Context())
		},
	}

	promoteCmd = &cobra.Command{
		Use:   "promote",
		Short: "Move your game from development to production",
		Long: `Promote your game project from development to production environment.
		
This command transitions your game from a development environment to production,
making it ready for a wider audience. This process ensures your game is deployed
with production-grade infrastructure and settings.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return deployment(cmd.Context(), "promote")
		},
	}
)

func InitForge() {
	// Set argusid flag
	if os.Getenv("WORLD_CLI_LOGIN_METHOD") == "argusid" {
		argusid = true
	} else if os.Getenv("WORLD_CLI_LOGIN_METHOD") == "github" {
		argusid = false
	}

	// Set base URL
	if globalconfig.Env == "PROD" {
		baseURL = worldForgeBaseURLProd
	} else {
		baseURL = worldForgeBaseURLLocal
	}

	// Set login URL
	loginURL = fmt.Sprintf("%s/api/user/login", baseURL)
	getTokenURL = fmt.Sprintf("%s/api/user/login/get-token", baseURL)

	// Set organization URL
	organizationURL = fmt.Sprintf("%s/api/organization", baseURL)

	// Set user URL
	userURL = fmt.Sprintf("%s/api/user", baseURL)

	// Add organization commands
	organizationCmd.AddCommand(createOrganizationCmd)
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
}

func AddCommands(rootCmd *cobra.Command) {
	// Add login command  `world login`
	rootCmd.AddCommand(loginCmd)

	// deployment and status commands
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(destroyCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(promoteCmd)
	rootCmd.AddCommand(resetCmd)

	// user commands
	rootCmd.AddCommand(userCmd)

	// add all the other 'forge' commands
	rootCmd.AddCommand(ForgeCmd)
}
