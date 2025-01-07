package forge

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/rotisserie/eris"

	"pkg.world.dev/world-cli/common/globalconfig"
)

type project struct {
	ID          string `json:"id"`
	OrgID       string `json:"org_id"`
	OwnerID     string `json:"owner_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	CreatedTime string `json:"created_time"`
	UpdatedTime string `json:"updated_time"`
	Deleted     bool   `json:"deleted"`
	DeletedTime string `json:"deleted_time"`
	RepoURL     string `json:"repo_url"`
	RepoToken   string `json:"repo_token"`
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

	selectedProject, err := getSelectedProject(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get selected project")
	}

	fmt.Println("Your projects:")
	fmt.Println("--------------")
	for _, project := range projects {
		if project.ID == selectedProject.ID {
			fmt.Printf("* %s (%s) [SELECTED]\n", project.Name, project.Slug)
		} else {
			fmt.Printf("  %s (%s)\n", project.Name, project.Slug)
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
	projectURL := fmt.Sprintf(projectURLPattern+"/%s", baseURL, selectedOrg.ID, config.ProjectID)
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
	// Get organization
	org, err := getSelectedOrganization(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get organization")
	}

	if org.ID == "" {
		printNoSelectedOrganization()
		return nil
	}

	name, slug, err := inputProjectNameAndSlug()
	if err != nil {
		return eris.Wrap(err, "Failed to get project name and slug")
	}

	repoURL, repoToken, err := inputRepoURLAndToken(ctx)
	if err != nil {
		return eris.Wrap(err, "Failed to get repository URL and token")
	}

	// Send request
	url := fmt.Sprintf(projectURLPattern, baseURL, org.ID)
	body, err := sendRequest(ctx, http.MethodPost, url, map[string]string{
		"name":       name,
		"slug":       slug,
		"repo_url":   repoURL,
		"repo_token": repoToken,
		"org_id":     org.ID,
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

func inputProjectNameAndSlug() (string, string, error) {
	var name, slug string
	var err error

	// Get project name
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		fmt.Print("Enter project name: ")
		name, err = getInput()
		if err != nil {
			return "", "", eris.Wrap(err, "Failed to read project name")
		}

		// Check for empty string
		if name == "" {
			fmt.Printf("Error: Project name cannot be empty\n")
			attempts++
			continue
		}

		// Check length (arbitrary max of 50 chars)
		if len(name) > 50 {
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

		break
	}

	if attempts >= maxAttempts {
		return "", "", eris.New("Maximum attempts reached for entering project name")
	}

	// Get and validate project slug
	attempts = 0
	maxAttempts = 5
	for attempts < maxAttempts {
		fmt.Print("Enter project slug (5 characters, alphanumeric only): ")
		slug, err = getInput()
		if err != nil {
			return "", "", eris.Wrap(err, "Failed to read project slug")
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

		return name, slug, nil
	}

	return "", "", eris.New("Maximum attempts reached for project slug")
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
