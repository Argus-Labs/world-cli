package forge

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/globalconfig"
	"pkg.world.dev/world-cli/tea/component/multiselect"
)

const MaxProjectNameLen = 50

var regionSelector *tea.Program

type project struct {
	ID          string        `json:"id"`
	OrgID       string        `json:"org_id"`
	OwnerID     string        `json:"owner_id"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	CreatedTime string        `json:"created_time"`
	UpdatedTime string        `json:"updated_time"`
	Deleted     bool          `json:"deleted"`
	DeletedTime string        `json:"deleted_time"`
	RepoURL     string        `json:"repo_url"`
	RepoToken   string        `json:"repo_token"`
	Config      projectConfig `json:"config"`
}

type projectConfig struct {
	EnvName  string   `json:"env_name"`
	TickRate int      `json:"tick_rate"`
	Region   []string `json:"region"`
}

// Show list of projects in selected organization
func showProjectList(ctx context.Context) error {
	projects, err := getListOfProjects(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		printNoProjectsInOrganization()
		return nil
	}

	project, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get selected project")
	}

	fmt.Println("\n📁 Project Information")
	fmt.Println("--------------------")
	if project.Name == "" {
		fmt.Println("No project selected")
	} else {
		fmt.Println("\nAvailable Projects:")
		for _, prj := range projects {
			if prj.ID == project.ID {
				fmt.Printf("* %s (%s) [SELECTED]\n", prj.Name, prj.Slug)
			} else {
				fmt.Printf("  %s (%s)\n", prj.Name, prj.Slug)
			}
		}
	}

	return nil
}

// Get selected project
func getSelectedProject(ctx context.Context) (project, error) {
	selectedOrg, err := getSelectedOrganization(ctx)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get organization")
	}

	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return project{}, nil
	}

	// Get config
	config, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get config")
	}

	if config.ProjectID == "" {
		printNoSelectedProject()
		return project{}, nil
	}

	// Send request
	projectURL := fmt.Sprintf(projectURLPattern, baseURL, selectedOrg.ID) + "/" + config.ProjectID
	body, err := sendRequest(ctx, http.MethodGet, projectURL, nil)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get project")
	}

	// Parse response
	prj, err := parseResponse[project](body)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to parse project")
	}

	return *prj, nil
}

// Get list of projects in selected organization
func getListOfProjects(ctx context.Context) ([]project, error) {
	selectedOrg, err := getSelectedOrganization(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get organization")
	}

	if selectedOrg.ID == "" {
		printNoSelectedOrganization()
		return nil, nil
	}

	url := fmt.Sprintf(projectURLPattern, baseURL, selectedOrg.ID)
	body, err := sendRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to get projects")
	}

	projects, err := parseResponse[[]project](body)
	if err != nil {
		return nil, eris.Wrap(err, "Failed to parse projects")
	}

	return *projects, nil
}

func createProject(ctx context.Context) error {
	projectModel, err := projectInput(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project input")
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, projectModel.OrgID)
	body, err := sendRequest(ctx, http.MethodPost, url, map[string]interface{}{
		"name":       projectModel.Name,
		"slug":       projectModel.Slug,
		"repo_url":   projectModel.RepoURL,
		"repo_token": projectModel.RepoToken,
		"org_id":     projectModel.OrgID,
		"config":     projectModel.Config,
	})
	if err != nil {
		return eris.Wrap(err, "Failed to create project")
	}

	prj, err := parseResponse[project](body)
	if err != nil {
		return eris.Wrap(err, "Failed to parse response")
	}

	fmt.Printf("Project created successfully: %s (%s)\n", prj.Name, prj.Slug)
	return nil
}

func inputProjectName(ctx context.Context) (string, error) {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Print("Enter project name: ")
			name, err := getInput()
			if err != nil {
				return "", eris.Wrap(err, "Failed to read project name")
			}

			// Check for empty string
			if name == "" {
				fmt.Printf("Error: Project name cannot be empty\n")
				attempts++
				continue
			}

			// Check length (arbitrary max of 50 chars)
			if len(name) > MaxProjectNameLen {
				fmt.Printf("Error: Project name cannot be longer than 50 characters\n")
				attempts++
				continue
			}

			// Check for problematic characters
			if strings.ContainsAny(name, "<>:\"/\\|?*") {
				fmt.Printf("Error: Project name contains invalid characters\n")
				attempts++
				continue
			}

			return name, nil
		}
	}

	return "", eris.New("Maximum attempts reached for entering project name")
}

func inputProjectSlug(ctx context.Context) (string, error) {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Print("Enter project slug (5 characters, alphanumeric only): ")
			slug, err := getInput()
			if err != nil {
				return "", eris.Wrap(err, "Failed to read project slug")
			}

			// Validate slug
			if len(slug) != 5 { //nolint:gomnd
				fmt.Printf("Error: Slug must be exactly 5 characters\n")
				attempts++
				continue
			}

			if !isAlphanumeric(slug) {
				fmt.Printf("Error: Slug must contain only letters (a-z|A-Z) and numbers (0-9)\n")
				attempts++
				continue
			}

			return slug, nil
		}
	}

	return "", eris.New("Maximum attempts reached for project slug")
}

func inputRepoURLAndToken(ctx context.Context) (string, string, error) {
	// Get repository URL and token, then validate them together
	var repoURL, repoToken string
	var err error
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		// Get repository URL
		fmt.Print("Enter repository URL (https format): ")
		repoURL, err = getInput()
		if err != nil {
			return "", "", eris.Wrap(err, "Failed to read repository URL")
		}

		if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
			fmt.Printf("Error: Repository URL must start with http:// or https://\n")
			attempts++
			continue
		}

		// Get repository token
		fmt.Print("Enter repository personal access token (leave empty for public repositories): ")
		repoToken, err = getInput()
		if err != nil {
			return "", "", eris.Wrap(err, "Failed to read personal access token")
		}

		// Validate repository access using the new validateRepoToken function
		if err := validateRepoToken(ctx, repoURL, repoToken); err != nil {
			fmt.Printf("Error: %v\n", err)
			attempts++
			continue
		}

		return repoURL, repoToken, nil
	}

	return "", "", eris.New("Maximum attempts reached for repository validation")
}

func selectProject(ctx context.Context) (project, error) {
	// Get projects from selected organization
	projects, err := getListOfProjects(ctx)
	if err != nil {
		return project{}, eris.Wrap(err, "Failed to get projects")
	}

	if len(projects) == 0 {
		printNoProjectsInOrganization()
		return project{}, nil
	}

	// Display projects as a numbered list
	fmt.Println("\nAvailable projects:")
	for i, proj := range projects {
		fmt.Printf("%d. %s (%s)\n", i+1, proj.Name, proj.Slug)
	}

	// Get user input
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		fmt.Print("\nEnter project number (or 'q' to quit): ")
		input, err := getInput()
		if err != nil {
			return project{}, eris.Wrap(err, "Failed to read input")
		}

		input = strings.TrimSpace(input)
		if input == "q" {
			return project{}, eris.New("Project selection canceled")
		}

		// Parse selection
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(projects) {
			fmt.Printf("Invalid selection. Please enter a number between 1 and %d\n", len(projects))
			attempts++
			continue
		}

		selectedProject := projects[num-1]

		// Save project to config file
		config, err := globalconfig.GetGlobalConfig()
		if err != nil {
			return project{}, eris.Wrap(err, "Failed to get config")
		}
		config.ProjectID = selectedProject.ID
		err = globalconfig.SaveGlobalConfig(config)
		if err != nil {
			return project{}, eris.Wrap(err, "Failed to save project")
		}

		return selectedProject, nil
	}

	return project{}, eris.New("Maximum attempts reached for selecting project")
}

func deleteProject(ctx context.Context) error {
	project, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project")
	}

	// Print project details with formatting
	fmt.Println("\n🗑️  Project Deletion")
	fmt.Println("------------------")
	fmt.Printf("Project Name: %s\n", project.Name)
	fmt.Printf("Project Slug: %s\n\n", project.Slug)

	// Warning message
	fmt.Println("⚠️  WARNING")
	fmt.Println("  This will permanently delete:")
	fmt.Println("  • All deployments")
	fmt.Println("  • All logs")
	fmt.Println("  • All associated resources")
	fmt.Println("")

	// Confirmation prompt
	fmt.Printf("❓ Are you sure you want to delete %s? (Y/n): ", project.Name)
	confirmation, err := getInput()
	if err != nil {
		return eris.Wrap(err, "Failed to read confirmation")
	}

	if confirmation != "Y" {
		if confirmation == "y" {
			fmt.Println("You need to put Y (uppercase) to confirm deletion")
			fmt.Println("\n❌ Project deletion canceled")
			return nil
		}

		fmt.Println("\n❌ Project deletion canceled")
		return nil
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, project.OrgID) + "/" + project.ID
	body, err := sendRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return eris.Wrap(err, "Failed to delete project")
	}

	// Parse response
	_, err = parseResponse[any](body)
	if err != nil {
		return eris.Wrap(err, "Failed to parse response")
	}

	fmt.Printf("Project deleted successfully: %s (%s)\n", project.Name, project.Slug)

	// Remove project from config
	config, err := globalconfig.GetGlobalConfig()
	if err != nil {
		return eris.Wrap(err, "Failed to get config")
	}
	config.ProjectID = ""
	err = globalconfig.SaveGlobalConfig(config)
	if err != nil {
		return eris.Wrap(err, "Failed to save config")
	}

	return nil
}

func updateProject(ctx context.Context) error {
	// get selected project
	project, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get selected project")
	}

	// get project input
	projectModel, err := projectInput(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get project input")
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, project.OrgID) + "/" + project.ID
	body, err := sendRequest(ctx, http.MethodPut, url, map[string]interface{}{
		"name":       projectModel.Name,
		"slug":       projectModel.Slug,
		"repo_url":   projectModel.RepoURL,
		"repo_token": projectModel.RepoToken,
		"config":     projectModel.Config,
	})
	if err != nil {
		return eris.Wrap(err, "Failed to update project")
	}

	_, err = parseResponse[any](body)
	if err != nil {
		return eris.Wrap(err, "Failed to parse response")
	}

	fmt.Printf("Project updated successfully: %s (%s)\n", projectModel.Name, projectModel.Slug)

	return nil
}

func projectInput(ctx context.Context) (project, error) {
	project := project{}

	// Get organization
	org, err := getSelectedOrganization(ctx)
	if err != nil {
		return project, eris.Wrap(err, "Failed to get organization")
	}
	project.OrgID = org.ID

	if org.ID == "" {
		printNoSelectedOrganization()
		return project, nil
	}

	name, err := inputProjectName(ctx)
	if err != nil {
		return project, eris.Wrap(err, "Failed to get project name")
	}
	project.Name = name

	slug, err := inputProjectSlug(ctx)
	if err != nil {
		return project, eris.Wrap(err, "Failed to get project slug")
	}
	project.Slug = slug

	repoURL, repoToken, err := inputRepoURLAndToken(ctx)
	if err != nil {
		return project, eris.Wrap(err, "Failed to get repository URL and token")
	}
	project.RepoURL = repoURL
	project.RepoToken = repoToken

	// Env Name
	envName, err := inputEnvName(ctx)
	if err != nil {
		return project, eris.Wrap(err, "Failed to get environment name")
	}
	project.Config.EnvName = envName

	// Tick Rate
	tickRate, err := inputTickRate(ctx)
	if err != nil {
		return project, eris.Wrap(err, "Failed to get environment name")
	}
	project.Config.TickRate = tickRate

	// Regions
	regions, err := chooseRegion(ctx)
	if err != nil {
		return project, eris.Wrap(err, "Failed to choose region")
	}
	project.Config.Region = regions

	return project, nil
}

// inputEnvName prompts the user to enter an environment name (e.g. dev, staging, prod)
// and validates that it is not empty. Returns error after max attempts or context cancellation.
func inputEnvName(ctx context.Context) (string, error) {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fmt.Print("Enter environment name (e.g. 'dev', 'staging', 'prod'): ")
			envName, err := getInput()
			if err != nil {
				attempts++
				fmt.Printf("Error: Failed to read environment name\n")
				continue
			}
			if envName == "" {
				attempts++
				fmt.Printf("Error: Environment name cannot be empty\n")
				continue
			}
			return envName, nil
		}
	}
	return "", eris.New("Maximum attempts reached for entering environment name")
}

// inputTickRate prompts the user to enter a tick rate value (default is 1)
// and validates that it is a valid number. Returns error after max attempts or context cancellation.
func inputTickRate(ctx context.Context) (int, error) {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			fmt.Print("Enter tick rate (e.g. 10, 20, 30, default is 1): ")
			tickRate, err := getInputInt()
			if err != nil {
				attempts++
				fmt.Printf("Error: Invalid input. Please enter a number\n")
				continue
			}
			return tickRate, nil
		}
	}
	return 0, eris.New("Maximum attempts reached for entering tick rate")
}

// chooseRegion displays an interactive menu for selecting one or more AWS regions
// using the bubbletea TUI library. Returns error if no regions selected after max attempts
// or context cancellation.
func chooseRegion(ctx context.Context) ([]string, error) {
	// TODO: get regions from backend
	regions := []string{
		"us-east-1",
		"us-west-1",
		"eu-west-1",
		"eu-central-1",
		"ap-southeast-1",
		"ap-northeast-1",
	}

	for attempts := 0; attempts < 5; attempts++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			selectedRegions, err := runRegionSelector(ctx, regions)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			if len(selectedRegions) > 0 {
				return selectedRegions, nil
			}
			fmt.Println("Error: At least one region must be selected")
		}
	}

	return nil, eris.New("Maximum attempts reached for selecting regions")
}

func runRegionSelector(ctx context.Context, regions []string) ([]string, error) {
	if regionSelector == nil {
		regionSelector = tea.NewProgram(multiselect.InitialMultiselectModel(ctx, regions))
	}
	m, err := regionSelector.Run()
	if err != nil {
		return nil, eris.Wrap(err, "failed to run region selector")
	}

	model, ok := m.(multiselect.Model)
	if !ok {
		return nil, eris.New("failed to get selected regions")
	}

	var selectedRegions []string
	for i, item := range regions {
		if model.Selected[i] {
			selectedRegions = append(selectedRegions, item)
		}
	}

	return selectedRegions, nil
}
