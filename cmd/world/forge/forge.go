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
		err := cmd.Help()
		if err != nil {
			return eris.Wrap(err, "Failed to show help")
		}
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
			org, err := selectOrganization(cmd.Context())
			if err != nil {
				return eris.Wrap(err, "Failed to select organization")
			}
			fmt.Println("Switched to organization: ", org.Name)
			return nil
		},
	}
)

// Project commands
var (
	projectCmd = &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return showProjectList(cmd.Context())
		},
	}

	switchProjectCmd = &cobra.Command{
		Use:   "switch",
		Short: "Switch to a project",
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			return createProject(cmd.Context())
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
	BaseCmd.AddCommand(organizationCmd)

	// Add project commands
	projectCmd.AddCommand(createProjectCmd)
	projectCmd.AddCommand(switchProjectCmd)
	BaseCmd.AddCommand(projectCmd)
}