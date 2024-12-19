package forge

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
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
			fmt.Printf("* %s (%s) [SELECTED]\n", project.Name, project.ID)
		} else {
			fmt.Printf("  %s (%s)\n", project.Name, project.ID)
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

	// Get project name
	fmt.Print("Enter project name: ")
	if _, err := fmt.Scanln(&name); err != nil {
		return "", "", eris.Wrap(err, "Failed to read project name")
	}

	// Get and validate project slug
	for {
		fmt.Print("Enter project slug (5 characters, alphanumeric lowercase only): ")
		if _, err := fmt.Scanln(&slug); err != nil {
			return "", "", eris.Wrap(err, "Failed to read project slug")
		}

		// Validate slug
		if len(slug) != 5 { //nolint:gomnd
			fmt.Println("Error: Slug must be exactly 5 characters")
			continue
		}

		if !isAlphanumeric(slug) {
			fmt.Println("Error: Slug must contain only lowercase letters (a-z) and numbers (0-9)")
			continue
		}

		break
	}

	return name, slug, nil
}

func inputRepoURLAndToken(ctx context.Context) (string, string, error) {
	// Get repository URL and token, then validate them together
	var repoURL, repoToken string
	for {
		// Get repository URL
		fmt.Print("Enter repository URL (https format): ")
		if _, err := fmt.Scanln(&repoURL); err != nil {
			return "", "", eris.Wrap(err, "Failed to read repository URL")
		}

		if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
			fmt.Println("Error: Repository URL must start with http:// or https://")
			continue
		}

		// Remove protocol prefix from URL for storage
		cleanURL := strings.TrimPrefix(repoURL, "http://")
		cleanURL = strings.TrimPrefix(cleanURL, "https://")

		// Get repository token
		fmt.Print("Enter repository personal access token (leave empty for public repositories): ")
		tokenInput, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", "", eris.Wrap(err, "Failed to read personal access token")
		}
		repoToken = strings.TrimSpace(tokenInput)

		// Validate repository access using the new validateRepoToken function
		if err := validateRepoToken(ctx, repoURL, repoToken); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		repoURL = cleanURL
		break
	}

	return repoURL, repoToken, nil
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
		fmt.Printf("%d. %s (%s)\n", i+1, proj.Name, proj.ID)
	}

	// Get user input
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nEnter project number (or 'q' to quit): ")
		input, err := reader.ReadString('\n')
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
			fmt.Println("Invalid selection. Please enter a number between 1 and", len(projects))
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
}
