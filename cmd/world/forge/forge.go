package forge

import (
	"fmt"
	"os"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/globalconfig"
)

const (
	// For local development
	worldForgeBaseURLLocal = "http://localhost:8001"

	// For production
	worldForgeBaseURLProd = "https://forge.world.dev"
)

var (
	// baseUrl is the base URL for the Forge API
	baseURL string

	// login url stuff
	loginURL    string
	getTokenURL string

	// organization url stuff
	organizationURL string

	// project url stuff
	projectURLPattern = "%s/api/organization/%s/project"

	// user url stuff
	userURL string

	// argusid flag
	// Set this to true if you want to use ArgusID for default login
	argusid = false
)

var ForgeCmd = &cobra.Command{
	Use:   "forge",
	Short: "Forge is a tool for managing World Forge projects",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !checkLogin() {
			return nil
		}

		// Get user info
		globalConfig, err := GetCurrentConfig()
		if err != nil {
			return eris.Wrap(err, "Failed to get user")
		}

		fmt.Println("✨ World Forge Status ✨")
		fmt.Println("=====================")
		fmt.Println("\n👤 User Information")
		fmt.Println("------------------")
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
	Short: "Authenticate with World Forge",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return login(cmd.Context())
	},
}

// Organization commands
var (
	organizationCmd = &cobra.Command{
		Use:   "organization",
		Short: "Manage organizations",
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
		Short: "Create an organization",
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
		Short: "Switch to an organization",
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

// User Commands
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

// Project commands
var (
	projectCmd = &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
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
		Short: "Switch to a project",
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
		Short: "Create a project",
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
		Short: "Delete a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return deleteProject(cmd.Context())
		},
	}

	updateProjectCmd = &cobra.Command{
		Use:   "update",
		Short: "Update a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return updateProject(cmd.Context())
		},
	}
)

// Deployment commands
var (
	deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a project",
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
		Short: "Destroy a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return deployment(cmd.Context(), "destroy")
		},
	}

	resetCmd = &cobra.Command{
		Use:   "reset",
		Short: "Reset a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return deployment(cmd.Context(), "reset")
		},
	}

	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show status of a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			return status(cmd.Context())
		},
	}

	promoteCmd = &cobra.Command{
		Use:   "promote",
		Short: "Promote a project from dev to prod environment",
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
