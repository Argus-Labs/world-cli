package forge

import (
	"fmt"

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
)

var BaseCmd = &cobra.Command{
	Use:   "forge",
	Short: "Forge is a tool for managing World Forge projects",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !checkLogin() {
			return nil
		}

		// Get user info
		globalConfig, err := globalconfig.GetGlobalConfig()
		if err != nil {
			return eris.Wrap(err, "Failed to get user")
		}

		fmt.Println("✨ World Forge Status ✨")
		fmt.Println("=====================")
		fmt.Println("\n👤 User Information")
		fmt.Println("------------------")
		fmt.Printf("ID:   %s\n", globalConfig.Credential.ID)
		fmt.Printf("Name: %s\n", globalConfig.Credential.Name)

		// Show organization list
		err = showOrganizationList(cmd.Context())
		if err != nil {
			return eris.Wrap(err, "Failed to show organization list")
		}

		// Show project list
		err = showProjectList(cmd.Context())
		if err != nil {
			return eris.Wrap(err, "Failed to show project list")
		}

		// add separator
		fmt.Println("\n================================================")

		return nil
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
			return showOrganizationList(cmd.Context())
		},
	}

	createOrganizationCmd = &cobra.Command{
		Use:   "create",
		Short: "Create an organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			org, err := createOrganization(cmd.Context())
			if err != nil {
				return eris.Wrap(err, "Failed to create organization")
			}
			fmt.Println("Organization created successfully")
			fmt.Println("Organization Name: ", org.Name)
			fmt.Println("Organization Slug: ", org.Slug)
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

	inviteUserToOrganizationCmd = &cobra.Command{
		Use:   "invite",
		Short: "Invite a user to an organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return inviteUserToOrganization(cmd.Context())
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
			return showProjectList(cmd.Context())
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
			return createProject(cmd.Context())
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
	deploymentCmd = &cobra.Command{
		Use:   "deployment",
		Short: "Manage deployments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !checkLogin() {
				return nil
			}
			// TODO: return deployment list and status
			return cmd.Help()
		},
	}

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
)

func init() {
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

	// Add login command
	BaseCmd.AddCommand(loginCmd)

	// Add organization commands
	organizationCmd.AddCommand(createOrganizationCmd)
	organizationCmd.AddCommand(switchOrganizationCmd)
	organizationCmd.AddCommand(inviteUserToOrganizationCmd)
	BaseCmd.AddCommand(organizationCmd)

	// Add project commands
	projectCmd.AddCommand(createProjectCmd)
	projectCmd.AddCommand(switchProjectCmd)
	projectCmd.AddCommand(deleteProjectCmd)
	projectCmd.AddCommand(updateProjectCmd)
	BaseCmd.AddCommand(projectCmd)

	// Add deployment commands
	deployCmd.Flags().Bool("force", false,
		"Start the deploy even if one is currently running. Cancels current running deploy.")
	deploymentCmd.AddCommand(deployCmd)
	deploymentCmd.AddCommand(destroyCmd)
	deploymentCmd.AddCommand(statusCmd)
	deploymentCmd.AddCommand(resetCmd)
	BaseCmd.AddCommand(deploymentCmd)
}
